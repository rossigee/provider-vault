package clients

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/version"
)

func newFakeK8SClient(t *testing.T, data map[string]map[string]string) client.Client {
	t.Helper()
	b := fake.NewClientBuilder()
	for k, kv := range data {
		parts := strings.SplitN(k, "/", 2)
		ns := "default"
		name := k
		if len(parts) == 2 {
			ns, name = parts[0], parts[1]
		}
		d := make(map[string][]byte)
		for dk, dv := range kv {
			d[dk] = []byte(dv)
		}
		b.WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Data: d,
		})
	}
	return b.Build()
}

func newTestVaultClient(t *testing.T, handler http.HandlerFunc) (*VaultClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	client, err := NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}
	return client, srv
}

//nolint:errcheck // test HTTP handlers intentionally ignore Fprint/Decode errors
// --- Constructor ---

func TestNewVaultClientFromConfig(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c, err := NewVaultClientFromConfig("https://vault.example.com:8200", "mytoken", false, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.token != "mytoken" {
			t.Errorf("token = %q, want %q", c.token, "mytoken")
		}
		if c.baseURL.String() != "https://vault.example.com:8200" {
			t.Errorf("baseURL = %q", c.baseURL.String())
		}
	})

	t.Run("insecure TLS", func(t *testing.T) {
		c, err := NewVaultClientFromConfig("https://vault.example.com", "tok", true, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		tr := c.httpClient.Transport.(*http.Transport)
		if !tr.TLSClientConfig.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify=true")
		}
	})

	t.Run("NextProtos http/1.1", func(t *testing.T) {
		c, err := NewVaultClientFromConfig("https://vault.example.com", "tok", true, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		tr := c.httpClient.Transport.(*http.Transport)
		got := tr.TLSClientConfig.NextProtos
		want := []string{"http/1.1"}
		if !cmp.Equal(got, want) {
			t.Errorf("NextProtos = %v, want %v", got, want)
		}
	})

	t.Run("TLSNextProto empty (HTTP/2 disabled)", func(t *testing.T) {
		c, err := NewVaultClientFromConfig("https://vault.example.com", "tok", true, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		tr := c.httpClient.Transport.(*http.Transport)
		if tr.TLSNextProto == nil || len(tr.TLSNextProto) != 0 {
			t.Errorf("TLSNextProto = %v, want empty map", tr.TLSNextProto)
		}
	})

	t.Run("TLS client cert", func(t *testing.T) {
		certPEM, keyPEM := generateTestCert(t)
		c, err := NewVaultClientFromConfig("https://vault.example.com", "tok", true, nil, certPEM, keyPEM)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		tr := c.httpClient.Transport.(*http.Transport)
		if len(tr.TLSClientConfig.Certificates) != 1 {
			t.Errorf("Certificates = %d, want 1", len(tr.TLSClientConfig.Certificates))
		}
	})

	t.Run("CA cert", func(t *testing.T) {
		caPEM, _ := generateTestCert(t)
		c, err := NewVaultClientFromConfig("https://vault.example.com", "tok", false, caPEM, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		tr := c.httpClient.Transport.(*http.Transport)
		if tr.TLSClientConfig.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := NewVaultClientFromConfig("://invalid", "tok", false, nil, nil, nil)
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})

	t.Run("invalid client cert", func(t *testing.T) {
		_, err := NewVaultClientFromConfig("https://vault.example.com", "tok", false, nil, []byte("bad"), []byte("bad"))
		if err == nil {
			t.Error("expected error for invalid cert")
		}
	})
}

// --- AppRoleSecretID ---

func TestGenerateAppRoleSecretID(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Vault-Token") != "test-token" {
			t.Errorf("token = %s", r.Header.Get("X-Vault-Token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid123","secret_id_accessor":"acc456"}}`)


	})
	defer srv.Close()

	data, err := client.GenerateAppRoleSecretID(context.Background(), "approle", "myrole", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["secret_id"] != "sid123" {
		t.Errorf("secret_id = %v", data["secret_id"])
	}
	if data["secret_id_accessor"] != "acc456" {
		t.Errorf("secret_id_accessor = %v", data["secret_id_accessor"])
	}
}

func TestGenerateAppRoleSecretID_WithMetadata(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["metadata"] != `{"env":"test"}` {
			t.Errorf("metadata = %v", body["metadata"])
		}
_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid","secret_id_accessor":"acc"}}`)
	})
	defer srv.Close()

	_, err := client.GenerateAppRoleSecretID(context.Background(), "approle", "myrole", map[string]interface{}{
		"metadata": `{"env":"test"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLookupAppRoleSecretID(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id-accessor/lookup" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
		if body["secret_id_accessor"] != "test-accessor" {
			t.Errorf("accessor = %s", body["secret_id_accessor"])
		}
_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid","secret_id_accessor":"test-accessor"}}`)
	})
	defer srv.Close()

	data, err := client.LookupAppRoleSecretID(context.Background(), "approle", "myrole", "test-accessor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["secret_id"] != "sid" {
		t.Errorf("secret_id = %v", data["secret_id"])
	}
}

