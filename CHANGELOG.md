# Changelog

All notable changes to this SDK are recorded here. This file did not exist
before this change; entries below cover the current `Unreleased` work only.

## Unreleased (BDT-530)

Contract work for Compono billing's integration. Not tagged yet — see
"Release status" below for why.

### Real, deployed-endpoint features

These are wired against endpoints that are live on `api.plaidly.io` today.

- `WithIdempotencyKey(key string) RequestOption`, plumbed through
  `CreatePaymentSession` and `RequestPayout` (both now take
  `opts ...RequestOption`, backward compatible with the old 2-argument call
  shape). Matches the live `Idempotency-Key` header contract on
  `POST /v1/payment_sessions` and `POST /v1/payouts`: merchant-scoped, 24h
  TTL, payload-fingerprinted. Replaying an identical payload under the same
  key returns the canonical resource; a different payload under the same key
  fails with `409` / `Error.Code == ErrorCodeConflict`.
- `telegram.InvoiceOptions.IdempotencyKey`, threaded into
  `CreateInvoiceWithOptions`'s underlying `CreatePaymentSession` call.
- **Bug fix:** `Error.Code` was typed `string` and always empty for real API
  errors, because the wire format is `{"code": <integer>, "message": string}`
  and `decodeResponse` unmarshaled it into a `string` field, which
  `encoding/json` silently fails to populate (the error was discarded via
  `_ = json.Unmarshal(...)`). `Error.Code` is now `int64`, matching the API
  and the SDK's own generated `plaidlyapi.Error` type, which was already
  correctly typed — only the hand-written `decodeResponse` had drifted from
  it. Added named constants (`ErrorCodeInternal`, `ErrorCodeNotFound`,
  `ErrorCodeBadRequest`, `ErrorCodeTooManyRequests`, `ErrorCodeConflict`,
  `ErrorCodeClientCancelled`, `ErrorCodeForbidden`, `ErrorCodeUnavailable`)
  mirroring `plaidly-api/internal/api/codes.go` 1:1, plus
  `(*Error).IsConflict()`, `(*Error).IsIdempotencyConflict()`, and the
  package-level `IsIdempotencyConflict(err error) bool` helper.
- Golden signed-webhook fixtures at `testdata/webhook_fixtures.json` (3 valid
  vectors including a secret-rotation case, 5 invalid vectors: tampered
  payload, wrong secret, expired timestamp, malformed header, missing
  header), loaded by `golden_webhook_test.go`. The verifier itself
  (`VerifyWebhook`/`VerifyWebhookAt`/`VerifyWebhookAny`) is unchanged — this
  only adds checked-in golden fixtures on top of the existing inline test
  vector, so the signature scheme has a reusable, cross-checkable reference
  independent of any one test file.

### Experimental — anticipating endpoints that are NOT deployed yet

BDT-528 (multi-method checkout intents) and BDT-529 (payment-method policy)
are plaidly-api tickets and have not shipped to any environment as of this
change. The methods below were built against the epic's locked phase-1
contract so the SDK is ready the moment they land, per BDT-530's
coordination note. **Calling any of these against a real Plaidly API today
returns 404.** They are covered by mock-transport tests only
(`checkout_intent_test.go`), never against a live server.

- `CreateCheckoutIntent(ctx, CreateCheckoutIntentRequest, ...RequestOption) (*CheckoutIntent, error)`
  — anticipates `POST /v1/checkout_intents`.
- `GetCheckoutIntent(ctx, intentID string, ...RequestOption) (*CheckoutIntent, error)`
  — anticipates `GET /v1/checkout_intents/{intent_id}`.
- `SelectCheckoutMethod(ctx, intentID string, SelectCheckoutMethodRequest, ...RequestOption) (*SelectedCheckoutMethod, error)`
  — anticipates `POST /v1/checkout_intents/{intent_id}/select`.
- New types: `CheckoutMethodOption`, `CreateCheckoutIntentRequest`,
  `CheckoutIntent` (carries `Methods []CheckoutMethodOption`,
  `PolicyVersion string`, `ExpiresAt time.Time`), `SelectCheckoutMethodRequest`,
  `SelectedCheckoutMethod` (carries the bound `Chain`/`Network`/`Token` plus
  `DepositAddress`/`PaymentURL`/`QRCodeURL`/`DeepLink`).
- New webhook event-type constants `EventCheckoutIntentMethodSelected`
  (`checkout_intent.method_selected`) and `EventCheckoutIntentExpired`
  (`checkout_intent.expired`) — **not emitted by any deployed environment
  today**; the server does not know about checkout intents yet.
- These three methods route through a new unexported `Client.doRequest`
  helper (manual `*http.Request` construction) rather than the generated
  `plaidlyapi.Client`, because the SDK's `spec/openapi.yaml` — and therefore
  `generated/plaidlyapi` — has no knowledge of these endpoints. `doRequest`
  still applies `RequestOption`s (including `WithIdempotencyKey`) and shares
  `Client.doJSON`'s retry/error-decoding path with every other method.
  **Follow-up:** once BDT-528/BDT-529 land, sync `spec/openapi.yaml` from
  `plaidly-api/api/openapi.yaml`, run `make generate`, and replace this
  hand-built path with the generated one (same pattern already used for
  every other endpoint).

### Known gap not addressed in this change

`spec/openapi.yaml` (this repo's bundled copy of the API spec, used by
`make generate`) has not been synced against the live
`plaidly-api/api/openapi.yaml` — notably, it does not declare the
`Idempotency-Key` header parameter that the live API has had on
`POST /v1/payment_sessions` and `POST /v1/payouts` since migration `000040`.
This SDK change does not depend on that sync (the generated client's
`...RequestEditorFn` mechanism already allows injecting arbitrary headers
without the spec declaring them), but the spec is still stale as
documentation/codegen input. Syncing it is lower-risk to do as its own
change, separately from this feature work, since the live spec has diverged
across ~18 more endpoints than the SDK currently wraps at all (see
plaidly-docs#4 / BDT-543's parity table for the full inventory).

## v0.3.0

Session polling/receipt helpers, webhook key-rotation verifiers
(`VerifyWebhookAny`, `VerifyWebhookSignatureAny`), x402 client, agent
integration example. Tagged and published to the Go module proxy
2026-07-11.

## v0.2.0

Initial contract-v1 alignment: wraps merchants, payment sessions
(including demo/simulate), wallets, payouts, payment methods, rates,
sandbox faucets, and the `t=,v1=` webhook verifier. Telegram
bot-integration subpackage.
