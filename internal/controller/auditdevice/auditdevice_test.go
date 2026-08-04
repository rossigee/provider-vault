package auditdevice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/auditdevice/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestAuditDevice(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AuditDevice) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AuditDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-audit",
			Namespace: "default",
		},
		Spec: v1beta1.AuditDeviceSpec{
			ForProvider: v1beta1.AuditDeviceParameters{
				Path:        "file-audit",
				Type:        "file",
				Description: "File audit device",
				Options:     map[string]string{"file_path": "/var/log/audit.log"},
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_Exists(t *testing.T) {
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/audit/file-audit" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"type":"file","description":"File audit device","local":false}}`)
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
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestCreate_FileAudit(t *testing.T) {
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
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

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_LocalAudit(t *testing.T) {
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["local"] != true {
			t.Errorf("local = %v, want true", body["local"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	local := true
	cr.Spec.ForProvider.Local = &local

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestDelete(t *testing.T) {
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/audit/file-audit" {
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
	e, srv, cr := newTestAuditDevice(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/audit/file-audit" {
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
