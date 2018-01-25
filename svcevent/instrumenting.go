package svcevent

import (
	"github.com/go-kit/kit/metrics"
)

/* MW Instrumenting interface */
func ServiceInstrumentingMiddleware(
	createSoiree_all metrics.Counter,
) Middleware {
	return func(next IService) IService {
		return serviceInstrumentingMiddleware{
			createSoiree_all: createSoiree_all,
			next:             next,
		}
	}
}

type serviceInstrumentingMiddleware struct {
	createSoiree_all metrics.Counter
	next             IService
}
