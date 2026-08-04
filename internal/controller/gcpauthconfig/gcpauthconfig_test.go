package gcpauthconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/gcpauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestGCPAuthConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.GCPAuthConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.GCPAuthConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gcp",
			Namespace: "default",
		},
		Spec: v1beta1.GCPAuthConfigSpec{
			ForProvider: v1beta1.GCPAuthConfigParameters{
				Backend:              "gcp-auth",
				ServiceAccountEmail: "vault@example.iam.gserviceaccount.com",
				ProjectID:            "my-project",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_ConfigExists(t *testing.T) {
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/gcp-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"service_account_email":"vault@example.iam.gserviceaccount.com","project_id":"my-project"}}`)
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
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"service_account_email":"old@example.iam.gserviceaccount.com","project_id":"my-project"}}`)
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
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/gcp-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["service_account_email"] != "vault@example.iam.gserviceaccount.com" {
			t.Errorf("service_account_email = %v", body["service_account_email"])
		}
		if body["project_id"] != "my-project" {
			t.Errorf("project_id = %v", body["project_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_WithZoneAndCluster(t *testing.T) {
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["zone"] != "us-central1-a" {
			t.Errorf("zone = %v", body["zone"])
		}
		if body["cluster_name"] != "my-cluster" {
			t.Errorf("cluster_name = %v", body["cluster_name"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.Zone = "us-central1-a"
	cr.Spec.ForProvider.ClusterName = "my-cluster"

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete_NoOp(t *testing.T) {
	e, srv, cr := newTestGCPAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
