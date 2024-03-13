package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	jobtemplatev1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobTemplate/v1alpha1"
	"github.com/myoperator/jobflowoperator/pkg/common"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

type JobTemplateController struct {
	client client.Client
	Scheme *runtime.Scheme
	event  record.EventRecorder
	log    logr.Logger
}

func NewJobTemplateController(client client.Client, log logr.Logger,
	scheme *runtime.Scheme, event record.EventRecorder) *JobTemplateController {
	return &JobTemplateController{
		client: client,
		log:    log,
		event:  event,
		Scheme: scheme,
	}
}

// Reconcile 调协 loop
func (r *JobTemplateController) Reconcile(ctx context.Context,
	req reconcile.Request) (reconcile.Result, error) {
	klog.Info("start JobTemplate Reconcile..........")

	// load JobTemplate by namespace
	jobTemplate := &jobtemplatev1alpha1.JobTemplate{}
	time.Sleep(time.Second)
	err := r.client.Get(ctx, req.NamespacedName, jobTemplate)
	if err != nil {
		// If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("not found JobTemplate : %v", req.Name))
			return reconcile.Result{}, nil
		}
		klog.Error(err, err.Error())
		r.event.Eventf(jobTemplate, v1.EventTypeWarning, "Created", err.Error())
		return reconcile.Result{}, err
	}

	// update status
	// 修改 JobTemplate 状态，list 出所有相关的 job ，並查看状态，存在 status 中
	if err = r.updateJobTemplateStatus(ctx, jobTemplate); err != nil {
		klog.Error("update jobTemplate status error: ", err)
		r.event.Eventf(jobTemplate, v1.EventTypeWarning, "Failed", err.Error())
		return reconcile.Result{}, err
	}
	klog.Info("end jobTemplate Reconcile........")

	return reconcile.Result{}, nil
}

// update status
func (r *JobTemplateController) updateJobTemplateStatus(ctx context.Context, jobTemplate *jobtemplatev1alpha1.JobTemplate) error {
	klog.Info(fmt.Sprintf("start to update JobTemplate status! JobTemplateName: %v, JobTemplateNamespace: %v ", jobTemplate.Name, jobTemplate.Namespace))
	// 获取 job 列表
	allJobList := new(batchv1.JobList)
	err := r.client.List(ctx, allJobList)
	if err != nil {
		klog.Error("list error: ", err)
		return err
	}

	// 记录出引用此 JobTemplate 的 Job 实例
	filterJobList := make([]batchv1.Job, 0)
	for _, item := range allJobList.Items {
		if item.Annotations[common.CreateByJobTemplate] ==
			common.GetConnectionOfJobAndJobTemplate(jobTemplate.Namespace, jobTemplate.Name) {
			filterJobList = append(filterJobList, item)
		}
	}

	if len(filterJobList) == 0 {
		return nil
	}

	jobListName := make([]string, 0)
	for _, job := range filterJobList {
		jobListName = append(jobListName, job.Name)
	}
	jobTemplate.Status.DependencyList = jobListName

	if err = r.client.Status().Update(ctx, jobTemplate); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *JobTemplateController) JobCreateTemplateHandler(e event.CreateEvent, w workqueue.RateLimitingInterface) {
	if e.Object.GetAnnotations()[common.CreateByJobTemplate] != "" {
		nameNamespace := strings.Split(e.Object.GetAnnotations()[common.CreateByJobTemplate], ".")
		namespace, name := nameNamespace[0], nameNamespace[1]
		w.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	}
}

func (r *JobTemplateController) JobUpdateTemplateHandler(e event.UpdateEvent, w workqueue.RateLimitingInterface) {
	if e.ObjectNew.GetAnnotations()[common.CreateByJobTemplate] != "" {
		nameNamespace := strings.Split(e.ObjectNew.GetAnnotations()[common.CreateByJobTemplate], ".")
		namespace, name := nameNamespace[0], nameNamespace[1]
		w.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	}
}

func (r *JobTemplateController) JobDeleteTemplateHandler(e event.DeleteEvent, w workqueue.RateLimitingInterface) {
	if e.Object.GetAnnotations()[common.CreateByJobTemplate] != "" {
		nameNamespace := strings.Split(e.Object.GetAnnotations()[common.CreateByJobTemplate], ".")
		namespace, name := nameNamespace[0], nameNamespace[1]
		w.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	}
}
