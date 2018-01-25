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
func (s Service) GetUserPreferences(_ context.Context, userID int64) ([]Preference, error) {
	var preferences []Preference
	var tmpPreferences Preference

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUserPreferences (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (u:USER)-[p:PREFER]->(e:ESTABLISHMENT_TYPE) WHERE ID(u) = {id} RETURN (e)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetUserPreferences (PrepareNeo) : " + err.Error())
		return preferences, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})

	if err != nil {
		fmt.Println("GetUserPreferences (QueryNeo) : " + err.Error())
		return preferences, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetUserPreferences (NextNeo)")
			panic(err)
		} else if err != io.EOF {
			(&tmpPreferences).NodeToPreference(row[0].(graph.Node))

			preferences = append(preferences, tmpPreferences)
		}
		row, _, err = rows.NextNeo()
	}

	return preferences, nil
}

/*************** Endpoint ***************/
type getUserPreferencesRequest struct {
	UserID int64 `json:"id"`
}

type getUserPreferencesResponse struct {
	Preference []Preference `json:"preferences"`
	Err        string       `json:"err,omitempty"`
}

func GetUserPreferencesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserPreferencesRequest)
		establishment, err := svc.GetUserPreferences(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserPreferencesEndpoint : ", err.Error())
			return getUserPreferencesResponse{establishment, err.Error()}, nil
		}
		return getUserPreferencesResponse{establishment, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserPreferencesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserPreferencesRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserPreferencesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserPreferencesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserPreferencesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/preferences").Handler(httptransport.NewServer(
		endpoints.GetUserPreferencesEndpoint,
		DecodeHTTPGetUserPreferencesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserPreferences", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserPreferences(ctx context.Context, userID int64) ([]Preference, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserPreferences",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserPreferences(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserPreferences(ctx context.Context, userID int64) ([]Preference, error) {
	v, err := mw.next.GetUserPreferences(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserPreferencesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetUserPreferences")
		gefmLogger := log.With(logger, "method", "GetUserPreferences")

		gefmEndpoint = GetUserPreferencesEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetUserPreferences")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserPreferences(ctx context.Context, userID int64) ([]Preference, error) {
	var s []Preference

	request := getUserPreferencesRequest{UserID: userID}
	response, err := e.GetUserPreferencesEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetUserPreferences : ", err.Error())
		return s, err
	}
	s = response.(getUserPreferencesResponse).Preference
	return s, str2err(response.(getUserPreferencesResponse).Err)
}

func EncodeHTTPGetUserPreferencesRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUserPreferencesRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetUserPreferencesRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserPreferences(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/{UserID:[0-9]+}/preferences"),
		EncodeHTTPGetUserPreferencesRequest,
		DecodeHTTPGetUserPreferencesResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserPreferences")(gefmEndpoint)
	return gefmEndpoint, nil
}
