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
func (s Service) GetMenuFromSoiree(_ context.Context, soireeID int64) (Menu, error) {
	var menu Menu

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetMenuFromSoiree (WaitConnection) : " + err.Error())
		return menu, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (s:SOIREE)-[:USE]->(m:MENU) WHERE ID(s) = {id} RETURN m`)

	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GetMenuFromSoiree (PrepareNeo) : " + err.Error())
		return menu, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})

	if err != nil {
		fmt.Println("Error GetMenuFromSoiree (QueryNeo) : " + err.Error())
		return menu, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetMenuFromSoiree (NextNeo) : " + err.Error())
		return menu, err
	}

	(&menu).NodeToMenu(row[0].(graph.Node))

	return menu, nil
}

/*************** Endpoint ***************/
type getMenuFromSoireeRequest struct {
	EstabID int64 `json:"id"`
}

type getMenuFromSoireeResponse struct {
	Menu Menu   `json:"menu"`
	Err  string `json:"err,omitempty"`
}

func GetMenuFromSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuFromSoireeRequest)
		menu, err := svc.GetMenuFromSoiree(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetMenuFromSoireeEndpoint 1 : ", err.Error())
			return getMenuFromSoireeResponse{Menu: menu, Err: err.Error()}, nil
		}

		menu.Consommations, err = svc.GetMenuConsos(ctx, menu.ID)
		if err != nil {
			fmt.Println("Error GetMenuFromSoireeEndpoint 2 : ", err.Error())
			return getMenuFromSoireeResponse{Menu: menu, Err: err.Error()}, nil
		}

		return getMenuFromSoireeResponse{menu, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuFromSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuFromSoireeRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeRequest 1 : ", err.Error())
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).EstabID = soireeID

	return request, nil
}

func DecodeHTTPGetMenuFromSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuFromSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetMenuFromSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/menuFromSoiree/{EstabID}").Handler(httptransport.NewServer(
		endpoints.GetMenuFromSoireeEndpoint,
		DecodeHTTPGetMenuFromSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenuFromSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenuFromSoiree(ctx context.Context, soireeID int64) (Menu, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getMenuFromSoiree",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetMenuFromSoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenuFromSoiree(ctx context.Context, soireeID int64) (Menu, error) {
	v, err := mw.next.GetMenuFromSoiree(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetMenuFromSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetMenuFromSoiree")
		gefmLogger := log.With(logger, "method", "GetMenuFromSoiree")

		gefmEndpoint = GetMenuFromSoireeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetMenuFromSoiree")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetMenuFromSoiree(ctx context.Context, soireeID int64) (Menu, error) {
	var menu Menu

	request := getMenuFromSoireeRequest{EstabID: soireeID}
	response, err := e.GetMenuFromSoireeEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetMenuFromSoiree : ", err.Error())
		return menu, nil
	}
	menu = response.(getMenuFromSoireeResponse).Menu
	return menu, str2err(response.(getMenuFromSoireeResponse).Err)
}

func EncodeHTTPGetMenuFromSoireeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := strconv.FormatInt(request.(getMenuFromSoireeRequest).EstabID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetMenuFromSoireeRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetMenuFromSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/menuFromSoiree/{EstabID}"),
		EncodeHTTPGetMenuFromSoireeRequest,
		DecodeHTTPGetMenuFromSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetMenuFromSoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
