package cocart

import (
	"context"
	"fmt"
)

// ProductsEndpoint handles all product-related API operations.
type ProductsEndpoint struct {
	endpoint
}

// All returns all products.
func (e *ProductsEndpoint) All(ctx context.Context, params ...*ProductListParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "", p)
}

// Find returns a single product by ID.
func (e *ProductsEndpoint) Find(ctx context.Context, productID int, params ...ProductParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, fmt.Sprintf("%d", productID), p)
}

// FindBySlug returns a single product by slug. Requires CoCart Starter.
func (e *ProductsEndpoint) FindBySlug(ctx context.Context, slug string, params ...ProductParams) (*Response, error) {
	if err := e.client.RequiresBasic("products()->findBySlug"); err != nil {
		return nil, err
	}
	var p map[string]string
	if len(params) > 0 {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, slug, p)
}

// Search searches for products by term.
func (e *ProductsEndpoint) Search(ctx context.Context, term string, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Search = term
	return e.All(ctx, &plp)
}

// ByCategory returns products in a category.
func (e *ProductsEndpoint) ByCategory(ctx context.Context, categorySlug string, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Category = categorySlug
	return e.All(ctx, &plp)
}

// ByTag returns products with a tag.
func (e *ProductsEndpoint) ByTag(ctx context.Context, tagSlug string, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Tag = tagSlug
	return e.All(ctx, &plp)
}

// Featured returns featured products.
func (e *ProductsEndpoint) Featured(ctx context.Context, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Featured = true
	return e.All(ctx, &plp)
}

// OnSale returns products on sale.
func (e *ProductsEndpoint) OnSale(ctx context.Context, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.OnSale = true
	return e.All(ctx, &plp)
}

// ByPriceRange returns products within a price range.
// Pass an empty string for minPrice or maxPrice to leave that bound open.
func (e *ProductsEndpoint) ByPriceRange(ctx context.Context, minPrice, maxPrice string, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	if minPrice != "" {
		plp.MinPrice = minPrice
	}
	if maxPrice != "" {
		plp.MaxPrice = maxPrice
	}
	return e.All(ctx, &plp)
}

// SortBy returns products sorted by a field.
func (e *ProductsEndpoint) SortBy(ctx context.Context, field ProductOrderBy, order SortOrder, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.OrderBy = field
	plp.Order = order
	return e.All(ctx, &plp)
}

// ByStockStatus returns products filtered by stock status.
func (e *ProductsEndpoint) ByStockStatus(ctx context.Context, status StockStatus, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.StockStatus = status
	return e.All(ctx, &plp)
}

// Paginate returns a specific page of products.
func (e *ProductsEndpoint) Paginate(ctx context.Context, page, perPage int, params ...*ProductListParams) (*Response, error) {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Page = page
	plp.PerPage = perPage
	return e.All(ctx, &plp)
}

// AllPaginated returns a Paginator that iterates through all pages of products.
func (e *ProductsEndpoint) AllPaginated(params ...*ProductListParams) *Paginator {
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	return NewPaginator(func(ctx context.Context, page int) (*Response, error) {
		plp.Page = page
		return e.All(ctx, &plp)
	})
}

// --- Variations ---

// Variations returns product variations.
func (e *ProductsEndpoint) Variations(ctx context.Context, productID int, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, fmt.Sprintf("%d/variations", productID), p)
}

// Variation returns a specific variation. Requires CoCart Starter.
func (e *ProductsEndpoint) Variation(ctx context.Context, productID, variationID int, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->variation"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, fmt.Sprintf("%d/variations/%d", productID, variationID), params...)
}

// --- Categories ---

// Categories returns product categories.
func (e *ProductsEndpoint) Categories(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "categories", p)
}

// Category returns a single category. Requires CoCart Starter.
func (e *ProductsEndpoint) Category(ctx context.Context, categoryID int, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->category"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, fmt.Sprintf("categories/%d", categoryID), params...)
}

// --- Tags ---

// Tags returns product tags.
func (e *ProductsEndpoint) Tags(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "tags", p)
}

// Tag returns a single tag. Requires CoCart Starter.
func (e *ProductsEndpoint) Tag(ctx context.Context, tagID int, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->tag"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, fmt.Sprintf("tags/%d", tagID), params...)
}

