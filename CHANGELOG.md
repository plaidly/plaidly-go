# Changelog

All notable changes to this SDK are recorded here. This file did not exist
before BDT-530; entries below cover all `Unreleased` work since `v0.3.0`.

## Unreleased

Not tagged yet — see `RELEASE.md` for how tagging works. Covers BDT-530
(idempotency-key contract, typed-error fix) and the subsequent spec-sync
change (BDT-542, BDT-223, BDT-529, BDT-528).

### Merchant credential + webhook management (BDT-544)

The generated `plaidlyapi` client already wrapped
`GET /v1/me/credentials`, `POST /v1/me/credentials/rotate`,
`POST /v1/me/credentials/revoke`, `PUT /v1/me/webhook`, and
`POST /v1/me/webhook/test`, but the hand-written high-level `Client` never
exposed them. Added, in `credentials.go`:

- `Client.GetMerchantCredentialStatus` — masked previews + rotation timestamps.
- `Client.RotateMerchantCredential(ctx, credentialType, opts...)` — returns the
  full `Merchant`, whose `ApiKey`/`WebhookSecret` fields carry the fresh secret
  exactly once (response-only). Accepts `RequestOption` for idempotency keys,
  same convention as the catalog mutators.
- `Client.RevokeMerchantCredential(ctx, credentialType, opts...)`.
- `Client.UpdateMerchantWebhook(ctx, url, opts...)`.
- `Client.TestMerchantWebhookDelivery(ctx)`.

New exported types: `MerchantCredentialStatus`, `MerchantCredentialOperationRequest`,
`MerchantCredentialType` (with `CredentialTypeAPIKey` / `CredentialTypeWebhookSecret`
constants), `UpdateMerchantWebhookRequest`, `WebhookTestDeliveryResult`.

### Spec sync (2026-07-12)

`spec/openapi.yaml` has been re-synced from the live, deployed
`plaidly-api/api/openapi.yaml`; it had drifted since the last sync (see the
now-resolved "Known gap" note that used to live here). `generated/plaidlyapi`
was regenerated with `make generate` (`oapi-codegen` `v2.4.1`, pinned).
Sync required adding `output-options.response-type-suffix: HTTPResponse` to
`oapi-cfg.yaml`: the new spec's `RequestEmailVerificationResponse` /
`ConfirmEmailVerificationResponse` / `RequestRegistrationProofOfWorkResponse`
/ `FundSepoliaFaucetResponse` schema names collided with oapi-codegen's
auto-generated per-operation `<OperationId>Response` wrapper struct names
(the collision produces duplicate-declaration compile errors without the
suffix override).

The regen also surfaced two small breaking-shape changes unrelated to the
four features below, now fixed everywhere in this SDK and its tests/examples:
- `CreatePaymentSessionRequest.PaymentMethod` is now `*PaymentMethod`
  (pointer, since it's mutually exclusive with the new `PaymentMethods`
  field — see below) instead of a bare value.
- `CreatePaymentSession` and `RequestPayout` now pass an explicit `nil`
  `*Params` argument to the regenerated raw client (both operations gained a
  formal `Idempotency-Key` header parameter in the spec; the SDK still sets
  that header via its own `RequestOption`/`RequestEditorFn` mechanism, not
  through the generated `Params` struct, so behavior is unchanged).

### Real, deployed-endpoint features

These are wired against endpoints that are live on `api.plaidly.io` today.

#### Multi-method checkout intents (BDT-528) — now deployed

Corrects this file's previous "Experimental" entry: BDT-528 shipped to
production, but **not** as the speculative `/v1/checkout_intents*` resource
this SDK had anticipated. It landed as a `paymentMethods` (plural) extension
of the existing payment-session endpoints instead:
- `POST /v1/payment_sessions` now accepts `paymentMethods` (an array of 1-20
  candidate `PaymentMethod`s) as an alternative to the singular
  `paymentMethod`. The response type is `PaymentSessionOrIntent`, which is
  a `PaymentSession`-shaped snapshot with `Status ==
  "awaiting_method_selection"` and `CandidatePaymentMethods` populated until
  a method is selected.
- `GET /v1/payment_sessions/{session_id}` returns the same
  `PaymentSessionOrIntent` shape.
