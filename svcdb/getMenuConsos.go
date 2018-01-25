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
func (s Service) GetMenuConsos(_ context.Context, menuID int64) ([]Conso, error) {
	var consos []Conso
	var tmpConso Conso

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("getMenuConsos (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (m:MENU)-[:USE]->(c:CONSO) WHERE ID(m) = {id} RETURN (c)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetMenuConsos (PrepareNeo) : " + err.Error())
		return consos, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": menuID,
	})

	if err != nil {
		fmt.Println("GetMenuConsos (QueryNeo) : " + err.Error())
		return consos, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetMenuConsos (---) : " + err.Error())
			panic(err)
		} else if err != io.EOF {
			(&tmpConso).NodeToConso(row[0].(graph.Node))
			consos = append(consos, tmpConso)
		}
		row, _, err = rows.NextNeo()
	}

	return consos, nil
}

/*************** Endpoint ***************/
type getMenuConsosRequest struct {
	MenuID int64 `json:"id"`
}

type getMenuConsosResponse struct {
	Consos []Conso `json:"consos"`
	Err    string  `json:"err,omitempty"`
}

func GetMenuConsosEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuConsosRequest)
		consos, err := svc.GetMenuConsos(ctx, req.MenuID)
		if err != nil {
			return getMenuConsosResponse{Consos: consos, Err: err.Error()}, nil
		}
		return getMenuConsosResponse{Consos: consos, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuConsosRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuConsosRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	menuID, err := strconv.ParseInt(mux.Vars(r)["MenuID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).MenuID = menuID

	return request, nil
}

func DecodeHTTPGetMenuConsosResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuConsosResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetMenuConsosHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/menus/{MenuID}/consos").Handler(httptransport.NewServer(
		endpoints.GetMenuConsosEndpoint,
		DecodeHTTPGetMenuConsosRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenuConsos", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenuConsos(ctx context.Context, menuID int64) ([]Conso, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getMenuConsos",
			"menuID", menuID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetMenuConsos(ctx, menuID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenuConsos(ctx context.Context, menuID int64) ([]Conso, error) {
	v, err := mw.next.GetMenuConsos(ctx, menuID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetMenuConsosEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetMenuConsos")
		gefmLogger := log.With(logger, "method", "GetMenuConsos")

		gefmEndpoint = GetMenuConsosEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetMenuConsos")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetMenuConsos(ctx context.Context, menuID int64) ([]Conso, error) {
	var s []Conso

	request := getMenuConsosRequest{MenuID: menuID}
	response, err := e.GetMenuConsosEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getMenuConsosResponse).Consos
	return s, str2err(response.(getMenuConsosResponse).Err)
}

func EncodeHTTPGetMenuConsosRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getMenuConsosRequest).MenuID)
	encodedUrl, err := route.Path(r.URL.Path).URL("MenuID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetMenuConsos(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/menus/{MenuID}/consos"),
		EncodeHTTPGetMenuConsosRequest,
		DecodeHTTPGetMenuConsosResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetMenuConsos")(gefmEndpoint)
	return gefmEndpoint, nil
}
