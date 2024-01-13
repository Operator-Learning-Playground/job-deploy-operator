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

func NewJobFlowController(client client.Client, log logr.Logger, scheme *runtime.Scheme, recorder record.EventRecorder) *JobFlowController {
	return &JobFlowController{
		client:   client,
		log:      log,
		Recorder: recorder,
		Scheme:   scheme,
	}
}

// Reconcile 调协 loop
func (r *JobFlowController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	klog.Info("start jobFlow Reconcile..........")

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

	if jobFlow.Status.State == jobflowv1alpha1.Succeed {
		return reconcile.Result{}, nil
	}

	// FIXME: 处理 Finalizer 字段
	// 考虑是否要在 jobflow status state 为 Running 时 不能删除？

	// deploy job by dependence order.
	if err = r.deployJobFlow(ctx, *jobFlow); err != nil {
		klog.Error("deployJob error: ", err)
		return reconcile.Result{}, err
	}

	// update status
	// 修改 job 狀態，list 出所有相關的 job ，並查看其狀態，並存在 status 中
	if err = r.updateJobFlowStatus(ctx, jobFlow); err != nil {
		klog.Error("update jobFlow status error: ", err)
		return reconcile.Result{}, err
	}
	klog.Info("end jobFlow Reconcile........")

	return reconcile.Result{}, nil
}
