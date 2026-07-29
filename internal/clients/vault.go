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

// PKIConfig

func (c *VaultClient) GenerateRootCA(ctx context.Context, backend, exportType, commonName, ttl string, params map[string]interface{}) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/root/generate/%s", backend, exportType)
	req := map[string]interface{}{
		"common_name": commonName,
	}
	if ttl != "" {
		req["ttl"] = ttl
	}
	for k, v := range params {
		req[k] = v
	}
	resp, err := c.request(ctx, http.MethodPost, apiPath, req)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse root CA response")
	}
	return result.Data, nil
}

func (c *VaultClient) GetPKICA(ctx context.Context, backend string) (string, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/"+backend+"/ca/pem", nil)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (c *VaultClient) GetPKICert(ctx context.Context, backend, serial string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/cert/%s", backend, serial)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse PKI cert response")
	}
	return result.Data, nil
}

func (c *VaultClient) ConfigurePKIURLs(ctx context.Context, backend string, issuingCerts, crlDps, ocspServers []string) error {
	apiPath := fmt.Sprintf("/v1/%s/config/urls", backend)
	req := make(map[string]interface{})
	if len(issuingCerts) > 0 {
		req["issuing_certificates"] = issuingCerts
	}
	if len(crlDps) > 0 {
		req["crl_distribution_points"] = crlDps
	}
	if len(ocspServers) > 0 {
		req["ocsp_servers"] = ocspServers
	}
	_, err := c.request(ctx, http.MethodPost, apiPath, req)
	return err
}

// Certificate

func (c *VaultClient) IssueCertificate(ctx context.Context, backend, role, commonName string, params map[string]interface{}) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/issue/%s", backend, role)
	req := map[string]interface{}{
		"common_name": commonName,
	}
	for k, v := range params {
		req[k] = v
	}
	resp, err := c.request(ctx, http.MethodPost, apiPath, req)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse issue certificate response")
	}
	return result.Data, nil
}

func (c *VaultClient) RevokeCertificate(ctx context.Context, backend, serial string) error {
	apiPath := fmt.Sprintf("/v1/%s/revoke", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, map[string]string{
		"serial_number": serial,
	})
	return err
}

// DatabaseRole

func (c *VaultClient) CreateDatabaseRole(ctx context.Context, backend, name string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetDatabaseRole(ctx context.Context, backend, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse database role response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteDatabaseRole(ctx context.Context, backend, name string) error {
	apiPath := fmt.Sprintf("/v1/%s/roles/%s", backend, name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// TransitKey

func (c *VaultClient) CreateTransitKey(ctx context.Context, backend, name string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/keys/%s", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetTransitKey(ctx context.Context, backend, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/keys/%s", backend, name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse transit key response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteTransitKey(ctx context.Context, backend, name string) error {
	apiPath := fmt.Sprintf("/v1/%s/keys/%s", backend, name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// Token

func (c *VaultClient) CreateToken(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodPost, "/v1/auth/token/create", params)
	if err != nil {
		return nil, err
	}
	var result struct {
		Auth map[string]interface{} `json:"auth"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse create token response")
	}
	return result.Auth, nil
}

func (c *VaultClient) LookupToken(ctx context.Context, accessor string) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodPost, "/v1/auth/token/lookup-accessor", map[string]string{
		"accessor": accessor,
	})
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse lookup token response")
	}
	return result.Data, nil
}

func (c *VaultClient) RenewToken(ctx context.Context, accessor string, increment int) error {
	_, err := c.request(ctx, http.MethodPost, "/v1/auth/token/renew-accessor", map[string]interface{}{
		"accessor":  accessor,
		"increment": increment,
	})
	return err
}

func (c *VaultClient) RevokeToken(ctx context.Context, accessor string) error {
	_, err := c.request(ctx, http.MethodPost, "/v1/auth/token/revoke-accessor", map[string]string{
		"accessor": accessor,
	})
	return err
}

// IdentityEntity

func (c *VaultClient) CreateIdentityEntity(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodPost, "/v1/identity/entity", params)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse create entity response")
	}
	return result.Data, nil
}

func (c *VaultClient) GetIdentityEntity(ctx context.Context, name string) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/identity/entity/name/"+name, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse get entity response")
	}
	return result.Data, nil
}

func (c *VaultClient) UpdateIdentityEntity(ctx context.Context, name string, params map[string]interface{}) error {
	_, err := c.request(ctx, http.MethodPost, "/v1/identity/entity/name/"+name, params)
	return err
}

func (c *VaultClient) DeleteIdentityEntity(ctx context.Context, name string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/identity/entity/name/"+name, nil)
	return err
}

// IdentityGroup

func (c *VaultClient) CreateIdentityGroup(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodPost, "/v1/identity/group", params)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse create group response")
	}
	return result.Data, nil
}

func (c *VaultClient) GetIdentityGroup(ctx context.Context, name string) (map[string]interface{}, error) {
	resp, err := c.request(ctx, http.MethodGet, "/v1/identity/group/name/"+name, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse get group response")
	}
	return result.Data, nil
}

func (c *VaultClient) UpdateIdentityGroup(ctx context.Context, name string, params map[string]interface{}) error {
	_, err := c.request(ctx, http.MethodPost, "/v1/identity/group/name/"+name, params)
	return err
}

func (c *VaultClient) DeleteIdentityGroup(ctx context.Context, name string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/identity/group/name/"+name, nil)
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
