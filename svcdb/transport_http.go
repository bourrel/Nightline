package svcdb

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

	/* Users */
	CreateUserHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateUserHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserByIDHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserProfileHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserSuccessHTTPHandler(endpoints, tracer, logger, r, options)
	GetUserPreferencesHTTPHandler(endpoints, tracer, logger, r, options)
	UpdatePreferenceHTTPHandler(endpoints, tracer, logger, r, options)
	SearchUsersHTTPHandler(endpoints, tracer, logger, r, options)
	GetRecommendationHTTPHandler(endpoints, tracer, logger, r, options)

	/* Friends */
	GetUserFriendsHTTPHandler(endpoints, tracer, logger, r, options)
	GetUsersInvitationsHTTPHandler(endpoints, tracer, logger, r, options)
	InviteFriendHTTPHandler(endpoints, tracer, logger, r, options)
	InvitationAcceptHTTPHandler(endpoints, tracer, logger, r, options)
	InvitationDeclineHTTPHandler(endpoints, tracer, logger, r, options)
	UsersConnectedHTTPHandler(endpoints, tracer, logger, r, options)
	SearchFriendsHTTPHandler(endpoints, tracer, logger, r, options)

	/* Pro */
	CreateProHTTPHandler(endpoints, tracer, logger, r, options)
	GetProHTTPHandler(endpoints, tracer, logger, r, options)
	GetProByIDStripeHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateProHTTPHandler(endpoints, tracer, logger, r, options)
	GetProByIDHTTPHandler(endpoints, tracer, logger, r, options)
	GetProBySoireeHTTPHandler(endpoints, tracer, logger, r, options)
	GetProEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)

	/* Establishments */
	CreateEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)
	SearchEstablishmentsHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentFromMenuHTTPHandler(endpoints, tracer, logger, r, options)
	GetMenuFromEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentSoireeHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentTypesHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentTypeHTTPHandler(endpoints, tracer, logger, r, options)
	RateEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	DeleteEstabHTTPHandler(endpoints, tracer, logger, r, options)

	/* Soiree */
	GetSoireesByEstablishmentHTTPHandler(endpoints, tracer, logger, r, options)
	GetConsoByIDHTTPHandler(endpoints, tracer, logger, r, options)
	GetSoireeByIDHTTPHandler(endpoints, tracer, logger, r, options)
	CreateSoireeHTTPHandler(endpoints, tracer, logger, r, options)
	UserJoinSoireeHTTPHandler(endpoints, tracer, logger, r, options)
	UserLeaveSoireeHTTPHandler(endpoints, tracer, logger, r, options)
	GetConnectedFriendsHTTPHandler(endpoints, tracer, logger, r, options)
	DeleteSoireeHTTPHandler(endpoints, tracer, logger, r, options)

	/* Menu */
	GetEstablishmentConsosHTTPHandler(endpoints, tracer, logger, r, options)
	GetEstablishmentMenusHTTPHandler(endpoints, tracer, logger, r, options)
	GetMenuConsosHTTPHandler(endpoints, tracer, logger, r, options)
	CreateMenuHTTPHandler(endpoints, tracer, logger, r, options)
	CreateConsoHTTPHandler(endpoints, tracer, logger, r, options)
	GetMenuFromSoireeHTTPHandler(endpoints, tracer, logger, r, options)

	/* Groups */
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

	/* Order */
	CreateOrderHTTPHandler(endpoints, tracer, logger, r, options)
	GetOrderHTTPHandler(endpoints, tracer, logger, r, options)
	SearchOrdersHTTPHandler(endpoints, tracer, logger, r, options)
	PutOrderHTTPHandler(endpoints, tracer, logger, r, options)
	AnswerOrderHTTPHandler(endpoints, tracer, logger, r, options)
	FailOrderHTTPHandler(endpoints, tracer, logger, r, options)
	UpdateOrderReferenceHTTPHandler(endpoints, tracer, logger, r, options)

	GetConsoByOrderIDHTTPHandler(endpoints, tracer, logger, r, options)
	GetOrdersBySoireeHTTPHandler(endpoints, tracer, logger, r, options)
	UserOrderHTTPHandler(endpoints, tracer, logger, r, options)
	GetSoireeOrdersHTTPHandler(endpoints, tracer, logger, r, options)

	/* Conversation */
	GetLastMessagesHTTPHandler(endpoints, tracer, logger, r, options)
	CreateMessageHTTPHandler(endpoints, tracer, logger, r, options)
	GetConversationByIDHTTPHandler(endpoints, tracer, logger, r, options)

	GetNodeTypeHTTPHandler(endpoints, tracer, logger, r, options)
	AddSuccessHTTPHandler(endpoints, tracer, logger, r, options)
	GetSuccessByValueHTTPHandler(endpoints, tracer, logger, r, options)

	/* Analyse */
	GetAnalysePHTTPHandler(endpoints, tracer, logger, r, options)

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
