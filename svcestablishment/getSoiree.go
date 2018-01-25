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
func (s Service) GetSoirees(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	soirees, err := s.svcdb.GetSoireesByEstablishment(ctx, estabID)
	return soirees, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetSoireesRequest struct {
	EstabID int64 `json:"id"`
}

type GetSoireesResponse struct {
	Soirees []svcdb.Soiree `json:"soirees"`
}

func GetSoireesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetSoireesRequest)
		soirees, err := svc.GetSoirees(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetSoireesEndpoint : ", err.Error())
			return nil, err
		}
		return GetSoireesResponse{Soirees: soirees}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetSoireesRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesRequest 1 : ", err.Error())
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetSoireesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetSoireesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetSoireesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabID}/soirees").Handler(httptransport.NewServer(
		endpoints.GetSoireesEndpoint,
		DecodeHTTPGetSoireesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoirees", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoirees(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	soirees, err := mw.next.GetSoirees(ctx, estabID)

	mw.logger.Log(
		"method", "GetSoirees",
		"estabID", estabID,
		"response", soirees,
		"took", time.Since(time.Now()),
	)
	return soirees, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetSoirees(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	return mw.next.GetSoirees(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoirees(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	return mw.next.GetSoirees(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetSoireesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSoirees")
		csLogger := log.With(logger, "method", "GetSoirees")

		csEndpoint = GetSoireesEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSoirees")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSoirees(ctx context.Context, etID int64) ([]svcdb.Soiree, error) {
	var soirees []svcdb.Soiree

	request := GetSoireesRequest{EstabID: etID}
	response, err := e.GetSoireesEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetSoirees : ", err.Error())
		return soirees, err
	}
	soirees = response.(GetSoireesResponse).Soirees
	return soirees, err
}

func EncodeHTTPGetSoireesRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(GetSoireesRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetSoireesRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSoirees(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{EstabID}/soirees"),
		EncodeHTTPGetSoireesRequest,
		DecodeHTTPGetSoireesResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetSoirees")(ceEndpoint)
	return ceEndpoint, nil
}
