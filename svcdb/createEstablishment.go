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
func (s Service) CreateEstablishment(_ context.Context, e Establishment, proID int64) (Establishment, error) {
	var establishment Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateEstablishment (WaitConnection) : " + err.Error())
		return establishment, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (p:PRO) WHERE ID(p) = {pID}
		MATCH (t:ESTABLISHMENT_TYPE) WHERE t.Name = {Type}
		CREATE (p)-[:OWN]->(e:ESTABLISHMENT {
			Name: {Name},
			Address: {Address},
			Lat: {Lat},
			Long: {Long},
			Description: {Description},
			Image: {Image}
		})-[:IS]->(t)
		RETURN e
	`)

	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateEstablishment (PrepareNeo) : " + err.Error())
		return establishment, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"Name":        e.Name,
		"Address":     e.Address,
		"Lat":         e.Lat,
		"Long":        e.Long,
		"Type":        e.Type,
		"Description": e.Description,
		"Image":       e.Image,
		"pID":         proID,
	})

	if err != nil {
		fmt.Println("CreateEstablishment (QueryNeo) : " + err.Error())
		return establishment, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateEstablishment (NextNeo) : " + err.Error())
		return establishment, err
	}
	(&establishment).NodeToEstablishment(row[0].(graph.Node))

	return establishment, err
}

/*************** Endpoint ***************/
type createEstablishmentRequest struct {
	Establishment Establishment `json:"establishment"`
	ProID         int64         `json:"proID"`
}

type createEstablishmentResponse struct {
	Establishment Establishment `json:"establishment"`
	Err           string        `json:"err,omitempty"`
}

func CreateEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createEstablishmentRequest)
		establishment, err := svc.CreateEstablishment(ctx, req.Establishment, req.ProID)
		if err != nil {
			return createEstablishmentResponse{establishment, err.Error()}, nil
		}
		return createEstablishmentResponse{establishment, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createEstablishmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/establishments/create_establishment").Handler(httptransport.NewServer(
		endpoints.CreateEstablishmentEndpoint,
		DecodeHTTPCreateEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateEstablishment", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateEstablishment(ctx context.Context, u Establishment, proID int64) (Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createEstablishment",
			"establishment", u,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateEstablishment(ctx, u, proID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateEstablishment(ctx context.Context, u Establishment, proID int64) (Establishment, error) {
	v, err := mw.next.CreateEstablishment(ctx, u, proID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateEstablishment")
		csLogger := log.With(logger, "method", "CreateEstablishment")

		csEndpoint = CreateEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) CreateEstablishment(ctx context.Context, et Establishment, proID int64) (Establishment, error) {
	request := createEstablishmentRequest{Establishment: et, ProID: proID}
	response, err := e.CreateEstablishmentEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	return response.(createEstablishmentResponse).Establishment, str2err(response.(createEstablishmentResponse).Err)
}

func ClientCreateEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/establishments/create_establishment"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateEstablishment")(ceEndpoint)
	return ceEndpoint, nil
}
