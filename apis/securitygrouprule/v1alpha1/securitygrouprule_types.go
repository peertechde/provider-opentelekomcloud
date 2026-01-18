package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// SecurityGroupRuleParameters are the configurable fields of a SecurityGroupRule.
type SecurityGroupRuleParameters struct {
	// SecurityGroupID is the ID of the security group to which the
	// SecurityGroupRule belongs.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/securitygroup/v1alpha1.SecurityGroup
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SecurityGroupID is immutable"
	SecurityGroupID string `json:"securityGroupId,omitempty"`

	// SecurityGroupIDRef references a SecurityGroup to retrieve its ID.
	// +optional
	SecurityGroupIDRef *xpv1.NamespacedReference `json:"securityGroupIdRef,omitempty"`

	// SecurityGroupIDSelector selects a reference to a SecurityGroup.
	// +optional
	SecurityGroupIDSelector *xpv1.NamespacedSelector `json:"securityGroupIdSelector,omitempty"`

	// Direction specifies whether the rule applies to ingress or egress traffic.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ingress;egress
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Direction is immutable"
	Direction string `json:"direction"`

	// Description specifies the description of the rule.
	// +optional
	Description *string `json:"description,omitempty"`

	// Ethertype specifies the IP version.
	// Valid values are "IPv4" and "IPv6".
	// +optional
	// +kubebuilder:validation:Enum=IPv4;IPv6
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Ethertype is immutable"
	Ethertype *string `json:"ethertype,omitempty"`

	// Protocol specifies the network protocol.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Protocol is immutable"
	Protocol *string `json:"protocol,omitempty"`

	// Multiport specifies the port or port range (e.g., "80", "80-90").
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Multiport is immutable"
	Multiport *string `json:"multiport,omitempty"`

	// RemoteIPPrefix specifies the remote IP prefix (CIDR).
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="RemoteIPPrefix is immutable"
	RemoteIPPrefix *string `json:"remoteIpPrefix,omitempty"`

	// RemoteGroupID specifies the ID of the remote security group.
	// +crossplane:generate:reference:type=github.com/peertechde/provider-opentelekomcloud/apis/securitygroup/v1alpha1.SecurityGroup
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="RemoteGroupID is immutable"
	RemoteGroupID *string `json:"remoteGroupId,omitempty"`

	// RemoteGroupIDRef references a SecurityGroup to retrieve its ID.
	// +optional
	RemoteGroupIDRef *xpv1.NamespacedReference `json:"remoteGroupIdRef,omitempty"`

	// RemoteGroupIDSelector selects a reference to a SecurityGroup.
	// +optional
	RemoteGroupIDSelector *xpv1.NamespacedSelector `json:"remoteGroupIdSelector,omitempty"`

	// RemoteAddressGroupID is the ID of the remote address group.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="RemoteAddressGroupID is immutable"
	RemoteAddressGroupID *string `json:"remoteAddressGroupId,omitempty"`

	// Action specifies the action of the rule.
	// Valid values are "allow" and "deny".
	// +optional
	// +kubebuilder:validation:Enum=allow;deny
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Action is immutable"
	Action *string `json:"action,omitempty"`

	// Priority specifies the priority of the rule.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Priority is immutable"
	Priority *int `json:"priority,omitempty"`
}

// SecurityGroupRuleObservation are the observable fields of a SecurityGroupRule.
type SecurityGroupRuleObservation struct {
	// ID is the unique identifier of the SecurityGroupRule.
	ID string `json:"id,omitempty"`

	// SecurityGroupID is the actual SecurityGroupID of the SecurityGroupRule.
	SecurityGroupID string `json:"securityGroupId,omitempty"`
}

// A SecurityGroupRuleSpec defines the desired state of a SecurityGroupRule.
type SecurityGroupRuleSpec struct {
	xpv2.ManagedResourceSpec `                            json:",inline"`
	ForProvider              SecurityGroupRuleParameters `json:"forProvider"`
}

// A SecurityGroupRuleStatus represents the observed state of a SecurityGroupRule.
type SecurityGroupRuleStatus struct {
	xpv1.ResourceStatus `                             json:",inline"`
	AtProvider          SecurityGroupRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityGroupRule is the Schema for the SecurityGroupRules.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="DIRECTION",type="string",JSONPath=".spec.forProvider.direction"
// +kubebuilder:printcolumn:name="PROTOCOL",type="string",JSONPath=".spec.forProvider.protocol"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,opentelekomcloud}
type SecurityGroupRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupRuleSpec   `json:"spec"`
	Status SecurityGroupRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityGroupRuleList contains a list of SecurityGroupRule
type SecurityGroupRuleList struct {
	metav1.TypeMeta `                    json:",inline"`
	metav1.ListMeta `                    json:"metadata,omitempty"`
	Items           []SecurityGroupRule `json:"items"`
}

// SecurityGroupRule type metadata.
var (
	SecurityGroupRuleKind      = reflect.TypeOf(SecurityGroupRule{}).Name()
	SecurityGroupRuleGroupKind = schema.GroupKind{
		Group: Group,
		Kind:  SecurityGroupRuleKind,
	}.String()
	SecurityGroupRuleKindAPIVersion   = SecurityGroupRuleKind + "." + SchemeGroupVersion.String()
	SecurityGroupRuleGroupVersionKind = SchemeGroupVersion.WithKind(SecurityGroupRuleKind)
)

func init() {
	SchemeBuilder.Register(&SecurityGroupRule{}, &SecurityGroupRuleList{})
}
