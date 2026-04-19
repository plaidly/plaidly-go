# plaidly-go

Official Go SDK for the [Plaidly](https://plaidly.io) cryptocurrency payment API.

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
    client := plaidly.NewClient("pk_live_...")

    session, err := client.Sessions.Create(context.Background(), plaidly.CreateSessionRequest{
        Amount:      "10.00",
        Currency:    "USDC",
        Chain:       "solana",
        Network:     "mainnet",
        CallbackURL: "https://yoursite.com/webhook",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Send funds to:", session.WalletAddress)
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
client := plaidly.NewClient(
    "pk_live_...",
    plaidly.WithBaseURL("https://sandbox.api.plaidly.io"),
    plaidly.WithHTTPClient(myHTTPClient),
)
```

## API Reference

See [docs.plaidly.io](https://docs.plaidly.io) for full API documentation.
