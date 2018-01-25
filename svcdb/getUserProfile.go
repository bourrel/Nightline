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
func (s Service) GetUserProfile(_ context.Context, userID int64) (Profile, error) {
	var user Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUserProfile (WaitConnection) : " + err.Error())
		return user, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)
		WHERE ID(u) = {userID}
		RETURN u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetUserProfile (PrepareNeo) : " + err.Error())
		return user, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"userID": userID,
	})
	if err != nil {
		fmt.Println("GetUserProfile (QueryNeo) : " + err.Error())
		return user, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetUserProfile (NextNeo) : " + err.Error())
		return user, err
	}

	(&user).NodeToProfile(row[0].(graph.Node))
	return user, nil
}

/*************** Endpoint ***************/
type getUserProfileRequest struct {
	UserID int64 `json:"user"`
}

type getUserProfileResponse struct {
	Profile Profile `json:"user"`
	Err     string  `json:"err,omitempty"`
}

func GetUserProfileEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserProfileRequest)
		user, err := svc.GetUserProfile(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserProfileEndpoint : ", err.Error())
			return getUserProfileResponse{user, err.Error()}, nil
		}
		return getUserProfileResponse{user, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserProfileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserProfileRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserProfileRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["userID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserProfileRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserProfileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserProfileResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserProfileResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserProfileHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/profile/{userID}").Handler(httptransport.NewServer(
		endpoints.GetUserProfileEndpoint,
		DecodeHTTPGetUserProfileRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserProfile", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserProfile(ctx context.Context, userID int64) (Profile, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserProfile",
			"user", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserProfile(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserProfile(ctx context.Context, userID int64) (Profile, error) {
	v, err := mw.next.GetUserProfile(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserProfileEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUserProfile")
		csLogger := log.With(logger, "method", "GetUserProfile")

		csEndpoint = GetUserProfileEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUserProfile")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserProfile(ctx context.Context, userID int64) (Profile, error) {
	var p Profile

	request := getUserProfileRequest{UserID: userID}
	response, err := e.GetUserProfileEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error client GetUserProfile : ", err.Error())
		return p, err
	}
	p = response.(getUserProfileResponse).Profile
	return p, str2err(response.(getUserProfileResponse).Err)
}

func EncodeHTTPGetUserProfileRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUserProfileRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("userID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetUserProfileRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserProfile(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/profile/{userID}"),
		EncodeHTTPGetUserProfileRequest,
		DecodeHTTPGetUserProfileResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserProfile")(gefmEndpoint)
	return gefmEndpoint, nil
}
