package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)

/*************** Service ***************/
func (s Service) GetProEstablishments(ctx context.Context, estabID int64) ([]svcdb.Establishment, error) {
	estabs, err := s.svcdb.GetProEstablishments(ctx, estabID)
	return estabs, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getEstablishmentRequest struct {
	estabID int64 `json:"id"`
}

type getEstablishmentResponse struct {
	Establishments []svcdb.Establishment `json:"establishments"`
}

func GetProEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentRequest)
		establishments, err := svc.GetProEstablishments(ctx, req.estabID)
		return getEstablishmentResponse{Establishments: establishments}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentRequest

	estabID, err := strconv.ParseInt(mux.Vars(r)["estabID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).estabID = estabID

	return request, nil
}

func DecodeHTTPGetProEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetProEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/pros/{estabID}/establishments").Handler(httptransport.NewServer(
		endpoints.GetProEstablishmentsEndpoint,
		DecodeHTTPGetProEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetProEstablishments", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetProEstablishments(ctx context.Context, estabID int64) ([]svcdb.Establishment, error) {
	Establishment, err := mw.next.GetProEstablishments(ctx, estabID)

	mw.logger.Log(
		"method", "getEstablishment",
		"estabID", estabID,
		"response", Establishment,
		"took", time.Since(time.Now()),
	)
	return Establishment, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetProEstablishments(ctx context.Context, estabID int64) ([]svcdb.Establishment, error) {
	return mw.next.GetProEstablishments(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetProEstablishments(ctx context.Context, estabID int64) ([]svcdb.Establishment, error) {
	return mw.next.GetProEstablishments(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetProEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetProEstablishments")
		csLogger := log.With(logger, "method", "GetProEstablishments")

		csEndpoint = GetProEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetProEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetProEstablishments(ctx context.Context, etID int64) ([]svcdb.Establishment, error) {
	var Establishment []svcdb.Establishment

	request := getEstablishmentRequest{estabID: etID}
	response, err := e.GetProEstablishmentsEndpoint(ctx, request)
	if err != nil {
		return Establishment, err
	}
	Establishment = response.(getEstablishmentResponse).Establishments
	return Establishment, err
}

func EncodeHTTPGetProEstablishmentsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentRequest).estabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("estabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetProEstablishments(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{estabID}/Establishments"),
		EncodeHTTPGetProEstablishmentsRequest,
		DecodeHTTPGetProEstablishmentsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetProEstablishments")(ceEndpoint)
	return ceEndpoint, nil
}
