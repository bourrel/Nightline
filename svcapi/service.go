package svcapi

import (
	"context"
	"errors"

	"svcevent"
	"svcdb"
	"svcpayment"
)

/* Service interface */
type IService interface {
	/* Authentication */
	Login(ctx context.Context, old svcdb.User) (svcdb.User, string, error)
	Register(ctx context.Context, old svcdb.User) (svcdb.User, string, error)

	/* User */
	SearchUsers(ctx context.Context, query string) ([]svcdb.SearchResponse, error)
	SearchFriends(ctx context.Context, query string, userID int64) ([]svcdb.SearchResponse, error)
	GetUser(ctx context.Context, userID int64) (svcdb.Profile, error)
	GetUserPreferences(ctx context.Context, userID int64) ([]svcdb.Preference, error)
	GetUserSuccess(ctx context.Context, userID int64) ([]svcdb.Success, error)
	UpdateUser(ctx context.Context, old svcdb.User) (svcdb.User, error)
	UpdatePreferences(ctx context.Context, userID int64, preferences []string) ([]string, error)
	UpdateStripeUser(ctx context.Context, user svcdb.User, token string) (svcdb.User, error)
	GetRecommendation(ctx context.Context, userID int64) ([]svcdb.Establishment, error)

	/* Friends */
	GetUsersInvitations(ctx context.Context, UserID int64) ([]svcdb.Invitation, error)
	GetUserFriends(ctx context.Context, userID int64) ([]svcdb.Profile, error)
	InviteFriend(ctx context.Context, userID, friendID int64) error
	InvitationAccept(ctx context.Context, invitationID int64) error
	InvitationDecline(ctx context.Context, invitationID int64) error

	/* Establishment */
	GetAllEstablishments(ctx context.Context) ([]svcdb.Establishment, error)
	SearchEstablishments(ctx context.Context, query string) ([]svcdb.SearchResponse, error)
	GetEstablishment(ctx context.Context, estabID int64) (svcdb.Establishment, error)
	GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]svcdb.Menu, error)
	GetSoireeFromEstablishment(ctx context.Context, estabID int64) (svcdb.Soiree, error)
	GetSoireesFromEstablishments(ctx context.Context, estabID int64) ([]svcdb.Soiree, error)
	GetEstablishmentTypes(ctx context.Context) ([]string, error)
	RateEstablishment(ctx context.Context, estabID, userID, rate int64) (svcdb.Establishment, error)

	/* Soiree */
	SoireeJoin(ctx context.Context, UserID, SoireeID int64) (svcdb.Soiree, string, error)
	SoireeOrder(ctx context.Context, UserID, ConsoID, SoireeID int64, Token string) (int64, error)
	SoireeLeave(ctx context.Context, UserID, SoireeID int64, token string) error
	GetMenuFromSoiree(ctx context.Context, soireeID int64) (svcdb.Menu, error)

	/* Group */
	GetGroup(ctx context.Context, groupID int64) (svcdb.Group, error)
	CreateGroup(ctx context.Context, g svcdb.Group, userID int64) (svcdb.Group, error)
	UpdateGroup(ctx context.Context, group svcdb.Group) (svcdb.Group, error)
	GroupInvite(ctx context.Context, groupID, friendID int64) error
	GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error
	GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error
	GetUserGroups(ctx context.Context, userID int64) ([]svcdb.GroupArrayElement, error)
	DeleteGroup(ctx context.Context, groupID int64) error
	DeleteGroupMember(ctx context.Context, groupID, userID int64) error
	GetGroupsInvitations(ctx context.Context, groupID int64) ([]svcdb.GroupInvitation, error)

	/* Notification */
	CreateNotification(ctx context.Context, name, text string, userID int64) (int32, int64, error)

	/* Order */
	GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error)
	CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error)
	AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order, error)
	SearchOrders(ctx context.Context, o svcdb.Order) ([]svcdb.Order, error)
}

/* Errors definition */
var (
	ConnError     = errors.New("The server was acting as a gateway or proxy and received an invalid response from the upstream server.")
	RequestError  = errors.New("The server cannot or will not process the request due to an apparent client error.")
	AuthError     = errors.New("The user might not have the necessary permissions for a resource, or may need an account of some sort.")
	NotFoundError = errors.New("The requested resource could not be found.")
)

/* Misc */
func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func dbToHTTPErr(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case "EOF":
		return NotFoundError
	}
	return err
}

/* Service implementation */
func NewService(db svcdb.IService, event svcevent.IService, payment svcpayment.IService) IService {
	return Service{
		svcdb:    db,
		svcevent: event,
		svcpayment: payment,
	}
}

type Service struct {
	svcdb    svcdb.IService
	svcevent svcevent.IService
	svcpayment svcpayment.IService
}

/* Middleware interface */
type Middleware func(IService) IService
