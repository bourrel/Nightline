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
func (s Service) GetUserFriends(_ context.Context, userID int64) ([]Profile, error) {
	var menus []Profile
	var tmpProfile Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUserFriends (WaitConnection) : " + err.Error())
		return menus, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (u:USER)-[:KNOW]-(friends:USER) WHERE ID(u) = {id} RETURN friends`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GetUserFriends (PrepareNeo) : " + err.Error())
		return menus, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})

	if err != nil {
		fmt.Println("Error GetUserFriends (QueryNeo) : " + err.Error())
		return menus, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("Error GetAllEstablishments (NextNeo)")
			panic(err)
		} else if err != io.EOF {
			(&tmpProfile).NodeToProfile(row[0].(graph.Node))
			menus = append(menus, tmpProfile)
		}
		row, _, err = rows.NextNeo()
	}

	return menus, nil
}

/*************** Endpoint ***************/
type getUserFriendsRequest struct {
	UserID int64 `json:"id"`
}

type getUserFriendsResponse struct {
	Profile []Profile `json:"friends"`
	Err     string    `json:"err,omitempty"`
}

func GetUserFriendsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserFriendsRequest)
		menu, err := svc.GetUserFriends(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserFriendsEndpoint : ", err.Error())
			return getUserFriendsResponse{Profile: menu, Err: err.Error()}, nil
		}
		return getUserFriendsResponse{Profile: menu, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserFriendsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserFriendsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserFriendsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserFriendsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserFriendsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID}/friends").Handler(httptransport.NewServer(
		endpoints.GetUserFriendsEndpoint,
		DecodeHTTPGetUserFriendsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserFriends", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserFriends(ctx context.Context, userID int64) ([]Profile, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserFriends",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserFriends(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserFriends(ctx context.Context, userID int64) ([]Profile, error) {
	v, err := mw.next.GetUserFriends(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserFriendsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetUserFriends")
		gefmLogger := log.With(logger, "method", "GetUserFriends")

		gefmEndpoint = GetUserFriendsEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetUserFriends")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserFriends(ctx context.Context, userID int64) ([]Profile, error) {
	var menu []Profile

	request := getUserFriendsRequest{UserID: userID}
	response, err := e.GetUserFriendsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetUserFriends : ", err.Error())
		return menu, nil
	}
	menu = response.(getUserFriendsResponse).Profile
	return menu, str2err(response.(getUserFriendsResponse).Err)
}

func EncodeHTTPGetUserFriendsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := strconv.FormatInt(request.(getUserFriendsRequest).UserID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetUserFriendsRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserFriends(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/{UserID}/friends"),
		EncodeHTTPGetUserFriendsRequest,
		DecodeHTTPGetUserFriendsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserFriends")(gefmEndpoint)
	return gefmEndpoint, nil
}
