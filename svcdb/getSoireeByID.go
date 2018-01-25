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
func (s Service) GetSoireeByID(_ context.Context, soireeID int64) (Soiree, error) {
	var soiree Soiree

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetSoireeByID (WaitConnection) : " + err.Error())
		return soiree, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (s:SOIREE)
	WHERE ID(s) = {id}
	RETURN s`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetSoireeByID (PrepareNeo) : " + err.Error())
		return soiree, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})
	if err != nil {
		fmt.Println("GetSoireeByID (QueryNeo) : " + err.Error())
		return soiree, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetSoireeByID (NextNeo) : " + err.Error())
		return soiree, err
	}

	(&soiree).NodeToSoiree(row[0].(graph.Node))
	return soiree, nil
}

/*************** Endpoint ***************/
type getSoireeByIDRequest struct {
	ID int64 `json:"id"`
}

type getSoireeByIDResponse struct {
	Soiree Soiree `json:"soiree"`
	Err    string `json:"err,omitempty"`
}

func GetSoireeByIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSoireeByIDRequest)
		soiree, err := svc.GetSoireeByID(ctx, req.ID)
		if err != nil {
			fmt.Println("Error GetSoireeByIDEndpoint 2 : ", err.Error())
			return getSoireeByIDResponse{soiree, err.Error()}, nil
		}

		soiree.Menu, err = svc.GetMenuFromSoiree(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error GetSoireeByIDEndpoint 2 : ", err.Error())
			return getSoireeByIDResponse{soiree, err.Error()}, nil
		}

		soiree.Friends, err = svc.GetConnectedFriends(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error GetSoireeByIDEndpoint 3 : ", err.Error())
			return getSoireeByIDResponse{soiree, err.Error()}, nil
		}

		return getSoireeByIDResponse{soiree, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireeByIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSoireeByIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	soireeiD, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = soireeiD

	return request, nil
}

func DecodeHTTPGetSoireeByIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSoireeByIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetSoireeByIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soirees/get_soiree/{id}").Handler(httptransport.NewServer(
		endpoints.GetSoireeByIDEndpoint,
		DecodeHTTPGetSoireeByIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireeByID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireeByID(ctx context.Context, soireeID int64) (Soiree, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getSoireeByID",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetSoireeByID(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireeByID(ctx context.Context, soireeID int64) (Soiree, error) {
	v, err := mw.next.GetSoireeByID(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetSoireeByIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSoireeByID")
		csLogger := log.With(logger, "method", "GetSoireeByID")

		csEndpoint = GetSoireeByIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSoireeByID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSoireeByID(ctx context.Context, ID int64) (Soiree, error) {
	var soiree Soiree

	request := getSoireeByIDRequest{ID: ID}
	response, err := e.GetSoireeByIDEndpoint(ctx, request)
	if err != nil {
		return soiree, err
	}
	soiree = response.(getSoireeByIDResponse).Soiree
	return soiree, str2err(response.(getSoireeByIDResponse).Err)
}

func EncodeHTTPGetSoireeByIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getSoireeByIDRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("ID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSoireeByID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/soirees/get_soiree/{ID}"),
		EncodeHTTPGetSoireeByIDRequest,
		DecodeHTTPGetSoireeByIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetSoireeByID")(gefmEndpoint)
	return gefmEndpoint, nil
}
