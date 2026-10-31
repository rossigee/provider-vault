package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMountMissingMountIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":["No secret engine mount at pki-missing/"]}`))
	}))
	defer srv.Close()

	c, err := NewVaultClientFromConfig(srv.URL, "token", false, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	_, err = c.GetMount(context.Background(), "pki-missing/")
	if err == nil {
		t.Fatal("GetMount: expected an error for a missing mount")
	}
	if !IsNotFound(err) {
		t.Fatalf("GetMount: expected not-found for a missing mount, got %v", err)
	}
}

func TestGetMountOtherErrorIsNotNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["permission denied"]}`))
	}))
	defer srv.Close()

	c, err := NewVaultClientFromConfig(srv.URL, "token", false, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewVaultClientFromConfig: %v", err)
	}

	_, err = c.GetMount(context.Background(), "pki-x/")
	if err == nil {
		t.Fatal("GetMount: expected an error for a 403")
	}
	if IsNotFound(err) {
		t.Fatalf("GetMount: a 403 must not be reported as not-found: %v", err)
	}
}
