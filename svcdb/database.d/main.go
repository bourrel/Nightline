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

	"svcdb"
)

func main() {
	var (
		debugAddr       = flag.String("debug.addr", ":8034", "Debug and metrics listen address")
		httpAddr        = flag.String("http.addr", ":8044", "HTTP listen address")
		zipkinAddr      = flag.String("zipkin.addr", "", "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")
		zipkinKafkaAddr = flag.String("zipkin.kafka.addr", "", "Enable Zipkin tracing via a Kafka server host:port")
		appdashAddr     = flag.String("appdash.addr", "", "Enable Appdash tracing via an Appdash server host:port")
		lightstepToken  = flag.String("lightstep.token", "", "Enable LightStep tracing via a LightStep access token")
	)
	flag.Parse()

	/* Database */
	svcdb.StartDriver()
	defer svcdb.CloseDriver()

	/* Logger */
	var logger log.Logger
	{
		logger = log.NewJSONLogger(os.Stdout)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	logger.Log("msg", "[SVCDB BEGIN]")
	defer logger.Log("msg", "[SVCDB END]")

	/* Metrics */
	var ints metrics.Counter
	{
		// Business level metrics.
		ints = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "svcdb",
			Name:      "integers_summed",
			Help:      "Total count of users summed via the createUser method.",
		}, []string{})

	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "svcdb",
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcdb"),
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
				zipkin.NewRecorder(collector, false, "localhost:80", "svcdb"),
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
	var service svcdb.IService
	{
		service = svcdb.NewService()
		service = svcdb.ServiceLoggingMiddleware(logger)(service)
		service = svcdb.ServiceInstrumentingMiddleware(ints)(service)
	}

	/* Endpoints domain */
	/* Pro */
	getProEndpoint := svcdb.BuildGetProEndpoint(service, logger, tracer, duration)
	getProByIDStripeEndpoint := svcdb.BuildGetProByIDStripeEndpoint(service, logger, tracer, duration)
	getProByIDEndpoint := svcdb.BuildGetProByIDEndpoint(service, logger, tracer, duration)
	getProBySoireeEndpoint := svcdb.BuildGetProBySoireeEndpoint(service, logger, tracer, duration)
	createProEndpoint := svcdb.BuildCreateProEndpoint(service, logger, tracer, duration)
	updateProEndpoint := svcdb.BuildUpdateProEndpoint(service, logger, tracer, duration)
	getProEstablishmentsEndpoint := svcdb.BuildGetProEstablishmentsEndpoint(service, logger, tracer, duration)

	/* User */
	createUserEndpoint := svcdb.BuildCreateUserEndpoint(service, logger, tracer, duration)
	updateUserEndpoint := svcdb.BuildUpdateUserEndpoint(service, logger, tracer, duration)
	getUserByIDEndpoint := svcdb.BuildGetUserByIDEndpoint(service, logger, tracer, duration)
	getUserEndpoint := svcdb.BuildGetUserEndpoint(service, logger, tracer, duration)
	getUserProfileEndpoint := svcdb.BuildGetUserProfileEndpoint(service, logger, tracer, duration)
	getUserPreferencesEndpoint := svcdb.BuildGetUserPreferencesEndpoint(service, logger, tracer, duration)
	getUserSuccessEndpoint := svcdb.BuildGetUserSuccessEndpoint(service, logger, tracer, duration)
	updatePreferenceEndpoint := svcdb.BuildUpdatePreferenceEndpoint(service, logger, tracer, duration)
	searchUsersEndpoint := svcdb.BuildSearchUsersEndpoint(service, logger, tracer, duration)
	getRecommendationEndpoint := svcdb.BuildGetRecommendationEndpoint(service, logger, tracer, duration)

	/* Friends */
	getUserFriendsEndpoint := svcdb.BuildGetUserFriendsEndpoint(service, logger, tracer, duration)
	inviteFriendEndpoint := svcdb.BuildInviteFriendEndpoint(service, logger, tracer, duration)
	getUsersInvitationsEndpoint := svcdb.BuildGetUsersInvitationsEndpoint(service, logger, tracer, duration)
	invitationAcceptEndpoint := svcdb.BuildInvitationAcceptEndpoint(service, logger, tracer, duration)
	invitationDeclineEndpoint := svcdb.BuildInvitationDeclineEndpoint(service, logger, tracer, duration)
	usersConnectedEndpoint := svcdb.BuildUsersConnectedEndpoint(service, logger, tracer, duration)
	searchFriendsEndpoint := svcdb.BuildSearchFriendsEndpoint(service, logger, tracer, duration)

	/* Establishment */
	createEstablishmentEndpoint := svcdb.BuildCreateEstablishmentEndpoint(service, logger, tracer, duration)
	getEstablishmentsEndpoint := svcdb.BuildGetEstablishmentsEndpoint(service, logger, tracer, duration)
	getEstablishmentEndpoint := svcdb.BuildGetEstablishmentEndpoint(service, logger, tracer, duration)
	getEstablishmentFromMenuEndpoint := svcdb.BuildGetEstablishmentFromMenuEndpoint(service, logger, tracer, duration)
	getMenuFromEstablishmentEndpoint := svcdb.BuildGetMenuFromEstablishmentEndpoint(service, logger, tracer, duration)
	getEstablishmentSoireeEndpoint := svcdb.BuildGetEstablishmentSoireeEndpoint(service, logger, tracer, duration)
	searchEstablishmentsEndpoint := svcdb.BuildSearchEstablishmentsEndpoint(service, logger, tracer, duration)
	updateEstablishmentEndpoint := svcdb.BuildUpdateEstablishmentEndpoint(service, logger, tracer, duration)
	getEstablishmentTypesEndpoint := svcdb.BuildGetEstablishmentTypesEndpoint(service, logger, tracer, duration)
	getEstablishmentTypeEndpoint := svcdb.BuildGetEstablishmentTypeEndpoint(service, logger, tracer, duration)
	rateEstablishmentEndpoint := svcdb.BuildRateEstablishmentEndpoint(service, logger, tracer, duration)
	deleteEstabEndpoint := svcdb.BuildDeleteEstabEndpoint(service, logger, tracer, duration)

	/* Soiree */
	userJoinSoireeEndpoint := svcdb.BuildUserJoinSoireeEndpoint(service, logger, tracer, duration)
	getConsoByIDEndpoint := svcdb.BuildGetConsoByIDEndpoint(service, logger, tracer, duration)
	getSoireeByIDEndpoint := svcdb.BuildGetSoireeByIDEndpoint(service, logger, tracer, duration)
	userLeaveSoireeEndpoint := svcdb.BuildUserLeaveSoireeEndpoint(service, logger, tracer, duration)
	getSoireesByEstablishmentEndpoint := svcdb.BuildGetSoireesByEstablishmentEndpoint(service, logger, tracer, duration)
	createSoireeEndpoint := svcdb.BuildCreateSoireeEndpoint(service, logger, tracer, duration)
	getConnectedFriendsEndpoint := svcdb.BuildGetConnectedFriendsEndpoint(service, logger, tracer, duration)
	deleteSoireeEndpoint := svcdb.BuildDeleteSoireeEndpoint(service, logger, tracer, duration)

	/* Order */
	getOrderEndpoint := svcdb.BuildGetOrderEndpoint(service, logger, tracer, duration)
	searchOrdersEndpoint := svcdb.BuildSearchOrdersEndpoint(service, logger, tracer, duration)
	createOrderEndpoint := svcdb.BuildCreateOrderEndpoint(service, logger, tracer, duration)
	putOrderEndpoint := svcdb.BuildPutOrderEndpoint(service, logger, tracer, duration)
	updateOrderReferenceEndpoint := svcdb.BuildUpdateOrderReferenceEndpoint(service, logger, tracer, duration)

	userOrderEndpoint := svcdb.BuildUserOrderEndpoint(service, logger, tracer, duration)
	getConsoByOrderIDEndpoint := svcdb.BuildGetConsoByOrderIDEndpoint(service, logger, tracer, duration)
	answerOrderEndpoint := svcdb.BuildAnswerOrderEndpoint(service, logger, tracer, duration)
	failOrderEndpoint := svcdb.BuildFailOrderEndpoint(service, logger, tracer, duration)

	/* Menu */
	getEstablishmentConsosEndpoint := svcdb.BuildGetEstablishmentConsosEndpoint(service, logger, tracer, duration)
	getEstablishmentMenusEndpoint := svcdb.BuildGetEstablishmentMenusEndpoint(service, logger, tracer, duration)
	getMenuConsosEndpoint := svcdb.BuildGetMenuConsosEndpoint(service, logger, tracer, duration)
	createMenuEndpoint := svcdb.BuildCreateMenuEndpoint(service, logger, tracer, duration)
	createConsoEndpoint := svcdb.BuildCreateConsoEndpoint(service, logger, tracer, duration)
	getMenuFromSoireeEndpoint := svcdb.BuildGetMenuFromSoireeEndpoint(service, logger, tracer, duration)

	/* Groups */
	getGroupEndpoint := svcdb.BuildGetGroupEndpoint(service, logger, tracer, duration)
	createGroupEndpoint := svcdb.BuildCreateGroupEndpoint(service, logger, tracer, duration)
	updateGroupEndpoint := svcdb.BuildUpdateGroupEndpoint(service, logger, tracer, duration)
	groupInviteEndpoint := svcdb.BuildGroupInviteEndpoint(service, logger, tracer, duration)
	groupInvitationDeclineEndpoint := svcdb.BuildGroupInvitationDeclineEndpoint(service, logger, tracer, duration)
	groupInvitationAcceptEndpoint := svcdb.BuildGroupInvitationAcceptEndpoint(service, logger, tracer, duration)
	getUserGroupsEndpoint := svcdb.BuildGetUserGroupsEndpoint(service, logger, tracer, duration)
	deleteGroupEndpoint := svcdb.BuildDeleteGroupEndpoint(service, logger, tracer, duration)
	deleteGroupMemberEndpoint := svcdb.BuildDeleteGroupMemberEndpoint(service, logger, tracer, duration)
	getGroupsInvitationsEndpoint := svcdb.BuildGetGroupsInvitationsEndpoint(service, logger, tracer, duration)

	/* Conversation */
	getLastMessagesEndpoint := svcdb.BuildGetLastMessagesEndpoint(service, logger, tracer, duration)
	createMessageEndpoint := svcdb.BuildCreateMessageEndpoint(service, logger, tracer, duration)
	getConversationByIDEndpoint := svcdb.BuildGetConversationByIDEndpoint(service, logger, tracer, duration)

	getNodeTypeEndpoint := svcdb.BuildGetNodeTypeEndpoint(service, logger, tracer, duration)
	addSuccessEndpoint := svcdb.BuildAddSuccessEndpoint(service, logger, tracer, duration)
	getSuccessByValueEndpoint := svcdb.BuildGetSuccessByValueEndpoint(service, logger, tracer, duration)

	/* Analyse */
	getAnalysePEndpoint := svcdb.BuildGetAnalysePEndpoint(service, logger, tracer, duration)

	endpoints := svcdb.Endpoints{
		/* Pro */
		CreateProEndpoint:            createProEndpoint,
		GetProEndpoint:               getProEndpoint,
		GetProByIDStripeEndpoint:     getProByIDStripeEndpoint,
		GetProByIDEndpoint:           getProByIDEndpoint,
		GetProBySoireeEndpoint:       getProBySoireeEndpoint,
		UpdateProEndpoint:            updateProEndpoint,
		GetProEstablishmentsEndpoint: getProEstablishmentsEndpoint,

		/* User */
		CreateUserEndpoint:         createUserEndpoint,
		UpdateUserEndpoint:         updateUserEndpoint,
		GetUserByIDEndpoint:        getUserByIDEndpoint,
		GetUserEndpoint:            getUserEndpoint,
		GetUserProfileEndpoint:     getUserProfileEndpoint,
		GetUserSuccessEndpoint:     getUserSuccessEndpoint,
		GetUserPreferencesEndpoint: getUserPreferencesEndpoint,
		UpdatePreferenceEndpoint:   updatePreferenceEndpoint,
		SearchUsersEndpoint:        searchUsersEndpoint,
		GetRecommendationEndpoint:  getRecommendationEndpoint,

		/* Friends */
		GetUserFriendsEndpoint:      getUserFriendsEndpoint,
		InviteFriendEndpoint:        inviteFriendEndpoint,
		GetUsersInvitationsEndpoint: getUsersInvitationsEndpoint,
		InvitationAcceptEndpoint:    invitationAcceptEndpoint,
		InvitationDeclineEndpoint:   invitationDeclineEndpoint,
		UsersConnectedEndpoint:      usersConnectedEndpoint,
		SearchFriendsEndpoint:       searchFriendsEndpoint,

		/* Establishment */
		CreateEstablishmentEndpoint:      createEstablishmentEndpoint,
		GetEstablishmentsEndpoint:        getEstablishmentsEndpoint,
		SearchEstablishmentsEndpoint:     searchEstablishmentsEndpoint,
		GetEstablishmentEndpoint:         getEstablishmentEndpoint,
		GetEstablishmentFromMenuEndpoint: getEstablishmentFromMenuEndpoint,
		GetMenuFromEstablishmentEndpoint: getMenuFromEstablishmentEndpoint,
		GetEstablishmentSoireeEndpoint:   getEstablishmentSoireeEndpoint,
		UpdateEstablishmentEndpoint:      updateEstablishmentEndpoint,
		GetEstablishmentTypesEndpoint:    getEstablishmentTypesEndpoint,
		GetEstablishmentTypeEndpoint:     getEstablishmentTypeEndpoint,
		RateEstablishmentEndpoint:        rateEstablishmentEndpoint,
		DeleteEstabEndpoint:              deleteEstabEndpoint,

		/* Soiree */
		GetConsoByIDEndpoint:              getConsoByIDEndpoint,
		GetSoireeByIDEndpoint:             getSoireeByIDEndpoint,
		UserLeaveSoireeEndpoint:           userLeaveSoireeEndpoint,
		UserJoinSoireeEndpoint:            userJoinSoireeEndpoint,
		GetSoireesByEstablishmentEndpoint: getSoireesByEstablishmentEndpoint,
		CreateSoireeEndpoint:              createSoireeEndpoint,
		GetConnectedFriendsEndpoint:       getConnectedFriendsEndpoint,
		DeleteSoireeEndpoint:              deleteSoireeEndpoint,

		/* Conso */
		GetEstablishmentConsosEndpoint: getEstablishmentConsosEndpoint,
		GetEstablishmentMenusEndpoint:  getEstablishmentMenusEndpoint,
		GetMenuConsosEndpoint:          getMenuConsosEndpoint,
		CreateMenuEndpoint:             createMenuEndpoint,
		CreateConsoEndpoint:            createConsoEndpoint,
		GetMenuFromSoireeEndpoint:      getMenuFromSoireeEndpoint,

		/* Order */
		GetOrderEndpoint:             getOrderEndpoint,
		SearchOrdersEndpoint:         searchOrdersEndpoint,
		CreateOrderEndpoint:          createOrderEndpoint,
		PutOrderEndpoint:             putOrderEndpoint,
		AnswerOrderEndpoint:          answerOrderEndpoint,
		FailOrderEndpoint:            failOrderEndpoint,
		UpdateOrderReferenceEndpoint: updateOrderReferenceEndpoint,

		GetConsoByOrderIDEndpoint: getConsoByOrderIDEndpoint,
		UserOrderEndpoint:         userOrderEndpoint,

		/* Groups */
		GetGroupEndpoint:               getGroupEndpoint,
		CreateGroupEndpoint:            createGroupEndpoint,
		UpdateGroupEndpoint:            updateGroupEndpoint,
		GroupInviteEndpoint:            groupInviteEndpoint,
		GroupInvitationDeclineEndpoint: groupInvitationDeclineEndpoint,
		GroupInvitationAcceptEndpoint:  groupInvitationAcceptEndpoint,
		GetUserGroupsEndpoint:          getUserGroupsEndpoint,
		DeleteGroupEndpoint:            deleteGroupEndpoint,
		DeleteGroupMemberEndpoint:      deleteGroupMemberEndpoint,
		GetGroupsInvitationsEndpoint:   getGroupsInvitationsEndpoint,

		/* Conversation */
		GetLastMessagesEndpoint:     getLastMessagesEndpoint,
		CreateMessageEndpoint:       createMessageEndpoint,
		GetConversationByIDEndpoint: getConversationByIDEndpoint,

		GetNodeTypeEndpoint:       getNodeTypeEndpoint,
		AddSuccessEndpoint:        addSuccessEndpoint,
		GetSuccessByValueEndpoint: getSuccessByValueEndpoint,

		/* Analyse */
		GetAnalysePEndpoint:	getAnalysePEndpoint,
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
		h := svcdb.MakeHTTPHandler(endpoints, tracer, logger)
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, h)
	}()

	/* Run */
	logger.Log("exit", <-errc)
}
