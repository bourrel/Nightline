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
func (s Service) GetEstablishmentConsos(_ context.Context, estabID int64) ([]Conso, error) {
	var consos []Conso
	var tmpConso Conso

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("getEstablishmentConsos (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:GOT]->(c:CONSO) WHERE ID(e) = {id} RETURN (c)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetEstablishmentConsos (PrepareNeo) : " + err.Error())
		return consos, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("GetEstablishmentConsos (QueryNeo) : " + err.Error())
		return consos, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetEstablishmentConsos (---) : " + err.Error())
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
type getEstablishmentConsosRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmentConsosResponse struct {
	Consos []Conso `json:"consos"`
	Err    string  `json:"err,omitempty"`
}

func GetEstablishmentConsosEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentConsosRequest)
		consos, err := svc.GetEstablishmentConsos(ctx, req.EstabID)
		if err != nil {
			return getEstablishmentConsosResponse{Consos: consos, Err: err.Error()}, nil
		}
		return getEstablishmentConsosResponse{Consos: consos, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentConsosRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentConsosRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetEstablishmentConsosResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentConsosResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentConsosHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabID}/consos").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentConsosEndpoint,
		DecodeHTTPGetEstablishmentConsosRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentConsos", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentConsos(ctx context.Context, estabID int64) ([]Conso, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentConsos",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentConsos(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentConsos(ctx context.Context, estabID int64) ([]Conso, error) {
	v, err := mw.next.GetEstablishmentConsos(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentConsosEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentConsos")
		gefmLogger := log.With(logger, "method", "GetEstablishmentConsos")

		gefmEndpoint = GetEstablishmentConsosEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentConsos")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentConsos(ctx context.Context, estabID int64) ([]Conso, error) {
	var s []Conso

	request := getEstablishmentConsosRequest{EstabID: estabID}
	response, err := e.GetEstablishmentConsosEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getEstablishmentConsosResponse).Consos
	return s, str2err(response.(getEstablishmentConsosResponse).Err)
}

func EncodeHTTPGetEstablishmentConsosRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentConsosRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetEstablishmentConsos(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{EstabID}/consos"),
		EncodeHTTPGetEstablishmentConsosRequest,
		DecodeHTTPGetEstablishmentConsosResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentConsos")(gefmEndpoint)
	return gefmEndpoint, nil
}
