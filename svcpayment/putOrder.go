package svcpayment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"strconv"
	"strings"
	"errors"
	"time"

	"github.com/stripe/stripe-go"
	// "github.com/stripe/stripe-go/token"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/refund"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
	"svcws"
)

/************ Steps Handling ***********/
/*** Specific ***/
type specificStepHandler func(Service, context.Context, svcdb.Order, svcdb.Pro, bool) ([]orderProgressNotif, bool, bool, error)
var specificStepHandlerMap = map[string]map[string]specificStepHandler{
	"Issued": { "Execute": stepExecuteIssued, "Condition": stepConditionIssued },
	"Confirmed": { "Execute": stepExecuteConfirmed, "Condition": stepConditionConfirmed },
	"Verified": { "Execute": stepExecuteVerified, "Condition": stepConditionVerified },
	"Ready": { "Execute": stepExecuteReady, "Condition": stepConditionReady },
	"Deliverpaid": { "Execute": stepExecuteDeliverpaid, "Condition": stepConditionDeliverpaid },
	"Completed": { "Execute": stepExecuteCompleted, "Condition": stepConditionCompleted },
}

/*** Execute ***/
/* Nothing */
func stepExecuteIssued(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif

	return notifs, flag, true, nil
}

/* Send websocket to users */
func stepExecuteConfirmed(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	for _, user := range order.Users {
		notifs = append(notifs, orderProgressNotif{
			Order: order,
			UserID: user.User.ID,
			Step: "Confirmed",
			Message: "Accept the order ?",
		})
	}

	return notifs, flag, false, nil
}

/* Create stripe requests */
func stepExecuteVerified(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	var chargesID []string
	stripe.Key = STRIPESKEY

	for _, user := range order.Users {
		var chID string
		// tokenParams := &stripe.TokenParams{
		// 	Customer: user.User.StripeID,
		// }
		// tokenParams.SetStripeAccount(pro.StripeID)
		// token, err := token.New(tokenParams)
		// if err == nil {
		chargeParams := &stripe.ChargeParams{
			Amount: uint64(user.Price),
			Desc: "NightLine fee",
			Statement: "NightLine fee",
			Currency: "eur",
			Fee: uint64(float64(user.Price) * STRIPEFEES),
			NoCapture: true,
			Destination: &stripe.DestinationParams{
				Account: pro.StripeID,
			},
			Customer: user.User.StripeID,
		}
		// chargeParams.SetStripeAccount(pro.StripeID)
		// chargeParams.SetSource(user.User.StripeID)
		// chargeParams.SetSource(token.ID)
		ch, err := charge.New(chargeParams)
		if err == nil {
			chID = (*ch).ID
			chargesID = append(chargesID, chID)
			user.Reference = chID
			err = s.svcdb.UpdateOrderReference(ctx, order.ID, user.User.ID, chID)
		}
		if err != nil {
			fmt.Println("Error on executeVerified : ", err)
			for _, chargeID := range chargesID {
				refund.New(&stripe.RefundParams{Charge: chargeID})
			}
			for _, user := range order.Users {
				notifs = append(notifs, orderProgressNotif{
					Order: order,
					UserID: user.User.ID,
					Step: "Verified",
					Message: "Couldn't reserve funds on bank account",
				})
			}
			return notifs, false, false, err
		}
		notifs = append(notifs, orderProgressNotif{
			Order: order,
			UserID: user.User.ID,
			Step: "Verified",
			Message: "Order price reserved on bank account",
		})
	}
    // }

	return notifs, flag, true, nil
}

/* Send websocket to pros */
func stepExecuteReady(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	return notifs, flag, false, nil
}

/* capture stripe then pass to completed */
func stepExecuteDeliverpaid(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	for _, user := range order.Users {
		_, err := charge.Capture(user.Reference, nil)
		if err != nil {
			for _, user := range order.Users {
				refund.New(&stripe.RefundParams{Charge: user.Reference})
			}
			return notifs, flag, false, err
		}
	}

	return notifs, flag, false, nil
}

/* Completed */
func stepExecuteCompleted(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	return notifs, flag, true, nil
}

/*** Condition ***/
func stepConditionIssued(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	return notifs, flag, true, nil
}

func stepConditionConfirmed(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	if !flag {
		for _, user := range order.Users {
			notifs = append(notifs, orderProgressNotif{
				Order: order,
				UserID: user.User.ID,
				Step: "Confirmed",
				Message: "An user refused the order",
			})
		}
		return notifs, flag, false, errors.New("An user refused the order")
	}
	for _, user := range order.Users {
		if user.Approved != "true" {
			return notifs, true, false, errors.New("Not all users approved the order")
		}
	}

	return notifs, flag, true, nil
}

