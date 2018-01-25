package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) UpdateOrderReference(c context.Context, orderID int64, userID int64, reference string) (error) {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdateOrderReference (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
        MATCH (o:ORDER)-[r:TO]->(u:USER)
        WHERE ID(o) = {OrderID} AND ID(u) = {UserID}
        SET r.Reference = {Reference}
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdateOrderReference (PrepareNeo) : " + err.Error())
		return err
	}

	_, err = stmt.QueryNeo(map[string]interface{}{
		"OrderID":       orderID,
		"UserID":        userID,
		"Reference":     reference,
	})
	if err != nil {
		fmt.Println("UpdateOrderReference (QueryNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type updateOrderReferenceRequest struct {
	OrderID	int64	 `json:"orderID"`
	UserID	int64	 `json:"userID"`
	Reference string `json:"reference"`
}

type updateOrderReferenceResponse struct {
	Err  string `json:"err,omitempty"`
}

func UpdateOrderReferenceEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateOrderReferenceRequest)
		err := svc.UpdateOrderReference(ctx, req.OrderID, req.UserID, req.Reference)
		if err != nil {
			fmt.Println("Error UpdateOrderReferenceEndpoint : ", err.Error())
			return updateOrderReferenceResponse{err.Error()}, nil
		}
		return updateOrderReferenceResponse{""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateOrderReferenceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updateOrderReferenceRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).OrderID = orderID

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).UserID = userID

	reference := mux.Vars(r)["Reference"]
	(&request).Reference = reference

	return request, nil
}

func DecodeHTTPUpdateOrderReferenceResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateOrderReferenceResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdateOrderReferenceResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdateOrderReferenceHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/order/{OrderID:[0-9]+}/reference/{UserID:[0-9]+}/{Reference}").Handler(httptransport.NewServer(
		endpoints.UpdateOrderReferenceEndpoint,
		DecodeHTTPUpdateOrderReferenceRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateOrderReference", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateOrderReference(ctx context.Context, orderID int64, userID int64, reference string) (error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updateOrderReference",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdateOrderReference(ctx, orderID, userID, reference)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateOrderReference(ctx context.Context, orderID int64, userID int64, reference string) (error) {
	err := mw.next.UpdateOrderReference(ctx, orderID, userID, reference)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildUpdateOrderReferenceEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateOrderReference")
		csLogger := log.With(logger, "method", "UpdateOrderReference")

		csEndpoint = UpdateOrderReferenceEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateOrderReference")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdateOrderReference(ctx context.Context, orderID int64, userID int64, reference string) (error) {
	request := updateOrderReferenceRequest{OrderID: orderID, UserID: userID, Reference: reference}
	response, err := e.UpdateOrderReferenceEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UpdateOrderReference : ", err.Error())
		return err
	}
	return str2err(response.(updateOrderReferenceResponse).Err)
}

func EncodeHTTPUpdateOrderReferenceRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	orderID := fmt.Sprintf("%v", request.(updateOrderReferenceRequest).OrderID)
	userID := fmt.Sprintf("%v", request.(updateOrderReferenceRequest).UserID)
	reference := fmt.Sprintf("%v", request.(updateOrderReferenceRequest).Reference)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", orderID, "UserID", userID, "Reference", reference)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientUpdateOrderReference(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/order/{OrderID:[0-9]+}/reference/{UserID:[0-9]+}/{Reference}"),
		EncodeHTTPUpdateOrderReferenceRequest,
		DecodeHTTPUpdateOrderReferenceResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdateOrderReference")(ceEndpoint)
	return ceEndpoint, nil
}
