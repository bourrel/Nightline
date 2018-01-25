package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
func (s Service) GetPro(_ context.Context, u Pro) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetPro (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:PRO) WHERE u.Email = {email}
		OPTIONAL MATCH (u)-[:OWN]-(e:ESTABLISHMENT)
		RETURN u, ID(e)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetPro (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"email": u.Email,
	})
	if err != nil {
		fmt.Println("GetPro (QueryNeo) : " + err.Error())
		return pro, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetPro (NextNeo) : " + err.Error())
		return pro, err
	}
	(&pro).NodeToPro(row[0].(graph.Node))

	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetPro (---)")
			panic(err)
		} else if err != io.EOF {
			if row[1] != nil {
				pro.Establishments = append(pro.Establishments, row[1].(int64))
			}
		}
		row, _, err = rows.NextNeo()
	}

	if pro.Password != u.Password {
		err = errors.New("Invalid password")
		return pro, err
	}

	return pro, nil
}

/*************** Endpoint ***************/
type getProRequest struct {
	Pro Pro `json:"pro"`
}

type getProResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func GetProEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getProRequest)
		pro, err := svc.GetPro(ctx, req.Pro)
		if err != nil {
			fmt.Println("Error GetProEndpoint", err)
			return getProResponse{Pro: pro, Err: err.Error()}, nil
		}
		return getProResponse{Pro: pro, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getProRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPGetProRequest" + err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPGetProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getProResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetProResponse" + err.Error())
		return nil, err
	}
	return response, nil
}

func GetProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/pros/get_pro").Handler(httptransport.NewServer(
		endpoints.GetProEndpoint,
		DecodeHTTPGetProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetPro", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetPro(ctx context.Context, u Pro) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getPro",
			"pro", u,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetPro(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetPro(ctx context.Context, u Pro) (Pro, error) {
	v, err := mw.next.GetPro(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetPro")
		csLogger := log.With(logger, "method", "GetPro")

		csEndpoint = GetProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetPro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetPro(ctx context.Context, u Pro) (Pro, error) {
	request := getProRequest{Pro: u}
	response, err := e.GetProEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetPro Client" + err.Error())
		return u, err
	}
	return response.(getProResponse).Pro, str2err(response.(getProResponse).Err)
}

func ClientGetPro(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/pros/get_pro"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetProResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetPro")(ceEndpoint)
	return ceEndpoint, nil
}
