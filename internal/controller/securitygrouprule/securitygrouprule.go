package securitygrouprule

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
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/vpc/v3/security/rules"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/securitygrouprule/v1alpha1"
	apisv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/v1alpha1"
	clients "github.com/peertechde/provider-opentelekomcloud/internal/clients"
	"github.com/peertechde/provider-opentelekomcloud/internal/pointer"
)

const (
	errNotSecurityGroupRule = "managed resource is not a SecurityGroupRule custom resource"
	errTrackPCUsage         = "cannot track ProviderConfig usage"
	errGetPC                = "cannot get ProviderConfig"
	errGetCPC               = "cannot get ClusterProviderConfig"
	errNewClient            = "cannot create new OTC client"
	errObserve              = "cannot observe SecurityGroupRule"
	errCreate               = "cannot create SecurityGroupRule"
	errDelete               = "cannot delete SecurityGroupRule"
)

// SetupGated adds a controller that reconciles SecurityGroupRule managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(errors.Wrap(err, "cannot setup SecurityGroupRule controller"))
		}
	}, v1alpha1.SecurityGroupRuleGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles SecurityGroupRule managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SecurityGroupRuleGroupKind)

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
			&v1alpha1.SecurityGroupRuleList{},
			o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(
				err,
				"cannot register MR state metrics recorder for kind v1alpha1.SecurityGroupRuleList",
			)
		}
	}

	r := managed.NewReconciler(
		mgr,
		resource.ManagedKind(v1alpha1.SecurityGroupRuleGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.SecurityGroupRule{}).
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
	cr, ok := mg.(*v1alpha1.SecurityGroupRule)
	if !ok {
		return nil, errors.New(errNotSecurityGroupRule)
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
	vpcClient, err := providerClient.NewVPCV3Client()
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: vpcClient}, nil
}

// external implements managed.ExternalClient for SecurityGroupRule resources.
type external struct {
	client *golangsdk.ServiceClient
}

func (e *external) Observe(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.SecurityGroupRule)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSecurityGroupRule)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	rule, err := rules.Get(e.client, externalName)
	if err != nil {
		var notFound golangsdk.ErrDefault404
		if errors.As(err, &notFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errObserve)
	}

	// Update observed state
	cr.Status.AtProvider = v1alpha1.SecurityGroupRuleObservation{
		ID: rule.ID,
	}

	// Set conditions
	cr.SetConditions(xpv1.Available())

	lateInitialized := e.detectLateInitialization(&cr.Spec.ForProvider, rule)
	needsUpdate := e.detectDrift(&cr.Spec.ForProvider, rule)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        !needsUpdate,
		ResourceLateInitialized: lateInitialized,
	}, nil
}

// detectLateInitialization fills optional Spec fields if they are empty but present at the provider.
//
//nolint:gocyclo
func (e *external) detectLateInitialization(
	spec *v1alpha1.SecurityGroupRuleParameters,
	actual *rules.SecurityGroupRule,
) bool {
	var initialized bool

	if spec.Description == nil && actual.Description != "" {
		spec.Description = pointer.To(actual.Description)
		initialized = true
	}
	if spec.Ethertype == nil && actual.Ethertype != "" {
		spec.Ethertype = pointer.To(actual.Ethertype)
		initialized = true
	}
	if spec.Protocol == nil && actual.Protocol != "" {
		spec.Protocol = pointer.To(actual.Protocol)
		initialized = true
	}
	if spec.Multiport == nil && actual.Multiport != "" {
		spec.Multiport = pointer.To(actual.Multiport)
		initialized = true
	}
	if spec.RemoteIPPrefix == nil && actual.RemoteIPPrefix != "" {
		spec.RemoteIPPrefix = pointer.To(actual.RemoteIPPrefix)
		initialized = true
	}
	if spec.RemoteGroupID == nil && actual.RemoteGroupID != "" {
		spec.RemoteGroupID = pointer.To(actual.RemoteGroupID)
		initialized = true
	}
	if spec.RemoteAddressGroupID == nil && actual.RemoteAddressGroupID != "" {
		spec.RemoteAddressGroupID = pointer.To(actual.RemoteAddressGroupID)
		initialized = true
	}
	if spec.Action == nil && actual.Action != "" {
		spec.Action = pointer.To(actual.Action)
		initialized = true
	}
	if spec.Priority == nil {
		spec.Priority = pointer.To(actual.Priority)
		initialized = true
	}

	return initialized
}

