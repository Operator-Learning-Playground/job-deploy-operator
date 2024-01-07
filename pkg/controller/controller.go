package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	jobflowv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobflow/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type JobFlowController struct {
	client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	log      logr.Logger
}

func NewJobFlowController(client client.Client, log logr.Logger, scheme *runtime.Scheme) *JobFlowController {
	return &JobFlowController{
		client: client,
		log:    log,
		Scheme: scheme,
	}
}

// Reconcile 调协 loop
func (r *JobFlowController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	klog.Info("start jobFlow Reconcile..........")
	klog.Info(fmt.Sprintf("req.%v", req))

	// load JobFlow by namespace
	jobFlow := &jobflowv1alpha1.JobFlow{}
	time.Sleep(time.Second)
	err := r.client.Get(ctx, req.NamespacedName, jobFlow)
	if err != nil {
		// If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("not found jobFlow : %v", req.Name))
			return reconcile.Result{}, nil
		}
		klog.Error(err, err.Error())
		r.Recorder.Eventf(jobFlow, v1.EventTypeWarning, "Created", err.Error())
		return reconcile.Result{}, err
	}

	// 启动 依序启动 job 任务

	// 改变 job 状态

	return reconcile.Result{}, nil
}
