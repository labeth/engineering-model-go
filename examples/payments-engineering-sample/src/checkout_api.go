package sample

import "fmt"

type CheckoutController struct{}
type CustomerMessageService struct{}

// TRACE-REQS: REQ-PAY-001
func (c *CheckoutController) StartSession(paymentID string, amountCents int) {
	fmt.Printf("checkout-controller: start session for %s (%d cents)\n", paymentID, amountCents)
}

// TRACE-REQS: REQ-PAY-001
func (c *CheckoutController) SubmitAuthorizationRequest(paymentID string) {
	fmt.Printf("checkout-controller: submit authorization request for %s\n", paymentID)
}

// TRACE-REQS: REQ-PAY-006
func (c *CheckoutController) HandleBankUnavailable(paymentID string) {
	fmt.Printf("checkout-controller: bank unavailable for %s, return temporary unavailable\n", paymentID)
}

// TRACE-REQS: REQ-PAY-003
func (m *CustomerMessageService) ShowDeclineReason(paymentID, declineReason string) {
	fmt.Printf("customer-message: payment %s declined (%s)\n", paymentID, declineReason)
}

// TRACE-REQS: REQ-PAY-003
func (m *CustomerMessageService) ShowRetryOption(paymentID string) {
	fmt.Printf("customer-message: offer retry for %s\n", paymentID)
}

// TRACE-REQS: REQ-PAY-006
func (m *CustomerMessageService) ShowTemporaryUnavailable(paymentID string) {
	fmt.Printf("customer-message: service temporarily unavailable for %s\n", paymentID)
}
