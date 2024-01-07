package v1alpha1

import (
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobFlow
type JobFlow struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec JobFlowSpec `json:"spec,omitempty"`

	Status JobFlowStatus `json:"status,omitempty"`
}

type JobFlowSpec struct {
	Flows []Flow `json:"flows,omitempty"`
}

type Flow struct {
	// job name, namespace 就是默认 namespace
	Name string `json:"name"`
	// 用于赋值
	JobTemplate v1.JobSpec `json:"jobTemplate"`
	// 依赖项
	Dependencies []string `json:"dependencies"`
}

type JobFlowStatus struct {
	// 用于存储 map 是 name/namespace 的方式 或是只要 name就行
	JobStatusList map[string]v1.JobStatus `json:"jobStatusList,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobFlowList
type JobFlowList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobFlow `json:"items"`
}
