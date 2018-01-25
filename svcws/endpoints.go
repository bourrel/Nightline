package svcws

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
)

/* Endpoints definition */
type Endpoints struct {
	OpenConnectionEndpoint endpoint.Endpoint
}

/* Logging Middleware */
func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				if err != nil {
					level.Error(logger).Log(
						"error", err,
						"took", time.Since(begin))
				}
			}(time.Now())
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
