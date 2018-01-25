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
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

/*************** Service ***************/
func (s Service) GetEstablishment(_ context.Context, estabID int64) (Establishment, error) {
	var establishment Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishment (WaitConnection) : " + err.Error())
		return establishment, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (e:ESTABLISHMENT) WHERE ID(e) = {id}
		OPTIONAL MATCH (e)<-[:OWN]-(p:PRO)
		OPTIONAL MATCH (e)<-[r:RATE]-(u:USER)
		RETURN e, ID(p) as owner, AVG(r.value)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetEstablishment (PrepareNeo) : " + err.Error())
		return establishment, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("GetEstablishment (QueryNeo) : " + err.Error())
		return establishment, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetEstablishment (NextNeo) : " + err.Error())
		return establishment, err
	}

	(&establishment).NodeToEstablishment(row[0].(graph.Node))
	if row[1] != nil {
		establishment.Owner = row[1].(int64)
	}

	if rate, ok := row[2].(float64); ok {
		(&establishment).Rate = rate
	}
	return establishment, nil
}

/*************** Endpoint ***************/
type getEstablishmentRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmentResponse struct {
	Establishment Establishment `json:"establishment"`
	Err           string        `json:"err,omitempty"`
}

func GetEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentRequest)
		establishment, err := svc.GetEstablishment(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetEstablishmentEndpoint 1 : ", err.Error())
			return getEstablishmentResponse{establishment, err.Error()}, nil
		}

		establishment.Type, err = svc.GetEstablishmentType(ctx, establishment.ID)
		if err != nil {
			fmt.Println("Error GetEstablishmentEndpoint 2 : ", err.Error())
			return getEstablishmentResponse{establishment, err.Error()}, nil
		}
		return getEstablishmentResponse{establishment, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPGetEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{id}").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentEndpoint,
		DecodeHTTPGetEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishment", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishment(ctx context.Context, estabID int64) (Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishment",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishment(ctx context.Context, estabID int64) (Establishment, error) {
	v, err := mw.next.GetEstablishment(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetEstablishment")
		csLogger := log.With(logger, "method", "GetEstablishment")

		csEndpoint = GetEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishment(ctx context.Context, etID int64) (Establishment, error) {
	var et Establishment

	request := getEstablishmentRequest{EstabID: etID}
	response, err := e.GetEstablishmentEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getEstablishmentResponse).Establishment
	return et, str2err(response.(getEstablishmentResponse).Err)
}

func ClientGetEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetEstablishment")(ceEndpoint)
	return ceEndpoint, nil
}