//nolint:gocyclo
func (e *external) detectDrift(
	spec *v1alpha1.SecurityGroupRuleParameters,
	actual *rules.SecurityGroupRule,
) bool {
	if actual.SecurityGroupID != spec.SecurityGroupID {
		return true
	}
	if actual.Direction != spec.Direction {
		return true
	}
	if pointer.Deref(spec.Description, actual.Description) != actual.Description {
		return true
	}
	if pointer.Deref(spec.Ethertype, actual.Ethertype) != actual.Ethertype {
		return true
	}
	if pointer.Deref(spec.Protocol, actual.Protocol) != actual.Protocol {
		return true
	}
	if pointer.Deref(spec.Multiport, actual.Multiport) != actual.Multiport {
		return true
	}
	if pointer.Deref(spec.RemoteIPPrefix, actual.RemoteIPPrefix) != actual.RemoteIPPrefix {
		return true
	}
	if pointer.Deref(spec.RemoteGroupID, actual.RemoteGroupID) != actual.RemoteGroupID {
		return true
	}
	if pointer.Deref(
		spec.RemoteAddressGroupID,
		actual.RemoteAddressGroupID,
	) != actual.RemoteAddressGroupID {
		return true
	}
	if pointer.Deref(spec.Action, actual.Action) != actual.Action {
		return true
	}
	if pointer.Deref(spec.Priority, actual.Priority) != actual.Priority {
		return true
	}

	return false
}

//nolint:gocyclo
func (e *external) Create(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.SecurityGroupRule)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSecurityGroupRule)
	}

	cr.SetConditions(xpv1.Creating())

	opts := rules.CreateOpts{
		SecurityGroupRule: rules.SecurityGroupRuleOptions{
			SecurityGroupID: cr.Spec.ForProvider.SecurityGroupID,
			Direction:       cr.Spec.ForProvider.Direction,
		},
	}

	if cr.Spec.ForProvider.Description != nil {
		opts.SecurityGroupRule.Description = *cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Ethertype != nil {
		opts.SecurityGroupRule.Ethertype = *cr.Spec.ForProvider.Ethertype
	}
	if cr.Spec.ForProvider.Protocol != nil {
		opts.SecurityGroupRule.Protocol = *cr.Spec.ForProvider.Protocol
	}
	if cr.Spec.ForProvider.Multiport != nil {
		opts.SecurityGroupRule.Multiport = *cr.Spec.ForProvider.Multiport
	}
	if cr.Spec.ForProvider.RemoteIPPrefix != nil {
		opts.SecurityGroupRule.RemoteIPPrefix = *cr.Spec.ForProvider.RemoteIPPrefix
	}
	if cr.Spec.ForProvider.RemoteGroupID != nil {
		opts.SecurityGroupRule.RemoteGroupID = *cr.Spec.ForProvider.RemoteGroupID
	}
	if cr.Spec.ForProvider.Action != nil {
		opts.SecurityGroupRule.Action = *cr.Spec.ForProvider.Action
	}
	if cr.Spec.ForProvider.Priority != nil {
		opts.SecurityGroupRule.Priority = *cr.Spec.ForProvider.Priority
	}

	rule, err := rules.Create(e.client, opts)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreate)
	}

	// Set external name to the Security Group Rule ID
	meta.SetExternalName(cr, rule.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalUpdate, error) {
	// Security Group Rules are generally immutable. We return an error if drift
	// is detected because we cannot update the resource. The user must recreate
	// the resource.
	return managed.ExternalUpdate{}, errors.New("SecurityGroupRule is immutable")
}

func (e *external) Delete(
	ctx context.Context,
	mg resource.Managed,
) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.SecurityGroupRule)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotSecurityGroupRule)
	}

	cr.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	err := rules.Delete(e.client, externalName)
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
