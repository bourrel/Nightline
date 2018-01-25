package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"io"
	"strconv"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetAnalyseP(_ context.Context, estabID int64, soireeID int64) ([]AnalyseP, error) {
	var req string
	var analyses	[]AnalyseP
	// var orders Orders
	// mapConsoIdx := make(map[int64]int64)

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetAnalyseP (WaitConnection) : " + err.Error())
		return analyses, err
	}
	defer CloseConnection(conn)

	// req = `MATCH (e:ESTABLISHMENT)--(s:SOIREE)<-[:DURING]-(o:ORDER { Done: "true" })-[:DONE]->(st:STEP { Name: "Deliverpaid" }), (o)-[str:FOR]->(c:CONSO) WHERE ID(e) = {estabID}`
	req = `MATCH (e:ESTABLISHMENT)-->(s:SOIREE)<--(u:USER) WHERE ID(e) = {estabID}`
	if (soireeID > 0) {
		req += ` AND ID(s) = {soireeID}`
	}
	req += `return ID(u), COUNT(DISTINCT(s))`
	
	stmt, err := conn.PrepareNeo(req)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetAnalyseP (PrepareNeo) : " + err.Error())
		return analyses, err
	}
	
	rows, err := stmt.QueryNeo(map[string]interface{}{
		"estabID": estabID,
		"soireeID": soireeID,
	})
	if err != nil {
		fmt.Println("GetAnalyseP (QueryNeo) : " + err.Error())
		return analyses, err
	}

	var tmpAnalyse AnalyseP
	tmpAnalyse.Type = "Population"
	tmpAnalyse.Values = make(map[string]int64)
	tmpAnalyse.Values["new"] = 0
	tmpAnalyse.Values["initied"] = 0
	tmpAnalyse.Values["regular"] = 0
	tmpAnalyse.Values["habitual"] = 0
	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetAnalyseP (NextNeo) : " + err.Error())
			return analyses, err
		} else if err != io.EOF {
			count := row[1].(int64)
			if count > 10 {
				tmpAnalyse.Values["habitual"] += 1
			} else if count > 5 {
				tmpAnalyse.Values["regular"] += 1
			} else if count > 2 {
				tmpAnalyse.Values["initied"] += 1
			} else {
				tmpAnalyse.Values["new"] += 1
			}
		}
		row, _, err = rows.NextNeo()
	}
	analyses = append(analyses, tmpAnalyse)

	return analyses, nil
}

/*************** Endpoint ***************/
type getAnalysePRequest struct {
	EstabID       int64   `json:"estabID"`
	SoireeID       int64   `json:"soireeID"`
}

type getAnalysePResponse struct {
	Analyses     []AnalyseP `json:"analyses"`
	Err         string  `json:"err,omitempty"`
}

func GetAnalysePEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getAnalysePRequest)
		analyses, err := svc.GetAnalyseP(ctx, req.EstabID, req.SoireeID)
		if err != nil {
			fmt.Println("Error GetAnalysePEndpoint : ", err.Error())
			return getAnalysePResponse{analyses, err.Error()}, nil
		}
		return getAnalysePResponse{analyses, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetAnalysePRequest(_ context.Context, r *http.Request) (interface{}, error) {
    var request getAnalysePRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["estabID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).EstabID = estabID

	soireeID, err := strconv.ParseInt(mux.Vars(r)["soireeID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).SoireeID = soireeID

	return request, nil
	
}

func DecodeHTTPGetAnalysePResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getAnalysePResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetAnalysePResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetAnalysePHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r*mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/analyses/Population/{estabID:[0-9]+}/{soireeID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetAnalysePEndpoint,
		DecodeHTTPGetAnalysePRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetAnalyseP",logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]AnalyseP, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getAnalyseP",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetAnalyseP(ctx, estabID, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]AnalyseP, error) {
	v, err := mw.next.GetAnalyseP(ctx, estabID, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetAnalysePEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetAnalyseP")
		csLogger := log.With(logger, "method", "GetAnalyseP")

		csEndpoint = GetAnalysePEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetAnalyseP")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]AnalyseP, error) {
	var analysesP []AnalyseP

	request := getAnalysePRequest{EstabID: estabID, SoireeID: soireeID}
	response, err := e.GetAnalysePEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error Client GetAnalyseP : ", err.Error())
		return analysesP, err
	}
	analysesP = response.(getAnalysePResponse).Analyses
	return analysesP, str2err(response.(getAnalysePResponse).Err)
}

func EncodeHTTPGetAnalysePRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	estabID := fmt.Sprintf("%v", request.(getAnalysePRequest).EstabID)
	soireeID := fmt.Sprintf("%v", request.(getAnalysePRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("estabID", estabID, "soireeID", soireeID)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetAnalyseP(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/analyses/Population/{estabID:[0-9]+}/{soireeID:[0-9]+}"),
		EncodeHTTPGetAnalysePRequest,
		DecodeHTTPGetAnalysePResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetAnalyseP")(ceEndpoint)
	return ceEndpoint, nil
}
