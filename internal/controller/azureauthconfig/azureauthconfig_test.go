package azureauthconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/azureauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestAzureAuthConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AzureAuthConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AzureAuthConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-azure",
			Namespace: "default",
		},
		Spec: v1beta1.AzureAuthConfigSpec{
			ForProvider: v1beta1.AzureAuthConfigParameters{
				Backend:  "azure-auth",
				TenantID: "tenant-123",
				ClientID: "client-456",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_ConfigExists(t *testing.T) {
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/azure-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"tenant_id":"tenant-123","client_id":"client-456"}}`)
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
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"tenant_id":"old-tenant","client_id":"client-456"}}`)
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

func TestCreate_FullConfig(t *testing.T) {
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/azure-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tenant_id"] != "tenant-123" {
			t.Errorf("tenant_id = %v", body["tenant_id"])
		}
		if body["client_id"] != "client-456" {
			t.Errorf("client_id = %v", body["client_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_WithOptionalFields(t *testing.T) {
	maxRetry := 10
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["environment"] != "AzureUSGovernment" {
			t.Errorf("environment = %v", body["environment"])
		}
		if body["login_max_user_retry"] != float64(10) {
			t.Errorf("login_max_user_retry = %v", body["login_max_user_retry"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.Environment = "AzureUSGovernment"
	cr.Spec.ForProvider.LoginMaxUserRetry = &maxRetry

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete_NoOp(t *testing.T) {
	e, srv, cr := newTestAzureAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
