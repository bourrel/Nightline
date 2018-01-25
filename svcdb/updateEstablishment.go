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
func (s Service) UpdateEstablishment(c context.Context, new Establishment) (Establishment, error) {
	var estab Establishment

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdateEstablishment (WaitConnection) : " + err.Error())
		return estab, err
	}
	defer CloseConnection(conn)

	estab, err = s.GetEstablishment(c, new.ID)
	if err != nil {
		return estab, err
	}
	err = nil

	estab.UpdateEstablishment(new)

	stmt, err := conn.PrepareNeo(`
	MATCH (n:ESTABLISHMENT)
	WHERE ID(n) = {ID}
	SET
		n.Name = {Name},
		n.Long = {Long},
		n.Lat = {Lat},
		n.Address = {Address},
		n.Description = {Description},
		n.Image = {Image},
		n.OpenHours = {OpenHours}
	RETURN n`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdateEstablishment (PrepareNeo) : " + err.Error())
		return estab, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID":          estab.ID,
		"Name":        estab.Name,
		"Long":        estab.Long,
		"Lat":         estab.Lat,
		"Address":     estab.Address,
		"Description": estab.Description,
		"Image":       estab.Image,
		"OpenHours":   estab.OpenHours,
	})

	if err != nil {
		fmt.Println("UpdateEstablishment (QueryNeo) : " + err.Error())
		return estab, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UpdateEstablishment (NextNeo) : " + err.Error())
		return estab, err
	}

	(&estab).NodeToEstablishment(data[0].(graph.Node))
	return estab, err
}

/*************** Endpoint ***************/
type updateEstablishmentRequest struct {
	Establishment Establishment `json:"estab"`
}

type updateEstablishmentResponse struct {
	Establishment Establishment `json:"estab"`
	Err           string        `json:"err,omitempty"`
}

func UpdateEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateEstablishmentRequest)
		estab, err := svc.UpdateEstablishment(ctx, req.Establishment)
		if err != nil {
			fmt.Println("UpdateEstablishmentEndpoint: " + err.Error())
			return updateEstablishmentResponse{Establishment: estab, Err: err.Error()}, nil
		}
		return updateEstablishmentResponse{Establishment: estab, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updateEstablishmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("DecodeHTTPUpdateEstablishmentRequest: " + err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPUpdateEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateEstablishmentResponse

	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("DecodeHTTPUpdateEstablishmentResponse: " + err.Error())
		return nil, err
	}

	return response, nil
}

func UpdateEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/estabs/update_estab").Handler(httptransport.NewServer(
		endpoints.UpdateEstablishmentEndpoint,
		DecodeHTTPUpdateEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateEstablishment", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateEstablishment(ctx context.Context, u Establishment) (Establishment, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updateEstablishment",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdateEstablishment(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateEstablishment(ctx context.Context, u Establishment) (Establishment, error) {
	v, err := mw.next.UpdateEstablishment(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUpdateEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateEstablishment")
		csLogger := log.With(logger, "method", "UpdateEstablishment")

		csEndpoint = UpdateEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdateEstablishment(ctx context.Context, et Establishment) (Establishment, error) {
	request := updateEstablishmentRequest{Establishment: et}
	response, err := e.UpdateEstablishmentEndpoint(ctx, request)
	if err != nil {
		fmt.Println("UpdateEstablishment: " + err.Error())
		return et, err
	}
	estab := response.(updateEstablishmentResponse).Establishment
	return estab, str2err(response.(updateEstablishmentResponse).Err)
}

func ClientUpdateEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/estabs/update_estab"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUpdateEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdateEstablishment")(ceEndpoint)
	return ceEndpoint, nil
}
