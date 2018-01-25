package svcevent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Shopify/sarama"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) Push(ctx context.Context, name string, body interface{}, userID int64) (int32, int64, error) {
	var topic string
	var partition int32

	topic = "user-topic"
	partition = -1
	msg := formatMessage(name, body, userID)
	message := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: partition,
		Value:     sarama.StringEncoder(msg),
	}

	return producer.SendMessage(message)
}

func formatMessage(name string, body interface{}, userID int64) string {
	pr := jsonMessage{
		Name: name,
		User: userID,
		Body: body,
	}
	out, err := json.Marshal(&pr)
	if err != nil {
		fmt.Println("formatMessage : " + err.Error())
		panic(err)
	}

	return string(out)
}

// /*************** Endpoint ***************/
type PushRequest struct {
	Name   string      `json:"name"`
	Body   interface{} `json:"body"`
	UserID int64       `json:"user"`
}

type PushResponse struct {
	Partition int32  `json:"partiion"`
	Offset    int64  `json:"offset"`
	Err       string `json:"err,omitempty"`
}

func PushEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PushRequest)

		p, o, err := svc.Push(ctx, req.Name, req.Body, req.UserID)
		if err != nil {
			return PushResponse{p, o, err.Error()}, err
		}
		return PushResponse{Partition: p, Offset: o}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPPushRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request PushRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPPushRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPPushResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response PushResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPPushResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func PushHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/push").Handler(httptransport.NewServer(
		endpoints.PushEndpoint,
		DecodeHTTPPushRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "Push", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) Push(ctx context.Context, name string, body interface{}, userID int64) (int32, int64, error) {
	p, o, err := mw.next.Push(ctx, name, body, userID)

	mw.logger.Log(
		"method", "Push",
		"name", name,
		"partition", p,
		"offset", o,
		"took", time.Since(time.Now()),
	)

	return p, o, err
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) Push(ctx context.Context, name string, body interface{}, userID int64) (int32, int64, error) {
	return mw.next.Push(ctx, name, body, userID)
}

/*************** Main ***************/
/* Main */
func BuildPushEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "Push")
		csLogger := log.With(logger, "method", "Push")

		csEndpoint = PushEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "Push")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) Push(ctx context.Context, name string, body interface{}, userID int64) (int32, int64, error) {
	request := PushRequest{Name: name, Body: body, UserID: userID}
	res, err := e.PushEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Push response error : " + err.Error())
		return res.(PushResponse).Partition, res.(PushResponse).Offset, err
	}

	return res.(PushResponse).Partition, res.(PushResponse).Offset, str2err(res.(PushResponse).Err)
}

func ClientPush(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/push"),
		EncodeHTTPGenericRequest,
		DecodeHTTPPushResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "Push")(ceEndpoint)
	return ceEndpoint, nil
}
