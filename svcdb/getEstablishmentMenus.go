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
func (s Service) GetEstablishmentMenus(_ context.Context, estabID int64) ([]Menu, error) {
	var menus []Menu
	var tmpMenu Menu

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishmentMenus (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:GOT]->(m:MENU) WHERE ID(e) = {id} RETURN (m)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetEstablishmentMenus (PrepareNeo) : " + err.Error())
		return menus, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("GetEstablishmentMenus (QueryNeo) : " + err.Error())
		return menus, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetEstablishmentMenus (---) : " + err.Error())
			panic(err)
		} else if err != io.EOF {
			(&tmpMenu).NodeToMenu(row[0].(graph.Node))
			menus = append(menus, tmpMenu)
		}
		row, _, err = rows.NextNeo()
	}

	return menus, nil
}

/*************** Endpoint ***************/
type getEstablishmentMenusRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmentMenusResponse struct {
	Menus []Menu `json:"menus"`
	Err   string `json:"err,omitempty"`
}

func GetEstablishmentMenusEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentMenusRequest)
		menus, err := svc.GetEstablishmentMenus(ctx, req.EstabID)

		for i := 0; i < len(menus); i++ {
			menus[i].Consommations, err = svc.GetMenuConsos(ctx, menus[i].ID)
		}

		if err != nil {
			return getEstablishmentMenusResponse{Menus: menus, Err: err.Error()}, nil
		}
		return getEstablishmentMenusResponse{Menus: menus, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentMenusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentMenusRequest

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

func DecodeHTTPGetEstablishmentMenusResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentMenusResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentMenusHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabID}/menus").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentMenusEndpoint,
		DecodeHTTPGetEstablishmentMenusRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentMenus", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentMenus(ctx context.Context, estabID int64) ([]Menu, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentMenus",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentMenus(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentMenus(ctx context.Context, estabID int64) ([]Menu, error) {
	v, err := mw.next.GetEstablishmentMenus(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentMenusEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentMenus")
		gefmLogger := log.With(logger, "method", "GetEstablishmentMenus")

		gefmEndpoint = GetEstablishmentMenusEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentMenus")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentMenus(ctx context.Context, estabID int64) ([]Menu, error) {
	var s []Menu

	request := getEstablishmentMenusRequest{EstabID: estabID}
	response, err := e.GetEstablishmentMenusEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getEstablishmentMenusResponse).Menus
	return s, str2err(response.(getEstablishmentMenusResponse).Err)
}

func EncodeHTTPGetEstablishmentMenusRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentMenusRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetEstablishmentMenus(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{EstabID}/menus"),
		EncodeHTTPGetEstablishmentMenusRequest,
		DecodeHTTPGetEstablishmentMenusResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentMenus")(gefmEndpoint)
	return gefmEndpoint, nil
}
