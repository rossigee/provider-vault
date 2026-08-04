package databasebackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/databasebackend/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestDatabaseBackend(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.DatabaseBackend) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.DatabaseBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-db-backend",
			Namespace: "default",
		},
		Spec: v1beta1.DatabaseBackendSpec{
			ForProvider: v1beta1.DatabaseBackendParameters{
				Backend:       "database",
				Name:          "mydb",
				PluginName:    "postgresql-database-plugin",
				ConnectionURL: "postgresql://{{username}}:{{password}}@localhost:5432/mydb",
				Username:      "vault",
				Password:      "vaultpass",
				AllowedRoles: []string{"readonly"},
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestDatabaseBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/config/mydb" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"connection_url":"postgresql://{{username}}:{{password}}@localhost:5432/mydb","username":"vault","plugin_name":"postgresql-database-plugin","allowed_roles":["readonly"]}}`)
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
	e, srv, cr := newTestDatabaseBackend(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestDatabaseBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/config/mydb" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["connection_url"] != "postgresql://{{username}}:{{password}}@localhost:5432/mydb" {
			t.Errorf("connection_url = %v", body["connection_url"])
		}
		if body["username"] != "vault" {
			t.Errorf("username = %v", body["username"])
		}
		if body["password"] != "vaultpass" {
			t.Errorf("password = %v", body["password"])
		}
		if body["plugin_name"] != "postgresql-database-plugin" {
			t.Errorf("plugin_name = %v", body["plugin_name"])
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
	e, srv, cr := newTestDatabaseBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/config/mydb" {
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
	e, srv, cr := newTestDatabaseBackend(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/config/mydb" {
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
