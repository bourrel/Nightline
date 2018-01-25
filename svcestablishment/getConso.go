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
func (s Service) GetConso(ctx context.Context, estabID int64) ([]svcdb.Conso, error) {
	consos, err := s.svcdb.GetEstablishmentConsos(ctx, estabID)
	return consos, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getConsoRequest struct {
	EstabID int64 `json:"id"`
}

type getConsoResponse struct {
	Consos []svcdb.Conso `json:"consos"`
}

func GetConsoEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConsoRequest)
		consos, err := svc.GetConso(ctx, req.EstabID)
		return getConsoResponse{Consos: consos}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConsoRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getConsoRequest

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetConsoResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConsoResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetConsoHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabID}/consos").Handler(httptransport.NewServer(
		endpoints.GetConsoEndpoint,
		DecodeHTTPGetConsoRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConso", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConso(ctx context.Context, estabID int64) ([]svcdb.Conso, error) {
	conso, err := mw.next.GetConso(ctx, estabID)

	mw.logger.Log(
		"method", "getConso",
		"estabID", estabID,
		"response", conso,
		"took", time.Since(time.Now()),
	)
	return conso, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetConso(ctx context.Context, estabID int64) ([]svcdb.Conso, error) {
	return mw.next.GetConso(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConso(ctx context.Context, estabID int64) ([]svcdb.Conso, error) {
	return mw.next.GetConso(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetConsoEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConso")
		csLogger := log.With(logger, "method", "GetConso")

		csEndpoint = GetConsoEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConso")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConso(ctx context.Context, etID int64) ([]svcdb.Conso, error) {
	var conso []svcdb.Conso

	request := getConsoRequest{EstabID: etID}
	response, err := e.GetConsoEndpoint(ctx, request)
	if err != nil {
		return conso, err
	}
	conso = response.(getConsoResponse).Consos
	return conso, err
}

func EncodeHTTPGetConsoRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getConsoRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConso(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{EstabID}/consos"),
		EncodeHTTPGetConsoRequest,
		DecodeHTTPGetConsoResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetConso")(ceEndpoint)
	return ceEndpoint, nil
}
