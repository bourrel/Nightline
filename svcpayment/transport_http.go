package svcpayment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

/* HTTP handlers & routing */
func MakeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer,
	logger log.Logger) http.Handler {
	r := mux.NewRouter()
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorLogger(logger),
	}

	/* Order */
	GetOrderHTTPHandler(endpoints, tracer, logger, r, options)
	CreateOrderHTTPHandler(endpoints, tracer, logger, r, options)
	PutOrderHTTPHandler(endpoints, tracer, logger, r, options)
	SearchOrdersHTTPHandler(endpoints, tracer, logger, r, options)
	AnswerOrderHTTPHandler(endpoints, tracer, logger, r, options)

	/* Pro */
	RegisterProHTTPHandler(endpoints, tracer, logger, r, options)
	
	return r
}

/* Errors *coders */
type errorWrapper struct {
	Error string `json:"error"`
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	msg := err.Error()

	switch err {
	case Err:
		code = http.StatusBadRequest
	}

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorWrapper{Error: msg})
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

/* Generic encoders */
func EncodeHTTPGenericRequest(ctx context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

func EncodeHTTPGenericResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	js := json.NewEncoder(w).Encode(response)
	return js
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}
