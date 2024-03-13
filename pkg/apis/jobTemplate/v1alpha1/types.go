package v1alpha1

import (
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobTemplate
type JobTemplate struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec JobTemplateSpec `json:"spec,omitempty"`

	Status JobTemplateStatus `json:"status,omitempty"`
}

type JobTemplateSpec struct {
	// JobTemplate 用于赋值 job 模版
	JobTemplate v1.JobSpec `json:"jobTemplate,omitempty"`
}

type JobTemplateStatus struct {
	// DependencyList 记录引用此模版的 Job 对象
	DependencyList []string `json:"dependencyList,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobTemplateList
type JobTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobTemplate `json:"items"`
}
