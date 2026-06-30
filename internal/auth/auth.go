// Package auth implements the browser login broker. It bridges a CLI client to
// the UIGraph frontend: the CLI opens /auth/login, the broker redirects to the
// frontend authorize page, and the frontend redirects back to /auth/callback
// with a token, which the broker hands back to the CLI's local callback.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/uigraph/mcp/internal/apiclient"
	"github.com/uigraph/mcp/internal/config"
)

const pendingTTL = 5 * time.Minute

type pending struct {
	cliRedirectURI string
	cliState       string
	createdAt      time.Time
}

type Handler struct {
	frontendURL string
	publicURL   string
	client      *apiclient.Client
	mu          sync.Mutex
	store       map[string]pending
}

func New(cfg *config.Config, client *apiclient.Client) *Handler {
	return &Handler{
		frontendURL: cfg.FrontendURL,
		publicURL:   cfg.MCPPublicURL,
		client:      client,
		store:       make(map[string]pending),
	}
}

// Login starts the browser login. The CLI passes its local callback as
// redirect_uri and a CSRF state. The broker stores a pending entry keyed by a
// fresh server-side state and redirects the browser to the frontend.
// GET /auth/login?redirect_uri=<cli-callback>&state=<cli-state>
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	cliRedirectURI := r.URL.Query().Get("redirect_uri")
	cliState := r.URL.Query().Get("state")
	if cliRedirectURI == "" || cliState == "" {
		http.Error(w, "redirect_uri and state are required", http.StatusBadRequest)
		return
	}

	serverState, err := randomState()
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.sweep()
	h.store[serverState] = pending{
		cliRedirectURI: cliRedirectURI,
		cliState:       cliState,
		createdAt:      time.Now(),
	}
	h.mu.Unlock()

	authorize, _ := url.Parse(h.frontendURL + "/authorize")
	q := authorize.Query()
	q.Set("redirect_uri", h.publicURL+"/auth/callback")
	q.Set("state", serverState)
	authorize.RawQuery = q.Encode()

	http.Redirect(w, r, authorize.String(), http.StatusFound)
}

// Callback receives the token from the frontend, looks up the pending CLI
// request by state, and redirects the browser back to the CLI's local callback.
// GET /auth/callback?token=<token>&state=<server-state>
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	serverState := r.URL.Query().Get("state")
	if token == "" || serverState == "" {
		http.Error(w, "token and state are required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.sweep()
	p, ok := h.store[serverState]
	if ok {
		delete(h.store, serverState)
	}
	h.mu.Unlock()

	if !ok {
		http.Error(w, "login session expired or not found", http.StatusNotFound)
		return
	}

	cliCallback, err := url.Parse(p.cliRedirectURI)
	if err != nil {
		http.Error(w, "invalid stored redirect_uri", http.StatusInternalServerError)
		return
	}
	q := cliCallback.Query()
	q.Set("token", token)
	q.Set("state", p.cliState)
	cliCallback.RawQuery = q.Encode()

	http.Redirect(w, r, cliCallback.String(), http.StatusFound)
}

type meResponse struct {
	Me   *apiclient.Me   `json:"me"`
	Orgs []apiclient.Org `json:"orgs"`
}

// Me returns the authenticated principal's profile and orgs by forwarding the
// caller's bearer token to uigraph-api. Used by the CLI's `auth status`.
// GET /auth/me  (Authorization: Bearer <token>)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}

	me, err := h.client.GetMe(r.Context(), token)
	if err != nil {
		http.Error(w, "failed to resolve identity", http.StatusBadGateway)
		return
	}

	// Service accounts have no org memberships endpoint; ignore the error.
	orgs, _ := h.client.GetMyOrgs(r.Context(), token)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meResponse{Me: me, Orgs: orgs})
}

func bearerToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(v, "Bearer "); ok {
		return after
	}
	return ""
}

// sweep removes expired pending entries. Caller must hold h.mu.
func (h *Handler) sweep() {
	now := time.Now()
	for k, v := range h.store {
		if now.Sub(v.createdAt) > pendingTTL {
			delete(h.store, k)
		}
	}
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
