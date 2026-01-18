package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// SNATRuleParameters are the configurable fields of a SNATRule.
type SNATRuleParameters struct {
	// NATGatewayID is the ID of the NAT Gateway to which this SNAT rule belongs.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/natgateway/v1alpha1.NATGateway
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="NATGatewayID is immutable"
	NATGatewayID string `json:"natGatewayId"`

	// NATGatewayIDRef references a NATGateway to retrieve its ID.
	// +optional
	NATGatewayIDRef *xpv1.NamespacedReference `json:"natGatewayIdRef,omitempty"`

	// NATGatewayIDSelector selects a reference to a NATGateway.
	// +optional
	NATGatewayIDSelector *xpv1.NamespacedSelector `json:"natGatewayIdSelector,omitempty"`

	// ElasticIPID is the ID of the Elastic IP (Public IP) used for SNAT.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/elasticip/v1alpha1.ElasticIP
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ElasticIPID is immutable"
	ElasticIPID string `json:"elasticIpId"`

	// ElasticIPIDRef references a ElasticIP to retrieve its ID.
	// +optional
	ElasticIPIDRef *xpv1.NamespacedReference `json:"elasticIPIDRef,omitempty"`

	// ElasticIPIDSelector selects a reference to a NATGateway.
	// +optional
	ElasticIPIDSelector *xpv1.NamespacedSelector `json:"elasticIPIDSelector,omitempty"`

	// SubnetID is the ID of the Subnet this SNAT rule connects to.
	// Either SubnetID or CIDR must be specified.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/subnet/v1alpha1.Subnet
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SubnetID is immutable"
	SubnetID *string `json:"subnetId,omitempty"`

	// SubnetIDRef references a Subnet to retrieve its ID.
	// +optional
	SubnetIDRef *xpv1.NamespacedReference `json:"subnetIDRef,omitempty"`

	// SubnetIDSelector selects a reference to a NATGateway.
	// +optional
	SubnetIDSelector *xpv1.NamespacedSelector `json:"subnetIDSelector,omitempty"`

	// CIDR is the CIDR block this SNAT rule connects to.
	// Either SubnetID or CIDR must be specified.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="CIDR is immutable"
	CIDR *string `json:"cidr,omitempty"`
}

// SNATRuleObservation are the observable fields of a SNATRule.
type SNATRuleObservation struct {
	// ID is the unique identifier of the SNAT rule.
	ID string `json:"id,omitempty"`

	// Status indicates the current status of the SNAT rule.
	Status string `json:"status,omitempty"`

	// ElasticIPAddress is the actual IP address of the elastic IP.
	ElasticIPAddress string `json:"elasticIpAddress,omitempty"`

	// AdminStateUp indicates whether the SNAT rule is enabled.
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// NATGatewayID is the actual NATGateway ID of the SNAT Rule.
	NATGatewayID string `json:"natGatewayId"`

	// ElasticIPID is the actual ElasticIP ID of the SNAT Rule.
	ElasticIPID string `json:"elasticIpId"`

	// SubnetID is the actual Subnet ID of the SNAT Rule.
	SubnetID string `json:"subnetId,omitempty"`
}

// A SNATRuleSpec defines the desired state of a SNATRule.
type SNATRuleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SNATRuleParameters `json:"forProvider"`
}

// A SNATRuleStatus represents the observed state of a SNATRule.
type SNATRuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SNATRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SNATRule is the Schema for the SNAT Rule.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="GATEWAY-ID",type="string",JSONPath=".spec.forProvider.natGatewayId"
// +kubebuilder:printcolumn:name="PUBLIC-IP",type="string",JSONPath=".status.atProvider.elasticIpAddress"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
// +kubebuilder:validation:XValidation:rule="has(self.spec.forProvider.subnetId) || has(self.spec.forProvider.cidr)",message="Either subnetId or cidr must be specified"
// +kubebuilder:validation:XValidation:rule="!(has(self.spec.forProvider.subnetId) && has(self.spec.forProvider.cidr))",message="Cannot specify both subnetId and cidr"
type SNATRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SNATRuleSpec   `json:"spec"`
	Status SNATRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SNATRuleList contains a list of SNATRule
type SNATRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SNATRule `json:"items"`
}

// SNATRule type metadata.
var (
	SNATRuleKind             = reflect.TypeOf(SNATRule{}).Name()
	SNATRuleGroupKind        = schema.GroupKind{Group: Group, Kind: SNATRuleKind}.String()
	SNATRuleKindAPIVersion   = SNATRuleKind + "." + SchemeGroupVersion.String()
	SNATRuleGroupVersionKind = SchemeGroupVersion.WithKind(SNATRuleKind)
)

func init() {
	SchemeBuilder.Register(&SNATRule{}, &SNATRuleList{})
}
