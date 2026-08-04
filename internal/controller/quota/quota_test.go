package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/quota/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestRateQuota(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Quota) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Quota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rate-quota",
			Namespace: "default",
		},
		Spec: v1beta1.QuotaSpec{
			ForProvider: v1beta1.QuotaParameters{
				Name:     "rate-limit",
				Type:     "rate",
				Path:     "sys/health",
				Rate:     "100",
				Interval: "second",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func newTestLeaseQuota(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Quota) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Quota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lease-quota",
			Namespace: "default",
		},
		Spec: v1beta1.QuotaSpec{
			ForProvider: v1beta1.QuotaParameters{
				Name:      "lease-limit",
				Type:      "lease",
				Path:      "secret/data/myapp",
				MaxLeases: func() *int { v := 50; return &v }(),
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_RateQuota_Exists(t *testing.T) {
	e, srv, cr := newTestRateQuota(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"name":"rate-limit","type":"rate","path":"sys/health","rate":100,"interval":"second"}}`)
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
	e, srv, cr := newTestRateQuota(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestCreate_RateQuota(t *testing.T) {
	e, srv, cr := newTestRateQuota(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["rate"] != "100" {
			t.Errorf("rate = %v", body["rate"])
		}
		if body["interval"] != "second" {
			t.Errorf("interval = %v", body["interval"])
		}
		if body["path"] != "sys/health" {
			t.Errorf("path = %v", body["path"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_LeaseQuota(t *testing.T) {
	e, srv, cr := newTestLeaseQuota(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/quotas/lease/lease-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["max_leases"] != float64(50) {
			t.Errorf("max_leases = %v", body["max_leases"])
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
	e, srv, cr := newTestRateQuota(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
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

func TestUpdate_RateQuota(t *testing.T) {
	e, srv, cr := newTestRateQuota(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
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
