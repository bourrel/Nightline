package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"time"

	"golang.org/x/crypto/bcrypt"

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
func (s Service) CreatePro(c context.Context, u Pro) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreatePro (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	_, err = s.GetPro(c, u)
	if err == nil {
		err = errors.New("Pro already exists")
		return pro, err
	}
	err = nil

	_, err = encryptPassword(u.Password)
	if err != nil {
		fmt.Println("CreatePro (encryptPassword) : " + err.Error())
		return pro, err
	}

	stmt, err := conn.PrepareNeo(`CREATE (u:PRO {
		Email: {Email},
		Pseudo: {Pseudo},
		Password: {Password},
		Firstname: {Firstname},
		Surname: {Surname},
		Image: {Image},
		Number: {Number},
		StripeID: {StripeID},
		StripeSKey: {StripeSKey},
		StripePKey: {StripePKey}
	}) RETURN u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreatePro (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"Email":  u.Email,
		"Pseudo": u.Pseudo,
		// "Password":   encryptPassword(u.Password),
		"Password":   u.Password,
		"Firstname":  u.Firstname,
		"Surname":    u.Surname,
		"Number":     u.Number,
		"Image":      u.Image,
		"StripeID": u.StripeID,
		"StripeSKey": u.StripeSKey,
		"StripePKey": u.StripePKey,
	})

	if err != nil {
		fmt.Println("CreatePro (QueryNeo) : " + err.Error())
		return pro, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreatePro (NextNeo) : " + err.Error())
		return pro, err
	}

	(&pro).NodeToPro(data[0].(graph.Node))
	return pro, err
}

func encryptPassword(proPassword string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(proPassword), bcrypt.DefaultCost)
	if err != nil {
		return *new([]byte), err // Dirty
	}
	return hash, nil
}

/*************** Endpoint ***************/
type createProRequest struct {
	Pro Pro `json:"pro"`
}

type createProResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func CreateProEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createProRequest)
		pro, err := svc.CreatePro(ctx, req.Pro)
		if err != nil {
			return createProResponse{pro, err.Error()}, nil
		}
		return createProResponse{pro, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createProRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createProResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/pros/create_pro").Handler(httptransport.NewServer(
		endpoints.CreateProEndpoint,
		DecodeHTTPCreateProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreatePro", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreatePro(ctx context.Context, u Pro) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createPro",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreatePro(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreatePro(ctx context.Context, u Pro) (Pro, error) {
	v, err := mw.next.CreatePro(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreatePro")
		csLogger := log.With(logger, "method", "CreatePro")

		csEndpoint = CreateProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreatePro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreatePro(ctx context.Context, et Pro) (Pro, error) {
	request := createProRequest{Pro: et}
	response, err := e.CreateProEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	return response.(createProResponse).Pro, str2err(response.(createProResponse).Err)
}

func ClientCreatePro(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/pros/create_pro"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateProResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreatePro")(ceEndpoint)
	return ceEndpoint, nil
}
