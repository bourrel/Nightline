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
func (s Service) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	order, err := s.svcpayment.PutOrder(ctx, orderID, step, flag)
	return order, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type putOrderRequest struct {
	OrderID int64	`json:"id"`
	Step	string	`json:"step"`
	Flag	bool	`json:"flag"`
}

type putOrderResponse struct {
	Order svcdb.Order  `json:"order"`
	Err   error `json:"err,omitempty"`
}

func PutOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(putOrderRequest)
		order, err := svc.PutOrder(ctx, req.OrderID, req.Step, req.Flag)
		return putOrderResponse{Order: order, Err: err}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPPutOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request putOrderRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).OrderID = orderID

	step := mux.Vars(r)["Step"]
	(&request).Step = step

	flag, err := strconv.ParseBool(mux.Vars(r)["Flag"])
	if err != nil {
		return nil, err
	}
	(&request).Flag = flag

	return request, nil
}

func DecodeHTTPPutOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response putOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func PutOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path(`/orders/{OrderID:[0-9]+}/{Step:(?:Ready|Deliverpaid)}/{Flag:(?:true|false)}`).Handler(httptransport.NewServer(
		endpoints.PutOrderEndpoint,
		DecodeHTTPPutOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "PutOrder", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "putOrder",
			"orderID", orderID,
			"step", step,
			"flag", flag,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.PutOrder(ctx, orderID, step, flag)
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	return mw.next.PutOrder(ctx, orderID, step, flag)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	return mw.next.PutOrder(ctx, orderID, step, flag)
}

/*************** Main ***************/
/* Main */
func BuildPutOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "PutOrder")
		csLogger := log.With(logger, "method", "PutOrder")

		csEndpoint = PutOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "PutOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	var order svcdb.Order

	request := putOrderRequest{OrderID: orderID, Step: step, Flag: flag}
	response, err := e.PutOrderEndpoint(ctx, request)
	if err != nil {
		return order, err
	}
	order = response.(putOrderResponse).Order
	return order, response.(putOrderResponse).Err
}

func EncodeHTTPPutOrderRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(putOrderRequest).OrderID)
	step := fmt.Sprintf("%v", request.(putOrderRequest).Step)
	flag := fmt.Sprintf("%v", request.(putOrderRequest).Flag)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", id, "Step", step, "Flag", flag)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientPutOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, `/orders/{OrderID:[0-9]+}/{Step:(?:Ready|Deliverpaid)}/{Flag:(?:true|false)}`),
		EncodeHTTPPutOrderRequest,
		DecodeHTTPPutOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "PutOrder")(ceEndpoint)
	return ceEndpoint, nil
}
