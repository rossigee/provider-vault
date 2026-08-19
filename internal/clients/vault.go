package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/version"
)

const vaultTokenHeader = "X-Vault-Token" // #nosec G101 -- HTTP header name, not a credential

// errNotFound is returned when the Vault API returns a 404 for a request.
type errNotFound struct{ error }

func (e *errNotFound) NotFound() bool { return true }

// IsNotFound returns true if the supplied error indicates that the requested
// Vault resource does not exist.
func IsNotFound(err error) bool {
	nf, ok := err.(interface {
		NotFound() bool
	})
	return ok && nf.NotFound()
}

type VaultClient struct {
	baseURL        *url.URL
	httpClient     *http.Client
	token          string
	vaultNamespace string
}

func NewVaultClientFromConfig(baseURL string, token string, tlsInsecure bool, caCertPEM, clientCertPEM, clientKeyPEM []byte) (*VaultClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse vault address")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsInsecure, // #nosec G402 -- gated by provider-config tlsInsecure flag; CA/client certs still honored when set
		NextProtos:         []string{"http/1.1"},
	}

	if len(caCertPEM) > 0 {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCertPEM) {
			return nil, errors.New("failed to parse CA certificate PEM")
		}
		tlsConfig.RootCAs = caPool
	}

	if len(clientCertPEM) > 0 && len(clientKeyPEM) > 0 {
		cert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse TLS client cert/key")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return &VaultClient{
		baseURL: u,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
				IdleConnTimeout: 30 * time.Second,
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
	req.Header.Set("User-Agent", "provider-vault/"+version.Version)
	if c.vaultNamespace != "" {
		req.Header.Set("X-Vault-Namespace", c.vaultNamespace)
	}

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
		if resp.StatusCode == http.StatusNotFound {
			return nil, &errNotFound{fmt.Errorf("vault API returned status %d: %s", resp.StatusCode, string(respBody))}
		}
		return nil, fmt.Errorf("vault API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// KV v2

func kvDataPath(mountPath, path string, version *int) string {
	if version != nil && *version == 1 {
		return fmt.Sprintf("/v1/%s/%s", mountPath, path)
	}
	return fmt.Sprintf("/v1/%s/data/%s", mountPath, path)
}

func (c *VaultClient) writeKVSecret(ctx context.Context, path, mountPath string, data map[string]interface{}, version *int) error {
	dataPath := kvDataPath(mountPath, path, version)
	if version != nil && *version == 1 {
		_, err := c.request(ctx, http.MethodPost, dataPath, data)
		return err
	}
	_, err := c.request(ctx, http.MethodPost, dataPath, map[string]interface{}{
		"data": data,
	})
	return err
}

func (c *VaultClient) CreateKVSecret(ctx context.Context, path, mountPath string, data map[string]string, version *int) error {
	d := make(map[string]interface{}, len(data))
	for k, v := range data {
		d[k] = v
	}
	return c.writeKVSecret(ctx, path, mountPath, d, version)
}

// UpdateKVSecret converges the Vault secret to the desired data set. Keys that
// are present in Vault but absent from desired are deleted by writing null, as
// per the KV v2 API.
func (c *VaultClient) UpdateKVSecret(ctx context.Context, path, mountPath string, desired map[string]string, version *int) error {
	existing, err := c.GetKVSecret(ctx, path, mountPath, version)
	if err != nil {
		if !IsNotFound(err) {
			return err
		}
		existing = map[string]string{}
	}
	d := make(map[string]interface{}, len(desired)+len(existing))
	for k, v := range desired {
		d[k] = v
	}
	for k := range existing {
		if _, ok := desired[k]; !ok {
			d[k] = nil
		}
	}
	return c.writeKVSecret(ctx, path, mountPath, d, version)
}

func (c *VaultClient) GetKVSecret(ctx context.Context, path, mountPath string, version *int) (map[string]string, error) {
	dataPath := kvDataPath(mountPath, path, version)
	resp, err := c.request(ctx, http.MethodGet, dataPath, nil)
	if err != nil {
		return nil, err
	}
	if version != nil && *version == 1 {
		var result struct {
			Data map[string]string `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, errors.Wrap(err, "failed to parse KV v1 secret response")
		}
		return result.Data, nil
	}
	var result struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse KV v2 secret response")
	}
	return result.Data.Data, nil
}

func (c *VaultClient) DeleteKVSecret(ctx context.Context, path, mountPath string, version *int) error {
	dataPath := kvDataPath(mountPath, path, version)
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
	return nil, &errNotFound{fmt.Errorf("auth method %s not found", mountPath)}
}

func (c *VaultClient) DisableAuthMethod(ctx context.Context, mountPath string) error {
	_, err := c.request(ctx, http.MethodDelete, "/v1/sys/auth/"+mountPath, nil)
	return err
}

// Quota

func (c *VaultClient) CreateRateQuota(ctx context.Context, name, path string, rate string, interval string, blocked []string) error {
	body := map[string]interface{}{
		"rate":     rate,
		"interval": interval,
	}
	if path != "" {
		body["path"] = path
	}
	if len(blocked) > 0 {
		body["blocked"] = blocked
	}
	apiPath := fmt.Sprintf("/v1/sys/quotas/rate/%s", name)
	_, err := c.request(ctx, http.MethodPut, apiPath, body)
	return err
}

func (c *VaultClient) CreateLeaseQuota(ctx context.Context, name, path string, maxLeases int, blocked []string) error {
	body := map[string]interface{}{
		"max_leases": maxLeases,
	}
	if path != "" {
		body["path"] = path
	}
	if len(blocked) > 0 {
		body["blocked"] = blocked
	}
	apiPath := fmt.Sprintf("/v1/sys/quotas/lease/%s", name)
	_, err := c.request(ctx, http.MethodPut, apiPath, body)
	return err
}

func (c *VaultClient) GetQuota(ctx context.Context, quotaType, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/sys/quotas/%s/%s", quotaType, name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse quota response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteQuota(ctx context.Context, quotaType, name string) error {
	apiPath := fmt.Sprintf("/v1/sys/quotas/%s/%s", quotaType, name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// Namespace

func (c *VaultClient) CreateNamespace(ctx context.Context, name, description string) error {
	body := map[string]interface{}{}
	if description != "" {
		body["description"] = description
	}
	apiPath := fmt.Sprintf("/v1/sys/namespaces/%s", name)
	_, err := c.request(ctx, http.MethodPost, apiPath, body)
	return err
}

func (c *VaultClient) GetNamespace(ctx context.Context, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/sys/namespaces/%s", name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse namespace response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteNamespace(ctx context.Context, name string) error {
	apiPath := fmt.Sprintf("/v1/sys/namespaces/%s", name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// Lease

type LeaseInfo struct {
	LeaseID       string `json:"lease_id"`
	Renewable     bool   `json:"renewable"`
	LeaseDuration int    `json:"lease_duration"`
}

func (c *VaultClient) RenewLease(ctx context.Context, leaseID string, increment *int) (*LeaseInfo, error) {
	body := map[string]interface{}{
		"lease_id": leaseID,
	}
	if increment != nil {
		body["increment"] = *increment
	}
	resp, err := c.request(ctx, http.MethodPut, "/v1/sys/leases/renew", body)
	if err != nil {
		return nil, err
	}
	var result struct {
		LeaseID       string `json:"lease_id"`
		Renewable     bool   `json:"renewable"`
		LeaseDuration int    `json:"lease_duration"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse lease renewal response")
	}
	return &LeaseInfo{
		LeaseID:       result.LeaseID,
		Renewable:     result.Renewable,
		LeaseDuration: result.LeaseDuration,
	}, nil
}

func (c *VaultClient) RevokeLease(ctx context.Context, leaseID string) error {
	body := map[string]interface{}{
		"lease_id": leaseID,
	}
	_, err := c.request(ctx, http.MethodPut, "/v1/sys/leases/revoke", body)
	return err
}

func (c *VaultClient) LookupLease(ctx context.Context, leaseID string) (*LeaseInfo, error) {
	body := map[string]interface{}{
		"lease_id": leaseID,
	}
	resp, err := c.request(ctx, http.MethodPut, "/v1/sys/leases/lookup", body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data struct {
			LeaseID       string `json:"lease_id"`
			Renewable     bool   `json:"renewable"`
			LeaseDuration int    `json:"lease_duration"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse lease lookup response")
	}
	return &LeaseInfo{
		LeaseID:       result.Data.LeaseID,
		Renewable:     result.Data.Renewable,
		LeaseDuration: result.Data.LeaseDuration,
	}, nil
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
	// Vault returns HTTP 204 No Content (empty body) when the PKI mount has no
	// default issuer configured. Treat that as NotFound so the reconciler
	// generates a root CA rather than adopting an empty certificate.
	if len(resp) == 0 {
		return "", &errNotFound{fmt.Errorf("no issuer configured for PKI backend %s", backend)}
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

// GetPKIURLs returns the PKI URL configuration for the supplied backend.
func (c *VaultClient) GetPKIURLs(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/config/urls", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse PKI URLs response")
	}
	return result.Data, nil
}

// ConfigurePKICRL writes the CRL/OCSP configuration for the supplied PKI
// backend.
func (c *VaultClient) ConfigurePKICRL(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/config/crl", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

// GetPKICRLConfig returns the CRL/OCSP configuration for the supplied PKI
// backend.
func (c *VaultClient) GetPKICRLConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/config/crl", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse PKI CRL config response")
	}
	return result.Data, nil
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

// ConfigureTransitKey updates the configurable properties of an existing
// transit key.
func (c *VaultClient) ConfigureTransitKey(ctx context.Context, backend, name string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/keys/%s/config", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

// RotateTransitKey rotates the version of the supplied transit key. Each call
// increments the key's latest version by one.
func (c *VaultClient) RotateTransitKey(ctx context.Context, backend, name string) error {
	apiPath := fmt.Sprintf("/v1/%s/keys/%s/rotate", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, nil)
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

// DatabaseBackend

func (c *VaultClient) CreateDatabaseBackendConfig(ctx context.Context, backend, name string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/%s/config/%s", backend, name)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetDatabaseBackendConfig(ctx context.Context, backend, name string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/%s/config/%s", backend, name)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse database backend config response")
	}
	return result.Data, nil
}

func (c *VaultClient) DeleteDatabaseBackendConfig(ctx context.Context, backend, name string) error {
	apiPath := fmt.Sprintf("/v1/%s/config/%s", backend, name)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// KubernetesAuthConfig

func (c *VaultClient) ConfigureKubernetesAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetKubernetesAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse kubernetes auth config response")
	}
	return result.Data, nil
}

// JWTAuthConfig

func (c *VaultClient) ConfigureJWTAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetJWTAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse JWT auth config response")
	}
	return result.Data, nil
}

// AuditDevice

func (c *VaultClient) EnableAuditDevice(ctx context.Context, path, methodType, description string, local bool, options map[string]string) error {
	body := map[string]interface{}{
		"type":        methodType,
		"description": description,
		"local":       local,
	}
	if len(options) > 0 {
		body["options"] = options
	}
	apiPath := fmt.Sprintf("/v1/sys/audit/%s", path)
	_, err := c.request(ctx, http.MethodPut, apiPath, body)
	return err
}

func (c *VaultClient) GetAuditDevice(ctx context.Context, path string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/sys/audit/%s", path)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse audit device response")
	}
	return result.Data, nil
}

func (c *VaultClient) DisableAuditDevice(ctx context.Context, path string) error {
	apiPath := fmt.Sprintf("/v1/sys/audit/%s", path)
	_, err := c.request(ctx, http.MethodDelete, apiPath, nil)
	return err
}

// LDAPAuthConfig

func (c *VaultClient) ConfigureLDAPAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetLDAPAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse LDAP auth config response")
	}
	return result.Data, nil
}

// AWSAuthConfig

func (c *VaultClient) ConfigureAWSAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetAWSAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse AWS auth config response")
	}
	return result.Data, nil
}

// AzureAuthConfig

func (c *VaultClient) ConfigureAzureAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetAzureAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse Azure auth config response")
	}
	return result.Data, nil
}

// GCPAuthConfig

func (c *VaultClient) ConfigureGCPAuth(ctx context.Context, backend string, params map[string]interface{}) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	_, err := c.request(ctx, http.MethodPost, apiPath, params)
	return err
}

func (c *VaultClient) GetGCPAuthConfig(ctx context.Context, backend string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/config", backend)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse GCP auth config response")
	}
	return result.Data, nil
}

// AppRoleSecretID

func (c *VaultClient) GenerateAppRoleSecretID(ctx context.Context, backend, roleName string, params map[string]interface{}) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s/secret-id", backend, roleName)
	resp, err := c.request(ctx, http.MethodPost, apiPath, params)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse secret-id response")
	}
	return result.Data, nil
}

func (c *VaultClient) LookupAppRoleSecretID(ctx context.Context, backend, roleName, accessor string) (map[string]interface{}, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s/secret-id-accessor/lookup", backend, roleName)
	resp, err := c.request(ctx, http.MethodPost, apiPath, map[string]string{
		"secret_id_accessor": accessor,
	})
	if err != nil {
		return nil, err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse secret-id lookup response")
	}
	return result.Data, nil
}

func (c *VaultClient) DestroyAppRoleSecretID(ctx context.Context, backend, roleName, secretID string) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s/secret-id/destroy", backend, roleName)
	_, err := c.request(ctx, http.MethodPost, apiPath, map[string]string{
		"secret_id": secretID,
	})
	return err
}

func (c *VaultClient) DestroyAppRoleSecretIDByAccessor(ctx context.Context, backend, roleName, accessor string) error {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s/secret-id-accessor/destroy", backend, roleName)
	_, err := c.request(ctx, http.MethodPost, apiPath, map[string]string{
		"secret_id_accessor": accessor,
	})
	return err
}

// ReadAppRoleRoleID returns the role-id for the supplied AppRole. The role-id
// is the public identifier used together with a SecretID to authenticate
// against the AppRole auth method.
func (c *VaultClient) ReadAppRoleRoleID(ctx context.Context, backend, roleName string) (string, error) {
	apiPath := fmt.Sprintf("/v1/auth/%s/role/%s/role-id", backend, roleName)
	resp, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", errors.Wrap(err, "failed to parse role-id response")
	}
	roleID, _ := result.Data["role_id"].(string)
	return roleID, nil
}

// Helper to read token from k8s secret and create client from ProviderConfig

type Config struct {
	Address        string
	Token          string
	Insecure       bool
	CACertPEM      []byte
	ClientCertPEM  []byte
	ClientKeyPEM   []byte
	VaultNamespace string
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

	raw, ok := secret.Data[pc.Spec.Credentials.SecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s missing key %s",
			pc.Spec.Credentials.SecretRef.Namespace,
			pc.Spec.Credentials.SecretRef.Name,
			pc.Spec.Credentials.SecretRef.Key)
	}

	token := strings.TrimSpace(string(raw))
	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(token), &parsed); err == nil && parsed.Token != "" {
		token = parsed.Token
	}

	insecure := false
	if pc.Spec.InsecureSkipVerify != nil {
		insecure = *pc.Spec.InsecureSkipVerify
	}

	cfg := &Config{
		Address:  pc.Spec.Address,
		Token:    token,
		Insecure: insecure,
	}

	if pc.Spec.VaultNamespace != nil {
		cfg.VaultNamespace = *pc.Spec.VaultNamespace
	}

	if pc.Spec.TLS != nil {
		if pc.Spec.TLS.CACertSecretRef != nil {
			caRef := pc.Spec.TLS.CACertSecretRef
			caSecret := &corev1.Secret{}
			if err := kube.Get(ctx, client.ObjectKey{
				Name:      caRef.Name,
				Namespace: caRef.Namespace,
			}, caSecret); err != nil {
				return nil, errors.Wrap(err, "cannot read CA certificate secret")
			}
			cfg.CACertPEM = caSecret.Data[caRef.Key]
		}

		if pc.Spec.TLS.ClientCertSecretRef != nil {
			ccRef := pc.Spec.TLS.ClientCertSecretRef
			ccSecret := &corev1.Secret{}
			if err := kube.Get(ctx, client.ObjectKey{
				Name:      ccRef.Name,
				Namespace: ccRef.Namespace,
			}, ccSecret); err != nil {
				return nil, errors.Wrap(err, "cannot read TLS client cert secret")
			}
			cfg.ClientCertPEM = ccSecret.Data["tls.crt"]
			cfg.ClientKeyPEM = ccSecret.Data["tls.key"]
		}
	}

	return cfg, nil
}

func (c *Config) NewClient() (*VaultClient, error) {
	vc, err := NewVaultClientFromConfig(c.Address, c.Token, c.Insecure, c.CACertPEM, c.ClientCertPEM, c.ClientKeyPEM)
	if err != nil {
		return nil, err
	}
	vc.vaultNamespace = c.VaultNamespace
	return vc, nil
}

func NewClientFromProviderConfig(ctx context.Context, kube client.Client, pc *vaultv1beta1.ProviderConfig) (*VaultClient, error) {
	cfg, err := GetConfig(ctx, kube, pc)
	if err != nil {
		return nil, err
	}
	return cfg.NewClient()
}
