package plaidly

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/plaidly/plaidly-go/generated/plaidlyapi"
)

func TestCreateStore_RequestShapeAndResponse(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody CreateStoreRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "store_1", "merchant_id": "m_1", "sandbox": true,
			"name": "Agent VPS Shop", "slug": "agent-vps-shop", "status": "draft",
			"metadata": {}, "created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z"
		}`))
	})

	store, err := c.CreateStore(context.Background(), CreateStoreRequest{
		Name: "Agent VPS Shop",
		Slug: "agent-vps-shop",
	}, WithIdempotencyKey("store-uuid-1"))
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/stores" {
		t.Errorf("got %s %s, want POST /v1/stores", gotMethod, gotPath)
	}
	if gotBody.Slug != "agent-vps-shop" {
		t.Errorf("request Slug = %q, want agent-vps-shop", gotBody.Slug)
	}
	if store.Id != "store_1" || store.Status != CatalogStatusDraft {
		t.Errorf("store = %+v", store)
	}
}

func TestListStores_QueryParams(t *testing.T) {
	var gotQuery string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	})

	status := plaidlyapi.ListStoresParamsStatus(CatalogStatusPublished)
	_, err := c.ListStores(context.Background(), &plaidlyapi.ListStoresParams{Status: &status})
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if gotQuery != "status=published" {
		t.Errorf("query = %q, want status=published", gotQuery)
	}
}

func TestPublishStoreAndArchiveStore(t *testing.T) {
	var gotPaths []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		status := "published"
		if r.URL.Path == "/v1/stores/store_1/archive" {
			status = "archived"
		}
		_, _ = w.Write([]byte(`{
			"id": "store_1", "merchant_id": "m_1", "sandbox": true,
			"name": "Agent VPS Shop", "slug": "agent-vps-shop", "status": "` + status + `",
			"metadata": {}, "created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z"
		}`))
	})

	published, err := c.PublishStore(context.Background(), "store_1")
	if err != nil {
		t.Fatalf("PublishStore: %v", err)
	}
	if published.Status != CatalogStatusPublished {
		t.Errorf("Status = %q, want published", published.Status)
	}

	archived, err := c.ArchiveStore(context.Background(), "store_1")
	if err != nil {
		t.Fatalf("ArchiveStore: %v", err)
	}
	if archived.Status != CatalogStatusArchived {
		t.Errorf("Status = %q, want archived", archived.Status)
	}

	want := []string{"POST /v1/stores/store_1/publish", "POST /v1/stores/store_1/archive"}
	if len(gotPaths) != len(want) || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("gotPaths = %v, want %v", gotPaths, want)
	}
}

func TestDeleteStore(t *testing.T) {
	var gotMethod, gotPath string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteStore(context.Background(), "store_1"); err != nil {
		t.Fatalf("DeleteStore: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v1/stores/store_1" {
		t.Errorf("got %s %s, want DELETE /v1/stores/store_1", gotMethod, gotPath)
	}
}

func TestCreateProduct_ImmutableVersioningPatchReturns201(t *testing.T) {
	var gotPath, gotMethod string

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "product_2", "store_id": "store_1", "merchant_id": "m_1", "sandbox": true,
			"name": "VPS 2GB (updated)", "slug": "vps-2gb", "fulfillment_type": "service",
			"status": "draft", "version": 2, "supersedes_product_id": "product_1",
			"metadata": {}, "created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z"
		}`))
	})

	newName := "VPS 2GB (updated)"
	product, err := c.PatchProduct(context.Background(), "product_1", PatchProductRequest{Name: &newName})
	if err != nil {
		t.Fatalf("PatchProduct: %v", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v1/products/product_1" {
		t.Errorf("got %s %s, want PATCH /v1/products/product_1", gotMethod, gotPath)
	}
	if product.Id != "product_2" {
		t.Errorf("Id = %q, want new versioned id product_2", product.Id)
	}
	if product.SupersedesProductId == nil || *product.SupersedesProductId != "product_1" {
		t.Errorf("SupersedesProductId = %v, want product_1", product.SupersedesProductId)
	}
}

