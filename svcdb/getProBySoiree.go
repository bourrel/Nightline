package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"strconv"
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
func (s Service) GetProBySoiree(ctx context.Context, soireeID int64) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetProBySoiree (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:PRO)-[:OWN]->(:ESTABLISHMENT)-[:SPAWNED]->(s:SOIREE) WHERE ID(s) = {id}
		RETURN ID(u)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetProBySoiree (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})
	if err != nil {
		fmt.Println("GetProBySoiree (QueryNeo) : " + err.Error())
		return pro, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetProBySoiree (NextNeo) : " + err.Error())
		return pro, err
	}

	proID := row[0].(int64)
	pro, err = s.GetProByID(ctx, proID)
	if err != nil {
		fmt.Println("GetProBySoiree (GetProByID) : " + err.Error())
		return pro, err
	}

	return pro, nil
}

/*************** Endpoint ***************/
type getProBySoireeRequest struct {
	ID int64 `json:"id"`
}

type getProBySoireeResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func GetProBySoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getProBySoireeRequest)
		pro, err := svc.GetProBySoiree(ctx, req.ID)
		if err != nil {
			return getProBySoireeResponse{pro, err.Error()}, nil
		}
		return getProBySoireeResponse{pro, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProBySoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getProBySoireeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = soireeID

	return request, nil
}

func DecodeHTTPGetProBySoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getProBySoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetProBySoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/pros/get_pro/soiree/{id}").Handler(httptransport.NewServer(
		endpoints.GetProBySoireeEndpoint,
		DecodeHTTPGetProBySoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetProBySoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetProBySoiree(ctx context.Context, soireeID int64) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getProBySoiree",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetProBySoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetProBySoiree(ctx context.Context, soireeID int64) (Pro, error) {
	v, err := mw.next.GetProBySoiree(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetProBySoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetProBySoiree")
		csLogger := log.With(logger, "method", "GetProBySoiree")

		csEndpoint = GetProBySoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetProBySoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetProBySoiree(ctx context.Context, soireeID int64) (Pro, error) {
	var pro Pro

	request := getProBySoireeRequest{ID: soireeID}
	response, err := e.GetProBySoireeEndpoint(ctx, request)
	if err != nil {
		return pro, err
	}
	pro = response.(getProBySoireeResponse).Pro
	return pro, str2err(response.(getProBySoireeResponse).Err)
}

func EncodeHTTPGetProBySoireeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getProBySoireeRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("id", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetProBySoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/pros/get_pro/soiree/{id}"),
		EncodeHTTPGetProBySoireeRequest,
		DecodeHTTPGetProBySoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetProBySoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
