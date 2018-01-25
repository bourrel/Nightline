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
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) DeleteEstab(_ context.Context, groupID int64) error {
	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("DeleteEstab (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (e:ESTABLISHMENT)
		WHERE ID(e) = {id}
		DETACH DELETE e
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("DeleteEstab (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": groupID,
	})

	if err != nil {
		fmt.Println("DeleteEstab (QueryNeo) : " + err.Error())
		return err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("DeleteEstab (NextNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type deleteEstabRequest struct {
	EstabID int64 `json:"id"`
}

type deleteEstabResponse struct {
	Err string `json:"err,omitempty"`
}

func DeleteEstabEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteEstabRequest)
		err := svc.DeleteEstab(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error DeleteEstabEndpoint 1 : ", err.Error())
			return deleteEstabResponse{err.Error()}, nil
		}

		return deleteEstabResponse{""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteEstabRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteEstabRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPDeleteEstabResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response deleteEstabResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func DeleteEstabHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("DELETE").Path("/estab/{id}").Handler(httptransport.NewServer(
		endpoints.DeleteEstabEndpoint,
		DecodeHTTPDeleteEstabRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteEstab", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteEstab(ctx context.Context, groupID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "deleteEstab",
			"groupID", groupID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.DeleteEstab(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteEstab(ctx context.Context, groupID int64) error {
	err := mw.next.DeleteEstab(ctx, groupID)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildDeleteEstabEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteEstab")
		csLogger := log.With(logger, "method", "DeleteEstab")

		csEndpoint = DeleteEstabEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteEstab")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) DeleteEstab(ctx context.Context, etID int64) error {
	request := deleteEstabRequest{EstabID: etID}
	response, err := e.DeleteEstabEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return str2err(response.(deleteEstabResponse).Err)
}

func ClientDeleteEstab(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"DELETE",
		copyURL(u, "/estab/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPDeleteEstabResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "DeleteEstab")(ceEndpoint)
	return ceEndpoint, nil
}
