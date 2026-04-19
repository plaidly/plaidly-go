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

## Webhook Verification

```go
import plaidly "github.com/plaidly/plaidly-go"

http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    sig := r.Header.Get("X-Plaidly-Signature")

    if !plaidly.VerifyWebhookSignature(body, sig, os.Getenv("PLAIDLY_WEBHOOK_SECRET")) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }
    // handle event
    w.WriteHeader(http.StatusNoContent)
})
```

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
