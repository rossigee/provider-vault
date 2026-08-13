package identitygroup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/identitygroup/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestIdentityGroup(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.IdentityGroup) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.IdentityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-identity-group",
			Namespace: "default",
		},
		Spec: v1beta1.IdentityGroupSpec{
			ForProvider: v1beta1.IdentityGroupParameters{
				Name:            "my-group",
				Type:            "internal",
				Policies:        []string{"default"},
				MemberEntityIDs: []string{"entity-id-1"},
				Metadata:        map[string]string{"env": "test"},
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestIdentityGroup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/identity/group/name/my-group" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"id":"group-id-123","name":"my-group","type":"internal","policies":["default"],"member_entity_ids":["entity-id-1"],"metadata":{"env":"test"}}}`)
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
	if cr.Status.AtProvider.ID != "group-id-123" {
		t.Errorf("ID = %q, want %q", cr.Status.AtProvider.ID, "group-id-123")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestIdentityGroup(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestIdentityGroup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/identity/group" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "my-group" {
			t.Errorf("name = %v", body["name"])
		}
		if body["type"] != "internal" {
			t.Errorf("type = %v", body["type"])
		}
		policies, ok := body["policies"].([]interface{})
		if !ok || len(policies) != 1 || policies[0] != "default" {
			t.Errorf("policies = %v", body["policies"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"id":"group-id-123"}}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cr.Status.AtProvider.ID != "group-id-123" {
		t.Errorf("ID = %q, want %q", cr.Status.AtProvider.ID, "group-id-123")
	}
}

func TestDelete(t *testing.T) {
	e, srv, cr := newTestIdentityGroup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/identity/group/name/my-group" {
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
	e, srv, cr := newTestIdentityGroup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/identity/group/name/my-group" {
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
