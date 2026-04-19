package plaidly

// SessionStatus represents the lifecycle state of a payment session.
type SessionStatus string

const (
	SessionStatusAwaitingPayment SessionStatus = "awaiting_payment"
	SessionStatusProcessing      SessionStatus = "processing"
	SessionStatusCompleted       SessionStatus = "completed"
	SessionStatusExpired         SessionStatus = "expired"
	SessionStatusFailed          SessionStatus = "failed"
)

// Session is a Plaidly payment session.
type Session struct {
	ID            string            `json:"id"`
	MerchantID    string            `json:"merchantId"`
	Amount        string            `json:"amount"`
	Currency      string            `json:"currency"`
	Chain         string            `json:"chain"`
	Network       string            `json:"network"`
	Status        SessionStatus     `json:"status"`
	WalletAddress string            `json:"walletAddress"`
	CallbackURL   string            `json:"callbackUrl"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     string            `json:"createdAt"`
	ExpiresAt     string            `json:"expiresAt"`
}

// CreateSessionRequest is the request body for POST /v1/sessions.
type CreateSessionRequest struct {
	Amount         string            `json:"amount"`
	Currency       string            `json:"currency"`
	Chain          string            `json:"chain"`
	Network        string            `json:"network,omitempty"`
	CallbackURL    string            `json:"callback_url,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// ListSessionsResponse is returned by GET /v1/sessions.
type ListSessionsResponse struct {
	Sessions []Session `json:"sessions"`
	Total    int       `json:"total"`
}

// SimulatePaymentRequest is the body for POST /v1/sessions/{id}/simulate.
type SimulatePaymentRequest struct {
	TxHash string `json:"tx_hash,omitempty"`
}

// PayoutStatus represents the lifecycle state of a payout.
type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted  PayoutStatus = "completed"
	PayoutStatusFailed     PayoutStatus = "failed"
)

// Payout represents a merchant withdrawal.
type Payout struct {
	ID         string       `json:"id"`
	MerchantID string       `json:"merchantId"`
	Amount     string       `json:"amount"`
	Currency   string       `json:"currency"`
	Chain      string       `json:"chain"`
	Network    string       `json:"network"`
	Status     PayoutStatus `json:"status"`
	ToAddress  string       `json:"toAddress"`
	TxHash     string       `json:"txHash,omitempty"`
	CreatedAt  string       `json:"createdAt"`
}

// CreatePayoutRequest is the request body for POST /v1/payouts.
type CreatePayoutRequest struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
	Chain    string `json:"chain"`
	Network  string `json:"network,omitempty"`
	Address  string `json:"address"`
}

// Merchant represents a Plaidly merchant account.
type Merchant struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	APIKey     string `json:"apiKey"`
	WebhookURL string `json:"webhookUrl"`
	Sandbox    bool   `json:"sandbox"`
	CreatedAt  string `json:"createdAt"`
}

// CreateMerchantRequest is the request body for POST /v1/merchants.
type CreateMerchantRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Sandbox    bool   `json:"sandbox"`
}

// Faucet describes a testnet faucet available in sandbox mode.
type Faucet struct {
	Chain       string `json:"chain"`
	Network     string `json:"network"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
