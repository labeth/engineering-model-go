// ENGMODEL-OWNER-UNIT: FU-PAYMENT-AUTHORIZATION
// ENGMODEL-CODE-DESCRIPTION: defines payment domain events used across authorization and risk paths
pub struct PaymentEvent {
    pub payment_id: String,
    pub amount_cents: u64,
}

impl PaymentEvent {
    pub fn new(payment_id: &str, amount_cents: u64) -> Self {
        Self {
            payment_id: payment_id.to_string(),
            amount_cents,
        }
    }
}
