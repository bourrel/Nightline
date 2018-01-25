package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) InviteFriend(_ context.Context, userID, friendID int64) (string, int64, int64, error) {
	var p Profile
	var successID int64

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("InviteFriend (WaitConnection) : " + err.Error())
		return "", 0, 0, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER) WHERE ID(u) = {userID}
		MATCH (f:USER) WHERE ID(f) = {friendID}
		OPTIONAL MATCH (u)-[]-(s:SUCCESS)
		CREATE (u)-[i:INVITE {
			Date: {date}
		}]->(f) RETURN u, i, ID(s)
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error InviteFriend (PrepareNeo) : " + err.Error())
		return "", 0, 0, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"userID":   userID,
		"friendID": friendID,
		"date":     time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("Error InviteFriend (QueryNeo) : " + err.Error())
		return "", 0, 0, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("Error InviteFriend (NextNeo) : " + err.Error())
		return "", 0, 0, err
	}

	(&p).NodeToProfile(data[0].(graph.Node))
	invitID := data[1].(graph.Relationship).RelIdentity
	if data[2] != nil {
		successID = data[2].(int64)
	}

	return p.Pseudo, invitID, successID, nil
}

/*************** Endpoint ***************/
type inviteFriendRequest struct {
	UserID   int64 `json:"userID"`
	FriendID int64 `json:"friendID"`
}

type inviteFriendResponse struct {
	Name         string `json:"name"`
	InvitationID int64  `json:"id"`
	SuccessID    int64  `json:"successID"`
	Err          string `json:"err,omitempty"`
}

func InviteFriendEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(inviteFriendRequest)

		// Check if there is a connection between users
		connected, err := svc.UsersConnected(ctx, req.UserID, req.FriendID)
		if err != nil {
			fmt.Println("Error InviteFriendEndpoint 1 : ", err.Error())
			return inviteFriendResponse{Err: err.Error()}, nil
		} else if connected == true {
			return inviteFriendResponse{Err: "The users are already friends or an invitation is already pending"}, nil
		} else if req.UserID == req.FriendID {
			return inviteFriendResponse{Err: "You can't invite yourself"}, nil
		}

		// Send an invitation
		name, invitID, success, err := svc.InviteFriend(ctx, req.UserID, req.FriendID)
		if err != nil {
			fmt.Println("Error InviteFriendEndpoint : ", err.Error())
			return inviteFriendResponse{Err: err.Error()}, nil
		}
		return inviteFriendResponse{Name: name, InvitationID: invitID, SuccessID: success}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPInviteFriendRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request inviteFriendRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	friendID, err := strconv.ParseInt(mux.Vars(r)["FriendID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).FriendID = friendID

	return request, nil
}

func DecodeHTTPInviteFriendResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response inviteFriendResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func InviteFriendHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/friends/{UserID:[0-9]+}/invite/{FriendID}").Handler(httptransport.NewServer(
		endpoints.InviteFriendEndpoint,
		DecodeHTTPInviteFriendRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InviteFriend", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InviteFriend(ctx context.Context, userID, friendID int64) (string, int64, int64, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "inviteFriend",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.InviteFriend(ctx, userID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InviteFriend(ctx context.Context, userID, friendID int64) (string, int64, int64, error) {
	name, invitID, success, err := mw.next.InviteFriend(ctx, userID, friendID)
	mw.ints.Add(1)
	return name, invitID, success, err
}

/*************** Main ***************/
/* Main */
func BuildInviteFriendEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "InviteFriend")
		gefmLogger := log.With(logger, "method", "InviteFriend")

		gefmEndpoint = InviteFriendEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "InviteFriend")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) InviteFriend(ctx context.Context, userID, friendID int64) (string, int64, int64, error) {
	request := inviteFriendRequest{UserID: userID, FriendID: friendID}
	response, err := e.InviteFriendEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error InviteFriend : ", err.Error())
		return "", 0, 0, err
	}

	r := response.(inviteFriendResponse)
	return r.Name, r.InvitationID, r.SuccessID, str2err(r.Err)
}

func EncodeHTTPInviteFriendRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	uid := strconv.FormatInt(request.(inviteFriendRequest).UserID, 10)
	fid := strconv.FormatInt(request.(inviteFriendRequest).FriendID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", uid, "FriendID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPInviteFriendRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientInviteFriend(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/friends/{UserID:[0-9]+}/invite/{FriendID}"),
		EncodeHTTPInviteFriendRequest,
		DecodeHTTPInviteFriendResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "InviteFriend")(gefmEndpoint)
	return gefmEndpoint, nil
}
