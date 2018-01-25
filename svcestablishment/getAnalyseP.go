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
func (s Service) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error) {
	var analyses []svcdb.AnalyseP
	tmpAnalyses, err := s.svcdb.GetAnalyseP(ctx, estabID, soireeID)
	analyses = append(analyses, tmpAnalyses...)
	return analyses, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetAnalysePRequest struct {
	EstabID int64 `json:"estabID"`
	SoireeID int64 `json:"soireeID"`
}

type GetAnalysePResponse struct {
	Analyses []svcdb.AnalyseP `json:"analyses"`
	Err   error       `json:"err,omitempty"`
}

func GetAnalysePEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetAnalysePRequest)
		analyses, err := svc.GetAnalyseP(ctx, req.EstabID, req.SoireeID)
		return GetAnalysePResponse{Analyses: analyses, Err: err}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetAnalysePRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetAnalysePRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["estabID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).EstabID = estabID
	soireeID, err := strconv.ParseInt(mux.Vars(r)["soireeID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).SoireeID = soireeID

	return request, nil
}

func DecodeHTTPGetAnalysePResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetAnalysePResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetAnalysePHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/analyses/Population/{estabID:[0-9]+}/{soireeID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetAnalysePEndpoint,
		DecodeHTTPGetAnalysePRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetAnalyseP", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error) {
	analyses, err := mw.next.GetAnalyseP(ctx, estabID, soireeID)

	mw.logger.Log(
		"method", "GetAnalyseP",
		"response", analyses,
		"took", time.Since(time.Now()),
	)
	return analyses, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error) {
	return mw.next.GetAnalyseP(ctx, estabID, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error) {
	v, err := mw.next.GetAnalyseP(ctx, estabID, soireeID)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetAnalysePEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetAnalyseP")
		csLogger := log.With(logger, "method", "GetAnalyseP")

		csEndpoint = GetAnalysePEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetAnalyseP")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error) {
	var analyses []svcdb.AnalyseP

	request := GetAnalysePRequest{EstabID: estabID, SoireeID: soireeID}
	response, err := e.GetAnalysePEndpoint(ctx, request)
	if err != nil {
		return analyses, err
	}
	analyses = response.(GetAnalysePResponse).Analyses
	return analyses, response.(GetAnalysePResponse).Err
}

func EncodeHTTPGetAnalysePRequest(ctx context.Context, r *http.Request, request interface{}) error {
    route := mux.NewRouter()
	estabID := fmt.Sprintf("%v", request.(GetAnalysePRequest).EstabID)
	soireeID := fmt.Sprintf("%v", request.(GetAnalysePRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("estabID", estabID, "soireeID", soireeID)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetAnalyseP(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/analyses/Population/{estabID:[0-9]+}/{soireeID:[0-9]+}"),
		EncodeHTTPGetAnalysePRequest,
		DecodeHTTPGetAnalysePResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetAnalyseP")(ceEndpoint)
	return ceEndpoint, nil
}
