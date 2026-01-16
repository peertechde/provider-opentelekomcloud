package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// VPCParameters are the configurable fields of a VPC.
type VPCParameters struct {
	// Name is the name of the VPC. The name must be unique for a tenant.
	// The value is a string of no more than 64 characters and can contain
	// digits, letters, underscores (_), and hyphens (-).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=64
	Name string `json:"name"`

	// CIDR is the range of available subnets in the VPC.
	// The value must be in CIDR format, for example, 192.168.0.0/16.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="CIDR is immutable"
	CIDR string `json:"cidr"`

	// Description provides supplementary information about the VPC.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Description *string `json:"description,omitempty"`
}

// VPCObservation are the observable fields of a VPC.
type VPCObservation struct {
	// ID is the unique identifier of the VPC.
	ID string `json:"id,omitempty"`

	// Status indicates the VPC status. Values can be CREATING, OK, DOWN,
	// PENDING_UPDATE, PENDING_DELETE, or ERROR.
	Status string `json:"status,omitempty"`

	// CIDR is the actual CIDR block of the VPC.
	CIDR string `json:"cidr,omitempty"`
}

// A VPCSpec defines the desired state of a VPC.
type VPCSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              VPCParameters `json:"forProvider"`
}

// A VPCStatus represents the observed state of a VPC.
type VPCStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VPCObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// VPC is the Schema for the VPC API.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="CIDR",type="string",JSONPath=".spec.forProvider.cidr"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
type VPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCSpec   `json:"spec"`
	Status VPCStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VPCList contains a list of VPC
type VPCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VPC `json:"items"`
}

// VPC type metadata.
var (
	VPCKind             = reflect.TypeOf(VPC{}).Name()
	VPCGroupKind        = schema.GroupKind{Group: Group, Kind: VPCKind}.String()
	VPCKindAPIVersion   = VPCKind + "." + SchemeGroupVersion.String()
	VPCGroupVersionKind = SchemeGroupVersion.WithKind(VPCKind)
)

func init() {
	SchemeBuilder.Register(&VPC{}, &VPCList{})
}
