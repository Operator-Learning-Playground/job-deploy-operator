package k8sconfig

import (
	"github.com/myoperator/jobflowoperator/pkg/common"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
)

// K8sRestConfigOrDie 集群外部使用
func K8sRestConfigOrDie() *rest.Config {
	// 读取配置
	if os.Getenv("Release") == "1" {
		klog.V(2).Info("run in the cluster")
		return k8sRestConfigInPod()
	}

	path := common.GetWd()
	config, err := clientcmd.BuildConfigFromFlags("", path+"/resources/config")
	if err != nil {
		klog.Fatal(err)
	}
	config.Insecure = true
	klog.V(2).Info("run outside the cluster")
	return config
}

// k8sRestConfigInPod 集群内部 Pod 里使用
func k8sRestConfigInPod() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}
	return config
}
