package svcpayment

var nextAllowedSteps = map[string]string{
	"Issued": "Confirmed",
	"Confirmed": "Verified",
	"Verified": "Ready",
	"Ready": "Deliverpaid",
	"Deliverpaid": "Completed",
	"Completed": "",
}