// --- Attributes ---

// Attributes returns product attributes.
func (e *ProductsEndpoint) Attributes(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "attributes", p)
}

// Attribute returns a single attribute.
func (e *ProductsEndpoint) Attribute(ctx context.Context, attributeID int, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, fmt.Sprintf("attributes/%d", attributeID), params...)
}

// AttributeTerms returns terms for an attribute.
func (e *ProductsEndpoint) AttributeTerms(ctx context.Context, attributeID int, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, fmt.Sprintf("attributes/%d/terms", attributeID), p)
}

// AttributeTerm returns a specific term for an attribute.
func (e *ProductsEndpoint) AttributeTerm(ctx context.Context, attributeID, termID int, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, fmt.Sprintf("attributes/%d/terms/%d", attributeID, termID), params...)
}

// AttributeBySlug returns an attribute by slug. Requires CoCart Starter.
func (e *ProductsEndpoint) AttributeBySlug(ctx context.Context, slug string, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->attributeBySlug"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, "attributes/"+slug, params...)
}

// AttributeTermsBySlug returns terms for an attribute by slug. Requires CoCart Starter.
func (e *ProductsEndpoint) AttributeTermsBySlug(ctx context.Context, slug string, params ...*PaginationParams) (*Response, error) {
	if err := e.client.RequiresBasic("products()->attributeTermsBySlug"); err != nil {
		return nil, err
	}
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "attributes/"+slug+"/terms", p)
}

// AttributeTermBySlug returns a term by slug for an attribute by slug. Requires CoCart Starter.
func (e *ProductsEndpoint) AttributeTermBySlug(ctx context.Context, attrSlug, termSlug string, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->attributeTermBySlug"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, "attributes/"+attrSlug+"/terms/"+termSlug, params...)
}

// --- Brands ---

// Brands returns product brands. Requires CoCart Starter.
func (e *ProductsEndpoint) Brands(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	if err := e.client.RequiresBasic("products()->brands"); err != nil {
		return nil, err
	}
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "brands", p)
}

// Brand returns a single brand. Requires CoCart Starter.
func (e *ProductsEndpoint) Brand(ctx context.Context, brandID int, params ...map[string]string) (*Response, error) {
	if err := e.client.RequiresBasic("products()->brand"); err != nil {
		return nil, err
	}
	return e.doGet(ctx, fmt.Sprintf("brands/%d", brandID), params...)
}

// ByBrand returns products by brand. Requires CoCart Starter.
func (e *ProductsEndpoint) ByBrand(ctx context.Context, brandSlug string, params ...*ProductListParams) (*Response, error) {
	if err := e.client.RequiresBasic("products()->byBrand"); err != nil {
		return nil, err
	}
	var plp ProductListParams
	if len(params) > 0 && params[0] != nil {
		plp = *params[0]
	}
	plp.Brand = brandSlug
	return e.All(ctx, &plp)
}

// --- Reviews ---

// Reviews returns product reviews.
func (e *ProductsEndpoint) Reviews(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "reviews", p)
}

// ProductReviews returns reviews for a specific product.
func (e *ProductsEndpoint) ProductReviews(ctx context.Context, productID int, params ...*PaginationParams) (*Response, error) {
	p := make(map[string]string)
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	p["product"] = fmt.Sprintf("%d", productID)
	return e.doGet(ctx, "reviews", p)
}

// MyReviews returns the authenticated user's product reviews. Requires CoCart Starter.
func (e *ProductsEndpoint) MyReviews(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	if err := e.client.RequiresBasic("products()->myReviews"); err != nil {
		return nil, err
	}
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "reviews/mine", p)
}

// --- SEO ---

// SEO returns SEO data for a product by ID.
func (e *ProductsEndpoint) SEO(ctx context.Context, productID int) (*Response, error) {
	return e.doGet(ctx, fmt.Sprintf("%d/seo", productID))
}

// SEOBySlug returns SEO data for a product by slug.
func (e *ProductsEndpoint) SEOBySlug(ctx context.Context, slug string) (*Response, error) {
	return e.doGet(ctx, slug+"/seo")
}
