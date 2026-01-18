package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// SubnetParameters defines the desired state of a Subnet.
type SubnetParameters struct {
	// Name is the name of the Subnet.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// CIDR is the network segment of the Subnet.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="CIDR is immutable"
	CIDR string `json:"cidr"`

	// GatewayIP is the gateway address of the Subnet.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="GatewayIP is immutable"
	GatewayIP string `json:"gatewayIp"`

	// VPCID is the ID of the VPC to which the Subnet belongs.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/vpc/v1alpha1.VPC
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VPCID is immutable"
	VPCID string `json:"vpcId,omitempty"`

	// VPCIDRef is a reference to a VPC to retrieve its ID.
	// +optional
	VPCIDRef *xpv1.NamespacedReference `json:"vpcIdRef,omitempty"`

	// VPCIDSelector selects a reference to a VPC to retrieve its ID.
	// +optional
	VPCIDSelector *xpv1.NamespacedSelector `json:"vpcIdSelector,omitempty"`

	// DHCPEnable specifies whether DHCP is enabled.
	// +optional
	DHCPEnable *bool `json:"dhcpEnable,omitempty"`

	// PrimaryDNS is the IP address of the primary DNS server.
	// +optional
	PrimaryDNS *string `json:"primaryDns,omitempty"`

	// SecondaryDNS is the IP address of the secondary DNS server.
	// +optional
	SecondaryDNS *string `json:"secondaryDns,omitempty"`

	// AvailabilityZone is the availability zone of the Subnet.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="AvailabilityZone is immutable"
	AvailabilityZone *string `json:"availabilityZone,omitempty"`

	// Description is the description of the Subnet.
	// +optional
	Description *string `json:"description,omitempty"`
}

// SubnetObservation are the observable fields of a Subnet.
type SubnetObservation struct {
	// ID is the unique identifier of the Subnet.
	ID string `json:"id,omitempty"`

	// Status indicates the current status of the Subnet.
	Status string `json:"status,omitempty"`

	// CIDR is the actual CIDR block of the Subnet.
	CIDR string `json:"cidr,omitempty"`

	// GatewayIP is the actual gateway address of the Subnet.
	GatewayIP string `json:"gatewayIp"`

	// VPCID is the actual VPC ID of the Subnet.
	VPCID string `json:"vpcId,omitempty"`
}

// A SubnetSpec defines the desired state of a Subnet.
type SubnetSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SubnetParameters `json:"forProvider"`
}

// A SubnetStatus represents the observed state of a Subnet.
type SubnetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SubnetObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Subnet is the Schema for the Subnets.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="VPC",type="string",JSONPath=".spec.forProvider.vpcId"
// +kubebuilder:printcolumn:name="CIDR",type="string",JSONPath=".spec.forProvider.cidr"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubnetList contains a list of Subnet
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}

// Subnet type metadata.
var (
	SubnetKind             = reflect.TypeOf(Subnet{}).Name()
	SubnetGroupKind        = schema.GroupKind{Group: Group, Kind: SubnetKind}.String()
	SubnetKindAPIVersion   = SubnetKind + "." + SchemeGroupVersion.String()
	SubnetGroupVersionKind = SchemeGroupVersion.WithKind(SubnetKind)
)

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
