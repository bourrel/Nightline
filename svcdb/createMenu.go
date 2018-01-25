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
func (s Service) CreateMenu(c context.Context, establishmentID int64, u Menu) (Menu, error) {
	var menu Menu

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateMenu (WaitConnection) : " + err.Error())
		return menu, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (et:ESTABLISHMENT) WHERE ID(et) = {id}
		CREATE (mn:MENU {
			Name: {name},
			Desc: {desc}
		}),
		(et)-[:GOT {
			Display: {display}
		}]->(mn)
		RETURN mn`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateMenu (PrepareNeo) : " + err.Error())
		return menu, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"name":    u.Name,
		"desc":    u.Desc,
		"display": true,
		"id":      establishmentID,
	})

	if err != nil {
		fmt.Println("CreateMenu (QueryNeo) : " + err.Error())
		return menu, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateMenu (NextNeo) : " + err.Error())
		return menu, err
	}

	(&menu).NodeToMenu(data[0].(graph.Node))
	return menu, err
}

/*************** Endpoint ***************/
type createMenuRequest struct {
	EstablishmentID int64 `json"establishmentID"`
	Menu            Menu  `json:"menu"`
}

type createMenuResponse struct {
	Menu Menu   `json:"menu"`
	Err  string `json:"err,omitempty"`
}

func CreateMenuEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createMenuRequest)

		existingMenus, err := svc.GetEstablishmentMenus(ctx, req.EstablishmentID)
		if err != nil {
			fmt.Println("CreateMenuEndpoint 1 : " + err.Error())
			return getGroupResponse{Err: err.Error()}, nil
		}

		for _, existingMenu := range existingMenus {
			if existingMenu.Name == req.Menu.Name {
				return getGroupResponse{Err: "A menu with this name already exists"}, nil
			}
		}

		menu, err := svc.CreateMenu(ctx, req.EstablishmentID, req.Menu)
		if err != nil {
			fmt.Println("CreateMenuEndpoint 2 : " + err.Error())
			return getGroupResponse{Err: err.Error()}, nil
		}

		for i := 0; i < len(req.Menu.Consommations); i++ {
			conso, err := svc.CreateConso(ctx, req.EstablishmentID, menu.ID, req.Menu.Consommations[i])
			if err != nil {
				fmt.Println("CreateMenuEndpoint 3 : " + err.Error())
				continue
			}
			menu.Consommations = append(menu.Consommations, conso)
		}

		if err != nil {
			return createMenuResponse{menu, err.Error()}, nil
		}
		return createMenuResponse{menu, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateMenuRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateMenuResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createMenuResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateMenuHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/menus").Handler(httptransport.NewServer(
		endpoints.CreateMenuEndpoint,
		DecodeHTTPCreateMenuRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateMenu", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateMenu(ctx context.Context, establishmentID int64, u Menu) (Menu, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createMenu",
			"establishmentID", establishmentID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateMenu(ctx, establishmentID, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateMenu(ctx context.Context, establishmentID int64, u Menu) (Menu, error) {
	v, err := mw.next.CreateMenu(ctx, establishmentID, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateMenuEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateMenu")
		csLogger := log.With(logger, "method", "CreateMenu")

		csEndpoint = CreateMenuEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateMenu")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateMenu(ctx context.Context, establishmentID int64, m Menu) (Menu, error) {
	request := createMenuRequest{EstablishmentID: establishmentID, Menu: m}
	response, err := e.CreateMenuEndpoint(ctx, request)
	if err != nil {
		return m, err
	}
	return response.(createMenuResponse).Menu, str2err(response.(createMenuResponse).Err)
}

func ClientCreateMenu(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/menus"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateMenuResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateMenu")(ceEndpoint)
	return ceEndpoint, nil
}
