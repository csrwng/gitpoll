package v1beta1

import (
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api/v1beta1"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
)

// Config contains a set of Kubernetes resources to be applied.
// TODO: Unify with Kubernetes Config
//       https://github.com/GoogleCloudPlatform/kubernetes/pull/1007
type Config struct {
	kapi.TypeMeta `json:",inline" yaml:",inline"`

	// Required: Name identifies the Config.
	Name string `json:"name" yaml:"name"`

	// Optional: Description describes the Config.
	Description string `json:"description" yaml:"description"`

	// Required: Items is an array of Kubernetes resources of Service,
	// Pod and/or ReplicationController kind.
	// TODO: Handle unregistered types. Define custom []runtime.Object
	//       type and its unmarshaller instead of []runtime.Object.
	Items []runtime.RawExtension `json:"items" yaml:"items"`
}
