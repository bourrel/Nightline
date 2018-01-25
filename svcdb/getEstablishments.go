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
func (s Service) GetEstablishments(_ context.Context) ([]Establishment, error) {
	var establishments []Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishments (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (e:ESTABLISHMENT)
		OPTIONAL MATCH (e)<-[:OWN]-(p:PRO)
		OPTIONAL MATCH (e)<-[r:RATE]-(u:USER)
		RETURN e, ID(p) as owner, AVG(r.value)
	`)
	if err != nil {
		fmt.Println("GetAllEstablishments (PrepareNeo)")
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(nil)
	if err != nil {
		fmt.Println("GetAllEstablishments (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpEstablishment Establishment

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetAllEstablishments (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpEstablishment).NodeToEstablishment(row[0].(graph.Node))

			if row[1] != nil {
				(&tmpEstablishment).Owner = row[1].(int64)
			}
			if rate, ok := row[2].(float64); ok {
				(&tmpEstablishment).Rate = rate
			}
			establishments = append(establishments, tmpEstablishment)
			tmpEstablishment = Establishment{}
		}
		row, _, err = rows.NextNeo()
	}
	fmt.Println(establishments)
	return establishments, nil
}

/*************** Endpoint ***************/
type getEstablishmentsRequest struct {
}

type getEstablishmentsResponse struct {
	Establishments []Establishment `json:"establishments"`
	Err            string          `json:"err,omitempty"`
}

func GetEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		establishments, err := svc.GetEstablishments(ctx)
		if err != nil {
			fmt.Println("Error GetEstablishmentsEndpoint 1 : ", err.Error())
			return getEstablishmentsResponse{establishments, err.Error()}, nil
		}

		for i := 0; i < len(establishments); i++ {
			establishments[i].Type, err = svc.GetEstablishmentType(ctx, establishments[i].ID)
			if err != nil {
				fmt.Println("Error GetEstablishmentsEndpoint 2 : ", err.Error())
				return getEstablishmentsResponse{establishments, err.Error()}, nil
			}
		}

		return getEstablishmentsResponse{establishments, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeHTTPGetEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentsEndpoint,
		DecodeHTTPGetEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishments", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishments(ctx context.Context) ([]Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishments",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishments(ctx)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishments(ctx context.Context) ([]Establishment, error) {
	v, err := mw.next.GetEstablishments(ctx)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetEstablishments")
		csLogger := log.With(logger, "method", "GetEstablishments")

		csEndpoint = GetEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishments(ctx context.Context) ([]Establishment, error) {
	var et []Establishment

	response, err := e.GetEstablishmentsEndpoint(ctx, nil)
	if err != nil {
		return et, err
	}
	et = response.(getEstablishmentsResponse).Establishments
	return et, str2err(response.(getEstablishmentsResponse).Err)
}

func ClientGetEstablishments(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetEstablishmentsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetEstablishments")(ceEndpoint)
	return ceEndpoint, nil
}
