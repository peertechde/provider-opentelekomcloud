package clients

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/peertechde/provider-opentelekomcloud/apis/v1alpha1"
)

const (
	// DefaultIdentityEndpoint is the default OTC identity endpoint.
	DefaultIdentityEndpoint = "https://iam.eu-de.otc.t-systems.com/v3"
)

// Credentials represents AK/SK credentials.
type Credentials struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type Client struct {
	ProviderClient *golangsdk.ProviderClient
	Region         string
}

// NewNetworkV1Client creates a client for VPC/Subnet service.
func (c *Client) NewNetworkV1Client() (*golangsdk.ServiceClient, error) {
	return openstack.NewNetworkV1(c.ProviderClient, golangsdk.EndpointOpts{
		Region: c.Region,
	})
}

// NewVPCV3Client creates a client for VPC V3 service.
func (c *Client) NewVPCV3Client() (*golangsdk.ServiceClient, error) {
	return openstack.NewVpcV3(c.ProviderClient, golangsdk.EndpointOpts{
		Region: c.Region,
	})
}

// session holds an active connection and metadata.
type session struct {
	client    *golangsdk.ProviderClient
	expiresAt time.Time
	hash      string
}

// Cache manages the lifecycle of OTC connections to prevent rate limiting.
type Cache struct {
	mu       sync.RWMutex
	sessions map[string]*session // Key: ProviderConfig Name
	client   client.Client
}

// NewCache creates a new cache. Call this in your Controller Setup.
func NewCache(kube client.Client) *Cache {
	return &Cache{
		sessions: make(map[string]*session),
		client:   kube,
	}
}

// GetClient returns a cached client or creates a new one.
func (c *Cache) GetClient(
	ctx context.Context,
	key string,
	spec v1alpha1.ProviderConfigSpec,
) (*Client, error) {
	// Resolve credentials (AK/SK) from secret
	creds, err := extractCredentials(ctx, c.client, spec)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract credentials")
	}

	// If the secret changes (new key) or spec changes.
	configHash := calculateHash(spec, creds)

	// Check the cache
	c.mu.RLock()
	cached, ok := c.sessions[key]
	c.mu.RUnlock()

	if ok && cached.hash == configHash && time.Now().Add(5*time.Minute).Before(cached.expiresAt) {
		return &Client{
			ProviderClient: cached.client,
			Region:         spec.Region,
		}, nil
	}

	// Create a new provider client
	endpoint := DefaultIdentityEndpoint
	if spec.IdentityEndpoint != nil && *spec.IdentityEndpoint != "" {
		endpoint = *spec.IdentityEndpoint
	}

	authOpts := golangsdk.AKSKAuthOptions{
		IdentityEndpoint: endpoint,
		AccessKey:        creds.AccessKey,
		SecretKey:        creds.SecretKey,
		ProjectId:        spec.ProjectID,
		Region:           spec.Region,
	}

	providerClient, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot authenticate with Open Telekom Cloud")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Cache for 23 hours (tokens usually last 24h)
	c.sessions[key] = &session{
		client:    providerClient,
		expiresAt: time.Now().Add(23 * time.Hour),
		hash:      configHash,
	}

	return &Client{
		ProviderClient: providerClient,
		Region:         spec.Region,
	}, nil
}

// extractCredentials extracts AK/SK credentials from the configured source.
func extractCredentials(
	ctx context.Context,
	kube client.Client,
	spec v1alpha1.ProviderConfigSpec,
) (*Credentials, error) {
	if spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		return nil, errors.Errorf("unsupported credentials source: %s", spec.Credentials.Source)
	}

	ref := spec.Credentials.SecretRef
	if ref == nil {
		return nil, errors.New("secretRef is required")
	}

	// Use Crossplane helper to fetch secret data
	data, err := resource.CommonCredentialExtractor(
		ctx,
		spec.Credentials.Source,
		kube,
		spec.Credentials.CommonCredentialSelectors,
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get credentials from secret")
	}

	// Unmarshal JSON into AK/SK struct
	creds := &Credentials{}
	if err := json.Unmarshal(data, creds); err != nil {
		return nil, errors.Wrap(
			err,
			"cannot unmarshal credentials JSON, expect keys: accessKey, secretKey",
		)
	}

	if creds.AccessKey == "" || creds.SecretKey == "" {
		return nil, errors.New("accessKey and secretKey are required in credentials secret")
	}

	return creds, nil
}

func calculateHash(spec v1alpha1.ProviderConfigSpec, creds *Credentials) string {
	// Concatenate fields that affect authentication identity
	s := fmt.Sprintf("%s|%s|%s|%s|%s",
		spec.DomainName,
		spec.ProjectID,
		spec.Region,
		creds.AccessKey,
		creds.SecretKey,
	)
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
