package plaidly

import (
	"context"
	"net/http"

	"github.com/plaidly/plaidly-go/generated/plaidlyapi"
)

// Catalog status values shared by stores, products, plans, and prices.
const (
	CatalogStatusDraft     = "draft"
	CatalogStatusPublished = "published"
	CatalogStatusArchived  = "archived"
)

// -- Stores -----------------------------------------------------------------

// CreateStore creates a store (a merchant storefront/business profile).
//
// POST /v1/stores
func (c *Client) CreateStore(ctx context.Context, req CreateStoreRequest, opts ...RequestOption) (*CatalogStore, error) {
	ro := buildRequestOptions(opts)
	var out CatalogStore
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreateStore(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListStores lists the authenticated merchant's stores. params may be nil.
//
// GET /v1/stores
func (c *Client) ListStores(ctx context.Context, params *plaidlyapi.ListStoresParams) (*StoreListResponse, error) {
	var out StoreListResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListStores(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetStore fetches a store by ID.
//
// GET /v1/stores/{store_id}
func (c *Client) GetStore(ctx context.Context, storeID string) (*CatalogStore, error) {
	var out CatalogStore
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetStore(ctx, storeID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchStore updates a store in place. Stores mutate in place regardless of
// status (no versioning).
//
// PATCH /v1/stores/{store_id}
func (c *Client) PatchStore(ctx context.Context, storeID string, req PatchStoreRequest) (*CatalogStore, error) {
	var out CatalogStore
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PatchStore(ctx, storeID, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteStore hard-deletes a store. Only legal for a draft, never-published
// store with no archived/published children.
//
// DELETE /v1/stores/{store_id}
func (c *Client) DeleteStore(ctx context.Context, storeID string) error {
	return c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.DeleteStore(ctx, storeID)
	}, nil)
}

// PublishStore publishes a store so it becomes visible to buyer-agent
// discovery.
//
// POST /v1/stores/{store_id}/publish
func (c *Client) PublishStore(ctx context.Context, storeID string) (*CatalogStore, error) {
	var out CatalogStore
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PublishStore(ctx, storeID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ArchiveStore archives a store. Archiving an already-archived store is a
// no-op.
//
// POST /v1/stores/{store_id}/archive
func (c *Client) ArchiveStore(ctx context.Context, storeID string) (*CatalogStore, error) {
	var out CatalogStore
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ArchiveStore(ctx, storeID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// -- Products -----------------------------------------------------------------

// CreateProduct creates a product within a store.
//
// POST /v1/products
func (c *Client) CreateProduct(ctx context.Context, req CreateProductRequest, opts ...RequestOption) (*CatalogProduct, error) {
	ro := buildRequestOptions(opts)
	var out CatalogProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreateProduct(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListProducts lists the authenticated merchant's products. params may be nil.
//
// GET /v1/products
func (c *Client) ListProducts(ctx context.Context, params *plaidlyapi.ListProductsParams) (*ProductListResponse, error) {
	var out ProductListResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListProducts(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProduct fetches a product by ID.
//
// GET /v1/products/{product_id}
func (c *Client) GetProduct(ctx context.Context, productID string) (*CatalogProduct, error) {
	var out CatalogProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetProduct(ctx, productID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchProduct updates a product. If the target is a mutable draft (never
// published, no checkout reference chain) it mutates in place. If the
// target is immutable (published, or checkout-referenced via one of its
// plans/prices) this creates a new version instead: the returned resource's
// Id differs from the passed productID, and SupersedesProductId points at
// the old id.
//
// PATCH /v1/products/{product_id}
func (c *Client) PatchProduct(ctx context.Context, productID string, req PatchProductRequest) (*CatalogProduct, error) {
	var out CatalogProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PatchProduct(ctx, productID, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteProduct hard-deletes a product. Only legal for a draft,
// never-published product with no checkout reference chain.
//
// DELETE /v1/products/{product_id}
func (c *Client) DeleteProduct(ctx context.Context, productID string) error {
	return c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.DeleteProduct(ctx, productID)
	}, nil)
}

// PublishProduct publishes a draft product. Only legal from draft.
//
// POST /v1/products/{product_id}/publish
func (c *Client) PublishProduct(ctx context.Context, productID string) (*CatalogProduct, error) {
	var out CatalogProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PublishProduct(ctx, productID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ArchiveProduct archives a product (legal from draft or published).
// Archiving an already-archived product is a no-op.
//
// POST /v1/products/{product_id}/archive
func (c *Client) ArchiveProduct(ctx context.Context, productID string) (*CatalogProduct, error) {
	var out CatalogProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ArchiveProduct(ctx, productID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// -- Plans -----------------------------------------------------------------

// CreatePlan creates a plan (a purchasable variant of a product).
//
// POST /v1/plans
func (c *Client) CreatePlan(ctx context.Context, req CreatePlanRequest, opts ...RequestOption) (*CatalogPlan, error) {
	ro := buildRequestOptions(opts)
	var out CatalogPlan
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreatePlan(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPlans lists the authenticated merchant's plans. params may be nil.
//
// GET /v1/plans
func (c *Client) ListPlans(ctx context.Context, params *plaidlyapi.ListPlansParams) (*PlanListResponse, error) {
	var out PlanListResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListPlans(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPlan fetches a plan by ID.
//
// GET /v1/plans/{plan_id}
func (c *Client) GetPlan(ctx context.Context, planID string) (*CatalogPlan, error) {
	var out CatalogPlan
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPlan(ctx, planID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchPlan updates a plan. If the target is a mutable draft it mutates in
// place. If the target is immutable (published, or checkout-referenced via
// one of its prices) this creates a new version instead: the returned
// resource's Id differs from the passed planID, and SupersedesPlanId points
// at the old id.
//
// PATCH /v1/plans/{plan_id}
func (c *Client) PatchPlan(ctx context.Context, planID string, req PatchPlanRequest) (*CatalogPlan, error) {
	var out CatalogPlan
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PatchPlan(ctx, planID, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePlan hard-deletes a plan. Only legal for a draft, never-published
// plan with no checkout reference chain.
//
// DELETE /v1/plans/{plan_id}
func (c *Client) DeletePlan(ctx context.Context, planID string) error {
	return c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.DeletePlan(ctx, planID)
	}, nil)
}

// PublishPlan publishes a draft plan. Only legal from draft.
//
// POST /v1/plans/{plan_id}/publish
func (c *Client) PublishPlan(ctx context.Context, planID string) (*CatalogPlan, error) {
	var out CatalogPlan
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PublishPlan(ctx, planID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ArchivePlan archives a plan (legal from draft or published). Archiving an
// already-archived plan is a no-op.
//
// POST /v1/plans/{plan_id}/archive
func (c *Client) ArchivePlan(ctx context.Context, planID string) (*CatalogPlan, error) {
	var out CatalogPlan
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ArchivePlan(ctx, planID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// -- Prices -----------------------------------------------------------------

// CreatePrice creates a price for a plan.
//
// POST /v1/prices
func (c *Client) CreatePrice(ctx context.Context, req CreatePriceRequest, opts ...RequestOption) (*CatalogPrice, error) {
	ro := buildRequestOptions(opts)
	var out CatalogPrice
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreatePrice(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPrices lists the authenticated merchant's prices. params may be nil.
//
// GET /v1/prices
func (c *Client) ListPrices(ctx context.Context, params *plaidlyapi.ListPricesParams) (*PriceListResponse, error) {
	var out PriceListResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListPrices(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPrice fetches a price by ID.
//
// GET /v1/prices/{price_id}
func (c *Client) GetPrice(ctx context.Context, priceID string) (*CatalogPrice, error) {
	var out CatalogPrice
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPrice(ctx, priceID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// PatchPrice updates a price. If the target is a mutable draft it mutates
// in place. If the target is immutable (published, or checkout-referenced)
// this creates a new version instead: the returned resource's Id differs
// from the passed priceID, and SupersedesPriceId points at the old id.
//
// PATCH /v1/prices/{price_id}
func (c *Client) PatchPrice(ctx context.Context, priceID string, req PatchPriceRequest) (*CatalogPrice, error) {
	var out CatalogPrice
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PatchPrice(ctx, priceID, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePrice hard-deletes a price. Only legal for a draft, never-published
// price with no checkout reference chain.
//
// DELETE /v1/prices/{price_id}
func (c *Client) DeletePrice(ctx context.Context, priceID string) error {
	return c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.DeletePrice(ctx, priceID)
	}, nil)
}

// PublishPrice publishes a draft price. Only legal from draft.
//
// POST /v1/prices/{price_id}/publish
func (c *Client) PublishPrice(ctx context.Context, priceID string) (*CatalogPrice, error) {
	var out CatalogPrice
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.PublishPrice(ctx, priceID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ArchivePrice archives a price (legal from draft or published). Archiving
// an already-archived price is a no-op.
//
// POST /v1/prices/{price_id}/archive
func (c *Client) ArchivePrice(ctx context.Context, priceID string) (*CatalogPrice, error) {
	var out CatalogPrice
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ArchivePrice(ctx, priceID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// -- Buyer-agent discovery ---------------------------------------------------

// ListCatalogProducts is the public, unauthenticated buyer-agent discovery
// listing of a merchant's published products, with each product's published
// plans (and their published prices) inlined so a buyer agent needs only one
// round trip. merchantID is required; params (if non-nil) additionally
// filters/paginates and merchantID is copied into params.MerchantId.
//
// GET /v1/catalog/products
func (c *Client) ListCatalogProducts(ctx context.Context, merchantID string, params *plaidlyapi.ListCatalogProductsParams) (*CatalogProductListResponse, error) {
	if params == nil {
		params = &plaidlyapi.ListCatalogProductsParams{}
	}
	params.MerchantId = merchantID
	var out CatalogProductListResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListCatalogProducts(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCatalogProduct is the public, unauthenticated buyer-agent discovery
// detail for a single published product, with its published plans (and
// their published prices) inlined.
//
// GET /v1/catalog/products/{product_id}
func (c *Client) GetCatalogProduct(ctx context.Context, productID string) (*CatalogDiscoveryProduct, error) {
	var out CatalogDiscoveryProduct
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetCatalogProduct(ctx, productID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCatalogCheckoutIntent resolves a published price for checkout. It
// validates the price and its parent plan/product/store are all published
// and not archived, records an immutable checkout snapshot (which flips the
// price/plan/product into checkout-referenced immutability), and returns the
// fields a buyer agent needs to build a POST /v1/payment_sessions request
// (placing CheckoutReferenceId into that request's metadata).
//
// POST /v1/catalog/prices/{price_id}/checkout-intent
func (c *Client) CreateCatalogCheckoutIntent(ctx context.Context, priceID string) (*CatalogCheckoutIntent, error) {
	var out CatalogCheckoutIntent
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreateCatalogCheckoutIntent(ctx, priceID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
