package authbackendrole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/authbackendrole/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestAuthBackendRole(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AuthBackendRole) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AuthBackendRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-auth-backend-role",
			Namespace: "default",
		},
		Spec: v1beta1.AuthBackendRoleSpec{
			ForProvider: v1beta1.AuthBackendRoleParameters{
				Backend:       "jwt",
				RoleName:      "my-role",
				TokenPolicies: []string{"default"},
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestAuthBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt/role/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"role_type":"jwt","token_policies":["default"],"token_ttl":0,"token_max_ttl":0}}`)
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
	e, srv, cr := newTestAuthBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestCreate(t *testing.T) {
	e, srv, cr := newTestAuthBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt/role/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		policies, ok := body["token_policies"].([]interface{})
		if !ok || len(policies) != 1 || policies[0] != "default" {
			t.Errorf("token_policies = %v", body["token_policies"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestDelete(t *testing.T) {
	e, srv, cr := newTestAuthBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt/role/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestAuthBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/jwt/role/my-role" {
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

func newTestOIDCBackendRole(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AuthBackendRole) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AuthBackendRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-oidc-admin-role",
			Namespace: "default",
		},
		Spec: v1beta1.AuthBackendRoleSpec{
			ForProvider: v1beta1.AuthBackendRoleParameters{
				Backend:             "oidc",
				RoleName:            "admin",
				RoleType:            "oidc",
				BoundAudiences:      []string{"vault"},
				BoundClaims:         map[string]string{"groups": "admin"},
				BoundClaimsType:     "string",
				UserClaim:           "sub",
				OidcScopes:          []string{"openid", "profile", "email", "groups"},
				AllowedRedirectURIs: []string{"https://vault.example.com/ui/vault/auth/oidc/oidc/callback"},
				TokenPolicies:       []string{"vault-admin", "default"},
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestCreate_OIDCBoundClaims(t *testing.T) {
	e, srv, cr := newTestOIDCBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/oidc/role/admin" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["role_type"] != "oidc" {
			t.Errorf("role_type = %v", body["role_type"])
		}
		claims, ok := body["bound_claims"].(map[string]interface{})
		if !ok || claims["groups"] != "admin" {
			t.Errorf("bound_claims = %v", body["bound_claims"])
		}
		if body["bound_claims_type"] != "string" {
			t.Errorf("bound_claims_type = %v", body["bound_claims_type"])
		}
		scopes, ok := body["oidc_scopes"].([]interface{})
		if !ok || len(scopes) != 4 || scopes[3] != "groups" {
			t.Errorf("oidc_scopes = %v", body["oidc_scopes"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestObserve_OIDCBoundClaimsUpToDate(t *testing.T) {
	e, srv, cr := newTestOIDCBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"role_type":"oidc","bound_audiences":["vault"],"bound_claims":{"groups":"admin"},"bound_claims_type":"string","user_claim":"sub","oidc_scopes":["openid","profile","email","groups"],"allowed_redirect_uris":["https://vault.example.com/ui/vault/auth/oidc/oidc/callback"],"token_policies":["vault-admin","default"]}}`)
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

func TestObserve_OIDCBoundClaimsDrift(t *testing.T) {
	e, srv, cr := newTestOIDCBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"role_type":"oidc","bound_audiences":["vault"],"bound_claims":{"groups":"everyone"},"bound_claims_type":"string","user_claim":"sub","token_policies":["vault-admin","default"]}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false on bound_claims drift")
	}
}
