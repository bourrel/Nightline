package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/card"
	"github.com/stripe/stripe-go/customer"

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
func (s Service) UpdateStripeUser(ctx context.Context, user svcdb.User, token string) (svcdb.User, error) {
	var sc *stripe.Customer
	stripe.Key = "sk_test_gs5myv9fGkIMYd0EcQUAnxEf"

	user, err := s.svcdb.GetUserByID(ctx, user.ID)
	if err != nil {
		return user, dbToHTTPErr(err)
	}

	if user.StripeID != "" && user.StripeID != "null" {
		/* Remove pre-existing stripe source */
		sc, err = customer.Get(user.StripeID, nil)
		if err != nil {
			return user, err
		}
		params := &stripe.CardListParams{Customer: (*sc).ID}
		params.Filters.AddFilter("limit", "", "10")
		i := card.List(params)
		for i.Next() {
			c := i.Card()
			card.Del(c.ID, &stripe.CardParams{Customer: (*sc).ID})
		}

	} else {
		/* Create Stripe Customer */
		params := &stripe.CustomerParams{
			Desc:  fmt.Sprintf("%d", user.ID),
			Email: user.Email,
		}
		sc, err = customer.New(params)
		if err != nil {
			return user, err
		}

		/* Update User.StripeID in db */
		user.StripeID = (*sc).ID
		s.svcdb.UpdateUser(ctx, user)
	}

	/* Create & link new source */
	_, err = card.New(&stripe.CardParams{
		Customer: (*sc).ID,
		Token:    token,
	})
	if err != nil {
		return user, dbToHTTPErr(err)
	}

	return user, nil
}

/*************** Endpoint ***************/
type updateStripeUserRequest struct {
	User  svcdb.User `json:"user"`
	Token string     `json:"token"`
}

type updateStripeUserResponse struct {
	User svcdb.User `json:"user"`
}

func UpdateStripeUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateStripeUserRequest)
		user, err := svc.UpdateStripeUser(ctx, req.User, req.Token)
		return updateStripeUserResponse{User: user}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateStripeUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateStripeUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return req, RequestError
	}
	return req, nil
}

func DecodeHTTPupdateStripeUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateStripeUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func UpdateStripeUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("PATCH").Path("/update_stripe_user").Handler(httptransport.NewServer(
		endpoints.UpdateStripeUserEndpoint,
		DecodeHTTPUpdateStripeUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateStripeUser", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateStripeUser(ctx context.Context, user svcdb.User, token string) (svcdb.User, error) {
	newUser, err := mw.next.UpdateStripeUser(ctx, user, token)

	mw.logger.Log(
		"method", "UpdateStripeUser",
		"request", user,
		"request", updateStripeUserResponse{User: newUser},
		"took", time.Since(time.Now()),
	)
	return newUser, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdateStripeUser(ctx context.Context, user svcdb.User, token string) (svcdb.User, error) {
	return mw.next.UpdateStripeUser(ctx, user, token)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateStripeUser(ctx context.Context, user svcdb.User, token string) (svcdb.User, error) {
	return mw.next.UpdateStripeUser(ctx, user, token)
}

/*************** Main ***************/
/* Main */
func BuildUpdateStripeUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateStripeUser")
		csLogger := log.With(logger, "method", "UpdateStripeUser")

		csEndpoint = UpdateStripeUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateStripeUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
