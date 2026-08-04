package token

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/token/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestToken(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Token) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Token{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-token",
			Namespace: "default",
		},
		Spec: v1beta1.TokenSpec{
			ForProvider: v1beta1.TokenParameters{
				RoleName:    "my-role",
				Policies:    []string{"default"},
				TTL:         "1h",
				DisplayName: "test-token",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_NoAccessor(t *testing.T) {
	e, srv, cr := newTestToken(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when accessor is empty")
	}
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestToken(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/token/lookup-accessor" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"accessor":"hmac-sha256:accessor","expire_time":"2099-01-01T00:00:00Z","policies":["default"]}}`)
	})
	defer srv.Close()

	cr.Status.AtProvider.Accessor = "test-accessor"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestToken(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	cr.Status.AtProvider.Accessor = "test-accessor"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Vault returns 404")
	}
}

func TestCreate(t *testing.T) {
	e, srv, cr := newTestToken(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/token/create" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["role_name"] != "my-role" {
			t.Errorf("role_name = %v", body["role_name"])
		}
		if body["display_name"] != "test-token" {
			t.Errorf("display_name = %v", body["display_name"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"auth":{"accessor":"hmac-sha256:accessor","client_token":"hmac-sha256:token","policies":["default"],"lease_duration":3600}}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cr.Status.AtProvider.Accessor != "hmac-sha256:accessor" {
		t.Errorf("accessor = %q, want %q", cr.Status.AtProvider.Accessor, "hmac-sha256:accessor")
	}
	if cr.Status.AtProvider.ClientToken != "hmac-sha256:token" {
		t.Errorf("client_token = %q, want %q", cr.Status.AtProvider.ClientToken, "hmac-sha256:token")
	}
}

func TestDelete(t *testing.T) {
	e, srv, cr := newTestToken(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/token/revoke-accessor" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Status.AtProvider.Accessor = "test-accessor"

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
