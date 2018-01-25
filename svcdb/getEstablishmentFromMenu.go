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
func (s Service) GetEstablishmentFromMenu(_ context.Context, menuID int64) (Establishment, error) {
	var establishment Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishmentFromMenu (WaitConnection) : " + err.Error())
		return establishment, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:GOT]->(m:MENU) WHERE ID(m) = {id} RETURN (e)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetEstablishmentFromMenu (PrepareNeo) : " + err.Error())
		return establishment, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": menuID,
	})

	if err != nil {
		fmt.Println("GetEstablishmentFromMenu (QueryNeo) : " + err.Error())
		return establishment, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetEstablishmentFromMenu (NextNeo) : " + err.Error())
		return establishment, err
	}

	(&establishment).NodeToEstablishment(row[0].(graph.Node))
	return establishment, nil
}

/*************** Endpoint ***************/
type getEstablishmentFromMenuRequest struct {
	MenuID int64 `json:"id"`
}

type getEstablishmentFromMenuResponse struct {
	Establishment Establishment `json:"establishment"`
	Err           string        `json:"err,omitempty"`
}

func GetEstablishmentFromMenuEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentFromMenuRequest)
		establishment, err := svc.GetEstablishmentFromMenu(ctx, req.MenuID)
		if err != nil {
			return getEstablishmentFromMenuResponse{establishment, err.Error()}, nil
		}
		return getEstablishmentFromMenuResponse{establishment, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentFromMenuRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentFromMenuRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	menuID, err := strconv.ParseInt(mux.Vars(r)["MenuID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).MenuID = menuID

	return request, nil
}

func DecodeHTTPGetEstablishmentFromMenuResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentFromMenuResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentFromMenuHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishmentsFromMenu/{MenuID}").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentFromMenuEndpoint,
		DecodeHTTPGetEstablishmentFromMenuRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentFromMenu", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentFromMenu(ctx context.Context, menuID int64) (Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentFromMenu",
			"menuID", menuID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentFromMenu(ctx, menuID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentFromMenu(ctx context.Context, menuID int64) (Establishment, error) {
	v, err := mw.next.GetEstablishmentFromMenu(ctx, menuID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentFromMenuEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentFromMenu")
		gefmLogger := log.With(logger, "method", "GetEstablishmentFromMenu")

		gefmEndpoint = GetEstablishmentFromMenuEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentFromMenu")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentFromMenu(ctx context.Context, menuID int64) (Establishment, error) {
	var et Establishment

	request := getEstablishmentFromMenuRequest{MenuID: menuID}
	response, err := e.GetEstablishmentFromMenuEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getEstablishmentFromMenuResponse).Establishment
	return et, str2err(response.(getEstablishmentFromMenuResponse).Err)
}

func EncodeHTTPGetEstablishmentFromMenuRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentFromMenuRequest).MenuID)
	encodedUrl, err := route.Path(r.URL.Path).URL("MenuID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetEstablishmentFromMenu(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishmentsFromMenu/{MenuID}"),
		EncodeHTTPGetEstablishmentFromMenuRequest,
		DecodeHTTPGetEstablishmentFromMenuResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentFromMenu")(gefmEndpoint)
	return gefmEndpoint, nil
}
