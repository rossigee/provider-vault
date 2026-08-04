package databaserole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/databaserole/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestDatabaseRole(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.DatabaseRole) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.DatabaseRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-db-role",
			Namespace: "default",
		},
		Spec: v1beta1.DatabaseRoleSpec{
			ForProvider: v1beta1.DatabaseRoleParameters{
				Backend:            "database",
				Name:               "my-role",
				DBName:             "mydb",
				CreationStatements: []string{"CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'"},
				RevocationStatements: []string{"REVOKE OWNERSHIP ON ALL TABLES IN SCHEMA public TO \"{{name}}\""},
				RollbackStatements:   []string{"DROP ROLE IF EXISTS \"{{name}}\""},
				RenewStatements:      []string{"ALTER ROLE \"{{name}}\" VALID UNTIL '{{expiration}}'"},
				DefaultTTL:           "1h",
				MaxTTL:               "24h",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestDatabaseRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/roles/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"db_name":"mydb","creation_statements":["CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'"],"revocation_statements":["REVOKE OWNERSHIP ON ALL TABLES IN SCHEMA public TO \"{{name}}\""],"rollback_statements":["DROP ROLE IF EXISTS \"{{name}}\""],"renew_statements":["ALTER ROLE \"{{name}}\" VALID UNTIL '{{expiration}}'"],"default_ttl":"1h","max_ttl":"24h"}}`)
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
	e, srv, cr := newTestDatabaseRole(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestDatabaseRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/roles/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["db_name"] != "mydb" {
			t.Errorf("db_name = %v", body["db_name"])
		}
		stmts, ok := body["creation_statements"].([]interface{})
		if !ok || len(stmts) != 1 {
			t.Errorf("expected 1 creation_statements, got %v", body["creation_statements"])
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
	e, srv, cr := newTestDatabaseRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/roles/my-role" {
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
	e, srv, cr := newTestDatabaseRole(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/database/roles/my-role" {
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
