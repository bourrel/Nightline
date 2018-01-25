package svcdb

import (
	"context"
	"errors"
)

/* Service interface */
type IService interface {
	/* Users */
	CreateUser(ctx context.Context, u User) (User, error)
	SearchUsers(ctx context.Context, query string) ([]SearchResponse, error)
	UpdateUser(ctx context.Context, u User) (User, error)
	GetUserByID(ctx context.Context, userID int64) (User, error)
	GetUser(ctx context.Context, u User) (User, error)
	GetUserProfile(ctx context.Context, userID int64) (Profile, error)
	GetUserPreferences(ctx context.Context, userID int64) ([]Preference, error)
	GetUserSuccess(ctx context.Context, userID int64) ([]Success, error)
	UpdatePreference(ctx context.Context, userID int64, preferences []string) ([]string, error)
	GetRecommendation(ctx context.Context, userID int64) ([]Establishment, error)

	/* Friends */
	InviteFriend(ctx context.Context, userID, friendID int64) (string, int64, int64, error)
	GetUserFriends(ctx context.Context, userID int64) ([]Profile, error)
	GetUsersInvitations(ctx context.Context, userID int64) ([]Invitation, error)
	InvitationAccept(ctx context.Context, invitationID int64) (Profile, Profile, error)
	InvitationDecline(ctx context.Context, invitationID int64) (Profile, Profile, error)
	UsersConnected(ctx context.Context, userID, friendID int64) (bool, error)
	SearchFriends(ctx context.Context, query string, userID int64) ([]SearchResponse, error)

	/* Pro */
	CreatePro(ctx context.Context, p Pro) (Pro, error)
	GetPro(ctx context.Context, p Pro) (Pro, error)
	GetProByIDStripe(ctx context.Context, proID int64) (Pro, error)
	GetProByID(ctx context.Context, userID int64) (Pro, error)
	GetProBySoiree(ctx context.Context, soireeID int64) (Pro, error)
	UpdatePro(ctx context.Context, new Pro) (Pro, error)
	GetProEstablishments(ctx context.Context, soireeID int64) ([]Establishment, error)

	/* Establishments */
	CreateEstablishment(ctx context.Context, e Establishment, proID int64) (Establishment, error)
	GetEstablishments(ctx context.Context) ([]Establishment, error)
	SearchEstablishments(ctx context.Context, query string) ([]SearchResponse, error)
	GetEstablishment(ctx context.Context, estabID int64) (Establishment, error)
	GetEstablishmentFromMenu(ctx context.Context, menuID int64) (Establishment, error)
	GetMenuFromEstablishment(ctx context.Context, menuID int64) ([]Menu, error)
	UpdateEstablishment(c context.Context, new Establishment) (Establishment, error)
	GetEstablishmentSoiree(ctx context.Context, estabID int64) (Soiree, error)
	GetSoireesByEstablishment(ctx context.Context, estabID int64) ([]Soiree, error)
	GetEstablishmentTypes(ctx context.Context) ([]string, error)
	GetEstablishmentType(ctx context.Context, estabID int64) (string, error)
	RateEstablishment(ctx context.Context, rate, estabID, userID int64) (Establishment, error)
	DeleteEstab(ctx context.Context, estabID int64) error

	/* Soiree  */
	GetConsoByID(ctx context.Context, consoID int64) (Conso, error)
	GetSoireeByID(ctx context.Context, soireeID int64) (Soiree, error)
	CreateSoiree(ctx context.Context, menuID, establishmentID int64, u Soiree) (Soiree, error)
	UserJoinSoiree(ctx context.Context, userID int64, soireeID int64) (Soiree, bool, error)
	UserLeaveSoiree(ctx context.Context, userID int64, soireeID int64) (Soiree, bool, error)
	GetConnectedFriends(ctx context.Context, soireeID int64) ([]Profile, error)
	DeleteSoiree(ctx context.Context, soireeID int64) error

	/* Menu */
	GetEstablishmentConsos(ctx context.Context, estabID int64) ([]Conso, error)
	GetEstablishmentMenus(ctx context.Context, estabID int64) ([]Menu, error)
	GetMenuConsos(ctx context.Context, menuID int64) ([]Conso, error)
	CreateMenu(ctx context.Context, establishmentID int64, u Menu) (Menu, error)
	CreateConso(ctx context.Context, establishmentID, menuID int64, c Conso) (Conso, error)
	GetMenuFromSoiree(ctx context.Context, soireeID int64) (Menu, error)

	/* Groups */
	GetGroup(ctx context.Context, groupID int64) (Group, error)
	CreateGroup(ctx context.Context, g Group, userID int64) (Group, error)
	UpdateGroup(ctx context.Context, new Group) (Group, error)
	GroupInvite(ctx context.Context, groupID, friendID int64) (string, int64, error)
	GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error
	GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error
	GetUserGroups(ctx context.Context, userID int64) ([]GroupArrayElement, error)
	DeleteGroup(ctx context.Context, groupID int64) error
	DeleteGroupMember(ctx context.Context, groupID, userID int64) error
	GetGroupsInvitations(ctx context.Context, groupID int64) ([]GroupInvitation, error)

	/* Order */
	GetOrder(ctx context.Context, orderID int64) (Order, error)
	SearchOrders(ctx context.Context, order Order) (Orders, error)
	CreateOrder(ctx context.Context, o Order) (Order, error)
	PutOrder(ctx context.Context, orderID int64, step string, flag bool) (Order, error)
	AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (Order, error)
	FailOrder(ctx context.Context, orderID int64) (Order, error)
	UpdateOrderReference(ctx context.Context, orderID int64, userID int64, reference string) (error)

	UserOrder(ctx context.Context, user User, soiree Soiree, conso Conso) (int64, error)
	GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]Order, error)
	GetSoireeOrders(ctx context.Context, soireeID int64) ([]Order, error)
	GetConsoByOrderID(ctx context.Context, orderID int64) (Conso, error)

	/* Conversation */
	GetLastMessages(ctx context.Context, recipient, initiator int64) ([]Message, error)
	CreateMessage(c context.Context, nodeType string, m Message) (Message, error)
	GetConversationByID(c context.Context, convID int64) (Conversation, error)

	GetNodeType(ctx context.Context, nodeID int64) (string, error)
	AddSuccess(c context.Context, userID int64, successValue string) error
	GetSuccessByValue(_ context.Context, value string) (Success, error)

	/* Analyse */
	GetAnalyseP(c context.Context, estabID int64, soireeID int64) ([]AnalyseP, error)
}

/* Errors definition */
var (
	Err          = errors.New("One basic error")
	ConnError    = errors.New("The server was acting as a gateway or proxy and received an invalid response from the upstream server")
	RequestError = errors.New("The server cannot or will not process the request due to an apparent client error")
	DateErr      = errors.New("Date error, logic invalid operation")
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

/* Service implementation */
func NewService() IService {
	return Service{}
}

type Service struct{}

/* Middleware interface */
type Middleware func(IService) IService
