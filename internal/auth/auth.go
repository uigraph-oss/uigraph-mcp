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

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token, scheme := credential(r)
	if token == "" {
		http.Error(w, "missing credentials", http.StatusUnauthorized)
		return
	}

	ctx := apiclient.WithScheme(r.Context(), scheme)
	me, err := h.client.GetMe(ctx, token)
	if err != nil {
		http.Error(w, "failed to resolve identity", http.StatusBadGateway)
		return
	}

	orgs, _ := h.client.GetMyOrgs(ctx, token)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meResponse{Me: me, Orgs: orgs})
}

func credential(r *http.Request) (string, apiclient.Scheme) {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key, apiclient.SchemeAPIKey
	}
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return after, apiclient.SchemeBearer
	}
	return "", ""
}

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
