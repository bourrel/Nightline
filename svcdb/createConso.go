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
func (s Service) CreateConso(ctx context.Context, establishmentID, menuID int64, c Conso) (Conso, error) {
	var conso Conso

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateConso (WaitConnection) : " + err.Error())
		return conso, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (et:ESTABLISHMENT)-[:GOT]->(m:MENU) WHERE ID(et) = {eid} AND ID(m) = {mid}
		CREATE (c:CONSO {
			Name: {name},
			Desc: {desc},
			Price: {price},
			Picture: {picture}
		}),
		(et)-[:GOT]->(c),
		(m)-[:USE]->(c)
		RETURN c`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateConso (PrepareNeo) : " + err.Error())
		return conso, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"eid":     establishmentID,
		"mid":     menuID,
		"name":    c.Name,
		"desc":    c.Description,
		"price":   c.Price,
		"picture": c.Picture,
	})

	if err != nil {
		fmt.Println("CreateConso (QueryNeo) : " + err.Error())
		return conso, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateConso (NextNeo) : " + err.Error())
		return conso, err
	}

	_ = establishmentID
	_ = menuID

	(&conso).NodeToConso(data[0].(graph.Node))
	return conso, err
}

/*************** Endpoint ***************/
type createConsoRequest struct {
	MenuID          int64 `json:"menuID"`
	EstablishmentID int64 `json:"establishmentID"`
	Conso           Conso `json:"conso"`
}

type createConsoResponse struct {
	Conso Conso  `json:"conso"`
	Err   string `json:"err,omitempty"`
}

func CreateConsoEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createConsoRequest)
		conso, err := svc.CreateConso(ctx, req.EstablishmentID, req.MenuID, req.Conso)

		// Create node
		if err != nil {
			fmt.Println("Error CreateConsoEndpoint 1 : ", err.Error())
			return createConsoResponse{Conso: conso, Err: err.Error()}, nil
		}

		return createConsoResponse{Conso: conso, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateConsoRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createConsoRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPCreateConsoRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateConsoResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createConsoResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPCreateConsoResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func CreateConsoHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/menus/consos").Handler(httptransport.NewServer(
		endpoints.CreateConsoEndpoint,
		DecodeHTTPCreateConsoRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateConso", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateConso(ctx context.Context, establishmentID, menuID int64, c Conso) (Conso, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createConso",
			"menuID", menuID,
			"establishmentID", establishmentID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateConso(ctx, establishmentID, menuID, c)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateConso(ctx context.Context, establishmentID, menuID int64, c Conso) (Conso, error) {
	v, err := mw.next.CreateConso(ctx, establishmentID, menuID, c)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateConsoEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateConso")
		csLogger := log.With(logger, "method", "CreateConso")

		csEndpoint = CreateConsoEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateConso")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateConso(ctx context.Context, establishmentID, menuID int64, c Conso) (Conso, error) {
	request := createConsoRequest{EstablishmentID: establishmentID, MenuID: menuID, Conso: c}
	response, err := e.CreateConsoEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error CreateConso : ", err.Error())
		return c, err
	}
	return response.(createConsoResponse).Conso, str2err(response.(createConsoResponse).Err)
}

func ClientCreateConso(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/menus/consos"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateConsoResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateConso")(ceEndpoint)
	return ceEndpoint, nil
}
