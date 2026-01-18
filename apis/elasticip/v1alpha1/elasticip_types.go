package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// PublicIP defines the public IP arguments.
type PublicIP struct {
	// Type specifies the EIP type.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="BGP";"Mail"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable"
	Type string `json:"type"`

	// IPAddress specifies the EIP address.
	// +optional
	IPAddress *string `json:"ipAddress,omitempty"`
}

// BandwidthConfig defines the bandwidth arguments for the EIP.
type BandwidthConfig struct {
	// Size specifies the bandwidth size.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Size is immutable"
	Size int `json:"size,omitempty"`

	// ShareType specifies the bandwidth share type.
	// Valid values are "Dedicated" and "Shared".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="Dedicated";"Shared"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ShareType is immutable"
	ShareType string `json:"shareType"`
}

// ElasticIPParameters are the configurable fields of a ElasticIP.
type ElasticIPParameters struct {
	// PublicIP specifies the public IP configuration.
	// +kubebuilder:validation:Required
	PublicIP PublicIP `json:"publicIP"`

	// Bandwidth specifies the bandwidth configuration.
	// +kubebuilder:validation:Required
	Bandwidth BandwidthConfig `json:"bandwidth"`
}

// ElasticIPObservation are the observable fields of a ElasticIP.
type ElasticIPObservation struct {
	// ID is the unique identifier of the ElasticIP.
	ID string `json:"id,omitempty"`

	// Status indicates the current status of the ElasticIP.
	Status string `json:"status,omitempty"`

	// IPAddress is the actual EIP address.
	IPAddress string `json:"ipAddress,omitempty"`

	// PrivateIPAddress is the private IP address bound to the EIP.
	PrivateIPAddress string `json:"privateIpAddress,omitempty"`

	// PortID is the port ID bound to the EIP.
	PortID string `json:"portId,omitempty"`

	// BandwidthID is the ID of the bandwidth associated with the EIP.
	BandwidthID string `json:"bandwidthId,omitempty"`

	// BandwidthSize is the size of the bandwidth.
	BandwidthSize int `json:"bandwidthSize,omitempty"`

	// BandwidthShareType is the share type of the bandwidth.
	BandwidthShareType string `json:"bandwidthShareType,omitempty"`
}

// A ElasticIPSpec defines the desired state of a ElasticIP.
type ElasticIPSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ElasticIPParameters `json:"forProvider"`
}

// A ElasticIPStatus represents the observed state of a ElasticIP.
type ElasticIPStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ElasticIPObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticIP is the Schema for the Elastic IP.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".status.atProvider.ipAddress"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
type ElasticIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticIPSpec   `json:"spec"`
	Status ElasticIPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticIPList contains a list of ElasticIP
type ElasticIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticIP `json:"items"`
}

// ElasticIP type metadata.
var (
	ElasticIPKind             = reflect.TypeOf(ElasticIP{}).Name()
	ElasticIPGroupKind        = schema.GroupKind{Group: Group, Kind: ElasticIPKind}.String()
	ElasticIPKindAPIVersion   = ElasticIPKind + "." + SchemeGroupVersion.String()
	ElasticIPGroupVersionKind = SchemeGroupVersion.WithKind(ElasticIPKind)
)

func init() {
	SchemeBuilder.Register(&ElasticIP{}, &ElasticIPList{})
}
