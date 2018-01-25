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
func (s Service) GetConnectedFriends(_ context.Context, soireeID int64) ([]Profile, error) {
	var friends []Profile
	var tmpProfile Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetConnectedFriends (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)-[j:JOIN]->(s:SOIREE)
		OPTIONAL MATCH (u)-[l:LEAVE]->(s)
		WHERE ID (s) = {id}
		RETURN u, count(j), count(l)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetConnectedFriends (PrepareNeo) : " + err.Error())
		return friends, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})
	if err != nil {
		fmt.Println("GetConnectedFriends (QueryNeo) : " + err.Error())
		return friends, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		var joinCount int64
		var leaveCount int64

		if err != nil && err != io.EOF {
			fmt.Println("GetConnectedFriends (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpProfile).NodeToProfile(row[0].(graph.Node))

			if join, ok := row[1].(int64); ok {
				joinCount = join
			}
			if leave, ok := row[2].(int64); ok {
				leaveCount = leave
			}

			if joinCount > leaveCount {
				friends = append(friends, tmpProfile)
				tmpProfile = Profile{}
			}
		}
		row, _, err = rows.NextNeo()
	}

	return friends, nil
}

/*************** Endpoint ***************/
type getConnectedFriendsRequest struct {
	ID int64 `json:"id"`
}

type getConnectedFriendsResponse struct {
	Profile []Profile `json:"soiree"`
	Err     string    `json:"err,omitempty"`
}

func GetConnectedFriendsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConnectedFriendsRequest)
		soiree, err := svc.GetConnectedFriends(ctx, req.ID)
		if err != nil {
			fmt.Println("Error GetConnectedFriendsEndpoint 2 : ", err.Error())
			return getConnectedFriendsResponse{soiree, err.Error()}, nil
		}

		return getConnectedFriendsResponse{soiree, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConnectedFriendsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getConnectedFriendsRequest

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

func DecodeHTTPGetConnectedFriendsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConnectedFriendsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetConnectedFriendsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soirees/get_soiree/{id}").Handler(httptransport.NewServer(
		endpoints.GetConnectedFriendsEndpoint,
		DecodeHTTPGetConnectedFriendsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConnectedFriends", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConnectedFriends(ctx context.Context, soireeID int64) ([]Profile, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getConnectedFriends",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetConnectedFriends(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConnectedFriends(ctx context.Context, soireeID int64) ([]Profile, error) {
	v, err := mw.next.GetConnectedFriends(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetConnectedFriendsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConnectedFriends")
		csLogger := log.With(logger, "method", "GetConnectedFriends")

		csEndpoint = GetConnectedFriendsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConnectedFriends")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConnectedFriends(ctx context.Context, ID int64) ([]Profile, error) {
	var soiree []Profile

	request := getConnectedFriendsRequest{ID: ID}
	response, err := e.GetConnectedFriendsEndpoint(ctx, request)
	if err != nil {
		return soiree, err
	}
	soiree = response.(getConnectedFriendsResponse).Profile
	return soiree, str2err(response.(getConnectedFriendsResponse).Err)
}

func EncodeHTTPGetConnectedFriendsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getConnectedFriendsRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("ID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConnectedFriends(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/soirees/get_soiree/{ID}"),
		EncodeHTTPGetConnectedFriendsRequest,
		DecodeHTTPGetConnectedFriendsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetConnectedFriends")(gefmEndpoint)
	return gefmEndpoint, nil
}
