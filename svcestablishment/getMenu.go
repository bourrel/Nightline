package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)

/*************** Service ***************/
func (s Service) GetMenu(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	menus, err := s.svcdb.GetEstablishmentMenus(ctx, estabID)
	return menus, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getMenuRequest struct {
	EstabID int64 `json:"id"`
}

type getMenuResponse struct {
	Menus []svcdb.Menu `json:"menus"`
}

func GetMenuEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuRequest)
		menus, err := svc.GetMenu(ctx, req.EstabID)
		return getMenuResponse{Menus: menus}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuRequest

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetMenuResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetMenuHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabID}/menus").Handler(httptransport.NewServer(
		endpoints.GetMenuEndpoint,
		DecodeHTTPGetMenuRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenu", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenu(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	menu, err := mw.next.GetMenu(ctx, estabID)

	mw.logger.Log(
		"method", "getMenu",
		"estabID", estabID,
		"response", menu,
		"took", time.Since(time.Now()),
	)
	return menu, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetMenu(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	return mw.next.GetMenu(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenu(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	return mw.next.GetMenu(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetMenuEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetMenu")
		csLogger := log.With(logger, "method", "GetMenu")

		csEndpoint = GetMenuEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetMenu")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetMenu(ctx context.Context, etID int64) ([]svcdb.Menu, error) {
	var menu []svcdb.Menu

	request := getMenuRequest{EstabID: etID}
	response, err := e.GetMenuEndpoint(ctx, request)
	if err != nil {
		return menu, err
	}
	menu = response.(getMenuResponse).Menus
	return menu, err
}

func EncodeHTTPGetMenuRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getMenuRequest).EstabID)
	encodedUrl, err := route.Path(r.URL.Path).URL("EstabID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetMenu(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/establishments/{EstabID}/menus"),
		EncodeHTTPGetMenuRequest,
		DecodeHTTPGetMenuResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetMenu")(ceEndpoint)
	return ceEndpoint, nil
}
