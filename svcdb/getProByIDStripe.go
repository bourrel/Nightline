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
func (s Service) GetProByIDStripe(_ context.Context, proID int64) (Pro, error) {
	var pro Pro

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetProByIDStripe (WaitConnection) : " + err.Error())
		return pro, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:PRO) WHERE ID(u) = {id}
		OPTIONAL MATCH (u)-[:OWN]-(e:ESTABLISHMENT)	
		RETURN u, ID(e)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetProByIDStripe (PrepareNeo) : " + err.Error())
		return pro, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": proID,
	})
	if err != nil {
		fmt.Println("GetProByIDStripe (QueryNeo) : " + err.Error())
		return pro, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetProByIDStripe (NextNeo) : " + err.Error())
		return pro, err
	}

	(&pro).NodeToProStripe(row[0].(graph.Node))
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetProByIDStripe (---)")
			panic(err)
		} else if err != io.EOF {
			if row[1] != nil {
				pro.Establishments = append(pro.Establishments, row[1].(int64))
			}
		}
		row, _, err = rows.NextNeo()
	}

	return pro, nil
}

/*************** Endpoint ***************/
type getProByIDStripeRequest struct {
	ID int64 `json:"id"`
}

type getProByIDStripeResponse struct {
	Pro Pro    `json:"pro"`
	Err string `json:"err,omitempty"`
}

func GetProByIDStripeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getProByIDStripeRequest)
		pro, err := svc.GetProByIDStripe(ctx, req.ID)
		if err != nil {
			return getProByIDStripeResponse{pro, err.Error()}, nil
		}
		return getProByIDStripeResponse{pro, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetProByIDStripeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getProByIDStripeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	proiD, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = proiD

	return request, nil
}

func DecodeHTTPGetProByIDStripeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getProByIDStripeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetProByIDStripeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/pros/get_pro/{id:[0-9]+}/stripe").Handler(httptransport.NewServer(
		endpoints.GetProByIDStripeEndpoint,
		DecodeHTTPGetProByIDStripeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetProByIDStripe", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetProByIDStripe(ctx context.Context, proID int64) (Pro, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getProByIDStripe",
			"proID", proID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetProByIDStripe(ctx, proID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetProByIDStripe(ctx context.Context, proID int64) (Pro, error) {
	v, err := mw.next.GetProByIDStripe(ctx, proID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetProByIDStripeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetProByIDStripe")
		csLogger := log.With(logger, "method", "GetProByIDStripe")

		csEndpoint = GetProByIDStripeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetProByIDStripe")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetProByIDStripe(ctx context.Context, ID int64) (Pro, error) {
	var pro Pro

	request := getProByIDStripeRequest{ID: ID}
	response, err := e.GetProByIDStripeEndpoint(ctx, request)
	if err != nil {
		return pro, err
	}
	pro = response.(getProByIDStripeResponse).Pro
	return pro, str2err(response.(getProByIDStripeResponse).Err)
}

func EncodeHTTPGetProByIDStripeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getProByIDStripeRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("ID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetProByIDStripe(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/pros/get_pro/{ID:[0-9]+}/stripe"),
		EncodeHTTPGetProByIDStripeRequest,
		DecodeHTTPGetProByIDStripeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetProByIDStripe")(gefmEndpoint)
	return gefmEndpoint, nil
}
