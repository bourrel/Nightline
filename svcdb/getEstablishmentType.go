package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
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
func (s Service) GetEstablishmentType(_ context.Context, estabID int64) (string, error) {
	var tmp EstablishmentType

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishmentType (WaitConnection) : " + err.Error())
		return "", err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:IS]->(t:ESTABLISHMENT_TYPE) WHERE ID(e) = {id} RETURN (t)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetEstablishmentType (PrepareNeo) : " + err.Error())
		return "", err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("GetEstablishmentType (QueryNeo) : " + err.Error())
		return "", err
	}

	// No result != error
	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetEstablishmentType (NextNeo) : " + err.Error())
		return "", nil
	}

	(&tmp).NodeToEstablishmentType(row[0].(graph.Node))
	return tmp.Name, nil
}

/*************** Endpoint ***************/
type getEstablishmenTypeRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmenTypeResponse struct {
	string string `json:"soiree"`
	Err    string `json:"err,omitempty"`
}

func GetEstablishmentTypeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmenTypeRequest)
		soiree, err := svc.GetEstablishmentType(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetEstablishmentTypeEndpoint 1 : ", err.Error())
			return getEstablishmenTypeResponse{soiree, err.Error()}, nil
		}

		return getEstablishmenTypeResponse{soiree, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentTypeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmenTypeRequest

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

func DecodeHTTPGetEstablishmentTypeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmenTypeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentTypeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishment/{EstabID}/type").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentTypeEndpoint,
		DecodeHTTPGetEstablishmentTypeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentType", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentType(ctx context.Context, estabID int64) (string, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmenType",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentType(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentType(ctx context.Context, estabID int64) (string, error) {
	v, err := mw.next.GetEstablishmentType(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentTypeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentType")
		gefmLogger := log.With(logger, "method", "GetEstablishmentType")

		gefmEndpoint = GetEstablishmentTypeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentType")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentType(ctx context.Context, estabID int64) (string, error) {
	var s string

	request := getEstablishmenTypeRequest{EstabID: estabID}
	response, err := e.GetEstablishmentTypeEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getEstablishmenTypeResponse).string
	return s, str2err(response.(getEstablishmenTypeResponse).Err)
}

func EncodeHTTPGetEstablishmentTypeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmenTypeRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetEstablishmentType(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishment/{EstabID}/type"),
		EncodeHTTPGetEstablishmentTypeRequest,
		DecodeHTTPGetEstablishmentTypeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentType")(gefmEndpoint)
	return gefmEndpoint, nil
}