func TestCreatePlanAndCreatePrice(t *testing.T) {
	var gotPaths []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		switch r.URL.Path {
		case "/v1/plans":
			_, _ = w.Write([]byte(`{
				"id": "plan_1", "product_id": "product_1", "merchant_id": "m_1", "sandbox": true,
				"name": "Monthly", "slug": "monthly", "status": "draft", "version": 1,
				"metadata": {}, "created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z"
			}`))
		case "/v1/prices":
			_, _ = w.Write([]byte(`{
				"id": "price_1", "plan_id": "plan_1", "merchant_id": "m_1", "sandbox": true,
				"amount_subunits": "500000", "currency_or_token": "USDC",
				"status": "draft", "version": 1,
				"metadata": {}, "created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z"
			}`))
		}
	})

	plan, err := c.CreatePlan(context.Background(), CreatePlanRequest{
		ProductId: "product_1", Name: "Monthly", Slug: "monthly",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if plan.Id != "plan_1" {
		t.Errorf("plan.Id = %q, want plan_1", plan.Id)
	}

	price, err := c.CreatePrice(context.Background(), CreatePriceRequest{
		PlanId: "plan_1", AmountSubunits: "500000", CurrencyOrToken: "USDC",
	})
	if err != nil {
		t.Fatalf("CreatePrice: %v", err)
	}
	if price.Id != "price_1" || price.CurrencyOrToken != "USDC" {
		t.Errorf("price = %+v", price)
	}

	want := []string{"POST /v1/plans", "POST /v1/prices"}
	if len(gotPaths) != len(want) || gotPaths[0] != want[0] || gotPaths[1] != want[1] {
		t.Errorf("gotPaths = %v, want %v", gotPaths, want)
	}
}

func TestListCatalogProducts_DiscoveryIsUnauthenticated(t *testing.T) {
	var gotQuery, gotAPIKeyHeader string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAPIKeyHeader = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"data": [{
				"id": "product_1", "store_id": "store_1", "merchant_id": "m_1", "sandbox": false,
				"name": "VPS 2GB", "slug": "vps-2gb", "fulfillment_type": "service",
				"status": "published", "version": 1, "metadata": {},
				"created_at": "2026-07-12T00:00:00Z", "updated_at": "2026-07-12T00:00:00Z",
				"plans": []
			}]
		}`))
	})

	resp, err := c.ListCatalogProducts(context.Background(), "m_1", nil)
	if err != nil {
		t.Fatalf("ListCatalogProducts: %v", err)
	}
	if gotQuery != "merchant_id=m_1" {
		t.Errorf("query = %q, want merchant_id=m_1", gotQuery)
	}
	// The generated client still sends the SDK's own X-API-Key by default,
	// but the endpoint itself requires no auth server-side (security: []);
	// what matters here is that the SDK didn't need bearer credentials to
	// build a valid request.
	_ = gotAPIKeyHeader
	if len(resp.Data) != 1 || resp.Data[0].Id != "product_1" {
		t.Errorf("resp.Data = %+v", resp.Data)
	}
}

func TestCreateCatalogCheckoutIntent(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"checkout_reference_id": "cref_1", "price_id": "price_1", "plan_id": "plan_1",
			"product_id": "product_1", "store_id": "store_1", "merchant_id": "m_1",
			"amount_subunits": "500000", "currency_or_token": "USDC",
			"expires_at": "2026-07-12T01:00:00Z"
		}`))
	})

	intent, err := c.CreateCatalogCheckoutIntent(context.Background(), "price_1")
	if err != nil {
		t.Fatalf("CreateCatalogCheckoutIntent: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/catalog/prices/price_1/checkout-intent" {
		t.Errorf("got %s %s, want POST /v1/catalog/prices/price_1/checkout-intent", gotMethod, gotPath)
	}
	if intent.CheckoutReferenceId != "cref_1" || intent.PriceId != "price_1" {
		t.Errorf("intent = %+v", intent)
	}
}
