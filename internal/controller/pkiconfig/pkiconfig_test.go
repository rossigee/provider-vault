package pkiconfig

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/pkiconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestPKIConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.PKIConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.PKIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pki",
			Namespace: "default",
		},
		Spec: v1beta1.PKIConfigSpec{
			ForProvider: v1beta1.PKIConfigParameters{
				Backend:    "pki",
				Type:       "root_internal",
				CommonName: "test-root",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func crlConfigBody() string {
	return `{"data":{"expiry":"72h","disable":false,"ocsp_disable":false,"auto_rebuild":true,"auto_rebuild_grace_period":"12h","enable_delta":true,"delta_rebuild_interval":"15m"}}`
}

func TestObserve_ExistsWithoutSubConfig(t *testing.T) {
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki/ca/pem" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")
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
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestObserve_NoIssuerReturnsNotFound(t *testing.T) {
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Vault returns 204 (no issuer)")
	}
}

func TestObserve_CRLUpToDate(t *testing.T) {
	autoRebuild := true
	enableDelta := true
	disable := false
	ocspDisable := false
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/ca/pem"):
			_, _ = fmt.Fprint(w, "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")
		case strings.HasSuffix(r.URL.Path, "/config/crl"):
			_, _ = fmt.Fprint(w, crlConfigBody())
		default:
			t.Errorf("unexpected path = %s", r.URL.Path)
		}
	})
	defer srv.Close()

	p := &cr.Spec.ForProvider
	p.CrlExpiry = "72h"
	p.CrlDisable = &disable
	p.OcspDisable = &ocspDisable
	p.CrlAutoRebuild = &autoRebuild
	p.CrlAutoRebuildGracePeriod = "12h"
	p.CrlEnableDelta = &enableDelta
	p.CrlDeltaRebuildInterval = "15m"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true when CRL config matches")
	}
}

func TestObserve_CRLDrift(t *testing.T) {
	autoRebuild := false
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/ca/pem"):
			_, _ = fmt.Fprint(w, "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")
		case strings.HasSuffix(r.URL.Path, "/config/crl"):
			_, _ = fmt.Fprint(w, crlConfigBody())
		default:
			t.Errorf("unexpected path = %s", r.URL.Path)
		}
	})
	defer srv.Close()

	p := &cr.Spec.ForProvider
	p.CrlAutoRebuild = &autoRebuild

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when CRL config drifts")
	}
}

func TestObserve_CRLExpiryDrift(t *testing.T) {
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/ca/pem"):
			_, _ = fmt.Fprint(w, "-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")
		case strings.HasSuffix(r.URL.Path, "/config/crl"):
			_, _ = fmt.Fprint(w, crlConfigBody())
		default:
			t.Errorf("unexpected path = %s", r.URL.Path)
		}
	})
	defer srv.Close()

	cr.Spec.ForProvider.CrlExpiry = "48h"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when CRL expiry drifts")
	}
}

func TestUpdate_ConfiguresCRL(t *testing.T) {
	autoRebuild := true
	callCount := 0
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path != "/v1/pki/config/crl" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()

	p := &cr.Spec.ForProvider
	p.CrlExpiry = "72h"
	p.CrlAutoRebuild = &autoRebuild

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 Vault call, got %d", callCount)
	}
}

func TestUpdate_NoCRLCallWithoutCRLFields(t *testing.T) {
	callCount := 0
	e, srv, cr := newTestPKIConfig(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		t.Errorf("unexpected Vault call: %s %s", r.Method, r.URL.Path)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 Vault calls, got %d", callCount)
	}
}
