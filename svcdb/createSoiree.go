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
func (s Service) CreateSoiree(c context.Context, menuID, establishmentID int64, u Soiree) (Soiree, error) {
	var soiree Soiree

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateSoiree (WaitConnection) : " + err.Error())
		return soiree, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (m:MENU), (e:ESTABLISHMENT)
		WHERE ID(m) = {mid} AND ID(e) = {eid}
		CREATE (s:SOIREE {
		Desc: {Desc},
		Begin: {Begin},
		End: {End}
	}),
	(e)-[:SPAWNED]->(s),
	(s)-[:USE]->(m)
	 RETURN s`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateSoiree (PrepareNeo) : " + err.Error())
		return soiree, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"mid":   menuID,
		"eid":   establishmentID,
		"Desc":  u.Desc,
		"Begin": u.Begin.Format(timeForm),
		"End":   u.End.Format(timeForm),
	})

	if err != nil {
		fmt.Println("CreateSoiree (QueryNeo) : " + err.Error())
		return soiree, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateSoiree (NextNeo) : " + err.Error())
		return soiree, err
	}

	(&soiree).NodeToSoiree(data[0].(graph.Node))
	return soiree, err
}

/*************** Endpoint ***************/
type createSoireeRequest struct {
	MenuID          int64  `json:"menuID"`
	EstablishmentID int64  `json"establishmentID"`
	Soiree          Soiree `json:"soiree"`
}

type createSoireeResponse struct {
	Soiree Soiree `json:"soiree"`
	Err    string `json:"err,omitempty"`
}

func CreateSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createSoireeRequest)
		soiree, err := svc.CreateSoiree(ctx, req.MenuID, req.EstablishmentID, req.Soiree)
		if err != nil {
			fmt.Println("Error CreateSoireeEndpoint 1 : ", err.Error())
			return createSoireeResponse{soiree, err.Error()}, nil
		}

		soiree.Menu, err = svc.GetMenuFromSoiree(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error CreateSoireeEndpoint 2 : ", err.Error())
			return createSoireeResponse{soiree, err.Error()}, nil
		}

		return createSoireeResponse{soiree, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createSoireeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPCreateSoireeRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPCreateSoireeResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func CreateSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soirees/create_soiree").Handler(httptransport.NewServer(
		endpoints.CreateSoireeEndpoint,
		DecodeHTTPCreateSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateSoiree(ctx context.Context, menuID, establishmentID int64, u Soiree) (Soiree, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createSoiree",
			"menuID", menuID,
			"establishmentID", establishmentID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateSoiree(ctx, menuID, establishmentID, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateSoiree(ctx context.Context, menuID, establishmentID int64, u Soiree) (Soiree, error) {
	v, err := mw.next.CreateSoiree(ctx, menuID, establishmentID, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateSoiree")
		csLogger := log.With(logger, "method", "CreateSoiree")

		csEndpoint = CreateSoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateSoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateSoiree(ctx context.Context, menuID, establishmentID int64, et Soiree) (Soiree, error) {
	request := createSoireeRequest{MenuID: menuID, EstablishmentID: establishmentID, Soiree: et}
	response, err := e.CreateSoireeEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error CreateSoiree : ", err.Error())
		return et, err
	}
	return response.(createSoireeResponse).Soiree, str2err(response.(createSoireeResponse).Err)
}

func ClientCreateSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/soirees/create_soiree"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateSoiree")(ceEndpoint)
	return ceEndpoint, nil
}
