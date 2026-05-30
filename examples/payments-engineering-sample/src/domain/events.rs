// ENGMODEL-OWNER-UNIT: FU-PAYMENT-AUTHORIZATION
// ENGMODEL-CODE-DESCRIPTION: defines payment domain events used across authorization and risk paths
// ENGMODEL-LINKS: FLOW-CUSTOMER-CHECKOUT, DO-PAYMENTS-AUTH-REQUEST, FU-PAYMENT-AUTHORIZATION
pub struct PaymentEvent {
    pub payment_id: String,
    pub amount_cents: u64,
}

impl PaymentEvent {
    // ENGMODEL-LINKS: FLOW-CUSTOMER-CHECKOUT, DO-PAYMENTS-AUTH-REQUEST
    // TRLC-LINKS: REQ-PAY-001
    pub fn new(payment_id: &str, amount_cents: u64) -> Self {
        Self {
            payment_id: payment_id.to_string(),
            amount_cents,
        }
    }
}
