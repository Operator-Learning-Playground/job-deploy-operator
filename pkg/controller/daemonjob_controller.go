package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	daemonjobv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/daemonjob/v1alpha1"
)

type DaemonJobController struct {
	client client.Client
	Scheme *runtime.Scheme
	event  record.EventRecorder
	log    logr.Logger
}

func NewDaemonJobController(client client.Client, log logr.Logger,
	scheme *runtime.Scheme, event record.EventRecorder) *DaemonJobController {
	return &DaemonJobController{
		client: client,
		log:    log,
		event:  event,
		Scheme: scheme,
	}
}

// Reconcile 调协 loop
func (r *DaemonJobController) Reconcile(ctx context.Context,
	req reconcile.Request) (reconcile.Result, error) {
	klog.V(2).Info("start DaemonJob Reconcile..........")

	// load JobFlow by namespace
	daemonJob := &daemonjobv1alpha1.DaemonJob{}
	time.Sleep(time.Second)
	err := r.client.Get(ctx, req.NamespacedName, daemonJob)
	if err != nil {
		// If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.V(2).Info(fmt.Sprintf("not found daemonJob : %v", req.Name))
			return reconcile.Result{}, nil
		}
		klog.Error("get DaemonJob error: ", err.Error())
		r.event.Eventf(daemonJob, v1.EventTypeWarning, "Created", err.Error())
		return reconcile.Result{}, err
	}

	if daemonJob.Status.State == daemonjobv1alpha1.Failed {
		return reconcile.Result{}, nil
	}

	// deploy job by dependence order.
	if err = r.deployDaemonJob(ctx, daemonJob); err != nil {
		klog.Error("deploy DaemonJob error: ", err)
		r.event.Eventf(daemonJob, v1.EventTypeWarning, "Failed", err.Error())
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 60}, err
	}

	// update status
	// 修改 job 状态，list 出所有相关的 job ，并查看其状态，存在 status 中
	if err = r.updateJobFlowStatus(ctx, daemonJob); err != nil {
		klog.Error("update DaemonJob status error: ", err)
		r.event.Eventf(daemonJob, v1.EventTypeWarning, "Failed", err.Error())
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 60}, err
	}
	klog.V(2).Info("end DaemonJob Reconcile........")

	return reconcile.Result{}, nil
}

// deployDaemonJob deploy job by dependence order.
func (r *DaemonJobController) deployDaemonJob(ctx context.Context,
	daemonJob *daemonjobv1alpha1.DaemonJob) error {
	nodeList := &v1.NodeList{}
	err := r.client.List(ctx, nodeList)
	if err != nil {
		return err
	}

	for _, v := range nodeList.Items {
		// 如果是需要跳过的节点，不处理
		if isJumpOverNode(v.Name, splitString(daemonJob.Spec.ExcludeNodeList, ",")) {
			continue
		}

		// 其他节点：先 get 一下，如果不存在则创建，若存在就不处理
		// job 对象
		job := prepareJobFromDaemonJob(daemonJob, daemonJob.Name, v.Name)
		namespacedNameJob := types.NamespacedName{
			Namespace: daemonJob.Namespace,
			Name:      daemonJob.Name,
		}
		if err := r.client.Get(ctx, namespacedNameJob, job); err != nil {
			if errors.IsNotFound(err) {
				if err = r.client.Create(ctx, job); err != nil {
					if errors.IsAlreadyExists(err) {
						break
					}
					return err
				}
				r.event.Eventf(daemonJob, v1.EventTypeNormal, "Created", fmt.Sprintf("create job named %v for next step", job.Name))
			}
			return err
		}
	}
	return nil
}

func splitString(input, separator string) []string {
	// 去除空格
	input = strings.ReplaceAll(input, " ", "")
	// 使用 strings.Split 进行分割
	result := strings.Split(input, separator)
	// 排序
	sort.StringSlice(result).Sort()
	return result
}

// isJumpOverNode 跳过特定 node
func isJumpOverNode(node string, excludeNodeList []string) bool {
	for _, v := range excludeNodeList {
		if node == v {
			return true
		}
	}
	return false
}

func prepareJobFromDaemonJob(daemonJob *daemonjobv1alpha1.DaemonJob, jobName, nodeName string) *batchv1.Job {
	// job 对象
	job := &batchv1.Job{}

	// 设置 ownerReferences
	job.OwnerReferences = append(job.OwnerReferences, metav1.OwnerReference{
		APIVersion: daemonJob.APIVersion,
		Kind:       daemonJob.Kind,
		Name:       daemonJob.Name,
		UID:        daemonJob.UID,
	})

	job.Name = jobName + "-" + nodeName
	job.Namespace = daemonJob.Namespace
	job.Spec = daemonJob.Spec.JobSpec
	job.Spec.Template.Spec.NodeName = nodeName

	// 强制设置 job 不重启与重试次数
	job.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyNever
	var cc int32
	job.Spec.BackoffLimit = &cc

	// 加入 flow 全局参数
	if daemonJob.Spec.GlobalParams.Annotations != nil {
		job.Annotations = daemonJob.Spec.GlobalParams.Annotations
		job.Spec.Template.Annotations = daemonJob.Spec.GlobalParams.Annotations
	}

	if daemonJob.Spec.GlobalParams.Labels != nil {
		job.Labels = daemonJob.Spec.GlobalParams.Labels
		job.Spec.Template.Labels = daemonJob.Spec.GlobalParams.Labels
	}

	if daemonJob.Spec.GlobalParams.Env != nil {
		for k := range job.Spec.Template.Spec.Containers {
			job.Spec.Template.Spec.Containers[k].Env = daemonJob.Spec.GlobalParams.Env
		}
	}

	return job
}

