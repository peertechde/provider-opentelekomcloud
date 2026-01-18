package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// NATGatewayParameters are the configurable fields of a NATGateway.
type NATGatewayParameters struct {
	// Name is the name of the NAT Gateway.
	// The value is a string of no more than 64 characters and can contain
	// digits, letters, underscores (_), and hyphens (-).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=64
	Name string `json:"name"`

	// Description is the description of the NAT Gateway.
	// +optional
	Description *string `json:"description,omitempty"`

	// Spec is the specification of the NAT Gateway.
	// It accepts either the API ID (e.g., "1", "2") or the human-readable size.
	// Valid values:
	// "1" or "Small"
	// "2" or "Medium"
	// "3" or "Large"
	// "4" or "Extra-Large"
	// "0" or "Micro"
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="0";"1";"2";"3";"4";"Micro";"Small";"Medium";"Large";"Extra-Large";"micro";"small";"medium";"large";"extra-large"
	Spec string `json:"spec"`

	// VPCID is the ID of the VPC (Router) this NAT Gateway belongs to.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/vpc/v1alpha1.VPC
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VPCID is immutable"
	VPCID string `json:"vpcId,omitempty"`

	// VPCIDRef references a VPC to retrieve its ID.
	// +optional
	VPCIDRef *xpv1.NamespacedReference `json:"vpcIdRef,omitempty"`

	// VPCIDSelector selects a reference to a VPC.
	// +optional
	VPCIDSelector *xpv1.NamespacedSelector `json:"vpcIdSelector,omitempty"`

	// SubnetID is the ID of the Network (Subnet) this NAT Gateway connects to.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/subnet/v1alpha1.Subnet
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SubnetID is immutable"
	SubnetID string `json:"subnetId,omitempty"`

	// SubnetIDRef references a Subnet to retrieve its ID.
	// +optional
	SubnetIDRef *xpv1.NamespacedReference `json:"subnetIdRef,omitempty"`

	// SubnetIDSelector selects a reference to a Subnet.
	// +optional
	SubnetIDSelector *xpv1.NamespacedSelector `json:"subnetIdSelector,omitempty"`
}

// NATGatewayObservation are the observable fields of a NATGateway.
type NATGatewayObservation struct {
	// ID is the unique identifier of the NAT Gateway.
	ID string `json:"id,omitempty"`

	// Status indicates the current status of the NAT Gateway.
	Status string `json:"status,omitempty"`

	// AdminStateUp indicates whether the NAT Gateway is enabled.
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// VPCID is the actual VPC ID of the NAT Gateway.
	VPCID string `json:"vpcId,omitempty"`

	// SubnetID is the actual Subnet ID of the NAT Gateway.
	SubnetID string `json:"subnetId,omitempty"`
}

// A NATGatewaySpec defines the desired state of a NATGateway.
type NATGatewaySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              NATGatewayParameters `json:"forProvider"`
}

// A NATGatewayStatus represents the observed state of a NATGateway.
type NATGatewayStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NATGatewayObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A NATGateway is the Schema for the NAT Gateway.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="SPEC",type="string",JSONPath=".spec.forProvider.spec"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
type NATGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NATGatewaySpec   `json:"spec"`
	Status NATGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NATGatewayList contains a list of NATGateway
type NATGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NATGateway `json:"items"`
}

// NATGateway type metadata.
var (
	NATGatewayKind             = reflect.TypeOf(NATGateway{}).Name()
	NATGatewayGroupKind        = schema.GroupKind{Group: Group, Kind: NATGatewayKind}.String()
	NATGatewayKindAPIVersion   = NATGatewayKind + "." + SchemeGroupVersion.String()
	NATGatewayGroupVersionKind = SchemeGroupVersion.WithKind(NATGatewayKind)
)

func init() {
	SchemeBuilder.Register(&NATGateway{}, &NATGatewayList{})
}
