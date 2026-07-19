package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uigraph/mcp/internal/apiclient"
)

func TestAuthenticatedOrgServiceAccount(t *testing.T) {
	client := apiclient.New("", "")
	me := &apiclient.Me{Kind: "service_account", OrgID: "service-org"}

	orgID, err := authenticatedOrg(context.Background(), client, "token", me, "")
	if err != nil {
		t.Fatalf("authenticatedOrg() error = %v", err)
	}
	if orgID != "service-org" {
		t.Fatalf("authenticatedOrg() = %q, want %q", orgID, "service-org")
	}
}

func TestAuthenticatedOrgServiceAccountRejectsRequestedOrg(t *testing.T) {
	client := apiclient.New("", "")
	me := &apiclient.Me{Kind: "service_account", OrgID: "service-org"}

	_, err := authenticatedOrg(context.Background(), client, "token", me, "other-org")
	if err == nil {
		t.Fatal("authenticatedOrg() error = nil, want error")
	}
}

func TestAuthenticatedOrgUser(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/orgs" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/auth/orgs")
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer token")
		}
		_, _ = w.Write([]byte(`{"orgs":[{"id":"selected-org","name":"Selected","role":"member"}]}`))
	}))
	defer api.Close()

	client := apiclient.New(api.URL, "")
	ctx := apiclient.WithScheme(context.Background(), apiclient.SchemeBearer)
	me := &apiclient.Me{Kind: "user"}

	orgID, err := authenticatedOrg(ctx, client, "token", me, "selected-org")
	if err != nil {
		t.Fatalf("authenticatedOrg() error = %v", err)
	}
	if orgID != "selected-org" {
		t.Fatalf("authenticatedOrg() = %q, want %q", orgID, "selected-org")
	}
}

func TestAuthenticatedOrgUserRejectsUnavailableOrg(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgs":[{"id":"available-org","name":"Available","role":"member"}]}`))
	}))
	defer api.Close()

	client := apiclient.New(api.URL, "")
	ctx := apiclient.WithScheme(context.Background(), apiclient.SchemeBearer)
	me := &apiclient.Me{Kind: "user"}

	_, err := authenticatedOrg(ctx, client, "token", me, "selected-org")
	if err == nil {
		t.Fatal("authenticatedOrg() error = nil, want error")
	}
}