func TestDestroyAppRoleSecretID(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id/destroy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
		if body["secret_id"] != "sid123" {
			t.Errorf("secret_id = %s", body["secret_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.DestroyAppRoleSecretID(context.Background(), "approle", "myrole", "sid123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDestroyAppRoleSecretIDByAccessor(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id-accessor/destroy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
		if body["secret_id_accessor"] != "acc456" {
			t.Errorf("accessor = %s", body["secret_id_accessor"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.DestroyAppRoleSecretIDByAccessor(context.Background(), "approle", "myrole", "acc456"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- KV ---

func TestCreateKVSecret(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/secret/data/mysecret" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		d := body["data"].(map[string]interface{})
		if d["key1"] != "val1" {
			t.Errorf("data = %v", body)
		}
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()

	if err := client.CreateKVSecret(context.Background(), "mysecret", "secret", map[string]string{"key1": "val1"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetKVSecret(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/secret/data/mysecret" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"data":{"data":{"key1":"val1"}}}`)
	})
	defer srv.Close()

	data, err := client.GetKVSecret(context.Background(), "mysecret", "secret", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["key1"] != "val1" {
		t.Errorf("key1 = %q", data["key1"])
	}
}

func TestGetKVSecret_Missing(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()

	_, err := client.GetKVSecret(context.Background(), "nope", "secret", nil)
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestDeleteKVSecret(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/secret/data/mysecret" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.DeleteKVSecret(context.Background(), "mysecret", "secret", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Quota ---

func TestCreateRateQuota(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
			t.Errorf("path = %s", r.URL.Path)
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

	if err := client.CreateRateQuota(context.Background(), "rate-limit", "sys/health", "100", "second", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateLeaseQuota(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/quotas/lease/lease-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["max_leases"] != float64(50) {
			t.Errorf("max_leases = %v", body["max_leases"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.CreateLeaseQuota(context.Background(), "lease-limit", "secret/data/myapp", 50, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetQuota_Rate(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"name":"rate-limit","type":"rate","path":"sys/health","rate":100,"interval":"second"}}`)
	})
	defer srv.Close()

	data, err := client.GetQuota(context.Background(), "rate", "rate-limit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["name"] != "rate-limit" {
		t.Errorf("name = %v", data["name"])
	}
}

func TestGetQuota_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	_, err := client.GetQuota(context.Background(), "rate", "not-here")
	if err == nil {
		t.Error("expected error for missing quota")
	}
}

func TestDeleteQuota(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/quotas/rate/rate-limit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.DeleteQuota(context.Background(), "rate", "rate-limit"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Policy ---

func TestCreatePolicy(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
		if body["policy"] != `path "secret/*" { capabilities = ["read"] }` {
			t.Errorf("policy = %s", body["policy"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.CreatePolicy(context.Background(), "mypolicy",
		`path "secret/*" { capabilities = ["read"] }`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPolicy(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/sys/policies/acl/mypolicy" {
			t.Errorf("path = %s", r.URL.Path)
		}
_, _ = fmt.Fprint(w, `{"data":{"name":"mypolicy","policy":"read"}}`)
	})
	defer srv.Close()

	policy, err := client.GetPolicy(context.Background(), "mypolicy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy != "read" {
		t.Errorf("policy = %q", policy)
	}
}

func TestGetPolicy_Missing(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
_, _ = fmt.Fprint(w, `{"errors":["no policy found"]}`)
	})
	defer srv.Close()

	_, err := client.GetPolicy(context.Background(), "nope")
	if err == nil {
		t.Error("expected error for missing policy")
	}
}

func TestDeletePolicy(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.DeletePolicy(context.Background(), "mypolicy"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- HTTP Error Handling ---

func TestRequest_VaultAPIError(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
_, _ = fmt.Fprint(w, `{"errors":["internal error"]}`)
	})
	defer srv.Close()

	_, err := client.request(context.Background(), "GET", "/v1/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRequest_NonJSONError(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
_, _ = fmt.Fprint(w, `Service Unavailable`)
	})
	defer srv.Close()

	_, err := client.request(context.Background(), "GET", "/v1/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRequest_NetworkError(t *testing.T) {
	client, err := NewVaultClientFromConfig("https://127.0.0.1:1", "tok", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}
	_, err = client.request(context.Background(), "GET", "/v1/test", nil)
	if err == nil {
		t.Error("expected error for bad connection")
	}
}

// --- Vault Namespace ---

func TestVaultNamespaceHeader(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Namespace") != "admin" {
			t.Errorf("X-Vault-Namespace = %q, want 'admin'", r.Header.Get("X-Vault-Namespace"))
		}
_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()
	client.vaultNamespace = "admin"

	_, err := client.request(context.Background(), "GET", "/v1/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVaultNamespaceHeader_Empty(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Vault-Namespace"); v != "" {
			t.Errorf("X-Vault-Namespace = %q, want empty", v)
		}
_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()

	_, err := client.request(context.Background(), "GET", "/v1/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- User-Agent ---

func TestUserAgentHeader(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "provider-vault/"+version.Version {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()

	_, err := client.request(context.Background(), "GET", "/v1/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- URL Construction ---

func TestRequest_URLConstruction(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/kv/data/mysecret" {
			t.Errorf("path = %s", r.URL.Path)
		}
_, _ = fmt.Fprint(w, `{}`)
	})
	defer srv.Close()

	_, err := client.request(context.Background(), "GET", "/v1/kv/data/mysecret", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetConfig ---

func TestGetConfig_JSONWrappedToken(t *testing.T) {
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds": {"credentials": `{"token":"hvs.realtoken123"}`},
	})
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Address: "https://vault.example.com",
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "credentials",
					},
				},
			},
		},
	}

	cfg, err := GetConfig(context.Background(), kube, pc)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.Token != "hvs.realtoken123" {
		t.Errorf("token = %q", cfg.Token)
	}
}

func TestGetConfig_RawToken(t *testing.T) {
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds": {"credentials": "hvs.rawtoken456"},
	})
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "credentials",
					},
				},
			},
		},
	}

	cfg, err := GetConfig(context.Background(), kube, pc)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.Token != "hvs.rawtoken456" {
		t.Errorf("token = %q", cfg.Token)
	}
}

func TestGetConfig_MissingKey(t *testing.T) {
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds": {"credentials": "sometoken"},
	})
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "nonexistent",
					},
				},
			},
		},
	}

	_, err := GetConfig(context.Background(), kube, pc)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetConfig_NoSecretRef(t *testing.T) {
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
			},
		},
	}
	_, err := GetConfig(context.Background(), nil, pc)
	if err == nil {
		t.Error("expected error for no secretRef")
	}
}

func TestGetConfig_VaultNamespace(t *testing.T) {
	ns := "admin"
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Address: "https://vault.example.com",
			VaultNamespace: &ns,
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "credentials",
					},
				},
			},
		},
	}
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds": {"credentials": `{"token":"hvs.tok"}`},
	})

	cfg, err := GetConfig(context.Background(), kube, pc)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.VaultNamespace != "admin" {
		t.Errorf("VaultNamespace = %q", cfg.VaultNamespace)
	}
}

// --- TLS Secret References (partial coverage) ---

func TestGetConfig_TLSSecretRefs(t *testing.T) {
	caPEM, _ := generateTestCert(t)
	clientPEM, clientKeyPEM := generateTestCert(t)
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds":  {"credentials": `{"token":"hvs.tok"}`},
		"crossplane-system/vault-ca":     {"ca.crt": string(caPEM)},
		"crossplane-system/vault-client": {"tls.crt": string(clientPEM), "tls.key": string(clientKeyPEM)},
	})
	caKey := "ca.crt"
	ccKey := "tls.crt"
	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Address: "https://vault.example.com",
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "credentials",
					},
				},
			},
			TLS: &vaultv1beta1.TLSConfig{
				CACertSecretRef: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "vault-ca",
						Namespace: "crossplane-system",
					},
					Key: caKey,
				},
				ClientCertSecretRef: &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "vault-client",
						Namespace: "crossplane-system",
					},
					Key: ccKey,
				},
			},
		},
	}

	cfg, err := GetConfig(context.Background(), kube, pc)
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(cfg.CACertPEM) == 0 {
		t.Error("expected CACertPEM to be set")
	}
	if len(cfg.ClientCertPEM) == 0 {
		t.Error("expected ClientCertPEM to be set")
	}
	if len(cfg.ClientKeyPEM) > 0 {
		t.Log("ClientKeyPEM is set (from fake data)")
	}
}

// --- NewClientFromProviderConfig ---

func TestNewClientFromProviderConfig(t *testing.T) {
	kube := newFakeK8SClient(t, map[string]map[string]string{
		"crossplane-system/vault-creds": {"credentials": `{"token":"hvs.tok"}`},
	})

	pc := &vaultv1beta1.ProviderConfig{
		Spec: vaultv1beta1.ProviderConfigSpec{
			Address: "https://vault.example.com",
			Credentials: vaultv1beta1.ProviderCredentials{
				Source: "Secret",
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "vault-creds",
							Namespace: "crossplane-system",
						},
						Key: "credentials",
					},
				},
			},
		},
	}

	vc, err := NewClientFromProviderConfig(context.Background(), kube, pc)
	if err != nil {
		t.Fatalf("NewClientFromProviderConfig: %v", err)
	}
	if vc.token != "hvs.tok" {
		t.Errorf("token = %q", vc.token)
	}
}

func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test Certificate",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	certBuf := &bytes.Buffer{}
	if err := pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("Encode cert: %v", err)
	}
	keyBuf := &bytes.Buffer{}
	if err := pem.Encode(keyBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		t.Fatalf("Encode key: %v", err)
	}
	return certBuf.Bytes(), keyBuf.Bytes()
}

func TestRotateTransitKey(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/transit/keys/mykey/rotate" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	if err := client.RotateTransitKey(context.Background(), "transit", "mykey"); err != nil {
		t.Fatalf("RotateTransitKey: %v", err)
	}
}

func TestReadAppRoleRoleID(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/approle/role/myrole/role-id" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
	})
	defer srv.Close()

	roleID, err := client.ReadAppRoleRoleID(context.Background(), "approle", "myrole")
	if err != nil {
		t.Fatalf("ReadAppRoleRoleID: %v", err)
	}
	if roleID != "role-123" {
		t.Errorf("role_id = %s", roleID)
	}
}

func TestReadAppRoleRoleID_Missing(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.ReadAppRoleRoleID(context.Background(), "approle", "missing"); err == nil {
		t.Error("expected error for missing role")
	}
}

func TestConfigurePKICRL(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/pki/config/crl" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["expiry"] != "72h" {
			t.Errorf("expiry = %v", body["expiry"])
		}
		if body["auto_rebuild"] != true {
			t.Errorf("auto_rebuild = %v", body["auto_rebuild"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigurePKICRL(context.Background(), "pki", map[string]interface{}{
		"expiry":        "72h",
		"auto_rebuild":  true,
	})
	if err != nil {
		t.Fatalf("ConfigurePKICRL: %v", err)
	}
}

func TestGetPKICRLConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/pki/config/crl" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"expiry":"72h","auto_rebuild":true}}`)
	})
	defer srv.Close()

	data, err := client.GetPKICRLConfig(context.Background(), "pki")
	if err != nil {
		t.Fatalf("GetPKICRLConfig: %v", err)
	}
	if data["expiry"] != "72h" {
		t.Errorf("expiry = %v", data["expiry"])
	}
	if data["auto_rebuild"] != true {
		t.Errorf("auto_rebuild = %v", data["auto_rebuild"])
	}
}

func TestGetPKIURLs(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/pki/config/urls" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"issuing_certificates":["http://vault/ca"]}}`)
	})
	defer srv.Close()

	data, err := client.GetPKIURLs(context.Background(), "pki")
	if err != nil {
		t.Fatalf("GetPKIURLs: %v", err)
	}
	if data["issuing_certificates"] == nil {
		t.Error("expected issuing_certificates in data")
	}
}

func TestConfigureJWTAuth(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/jwt-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["oidc_discovery_url"] != "https://accounts.google.com" {
			t.Errorf("oidc_discovery_url = %v", body["oidc_discovery_url"])
		}
		if body["bound_issuer"] != "https://accounts.google.com" {
			t.Errorf("bound_issuer = %v", body["bound_issuer"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigureJWTAuth(context.Background(), "jwt-auth", map[string]interface{}{
		"oidc_discovery_url": "https://accounts.google.com",
		"bound_issuer":       "https://accounts.google.com",
	})
	if err != nil {
		t.Fatalf("ConfigureJWTAuth: %v", err)
	}
}

func TestGetJWTAuthConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/jwt-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"oidc_discovery_url":"https://accounts.google.com","bound_issuer":"https://accounts.google.com"}}`)
	})
	defer srv.Close()

	data, err := client.GetJWTAuthConfig(context.Background(), "jwt-auth")
	if err != nil {
		t.Fatalf("GetJWTAuthConfig: %v", err)
	}
	if data["oidc_discovery_url"] != "https://accounts.google.com" {
		t.Errorf("oidc_discovery_url = %v", data["oidc_discovery_url"])
	}
	if data["bound_issuer"] != "https://accounts.google.com" {
		t.Errorf("bound_issuer = %v", data["bound_issuer"])
	}
}

func TestGetJWTAuthConfig_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetJWTAuthConfig(context.Background(), "jwt-auth"); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestEnableAuditDevice(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/v1/sys/audit/file-audit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "file" {
			t.Errorf("type = %v", body["type"])
		}
		if body["description"] != "File audit device" {
			t.Errorf("description = %v", body["description"])
		}
		if body["local"] != false {
			t.Errorf("local = %v", body["local"])
		}
		opts, ok := body["options"].(map[string]interface{})
		if !ok {
			t.Fatal("expected options map")
		}
		if opts["file_path"] != "/var/log/audit.log" {
			t.Errorf("file_path = %v", opts["file_path"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.EnableAuditDevice(context.Background(), "file-audit", "file", "File audit device", false, map[string]string{"file_path": "/var/log/audit.log"})
	if err != nil {
		t.Fatalf("EnableAuditDevice: %v", err)
	}
}

func TestEnableAuditDevice_Local(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["local"] != true {
			t.Errorf("local = %v, want true", body["local"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.EnableAuditDevice(context.Background(), "local-audit", "syslog", "Local audit device", true, nil)
	if err != nil {
		t.Fatalf("EnableAuditDevice: %v", err)
	}
}

func TestGetAuditDevice(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/sys/audit/file-audit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"type":"file","description":"File audit device","local":false}}`)
	})
	defer srv.Close()

	data, err := client.GetAuditDevice(context.Background(), "file-audit")
	if err != nil {
		t.Fatalf("GetAuditDevice: %v", err)
	}
	if data["type"] != "file" {
		t.Errorf("type = %v", data["type"])
	}
	if data["description"] != "File audit device" {
		t.Errorf("description = %v", data["description"])
	}
}

func TestGetAuditDevice_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetAuditDevice(context.Background(), "missing-audit"); err == nil {
		t.Error("expected error for missing audit device")
	}
}

func TestDisableAuditDevice(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/v1/sys/audit/file-audit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.DisableAuditDevice(context.Background(), "file-audit")
	if err != nil {
		t.Fatalf("DisableAuditDevice: %v", err)
	}
}
func TestConfigureLDAPAuth(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/ldap-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["url"] != "ldap://ldap.example.com" {
			t.Errorf("url = %v", body["url"])
		}
		if body["binddn"] != "cn=vault,ou=apps,dc=example,dc=com" {
			t.Errorf("binddn = %v", body["binddn"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigureLDAPAuth(context.Background(), "ldap-auth", map[string]interface{}{
		"url":      "ldap://ldap.example.com",
		"binddn":   "cn=vault,ou=apps,dc=example,dc=com",
		"bindpass": "secret",
	})
	if err != nil {
		t.Fatalf("ConfigureLDAPAuth: %v", err)
	}
}

func TestGetLDAPAuthConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/ldap-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"url":"ldap://ldap.example.com","binddn":"cn=vault,ou=apps,dc=example,dc=com"}}`)
	})
	defer srv.Close()

	data, err := client.GetLDAPAuthConfig(context.Background(), "ldap-auth")
	if err != nil {
		t.Fatalf("GetLDAPAuthConfig: %v", err)
	}
	if data["url"] != "ldap://ldap.example.com" {
		t.Errorf("url = %v", data["url"])
	}
	if data["binddn"] != "cn=vault,ou=apps,dc=example,dc=com" {
		t.Errorf("binddn = %v", data["binddn"])
	}
}

func TestGetLDAPAuthConfig_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetLDAPAuthConfig(context.Background(), "ldap-auth"); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestConfigureAWSAuth(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/aws-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["iam_server_id_header_value"] != "vault.example.com" {
			t.Errorf("iam_server_id_header_value = %v", body["iam_server_id_header_value"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigureAWSAuth(context.Background(), "aws-auth", map[string]interface{}{
		"iam_server_id_header_value": "vault.example.com",
	})
	if err != nil {
		t.Fatalf("ConfigureAWSAuth: %v", err)
	}
}

func TestGetAWSAuthConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/aws-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"iam_server_id_header_value":"vault.example.com"}}`)
	})
	defer srv.Close()

	data, err := client.GetAWSAuthConfig(context.Background(), "aws-auth")
	if err != nil {
		t.Fatalf("GetAWSAuthConfig: %v", err)
	}
	if data["iam_server_id_header_value"] != "vault.example.com" {
		t.Errorf("iam_server_id_header_value = %v", data["iam_server_id_header_value"])
	}
}

func TestGetAWSAuthConfig_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetAWSAuthConfig(context.Background(), "aws-auth"); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestConfigureAzureAuth(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/azure-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tenant_id"] != "tenant-123" {
			t.Errorf("tenant_id = %v", body["tenant_id"])
		}
		if body["client_id"] != "client-456" {
			t.Errorf("client_id = %v", body["client_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigureAzureAuth(context.Background(), "azure-auth", map[string]interface{}{
		"tenant_id": "tenant-123",
		"client_id": "client-456",
	})
	if err != nil {
		t.Fatalf("ConfigureAzureAuth: %v", err)
	}
}

func TestGetAzureAuthConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/azure-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"tenant_id":"tenant-123","client_id":"client-456"}}`)
	})
	defer srv.Close()

	data, err := client.GetAzureAuthConfig(context.Background(), "azure-auth")
	if err != nil {
		t.Fatalf("GetAzureAuthConfig: %v", err)
	}
	if data["tenant_id"] != "tenant-123" {
		t.Errorf("tenant_id = %v", data["tenant_id"])
	}
	if data["client_id"] != "client-456" {
		t.Errorf("client_id = %v", data["client_id"])
	}
}

func TestGetAzureAuthConfig_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetAzureAuthConfig(context.Background(), "azure-auth"); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestConfigureGCPAuth(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/auth/gcp-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["service_account_email"] != "vault@example.iam.gserviceaccount.com" {
			t.Errorf("service_account_email = %v", body["service_account_email"])
		}
		if body["project_id"] != "my-project" {
			t.Errorf("project_id = %v", body["project_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	err := client.ConfigureGCPAuth(context.Background(), "gcp-auth", map[string]interface{}{
		"service_account_email": "vault@example.iam.gserviceaccount.com",
		"project_id":            "my-project",
	})
	if err != nil {
		t.Fatalf("ConfigureGCPAuth: %v", err)
	}
}

func TestGetGCPAuthConfig(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/auth/gcp-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"service_account_email":"vault@example.iam.gserviceaccount.com","project_id":"my-project"}}`)
	})
	defer srv.Close()

	data, err := client.GetGCPAuthConfig(context.Background(), "gcp-auth")
	if err != nil {
		t.Fatalf("GetGCPAuthConfig: %v", err)
	}
	if data["service_account_email"] != "vault@example.iam.gserviceaccount.com" {
		t.Errorf("service_account_email = %v", data["service_account_email"])
	}
	if data["project_id"] != "my-project" {
		t.Errorf("project_id = %v", data["project_id"])
	}
}

func TestGetGCPAuthConfig_NotFound(t *testing.T) {
	client, srv := newTestVaultClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	if _, err := client.GetGCPAuthConfig(context.Background(), "gcp-auth"); err == nil {
		t.Error("expected error for missing config")
	}
}
