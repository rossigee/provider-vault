package transitkey

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-vault/apis/transitkey/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
)

func newTestTransitKey(t *testing.T, handler http.HandlerFunc) (*external, *httptest.Server, *v1beta1.TransitKey) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	vc, err := clients.NewVaultClientFromConfig(srv.URL, "test-token", true, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	cr := &v1beta1.TransitKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-key",
			Namespace: "default",
		},
		Spec: v1beta1.TransitKeySpec{
			ForProvider: v1beta1.TransitKeyParameters{
				Backend: "transit",
				Name:    "test-key",
				Type:    "aes256-gcm96",
			},
		},
	}

	return &external{service: vc}, srv, cr
}

func keyResponseBody() string {
	return `{"data":{"type":"aes256-gcm96","deletion_allowed":true,"min_decryption_version":1,"latest_version":3}}`
}

func TestObserve_KeyExists(t *testing.T) {
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/transit/keys/test-key" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, keyResponseBody())
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
	if cr.Status.AtProvider.LatestVersion != 3 {
		t.Errorf("LatestVersion = %d, want 3", cr.Status.AtProvider.LatestVersion)
	}
}

func TestObserve_KeyNotFound(t *testing.T) {
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestObserve_RotateToVersionBehind(t *testing.T) {
	target := 5
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, keyResponseBody())
	})
	defer srv.Close()

	cr.Spec.ForProvider.RotateToVersion = &target

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when latest version is behind target")
	}
}

func TestObserve_RotateToVersionMet(t *testing.T) {
	target := 3
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, keyResponseBody())
	})
	defer srv.Close()

	cr.Spec.ForProvider.RotateToVersion = &target

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true when latest version meets target")
	}
}

func TestObserve_MinDecryptionVersionDrift(t *testing.T) {
	mdv := 2
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, keyResponseBody())
	})
	defer srv.Close()

	cr.Spec.ForProvider.MinDecryptionVersion = &mdv

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when min_decryption_version drifts")
	}
}

func TestUpdate_RotatesToTarget(t *testing.T) {
	target := 5
	rotations := 0
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/rotate") {
			t.Errorf("path = %s", r.URL.Path)
		}
		rotations++
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.RotateToVersion = &target
	cr.Status.AtProvider.LatestVersion = 3

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rotations != 2 {
		t.Errorf("expected 2 rotations, got %d", rotations)
	}
}

func TestUpdate_NoRotationWhenMet(t *testing.T) {
	target := 3
	rotations := 0
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		rotations++
		w.WriteHeader(204)
	})
	defer srv.Close()

	cr.Spec.ForProvider.RotateToVersion = &target
	cr.Status.AtProvider.LatestVersion = 3

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rotations != 0 {
		t.Errorf("expected 0 rotations, got %d", rotations)
	}
}

func TestUpdate_ConfigureAndRotate(t *testing.T) {
	target := 4
	mdv := 1
	callCount := 0
	e, srv, cr := newTestTransitKey(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			if r.URL.Path != "/v1/transit/keys/test-key/config" {
				t.Errorf("config path = %s", r.URL.Path)
			}
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["min_decryption_version"] != float64(1) {
				t.Errorf("min_decryption_version = %v", body["min_decryption_version"])
			}
			w.WriteHeader(204)
		case 2:
			if r.URL.Path != "/v1/transit/keys/test-key/rotate" {
				t.Errorf("rotate path = %s", r.URL.Path)
			}
			w.WriteHeader(204)
		}
	})
	defer srv.Close()

	cr.Spec.ForProvider.MinDecryptionVersion = &mdv
	cr.Spec.ForProvider.RotateToVersion = &target
	cr.Status.AtProvider.LatestVersion = 3

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 Vault calls, got %d", callCount)
	}
}