- `POST /v1/payment_sessions/{session_id}/select_method` binds one method
  from the candidate set, materializing an ordinary `PaymentSession` with a
  deposit address.

`CreateCheckoutIntent`, `GetCheckoutIntent`, and `SelectCheckoutMethod` keep
their names but are rewritten against this real contract: they now call
`c.raw.CreatePaymentSession` / `c.raw.GetPaymentSession` /
`c.raw.SelectPaymentMethod` (the generated client) like every other wrapper,
instead of the unexported `Client.doRequest`/manual-`*http.Request` path used
previously (that path existed only because the old spec had no knowledge of
these endpoints — no longer true). The speculative
`EventCheckoutIntentMethodSelected` / `EventCheckoutIntentExpired` webhook
event constants and the standalone `CheckoutMethodOption` /
`CreateCheckoutIntentRequest` / `CheckoutIntent` /
`SelectCheckoutMethodRequest` / `SelectedCheckoutMethod` types are removed —
no such webhook events exist, and the real endpoints reuse
`CreatePaymentSessionRequest` / `PaymentSessionOrIntent` /
`PaymentMethod` / `PaymentSession` directly. New constant:
`CheckoutIntentStatusAwaitingSelection`.

#### Public/sandbox merchant registration anti-abuse flow (BDT-542)

New file `registration.go`:
- `RequestEmailVerification`, `ConfirmEmailVerification`,
  `RequestRegistrationProofOfWork` wrap the three new
  `POST /v1/merchants/email-verification`,
  `POST /v1/merchants/email-verification/confirm`, and
  `POST /v1/merchants/registration-proof-of-work` endpoints.
- `RegisterMerchantRequest` gains optional `RegistrationIntentId`,
  `ProofOfWorkChallengeId`, `ProofOfWorkNonce` fields for public/sandbox
  registration (bearer-authenticated live merchant creation ignores them).
- `SolveProofOfWork(challenge string, difficultyBits int) string` — a local,
  offline `sha256-leading-zero-bits-v1` solver (find a nonce such that
  `sha256(challenge + ":" + nonce)` has at least `difficultyBits` leading
  zero bits) so callers don't need to reimplement bit-counting. Mirrors
  `plaidly-api/internal/pow`'s `leadingZeroBits` exactly; verified in
  `registration_test.go` against an independently inlined copy of that same
  reference logic, plus an end-to-end test that feeds a solved nonce into
  `RegisterMerchant`.

#### Generic commerce catalog (BDT-223)

New file `catalog.go`: full CRUD (`Create`/`List`/`Get`/`Patch`/`Delete`)
plus `Publish`/`Archive` transitions for stores, products, plans, and prices
(`CreateStore`, `ListStores`, `GetStore`, `PatchStore`, `DeleteStore`,
`PublishStore`, `ArchiveStore`, and the equivalent 7 methods each for
`*Product`, `*Plan`, `*Price`), matching the immutable/versioned semantics of
the live API — `PatchProduct`/`PatchPlan`/`PatchPrice` return a *new*
resource version (different `Id`, `SupersedesProductId`/`SupersedesPlanId`/
`SupersedesPriceId` set) once the target is published or
checkout-referenced, rather than mutating in place; stores always mutate in
place. Also wraps the public, unauthenticated buyer-agent discovery
endpoints — `ListCatalogProducts`, `GetCatalogProduct` — and
`CreateCatalogCheckoutIntent`, which resolves a published price into the
fields needed to build a `CreatePaymentSession` call. New constants
`CatalogStatusDraft` / `CatalogStatusPublished` / `CatalogStatusArchived`.

#### Merchant payment-method policy (BDT-529)

`merchants.go` gains `GetPaymentMethodPolicy` / `UpdatePaymentMethodPolicy`
(`GET`/`PUT /v1/me/payment_method_policy`), returning/accepting
`PaymentMethodPolicyState` (declared policy version plus live-computed
per-identity eligibility) and `UpdatePaymentMethodPolicyRequest`.

#### Carried over from BDT-530

- `WithIdempotencyKey(key string) RequestOption`, plumbed through
  `CreatePaymentSession` and `RequestPayout` (both take
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
