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

func NewVaultClientFromConfig(baseURL string, token string, tlsInsecure bool) (*VaultClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse vault address")
	}
	return &VaultClient{
		baseURL: u,
		httpClient: &http.Client{
			Transport: &http.Transport{
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

// Mount

func (c *VaultClient) EnableMount(ctx context.Context, path, engineType, description string, defaultLeaseTTL, maxLeaseTTL int, options map[string]string, config map[string]string) error {
	body := map[string]interface{}{
		"type": engineType,
	}
	if description != "" {
		body["description"] = description
	}
	if defaultLeaseTTL > 0 {
		body["default_lease_ttl"] = defaultLeaseTTL
	}
	if maxLeaseTTL > 0 {
		body["max_lease_ttl"] = maxLeaseTTL
	}
	if len(options) > 0 {
		body["options"] = options
	}
	if len(config) > 0 {
		body["config"] = config
	}
	_, err := c.request(ctx, http.MethodPost, "/v1/sys/mounts/"+path, body)
	return err
}

func (c *VaultClient) GetMount(ctx context.Context, path string) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/sys/mounts/"+path, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse mount response")
	}
	return result.Data, nil
}

func (c *VaultClient) DisableMount(ctx context.Context, path string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/sys/mounts/"+path, nil)
	return err
}

// SecretBackendRole

func (c *VaultClient) CreateSecretBackendRole(ctx context.Context, backend, name string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetSecretBackendRole(ctx context.Context, backend, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse secret backend role response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteSecretBackendRole(ctx context.Context, backend, name string) error {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// AuthBackendRole

func (c *VaultClient) CreateAuthBackendRole(ctx context.Context, backend, roleName string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s", backend, roleName)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetAuthBackendRole(ctx context.Context, backend, roleName string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s", backend, roleName)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse auth backend role response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteAuthBackendRole(ctx context.Context, backend, roleName string) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s", backend, roleName)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// Helper to read token from k8s secret and create client from ProviderConfig

type Config struct {
	Address  string
	Token    string
	Insecure bool
}

func GetConfig(ctx context.Context, kube client.Client, pc *vaultv1beta1.ProviderConfig) (*Config, error) {
	if pc.Spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		return nil, fmt.Errorf("unsupported credentials source: %s", pc.Spec.Credentials.Source)
	}
	if pc.Spec.Credentials.SecretRef == nil {
		return nil, fmt.Errorf("secretRef is required when credentials source is Secret")
	}

	secret := &corev1.Secret{}
	if err := kube.Get(ctx, client.ObjectKey{
		Name:      pc.Spec.Credentials.SecretRef.Name,
		Namespace: pc.Spec.Credentials.SecretRef.Namespace,
	}, secret); err != nil {
		return nil, errors.Wrap(err, "cannot read vault token secret")
	}

	token, ok := secret.Data[pc.Spec.Credentials.SecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s missing key %s",
			pc.Spec.Credentials.SecretRef.Namespace,
			pc.Spec.Credentials.SecretRef.Name,
			pc.Spec.Credentials.SecretRef.Key)
	}

	insecure := false
	if pc.Spec.InsecureSkipVerify != nil {
		insecure = *pc.Spec.InsecureSkipVerify
	}

	return &Config{
		Address:  pc.Spec.Address,
		Token:    string(token),
		Insecure: insecure,
	}, nil
}

func NewClientFromProviderConfig(ctx context.Context, kube client.Client, pc *vaultv1beta1.ProviderConfig) (*VaultClient, error) {
	cfg, err := GetConfig(ctx, kube, pc)
	if err != nil {
		return nil, err
	}
	return NewVaultClientFromConfig(cfg.Address, cfg.Token, cfg.Insecure)
}
