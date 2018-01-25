package svcws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

var clientCount int = 0

/* HTTP handlers & routing */
func MakeHTTPHandler(tracer stdopentracing.Tracer, logger log.Logger, svc IService, c chan *sarama.ConsumerMessage) http.Handler {
	r := mux.NewRouter()
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
	}

	OpenConnectionHTTPHandler(tracer, logger, r, svc, options, c)

	return r
}

/* Errors *coders */
type errorWrapper struct {
	Error string `json:"error"`
}

func setDefaultHeaders(w http.ResponseWriter) http.ResponseWriter {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	return w
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	setDefaultHeaders(w)
	code := http.StatusInternalServerError
	msg := err.Error()

	switch err {
	case ConnError:
		code = http.StatusBadGateway
	case AuthError:
		code = http.StatusForbidden
	case RequestError:
		code = http.StatusBadRequest
	case NotFoundError:
		code = http.StatusNotFound
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
func EncodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

func EncodeHTTPGenericResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	setDefaultHeaders(w)
	return json.NewEncoder(w).Encode(response)
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}
