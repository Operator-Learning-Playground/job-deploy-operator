package v1alpha1

import (
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonJob
type DaemonJob struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DaemonJobSpec `json:"spec,omitempty"`

	Status DaemonJobStatus `json:"status,omitempty"`
}

type DaemonJobSpec struct {
	// GlobalParams 全局参数
	GlobalParams GlobalParams `json:"globalParams,omitempty"`
	// ExcludeNodeList 过滤出不需运行的 node List, ex: 如果 node3 node4 不需要
	// 运行 DaemonJob ，则填入： "node3,node4"
	ExcludeNodeList string `json:"excludeNodeList,omitempty"`
	v1.JobSpec
}

// GlobalParams 全局参数
type GlobalParams struct {
	// Env 容器环境变量
	Env []corev1.EnvVar `json:"env,omitempty"`
	// NodeName 选择调度节点
	NodeName string `json:"nodeName,omitempty"`
	// Labels job pod 的 labels
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations job pod 的 annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

type DaemonJobStatus struct {
	// 用于存储 map 是 name/namespace 进行存储
	JobStatusList map[string]v1.JobStatus `json:"jobStatusList,omitempty"`
	// 记录 JobFlow 状态
	// TODO: 使用特定类型封装
	State string `json:"state,omitempty"`
}

const (
	Succeed     = "Succeed"     // 代表 JobFlow 中所有 Job 都執行成功
	Terminating = "Terminating" // 代表 JobFlow 正在被刪除
	Failed      = "Failed"      // 代表 JobFlow 執行失敗
	Running     = "Running"     // 代表 JobFlow 有任何一個 Job 正在執行
	Pending     = "Pending"     // 代表 JobFlow 正在等待
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonJobList
type DaemonJobList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DaemonJob `json:"items"`
}
