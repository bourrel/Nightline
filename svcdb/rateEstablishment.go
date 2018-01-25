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
func (s Service) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (Establishment, error) {
	var estab Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("RateEstablishment (WaitConnection) : " + err.Error())
		return estab, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER) WHERE ID(u) = {userID}
		MATCH (e:ESTABLISHMENT) WHERE ID(e) = {estabID} AND NOT (u)-[:RATE]-(e)
		CREATE (u)-[:RATE {
			value: {rate}
		}]->(e)
		WITH u, e
		OPTIONAL MATCH (e)<-[r:RATE]-(u:USER)
		RETURN u, AVG(r.value), e
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("RateEstablishment (PrepareNeo) : " + err.Error())
		return estab, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"userID":  userID,
		"estabID": estabID,
		"rate":    rate,
	})

	if err != nil {
		fmt.Println("RateEstablishment (QueryNeo) : " + err.Error())
		return estab, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("RateEstablishment (NextNeo) : " + err.Error())
		return estab, err
	}

	(&estab).NodeToEstablishment(data[2].(graph.Node))
	return estab, err
}

/*************** Endpoint ***************/
type rateEstablishmentRequest struct {
	UserID  int64 `json:"userID"`
	EstabID int64 `json:"estabID"`
	Rate    int64 `json:"rate"`
}

type rateEstablishmentResponse struct {
	Establishment Establishment `json:"estab"`
	Err           string        `json:"err,omitempty"`
}

func RateEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(rateEstablishmentRequest)
		estab, err := svc.RateEstablishment(ctx, req.EstabID, req.UserID, req.Rate)
		if err != nil {
			fmt.Println("Error RateEstablishmentEndpoint 1 : ", err.Error())
			return rateEstablishmentResponse{Establishment: estab, Err: err.Error()}, nil
		}

		return rateEstablishmentResponse{Establishment: estab, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPRateEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request rateEstablishmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPRateEstablishmentRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPRateEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response rateEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPRateEstablishmentResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func RateEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/rate_establishment").Handler(httptransport.NewServer(
		endpoints.RateEstablishmentEndpoint,
		DecodeHTTPRateEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "RateEstablishment", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "rateEstablishment",
			"estabID", estabID,
			"userID", userID,
			"rate", rate,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.RateEstablishment(ctx, estabID, userID, rate)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (Establishment, error) {
	v, err := mw.next.RateEstablishment(ctx, estabID, userID, rate)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildRateEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "RateEstablishment")
		csLogger := log.With(logger, "method", "RateEstablishment")

		csEndpoint = RateEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "RateEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (Establishment, error) {
	request := rateEstablishmentRequest{EstabID: estabID, UserID: userID, Rate: rate}
	response, err := e.RateEstablishmentEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error RateEstablishment : ", err.Error())
		return response.(rateEstablishmentResponse).Establishment, err
	}
	return response.(rateEstablishmentResponse).Establishment, str2err(response.(rateEstablishmentResponse).Err)
}

func ClientRateEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/rate_establishment"),
		EncodeHTTPGenericRequest,
		DecodeHTTPRateEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "RateEstablishment")(ceEndpoint)
	return ceEndpoint, nil
}
