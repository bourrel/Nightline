package svcapi

type successNotif struct {
	Name      string `json:"name"`
	SuccessID int64  `json:"successID"`
}

type userInviteNotif struct {
	UserID       int64  `json:"userID"`
	Name         string `json:"name"`
	InvitationID int64  `json:"invitationID"`
}

type groupInviteNotif struct {
	GroupID      int64  `json:"groupID"`
	Name         string `json:"name"`
	InvitationID int64  `json:"invitationID"`
}

type invitationAcceptNotif struct {
	Name         string `json:"name"`
	UserID       int64  `json:"userID"`
	InvitationID int64  `json:"invitationID"`
	Accepted     bool   `json:"accepted"`
}

type invitationDeclineNotif struct {
	Name         string `json:"name"`
	UserID       int64  `json:"userID"`
	InvitationID int64  `json:"invitationID"`
	Accepted     bool   `json:"accepted"`
}
