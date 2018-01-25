package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetSuccessByValue(_ context.Context, value string) (Success, error) {
	var success Success

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetSuccessByValue (WaitConnection) : " + err.Error())
		return success, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (s)
		WHERE s.Value = {value}
		RETURN s
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetSuccessByValue (PrepareNeo) : " + err.Error())
		return success, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"value": value,
	})
	if err != nil {
		fmt.Println("GetSuccessByValue (QueryNeo) : " + err.Error())
		return success, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetSuccessByValue (NextNeo) : " + err.Error())
		return success, err
	}

	(&success).NodeToSuccess(row[0].(graph.Node))
	return success, nil
}

/*************** Endpoint ***************/
type getSuccessByValueRequest struct {
	Value string `json:"value"`
}

type getSuccessByValueResponse struct {
	Success Success `json:"success"`
	Err     string  `json:"err,omitempty"`
}

func GetSuccessByValueEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSuccessByValueRequest)
		success, err := svc.GetSuccessByValue(ctx, req.Value)
		if err != nil {
			fmt.Println("GetSuccessByValueEndpoint")
			return getSuccessByValueResponse{success, err.Error()}, nil
		}
		return getSuccessByValueResponse{success, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSuccessByValueRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSuccessByValueRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("DecodeHTTPGetSuccessByValueRequest")
		return nil, err
	}
	(&request).Value = mux.Vars(r)["value"]

	return request, nil
}

func DecodeHTTPGetSuccessByValueResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSuccessByValueResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("DecodeHTTPGetSuccessByValueResponse")
		return nil, err
	}
	return response, nil
}

func GetSuccessByValueHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/success/{value}").Handler(httptransport.NewServer(
		endpoints.GetSuccessByValueEndpoint,
		DecodeHTTPGetSuccessByValueRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSuccessByValue", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSuccessByValue(ctx context.Context, value string) (Success, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getSuccessByValue",
			"value", value,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetSuccessByValue(ctx, value)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSuccessByValue(ctx context.Context, value string) (Success, error) {
	v, err := mw.next.GetSuccessByValue(ctx, value)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetSuccessByValueEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSuccessByValue")
		csLogger := log.With(logger, "method", "GetSuccessByValue")

		csEndpoint = GetSuccessByValueEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSuccessByValue")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSuccessByValue(ctx context.Context, value string) (Success, error) {
	var success Success

	request := getSuccessByValueRequest{Value: value}
	response, err := e.GetSuccessByValueEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Client GetSuccessByValue")
		return success, err
	}
	success = response.(getSuccessByValueResponse).Success
	return success, str2err(response.(getSuccessByValueResponse).Err)
}

func EncodeHTTPGetSuccessByValueRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()

	id := fmt.Sprintf("%v", request.(getSuccessByValueRequest).Value)
	encodedUrl, err := route.Path(r.URL.Path).URL("value", id)
	if err != nil {
		fmt.Println("Client EncodeHTTPGetSuccessByValueRequest")
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSuccessByValue(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/success/{value}"),
		EncodeHTTPGetSuccessByValueRequest,
		DecodeHTTPGetSuccessByValueResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetSuccessByValue")(gefmEndpoint)
	return gefmEndpoint, nil
}
