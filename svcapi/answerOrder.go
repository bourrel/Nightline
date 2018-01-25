package svcapi

import (
	"context"
	"encoding/json"
	"net/http"
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
func (s Service) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order, error) {
	order, err := s.svcpayment.AnswerOrder(ctx, orderID, userID, answer)
	return order, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type answerOrderRequest struct {
	OrderID	int64 `json:"order"`
	UserID	int64 `json:"user"`
	Answer  bool  `json:"answer"`
}

type answerOrderResponse struct {
	Order	svcdb.Order  `json:"order"`
}

func AnswerOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(answerOrderRequest)
		order, err := svc.AnswerOrder(ctx, req.OrderID, req.UserID, req.Answer)
		return answerOrderResponse{Order: order}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPAnswerOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request answerOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPAnswerOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response answerOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func AnswerOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/order/answer").Handler(httptransport.NewServer(
		endpoints.AnswerOrderEndpoint,
		DecodeHTTPAnswerOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "AnswerOrder", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "answerOrder",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.AnswerOrder(ctx, orderID, userID, answer)
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order,error) {
	return mw.next.AnswerOrder(ctx, orderID, userID, answer)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order, error) {
	return mw.next.AnswerOrder(ctx, orderID, userID, answer)
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
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
