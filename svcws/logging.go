package svcws

import (
	"io"
	"os"

	"github.com/go-kit/kit/log"
	slackcat "github.com/whosonfirst/go-writer-slackcat"
)

func DefineSlackWriter() (io.Writer, error) {
	var config = "/etc/slackcat-ws.conf"

	slack, err := slackcat.NewWriter(config)
	if err != nil {
		return os.Stdout, err
	}
	writer := io.MultiWriter(os.Stdout, slack)

	return writer, err
}

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
