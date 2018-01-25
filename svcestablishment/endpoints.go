package svcestablishment

import (
	"context"
	"fmt"
	"time"

	stdjwt "github.com/dgrijalva/jwt-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

/* Endpoints definition */
type Endpoints struct {
	/* Soiree */
	CreateSoireeEndpoint endpoint.Endpoint
	GetSoireesEndpoint   endpoint.Endpoint
	DeleteSoireeEndpoint endpoint.Endpoint

	/* Order */
	GetOrderEndpoint          endpoint.Endpoint
	DeliverOrderEndpoint      endpoint.Endpoint
	GetOrdersBySoireeEndpoint endpoint.Endpoint
	SearchOrdersEndpoint      endpoint.Endpoint
	PutOrderEndpoint          endpoint.Endpoint
	GetConsoEndpoint          endpoint.Endpoint
	GetSoireeOrdersEndpoint   endpoint.Endpoint
	GetConsoByOrderIDEndpoint endpoint.Endpoint

	/* Menu */
	GetMenuEndpoint     endpoint.Endpoint
	CreateMenuEndpoint  endpoint.Endpoint
	CreateConsoEndpoint endpoint.Endpoint

	/* Establishment */
	CreateEstabEndpoint          endpoint.Endpoint
	UpdateEstabEndpoint          endpoint.Endpoint
	DeleteEstabEndpoint          endpoint.Endpoint
	GetEstablishmentTypeEndpoint endpoint.Endpoint

	/* Pro */
	LoginProEndpoint             endpoint.Endpoint
	RegisterProEndpoint          endpoint.Endpoint
	UpdateProEndpoint            endpoint.Endpoint
	GetProEstablishmentsEndpoint endpoint.Endpoint

	/* Stat */
	GetStatEndpoint endpoint.Endpoint
	GetAnalysePEndpoint endpoint.Endpoint
}

var claims = &stdjwt.StandardClaims{
	ExpiresAt: time.Now().UTC().Add(time.Hour * 1).Unix(), // Add 1 Hour
	Issuer:    "NightLine",
}

/* Logging Middleware */
func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				if err != nil {
					logger.Log("error", err, "took", time.Since(begin))
				}
			}(time.Now())
			return next(ctx, request)
		}
	}
}

/* Authentication Middleware */
func EndpointAuthenticationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// defer func() {
			// 	kf := func(token *stdjwt.Token) (interface{}, error) { return verifKey, nil }
			// 	next = jwt.NewParser(kf, stdjwt.SigningMethodHS256, claims)(next)
			// }()

			return next(ctx, request)
		}
	}
}

/* Instrumenting Middleware */
func EndpointInstrumentingMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}
