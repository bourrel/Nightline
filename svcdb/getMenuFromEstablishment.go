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
func (s Service) GetMenuFromEstablishment(_ context.Context, estabID int64) ([]Menu, error) {
	var menus []Menu
	var tmpMenu Menu

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetMenuFromEstablishment (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:GOT]->(m:MENU) WHERE ID(e) = {id} RETURN m`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GetMenuFromEstablishment (PrepareNeo) : " + err.Error())
		return menus, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("Error GetMenuFromEstablishment (QueryNeo) : " + err.Error())
		return menus, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("Error GetAllEstablishments (NextNeo)")
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
type getMenuFromEstablishmentRequest struct {
	EstabID int64 `json:"id"`
}

type getMenuFromEstablishmentResponse struct {
	Menu []Menu `json:"menu"`
	Err  string `json:"err,omitempty"`
}

func GetMenuFromEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuFromEstablishmentRequest)
		menus, err := svc.GetMenuFromEstablishment(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetMenuFromEstablishmentEndpoint : ", err.Error())
			return getMenuFromEstablishmentResponse{Menu: menus, Err: err.Error()}, nil
		}

		for i := 0; i < len(menus); i++ {
			menus[i].Consommations, err = svc.GetMenuConsos(ctx, menus[i].ID)

			if err != nil {
				fmt.Println("Error GetMenuFromEstablishmentEndpoint 2 : ", err.Error())
				return getMenuFromEstablishmentResponse{Menu: menus, Err: err.Error()}, nil
			}
		}

		return getMenuFromEstablishmentResponse{menus, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuFromEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuFromEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentRequest 1 : ", err.Error())
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetMenuFromEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuFromEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetMenuFromEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/menuFromEstablishment/{EstabID}").Handler(httptransport.NewServer(
		endpoints.GetMenuFromEstablishmentEndpoint,
		DecodeHTTPGetMenuFromEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenuFromEstablishment", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]Menu, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getMenuFromEstablishment",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetMenuFromEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]Menu, error) {
	v, err := mw.next.GetMenuFromEstablishment(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetMenuFromEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetMenuFromEstablishment")
		gefmLogger := log.With(logger, "method", "GetMenuFromEstablishment")

		gefmEndpoint = GetMenuFromEstablishmentEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetMenuFromEstablishment")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]Menu, error) {
	var menu []Menu

	request := getMenuFromEstablishmentRequest{EstabID: estabID}
	response, err := e.GetMenuFromEstablishmentEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetMenuFromEstablishment : ", err.Error())
		return menu, nil
	}
	menu = response.(getMenuFromEstablishmentResponse).Menu
	return menu, str2err(response.(getMenuFromEstablishmentResponse).Err)
}

func EncodeHTTPGetMenuFromEstablishmentRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := strconv.FormatInt(request.(getMenuFromEstablishmentRequest).EstabID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetMenuFromEstablishmentRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetMenuFromEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/menuFromEstablishment/{EstabID}"),
		EncodeHTTPGetMenuFromEstablishmentRequest,
		DecodeHTTPGetMenuFromEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetMenuFromEstablishment")(gefmEndpoint)
	return gefmEndpoint, nil
}
