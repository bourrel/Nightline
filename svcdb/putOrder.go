package svcdb

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

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/************ Steps Handling ***********/
/*** Misc ***/
func stepDone(order Order, step StepOrder, flag bool) (string, map[string]interface{}, error) {
	args := make(map[string]interface{})

	req := `MATCH (o:ORDER)-[:DONE]->(st:STEP)
            WHERE ID(o) = {oid} AND ID(st) = {stid}
            SET st.Result = {stresult}`
	args["oid"] = order.ID
	args["stid"] = step.ID
	args["stresult"] = strconv.FormatBool(flag)
	return req, args, nil
}

func stepOpenNext(order Order, step string) (string, map[string]interface{}, error) {
	args := make(map[string]interface{})
	req := `MATCH (o:ORDER) WHERE ID(o) = {oid}
                CREATE (st:STEP {Name: {stname}, Date: {stdate}}),
                (o)-[:DONE]->(st)`
	args["oid"] = order.ID
	args["stname"] = step
	args["stdate"] = time.Now().Format(timeForm)

	return req, args, nil
}

/*** Specific ***/
type specificStepHandler func(Order, bool) (bool, []string, []map[string]interface{}, error)
var specificStepHandlerMap = map[string]map[string]specificStepHandler{
	"Issued": {  "Condition": stepConditionIssued },
	"Confirmed": { "Condition": stepConditionConfirmed },
	"Verified": { "Condition": stepConditionVerified },
	"Ready": { "Condition": stepConditionReady },
	"Deliverpaid": { "Condition": stepConditionDeliverpaid },
	"Completed": { "Condition": stepConditionCompleted },
}

/*** Condition ***/
func stepConditionIssued(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

func stepConditionConfirmed(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

func stepConditionVerified(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

func stepConditionReady(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

func stepConditionDeliverpaid(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

func stepConditionCompleted(order Order, flag bool) (bool, []string, []map[string]interface{}, error) {
	var reqs []string
	var argss []map[string]interface{}
	return true, reqs, argss, nil
}

/*** Handler ***/
func stepHandler(order Order, step StepOrder, flag bool) ([]string, []map[string]interface{}, error) {
	var finalReqs, reqs []string
	var finalArgss, argss []map[string]interface{}

	/* Handle specific */
	specificStepHandlerFn, ok := specificStepHandlerMap[step.Name]
	if !ok {
		return finalReqs, finalArgss, errors.New("Internal Error : step.Name not in StepHandler")
	}
	validated, reqs, argss, err := specificStepHandlerFn["Condition"](order, flag)
	if err != nil {
		return finalReqs, finalArgss, err
	}
	if validated == true {
		copy(finalReqs, reqs)
		copy(finalArgss, argss)

		/* Mark step result */
		req, args, err := stepDone(order, step, flag)
		if err != nil {
			return finalReqs, finalArgss, err
		}
		finalReqs = append(finalReqs, req)
		finalArgss = append(finalArgss, args)
		for i := range order.Steps {
			if order.Steps[i].Name == step.Name {
				order.Steps[i].Result = "wip"
				break
			}
		}

		if flag == true {
			/* Get next steps */
			nextStep, err := getNextAllowedStep(order)
			if err != nil {
				return finalReqs, finalArgss, err
			}

			/* Open them */
			if len(nextStep) > 0 {
				req, args, err = stepOpenNext(order, nextStep)
				if err != nil {
					return finalReqs, finalArgss, err
				}
				finalReqs = append(finalReqs, req)
				finalArgss = append(finalArgss, args)
			} else {
				/* Set node as done */
				finalReqs = append(finalReqs, `MATCH (o:ORDER) WHERE ID(o) = {oid} SET o.Done = {odone}`)
				finalArgss = append(finalArgss, map[string]interface{}{
					"oid": order.ID,
					"odone": "true",
				})
			}
		} else {
			/* Set node as done */
			finalReqs = append(finalReqs, `MATCH (o:ORDER) WHERE ID(o) = {oid} SET o.Done = {odone}`)
			finalArgss = append(finalArgss, map[string]interface{}{
				"oid": order.ID,
				"odone": "false",
			})
		}
	}
	return finalReqs, finalArgss, nil
}

/*************** Service ***************/
func (s Service) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (Order, error) {
	var order Order
	var stepNode StepOrder

	step = strings.Title(strings.ToLower(step))

	/* Get Order */
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		fmt.Println("PutOrder (GetOrder 1) : " + err.Error())
		return order, err
	}

	/* Error handling */
	if err != nil {
		fmt.Println("PutOrder (ErrorHandling) : " + err.Error())
		return order, err
	} else if order.ID == 0 {
		err = errors.New("Target order not found")
		fmt.Println("PutOrder (ErrorHandling) : " + err.Error())
		return order, err
	} else if order.Done == "true" {
		err = errors.New("Order already done")
		fmt.Println("PutOrder (ErrorHandling) : " + err.Error())
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
		fmt.Println("PutOrder (ErrorHandling) : " + err.Error())
		return order, err
	} else if len(stepNode.Result) > 0 {
		err = errors.New("Target step is closed already")
		fmt.Println("PutOrder (ErrorHandling) : " + err.Error())
		return order, err
	}

	/* Step handling */
	reqs, argss, err := stepHandler(order, stepNode, flag)
	if err != nil {
		fmt.Println("PutOrder (stepHandlerCall) : " + err.Error())
		return order, err
	}

	if len(reqs) > 0 && len(argss) > 0 {
		/* Conn */
		conn, err := WaitConnection(5)
		if err != nil {
			fmt.Println("PutOrder (WaitConnection) : " + err.Error())
			return order, err
		}
		defer CloseConnection(conn)

		/* Pipeline changes */
		pipeline, err := conn.PreparePipeline(reqs...)
		if err != nil {
			fmt.Println("PutOrder (PreparePipeline) : " + err.Error())
			return order, err
		}

		_, err = pipeline.ExecPipeline(argss...)
		if err != nil {
			fmt.Println("PutOrder (ExecPipeline) : " + err.Error())
			return order, err
		}
		pipeline.Close()

		/* Get updated order */
		order, err = s.GetOrder(ctx, order.ID)
		if err != nil {
			fmt.Println("PutOrder (GetOrder 2) : " + err.Error())
			return order, err
		}
	}

	return order, nil
}

/*************** Endpoint ***************/
type putOrderRequest struct {
	OrderID int64	`json:"id"`
	Step	string	`json:"step"`
	Flag	bool	`json:"flag"`
}

type putOrderResponse struct {
	Order Order  `json:"order"`
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
func (mw serviceLoggingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (Order, error) {
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
func (mw serviceInstrumentingMiddleware) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (Order, error) {
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
func (e Endpoints) PutOrder(ctx context.Context, orderID int64, step string, flag bool) (Order, error) {
	var order Order

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
