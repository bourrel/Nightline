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
func (s Service) GetSoireeOrders(ctx context.Context, SoireeID int64) ([]svcdb.Order, error) {
	orders, err := s.svcdb.GetSoireeOrders(ctx, SoireeID)
	return orders, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetSoireeOrdersRequest struct {
	SoireeID int64 `json:"id"`
}

type GetSoireeOrdersResponse struct {
	Orders []svcdb.Order `json:"Orders"`
}

func GetSoireeOrdersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetSoireeOrdersRequest)
		Orders, err := svc.GetSoireeOrders(ctx, req.SoireeID)
		if err != nil {
			fmt.Println("Error GetSoireeOrdersEndpoint : ", err.Error())
			return nil, err
		}
		return GetSoireeOrdersResponse{Orders: Orders}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireeOrdersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetSoireeOrdersRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeOrdersRequest 1 : ", err.Error())
		return nil, err
	}

	SoireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeOrdersRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).SoireeID = SoireeID

	return request, nil
}

func DecodeHTTPGetSoireeOrdersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetSoireeOrdersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeOrdersResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetSoireeOrdersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soirees/{SoireeID}/orders").Handler(httptransport.NewServer(
		endpoints.GetSoireeOrdersEndpoint,
		DecodeHTTPGetSoireeOrdersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireeOrders", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireeOrders(ctx context.Context, SoireeID int64) ([]svcdb.Order, error) {
	Orders, err := mw.next.GetSoireeOrders(ctx, SoireeID)

	mw.logger.Log(
		"method", "GetSoireeOrders",
		"SoireeID", SoireeID,
		"response", Orders,
		"took", time.Since(time.Now()),
	)
	return Orders, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetSoireeOrders(ctx context.Context, SoireeID int64) ([]svcdb.Order, error) {
	return mw.next.GetSoireeOrders(ctx, SoireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireeOrders(ctx context.Context, SoireeID int64) ([]svcdb.Order, error) {
	return mw.next.GetSoireeOrders(ctx, SoireeID)
}

/*************** Main ***************/
/* Main */
func BuildGetSoireeOrdersEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSoireeOrders")
		csLogger := log.With(logger, "method", "GetSoireeOrders")

		csEndpoint = GetSoireeOrdersEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSoireeOrders")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSoireeOrders(ctx context.Context, etID int64) ([]svcdb.Order, error) {
	var Orders []svcdb.Order

	request := GetSoireeOrdersRequest{SoireeID: etID}
	response, err := e.GetSoireeOrdersEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetSoireeOrders : ", err.Error())
		return Orders, err
	}
	Orders = response.(GetSoireeOrdersResponse).Orders
	return Orders, err
}

func EncodeHTTPGetSoireeOrdersRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(GetSoireeOrdersRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("SoireeID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetSoireeOrdersRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSoireeOrders(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/Soirees/{SoireeID}/Orders"),
		EncodeHTTPGetSoireeOrdersRequest,
		DecodeHTTPGetSoireeOrdersResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetSoireeOrders")(ceEndpoint)
	return ceEndpoint, nil
}
