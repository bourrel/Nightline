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
func (s Service) GetProEstablishments(_ context.Context, soireeID int64) ([]Establishment, error) {
	var establishments []Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetProEstablishments (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (p:PRO)-[:OWN]->(e:ESTABLISHMENT) WHERE ID(p) = {id}
		OPTIONAL MATCH (e)<-[r:RATE]-(u:USER)
		RETURN e, AVG(r.value)
	`)

	defer stmt.Close()
	if err != nil {
		fmt.Println("GetProEstablishments (PrepareNeo)")
		panic(err)
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})
	if err != nil {
		fmt.Println("GetProEstablishments (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpEstablishment Establishment

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetProEstablishments (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpEstablishment).NodeToEstablishment(row[0].(graph.Node))
			if rate, ok := row[1].(float64); ok {
				(&tmpEstablishment).Rate = rate
			}
			establishments = append(establishments, tmpEstablishment)
		}
		row, _, err = rows.NextNeo()
	}
	return establishments, nil
}

/*************** Endpoint ***************/
type getEstablishmentsBySoireeRequest struct {
	SoireeID int64 `json:"id"`
}

type getEstablishmentsBySoireeResponse struct {
	Establishments []Establishment `json:"establishments"`
	Err            string          `json:"err,omitempty"`
}

func GetProEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentsBySoireeRequest)
		soirees, err := svc.GetProEstablishments(ctx, req.SoireeID)
		if err != nil {
			return getEstablishmentsBySoireeResponse{soirees, err.Error()}, nil
		}
		return getEstablishmentsBySoireeResponse{soirees, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentsBySoireeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).SoireeID = soireeID

	return request, nil
}

func DecodeHTTPGetProEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentsBySoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetProEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishmentsBySoiree/{SoireeID}").Handler(httptransport.NewServer(
		endpoints.GetProEstablishmentsEndpoint,
		DecodeHTTPGetProEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetProEstablishments", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetProEstablishments(ctx context.Context, soireeID int64) ([]Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentsBySoiree",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetProEstablishments(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetProEstablishments(ctx context.Context, soireeID int64) ([]Establishment, error) {
	v, err := mw.next.GetProEstablishments(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetProEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetProEstablishments")
		csLogger := log.With(logger, "method", "GetProEstablishments")

		csEndpoint = GetProEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetProEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetProEstablishments(ctx context.Context, soireeID int64) ([]Establishment, error) {
	var et []Establishment

	request := getEstablishmentsBySoireeRequest{SoireeID: soireeID}
	response, err := e.GetProEstablishmentsEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getEstablishmentsBySoireeResponse).Establishments
	return et, str2err(response.(getEstablishmentsBySoireeResponse).Err)
}

func EncodeHTTPGetProEstablishmentsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentsBySoireeRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("SoireeID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetProEstablishments(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishmentsBySoiree/{SoireeID}"),
		EncodeHTTPGetProEstablishmentsRequest,
		DecodeHTTPGetProEstablishmentsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetProEstablishments")(ceEndpoint)
	return ceEndpoint, nil
}
