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
func (s Service) UpdatePro(c context.Context, new Pro) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdatePro (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	pro, err = s.GetProByID(c, new.ID)
	if err != nil {
		return pro, err
	}
	err = nil

	pro.UpdatePro(new)

	stmt, err := conn.PrepareNeo(`
	MATCH (n:PRO)
	WHERE ID(n) = {ID}
	SET
		n.Email = {Email},
		n.Pseudo = {Pseudo},
		n.Password = {Password},
		n.Firstname = {Firstname},
		n.Surname = {Surname},
		n.Number = {Number}
	RETURN n`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdatePro (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID":       pro.ID,
		"Email":    pro.Email,
		"Pseudo":   pro.Pseudo,
		"Password": pro.Password,
		// "Password":   encryptPassword(u.Password),
		"Firstname": pro.Firstname,
		"Surname":   pro.Surname,
		"Number":    pro.Number,
	})

	if err != nil {
		fmt.Println("UpdatePro (QueryNeo) : " + err.Error())
		return pro, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UpdatePro (NextNeo) : " + err.Error())
		return pro, err
	}

	(&pro).NodeToPro(data[0].(graph.Node))
	return pro, err
}

/*************** Endpoint ***************/
type updateProRequest struct {
	Pro Pro `json:"pro"`
}

type updateProResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func UpdateProEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateProRequest)
		pro, err := svc.UpdatePro(ctx, req.Pro)
		if err != nil {
			return updateProResponse{pro, err.Error()}, nil
		}
		return updateProResponse{pro, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updateProRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPUpdateProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateProResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func UpdateProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/pros/update_pro").Handler(httptransport.NewServer(
		endpoints.UpdateProEndpoint,
		DecodeHTTPUpdateProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdatePro", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdatePro(ctx context.Context, u Pro) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updatePro",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdatePro(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdatePro(ctx context.Context, u Pro) (Pro, error) {
	v, err := mw.next.UpdatePro(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUpdateProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdatePro")
		csLogger := log.With(logger, "method", "UpdatePro")

		csEndpoint = UpdateProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdatePro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdatePro(ctx context.Context, et Pro) (Pro, error) {
	request := updateProRequest{Pro: et}
	response, err := e.UpdateProEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	return response.(updateProResponse).Pro, str2err(response.(updateProResponse).Err)
}

func ClientUpdatePro(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/pros/update_pro"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUpdateProResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdatePro")(ceEndpoint)
	return ceEndpoint, nil
}
