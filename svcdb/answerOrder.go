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
func (s Service) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (Order, error) {
    var order Order

    conn, err := WaitConnection(5)
    if err != nil {
		fmt.Println("AnswerOrder (WaitConnection) : " + err.Error())
		return order, err
	}
    defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
    MATCH (o:ORDER)-[r:TO]->(u:USER)
    WHERE ID(o) = {orderID} AND ID(u) = {userID}
    SET r.Approved = {answer}
    RETURN ID(o)`)
	defer stmt.Close()

    if err != nil {
		fmt.Println("AnswerOrder (PrepareNeo) : " + err.Error())
		return order, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"orderID":     orderID,
		"userID":      userID,
		"answer":      strconv.FormatBool(answer),
	})

	if err != nil {
		fmt.Println("AnswerOrder (QueryNeo) : " + err.Error())
		return order, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("AnswerOrder (NextNeo) : " + err.Error())
		return order, err
	}

	orderID = data[0].(int64)

	order, err = s.GetOrder(ctx, orderID)
	if err != nil {
		fmt.Println("AnswerOrder (GetOrder) : " + err.Error())
		return order, err
	}
	return order, err
}

/*************** Endpoint ***************/
type answerOrderRequest struct {
	OrderID	int64 `json:"order"`
	UserID  int64 `json:"user"`
	Answer  bool  `json:"answer"`
}

type answerOrderResponse struct {
	Order	Order  `json:"order"`
	Err		string `json:"err,omitempty"`
}

func AnswerOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(answerOrderRequest)
		order, err := svc.AnswerOrder(ctx, req.OrderID, req.UserID, req.Answer)

		// Create node
		if err != nil {
			fmt.Println("Error AnswerOrderEndpoint 1 : ", err.Error())
			return answerOrderResponse{Order: order, Err: err.Error()}, nil
		}

		return answerOrderResponse{Order: order, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPAnswerOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request answerOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPAnswerOrderRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPAnswerOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response answerOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPAnswerOrderResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func AnswerOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/order/answer").Handler(httptransport.NewServer(
		endpoints.AnswerOrderEndpoint,
		DecodeHTTPAnswerOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "AnswerOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "answerOrder",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.AnswerOrder(ctx, orderID, userID, answer)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (Order, error) {
	v, err := mw.next.AnswerOrder(ctx, orderID, userID, answer)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildAnswerOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "AnswerOrder")
		csLogger := log.With(logger, "method", "AnswerOrder")

		csEndpoint = AnswerOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "AnswerOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (Order, error) {
	request := answerOrderRequest{OrderID: orderID, UserID: userID, Answer: answer}
	response, err := e.AnswerOrderEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error AnswerOrder : ", err.Error())
		return response.(answerOrderResponse).Order, err
	}
	return response.(answerOrderResponse).Order, str2err(response.(answerOrderResponse).Err)
}

func ClientAnswerOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/order/answer"),
		EncodeHTTPGenericRequest,
		DecodeHTTPAnswerOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "AnswerOrder")(ceEndpoint)
	return ceEndpoint, nil
}
