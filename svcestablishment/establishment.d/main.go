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
	"time"

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

	csvcdb "svcdb/client"
	csvcpayment "svcpayment/client"
	csvcsoiree "svcsoiree/client"

	"svcestablishment"
)

func main() {
	var env = os.Getenv("GOLANG_ENV")

	var (
		debugAddr       = flag.String("debug.addr", ":8036", "Debug and metrics listen address")
		httpAddr        = flag.String("http.addr", ":8046", "HTTP listen address")
		zipkinAddr      = flag.String("zipkin.addr", "", "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")
		zipkinKafkaAddr = flag.String("zipkin.kafka.addr", "", "Enable Zipkin tracing via a Kafka server host:port")
		appdashAddr     = flag.String("appdash.addr", "", "Enable Appdash tracing via an Appdash server host:port")
		lightstepToken  = flag.String("lightstep.token", "", "Enable LightStep tracing via a LightStep access token")
	)
	flag.Parse()

	/* Logger */
	var logger log.Logger
	{
		slackWriter, err := svcestablishment.DefineSlackWriter()

		if env == "production" && err == nil {
			logger = log.NewJSONLogger(slackWriter)
		} else {
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
	logger.Log("msg", "[SVCESTABLISHMENT BEGIN]")
	defer logger.Log("msg", "[SVCESTABLISHMENT END]")

	/* Authentication */
	// svcestablishment.InitKey()

	/* Geocoding */
	svcestablishment.InitMapQuest()

	/* Metrics */
	var createSoiree_all metrics.Counter
	{
		// Business level metrics.
		createSoiree_all = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "svcestablishment",
			Name:      "soirees_created",
			Help:      "Total count of soirees summed via the createSoiree method.",
		}, []string{})

	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "svcestablishment",
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcestablishment"),
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcestablishment"),
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
	var service svcestablishment.IService
	{
		svcdb, err := csvcdb.New("localhost:8044", tracer, logger)
		svcsoiree, err := csvcsoiree.New("localhost:8045", tracer, logger)
		svcpayment, err := csvcpayment.New("localhost:8047", tracer, logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}

		service = svcestablishment.NewService(svcdb, svcsoiree, svcpayment)
		service = svcestablishment.ServiceLoggingMiddleware(logger)(service)
		service = svcestablishment.ServiceInstrumentingMiddleware(
			createSoiree_all,
		)(service)
	}

	/* Endpoints domain */
	createSoireeEndpoint := svcestablishment.BuildCreateSoireeEndpoint(service, logger, tracer, duration)
	deliverOrderEndpoint := svcestablishment.BuildDeliverOrderEndpoint(service, logger, tracer, duration)
	getOrderEndpoint := svcestablishment.BuildGetOrderEndpoint(service, logger, tracer, duration)
	getOrdersBySoireeEndpoint := svcestablishment.BuildGetOrdersBySoireeEndpoint(service, logger, tracer, duration)
	searchOrdersEndpoint := svcestablishment.BuildSearchOrdersEndpoint(service, logger, tracer, duration)
	putOrderEndpoint := svcestablishment.BuildPutOrderEndpoint(service, logger, tracer, duration)
	getSoireesEndpoint := svcestablishment.BuildGetSoireesEndpoint(service, logger, tracer, duration)
	getStatEndpoint := svcestablishment.BuildGetStatEndpoint(service, logger, tracer, duration)
	loginProEndpoint := svcestablishment.BuildLoginProEndpoint(service, logger, tracer, duration)
	registerProEndpoint := svcestablishment.BuildRegisterProEndpoint(service, logger, tracer, duration)
	createEstabEndpoint := svcestablishment.BuildCreateEstabEndpoint(service, logger, tracer, duration)
	updateEstabEndpoint := svcestablishment.BuildUpdateEstabEndpoint(service, logger, tracer, duration)
	deleteEstabEndpoint := svcestablishment.BuildDeleteEstabEndpoint(service, logger, tracer, duration)
	updateProEndpoint := svcestablishment.BuildUpdateProEndpoint(service, logger, tracer, duration)
	createMenuEndpoint := svcestablishment.BuildCreateMenuEndpoint(service, logger, tracer, duration)
	createConsoEndpoint := svcestablishment.BuildCreateConsoEndpoint(service, logger, tracer, duration)
	getConsoEndpoint := svcestablishment.BuildGetConsoEndpoint(service, logger, tracer, duration)
	GetConsoByOrderIDEndpoint := svcestablishment.BuildGetConsoByOrderIDEndpoint(service, logger, tracer, duration)
	getMenuEndpoint := svcestablishment.BuildGetMenuEndpoint(service, logger, tracer, duration)
	getEstablishmentTypeEndpoint := svcestablishment.BuildGetEstablishmentTypeEndpoint(service, logger, tracer, duration)
	getSoireeOrdersEndpoint := svcestablishment.BuildGetSoireeOrdersEndpoint(service, logger, tracer, duration)
	getProEstablishmentEndpoint := svcestablishment.BuildGetProEstablishmentsEndpoint(service, logger, tracer, duration)
	deleteSoireeEndpoint := svcestablishment.BuildDeleteSoireeEndpoint(service, logger, tracer, duration)
	getAnalysePEndpoint := svcestablishment.BuildGetAnalysePEndpoint(service, logger, tracer, duration)

	endpoints := svcestablishment.Endpoints{
		CreateSoireeEndpoint:         createSoireeEndpoint,
		DeliverOrderEndpoint:         deliverOrderEndpoint,
		GetOrderEndpoint:             getOrderEndpoint,
		GetOrdersBySoireeEndpoint:    getOrdersBySoireeEndpoint,
		SearchOrdersEndpoint:         searchOrdersEndpoint,
		PutOrderEndpoint:             putOrderEndpoint,
		GetSoireesEndpoint:           getSoireesEndpoint,
		GetStatEndpoint:              getStatEndpoint,
		LoginProEndpoint:             loginProEndpoint,
		RegisterProEndpoint:          registerProEndpoint,
		CreateEstabEndpoint:          createEstabEndpoint,
		UpdateEstabEndpoint:          updateEstabEndpoint,
		UpdateProEndpoint:            updateProEndpoint,
		CreateMenuEndpoint:           createMenuEndpoint,
		CreateConsoEndpoint:          createConsoEndpoint,
		GetConsoEndpoint:             getConsoEndpoint,
		GetConsoByOrderIDEndpoint:    GetConsoByOrderIDEndpoint,
		GetMenuEndpoint:              getMenuEndpoint,
		GetEstablishmentTypeEndpoint: getEstablishmentTypeEndpoint,
		GetSoireeOrdersEndpoint:      getSoireeOrdersEndpoint,
		GetProEstablishmentsEndpoint: getProEstablishmentEndpoint,
		DeleteEstabEndpoint:          deleteEstabEndpoint,
		DeleteSoireeEndpoint:         deleteSoireeEndpoint,
		GetAnalysePEndpoint:		  getAnalysePEndpoint,
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
		// m.Handle("/", svcestablishment.AccessControl(m))

		logger.Log("addr", *debugAddr)
		errc <- http.ListenAndServe(*debugAddr, m)
	}()

	/* HTTP transport */
	go func() {
		logger := log.With(logger, "transport", "HTTP")
		h := svcestablishment.MakeHTTPHandler(endpoints, tracer, logger)
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, h)
	}()

	/* Run */
	logger.Log("exit", <-errc)
}
