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
func (s Service) GetSoireesByEstablishment(_ context.Context, estabID int64) ([]Soiree, error) {
	var soirees []Soiree

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetSoireesByEstablishment (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH (u:ESTABLISHMENT)-[SPAWNED]->(n:SOIREE) WHERE ID(u) = {id} RETURN n")
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetSoireesByEstablishment (PrepareNeo)")
		panic(err)
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})
	if err != nil {
		fmt.Println("GetSoireesByEstablishment (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpSoiree Soiree

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetSoireesByEstablishment (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpSoiree).NodeToSoiree(row[0].(graph.Node))

			soirees = append(soirees, tmpSoiree)
		}
		row, _, err = rows.NextNeo()
	}
	return soirees, nil
}

/*************** Endpoint ***************/
type getSoireesByEstablishmentRequest struct {
	EstabID int64 `json:"id"`
}

type getSoireesByEstablishmentResponse struct {
	Soirees []Soiree `json:"soirees"`
	Err     string   `json:"err,omitempty"`
}

func GetSoireesByEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSoireesByEstablishmentRequest)
		soirees, err := svc.GetSoireesByEstablishment(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetSoireesByEstablishmentEndpoint 1 : ", err.Error())
			return getSoireesByEstablishmentResponse{soirees, err.Error()}, nil
		}

		for i := 0; i < len(soirees); i++ {
			soirees[i].Menu, err = svc.GetMenuFromSoiree(ctx, soirees[i].ID)
			if err != nil {
				fmt.Println("Error GetSoireesByEstablishmentEndpoint 2 : ", err.Error())
				return getSoireesByEstablishmentResponse{soirees, err.Error()}, nil
			}

			soirees[i].Friends, err = svc.GetConnectedFriends(ctx, soirees[i].ID)
			if err != nil {
				fmt.Println("Error GetSoireesByEstablishmentEndpoint 3 : ", err.Error())
				return getSoireesByEstablishmentResponse{soirees, err.Error()}, nil
			}
		}

		return getSoireesByEstablishmentResponse{soirees, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireesByEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSoireesByEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesByEstablishmentRequest 1 : ", err.Error())
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesByEstablishmentRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetSoireesByEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSoireesByEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireesByEstablishmentResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetSoireesByEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").
		Path("/soireesByEstablishment/{EstabID}").
		Handler(httptransport.NewServer(
			endpoints.GetSoireesByEstablishmentEndpoint,
			DecodeHTTPGetSoireesByEstablishmentRequest,
			EncodeHTTPGenericResponse,
			append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireesByEstablishment", logger)))...,
		))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireesByEstablishment(ctx context.Context, estabID int64) ([]Soiree, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getSoireesByEstablishment",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetSoireesByEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireesByEstablishment(ctx context.Context, estabID int64) ([]Soiree, error) {
	v, err := mw.next.GetSoireesByEstablishment(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetSoireesByEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSoireesByEstablishment")
		csLogger := log.With(logger, "method", "GetSoireesByEstablishment")

		csEndpoint = GetSoireesByEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSoireesByEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSoireesByEstablishment(ctx context.Context, estabID int64) ([]Soiree, error) {
	var et []Soiree

	request := getSoireesByEstablishmentRequest{EstabID: estabID}
	response, err := e.GetSoireesByEstablishmentEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetSoireesByEstablishment : ", err.Error())
		return et, err
	}
	et = response.(getSoireesByEstablishmentResponse).Soirees
	return et, str2err(response.(getSoireesByEstablishmentResponse).Err)
}

func EncodeHTTPGetSoireesByEstablishmentRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getSoireesByEstablishmentRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetSoireesByEstablishmentRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSoireesByEstablishment(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/soireesByEstablishment/{EstabID}"),
		EncodeHTTPGetSoireesByEstablishmentRequest,
		DecodeHTTPGetSoireesByEstablishmentResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetSoireesByEstablishment")(ceEndpoint)
	return ceEndpoint, nil
}
