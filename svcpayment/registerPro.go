package svcpayment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/account"
	
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)
 
/*************** Service ***************/
func (s Service) RegisterPro(_ context.Context, p svcdb.Pro) (svcdb.Pro, error) {
	stripe.Key = STRIPESKEY
	acct, err := account.New(&stripe.AccountParams{
		Type: "standard",
		Country: "FR",
		Email: p.Email,
	})

	if err != nil {
		fmt.Println("RegisterPro (StripeAccountNew) : " + err.Error())
		return p, err
	}

	p.StripeID = (*acct).ID
	p.StripeSKey = (*acct).Keys.Secret
	p.StripePKey = (*acct).Keys.Publish
	return p, nil
}

/*************** Endpoint ***************/
type registerProRequest struct {
	Pro	    svcdb.Pro  `json:"pro"`
}

type registerProResponse struct {
	Pro	    svcdb.Pro  `json:"pro"`
	Err		string     `json:"err,omitempty"`
}

func RegisterProEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(registerProRequest)
		pro, err := svc.RegisterPro(ctx, req.Pro)

		// Create node
		if err != nil {
			fmt.Println("Error RegisterProEndpoint 1 : ", err.Error())
			return registerProResponse{Pro: pro, Err: err.Error()}, nil
		}

		return registerProResponse{Pro: pro, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPRegisterProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request registerProRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPRegisterProRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPRegisterProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response registerProResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPRegisterProResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func RegisterProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/pros/stripe").Handler(httptransport.NewServer(
		endpoints.RegisterProEndpoint,
		DecodeHTTPRegisterProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "RegisterPro", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) RegisterPro(ctx context.Context, o svcdb.Pro) (svcdb.Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "registerPro",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.RegisterPro(ctx, o)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) RegisterPro(ctx context.Context, o svcdb.Pro) (svcdb.Pro, error) {
	v, err := mw.next.RegisterPro(ctx, o)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildRegisterProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "RegisterPro")
		csLogger := log.With(logger, "method", "RegisterPro")

		csEndpoint = RegisterProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "RegisterPro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) RegisterPro(ctx context.Context, o svcdb.Pro) (svcdb.Pro, error) {
	request := registerProRequest{Pro: o}
	response, err := e.RegisterProEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error RegisterPro : ", err.Error())
		return o, err
	}
	return response.(registerProResponse).Pro, str2err(response.(registerProResponse).Err)
}

func ClientRegisterPro(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/pros/stripe"),
		EncodeHTTPGenericRequest,
		DecodeHTTPRegisterProResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "RegisterPro")(ceEndpoint)
	return ceEndpoint, nil
}
