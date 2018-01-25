package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
func (s Service) SoireeOrder(ctx context.Context, UserID, ConsoID, SoireeID int64, Token string) (int64, error) {

	//TODO: vérifier token
	//TODO: vérifier qu'il y ai bien un lien entre l'user et la soiree
	//TODO: vérifier qu'il y ai bien un lien entre la soiree et la conso

	user, err := s.svcdb.GetUserByID(ctx, UserID)
	soiree, err := s.svcdb.GetSoireeByID(ctx, SoireeID)
	conso, err := s.svcdb.GetConsoByID(ctx, ConsoID)

	orderID, err := s.svcdb.UserOrder(ctx, user, soiree, conso)
	if err != nil {
		return 0, dbToHTTPErr(err)
	}

	return orderID, nil
}

/*************** Endpoint ***************/
type SoireeOrderRequest struct {
	// SoireeID int64  `json:"soireeID"`
	// ConsoID  int64  `json:"consoID"`
	// UserID   int64  `json:"userID"`
	// Token    string `json:"token"`

	SoireeID int64          `json:"soireeID"`
	Token    string         `json:"token"`
	Sb       ShoppingBasket `json:"shopping_basket"`
}

type SoireeOrderResponse struct {
	OrderID int64 `json:"orderID"`
}

func SoireeOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// req := request.(SoireeOrderRequest)
		// orderID, err := svc.SoireeOrder(ctx, req.UserID, req.ConsoID, req.SoireeID, req.Token)

		// if err != nil {
		// 	fmt.Println("Error SoireeOrderEndpoint : ", err.Error())
		// 	return nil, err
		// }

		return SoireeOrderResponse{OrderID: -1}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSoireeOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SoireeOrderRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPSoireeOrderRequest 1 : ", err.Error())
		return req, RequestError
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPSoireeOrderRequest 2 : ", err.Error())
		return nil, RequestError
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPSoireeOrderRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&req).SoireeID = soireeID

	return req, nil
}

func DecodeHTTPSoireeOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SoireeOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func SoireeOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soiree/{SoireeId:[0-9]+}/order").Handler(httptransport.NewServer(
		endpoints.SoireeOrderEndpoint,
		DecodeHTTPSoireeOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SoireeOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SoireeOrder(ctx context.Context, UserID, ConsoID, SoireeID int64, Token string) (int64, error) {
	OrderID, err := mw.next.SoireeOrder(ctx, UserID, ConsoID, SoireeID, Token)

	mw.logger.Log(
		"method", "SoireeOrder",
		"request", SoireeOrderRequest{SoireeID: SoireeID, Token: Token},
		"response", OrderID,
		"took", time.Since(time.Now()),
	)
	return OrderID, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SoireeOrder(ctx context.Context, UserID, ConsoID, SoireeID int64, Token string) (int64, error) {
	return mw.next.SoireeOrder(ctx, UserID, ConsoID, SoireeID, Token)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SoireeOrder(ctx context.Context, UserID, ConsoID, SoireeID int64, Token string) (int64, error) {
	return mw.next.SoireeOrder(ctx, UserID, ConsoID, SoireeID, Token)
}

/*************** Main ***************/
/* Main */
func BuildSoireeOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SoireeOrder")
		csLogger := log.With(logger, "method", "SoireeOrder")

		csEndpoint = SoireeOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SoireeOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
