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
func (s Service) UpdateUser(c context.Context, new User) (User, error) {
	var user User

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdateUser (WaitConnection) : " + err.Error())
		return user, err
	}
	defer CloseConnection(conn)

	user, err = s.GetUserByID(c, new.ID)
	if err != nil {
		fmt.Println("Error UpdateUser (GetUserByID) : ", err.Error())
		return user, err
	}
	err = nil

	user.UpdateUser(new)

	stmt, err := conn.PrepareNeo(`
	MATCH (n:USER)
	WHERE ID(n) = {ID}
	SET
		n.Email = {Email},
		n.Pseudo = {Pseudo},
		n.Password = {Password},
		n.Birthdate = {Birthdate},
		n.Firstname = {Firstname},
		n.Surname = {Surname},
		n.Number = {Number},
		n.Image = {Image},
		n.SuccessPoints = {SuccessPoints},
		n.StripeID = {StripeID}
	RETURN n`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdateUser (PrepareNeo) : " + err.Error())
		return user, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID":       user.ID,
		"Email":    user.Email,
		"Pseudo":   user.Pseudo,
		"Password": user.Password,
		// "Password":   encryptPassword(u.Password),
		"Birthdate":     user.Birthdate.Format(timeForm),
		"Firstname":     user.Firstname,
		"Surname":       user.Surname,
		"Number":        user.Number,
		"Image":         user.Image,
		"SuccessPoints": user.SuccessPoints,
		"StripeID":      user.StripeID,
	})

	if err != nil {
		fmt.Println("UpdateUser (QueryNeo) : " + err.Error())
		return user, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UpdateUser (NextNeo) : " + err.Error())
		return user, err
	}

	(&user).NodeToUser(data[0].(graph.Node))
	return user, err
}

/*************** Endpoint ***************/
type updateUserRequest struct {
	User User `json:"user"`
}

type updateUserResponse struct {
	User User   `json:"user"`
	Err  string `json:"err,omitempty"`
}

func UpdateUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserRequest)
		user, err := svc.UpdateUser(ctx, req.User)
		if err != nil {
			fmt.Println("Error UpdateUserEndpoint : ", err.Error())
			return updateUserResponse{user, err.Error()}, nil
		}
		return updateUserResponse{user, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPUpdateUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdateUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/update_user").Handler(httptransport.NewServer(
		endpoints.UpdateUserEndpoint,
		DecodeHTTPUpdateUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateUser", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateUser(ctx context.Context, u User) (User, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updateUser",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdateUser(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateUser(ctx context.Context, u User) (User, error) {
	v, err := mw.next.UpdateUser(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUpdateUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateUser")
		csLogger := log.With(logger, "method", "UpdateUser")

		csEndpoint = UpdateUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdateUser(ctx context.Context, u User) (User, error) {
	request := updateUserRequest{User: u}
	response, err := e.UpdateUserEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UpdateUser : ", err.Error())
		return response.(updateUserResponse).User, err
	}
	return response.(updateUserResponse).User, str2err(response.(updateUserResponse).Err)
}

func ClientUpdateUser(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/users/update_user"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUpdateUserResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdateUser")(ceEndpoint)
	return ceEndpoint, nil
}
