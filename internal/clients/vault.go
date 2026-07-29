package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
)

const vaultTokenHeader = "X-Vault-Token"

type VaultClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
}

func NewClient(ctx context.Context, kube client.Client, mg xpv1.ManagedResource) (*VaultClient, error) {
	// In a real Crossplane provider, the connector pattern extracts
	// ProviderConfig and reads the token from the referenced Secret.
	// This is a stub - the actual connector in the controller handles this.
	panic("implemented via connector pattern")
}

func NewVaultClientFromConfig(baseURL string, token string, tlsInsecure bool) (*VaultClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse vault address")
	}
	return &VaultClient{
		baseURL: u,
		httpClient: &http.Client{
			Transport: &http.Transport{
				// TLS config is minimal - SSL_CERT_FILE env handles CA trust
				TLSClientConfig: nil,
			},
		},
		token: token,
	}, nil
}

func (c *VaultClient) request(ctx context.Context, method, reqPath string, body interface{}) ([]byte, error) {
	u, err := url.Parse(c.baseURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse base URL")
	}
	u.Path = path.Join(u.Path, reqPath)

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set(vaultTokenHeader, c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vault API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// KV v2

func (c *VaultClient) CreateKVSecret(ctx context.Context, path, mountPath string, data map[string]string) error {
	dataPath := fmt.Sprintf("/v1/%s/data/%s", mountPath, path)
	_, err := c.request(ctx, http.MethodPost, dataPath, map[string]interface{}{
		"data": data,
	})
	return err
}

func (c *VaultClient) GetKVSecret(ctx context.Context, path, mountPath string) (map[string]string, error) {
	dataPath := fmt.Sprintf("/v1/%s/data/%s", mountPath, path)
	resp, err := c.request(ctx, http.MethodGet, dataPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse KV secret response")
	}
	return result.Data.Data, nil
}

func (c *VaultClient) DeleteKVSecret(ctx context.Context, path, mountPath string) error {
	dataPath := fmt.Sprintf("/v1/%s/data/%s", mountPath, path)
	_, err := c.request(ctx, http.MethodDelete, dataPath, nil)
	return err
}

// Policy

func (c *VaultClient) CreatePolicy(ctx context.Context, name, policy string) error {
	_, err := c.request(ctx, http.MethodPut, "/v1/sys/policies/acl/"+name, map[string]string{
		"policy": policy,
	})
	return err
}

func (c *VaultClient) GetPolicy(ctx context.Context, name string) (string, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/sys/policies/acl/"+name, nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			Policy string `json:"policy"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse policy response")
	}
	return result.Data.Policy, nil
}

func (c *VaultClient) DeletePolicy(ctx context.Context, name string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/sys/policies/acl/"+name, nil)
	return err
}

// AuthMethod

func (c *VaultClient) EnableAuthMethod(ctx context.Context, mountPath, methodType string, config map[string]string) error {
	body := map[string]interface{}{
		"type": methodType,
	}
	if len(config) > 0 {
		body["config"] = config
	}
	_, err := c.request(ctx, http.MethodPost, "/v1/sys/auth/"+mountPath, body)
	return err
}

func (c *VaultClient) GetAuthMethod(ctx context.Context, mountPath string) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/sys/auth", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse auth methods response")
	}
	for k, v := range result.Data {
		if strings.TrimRight(k, "/") == mountPath {
			return v.(map[string]interface{}), nil
		}
	}
	return nil, fmt.Errorf("auth method %s not found", mountPath)
}

func (c *VaultClient) DisableAuthMethod(ctx context.Context, mountPath string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/sys/auth/"+mountPath, nil)
	return err
}

// Helper to read token from k8s secret and create client from ProviderConfig

func NewClientFromProviderConfig(ctx context.Context, kube client.Client, pc *vaultv1beta1.ProviderConfig) (*VaultClient, error) {
	secret := &corev1.Secret{}
	if err := kube.Get(ctx, client.ObjectKey{
		Name:      pc.Spec.TokenSecretRef.Name,
		Namespace: pc.Spec.TokenSecretRef.Namespace,
	}, secret); err != nil {
		return nil, errors.Wrap(err, "cannot read vault token secret")
	}

	token, ok := secret.Data[pc.Spec.TokenSecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s missing key %s",
			pc.Spec.TokenSecretRef.Namespace,
			pc.Spec.TokenSecretRef.Name,
			pc.Spec.TokenSecretRef.Key)
	}

	insecure := false
	if pc.Spec.InsecureSkipVerify != nil {
		insecure = *pc.Spec.InsecureSkipVerify
	}

	return NewVaultClientFromConfig(pc.Spec.Address, string(token), insecure)
}
