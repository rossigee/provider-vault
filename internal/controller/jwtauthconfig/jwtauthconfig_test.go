package jwtauthconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/jwtauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestJWTAuthConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.JWTAuthConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.JWTAuthConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-jwt",
			Namespace: "default",
		},
		Spec: v1beta1.JWTAuthConfigSpec{
			ForProvider: v1beta1.JWTAuthConfigParameters{
				Backend:          "jwt-auth",
				OIDCDiscoveryURL: "https://accounts.google.com",
				BoundIssuer:      "https://accounts.google.com",
				DefaultRole:      "default",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_ConfigExists(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"oidc_discovery_url":"https://accounts.google.com","bound_issuer":"https://accounts.google.com","default_role":"default"}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Vault returns 404")
	}
}

func TestObserve_Drift(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"oidc_discovery_url":"https://old.example.com","bound_issuer":"https://old.example.com","default_role":"old"}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when config drifts")
	}
}

func TestCreate_OIDCConfig(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["oidc_discovery_url"] != "https://accounts.google.com" {
			t.Errorf("oidc_discovery_url = %v", body["oidc_discovery_url"])
		}
		if body["bound_issuer"] != "https://accounts.google.com" {
			t.Errorf("bound_issuer = %v", body["bound_issuer"])
		}
		if body["default_role"] != "default" {
			t.Errorf("default_role = %v", body["default_role"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_JWTConfig(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["oidc_discovery_url"] != nil {
			t.Errorf("oidc_discovery_url = %v, expected nil", body["oidc_discovery_url"])
		}
		if body["jwt_validation_pubkeys"] == nil {
			t.Error("expected jwt_validation_pubkeys")
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.OIDCDiscoveryURL = ""
	cr.Spec.ForProvider.BoundIssuer = ""
	cr.Spec.ForProvider.DefaultRole = ""
	cr.Spec.ForProvider.JWTValidationPubKeys = []string{"-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestUpdate_Configures(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestUpdate_Drift(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.BoundIssuer = "https://new-issuer.example.com"

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete_NoOp(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestObserve_FullOIDCConfig(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/jwt-auth/config" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":{"oidc_discovery_url":"https://accounts.google.com","oidc_client_id":"my-client","oidc_client_secret":"","bound_issuer":"https://accounts.google.com","default_role":"default"}}`)
		} else {
			t.Errorf("unexpected path = %s", r.URL.Path)
		}
	})
	defer srv.Close()

	cr.Spec.ForProvider.OIDCClientID = "my-client"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true")
	}
}

func TestConfigureJWTAuth_FullConfig(t *testing.T) {
	e, srv, cr := newTestJWTAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/config") {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["oidc_client_secret"] != "my-secret" {
			t.Errorf("oidc_client_secret = %v", body["oidc_client_secret"])
		}
		if body["max_ttl"] != "24h" {
			t.Errorf("max_ttl = %v", body["max_ttl"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.OIDCClientSecret = "my-secret"
	cr.Spec.ForProvider.MaxTTL = "24h"

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}
