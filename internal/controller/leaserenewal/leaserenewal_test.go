package leaserenewal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/leaserenewal/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestLeaseRenewal(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.LeaseRenewal) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.LeaseRenewal{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-renewal",
			Namespace: "default",
		},
		Spec: v1beta1.LeaseRenewalSpec{
			ForProvider: v1beta1.LeaseRenewalParameters{
				LeaseID: "hmac-sha256:abc123",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/leases/lookup" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["lease_id"] != "hmac-sha256:abc123" {
			t.Errorf("lease_id = %v", body["lease_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"lease_id":"hmac-sha256:abc123","renewable":true,"lease_duration":3600}}`)
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
	if cr.Status.AtProvider.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", cr.Status.AtProvider.TTL)
	}
	if !cr.Status.AtProvider.Renewable {
		t.Error("expected Renewable=true")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
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
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/leases/renew" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["lease_id"] != "hmac-sha256:abc123" {
			t.Errorf("lease_id = %v", body["lease_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"lease_id":"hmac-sha256:abc123","renewable":true,"lease_duration":3600}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cr.Status.AtProvider.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", cr.Status.AtProvider.TTL)
	}
	if cr.Status.AtProvider.LastRenewed == nil {
		t.Error("expected LastRenewed to be set")
	}
}

func TestCreate_WithIncrement(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["increment"] != float64(1800) {
			t.Errorf("increment = %v, want 1800", body["increment"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"lease_id":"hmac-sha256:abc123","renewable":true,"lease_duration":1800}`)
	})
	defer srv.Close()

	incr := 1800
	cr.Spec.ForProvider.Increment = &incr

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestDelete_NoRevoke(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("expected no request to Vault when revokeOnDelete=false")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_WithRevoke(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/leases/revoke" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["lease_id"] != "hmac-sha256:abc123" {
			t.Errorf("lease_id = %v", body["lease_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	revokeOnDelete := true
	cr.Spec.ForProvider.RevokeOnDelete = &revokeOnDelete

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestLeaseRenewal(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/leases/renew" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"lease_id":"hmac-sha256:abc123","renewable":true,"lease_duration":7200}`)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if cr.Status.AtProvider.TTL != 7200 {
		t.Errorf("TTL = %d, want 7200", cr.Status.AtProvider.TTL)
	}
}
