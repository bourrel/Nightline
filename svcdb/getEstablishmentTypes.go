package svcdb

import (
	"context"
	"encoding/json"
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
func (s Service) GetEstablishmentTypes(_ context.Context) ([]string, error) {
	var establishments []string

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishmentTypes (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH (n:ESTABLISHMENT_TYPE) RETURN n")
	if err != nil {
		fmt.Println("GetEstablishmentTypes (PrepareNeo)")
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(nil)
	if err != nil {
		fmt.Println("GetEstablishmentTypes (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpEstablishment EstablishmentType

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetEstablishmentTypes (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpEstablishment).NodeToEstablishmentType(row[0].(graph.Node))

			establishments = append(establishments, tmpEstablishment.Name)
		}
		row, _, err = rows.NextNeo()
	}
	fmt.Println(establishments)
	return establishments, nil
}

/*************** Endpoint ***************/
type getEstablishmentTypesRequest struct {
}

type getEstablishmentTypesResponse struct {
	EstablishmentTypes []string `json:"types"`
	Err                string   `json:"err,omitempty"`
}

func GetEstablishmentTypesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		et, err := svc.GetEstablishmentTypes(ctx)
		if err != nil {
			return getEstablishmentTypesResponse{EstablishmentTypes: et, Err: err.Error()}, nil
		}
		return getEstablishmentTypesResponse{EstablishmentTypes: et, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentTypesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeHTTPGetEstablishmentTypesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentTypesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentTypesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishment_types").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentTypesEndpoint,
		DecodeHTTPGetEstablishmentTypesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentTypes", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentTypes",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentTypes(ctx)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	v, err := mw.next.GetEstablishmentTypes(ctx)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentTypesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentTypes")
		gefmLogger := log.With(logger, "method", "GetEstablishmentTypes")

		gefmEndpoint = GetEstablishmentTypesEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentTypes")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	var s []string

	request := getEstablishmentTypesRequest{}
	response, err := e.GetEstablishmentTypesEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getEstablishmentTypesResponse).EstablishmentTypes
	return s, str2err(response.(getEstablishmentTypesResponse).Err)
}

func ClientGetEstablishmentTypes(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "establishment_types"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetEstablishmentTypesResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentTypes")(gefmEndpoint)
	return gefmEndpoint, nil
}
