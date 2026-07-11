# plaidly-go

Official Go SDK for the [Plaidly](https://plaidly.io) cryptocurrency payment API.

Types and the underlying HTTP client are auto-generated from the Plaidly
OpenAPI 3.1 spec with [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen).
The `Client` type in this package is a hand-written wrapper that adds the
`X-API-Key` header, retries on transient 5xx failures, and surfaces typed
`*plaidly.Error` values.

## Installation

```bash
go get github.com/plaidly/plaidly-go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    plaidly "github.com/plaidly/plaidly-go"
)

func main() {
    client, err := plaidly.NewClient("pk_live_...")
    if err != nil {
        log.Fatal(err)
    }

    session, err := client.CreatePaymentSession(context.Background(), plaidly.CreatePaymentSessionRequest{
        Amount:    10.0,
        ExpiresIn: "15m",
        PaymentMethod: plaidly.PaymentMethod{
            MethodID: 0,
            Chain:    "solana",
            Token:    "USDC",
            Network:  "mainnet",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Send funds to:", session.Address)
}
```

## Idempotency

`CreatePaymentSession` and `RequestPayout` accept `plaidly.WithIdempotencyKey`,
matching the live `Idempotency-Key` header contract on
`POST /v1/payment_sessions` and `POST /v1/payouts` (merchant-scoped, 24h TTL,
payload-fingerprinted). Replaying the same key with an identical payload
returns the original resource; replaying it with a different payload fails
with a `409` whose `*plaidly.Error.Code` is `plaidly.ErrorCodeConflict`
(`IsIdempotencyConflict(err)` checks this in one call).

```go
session, err := client.CreatePaymentSession(ctx, req, plaidly.WithIdempotencyKey(componoPaymentUUID))
if plaidly.IsIdempotencyConflict(err) {
    // componoPaymentUUID was already used with a different payload
}
```

## Demo & sandbox

```go
// Public demo session (no API key required server-side).
demo, _ := client.CreateDemoPaymentSession(ctx)

// Instantly complete a demo/sandbox session.
done, _ := client.SimulatePayment(ctx, demo.SessionId)
fmt.Println(plaidly.IsSuccess(done.Status)) // true

methods, _ := client.ListPaymentMethods(ctx)       // GET /v1/payment_methods
rates, _ := client.GetRates(ctx, "ETH", "SOL")     // GET /v1/rates?symbols=ETH,SOL
faucets, _ := client.ListSandboxFaucets(ctx)       // GET /v1/sandbox/faucets
```

## Webhook Verification

Plaidly signs each delivery with `X-Plaidly-Signature: t=<unix>,v1=<hex>`, where
the hex value is `HMAC-SHA256(secret, "<t>.<rawBody>")`. `VerifyWebhook` does a
constant-time compare and enforces a timestamp tolerance (default 5 minutes).

```go
import plaidly "github.com/plaidly/plaidly-go"

http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    sig := r.Header.Get(plaidly.SignatureHeader)

    event, err := plaidly.VerifyWebhook(body, sig, os.Getenv("PLAIDLY_WEBHOOK_SECRET"), plaidly.DefaultWebhookTolerance)
    if err != nil {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }
    if event.EventType == plaidly.EventPaymentCompleted {
        // fulfil order for event.SessionID
    }
    w.WriteHeader(http.StatusNoContent)
})
```

During webhook-secret rotation, accept the new and previous secret for a short
window:

```go
event, err := plaidly.VerifyWebhookAny(
    body,
    sig,
    []string{
        os.Getenv("PLAIDLY_WEBHOOK_SECRET_NEW"),
        os.Getenv("PLAIDLY_WEBHOOK_SECRET_OLD"),
    },
    plaidly.DefaultWebhookTolerance,
    time.Now(),
)
```

## Telegram bots

The `telegram` subpackage renders bot-ready invoices and waits for payment via
webhook or polling. The keyboard structs are framework-agnostic — map them onto
whatever Telegram library you use.

```go
import (
    plaidly "github.com/plaidly/plaidly-go"
    "github.com/plaidly/plaidly-go/telegram"
)

client, _ := plaidly.NewClient("pk_live_...")
tg := telegram.New(client)

// 1. Create an invoice and present it to the user.
inv, err := tg.CreateInvoice(ctx, 12.50, "USDC", "solana", "mainnet",
    map[string]any{"order_id": "A-1001"})
if err != nil {
    log.Fatal(err)
}
// inv.MessageText  -> HTML (parse_mode=HTML) with amount/chain/address
// inv.PayButton    -> telegram.URLButton{Text, URL}      (open in browser)
// inv.WebAppButton -> telegram.WebAppButton{Text, URL}   (open as Web App)
//
// bot.Send(chatID, inv.MessageText, &telebot.ReplyMarkup{
//     InlineKeyboard: [][]telebot.InlineButton{{
//         {Text: inv.PayButton.Text, URL: inv.PayButton.URL},
//     }},
// })

// Use CreateInvoiceWithOptions + InvoiceOptions.IdempotencyKey to make a
// button tap (or Telegram's own delivery retries) safe to fire twice for the
// same order:
//   tg.CreateInvoiceWithOptions(ctx, telegram.InvoiceOptions{
//       Amount: 12.50, Token: "USDC", Chain: "solana", Network: "mainnet",
//       IdempotencyKey: fmt.Sprintf("tg-%d-%s", chatID, orderID),
//   })

// 2a. Confirm payment via webhook.
http.HandleFunc("/plaidly/webhook", func(w http.ResponseWriter, r *http.Request) {
    event, err := telegram.VerifyWebhook(r, webhookSecret, plaidly.DefaultWebhookTolerance)
    if err != nil {
        http.Error(w, "bad signature", http.StatusUnauthorized)
        return
    }
    if event.EventType == plaidly.EventPaymentCompleted {
        // notify the buyer in Telegram
    }
    w.WriteHeader(http.StatusNoContent)
})

// 2b. Or poll to completion (for long-polling bots with no public webhook).
session, err := tg.PollSession(ctx, inv.Session.SessionId, 3*time.Second)
if err == nil && plaidly.IsSuccess(session.Status) {
    // payment settled
}
```

## Multi-method checkout intents (experimental — not deployed)

`CreateCheckoutIntent`, `GetCheckoutIntent`, and `SelectCheckoutMethod` are
built against the BDT-541 phase-1 checkout-intent contract (a session that
carries a *set* of merchant-approved payment methods plus a policy version,
rather than one fixed chain/token), so the SDK is ready the moment the
server side (BDT-528/BDT-529) ships. **As of this SDK version, the
`/v1/checkout_intents*` endpoints do not exist on any deployed Plaidly API**
— calling these methods against `api.plaidly.io` today returns 404. They are
covered by mock-transport tests only; do not use them in production code
until this note is removed. See `CHANGELOG.md`.

```go
intent, err := client.CreateCheckoutIntent(ctx, plaidly.CreateCheckoutIntentRequest{
    Amount:   50,
    Currency: "USDC",
    Methods: []plaidly.CheckoutMethodOption{
        {Chain: "solana", Network: "mainnet", Token: "USDC"},
        {Chain: "ethereum", Network: "mainnet", Token: "USDC"},
    },
}, plaidly.WithIdempotencyKey(componoPaymentUUID))

selected, err := client.SelectCheckoutMethod(ctx, intent.IntentID, plaidly.SelectCheckoutMethodRequest{
    Chain: "solana", Network: "mainnet", Token: "USDC",
})
// selected.DepositAddress / selected.PaymentURL / selected.QRCodeURL / selected.DeepLink
```

## Agent integrations

The same payment-session and webhook primitives are used by the agent docs.
See `examples/README.md` for a runnable onboarding and verification example.

## Configuration

```go
client, _ := plaidly.NewClient(
    "pk_live_...",
    plaidly.WithBaseURL("https://sandbox.api.plaidly.io"),
    plaidly.WithHTTPClient(myHTTPClient),
)
```

## Escape hatch — generated client

```go
resp, err := client.Raw().GetMe(ctx)
```

## Regenerating from the spec

The committed copy of the Plaidly spec lives at `spec/openapi.yaml`.

```bash
make generate              # default
make generate SPEC=path/to/openapi.yaml
make generate OAPI_CODEGEN_VERSION=v2.4.1
```

Generated output: `generated/plaidlyapi/plaidlyapi.gen.go`. Do not edit by hand.

Pinned versions:

- `oapi-codegen` — `v2.4.1` (same as plaidly-api)

## API Reference

See [docs.plaidly.io](https://docs.plaidly.io) for full API documentation.
