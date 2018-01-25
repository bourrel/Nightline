package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"strconv"
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
func (s Service) GetProByID(_ context.Context, proID int64) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetProByID (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:PRO) WHERE ID(u) = {id}
		OPTIONAL MATCH (u)-[:OWN]-(e:ESTABLISHMENT)	
		RETURN u, ID(e)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetProByID (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": proID,
	})
	if err != nil {
		fmt.Println("GetProByID (QueryNeo) : " + err.Error())
		return pro, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetProByID (NextNeo) : " + err.Error())
		return pro, err
	}

	(&pro).NodeToPro(row[0].(graph.Node))
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetProByID (---)")
			panic(err)
		} else if err != io.EOF {
			if row[1] != nil {
				pro.Establishments = append(pro.Establishments, row[1].(int64))
			}
		}
		row, _, err = rows.NextNeo()
	}

	return pro, nil
}

/*************** Endpoint ***************/
type getProByIDRequest struct {
	ID int64 `json:"id"`
}

type getProByIDResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func GetProByIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getProByIDRequest)
		pro, err := svc.GetProByID(ctx, req.ID)
		if err != nil {
			return getProByIDResponse{pro, err.Error()}, nil
		}
		return getProByIDResponse{pro, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProByIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getProByIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	proiD, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = proiD

	return request, nil
}

func DecodeHTTPGetProByIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getProByIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetProByIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/pros/get_pro/{id:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetProByIDEndpoint,
		DecodeHTTPGetProByIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetProByID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetProByID(ctx context.Context, proID int64) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getProByID",
			"proID", proID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetProByID(ctx, proID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetProByID(ctx context.Context, proID int64) (Pro, error) {
	v, err := mw.next.GetProByID(ctx, proID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetProByIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetProByID")
		csLogger := log.With(logger, "method", "GetProByID")

		csEndpoint = GetProByIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetProByID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetProByID(ctx context.Context, ID int64) (Pro, error) {
	var pro Pro

	request := getProByIDRequest{ID: ID}
	response, err := e.GetProByIDEndpoint(ctx, request)
	if err != nil {
		return pro, err
	}
	pro = response.(getProByIDResponse).Pro
	return pro, str2err(response.(getProByIDResponse).Err)
}

func ClientGetProByID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/pros/get_pro/{ID:[0-9]+}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetProByIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetProByID")(gefmEndpoint)
	return gefmEndpoint, nil
}
