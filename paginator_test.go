package cocart

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestPaginatorAll(t *testing.T) {
	totalPages := 3
	pagesCalled := 0

	paginator := NewPaginator(func(ctx context.Context, page int) (*Response, error) {
		pagesCalled++
		headers := http.Header{}
		headers.Set("X-WP-Total", "30")
		headers.Set("X-WP-TotalPages", fmt.Sprintf("%d", totalPages))
		return &Response{
			StatusCode: 200,
			Headers:    headers,
			Body:       []byte(fmt.Sprintf(`[{"page":%d}]`, page)),
		}, nil
	})

	count := 0
	for resp, err := range paginator.All(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		count++
	}

	if count != totalPages {
		t.Errorf("iterated %d pages, want %d", count, totalPages)
	}
}

func TestPaginatorCollect(t *testing.T) {
	paginator := NewPaginator(func(ctx context.Context, page int) (*Response, error) {
		headers := http.Header{}
		headers.Set("X-WP-TotalPages", "2")
		return &Response{
			StatusCode: 200,
			Headers:    headers,
			Body:       []byte(`[]`),
		}, nil
	})

	pages, err := paginator.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if len(pages) != 2 {
		t.Errorf("Collect returned %d pages, want 2", len(pages))
	}
}

func TestPaginatorSinglePage(t *testing.T) {
	paginator := NewPaginator(func(ctx context.Context, page int) (*Response, error) {
		headers := http.Header{}
		headers.Set("X-WP-TotalPages", "1")
		return &Response{
			StatusCode: 200,
			Headers:    headers,
			Body:       []byte(`[]`),
		}, nil
	})

	pages, err := paginator.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}
}

func TestPaginatorError(t *testing.T) {
	paginator := NewPaginator(func(ctx context.Context, page int) (*Response, error) {
		if page == 2 {
			return nil, NewCoCartError("test error", 500, "test")
		}
		headers := http.Header{}
		headers.Set("X-WP-TotalPages", "3")
		return &Response{
			StatusCode: 200,
			Headers:    headers,
			Body:       []byte(`[]`),
		}, nil
	})

	pages, err := paginator.Collect(context.Background())
	if err == nil {
		t.Error("expected error on page 2")
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 successful page before error, got %d", len(pages))
	}
}
