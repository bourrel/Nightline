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
	csvcevent "svcevent/client"
	csvcpayment "svcpayment/client"

	"svcapi"
)

func main() {
	var env = os.Getenv("GOLANG_ENV")
	var (
		debugAddr       = flag.String("debug.addr", ":8033", "Debug and metrics listen address")
		httpAddr        = flag.String("http.addr", ":8043", "HTTP listen address")
		zipkinAddr      = flag.String("zipkin.addr", "", "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")
		zipkinKafkaAddr = flag.String("zipkin.kafka.addr", "", "Enable Zipkin tracing via a Kafka server host:port")
		appdashAddr     = flag.String("appdash.addr", "", "Enable Appdash tracing via an Appdash server host:port")
		lightstepToken  = flag.String("lightstep.token", "", "Enable LightStep tracing via a LightStep access token")
	)
	flag.Parse()

	/* Logger */
	var logger log.Logger
	{
		slackWriter, err := svcapi.DefineSlackWriter()

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
	logger.Log("msg", "[SVCAPI BEGIN]")
	defer logger.Log("msg", "[SVCAPI END]")

	/* Authentication */
	// svcapi.InitKey()

	/* Metrics */
	var createSoiree_all metrics.Counter
	{
		// Business level metrics.
		createSoiree_all = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "svcapi",
			Name:      "soirees_created",
			Help:      "Total count of soirees summed via the createSoiree method.",
		}, []string{})

	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "svcapi",
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcapi"),
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcapi"),
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
	var service svcapi.IService
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

		svcpayment, err := csvcpayment.New("localhost:8047", tracer, logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}

		service = svcapi.NewService(svcdb, svcevent, svcpayment)
		service = svcapi.ServiceLoggingMiddleware(logger)(service)
		service = svcapi.ServiceAuthenticationMiddleware()(service)
		service = svcapi.ServiceInstrumentingMiddleware(
			createSoiree_all,
		)(service)
	}

	/* Endpoints domain */
	// Authentication
	loginEndpoint := svcapi.BuildLoginEndpoint(service, logger, tracer, duration)
	registerEndpoint := svcapi.BuildRegisterEndpoint(service, logger, tracer, duration)

	// Users
	searchUsersEndpoint := svcapi.BuildSearchUsersEndpoint(service, logger, tracer, duration)
	getUserEndpoint := svcapi.BuildGetUserEndpoint(service, logger, tracer, duration)
	getUserSuccessEndpoint := svcapi.BuildGetUserSuccessEndpoint(service, logger, tracer, duration)
	getUserFriendsEndpoint := svcapi.BuildGetUserFriendsEndpoint(service, logger, tracer, duration)
	getUserPreferencesEndpoint := svcapi.BuildGetUserPreferencesEndpoint(service, logger, tracer, duration)
	updatePreferencesEndpoint := svcapi.BuildUpdatePreferencesEndpoint(service, logger, tracer, duration)
	updateUserEndpoint := svcapi.BuildUpdateUserEndpoint(service, logger, tracer, duration)
	updateStripeUserEndpoint := svcapi.BuildUpdateStripeUserEndpoint(service, logger, tracer, duration)
	searchFriendsEndpoint := svcapi.BuildSearchFriendsEndpoint(service, logger, tracer, duration)
	getRecommendationEndpoint := svcapi.BuildGetRecommendationEndpoint(service, logger, tracer, duration)

	// Establishment
	searchEstablishmentsEndpoint := svcapi.BuildSearchEstablishmentsEndpoint(service, logger, tracer, duration)
	getAllEstablishmentsEndpoint := svcapi.BuildGetAllEstablishmentsEndpoint(service, logger, tracer, duration)
	getEstablishmentTypesEndpoint := svcapi.BuildGetEstablishmentTypesEndpoint(service, logger, tracer, duration)
	getEstablishmentEndpoint := svcapi.BuildGetEstablishmentEndpoint(service, logger, tracer, duration)
	getMenuFromEstablishmentEndpoint := svcapi.BuildGetMenuFromEstablishmentEndpoint(service, logger, tracer, duration)
	rateEstablishmentEndpoint := svcapi.BuildRateEstablishmentEndpoint(service, logger, tracer, duration)

	// Soiree
	getSoireeFromEstablishmentEndpoint := svcapi.BuildGetSoireeFromEstablishmentEndpoint(service, logger, tracer, duration)
	getSoireesFromEstablishmentsEndpoint := svcapi.BuildGetSoireesFromEstablishmentsEndpoint(service, logger, tracer, duration)
	soireeJoinEndpoint := svcapi.BuildSoireeJoinEndpoint(service, logger, tracer, duration)
	soireeOrderEndpoint := svcapi.BuildSoireeOrderEndpoint(service, logger, tracer, duration)
	soireeLeaveEndpoint := svcapi.BuildSoireeLeaveEndpoint(service, logger, tracer, duration)
	getMenuFromSoireeEndpoint := svcapi.BuildGetMenuFromSoireeEndpoint(service, logger, tracer, duration)
	inviteFriendEndpoint := svcapi.BuildInviteFriendEndpoint(service, logger, tracer, duration)
	getUsersInvitationsEndpoint := svcapi.BuildGetUsersInvitationsEndpoint(service, logger, tracer, duration)
	invitationAcceptEndpoint := svcapi.BuildInvitationAcceptEndpoint(service, logger, tracer, duration)
	invitationDeclineEndpoint := svcapi.BuildInvitationDeclineEndpoint(service, logger, tracer, duration)

	// Groups
	getGroupEndpoint := svcapi.BuildGetGroupEndpoint(service, logger, tracer, duration)
	createGroupEndpoint := svcapi.BuildCreateGroupEndpoint(service, logger, tracer, duration)
	updateGroupEndpoint := svcapi.BuildUpdateGroupEndpoint(service, logger, tracer, duration)
	groupInviteEndpoint := svcapi.BuildGroupInviteEndpoint(service, logger, tracer, duration)
	groupInvitationAcceptEndpoint := svcapi.BuildGroupInvitationAcceptEndpoint(service, logger, tracer, duration)
	groupInvitationDeclineEndpoint := svcapi.BuildGroupInvitationDeclineEndpoint(service, logger, tracer, duration)
	getUserGroupsEndpoint := svcapi.BuildGetUserGroupsEndpoint(service, logger, tracer, duration)
	deleteGroupEndpoint := svcapi.BuildDeleteGroupEndpoint(service, logger, tracer, duration)
	deleteGroupMemberEndpoint := svcapi.BuildDeleteGroupMemberEndpoint(service, logger, tracer, duration)
	getGroupsInvitationsEndpoint := svcapi.BuildGetGroupsInvitationsEndpoint(service, logger, tracer, duration)

	/* Notification */
	createNotificationEndpoint := svcapi.BuildCreateNotificationEndpoint(service, logger, tracer, duration)

	/* Order */
	getOrderEndpoint := svcapi.BuildGetOrderEndpoint(service, logger, tracer, duration)
	createOrderEndpoint := svcapi.BuildCreateOrderEndpoint(service, logger, tracer, duration)
	answerOrderEndpoint := svcapi.BuildAnswerOrderEndpoint(service, logger, tracer, duration)
	searchOrdersEndpoint := svcapi.BuildSearchOrdersEndpoint(service, logger, tracer, duration)

	endpoints := svcapi.Endpoints{

		// Authentication
		LoginEndpoint:    loginEndpoint,
		RegisterEndpoint: registerEndpoint,

		// Users
		SearchUsersEndpoint:        searchUsersEndpoint,
		SearchFriendsEndpoint:      searchFriendsEndpoint,
		GetUserEndpoint:            getUserEndpoint,
		GetUserSuccessEndpoint:     getUserSuccessEndpoint,
		GetUserFriendsEndpoint:     getUserFriendsEndpoint,
		GetUserPreferencesEndpoint: getUserPreferencesEndpoint,
		UpdateUserEndpoint:         updateUserEndpoint,
		UpdatePreferencesEndpoint:  updatePreferencesEndpoint,
		UpdateStripeUserEndpoint:   updateStripeUserEndpoint,
		GetRecommendationEndpoint:  getRecommendationEndpoint,

		// Establishment
		SearchEstablishmentsEndpoint:         searchEstablishmentsEndpoint,
		GetAllEstablishmentsEndpoint:         getAllEstablishmentsEndpoint,
		GetEstablishmentEndpoint:             getEstablishmentEndpoint,
		GetMenuFromEstablishmentEndpoint:     getMenuFromEstablishmentEndpoint,
		GetSoireeFromEstablishmentEndpoint:   getSoireeFromEstablishmentEndpoint,
		GetSoireesFromEstablishmentsEndpoint: getSoireesFromEstablishmentsEndpoint,
		GetEstablishmentTypesEndpoint:        getEstablishmentTypesEndpoint,
		RateEstablishmentEndpoint:            rateEstablishmentEndpoint,

		// Soiree
		SoireeJoinEndpoint:          soireeJoinEndpoint,
		SoireeOrderEndpoint:         soireeOrderEndpoint,
		SoireeLeaveEndpoint:         soireeLeaveEndpoint,
		GetMenuFromSoireeEndpoint:   getMenuFromSoireeEndpoint,
		InviteFriendEndpoint:        inviteFriendEndpoint,
		GetUsersInvitationsEndpoint: getUsersInvitationsEndpoint,
		InvitationAcceptEndpoint:    invitationAcceptEndpoint,
		InvitationDeclineEndpoint:   invitationDeclineEndpoint,

		// Groups
		GetGroupEndpoint:               getGroupEndpoint,
		CreateGroupEndpoint:            createGroupEndpoint,
		UpdateGroupEndpoint:            updateGroupEndpoint,
		GroupInviteEndpoint:            groupInviteEndpoint,
		GroupInvitationAcceptEndpoint:  groupInvitationAcceptEndpoint,
		GroupInvitationDeclineEndpoint: groupInvitationDeclineEndpoint,
		GetUserGroupsEndpoint:          getUserGroupsEndpoint,
		DeleteGroupEndpoint:            deleteGroupEndpoint,
		DeleteGroupMemberEndpoint:      deleteGroupMemberEndpoint,
		GetGroupsInvitationsEndpoint:   getGroupsInvitationsEndpoint,

		/* Notification */
		CreateNotificationEndpoint: createNotificationEndpoint,

		/* Order */
		GetOrderEndpoint: getOrderEndpoint,
		CreateOrderEndpoint: createOrderEndpoint,
		AnswerOrderEndpoint: answerOrderEndpoint,
		SearchOrdersEndpoint: searchOrdersEndpoint,
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
		h := svcapi.MakeHTTPHandler(endpoints, tracer, logger)
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, h)
	}()

	/* Run */
	logger.Log("exit", <-errc)
}
