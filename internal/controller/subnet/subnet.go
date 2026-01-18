package subnet

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
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/subnet/v1alpha1"
	apisv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/v1alpha1"
	clients "github.com/peertechde/provider-opentelekomcloud/internal/clients"
	"github.com/peertechde/provider-opentelekomcloud/internal/pointer"
)

const (
	errNotSubnet    = "managed resource is not a Subnet custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errNewClient    = "cannot create new OTC client"
	errObserve      = "cannot observe Subnet"
	errCreate       = "cannot create Subnet"
	errUpdate       = "cannot update Subnet"
	errDelete       = "cannot delete Subnet"
)

// SetupGated adds a controller that reconciles Subnet managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup Subnet controller"))
		}
	}, v1alpha1.SubnetGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles Subnet managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SubnetGroupKind)

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
			&v1alpha1.SubnetList{},
			o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(
				err,
				"cannot register MR state metrics recorder for kind v1alpha1.SubnetList",
			)
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.SubnetGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Subnet{}).
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
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return nil, errors.New(errNotSubnet)
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

// external implements managed.ExternalClient for Subnet resources.
type external struct {
	client *golangsdk.ServiceClient
}

func (e *external) Observe(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSubnet)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	subnet, err := subnets.Get(e.client, externalName).Extract()
	if err != nil {
		var notFound golangsdk.ErrDefault404
		if errors.As(err, &notFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errObserve)
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.SubnetObservation{
		ID:     subnet.ID,
		Status: subnet.Status,
		CIDR:   subnet.CIDR,
		VPCID:  subnet.VpcID,
	}

	// Set conditions based on status
	switch subnet.Status {
	case "ACTIVE", "OK":
		cr.SetConditions(xpv1.Available())
	case "CREATING", "UNKNOWN":
		cr.SetConditions(xpv1.Creating())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}

	lateInitialized := e.detectLateInitialization(&cr.Spec.ForProvider, subnet)
	needsUpdate := e.detectDrift(&cr.Spec.ForProvider, subnet)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: lateInitialized,
	}, nil
}

// detectLateInitialization fills optional Spec fields if they are empty but present at the provider.
func (e *external) detectLateInitialization(
	spec *v1alpha1.SubnetParameters,
	actual *subnets.Subnet,
) bool {
	var initialized bool // false

	if spec.DHCPEnable == nil {
		spec.DHCPEnable = pointer.To(actual.EnableDHCP)
		initialized = true
	}

	if spec.PrimaryDNS == nil && actual.PrimaryDNS != "" {
		spec.PrimaryDNS = pointer.To(actual.PrimaryDNS)
		initialized = true
	}
	if spec.SecondaryDNS == nil && actual.SecondaryDNS != "" {
		spec.SecondaryDNS = pointer.To(actual.SecondaryDNS)
		initialized = true
	}
	if spec.Description == nil && actual.Description != "" {
		spec.Description = pointer.To(actual.Description)
		initialized = true
	}

	return initialized
}

func (e *external) detectDrift(spec *v1alpha1.SubnetParameters, actual *subnets.Subnet) bool {
	if actual.Name != spec.Name {
		return true
	}
	if actual.CIDR != spec.CIDR {
		return true
	}
	if actual.GatewayIP != spec.GatewayIP {
		return true
	}
	if pointer.Deref(spec.Description, actual.Description) != actual.Description {
		return true
	}
	if pointer.Deref(spec.DHCPEnable, actual.EnableDHCP) != actual.EnableDHCP {
		return true
	}
	if pointer.Deref(spec.PrimaryDNS, actual.PrimaryDNS) != actual.PrimaryDNS {
		return true
	}
	if pointer.Deref(spec.SecondaryDNS, actual.SecondaryDNS) != actual.SecondaryDNS {
		return true
	}

	return false
}

func (e *external) Create(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSubnet)
	}

	cr.SetConditions(xpv1.Creating())

	opts := subnets.CreateOpts{
		Name:      cr.Spec.ForProvider.Name,
		CIDR:      cr.Spec.ForProvider.CIDR,
		GatewayIP: cr.Spec.ForProvider.GatewayIP,
		VpcID:     cr.Spec.ForProvider.VPCID,
	}

	if cr.Spec.ForProvider.DHCPEnable != nil {
		opts.EnableDHCP = cr.Spec.ForProvider.DHCPEnable
	}
	if cr.Spec.ForProvider.PrimaryDNS != nil {
		opts.PrimaryDNS = *cr.Spec.ForProvider.PrimaryDNS
	}
	if cr.Spec.ForProvider.SecondaryDNS != nil {
		opts.SecondaryDNS = *cr.Spec.ForProvider.SecondaryDNS
	}
	if cr.Spec.ForProvider.AvailabilityZone != nil {
		opts.AvailabilityZone = *cr.Spec.ForProvider.AvailabilityZone
	}
	if cr.Spec.ForProvider.Description != nil {
		opts.Description = *cr.Spec.ForProvider.Description
	}

	subnet, err := subnets.Create(e.client, opts).Extract()
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	// Set external name to the subnet ID
	meta.SetExternalName(cr, subnet.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSubnet)
	}

	// Verify immutable fields
	if cr.Spec.ForProvider.VPCID != cr.Status.AtProvider.VPCID {
		return managed.ExternalUpdate{}, errors.New("cannot update immutable field: VPCID")
	}
	if cr.Spec.ForProvider.CIDR != cr.Status.AtProvider.CIDR {
		return managed.ExternalUpdate{}, errors.New("cannot update immutable field: CIDR")
	}
	if cr.Spec.ForProvider.GatewayIP != cr.Status.AtProvider.GatewayIP {
		return managed.ExternalUpdate{}, errors.New("cannot update immutable field: GatewayIP")
	}

	externalName := meta.GetExternalName(cr)

	opts := subnets.UpdateOpts{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.DHCPEnable != nil {
		opts.EnableDHCP = cr.Spec.ForProvider.DHCPEnable
	}
	if cr.Spec.ForProvider.PrimaryDNS != nil {
		opts.PrimaryDNS = *cr.Spec.ForProvider.PrimaryDNS
	}
	if cr.Spec.ForProvider.SecondaryDNS != nil {
		opts.SecondaryDNS = *cr.Spec.ForProvider.SecondaryDNS
	}
	if cr.Spec.ForProvider.Description != nil {
		opts.Description = cr.Spec.ForProvider.Description
	}

	_, err := subnets.Update(e.client, cr.Spec.ForProvider.VPCID, externalName, opts).Extract()
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Subnet)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotSubnet)
	}

	cr.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := subnets.Delete(e.client, cr.Spec.ForProvider.VPCID, externalName).ExtractErr()
	if err != nil {
		var notFound golangsdk.ErrDefault404
		if errors.As(err, &notFound) {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, errors.Wrap(err, errDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