func stepConditionVerified(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif

	for _, user := range order.Users {
		notifs = append(notifs, orderProgressNotif{
			Order: order,
			UserID: user.User.ID,
			Step: "Verified",
			Message: "Order succesfully passed to the waiter",
		})
	}
	return notifs, flag, true, nil
}

func stepConditionReady(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	if !flag {
		for _, user := range order.Users {
			refund.New(&stripe.RefundParams{Charge: user.Reference})
			notifs = append(notifs, orderProgressNotif{
				Order: order,
				UserID: user.User.ID,
				Step: "Ready",
				Message: "Order refunded",
			})
		}
	} else {
		for _, user := range order.Users {
			notifs = append(notifs, orderProgressNotif{
				Order: order,
				UserID: user.User.ID,
				Step: "Ready",
				Message: "Waiter accepted the order",
			})
		}
	}
	return notifs, flag, true, nil
}

func stepConditionDeliverpaid(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	for _, user := range order.Users {
		notifs = append(notifs, orderProgressNotif{
			Order: order,
			UserID: user.User.ID,
			Step: "Deliverpaid",
			Message: "Waiter delivered the order. Congratulations !",
		})
	}
	return notifs, flag, true, nil
}

func stepConditionCompleted(s Service, ctx context.Context, order svcdb.Order, pro svcdb.Pro, flag bool) ([]orderProgressNotif, bool, bool, error) {
	var notifs []orderProgressNotif
	return notifs, flag, true, nil
}

/*************** Helper ***************/
func (s Service) returnSendNotifs(ctx context.Context, notifs []orderProgressNotif, order svcdb.Order, err error) (svcdb.Order, error) {
	for _, notif := range notifs {
		if notif.Order.ID == order.ID {
			notif.Order = order
		}
		s.svcevent.Push(ctx, svcws.OrderProgress, notif, notif.UserID)
	}
	return order, err
}


/*************** Service ***************/
func (s Service) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
    var notifs []orderProgressNotif
	var pro svcdb.Pro
	var order svcdb.Order
	var stepNode svcdb.StepOrder

	step = strings.Title(strings.ToLower(step))

	/* Get Order */
	order, err := s.svcdb.GetOrder(ctx, orderID)
	if err != nil {
		fmt.Println("PutOrder (GetOrder 1) : " + err.Error())
		return order, err
	}

	/* Error handling */
	if err != nil {
		fmt.Println("PutOrder (GetOrder) : " + err.Error())
		return order, err
	} else if order.ID == 0 {
		err = errors.New("Target order not found")
		fmt.Println("PutOrder (GetOrder) : " + err.Error())
		return order, err
	} else if order.Done == "true" {
		err = errors.New("Order already done")
		fmt.Println("PutOrder (GetOrder) : " + err.Error())
		return order, err
	}
	for i := range order.Steps {
		if order.Steps[i].Name == step {
			stepNode = order.Steps[i]
			break
		}
	}
	if stepNode.ID == 0 {
		err = errors.New("Target step have yet to open")
		fmt.Println("PutOrder (GetOrder) : " + err.Error())
		return order, err
	} else if len(stepNode.Result) > 0 {
		err = errors.New("Target step is closed already")
		fmt.Println("PutOrder (GetOrder) : " + err.Error())
		return order, err
	}

	/* Get Pro */
	pro, err = s.svcdb.GetProBySoiree(ctx, order.Soiree.ID)
	if err != nil {
		fmt.Println("PutOrder (GetProBySoiree) : " + err.Error())
		return order, err
	}
	pro, err = s.svcdb.GetProByIDStripe(ctx, pro.ID)
	if err != nil {
		fmt.Println("PutOrder (GetProByIDStripe) : " + err.Error())
		return order, err
	}

	/* Handler Condition */
	/* Handle specific */
	specificStepHandlerFn, ok := specificStepHandlerMap[step]
	if !ok {
		fmt.Println("PutOrder (StepHandlerMap) : step.Name not in StepHandler")
		return order, errors.New("Internal Error : step.Name not in StepHandler")
	}
	tmpNotifs, validOrder, validStep, err := specificStepHandlerFn["Condition"](s, ctx, order, pro, flag)
	notifs = append(notifs, tmpNotifs...)
	if validOrder == false {
		lastErr := err
		order, err = s.svcdb.FailOrder(ctx, orderID)
		if err != nil {
			fmt.Println("PutOrder (PutOrderFail) : " + err.Error())
			return s.returnSendNotifs(ctx, notifs, order, err)
		}
		return s.returnSendNotifs(ctx, notifs, order, lastErr)
	} else if validStep {
	/* Svcdb PutOrder call */
		order, err = s.svcdb.PutOrder(ctx, orderID, step, flag)
		if err != nil {
			fmt.Println("PutOrder (PutOrder) : " + err.Error())
			return s.returnSendNotifs(ctx, notifs, order, err)
		}

		/* Handler Execute */
		nextStep := nextAllowedSteps[step]
		nextStepHandlerFn, ok := specificStepHandlerMap[nextStep]
		if !ok {
			fmt.Println("PutOrder (StepHandlerMap) : step.Name not in StepHandler")
			return s.returnSendNotifs(ctx, notifs, order, errors.New("Internal Error : step.Name not in StepHandler"))
		}
		if len(nextStep) > 0 {
			tmpNotifs, validateOrder, goNextStep, err := nextStepHandlerFn["Execute"](s, ctx, order, pro, flag)
			notifs = append(notifs, tmpNotifs...)
			if validateOrder == false {
				lastErr := err
				order, err = s.svcdb.FailOrder(ctx, orderID)
				if err != nil {
					fmt.Println("PutOrder (PutOrderFail) : " + err.Error())
					return s.returnSendNotifs(ctx, notifs, order, err)
				}
				return s.returnSendNotifs(ctx, notifs, order, lastErr)
			} else if goNextStep {
				/* Trigger newStep immediately, without checking its result */
				s.PutOrder(ctx, orderID, nextStep, true)
			}
			if err != nil {
				fmt.Println("PutOrder (Execute) : " + err.Error())
				return s.returnSendNotifs(ctx, notifs, order, err)
			}
		}
	}

	return s.returnSendNotifs(ctx, notifs, order, nil)
}

