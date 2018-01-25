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
func (s Service) DeleteSoiree(_ context.Context, soireeID int64) error {
	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("DeleteSoiree (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (s:SOIREE)
		WHERE ID(s) = {id}
		DETACH DELETE s
		RETURN s
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("DeleteSoiree (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})

	if err != nil {
		fmt.Println("DeleteSoiree (QueryNeo) : " + err.Error())
		return err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("DeleteSoiree (NextNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type deleteSoireeRequest struct {
	SoireeID int64 `json:"id"`
}

type deleteSoireeResponse struct {
	Err string `json:"err,omitempty"`
}

func DeleteSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteSoireeRequest)
		err := svc.DeleteSoiree(ctx, req.SoireeID)
		if err != nil {
			fmt.Println("Error DeleteSoireeEndpoint 1 : ", err.Error())
			return deleteSoireeResponse{err.Error()}, nil
		}

		return deleteSoireeResponse{""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteSoireeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPDeleteSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response deleteSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func DeleteSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("DELETE").Path("/soiree/{id}").Handler(httptransport.NewServer(
		endpoints.DeleteSoireeEndpoint,
		DecodeHTTPDeleteSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteSoiree(ctx context.Context, soireeID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "deleteSoiree",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.DeleteSoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteSoiree(ctx context.Context, soireeID int64) error {
	err := mw.next.DeleteSoiree(ctx, soireeID)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildDeleteSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteSoiree")
		csLogger := log.With(logger, "method", "DeleteSoiree")

		csEndpoint = DeleteSoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteSoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) DeleteSoiree(ctx context.Context, etID int64) error {
	request := deleteSoireeRequest{SoireeID: etID}
	response, err := e.DeleteSoireeEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return str2err(response.(deleteSoireeResponse).Err)
}

func ClientDeleteSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"DELETE",
		copyURL(u, "/soiree/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPDeleteSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "DeleteSoiree")(ceEndpoint)
	return ceEndpoint, nil
}
