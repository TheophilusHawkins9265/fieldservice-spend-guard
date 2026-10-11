package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChatDecodesOpenAIResponse(t *testing.T) {
	client := &InfraiClient{key: "test", http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"Next action"}}]}`)), Header: make(http.Header)}, nil
	})}}
	got, err := client.Chat("work order")
	if err != nil || got != "Next action" {
		t.Fatalf("Chat() = %q, %v", got, err)
	}
}

func TestRequestIDDeterministic(t *testing.T) {
	order := WorkOrder{ID: "WO-42", DispatchStatus: "en_route"}
	got := requestID(order)
	if got != "ed5988e8ae9cc0be" {
		t.Fatalf("request id = %q", got)
	}
}

func TestSummarizeWorkOrderDecisionInput(t *testing.T) {
	order := WorkOrder{ID: "WO-7", DispatchStatus: "arrived", PhotoURLs: []string{"a", "b"}, FollowUp: "replace valve"}
	got := summarizeWorkOrder(order)
	want := "Work order WO-7. Dispatch status: arrived. Photos: 2. Technician follow-up: replace valve. Return a concise next-action summary."
	if got != want {
		t.Fatalf("summary = %q", got)
	}
}