/*************** Endpoint ***************/
type putOrderRequest struct {
	OrderID int64	`json:"id"`
	Step	string	`json:"step"`
	Flag	bool	`json:"flag"`
}

type putOrderResponse struct {
	Order svcdb.Order  `json:"order"`
	Err   string `json:"err,omitempty"`
}

func PutOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(putOrderRequest)
		order, err := svc.PutOrder(ctx, req.OrderID, req.Step, req.Flag)
		if err != nil {
			return putOrderResponse{order, err.Error()}, nil
		}
		return putOrderResponse{order, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPPutOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request putOrderRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).OrderID = orderID

	step := mux.Vars(r)["Step"]
	(&request).Step = step

	flag, err := strconv.ParseBool(mux.Vars(r)["Flag"])
	if err != nil {
		return nil, err
	}
	(&request).Flag = flag

	return request, nil
}

func DecodeHTTPPutOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response putOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func PutOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("PUT").Path(`/orders/{OrderID:[0-9]+}/{Step}/{Flag:(?:true|false)}`).Handler(httptransport.NewServer(
		endpoints.PutOrderEndpoint,
		DecodeHTTPPutOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "PutOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "putOrder",
			"orderID", orderID,
			"step", step,
			"flag", flag,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.PutOrder(ctx, orderID, step, flag)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	v, err := mw.next.PutOrder(ctx, orderID, step, flag)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildPutOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "PutOrder")
		csLogger := log.With(logger, "method", "PutOrder")

		csEndpoint = PutOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "PutOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error) {
	var order svcdb.Order

	request := putOrderRequest{OrderID: orderID, Step: step, Flag: flag}
	response, err := e.PutOrderEndpoint(ctx, request)
	if err != nil {
		return order, err
	}
	order = response.(putOrderResponse).Order
	return order, str2err(response.(putOrderResponse).Err)
}

func EncodeHTTPPutOrderRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(putOrderRequest).OrderID)
	step := fmt.Sprintf("%v", request.(putOrderRequest).Step)
	flag := fmt.Sprintf("%v", request.(putOrderRequest).Flag)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", id, "Step", step, "Flag", flag)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientPutOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"PUT",
		copyURL(u, `/orders/{OrderID:[0-9]+}/{Step}/{Flag:(?:true|false)}`),
		EncodeHTTPPutOrderRequest,
		DecodeHTTPPutOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "PutOrder")(ceEndpoint)
	return ceEndpoint, nil
}
