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
func (s Service) SearchEstablishments(_ context.Context, query string) ([]SearchResponse, error) {
	var tmpEstablishment Establishment
	var tmpResponse SearchResponse
	var response []SearchResponse

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("SearchEstablishments (WaitConnection) : " + err.Error())
		return response, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH(e:ESTABLISHMENT) WHERE LOWER(e.Name) CONTAINS LOWER({query}) RETURN e")
	defer stmt.Close()
	if err != nil {
		fmt.Println("SearchAllEstablishments (PrepareNeo)")
		panic(err)
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"query": query,
	})

	if err != nil {
		fmt.Println("SearchAllEstablishments (QueryNeo)")
		panic(err)
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("SearchAllEstablishments (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpEstablishment).NodeToEstablishment(row[0].(graph.Node))

			tmpResponse.FromEstablishment(tmpEstablishment)

			response = append(response, tmpResponse)
		}
		row, _, err = rows.NextNeo()
	}
	return response, nil
}

/*************** Endpoint ***************/
type searchEstablishmentsRequest struct {
	query string `json:"query"`
}

type searchEstablishmentsResponse struct {
	Establishments []SearchResponse `json:"establishments"`
	Err            string           `json:"err,omitempty"`
}

func SearchEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchEstablishmentsRequest)
		establishments, err := svc.SearchEstablishments(ctx, req.query)
		if err != nil {
			fmt.Println("Error SearchEstablishmentsEndpoint : ", err.Error())
			return searchEstablishmentsResponse{establishments, err.Error()}, nil
		}
		return searchEstablishmentsResponse{establishments, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req searchEstablishmentsRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	(&req).query = mux.Vars(r)["query"]

	if (&req).query == "" {
		err := errors.New("Invalid query")
		fmt.Println("Error DecodeHTTPSearchEstablishmentsRequest : ", err.Error())
		return nil, err
	}

	return req, nil
}

func DecodeHTTPSearchEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchEstablishmentsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPSearchEstablishmentsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func SearchEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/establishments?q={query}").Handler(httptransport.NewServer(
		endpoints.SearchEstablishmentsEndpoint,
		DecodeHTTPSearchEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchEstablishments", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchEstablishments(ctx context.Context, query string) ([]SearchResponse, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchEstablishments",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchEstablishments(ctx, query)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchEstablishments(ctx context.Context, query string) ([]SearchResponse, error) {
	v, err := mw.next.SearchEstablishments(ctx, query)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildSearchEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchEstablishments")
		csLogger := log.With(logger, "method", "SearchEstablishments")

		csEndpoint = SearchEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) SearchEstablishments(ctx context.Context, query string) ([]SearchResponse, error) {
	var et []SearchResponse

	request := searchEstablishmentsRequest{query: query}
	response, err := e.SearchEstablishmentsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error Client SearchEstablishments : ", err.Error())
		return et, err
	}
	estabs := response.(searchEstablishmentsResponse).Establishments
	return estabs, str2err(response.(searchEstablishmentsResponse).Err)
}

func EncodeHTTPSearchEstablishmentsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()

	query := fmt.Sprintf("%v", request.(searchEstablishmentsRequest).query)
	encodedURL, err := route.Path(r.URL.Path).Queries("query", query).URL("query", query)
	if err != nil {
		fmt.Println("Error EncodeHTTPSearchEstablishmentsRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedURL.Path
	return nil
}

func ClientSearchEstablishments(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/search/establishments?q={query}"),
		EncodeHTTPSearchEstablishmentsRequest,
		DecodeHTTPSearchEstablishmentsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "SearchEstablishments")(ceEndpoint)
	return ceEndpoint, nil
}
