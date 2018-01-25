package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"io"

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
func (s Service) GetRecommendation(_ context.Context, userID int64) ([]Establishment, error) {
	var establishments []Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetRecommendation (WaitConnection) : " + err.Error())
		return establishments, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (e:ESTABLISHMENT)<-[r:RATE]-(USER) WITH e, AVG(r.value) AS rav
		ORDER BY rav DESC LIMIT 3 RETURN e
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetRecommendation (PrepareNeo) : " + err.Error())
		return establishments, err
	}
	
	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})

	if err != nil {
		fmt.Println("GetRecommendation (QueryNeo) : " + err.Error())
		return establishments, err
	}

	var tmpEstab Establishment

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetRecommendation (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpEstab).NodeToEstablishment(row[0].(graph.Node))

			establishments = append(establishments, tmpEstab)
		}
		row, _, err = rows.NextNeo()
	}
	return establishments, nil
}

/*************** Endpoint ***************/
type getRecommendationRequest struct {
	UserID int64 `json:"id"`
}

type getRecommendationResponse struct {
	Establishments	[]Establishment `json:"establishments"`
	Err				string        `json:"err,omitempty"`
}

func GetRecommendationEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getRecommendationRequest)
		establishments, err := svc.GetRecommendation(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetRecommendationEndpoint 1 : ", err.Error())
			return getRecommendationResponse{establishments, err.Error()}, nil
		}
		return getRecommendationResponse{establishments, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetRecommendationRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPGetRecommendationResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getRecommendationResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetRecommendationHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/recommendation/{id}").Handler(httptransport.NewServer(
		endpoints.GetRecommendationEndpoint,
		DecodeHTTPGetRecommendationRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetRecommendation", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetRecommendation(ctx context.Context, userID int64) ([]Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getRecommendation",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetRecommendation(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetRecommendation(ctx context.Context, userID int64) ([]Establishment, error) {
	v, err := mw.next.GetRecommendation(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetRecommendationEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetRecommendation")
		csLogger := log.With(logger, "method", "GetRecommendation")

		csEndpoint = GetRecommendationEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetRecommendation")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetRecommendation(ctx context.Context, userID int64) ([]Establishment, error) {
	var ets []Establishment

	request := getRecommendationRequest{UserID: userID}
	response, err := e.GetRecommendationEndpoint(ctx, request)
	if err != nil {
		return ets, err
	}
	ets = response.(getRecommendationResponse).Establishments
	return ets, str2err(response.(getRecommendationResponse).Err)
}

func ClientGetRecommendation(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/recommendation/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetRecommendationResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetRecommendation")(ceEndpoint)
	return ceEndpoint, nil
}
