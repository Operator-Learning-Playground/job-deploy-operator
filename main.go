package main

import (
	daemonjobv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/daemonjob/v1alpha1"
	jobflowv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobflow/v1alpha1"
	"github.com/myoperator/jobflowoperator/pkg/controller"
	"github.com/myoperator/jobflowoperator/pkg/k8sconfig"
	batchv1 "k8s.io/api/batch/v1"
	_ "k8s.io/code-generator"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

/*
	[root@VM-0-16-centos myoperator]# cd
	[root@VM-0-16-centos ~]# cp -r jobflowoperator/ /root/go/src/github.com/myoperator/jobflowoperator/
	[root@VM-0-16-centos ~]# cd /root/go/src/github.com/myoperator
	[root@VM-0-16-centos myoperator]# cd jobflowoperator/
	[root@VM-0-16-centos jobflowoperator]# ls
	Dockerfile  go.mod  go.sum  main.go  pkg  resources  yaml
	[root@VM-0-16-centos jobflowoperator]# $GOPATH/src/k8s.io/code-generator/generate-groups.sh all  github.com/myoperator/jobflowoperator/pkg/client github.com/myoperator/jobflowoperator/pkg/apis jobflow:v1alpha1
	Generating deepcopy funcs
	Generating clientset for jobflow:v1alpha1 at github.com/myoperator/jobflowoperator/pkg/client/clientset
	Generating listers for jobflow:v1alpha1 at github.com/myoperator/jobflowoperator/pkg/client/listers
	Generating informers for jobflow:v1alpha1 at github.com/myoperator/jobflowoperator/pkg/client/informers
	[root@VM-0-16-centos jobflowoperator]# pwd
	/root/go/src/github.com/myoperator/jobflowoperator
	[root@VM-0-16-centos jobflowoperator]# cd
	[root@VM-0-16-centos ~]# rm -rf jobflowoperator/
	[root@VM-0-16-centos ~]# cp -r /root/go/src/github.com/myoperator/jobflowoperator/ ~/jobflowoperator
	[root@VM-0-16-centos ~]# cd jobflowoperator/
	[root@VM-0-16-centos jobflowoperator]# ls
	Dockerfile  go.mod  go.sum  main.go  pkg  resources  yaml



	manager 主要用来管理Controller Admission Webhook 包括：
	访问资源对象的client cache scheme 并提供依赖注入机制 优雅关闭机制

	operator = crd + controller + webhook
*/

func main() {

	logf.SetLogger(zap.New())
	var d time.Duration = 0
	// 1. 管理器初始化
	mgr, err := manager.New(k8sconfig.K8sRestConfig(), manager.Options{
		Logger:     logf.Log.WithName("JobFlow operator"),
		SyncPeriod: &d, // resync不设置触发
	})
	if err != nil {
		mgr.GetLogger().Error(err, "unable to set up manager")
		os.Exit(1)
	}

	// 2. ++ 注册进入序列化表
	err = jobflowv1alpha1.SchemeBuilder.AddToScheme(mgr.GetScheme())
	if err != nil {
		klog.Error(err, "unable add schema")
		os.Exit(1)
	}

	err = daemonjobv1alpha1.SchemeBuilder.AddToScheme(mgr.GetScheme())
	if err != nil {
		klog.Error(err, "unable add schema")
		os.Exit(1)
	}

	// 3. 控制器相关
	jobFlowCtl := controller.NewJobFlowController(mgr.GetClient(), mgr.GetLogger(),
		mgr.GetScheme(), mgr.GetEventRecorderFor("JobFlow operator"))

	err = builder.ControllerManagedBy(mgr).For(&jobflowv1alpha1.JobFlow{}).
		Watches(&source.Kind{Type: &batchv1.Job{}},
			handler.Funcs{
				UpdateFunc: jobFlowCtl.OnUpdateJobHandlerByJobFlow,
				DeleteFunc: jobFlowCtl.OnDeleteJobHandlerByJobFlow,
			},
		).Complete(jobFlowCtl)

	// 3. 控制器相关
	daemonJobCtl := controller.NewDaemonJobController(mgr.GetClient(), mgr.GetLogger(),
		mgr.GetScheme(), mgr.GetEventRecorderFor("DaemonJob operator"))

	err = builder.ControllerManagedBy(mgr).For(&daemonjobv1alpha1.DaemonJob{}).
		Watches(&source.Kind{Type: &batchv1.Job{}},
			handler.Funcs{
				UpdateFunc: daemonJobCtl.OnUpdateJobHandlerByDaemonJob,
				DeleteFunc: daemonJobCtl.OnDeleteJobHandlerByDaemonJob,
			},
		).Complete(daemonJobCtl)

	errC := make(chan error)

	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		errC <- err
	}
}
