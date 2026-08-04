package approlesecretid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1beta1 "github.com/rossigee/provider-vault/apis/approlesecretid/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestHarness(t *testing.T, vaultHandler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AppRoleSecretID) {
	t.Helper()

	srv := httptest.NewTLSServer(vaultHandler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	kube := fake.NewClientBuilder().WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-connection",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"secret_id":          []byte("sid-from-secret"),
				"secret_id_accessor": []byte("acc-from-secret"),
			},
		},
	).Build()

	cr := &v1beta1.AppRoleSecretID{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
		Spec: v1beta1.AppRoleSecretIDSpec{
			ForProvider: v1beta1.AppRoleSecretIDParameters{
				Backend:  "approle",
				RoleName: "myrole",
				Metadata: map[string]string{"env": "test"},
			},
		},
	}

	return &external{service: vc, kube: kube}, srv, cr
}

// --- Observe ---

func TestObserve_NoExternalName(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call: %s %s", r.Method, r.URL.Path)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when no external-name")
	}
}

func TestObserve_WithAccessor(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/approle/role/myrole/role-id":
			_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
			return
		case r.URL.Path != "/v1/auth/approle/role/myrole/secret-id-accessor/lookup":
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

	meta.SetExternalName(cr, "test-accessor")

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
	if cr.Status.AtProvider.RoleID != "role-123" {
		t.Errorf("RoleID = %s", cr.Status.AtProvider.RoleID)
	}
}

func TestObserve_AccessorNotFound(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	meta.SetExternalName(cr, "test-accessor")

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false when Vault returns 404")
	}
}

// --- Create ---

func TestCreate(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/approle/role/myrole/role-id":
			_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
			return
		case r.URL.Path != "/v1/auth/approle/role/myrole/secret-id":
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["metadata"] != `{"env":"test"}` {
			t.Errorf("metadata = %v", body["metadata"])
		}
		_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid-new","secret_id_accessor":"acc-new"}}`)
	})
	defer srv.Close()

	creation, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if string(creation.ConnectionDetails["secret_id"]) != "sid-new" {
		t.Errorf("secret_id = %s", creation.ConnectionDetails["secret_id"])
	}
	if string(creation.ConnectionDetails["secret_id_accessor"]) != "acc-new" {
		t.Errorf("secret_id_accessor = %s", creation.ConnectionDetails["secret_id_accessor"])
	}
	if string(creation.ConnectionDetails["role_id"]) != "role-123" {
		t.Errorf("role_id = %s", creation.ConnectionDetails["role_id"])
	}
	if meta.GetExternalName(cr) != "acc-new" {
		t.Errorf("external-name = %s", meta.GetExternalName(cr))
	}
}

func TestCreate_WithMetadata(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/role-id") {
			_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["metadata"] != `{"env":"test"}` {
			t.Errorf("metadata = %v", body["metadata"])
		}
		_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid","secret_id_accessor":"acc"}}`)
	})
	defer srv.Close()

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_CustomBackendRole(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/custom/role/custom-role/role-id":
			_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
			return
		case r.URL.Path != "/v1/auth/custom/role/custom-role/secret-id":
			t.Fatalf("path = %s, want /v1/auth/custom/role/custom-role/secret-id", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid","secret_id_accessor":"acc"}}`)
	})
	defer srv.Close()
	cr.Spec.ForProvider.Backend = "custom"
	cr.Spec.ForProvider.RoleName = "custom-role"

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

// --- Update ---

func TestUpdate_DestroyOldByAccessor(t *testing.T) {
	callCount := 0
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id-accessor/destroy" {
				t.Errorf("destroy path = %s", r.URL.Path)
			}
			w.WriteHeader(204)
		case 2:
			_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid-new","secret_id_accessor":"acc-new"}}`)
		case 3:
			if r.URL.Path != "/v1/auth/approle/role/myrole/role-id" {
				t.Errorf("role-id path = %s", r.URL.Path)
			}
			_, _ = fmt.Fprint(w, `{"data":{"role_id":"role-123"}}`)
		}
	})
	defer srv.Close()

	meta.SetExternalName(cr, "acc-old")

	update, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if string(update.ConnectionDetails["secret_id"]) != "sid-new" {
		t.Errorf("secret_id = %s", update.ConnectionDetails["secret_id"])
	}
	if string(update.ConnectionDetails["role_id"]) != "role-123" {
		t.Errorf("role_id = %s", update.ConnectionDetails["role_id"])
	}
	if callCount != 3 {
		t.Errorf("expected 3 Vault calls, got %d", callCount)
	}
}

// --- Delete ---

func TestDelete_ByAccessor(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id-accessor/destroy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["secret_id_accessor"] != "test-accessor" {
			t.Errorf("accessor = %s", body["secret_id_accessor"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	meta.SetExternalName(cr, "test-accessor")

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_FallbackToSecretID(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/approle/role/myrole/secret-id/destroy" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["secret_id"] != "sid-from-secret" {
			t.Errorf("secret_id = %s", body["secret_id"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{
		Name: "test-connection",
	}
	meta.SetExternalName(cr, cr.GetName())

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDelete_NoIdentifier(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// --- getSecretID ---

func TestGetSecretID_FromExternalName(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
_, _ = fmt.Fprint(w, `{"data":{"secret_id":"sid-vault","secret_id_accessor":"acc-vault"}}`)
	})
	defer srv.Close()

	meta.SetExternalName(cr, "acc-vault")
	sid, acc := e.getSecretID(context.Background(), cr)
	if sid != "sid-vault" {
		t.Errorf("secret_id = %s", sid)
	}
	if acc != "acc-vault" {
		t.Errorf("accessor = %s", acc)
	}
}

func TestGetSecretID_FromConnectionSecret(t *testing.T) {
	e, srv, cr := newTestHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	})
	defer srv.Close()

	meta.SetExternalName(cr, cr.GetName())
	cr.Spec.WriteConnectionSecretToReference = &xpv1.LocalSecretReference{
		Name: "test-connection",
	}

	sid, acc := e.getSecretID(context.Background(), cr)
	if sid != "sid-from-secret" {
		t.Errorf("secret_id = %s", sid)
	}
	if acc != "acc-from-secret" {
		t.Errorf("accessor = %s", acc)
	}
}
