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
)

/*************** Service ***************/
/* Service - Business logic */
func (s Service) DeliverOrder(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	// orderID, err := s.svcsoiree.DeliverOrder(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
	// return orderID, dbToHTTPErr(err)

	return 0, dbToHTTPErr(nil)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type DeliverOrderRequest struct {
	EstablishmentID int64
	MenuID          int64
	OrderID         int64
	SoireeBegin     time.Time
	SoireeEnd       time.Time
}
type DeliverOrderResponse struct {
	SoireeID int64
}

/* Endpoint - Create endpoint */
func DeliverOrderEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(DeliverOrderRequest)
		soireeID, err := s.DeliverOrder(ctx, csReq.EstablishmentID, csReq.MenuID, csReq.SoireeBegin, csReq.SoireeEnd)
		return DeliverOrderResponse{
			SoireeID: soireeID,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPDeliverOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req DeliverOrderRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPDeliverOrderRequest 1 : ", err.Error())
		return nil, RequestError
	}
	(&req).OrderID = orderID

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPDeliverOrderRequest 1 : ", err.Error())
		return nil, RequestError
	}
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPDeliverOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp DeliverOrderResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func DeliverOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/orders/{OrderID}/deliver").Handler(httptransport.NewServer(
		endpoints.DeliverOrderEndpoint,
		DecodeHTTPDeliverOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeliverOrder", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeliverOrder(ctx context.Context, estabID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	soireeID, err := mw.next.DeliverOrder(ctx, estabID, menuID, soireeBegin, soireeEnd)

	mw.logger.Log(
		"method", "DeliverOrder",
		"request", DeliverOrderRequest{EstablishmentID: estabID, MenuID: menuID, SoireeBegin: soireeBegin, SoireeEnd: soireeEnd},
		"soireeID", soireeID,
		"error", err,
		"took", time.Since(time.Now()),
	)
	return soireeID, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) DeliverOrder(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (soireeID int64, err error) {
	return mw.next.DeliverOrder(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeliverOrder(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	return mw.next.DeliverOrder(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
}

/*************** Main ***************/
/* Main */
func BuildDeliverOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeliverOrder")
		csLogger := log.With(logger, "method", "DeliverOrder")

		csEndpoint = DeliverOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeliverOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) DeliverOrder(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	var soireeID int64

	request := DeliverOrderRequest{
		EstablishmentID: establishmentID,
		MenuID:          menuID,
		SoireeBegin:     soireeBegin,
		SoireeEnd:       soireeEnd,
	}
	response, err := e.DeliverOrderEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	soireeID = response.(DeliverOrderResponse).SoireeID
	return soireeID, err
}

func ClientDeliverOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/orders/{OrderID}/deliver"),
		EncodeHTTPGenericRequest,
		DecodeHTTPDeliverOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "DeliverOrder")(gefmEndpoint)
	return gefmEndpoint, nil
}
