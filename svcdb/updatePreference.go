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
func (s Service) UpdatePreference(c context.Context, userID int64, preferences []string) ([]string, error) {
	IPrefs := StrArrayToIArray(preferences)

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdatePreference (WaitConnection) : " + err.Error())
		return preferences, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (u:USER), (t:ESTABLISHMENT_TYPE)
	WHERE
		ID (u) = {ID} AND
		t.Name IN {preferences}
	CREATE (u)-[:PREFER]->(t)
	RETURN t, u
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdatePreference (PrepareNeo) : " + err.Error())
		return preferences, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID":          userID,
		"preferences": IPrefs,
	})

	if err != nil {
		fmt.Println("UpdatePreference (QueryNeo) : " + err.Error())
		return preferences, err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("UpdatePreference (NextNeo) : " + err.Error())
		return preferences, nil
	}

	return preferences, nil
}

/*************** Service ***************/
func DeletePreferences(c context.Context, userID int64) error {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("DeletePreferences (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (u:USER)-[p:PREFER]->(t:ESTABLISHMENT_TYPE)
	WHERE ID(u) = {ID}
	DETACH DELETE p
	RETURN u
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("DeletePreferences (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID": userID,
	})

	if err != nil {
		fmt.Println("DeletePreferences (QueryNeo) : " + err.Error())
		return err
	}

	// No return != error
	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("DeletePreferences (NextNeo) : " + err.Error())
		return nil
	}
	return nil
}

/*************** Endpoint ***************/
type updatePreferenceRequest struct {
	UserID      int64        `json:"userID"`
	Preferences []Preference `json:"preferences"`
}

type updatePreferenceResponse struct {
	Preferences []string `json:"preferences"`
	Err         string   `json:"err,omitempty"`
}

func UpdatePreferenceEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var prefs []string

		req := request.(updatePreferenceRequest)
		for i := 0; i < len(req.Preferences); i++ {
			prefs = append(prefs, req.Preferences[i].Name)
		}

		err := DeletePreferences(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error UpdatePreferenceEndpoint 1 : ", err.Error())
			return updatePreferenceResponse{prefs, err.Error()}, nil
		}

		pref, err := svc.UpdatePreference(ctx, req.UserID, prefs)
		if err != nil {
			fmt.Println("Error UpdatePreferenceEndpoint 2 : ", err.Error())
			return updatePreferenceResponse{pref, err.Error()}, nil
		}
		return updatePreferenceResponse{pref, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdatePreferenceRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updatePreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferenceRequest : ", err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPUpdatePreferenceResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updatePreferenceResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferenceResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdatePreferenceHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/{UserId:[0-9]+}/preferences").Handler(httptransport.NewServer(
		endpoints.UpdatePreferenceEndpoint,
		DecodeHTTPUpdatePreferenceRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdatePreference", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdatePreference(ctx context.Context, userID int64, Preferences []string) ([]string, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updatePreference",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdatePreference(ctx, userID, Preferences)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdatePreference(ctx context.Context, userID int64, Preferences []string) ([]string, error) {
	v, err := mw.next.UpdatePreference(ctx, userID, Preferences)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUpdatePreferenceEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdatePreference")
		csLogger := log.With(logger, "method", "UpdatePreference")

		csEndpoint = UpdatePreferenceEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdatePreference")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdatePreference(ctx context.Context, userID int64, Preferences []string) ([]string, error) {
	var prefs []Preference

	// req := request.(updatePreferenceRequest)
	for i := 0; i < len(Preferences); i++ {
		var tmpPref Preference

		tmpPref.Name = Preferences[i]
		prefs = append(prefs, tmpPref)
	}

	request := updatePreferenceRequest{userID, prefs}
	response, err := e.UpdatePreferenceEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UpdatePreference : ", err.Error())
		return response.(updatePreferenceResponse).Preferences, err
	}
	return response.(updatePreferenceResponse).Preferences, str2err(response.(updatePreferenceResponse).Err)
}

func EncodeHTTPUpdatePreferenceRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	uid := strconv.FormatInt(request.(updatePreferenceRequest).UserID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("UserId", uid)
	if err != nil {
		fmt.Println("Error EncodeHTTPUpdatePreferenceRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return EncodeHTTPGenericRequest(ctx, r, request)
}

func ClientUpdatePreference(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/users/{UserId:[0-9]+}/preferences"),
		EncodeHTTPUpdatePreferenceRequest,
		DecodeHTTPUpdatePreferenceResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdatePreference")(ceEndpoint)
	return ceEndpoint, nil
}
