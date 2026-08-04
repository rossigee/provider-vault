package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/policy/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestPolicy(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Policy) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Policy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: v1beta1.PolicySpec{
			ForProvider: v1beta1.PolicyParameters{
				Name:   "mypolicy",
				Policy: `path "secret/*" { capabilities = ["read"] }`,
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"name":"mypolicy","policy":"path \"secret/*\" { capabilities = [\"read\"] }"}}`)
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

func TestObserve_UpToDate(t *testing.T) {
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"name":"mypolicy","policy":"path \"secret/*\" { capabilities = [\"read\"] }"}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true when policy matches")
	}
}

func TestObserve_NotUpToDate(t *testing.T) {
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"name":"mypolicy","policy":"path \"secret/*\" { capabilities = [\"write\"] }"}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when policy differs")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["policy"] != `path "secret/*" { capabilities = ["read"] }` {
			t.Errorf("policy = %v", body["policy"])
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
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
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
	e, srv, cr := newTestPolicy(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
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
