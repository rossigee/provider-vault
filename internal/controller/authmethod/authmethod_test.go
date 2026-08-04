package authmethod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestAuthMethod(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AuthMethod) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AuthMethod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-auth-method",
			Namespace: "default",
		},
		Spec: v1beta1.AuthMethodSpec{
			ForProvider: v1beta1.AuthMethodParameters{
				MountPath: "my-auth",
				Type:      "approle",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestAuthMethod(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/auth" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"my-auth/":{"type":"approle","description":"AppRole auth","config":{"default_lease_ttl":0,"max_lease_ttl":0},"local":false,"seal_wrap":false,"external_entropy_access":false}}}`)
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
	e, srv, cr := newTestAuthMethod(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"token/":{"type":"token","description":"token based credentials","config":{}}}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when auth method is not found")
	}
}

func TestCreate(t *testing.T) {
	e, srv, cr := newTestAuthMethod(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/auth/my-auth" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "approle" {
			t.Errorf("type = %v", body["type"])
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
	e, srv, cr := newTestAuthMethod(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/auth/my-auth" {
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
	e, srv, cr := newTestAuthMethod(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/auth/my-auth" {
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
