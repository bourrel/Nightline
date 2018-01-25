package svcapi

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
	}

	/* Authentication */
	LoginHTTPHandler(endpoints, tracer, logger, r, options)
	RegisterHTTPHandler(endpoints, tracer, logger, r, options)

	/* User */
	SearchUsersHTTPHandler(endpoints, tracer, logger, r, options)
	SearchFriendsHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserPreferencesHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserSuccessHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateUserHTTPHandler(endpoints, tracer, logger, r, options)
	UpdatePreferencesHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateStripeUserHTTPHandler(endpoints, tracer, logger, r, options)
	GetRecommendationHTTPHandler(endpoints, tracer, logger, r, options)

	/* Friends */
	GetUserFriendsHTTPHandler(endpoints, tracer, logger, r, options)
	InviteFriendHTTPHandler(endpoints, tracer, logger, r, options)
	GetUsersInvitationsHTTPHandler(endpoints, tracer, logger, r, options)
	InvitationAcceptHTTPHandler(endpoints, tracer, logger, r, options)
	InvitationDeclineHTTPHandler(endpoints, tracer, logger, r, options)

	/* Establishment */
	SearchEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)
	GetAllEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetMenuFromEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetSoireeFromEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetSoireesFromEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentTypesHTTPHandler(endpoints, tracer, logger, r, options)
	RateEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)

	/* Soiree */
	SoireeJoinHTTPHandler(endpoints, tracer, logger, r, options)
	SoireeOrderHTTPHandler(endpoints, tracer, logger, r, options)
	SoireeLeaveHTTPHandler(endpoints, tracer, logger, r, options)
	GetMenuFromSoireeHTTPHandler(endpoints, tracer, logger, r, options)

	/* Group */
	GetGroupHTTPHandler(endpoints, tracer, logger, r, options)
	CreateGroupHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateGroupHTTPHandler(endpoints, tracer, logger, r, options)
	GroupInviteHTTPHandler(endpoints, tracer, logger, r, options)
	GroupInvitationAcceptHTTPHandler(endpoints, tracer, logger, r, options)
	GroupInvitationDeclineHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserGroupsHTTPHandler(endpoints, tracer, logger, r, options)
	DeleteGroupHTTPHandler(endpoints, tracer, logger, r, options)
	DeleteGroupMemberHTTPHandler(endpoints, tracer, logger, r, options)
	GetGroupsInvitationsHTTPHandler(endpoints, tracer, logger, r, options)

	/* Notification */
	CreateNotificationHTTPHandler(endpoints, tracer, logger, r, options)

	/* Order */
	GetOrderHTTPHandler(endpoints, tracer, logger, r, options)
	CreateOrderHTTPHandler(endpoints, tracer, logger, r, options)
	AnswerOrderHTTPHandler(endpoints, tracer, logger, r, options)
	SearchOrdersHTTPHandler(endpoints, tracer, logger, r, options)
	
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
