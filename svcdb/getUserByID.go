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
func (s Service) GetUserByID(_ context.Context, userID int64) (User, error) {
	var user User

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUserByID (WaitConnection) : " + err.Error())
		return user, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (u:USER)
	WHERE ID(u) = {id}
	RETURN u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetUserByID (PrepareNeo) : " + err.Error())
		return user, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})
	if err != nil {
		fmt.Println("GetUserByID (QueryNeo) : " + err.Error())
		return user, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetUserByID (NextNeo) : " + err.Error())
		return user, err
	}

	(&user).NodeToUser(row[0].(graph.Node))
	return user, nil
}

/*************** Endpoint ***************/
type getUserByIDRequest struct {
	ID int64 `json:"id"`
}

type getUserByIDResponse struct {
	User User   `json:"user"`
	Err  string `json:"err,omitempty"`
}

func GetUserByIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserByIDRequest)
		user, err := svc.GetUserByID(ctx, req.ID)
		if err != nil {
			return getUserByIDResponse{user, err.Error()}, nil
		}
		return getUserByIDResponse{user, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserByIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserByIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	useriD, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = useriD

	return request, nil
}

func DecodeHTTPGetUserByIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserByIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetUserByIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/get_user/{id}").Handler(httptransport.NewServer(
		endpoints.GetUserByIDEndpoint,
		DecodeHTTPGetUserByIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserByID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserByID(ctx context.Context, userID int64) (User, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserByID",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserByID(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserByID(ctx context.Context, userID int64) (User, error) {
	v, err := mw.next.GetUserByID(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserByIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUserByID")
		csLogger := log.With(logger, "method", "GetUserByID")

		csEndpoint = GetUserByIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUserByID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserByID(ctx context.Context, ID int64) (User, error) {
	var user User

	request := getUserByIDRequest{ID: ID}
	response, err := e.GetUserByIDEndpoint(ctx, request)
	if err != nil {
		return user, err
	}
	user = response.(getUserByIDResponse).User
	return user, str2err(response.(getUserByIDResponse).Err)
}

func EncodeHTTPGetUserByIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUserByIDRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("ID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserByID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/get_user/{ID}"),
		EncodeHTTPGetUserByIDRequest,
		DecodeHTTPGetUserByIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserByID")(gefmEndpoint)
	return gefmEndpoint, nil
}
