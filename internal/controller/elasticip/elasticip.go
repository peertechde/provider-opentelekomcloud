package elasticip

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/eips"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/elasticip/v1alpha1"
	apisv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/v1alpha1"
	clients "github.com/peertechde/provider-opentelekomcloud/internal/clients"
)

const (
	errNotElasticIP = "managed resource is not a ElasticIP custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errNewClient    = "cannot create new OTC client"
	errObserve      = "cannot observe ElasticIP"
	errCreate       = "cannot create ElasticIP"
	errDelete       = "cannot delete ElasticIP"
	errImmutable    = "ElasticIP is immutable"
)

// SetupGated adds a controller that reconciles ElasticIP managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup ElasticIP controller"))
		}
	}, v1alpha1.ElasticIPGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles ElasticIP managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ElasticIPGroupKind)

	// Initialize the client caching
	clientCache := clients.NewCache(mgr.GetClient())

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(
				mgr.GetClient(),
				&apisv1alpha1.ProviderConfigUsage{},
			),
			clientCache: clientCache,
		}),
		managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(),
			o.Logger,
			o.MetricOptions.MRStateMetrics,
			&v1alpha1.ElasticIPList{},
			o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(
				err,
				"cannot register MR state metrics recorder for kind v1alpha1.ElasticIPList",
			)
		}
	}

	r := managed.NewReconciler(
		mgr,
		resource.ManagedKind(v1alpha1.ElasticIPGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ElasticIP{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube        client.Client
	usage       *resource.ProviderConfigUsageTracker
	clientCache *clients.Cache
}

// Connect creates an ExternalClient using the ProviderConfig credentials.
func (c *connector) Connect(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.ElasticIP)
	if !ok {
		return nil, errors.New(errNotElasticIP)
	}
	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// Get ProviderConfig reference
	m := mg.(resource.ModernManaged)
	ref := m.GetProviderConfigReference()

	var spec apisv1alpha1.ProviderConfigSpec
	var cacheKey string

	switch ref.Kind {
	case "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: m.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, errGetPC)
		}
		spec = pc.Spec
		cacheKey = fmt.Sprintf("ProviderConfig/%s/%s", pc.Namespace, pc.Name)
	case "ClusterProviderConfig":
		cpc := &apisv1alpha1.ClusterProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc); err != nil {
			return nil, errors.Wrap(err, errGetCPC)
		}
		spec = cpc.Spec
		cacheKey = fmt.Sprintf("ClusterProviderConfig/%s", cpc.Name)
	default:
		return nil, errors.Errorf("unsupported provider config kind: %s", ref.Kind)
	}

	// Get authenticated provider client from the cache
	providerClient, err := c.clientCache.GetClient(ctx, cacheKey, spec)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	// Create service specific client
	networkClient, err := providerClient.NewNetworkV1Client()
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: networkClient}, nil
}

// external implements managed.ExternalClient for ElasticIP resources.
type external struct {
	client *golangsdk.ServiceClient
}

func (e *external) Observe(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ElasticIP)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotElasticIP)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	eip, err := eips.Get(e.client, externalName).Extract()
	if err != nil {
		var notFound golangsdk.ErrDefault404
		if errors.As(err, &notFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errObserve)
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.ElasticIPObservation{
		ID:                 eip.ID,
		Status:             eip.Status,
		IPAddress:          eip.PublicAddress,
		PrivateIPAddress:   eip.PrivateAddress,
		PortID:             eip.PortID,
		BandwidthID:        eip.BandwidthID,
		BandwidthSize:      eip.BandwidthSize,
		BandwidthShareType: eip.BandwidthShareType,
	}

	// Set conditions based on status
	switch eip.Status {
	case "ACTIVE", "DOWN": // DOWN means no port attached
		cr.SetConditions(xpv1.Available())
	case "ERROR":
		cr.SetConditions(xpv1.Unavailable())
	default:
		cr.SetConditions(xpv1.Creating())
	}

	needsUpdate := e.detectDrift(&cr.Spec.ForProvider, eip)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: !needsUpdate,
	}, nil
}

func (e *external) detectDrift(cr *v1alpha1.ElasticIPParameters, eip *eips.PublicIp) bool {
	return false
}

func (e *external) Create(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ElasticIP)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotElasticIP)
	}

	cr.SetConditions(xpv1.Creating())

	var shareType string
	if cr.Spec.ForProvider.Bandwidth.ShareType == "BGP" {
		shareType = "5_bgp"
	} else {
		shareType = "5_mailbgp"
	}

	bw := eips.BandwidthOpts{
		Size:      cr.Spec.ForProvider.Bandwidth.Size,
		ShareType: shareType,
	}

	pubIP := eips.PublicIpOpts{
		Type: cr.Spec.ForProvider.PublicIP.Type,
	}
	if cr.Spec.ForProvider.PublicIP.IPAddress != nil {
		pubIP.Address = *cr.Spec.ForProvider.PublicIP.IPAddress
	}

	opts := eips.ApplyOpts{
		IP:        pubIP,
		Bandwidth: bw,
	}

	eip, err := eips.Apply(e.client, opts).Extract()
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	// Set external name to the elasticip ID
	meta.SetExternalName(cr, eip.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalUpdate, error) {
	// Elastic IPs are immutable. Any detected drift requires recreation, so we
	// return an error.
	return managed.ExternalUpdate{}, errors.New(errImmutable)
}

func (e *external) Delete(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ElasticIP)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotElasticIP)
	}

	cr.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := eips.Delete(e.client, externalName).ExtractErr()
	if err != nil {
		if errors.Is(err, golangsdk.ErrDefault404{}) {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, errDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
