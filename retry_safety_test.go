package cocart

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// queueTransport answers each request from a queue: a status code, or 0 for a timeout.
type queueTransport struct {
	answers []int
	calls   int
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func (q *queueTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	q.calls++
	answer := q.answers[0]
	q.answers = q.answers[1:]
	if answer == 0 {
		return nil, timeoutError{}
	}
	body := `{"order_id":217}`
	if answer >= 400 {
		body = `{"code":"error","message":"Error"}`
	}
	return &http.Response{
		StatusCode: answer,
		Header:     http.Header{"Retry-After": []string{"0"}, "Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func retryClient(answers ...int) (*Client, *queueTransport) {
	transport := &queueTransport{answers: answers}
	return NewClient("https://store.example.com", WithHTTPClient(&http.Client{Transport: transport}), WithMaxRetries(2)), transport
}

func TestRetryGetThatTimesOutIsSentAgain(t *testing.T) {
	c, transport := retryClient(0, 200)
	if _, err := c.Get(context.Background(), "cart", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.calls != 2 {
		t.Errorf("calls = %d, want 2", transport.calls)
	}
}

func TestRetryPostThatTimesOutIsNotSentAgain(t *testing.T) {
	c, transport := retryClient(0, 200)
	_, err := c.Post(context.Background(), "checkout", map[string]any{"payment_method": "stripe"})
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) {
		t.Fatalf("expected a CoCartError, got %v", err)
	}
	if transport.calls != 1 {
		t.Errorf("calls = %d, want 1", transport.calls)
	}
}

func TestRetryPostAnswered503IsNotSentAgain(t *testing.T) {
	c, transport := retryClient(503, 200)
	if _, err := c.Post(context.Background(), "checkout", map[string]any{"payment_method": "stripe"}); err == nil {
		t.Fatal("expected the 503 to reach the caller")
	}
	if transport.calls != 1 {
		t.Errorf("calls = %d, want 1", transport.calls)
	}
}

func TestRetryPostAnswered429IsSentAgain(t *testing.T) {
	c, transport := retryClient(429, 200)
	resp, err := c.Post(context.Background(), "checkout", map[string]any{"payment_method": "stripe"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.calls != 2 {
		t.Errorf("calls = %d, want 2", transport.calls)
	}
	if id := resp.Get("order_id"); id != float64(217) {
		t.Errorf("order_id = %v", id)
	}
}

func TestRetryGetAnswered503IsSentAgain(t *testing.T) {
	c, transport := retryClient(503, 200)
	if _, err := c.Get(context.Background(), "cart", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.calls != 2 {
		t.Errorf("calls = %d, want 2", transport.calls)
	}
}
