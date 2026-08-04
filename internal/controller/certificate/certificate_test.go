package certificate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/certificate/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestCertificate(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.Certificate) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cert",
			Namespace: "default",
		},
		Spec: v1beta1.CertificateSpec{
			ForProvider: v1beta1.CertificateParameters{
				Backend:    "pki",
				Role:       "my-role",
				CommonName: "example.com",
				AltNames:   []string{"www.example.com"},
				TTL:        "2160h",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki/cert/abc-123" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"serial_number":"abc-123","certificate":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","issuing_ca":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","expiration":1893456000}}`)
	})
	defer srv.Close()

	cr.Status.AtProvider.Serial = "abc-123"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
}

func TestObserve_NotFound(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	cr.Status.AtProvider.Serial = "nonexistent"

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Vault returns 404")
	}
}

func TestObserve_NoSerial(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make any HTTP request when serial is empty")
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when serial is empty")
	}
}

func TestCreate(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki/issue/my-role" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["common_name"] != "example.com" {
			t.Errorf("common_name = %v", body["common_name"])
		}
		alts, ok := body["alt_names"].([]interface{})
		if !ok || len(alts) != 1 || alts[0] != "www.example.com" {
			t.Errorf("alt_names = %v", body["alt_names"])
		}
		if body["ttl"] != "2160h" {
			t.Errorf("ttl = %v", body["ttl"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"serial_number":"abc-123","certificate":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","issuing_ca":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","ca_chain":["-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----"],"private_key":"-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----","private_key_type":"rsa","expiration":1893456000}}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cr.Status.AtProvider.Serial != "abc-123" {
		t.Errorf("Serial = %s, want abc-123", cr.Status.AtProvider.Serial)
	}
}

func TestDelete(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/pki/revoke" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["serial_number"] != "abc-123" {
			t.Errorf("serial_number = %v", body["serial_number"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Status.AtProvider.Serial = "abc-123"

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_NoSerial(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make any HTTP request when serial is empty")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	requestCount := 0
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if strings.HasSuffix(r.URL.Path, "/revoke") {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["serial_number"] != "old-serial" {
				t.Errorf("revoke serial_number = %v", body["serial_number"])
			}
			w.WriteHeader(204)
			return
		}
		if strings.Contains(r.URL.Path, "/issue/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":{"serial_number":"new-serial","certificate":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","issuing_ca":"-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----","private_key":"-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----","private_key_type":"rsa","expiration":1893456000}}`)
			return
		}
		t.Errorf("unexpected path = %s", r.URL.Path)
	})
	defer srv.Close()

	cr.Status.AtProvider.Serial = "old-serial"

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests (revoke + issue), got %d", requestCount)
	}
	if cr.Status.AtProvider.Serial != "new-serial" {
		t.Errorf("Serial = %s, want new-serial", cr.Status.AtProvider.Serial)
	}
}

func TestUpdate_NoBackendOrRole(t *testing.T) {
	e, srv, cr := newTestCertificate(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make any HTTP request when backend or role is empty")
	})
	defer srv.Close()

	cr.Spec.ForProvider.Backend = ""
	cr.Spec.ForProvider.Role = ""

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}