// update status
func (r *DaemonJobController) updateJobFlowStatus(ctx context.Context, daemonJob *daemonjobv1alpha1.DaemonJob) error {
	klog.V(2).Info(fmt.Sprintf("start to update daemonJob status! daemonJobName: %v, daemonJobNamespace: %v ", daemonJob.Name, daemonJob.Namespace))
	// 获取 job 列表
	allJobList := new(batchv1.JobList)
	err := r.client.List(ctx, allJobList)
	if err != nil {
		klog.Error("list error: ", err)
		return err
	}
	jobFlowStatus, err := r.getAllJobStatusFromDaemonJob(ctx, daemonJob, allJobList)
	if err != nil {
		return err
	}
	daemonJob.Status = *jobFlowStatus
	if jobFlowStatus.State == daemonjobv1alpha1.Succeed || jobFlowStatus.State == daemonjobv1alpha1.Failed {
		r.event.Eventf(daemonJob, v1.EventTypeNormal, jobFlowStatus.State, fmt.Sprintf("finshed JobFlow named %s", daemonJob.Name))
	}
	if err = r.client.Status().Update(ctx, daemonJob); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// getAllJobStatus 记录 Job Status
func (r *DaemonJobController) getAllJobStatusFromDaemonJob(ctx context.Context, daemonJob *daemonjobv1alpha1.DaemonJob, allJobList *batchv1.JobList) (*daemonjobv1alpha1.DaemonJobStatus, error) {
	// 过去掉只留 daemonJob 相关的 job
	jobListRes := make([]batchv1.Job, 0)
	for _, job := range allJobList.Items {
		for _, reference := range job.OwnerReferences {
			if reference.Kind == daemonjobv1alpha1.DaemonJobKind && reference.Name == daemonJob.Name {
				jobListRes = append(jobListRes, job)
			}
		}
	}

	runningJobs := make([]string, 0)
	failedJobs := make([]string, 0)
	completedJobs := make([]string, 0)

	jobList := make([]string, 0)

	// 创建一个空的 Node 列表对象
	nodeList := &v1.NodeList{}
	err := r.client.List(ctx, nodeList)
	if err != nil {
		return nil, err
	}

	for _, v := range nodeList.Items {
		if isJumpOverNode(v.Name, splitString(daemonJob.Spec.ExcludeNodeList, ",")) {
			continue
		}
		jobList = append(jobList, v.Name)
	}

	jobFlowStatus := daemonjobv1alpha1.DaemonJobStatus{
		JobStatusList: map[string]batchv1.JobStatus{},
	}

	for _, job := range jobListRes {
		key := fmt.Sprintf("%s/%s", job.Name, job.Namespace)
		jobFlowStatus.JobStatusList[key] = job.Status

		if job.Status.Succeeded == 1 {
			completedJobs = append(completedJobs, job.Name)
		} else if job.Status.Failed == 1 {
			failedJobs = append(failedJobs, job.Name)
		} else if job.Status.Active == 1 {
			runningJobs = append(runningJobs, job.Name)
		}
	}

	// 确认 daemonJob 狀態
	if daemonJob.DeletionTimestamp != nil {
		jobFlowStatus.State = daemonjobv1alpha1.Terminating
	} else {
		if len(jobList) != len(completedJobs) {
			if len(failedJobs) > 0 {
				jobFlowStatus.State = daemonjobv1alpha1.Failed
			} else if len(runningJobs) > 0 || len(completedJobs) > 0 {
				jobFlowStatus.State = daemonjobv1alpha1.Running
			} else {
				jobFlowStatus.State = daemonjobv1alpha1.Pending
			}
		} else {
			jobFlowStatus.State = daemonjobv1alpha1.Succeed
		}
	}

	return &jobFlowStatus, nil
}

func (r *DaemonJobController) OnUpdateJobHandlerByDaemonJob(event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.ObjectNew.GetOwnerReferences() {
		if ref.Kind == daemonjobv1alpha1.DaemonJobKind && ref.APIVersion == daemonjobv1alpha1.DaemonJobApiVersion {
			// 重新放入 Reconcile 调协方法
			klog.V(5).Info("update job: ", event.ObjectNew.GetName(), event.ObjectNew.GetObjectKind())
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ref.Name, Namespace: event.ObjectNew.GetNamespace(),
				},
			})
		}
	}
}

func (r *DaemonJobController) OnDeleteJobHandlerByDaemonJob(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.Object.GetOwnerReferences() {
		if ref.Kind == daemonjobv1alpha1.DaemonJobKind && ref.APIVersion == daemonjobv1alpha1.DaemonJobApiVersion {
			// 重新入列
			klog.V(5).Info("delete job: ", event.Object.GetName(), event.Object.GetObjectKind())
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: ref.Name,
					Namespace: event.Object.GetNamespace()}})
		}
	}
}
