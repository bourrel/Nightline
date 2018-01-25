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
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetConsoByID(_ context.Context, consoID int64) (Conso, error) {
	var conso Conso

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetConsoByID (WaitConnection) : " + err.Error())
		return conso, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (c:CONSO)
	WHERE ID(c) = {id}
	RETURN c`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetConsoByID (PrepareNeo) : " + err.Error())
		return conso, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": consoID,
	})
	if err != nil {
		fmt.Println("GetConsoByID (QueryNeo) : " + err.Error())
		return conso, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetConsoByID (NextNeo) : " + err.Error())
		return conso, err
	}

	(&conso).NodeToConso(row[0].(graph.Node))
	return conso, nil
}

/*************** Endpoint ***************/
type getConsoByIDRequest struct {
	ID int64 `json:"id"`
}

type getConsoByIDResponse struct {
	Conso Conso  `json:"conso"`
	Err   string `json:"err,omitempty"`
}

func GetConsoByIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConsoByIDRequest)
		conso, err := svc.GetConsoByID(ctx, req.ID)
		if err != nil {
			return getConsoByIDResponse{conso, err.Error()}, nil
		}
		return getConsoByIDResponse{conso, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConsoByIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getConsoByIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	consoiD, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = consoiD

	return request, nil
}

func DecodeHTTPGetConsoByIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConsoByIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetConsoByIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/consos/get_conso/{id}").Handler(httptransport.NewServer(
		endpoints.GetConsoByIDEndpoint,
		DecodeHTTPGetConsoByIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConsoByID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConsoByID(ctx context.Context, consoID int64) (Conso, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getConsoByID",
			"consoID", consoID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetConsoByID(ctx, consoID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConsoByID(ctx context.Context, consoID int64) (Conso, error) {
	v, err := mw.next.GetConsoByID(ctx, consoID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetConsoByIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConsoByID")
		csLogger := log.With(logger, "method", "GetConsoByID")

		csEndpoint = GetConsoByIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConsoByID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConsoByID(ctx context.Context, ID int64) (Conso, error) {
	var conso Conso

	request := getConsoByIDRequest{ID: ID}
	response, err := e.GetConsoByIDEndpoint(ctx, request)
	if err != nil {
		return conso, err
	}
	conso = response.(getConsoByIDResponse).Conso
	return conso, str2err(response.(getConsoByIDResponse).Err)
}

func EncodeHTTPGetConsoByIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getConsoByIDRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("ID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConsoByID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/consos/get_conso/{ID}"),
		EncodeHTTPGetConsoByIDRequest,
		DecodeHTTPGetConsoByIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetConsoByID")(gefmEndpoint)
	return gefmEndpoint, nil
}
