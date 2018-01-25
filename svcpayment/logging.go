package svcpayment

import (
	"github.com/go-kit/kit/log"
)

/* MW Logging interface */
func ServiceLoggingMiddleware(logger log.Logger) Middleware {
	return func(next IService) IService {
		return serviceLoggingMiddleware{
			logger: logger,
			next:   next,
		}
	}
}

type serviceLoggingMiddleware struct {
	logger log.Logger
	next   IService
}
