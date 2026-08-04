package kubernetesauthconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/kubernetesauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestKubernetesAuthConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.KubernetesAuthConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.KubernetesAuthConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8s-auth-config",
			Namespace: "default",
		},
		Spec: v1beta1.KubernetesAuthConfigSpec{
			ForProvider: v1beta1.KubernetesAuthConfigParameters{
				Backend:        "k8s",
				KubernetesHost: "https://kubernetes.default.svc",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestKubernetesAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/k8s/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"kubernetes_host":"https://kubernetes.default.svc","kubernetes_ca_cert":"","disable_iss_validation":false,"disable_local_ca_jwt":false}}`)
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
	if cr.Status.AtProvider.Backend != "k8s" {
		t.Errorf("Backend = %q, want %q", cr.Status.AtProvider.Backend, "k8s")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestKubernetesAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestKubernetesAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/k8s/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["kubernetes_host"] != "https://kubernetes.default.svc" {
			t.Errorf("kubernetes_host = %v", body["kubernetes_host"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestKubernetesAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/k8s/config" {
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
