package secretbackendrole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/secretbackendrole/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestSecretBackendRole(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.SecretBackendRole) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	boolPtr := func(b bool) *bool { return &b }

	cr := &v1beta1.SecretBackendRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ssh-role",
			Namespace: "default",
		},
		Spec: v1beta1.SecretBackendRoleSpec{
			ForProvider: v1beta1.SecretBackendRoleParameters{
				Backend:         "ssh",
				Name:            "my-ssh-role",
				AllowedDomains:  []string{"example.com"},
				AllowSubdomains: boolPtr(true),
				AllowAnyName:    boolPtr(false),
				TTL:             "1h",
				MaxTTL:          "24h",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestSecretBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ssh/roles/my-ssh-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"allowed_domains":["example.com"],"allow_subdomains":true,"allow_any_name":false,"ttl":"1h","max_ttl":"24h"}}`)
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
	e, srv, cr := newTestSecretBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestSecretBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ssh/roles/my-ssh-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		domains, ok := body["allowed_domains"].([]interface{})
		if !ok || len(domains) != 1 || domains[0] != "example.com" {
			t.Errorf("allowed_domains = %v", body["allowed_domains"])
		}
		if body["allow_subdomains"] != true {
			t.Errorf("allow_subdomains = %v", body["allow_subdomains"])
		}
		if body["allow_any_name"] != false {
			t.Errorf("allow_any_name = %v", body["allow_any_name"])
		}
		if body["ttl"] != "1h" {
			t.Errorf("ttl = %v", body["ttl"])
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
	e, srv, cr := newTestSecretBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ssh/roles/my-ssh-role" {
			t.Errorf("path = %s", r.URL.Path)
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
	e, srv, cr := newTestSecretBackendRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ssh/roles/my-ssh-role" {
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
