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
func (s Service) GetUserSuccess(_ context.Context, userID int64) ([]Success, error) {
	var success []Success
	var tmpSuccess Success

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUserSuccess (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (s:SUCCESS) OPTIONAL MATCH (u:USER)-[g:GOT]->(s) WHERE ID(u) = {id} RETURN s, g, u`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetUserSuccess (PrepareNeo) : " + err.Error())
		return success, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})

	if err != nil {
		fmt.Println("GetUserSuccess (QueryNeo) : " + err.Error())
		return success, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetUserSuccess (NextNeo)")
			panic(err)
		} else if err != io.EOF {
			for i := 0; i < len(row); i++ {
				if node, ok := row[i].(graph.Node); ok {
					if node.Labels[0] == "SUCCESS" {
						(&tmpSuccess).NodeToSuccess(node)
						success = append(success, tmpSuccess)
					}
				} else if relation, ok := row[i].(graph.Relationship); ok {
					if success[len(success)-1].ID == relation.EndNodeIdentity {
						success[len(success)-1].Active = true
					}
				}
			}
		}
		row, _, err = rows.NextNeo()
	}

	return success, nil
}

/*************** Endpoint ***************/
type getUserSuccessRequest struct {
	UserID int64 `json:"id"`
}

type getUserSuccessResponse struct {
	Success []Success `json:"success"`
	Err     string    `json:"err,omitempty"`
}

func GetUserSuccessEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserSuccessRequest)
		success, err := svc.GetUserSuccess(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserSuccessEndpoint : ", err.Error())
			return getUserSuccessResponse{success, err.Error()}, nil
		}
		return getUserSuccessResponse{success, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserSuccessRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserSuccessRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserSuccessResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserSuccessResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessResponse 2 : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserSuccessHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/success").Handler(httptransport.NewServer(
		endpoints.GetUserSuccessEndpoint,
		DecodeHTTPGetUserSuccessRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserSuccess", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserSuccess(ctx context.Context, userID int64) ([]Success, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserSuccess",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserSuccess(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserSuccess(ctx context.Context, userID int64) ([]Success, error) {
	v, err := mw.next.GetUserSuccess(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserSuccessEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetUserSuccess")
		gefmLogger := log.With(logger, "method", "GetUserSuccess")

		gefmEndpoint = GetUserSuccessEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetUserSuccess")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserSuccess(ctx context.Context, userID int64) ([]Success, error) {
	var s []Success

	request := getUserSuccessRequest{UserID: userID}
	response, err := e.GetUserSuccessEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error Client GetUserSuccess : ", err.Error())
		return s, err
	}
	s = response.(getUserSuccessResponse).Success
	return s, str2err(response.(getUserSuccessResponse).Err)
}

func EncodeHTTPGetUserSuccessRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUserSuccessRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserSuccess(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/{UserID:[0-9]+}/success"),
		EncodeHTTPGetUserSuccessRequest,
		DecodeHTTPGetUserSuccessResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserSuccess")(gefmEndpoint)
	return gefmEndpoint, nil
}
