package svcdb

import (
	"github.com/go-kit/kit/metrics"
)

/* MW Instrumenting interface */
func ServiceInstrumentingMiddleware(ints metrics.Counter) Middleware {
	return func(next IService) IService {
		return serviceInstrumentingMiddleware{
			ints: ints,
			next: next,
		}
	}
}

type serviceInstrumentingMiddleware struct {
	ints metrics.Counter
	next IService
}
