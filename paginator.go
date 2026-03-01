package cocart

import (
	"context"
	"iter"
)

// Paginator iterates through paginated API results.
type Paginator struct {
	fetchPage func(ctx context.Context, page int) (*Response, error)
	startPage int
}

// NewPaginator creates a new Paginator.
func NewPaginator(fetchPage func(ctx context.Context, page int) (*Response, error), startPage ...int) *Paginator {
	sp := 1
	if len(startPage) > 0 && startPage[0] > 0 {
		sp = startPage[0]
	}
	return &Paginator{
		fetchPage: fetchPage,
		startPage: sp,
	}
}

// All returns an iterator that yields each page's Response.
// Compatible with Go 1.23+ range-over-func.
//
// Usage:
//
//	for resp, err := range paginator.All(ctx) {
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    products, _ := cocart.Unmarshal[[]cocart.Product](resp)
//	}
func (p *Paginator) All(ctx context.Context) iter.Seq2[*Response, error] {
	return func(yield func(*Response, error) bool) {
		page := p.startPage
		for {
			resp, err := p.fetchPage(ctx, page)
			if !yield(resp, err) {
				return
			}
			if err != nil {
				return
			}

			totalPages := resp.GetTotalPages()
			if totalPages <= 0 || page >= totalPages {
				return
			}

			page++
		}
	}
}

// Collect fetches all pages and returns them as a slice.
func (p *Paginator) Collect(ctx context.Context) ([]*Response, error) {
	var results []*Response
	for resp, err := range p.All(ctx) {
		if err != nil {
			return results, err
		}
		results = append(results, resp)
	}
	return results, nil
}
