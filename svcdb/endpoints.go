package svcdb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

/* Endpoints definition */
type Endpoints struct {
	/* Pro */
	CreateProEndpoint            endpoint.Endpoint
	GetProEndpoint               endpoint.Endpoint
	GetProByIDStripeEndpoint     endpoint.Endpoint
	UpdateProEndpoint            endpoint.Endpoint
	GetProByIDEndpoint           endpoint.Endpoint
	GetProBySoireeEndpoint       endpoint.Endpoint
	GetProEstablishmentsEndpoint endpoint.Endpoint

	/* User */
	CreateUserEndpoint         endpoint.Endpoint
	SearchUsersEndpoint        endpoint.Endpoint
	UpdateUserEndpoint         endpoint.Endpoint
	GetUserByIDEndpoint        endpoint.Endpoint
	GetUserEndpoint            endpoint.Endpoint
	GetUserProfileEndpoint     endpoint.Endpoint
	GetUserPreferencesEndpoint endpoint.Endpoint
	GetUserSuccessEndpoint     endpoint.Endpoint
	UpdatePreferenceEndpoint   endpoint.Endpoint
	GetRecommendationEndpoint  endpoint.Endpoint

	/* Friends */
	GetUserFriendsEndpoint      endpoint.Endpoint
	InviteFriendEndpoint        endpoint.Endpoint
	GetUsersInvitationsEndpoint endpoint.Endpoint
	InvitationAcceptEndpoint    endpoint.Endpoint
	InvitationDeclineEndpoint   endpoint.Endpoint
	UsersConnectedEndpoint      endpoint.Endpoint
	SearchFriendsEndpoint       endpoint.Endpoint

	/* Establishment */
	CreateEstablishmentEndpoint      endpoint.Endpoint
	SearchEstablishmentsEndpoint     endpoint.Endpoint
	GetEstablishmentsEndpoint        endpoint.Endpoint
	GetEstablishmentEndpoint         endpoint.Endpoint
	GetEstablishmentFromMenuEndpoint endpoint.Endpoint
	GetMenuFromEstablishmentEndpoint endpoint.Endpoint
	UpdateEstablishmentEndpoint      endpoint.Endpoint
	GetEstablishmentSoireeEndpoint   endpoint.Endpoint
	GetEstablishmentTypesEndpoint    endpoint.Endpoint
	GetEstablishmentTypeEndpoint     endpoint.Endpoint
	RateEstablishmentEndpoint        endpoint.Endpoint
	DeleteEstabEndpoint              endpoint.Endpoint

	/* Soiree */
	GetConsoByIDEndpoint              endpoint.Endpoint
	GetSoireeByIDEndpoint             endpoint.Endpoint
	UserLeaveSoireeEndpoint           endpoint.Endpoint
	UserJoinSoireeEndpoint            endpoint.Endpoint
	GetSoireesByEstablishmentEndpoint endpoint.Endpoint
	CreateSoireeEndpoint              endpoint.Endpoint
	GetConnectedFriendsEndpoint       endpoint.Endpoint
	DeleteSoireeEndpoint              endpoint.Endpoint

	/* Menu */
	GetEstablishmentConsosEndpoint endpoint.Endpoint
	GetEstablishmentMenusEndpoint  endpoint.Endpoint
	GetMenuConsosEndpoint          endpoint.Endpoint
	CreateMenuEndpoint             endpoint.Endpoint
	CreateConsoEndpoint            endpoint.Endpoint
	GetMenuFromSoireeEndpoint      endpoint.Endpoint

	/* Groups */
	GetGroupEndpoint               endpoint.Endpoint
	CreateGroupEndpoint            endpoint.Endpoint
	UpdateGroupEndpoint            endpoint.Endpoint
	GroupInviteEndpoint            endpoint.Endpoint
	GroupInvitationDeclineEndpoint endpoint.Endpoint
	GroupInvitationAcceptEndpoint  endpoint.Endpoint
	GetUserGroupsEndpoint          endpoint.Endpoint
	DeleteGroupEndpoint            endpoint.Endpoint
	DeleteGroupMemberEndpoint      endpoint.Endpoint
	GetGroupsInvitationsEndpoint   endpoint.Endpoint

	/* Order */
	GetOrderEndpoint     endpoint.Endpoint
	SearchOrdersEndpoint endpoint.Endpoint
	CreateOrderEndpoint  endpoint.Endpoint
	PutOrderEndpoint     endpoint.Endpoint
	AnswerOrderEndpoint  endpoint.Endpoint
	FailOrderEndpoint    endpoint.Endpoint
	UpdateOrderReferenceEndpoint endpoint.Endpoint

	UserOrderEndpoint         endpoint.Endpoint
	GetConsoByOrderIDEndpoint endpoint.Endpoint
	GetSoireeOrdersEndpoint   endpoint.Endpoint
	GetOrdersBySoireeEndpoint endpoint.Endpoint

	/* Conversation */
	GetLastMessagesEndpoint     endpoint.Endpoint
	CreateMessageEndpoint       endpoint.Endpoint
	GetConversationByIDEndpoint endpoint.Endpoint

	GetNodeTypeEndpoint       endpoint.Endpoint
	AddSuccessEndpoint        endpoint.Endpoint
	GetSuccessByValueEndpoint endpoint.Endpoint

	/* Analyse */
	GetAnalysePEndpoint	endpoint.Endpoint
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
