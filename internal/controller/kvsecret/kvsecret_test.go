package kvsecret

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestKVSecret(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.KVSecret) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	v2 := 2
	cr := &v1beta1.KVSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kv-secret",
			Namespace: "default",
		},
		Spec: v1beta1.KVSecretSpec{
			ForProvider: v1beta1.KVSecretParameters{
				MountPath: "secret",
				Path:      "myapp/config",
				Data:      map[string]string{"username": "admin", "password": "s3cret"},
				Version:   &v2,
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_V2_Exists(t *testing.T) {
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"data":{"username":"admin","password":"s3cret"}}}`)
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

func TestObserve_V1_Exists(t *testing.T) {
	v1 := 1
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"username":"admin","password":"s3cret"}}`)
	})
	defer srv.Close()
	cr.Spec.ForProvider.Version = &v1

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
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestCreate_V2(t *testing.T) {
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data field in request body")
		}
		if data["username"] != "admin" {
			t.Errorf("username = %v", data["username"])
		}
		if data["password"] != "s3cret" {
			t.Errorf("password = %v", data["password"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_V1(t *testing.T) {
	v1 := 1
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["username"] != "admin" {
			t.Errorf("username = %v", body["username"])
		}
		if body["password"] != "s3cret" {
			t.Errorf("password = %v", body["password"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()
	cr.Spec.ForProvider.Version = &v1

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestDelete_V2(t *testing.T) {
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUpdate_V2(t *testing.T) {
	var reqCount int32

	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/myapp/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		n := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"data":{"data":{"username":"admin","password":"old"}}}`)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data field in request body")
			return
		}
		if data["password"] != "s3cret" {
			t.Errorf("password = %v, want s3cret", data["password"])
			return
		}
		if n > 2 {
			t.Errorf("unexpected extra request #%d", n)
		}
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestUpdate_V2_DeletesRemovedKeys(t *testing.T) {
	e, srv, cr := newTestKVSecret(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"data":{"data":{"username":"admin","oldkey":"val","password":"old"}}}`)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		_ = json.Unmarshal(body, &parsed)
		data := parsed["data"].(map[string]interface{})
		if val, exists := data["oldkey"]; !exists {
			t.Error("expected oldkey to be present (null) for deletion")
		} else if val != nil {
			t.Errorf("expected oldkey to be null, got %v", val)
		}
		_, _ = fmt.Fprint(w, `{"data":{}}`)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}
