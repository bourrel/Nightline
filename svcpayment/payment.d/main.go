package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"

	csvcdb "svcdb/client"
	csvcevent "svcevent/client"
	
	"svcpayment"
)

func main() {
	var (
		debugAddr       = flag.String("debug.addr", ":8037", "Debug and metrics listen address")
		httpAddr        = flag.String("http.addr", ":8047", "HTTP listen address")
		zipkinAddr      = flag.String("zipkin.addr", "", "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")
		zipkinKafkaAddr = flag.String("zipkin.kafka.addr", "", "Enable Zipkin tracing via a Kafka server host:port")
		appdashAddr     = flag.String("appdash.addr", "", "Enable Appdash tracing via an Appdash server host:port")
		lightstepToken  = flag.String("lightstep.token", "", "Enable LightStep tracing via a LightStep access token")
	)
	flag.Parse()
	
	/* Logger */
	var logger log.Logger
	{
		logger = log.NewJSONLogger(os.Stdout)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	logger.Log("msg", "[SVCPAYMENT BEGIN]")
	defer logger.Log("msg", "[SVCPAYMENT END]")

	/* Metrics */
	var ints metrics.Counter
	{
		// Business level metrics.
		ints = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "svcpayment",
			Name:      "integers_summed",
			Help:      "Total count of users summed via the createUser method.",
		}, []string{})
	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "svcpayment",
			Name:      "request_duration_ns",
			Help:      "Request duration in nanoseconds.",
		}, []string{"method", "success"})
	}

	/* Tracer */
	var tracer stdopentracing.Tracer
	{
		if *zipkinAddr != "" {
			logger := log.With(logger, "tracer", "ZipkinHTTP")
			logger.Log("addr", *zipkinAddr)

			// endpoint typically looks like: http://zipkinhost:9411/api/v1/spans
			collector, err := zipkin.NewHTTPCollector(*zipkinAddr)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
			defer collector.Close()

			tracer, err = zipkin.NewTracer(
				zipkin.NewRecorder(collector, false, "localhost:80", "svcpayment"),
			)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
		} else if *zipkinKafkaAddr != "" {
			logger := log.With(logger, "tracer", "ZipkinKafka")
			logger.Log("addr", *zipkinKafkaAddr)

			collector, err := zipkin.NewKafkaCollector(
				strings.Split(*zipkinKafkaAddr, ","),
				zipkin.KafkaLogger(log.NewNopLogger()),
			)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
			defer collector.Close()

			tracer, err = zipkin.NewTracer(
				zipkin.NewRecorder(collector, false, "localhost:80", "svcpayment"),
			)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
		} else if *appdashAddr != "" {
			logger := log.With(logger, "tracer", "Appdash")
			logger.Log("addr", *appdashAddr)
			tracer = appdashot.NewTracer(appdash.NewRemoteCollector(*appdashAddr))
		} else if *lightstepToken != "" {
			logger := log.With(logger, "tracer", "LightStep")
			logger.Log() // probably don't want to print out the token :)
			tracer = lightstep.NewTracer(lightstep.Options{
				AccessToken: *lightstepToken,
			})
			defer lightstep.FlushLightStepTracer(tracer)
		} else {
			logger := log.With(logger, "tracer", "none")
			logger.Log()
			tracer = stdopentracing.GlobalTracer() // no-op
		}
	}

	/* Business domain */
	var service svcpayment.IService
	{
		svcdb, err := csvcdb.New("localhost:8044", tracer, logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		svcevent, err := csvcevent.New("localhost:8049", tracer, logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		
		service = svcpayment.NewService(svcdb, svcevent)
		service = svcpayment.ServiceLoggingMiddleware(logger)(service)
		service = svcpayment.ServiceInstrumentingMiddleware(ints)(service)
	}
	
	/* Endpoints domain */
	/* Order */
	getOrderEndpoint := svcpayment.BuildGetOrderEndpoint(service, logger, tracer, duration)
	createOrderEndpoint := svcpayment.BuildCreateOrderEndpoint(service, logger, tracer, duration)
	putOrderEndpoint := svcpayment.BuildPutOrderEndpoint(service, logger, tracer, duration)
	searchOrdersEndpoint := svcpayment.BuildSearchOrdersEndpoint(service, logger, tracer, duration)
	answerOrderEndpoint := svcpayment.BuildAnswerOrderEndpoint(service, logger, tracer, duration)

	/* Pro */
	registerProEndpoint := svcpayment.BuildRegisterProEndpoint(service, logger, tracer, duration)
	
	endpoints := svcpayment.Endpoints{
		/* Order */
		GetOrderEndpoint:					getOrderEndpoint,
		CreateOrderEndpoint:				createOrderEndpoint,
		PutOrderEndpoint:					putOrderEndpoint,
		SearchOrdersEndpoint:				searchOrdersEndpoint,
		AnswerOrderEndpoint:				answerOrderEndpoint,

		/* Pro */
		RegisterProEndpoint:				registerProEndpoint,
	}

	/* Mechanical domain */
	errc := make(chan error)

	/* interrupt handler */
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	/* debug listener */
	go func() {
		logger := log.With(logger, "transport", "debug")

		m := http.NewServeMux()
		m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		m.Handle("/metrics", promhttp.Handler())

		logger.Log("addr", *debugAddr)
		errc <- http.ListenAndServe(*debugAddr, m)
	}()

	/* HTTP transport */
	go func() {
		logger := log.With(logger, "transport", "HTTP")
		h := svcpayment.MakeHTTPHandler(endpoints, tracer, logger)
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, h)
	}()

	/* Run */
	logger.Log("exit", <-errc)
}
