package main

import (
	"context"
	"fmt"
	"log"
	"os"

	plaidly "github.com/plaidly/plaidly-go"
)

func ptr[T any](v T) *T {
	return &v
}

func main() {
	ctx := context.Background()

	client, err := plaidly.NewClient(os.Getenv("PLAIDLY_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	webhookURL := os.Getenv("PLAIDLY_WEBHOOK_URL")
	merchant, err := client.RegisterMerchant(ctx, plaidly.RegisterMerchantRequest{
		Name:       "Acme Demo Store",
		WebhookUrl: ptr(webhookURL),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("merchant:", merchant.Name)

	demo, err := client.CreateDemoPaymentSession(ctx)
	if err != nil {
		log.Fatal(err)
	}

	done, err := client.SimulatePayment(ctx, demo.SessionId)
	if err != nil {
		log.Fatal(err)
	}

	receipt, settled, err := plaidly.GetSessionReceiptIfSettled(ctx, client, done.SessionId)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("session:", settled.SessionId, "receipt-bytes:", len(receipt))

	payload := []byte(`{"event_type":"payment_session.completed","session_id":"` + settled.SessionId + `","status":"completed","amount":1,"currency":"USDC","chain":"solana","network":"testnet","timestamp":"1700000000"}`)
	signature := os.Getenv("PLAIDLY_WEBHOOK_SIGNATURE")
	event, err := plaidly.VerifyWebhook(payload, signature, os.Getenv("PLAIDLY_WEBHOOK_SECRET"), plaidly.DefaultWebhookTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("webhook:", event.EventType, event.SessionID, event.Status)
}
