// Package apis contains Kubernetes API for the OpenTelekomCloud provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	natgatewayv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/natgateway/v1alpha1"
	securitygroupv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/securitygroup/v1alpha1"
	securitygrouprulev1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/securitygrouprule/v1alpha1"
	subnetv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/subnet/v1alpha1"
	opentelekomcloudv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/v1alpha1"
	vpcv1alpha1 "github.com/peertechde/provider-opentelekomcloud/apis/vpc/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		opentelekomcloudv1alpha1.SchemeBuilder.AddToScheme,
		vpcv1alpha1.SchemeBuilder.AddToScheme,
		subnetv1alpha1.SchemeBuilder.AddToScheme,
		securitygroupv1alpha1.SchemeBuilder.AddToScheme,
		securitygrouprulev1alpha1.SchemeBuilder.AddToScheme,
		natgatewayv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
