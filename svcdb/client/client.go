package client

import (
	"net/url"
	"strings"

	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"

	"svcdb"
)

func New(instance string, tracer stdopentracing.Tracer, logger log.Logger) (svcdb.IService, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	/* Pro */
	clientCreatePro, err := svcdb.ClientCreatePro(u, logger, tracer)
	clientGetPro, err := svcdb.ClientGetPro(u, logger, tracer)
	clientGetProByIDStripe, err := svcdb.ClientGetProByIDStripe(u, logger, tracer)
	clientGetProByID, err := svcdb.ClientGetProByID(u, logger, tracer)
	clientGetProBySoiree, err := svcdb.ClientGetProBySoiree(u, logger, tracer)
	clientUpdatePro, err := svcdb.ClientUpdatePro(u, logger, tracer)
	clientGetProEstablishments, err := svcdb.ClientGetProEstablishments(u, logger, tracer)

	/* User */
	clientCreateUser, err := svcdb.ClientCreateUser(u, logger, tracer)
	clientUpdateUser, err := svcdb.ClientUpdateUser(u, logger, tracer)
	clientGetUserByID, err := svcdb.ClientGetUserByID(u, logger, tracer)
	clientGetUser, err := svcdb.ClientGetUser(u, logger, tracer)
	clientGetUserProfile, err := svcdb.ClientGetUserProfile(u, logger, tracer)
	clientGetUserPreferences, err := svcdb.ClientGetUserPreferences(u, logger, tracer)
	clientGetUserSuccess, err := svcdb.ClientGetUserSuccess(u, logger, tracer)
	clientUpdatePreference, err := svcdb.ClientUpdatePreference(u, logger, tracer)
	clientSearchUser, err := svcdb.ClientSearchUsers(u, logger, tracer)

	/* Friends */
	clientGetUserFriends, err := svcdb.ClientGetUserFriends(u, logger, tracer)
	clientGetUsersInvitations, err := svcdb.ClientGetUsersInvitations(u, logger, tracer)
	clientInviteFriend, err := svcdb.ClientInviteFriend(u, logger, tracer)
	clientInvitationAccept, err := svcdb.ClientInvitationAccept(u, logger, tracer)
	clientInvitationDecline, err := svcdb.ClientInvitationDecline(u, logger, tracer)
	clientUsersConnected, err := svcdb.ClientUsersConnected(u, logger, tracer)
	clientSearchFriends, err := svcdb.ClientSearchFriends(u, logger, tracer)
	clientGetRecommendation, err := svcdb.ClientGetRecommendation(u, logger, tracer)

	/* Establishment */
	clientCreateEstablishment, err := svcdb.ClientCreateEstablishment(u, logger, tracer)
	clientGetEstablishment, err := svcdb.ClientGetEstablishment(u, logger, tracer)
	clientSearchEstablishment, err := svcdb.ClientSearchEstablishments(u, logger, tracer)
	clientGetEstablishments, err := svcdb.ClientGetEstablishments(u, logger, tracer)
	clientGetEstablishmentFromMenu, err := svcdb.ClientGetEstablishmentFromMenu(u, logger, tracer)
	clientGetMenuFromEstablishment, err := svcdb.ClientGetMenuFromEstablishment(u, logger, tracer)
	clientGetEstablishmentSoiree, err := svcdb.ClientGetEstablishmentSoiree(u, logger, tracer)
	clientUpdateEstablishment, err := svcdb.ClientUpdateEstablishment(u, logger, tracer)
	clientGetEstablishmentTypes, err := svcdb.ClientGetEstablishmentTypes(u, logger, tracer)
	clientGetEstablishmentType, err := svcdb.ClientGetEstablishmentType(u, logger, tracer)
	clientRateEstablishment, err := svcdb.ClientRateEstablishment(u, logger, tracer)
	clientDeleteEstab, err := svcdb.ClientDeleteEstab(u, logger, tracer)

	/* Soiree */
	clientGetConsoByID, err := svcdb.ClientGetConsoByID(u, logger, tracer)
	clientGetSoireeByID, err := svcdb.ClientGetSoireeByID(u, logger, tracer)
	clientUserJoinSoiree, err := svcdb.ClientUserJoinSoiree(u, logger, tracer)
	clientUserLeaveSoiree, err := svcdb.ClientUserLeaveSoiree(u, logger, tracer)
	clientGetSoireesByEstablishment, err := svcdb.ClientGetSoireesByEstablishment(u, logger, tracer)
	clientCreateSoiree, err := svcdb.ClientCreateSoiree(u, logger, tracer)
	clientGetConnectedFriends, err := svcdb.ClientGetConnectedFriends(u, logger, tracer)
	clientDeleteSoiree, err := svcdb.ClientDeleteSoiree(u, logger, tracer)

	/* Menu */
	clientCreateMenuEndpoint, err := svcdb.ClientCreateMenu(u, logger, tracer)
	clientCreateConsoEndpoint, err := svcdb.ClientCreateConso(u, logger, tracer)
	clientGetEstablishmentConsos, err := svcdb.ClientGetEstablishmentConsos(u, logger, tracer)
	clientGetEstablishmentMenus, err := svcdb.ClientGetEstablishmentMenus(u, logger, tracer)
	clientGetMenuConsos, err := svcdb.ClientGetMenuConsos(u, logger, tracer)
	clientGetMenuFromSoiree, err := svcdb.ClientGetMenuFromSoiree(u, logger, tracer)

	/* Groups */
	clientGetGroupEndpoint, err := svcdb.ClientGetGroup(u, logger, tracer)
	clientCreateGroupEndpoint, err := svcdb.ClientCreateGroup(u, logger, tracer)
	clientUpdateGroup, err := svcdb.ClientUpdateGroup(u, logger, tracer)
	clientGroupInvite, err := svcdb.ClientGroupInvite(u, logger, tracer)
	clientGroupInvitationAccept, err := svcdb.ClientGroupInvitationAccept(u, logger, tracer)
	clientGroupInvitationDecline, err := svcdb.ClientGroupInvitationDecline(u, logger, tracer)
	clientGetUserGroups, err := svcdb.ClientGetUserGroups(u, logger, tracer)
	clientDeleteGroup, err := svcdb.ClientDeleteGroup(u, logger, tracer)
	clientDeleteGroupMember, err := svcdb.ClientDeleteGroupMember(u, logger, tracer)
	clientGetGroupsInvitations, err := svcdb.ClientGetGroupsInvitations(u, logger, tracer)

	/* Order */
	clientGetOrder, err := svcdb.ClientGetOrder(u, logger, tracer)
	clientSearchOrders, err := svcdb.ClientSearchOrders(u, logger, tracer)
	clientCreateOrderEndpoint, err := svcdb.ClientCreateOrder(u, logger, tracer)
	clientPutOrderEndpoint, err := svcdb.ClientPutOrder(u, logger, tracer)
	clientAnswerOrderEndpoint, err := svcdb.ClientAnswerOrder(u, logger, tracer)
	clientFailOrderEndpoint, err := svcdb.ClientFailOrder(u, logger, tracer)
	clientUpdateOrderReferenceEndpoint, err := svcdb.ClientUpdateOrderReference(u, logger, tracer)

	clientUserOrder, err := svcdb.ClientUserOrder(u, logger, tracer)
	clientGetOrdersBySoiree, err := svcdb.ClientGetOrdersBySoiree(u, logger, tracer)
	clientGetSoireeOrders, err := svcdb.ClientGetSoireeOrders(u, logger, tracer)
	clientGetConsoByOrderID, err := svcdb.ClientGetConsoByOrderID(u, logger, tracer)

	/* Conversation */
	clientGetLastMessages, err := svcdb.ClientGetLastMessages(u, logger, tracer)
	clientCreateMessage, err := svcdb.ClientCreateMessage(u, logger, tracer)
	clientGetConversationByIDEndpoint, err := svcdb.ClientGetConversationByID(u, logger, tracer)

	clientGetNodeTypeEndpoint, err := svcdb.ClientGetNodeType(u, logger, tracer)
	clientAddSuccessEndpoint, err := svcdb.ClientAddSuccess(u, logger, tracer)
	clientGetSuccessByValueEndpoint, err := svcdb.ClientGetSuccessByValue(u, logger, tracer)

	/* Analyse */
	clientGetAnalysePEndpoint, err := svcdb.ClientGetAnalyseP(u, logger, tracer)

	return svcdb.Endpoints{
		/* Pro */
		CreateProEndpoint:            clientCreatePro,
		GetProEndpoint:               clientGetPro,
		GetProByIDStripeEndpoint:     clientGetProByIDStripe,
		GetProByIDEndpoint:           clientGetProByID,
		GetProBySoireeEndpoint:       clientGetProBySoiree,
		UpdateProEndpoint:            clientUpdatePro,
		GetProEstablishmentsEndpoint: clientGetProEstablishments,

		/* User */
		CreateUserEndpoint:         clientCreateUser,
		SearchUsersEndpoint:        clientSearchUser,
		UpdateUserEndpoint:         clientUpdateUser,
		GetUserByIDEndpoint:        clientGetUserByID,
		GetUserEndpoint:            clientGetUser,
		GetUserProfileEndpoint:     clientGetUserProfile,
		GetUserSuccessEndpoint:     clientGetUserSuccess,
		GetUserPreferencesEndpoint: clientGetUserPreferences,
		UpdatePreferenceEndpoint:   clientUpdatePreference,
		GetRecommendationEndpoint:  clientGetRecommendation,

		/* Friends */
		GetUserFriendsEndpoint:      clientGetUserFriends,
		InviteFriendEndpoint:        clientInviteFriend,
		GetUsersInvitationsEndpoint: clientGetUsersInvitations,
		InvitationAcceptEndpoint:    clientInvitationAccept,
		InvitationDeclineEndpoint:   clientInvitationDecline,
		UsersConnectedEndpoint:      clientUsersConnected,
		SearchFriendsEndpoint:       clientSearchFriends,

		/* Establishment */
		CreateEstablishmentEndpoint:      clientCreateEstablishment,
		SearchEstablishmentsEndpoint:     clientSearchEstablishment,
		GetEstablishmentsEndpoint:        clientGetEstablishments,
		GetEstablishmentEndpoint:         clientGetEstablishment,
		GetEstablishmentFromMenuEndpoint: clientGetEstablishmentFromMenu,
		GetMenuFromEstablishmentEndpoint: clientGetMenuFromEstablishment,
		GetEstablishmentSoireeEndpoint:   clientGetEstablishmentSoiree,
		UpdateEstablishmentEndpoint:      clientUpdateEstablishment,
		GetEstablishmentTypesEndpoint:    clientGetEstablishmentTypes,
		GetEstablishmentTypeEndpoint:     clientGetEstablishmentType,
		RateEstablishmentEndpoint:        clientRateEstablishment,
		DeleteEstabEndpoint:              clientDeleteEstab,

		/* Soiree */
		UserOrderEndpoint:                 clientUserOrder,
		GetConsoByIDEndpoint:              clientGetConsoByID,
		GetSoireeByIDEndpoint:             clientGetSoireeByID,
		UserJoinSoireeEndpoint:            clientUserJoinSoiree,
		UserLeaveSoireeEndpoint:           clientUserLeaveSoiree,
		GetSoireesByEstablishmentEndpoint: clientGetSoireesByEstablishment,
		CreateSoireeEndpoint:              clientCreateSoiree,
		GetOrdersBySoireeEndpoint:         clientGetOrdersBySoiree,
		GetConnectedFriendsEndpoint:       clientGetConnectedFriends,
		GetSoireeOrdersEndpoint:           clientGetSoireeOrders,
		DeleteSoireeEndpoint:              clientDeleteSoiree,

		/* Menu */
		GetEstablishmentConsosEndpoint: clientGetEstablishmentConsos,
		GetEstablishmentMenusEndpoint:  clientGetEstablishmentMenus,
		GetMenuConsosEndpoint:          clientGetMenuConsos,
		CreateMenuEndpoint:             clientCreateMenuEndpoint,
		CreateConsoEndpoint:            clientCreateConsoEndpoint,
		GetMenuFromSoireeEndpoint:      clientGetMenuFromSoiree,
		GetConsoByOrderIDEndpoint:      clientGetConsoByOrderID,

		/* Group */
		GetGroupEndpoint:               clientGetGroupEndpoint,
		CreateGroupEndpoint:            clientCreateGroupEndpoint,
		UpdateGroupEndpoint:            clientUpdateGroup,
		GroupInviteEndpoint:            clientGroupInvite,
		GroupInvitationAcceptEndpoint:  clientGroupInvitationAccept,
		GroupInvitationDeclineEndpoint: clientGroupInvitationDecline,
		GetUserGroupsEndpoint:          clientGetUserGroups,
		DeleteGroupEndpoint:            clientDeleteGroup,
		DeleteGroupMemberEndpoint:      clientDeleteGroupMember,
		GetGroupsInvitationsEndpoint:   clientGetGroupsInvitations,

		/* Order */
		SearchOrdersEndpoint: clientSearchOrders,
		GetOrderEndpoint:     clientGetOrder,
		CreateOrderEndpoint:  clientCreateOrderEndpoint,
		PutOrderEndpoint:     clientPutOrderEndpoint,
		AnswerOrderEndpoint:  clientAnswerOrderEndpoint,
		FailOrderEndpoint:    clientFailOrderEndpoint,
		UpdateOrderReferenceEndpoint: clientUpdateOrderReferenceEndpoint,

		/* Conversation */
		GetLastMessagesEndpoint:     clientGetLastMessages,
		CreateMessageEndpoint:       clientCreateMessage,
		GetConversationByIDEndpoint: clientGetConversationByIDEndpoint,

		GetNodeTypeEndpoint:       clientGetNodeTypeEndpoint,
		AddSuccessEndpoint:        clientAddSuccessEndpoint,
		GetSuccessByValueEndpoint: clientGetSuccessByValueEndpoint,

		/* Analyse */
		GetAnalysePEndpoint: clientGetAnalysePEndpoint,
	}, nil

}
