package svcapi

import (
	"context"
	"fmt"
	"time"

	stdjwt "github.com/dgrijalva/jwt-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
)

var claims = &stdjwt.StandardClaims{
	ExpiresAt: time.Now().UTC().Add(time.Hour * 1).Unix(), // Add 1 Hour
	Issuer:    "NightLine",
}

/* Endpoints definition */
type Endpoints struct {
	/* Authentication */
	RegisterEndpoint endpoint.Endpoint
	LoginEndpoint    endpoint.Endpoint

	/* User */
	SearchUsersEndpoint        endpoint.Endpoint
	SearchFriendsEndpoint      endpoint.Endpoint
	GetUserPreferencesEndpoint endpoint.Endpoint
	GetUserSuccessEndpoint     endpoint.Endpoint
	GetUserEndpoint            endpoint.Endpoint
	UpdateUserEndpoint         endpoint.Endpoint
	UpdatePreferencesEndpoint  endpoint.Endpoint
	UpdateStripeUserEndpoint   endpoint.Endpoint
	GetRecommendationEndpoint  endpoint.Endpoint

	/* Friends */
	GetUserFriendsEndpoint      endpoint.Endpoint
	InviteFriendEndpoint        endpoint.Endpoint
	GetUsersInvitationsEndpoint endpoint.Endpoint
	InvitationAcceptEndpoint    endpoint.Endpoint
	InvitationDeclineEndpoint   endpoint.Endpoint

	/* Establishment */
	SearchEstablishmentsEndpoint         endpoint.Endpoint
	GetAllEstablishmentsEndpoint         endpoint.Endpoint
	GetEstablishmentEndpoint             endpoint.Endpoint
	GetMenuFromEstablishmentEndpoint     endpoint.Endpoint
	GetSoireeEndpoint                    endpoint.Endpoint
	GetSoireeFromEstablishmentEndpoint   endpoint.Endpoint
	GetSoireesFromEstablishmentsEndpoint endpoint.Endpoint
	GetMenuFromSoireeEndpoint            endpoint.Endpoint
	GetEstablishmentTypesEndpoint        endpoint.Endpoint
	RateEstablishmentEndpoint            endpoint.Endpoint

	/* Soiree */
	SoireeJoinEndpoint  endpoint.Endpoint
	SoireeLeaveEndpoint endpoint.Endpoint
	SoireeOrderEndpoint endpoint.Endpoint

	/* Group */
	GetGroupEndpoint               endpoint.Endpoint
	CreateGroupEndpoint            endpoint.Endpoint
	UpdateGroupEndpoint            endpoint.Endpoint
	GroupInviteEndpoint            endpoint.Endpoint
	GroupInvitationAcceptEndpoint  endpoint.Endpoint
	GroupInvitationDeclineEndpoint endpoint.Endpoint
	GetUserGroupsEndpoint          endpoint.Endpoint
	DeleteGroupEndpoint            endpoint.Endpoint
	DeleteGroupMemberEndpoint      endpoint.Endpoint
	GetGroupsInvitationsEndpoint   endpoint.Endpoint

	/* Notification */
	CreateNotificationEndpoint endpoint.Endpoint

	/* Order */
	GetOrderEndpoint               endpoint.Endpoint
	CreateOrderEndpoint            endpoint.Endpoint
	AnswerOrderEndpoint            endpoint.Endpoint
	SearchOrdersEndpoint           endpoint.Endpoint
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

/* Authentication Middleware */
func EndpointAuthenticationMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// defer func() {
			// 	kf := func(token *stdjwt.Token) (interface{}, error) { return verifKey, nil }
			// 	next = jwt.NewParser(kf, stdjwt.SigningMethodHS256, claims)(next)
			// }()

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
