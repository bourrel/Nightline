package svcdb

import (
	"context"
	"encoding/json"
	"errors"
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
func (s Service) GetEstablishmentSoiree(_ context.Context, estabID int64) (Soiree, error) {
	var soiree Soiree
	var soirees []Soiree
	timeNow := time.Now()

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetEstablishmentSoiree (WaitConnection) : " + err.Error())
		return soiree, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (e:ESTABLISHMENT)-[:SPAWNED]->(s:SOIREE) WHERE ID(e) = {id} RETURN (s)`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetEstablishmentSoiree (PrepareNeo) : " + err.Error())
		return soiree, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": estabID,
	})

	if err != nil {
		fmt.Println("GetEstablishmentSoiree (QueryNeo) : " + err.Error())
		return soiree, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetEstablishmentSoiree (NextNeo) : " + err.Error())
		return soiree, err
	}

	/* select best */
	tsub1min, _ := time.ParseDuration("-1h") // comparison placeholder
	soiree.End = timeNow.Add(tsub1min)
	var tmpSoiree Soiree
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetEstablishmentSoirees (---) : " + err.Error())
			panic(err)
		} else if err != io.EOF {
			(&tmpSoiree).NodeToSoiree(row[0].(graph.Node))
			soirees = append(soirees, tmpSoiree)
			if tmpSoiree.End.After(timeNow) &&
				(tmpSoiree.End.Before(soiree.End) ||
					soiree.End.Before(timeNow)) {
				soiree = tmpSoiree
			}
		}
		row, _, err = rows.NextNeo()
	}

	if soiree.End.Before(timeNow) {
		fmt.Println("GetEstablishmentSoiree final", DateErr)
		return soiree, errors.New("No party incoming")
	}
	return soiree, nil
}

/*************** Endpoint ***************/
type getEstablishmentSoireeRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmentSoireeResponse struct {
	Soiree Soiree `json:"soiree"`
	Err    string `json:"err,omitempty"`
}

func GetEstablishmentSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentSoireeRequest)
		soiree, err := svc.GetEstablishmentSoiree(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetEstablishmentSoireeEndpoint 1 : ", err.Error())
			return getEstablishmentSoireeResponse{soiree, err.Error()}, nil
		}

		soiree.Menu, err = svc.GetMenuFromSoiree(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error GetEstablishmentSoireeEndpoint 2 : ", err.Error())
			return getEstablishmentSoireeResponse{soiree, err.Error()}, nil
		}

		soiree.Friends, err = svc.GetConnectedFriends(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error GetEstablishmentSoireeEndpoint 3 : ", err.Error())
			return getEstablishmentSoireeResponse{soiree, err.Error()}, nil
		}

		return getEstablishmentSoireeResponse{soiree, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentSoireeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetEstablishmentSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishmentSoiree/{EstabID}").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentSoireeEndpoint,
		DecodeHTTPGetEstablishmentSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentSoiree(ctx context.Context, estabID int64) (Soiree, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getEstablishmentSoiree",
			"estabID", estabID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetEstablishmentSoiree(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentSoiree(ctx context.Context, estabID int64) (Soiree, error) {
	v, err := mw.next.GetEstablishmentSoiree(ctx, estabID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetEstablishmentSoiree")
		gefmLogger := log.With(logger, "method", "GetEstablishmentSoiree")

		gefmEndpoint = GetEstablishmentSoireeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentSoiree")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentSoiree(ctx context.Context, estabID int64) (Soiree, error) {
	var s Soiree

	request := getEstablishmentSoireeRequest{EstabID: estabID}
	response, err := e.GetEstablishmentSoireeEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getEstablishmentSoireeResponse).Soiree
	return s, str2err(response.(getEstablishmentSoireeResponse).Err)
}

func EncodeHTTPGetEstablishmentSoireeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getEstablishmentSoireeRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetEstablishmentSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishmentSoiree/{EstabID}"),
		EncodeHTTPGetEstablishmentSoireeRequest,
		DecodeHTTPGetEstablishmentSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentSoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
