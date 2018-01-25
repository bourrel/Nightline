package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"svcevent"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log/term"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
)

func main() {
	var env = os.Getenv("GOLANG_ENV")
	var (
		debugAddr       = flag.String("debug.addr", ":8039", "Debug and metrics listen address")
		httpAddr        = flag.String("http.addr", ":8049", "HTTP listen address")
		zipkinAddr      = flag.String("zipkin.addr", "", "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")
		zipkinKafkaAddr = flag.String("zipkin.kafka.addr", "", "Enable Zipkin tracing via a Kafka server host:port")
		appdashAddr     = flag.String("appdash.addr", "", "Enable Appdash tracing via an Appdash server host:port")
		lightstepToken  = flag.String("lightstep.token", "", "Enable LightStep tracing via a LightStep access token")
	)

	flag.Parse()

	/* Logger */
	var logger log.Logger
	{
		slackWriter, err := svcevent.DefineSlackWriter()

		if env == "production" && err == nil {
			logger = log.NewJSONLogger(slackWriter)
		} else {
			fmt.Println(" Fail : ", err)
			colorFn := func(keyvals ...interface{}) term.FgBgColor {
				for i := 0; i < len(keyvals); i++ {
					if _, ok := keyvals[i].(error); ok {
						return term.FgBgColor{Fg: term.Red}
					}
				}
				return term.FgBgColor{}
			}
			logger = term.NewLogger(os.Stdout, log.NewJSONLogger, colorFn)
		}
		logger = level.NewFilter(logger, level.AllowInfo())
		logger = level.NewInjector(logger, level.InfoValue())
		logger = log.With(logger, "timestamp", log.TimestampFormat(time.Now().UTC, time.RFC3339))
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	logger.Log("msg", "[SVCEVENT BEGIN]")
	defer logger.Log("msg", "[SVCEVENT END]")

	//addresses of available kafka brokers
	brokers := []string{"localhost:9092"}
	//setup relevant config info
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)

	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	/* Metrics */
	var createSoiree_all metrics.Counter
	{
		// Business level metrics.
		createSoiree_all = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "svcevent",
			Name:      "soirees_created",
			Help:      "Total count of soirees summed via the createSoiree method.",
		}, []string{})

	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "svcevent",
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

			// endpoint typically looks like: http://zipkinhost:9411/event/v1/spans
			collector, err := zipkin.NewHTTPCollector(*zipkinAddr)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
			defer collector.Close()

			tracer, err = zipkin.NewTracer(
				zipkin.NewRecorder(collector, false, "localhost:80", "svcevent"),
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcevent"),
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
	var service svcevent.IService
	{
		service = svcevent.NewService()
		service = svcevent.ServiceLoggingMiddleware(logger)(service)
		service = svcevent.ServiceInstrumentingMiddleware(
			createSoiree_all,
		)(service)
	}

	/* Endpoints domain */
	pushEndpoint := svcevent.BuildPushEndpoint(service, logger, tracer, duration)

	endpoints := svcevent.Endpoints{
		PushEndpoint: pushEndpoint,
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
		h := svcevent.MakeHTTPHandler(endpoints, tracer, logger, producer)
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, h)
	}()

	/* Run */
	logger.Log("exit", <-errc)
}
