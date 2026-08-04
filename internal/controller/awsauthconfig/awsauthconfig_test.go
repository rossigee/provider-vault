package awsauthconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/awsauthconfig/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestAWSAuthConfig(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.AWSAuthConfig) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.AWSAuthConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aws",
			Namespace: "default",
		},
		Spec: v1beta1.AWSAuthConfigSpec{
			ForProvider: v1beta1.AWSAuthConfigParameters{
				Backend:                "aws-auth",
				IAMServerIDHeaderValue: "vault.example.com",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func TestObserve_ConfigExists(t *testing.T) {
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/aws-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"iam_server_id_header_value":"vault.example.com"}}`)
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
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestObserve_Drift(t *testing.T) {
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"iam_server_id_header_value":"old.example.com"}}`)
	})
	defer srv.Close()

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when config drifts")
	}
}

func TestCreate_FullConfig(t *testing.T) {
	disableRedirect := true
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/aws-auth/config" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["iam_server_id_header_value"] != "vault.example.com" {
			t.Errorf("iam_server_id_header_value = %v", body["iam_server_id_header_value"])
		}
		if body["sts_disable_redirect"] != true {
			t.Errorf("sts_disable_redirect = %v", body["sts_disable_redirect"])
		}
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.STSDisableRedirect = &disableRedirect

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	defer srv.Close()

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete_NoOp(t *testing.T) {
	e, srv, cr := newTestAWSAuthConfig(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Vault call")
	})
	defer srv.Close()

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
