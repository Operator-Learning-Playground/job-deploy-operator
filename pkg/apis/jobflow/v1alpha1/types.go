package v1alpha1

import (
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	// GlobalParameters 全局参数
	GlobalParams GlobalParams `json:"globalParams"`
	// Flows 多个 flow 步骤流程
	Flows []Flow `json:"flows"`
	// TODO: ErrorHandler 错误处理逻辑
	ErrorHandler ErrorHandler
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

type ErrorHandler struct {
	// 用于赋值 job 模版
	corev1.PodSpec

	JobTemplate v1.JobSpec `json:"jobTemplate,omitempty"`
	// 支持脚本命令
	Script string `json:"script,omitempty"`
}

type Flow struct {
	// Name job name, namespace 就是默认 JobFlow namespace
	Name string `json:"name"`
	// JobTemplate 用于赋值 job 模版
	JobTemplate v1.JobSpec `json:"jobTemplate,omitempty"`
	// JobTemplateRef 模版实例对象 JobTemplate
	JobTemplateRef string `json:"jobTemplateRef,omitempty"`
	// Dependencies 依赖项，其中可以填写多个 依赖的 job name
	// ex: 如果 job3 依赖 job1 and job2, 就能
	Dependencies []string `json:"dependencies"`
}

type JobFlowStatus struct {
	// 用于存储 map 是 name/namespace 进行存储
	JobStatusList map[string]v1.JobStatus `json:"jobStatusList,omitempty"`
	// 记录 JobFlow 状态
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

// JobFlowList
type JobFlowList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobFlow `json:"items"`
}
