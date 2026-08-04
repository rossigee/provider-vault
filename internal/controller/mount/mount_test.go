package mount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/mount/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestMount(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Mount) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Mount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mount",
			Namespace: "default",
		},
		Spec: v1beta1.MountSpec{
			ForProvider: v1beta1.MountParameters{
				Path:        "secret",
				Type:        "kv",
				Description: "KV secrets engine",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestMount(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/mounts/secret" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"type":"kv","description":"KV secrets engine","config":{"default_lease_ttl":0,"max_lease_ttl":0}}}`)
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
	e, srv, cr := newTestMount(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestMount(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/mounts/secret" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "kv" {
			t.Errorf("type = %v", body["type"])
		}
		if body["description"] != "KV secrets engine" {
			t.Errorf("description = %v", body["description"])
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
	e, srv, cr := newTestMount(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/mounts/secret" {
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
	e, srv, cr := newTestMount(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/mounts/secret" {
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
