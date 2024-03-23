package controller

import (
	"context"
	"fmt"
	jobtemplatev1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobTemplate/v1alpha1"
	jobflowv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobflow/v1alpha1"
	"github.com/myoperator/jobflowoperator/pkg/common"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// deploy job by dependence order.
func (r *JobFlowController) deployJobFlow(ctx context.Context, jobFlow jobflowv1alpha1.JobFlow) error {
	// 启动 job
	for i, flow := range jobFlow.Spec.Flows {
		jobName := getJobName(jobFlow.Name, flow.Name)
		namespacedNameJob := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      jobName,
		}
		// job 对象
		job := r.prepareJob(&jobFlow, &jobFlow.Spec.Flows[i], jobName)

		// 如果没拿到这个 job
		if err := r.client.Get(ctx, namespacedNameJob, job); err != nil {
			if errors.IsNotFound(err) {
				// 判斷 job 是否有 Dependencies，
				// 如果沒有，直接創建，如果有，則要判斷 Dependencies 中的 job 是否已經成功
				if len(flow.Dependencies) == 0 {
					// 直接创建
					if err = r.client.Create(ctx, job); err != nil {
						if errors.IsAlreadyExists(err) {
							continue
						}
						return err
					}
					r.event.Eventf(&jobFlow, v1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v without dependencies", job.Name))
				} else {
					// 如果有依赖的情况
					// query dependency meets the requirements
					flag := true
					// 查看依赖的 job 是否已经完成，
					for _, dependencyName := range flow.Dependencies {
						dependencyJobName := getJobName(jobFlow.Name, dependencyName)
						namespacedName := types.NamespacedName{
							Namespace: jobFlow.Namespace,
							Name:      dependencyJobName,
						}
						// 获取依赖的 job
						dependenciesJob := &batchv1.Job{}
						if err = r.client.Get(ctx, namespacedName, dependenciesJob); err != nil {
							if err != nil {
								if errors.IsNotFound(err) {
									klog.Info(fmt.Sprintf("No %v Job found！", namespacedName.Name))
									flag = false
									break
								}
								return err
							}
						}
						// 如果 job 没完成， false，代表不进行下去
						if dependenciesJob.Status.Succeeded != 1 {
							flag = false
						}

						// 如果依赖的 job 出错，直接退出
						if dependenciesJob.Status.Failed == 1 { // TODO: 这里要加入出错逻辑
							return errors.NewBadRequest(fmt.Sprintf("dependencies Job %s execute error", dependenciesJob.Name))
						}
					}
					// 如果已经完成，就进行下去
					if flag {
						if err = r.client.Create(ctx, job); err != nil {
							if errors.IsAlreadyExists(err) {
								break
							}
							return err
						}
						r.event.Eventf(&jobFlow, v1.EventTypeNormal, "Created", fmt.Sprintf("create job named %v for next step", job.Name))
					}
				}
				continue
			}
			return err
		}
	}
	return nil
}

func getJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}

// update status
func (r *JobFlowController) updateJobFlowStatus(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow) error {
	klog.Info(fmt.Sprintf("start to update jobFlow status! jobFlowName: %v, jobFlowNamespace: %v ", jobFlow.Name, jobFlow.Namespace))
	// 获取 job 列表
	allJobList := new(batchv1.JobList)
	err := r.client.List(ctx, allJobList)
	if err != nil {
		klog.Error("list error: ", err)
		return err
	}
	jobFlowStatus, err := getAllJobStatus(jobFlow, allJobList)
	if err != nil {
		return err
	}
	jobFlow.Status = *jobFlowStatus
	if jobFlowStatus.State == jobflowv1alpha1.Succeed || jobFlowStatus.State == jobflowv1alpha1.Failed {
		r.event.Eventf(jobFlow, v1.EventTypeNormal, jobFlowStatus.State, fmt.Sprintf("finshed JobFlow named %s", jobFlow.Name))
	}
	if err = r.client.Status().Update(ctx, jobFlow); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *JobFlowController) findJobTemplateByNameNamespace(name, namespace string) (batchv1.JobSpec, error) {
	// load JobTemplate by namespace
	jobTemplate := &jobtemplatev1alpha1.JobTemplate{}
	time.Sleep(time.Second)
	nn := types.NamespacedName{Name: name, Namespace: namespace}
	err := r.client.Get(context.TODO(), nn, jobTemplate)
	if err != nil {
		// If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("not found JobTemplate : %v", name))
			return jobTemplate.Spec.JobTemplate, nil
		}
		klog.Error(err, err.Error())
		r.event.Eventf(jobTemplate, v1.EventTypeWarning, "Created", err.Error())
		return jobTemplate.Spec.JobTemplate, err
	}
	return jobTemplate.Spec.JobTemplate, nil
}

func (r *JobFlowController) prepareJob(jobFlow *jobflowv1alpha1.JobFlow, flow *jobflowv1alpha1.Flow, jobName string) *batchv1.Job {
	// job 对象
	job := &batchv1.Job{}

	// 设置 ownerReferences
	job.OwnerReferences = append(job.OwnerReferences, metav1.OwnerReference{
		APIVersion: jobFlow.APIVersion,
		Kind:       jobFlow.Kind,
		Name:       jobFlow.Name,
		UID:        jobFlow.UID,
	})

	job.Name = jobName
	job.Namespace = jobFlow.Namespace
	job.Annotations = map[string]string{}
	job.Labels = map[string]string{}

	// FIXME: 这里要判断使用 JobTemplate or JobTemplateRef
	// 如果使用 JobTemplateRef 记得要打上 annotation 标签
	if flow.JobTemplateRef != "" {
		jobTemplateRefSpec, err := r.findJobTemplateByNameNamespace(flow.JobTemplateRef, jobFlow.Namespace)
		if err != nil {
			klog.Error("find JobTemplate failed: ", err)
			return nil
		}
		job.Spec = jobTemplateRefSpec
		// 加上 annotation 注名 此 job 使用此 JobTemplate
		// 用于 JobTemplate 状态标记
		job.Annotations[common.CreateByJobTemplate] =
			common.GetConnectionOfJobAndJobTemplate(jobFlow.Namespace, flow.JobTemplateRef)
	} else {
		job.Spec = flow.JobTemplate
	}

	// 强制设置 job 不重启与重试次数
	job.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyNever
	var cc int32
	job.Spec.BackoffLimit = &cc

	// 加入 flow 全局参数
	if jobFlow.Spec.GlobalParams.Annotations != nil {
		job.Spec.Template.Annotations = map[string]string{}
		for key, value := range jobFlow.Spec.GlobalParams.Annotations {
			job.Annotations[key] = value
			job.Spec.Template.Annotations[key] = value
		}
	}

	if jobFlow.Spec.GlobalParams.Labels != nil {
		job.Spec.Template.Labels = map[string]string{}
		for key, value := range jobFlow.Spec.GlobalParams.Labels {
			job.Labels[key] = value
			job.Spec.Template.Labels[key] = value
		}
	}

	if jobFlow.Spec.GlobalParams.NodeName != "" {
		job.Spec.Template.Spec.NodeName = jobFlow.Spec.GlobalParams.NodeName
	}

	if jobFlow.Spec.GlobalParams.Env != nil {
		for k := range job.Spec.Template.Spec.Containers {
			job.Spec.Template.Spec.Containers[k].Env = jobFlow.Spec.GlobalParams.Env
		}
	}

	return job
}

// getAllJobStatus 记录 Job Status
func getAllJobStatus(jobFlow *jobflowv1alpha1.JobFlow, allJobList *batchv1.JobList) (*jobflowv1alpha1.JobFlowStatus, error) {
	// 过去掉只留 jobflow 相关的 job
	jobListRes := make([]batchv1.Job, 0)
	for _, job := range allJobList.Items {
		for _, reference := range job.OwnerReferences {
			if reference.Kind == jobflowv1alpha1.JobFlowKind && reference.Name == jobFlow.Name {
				jobListRes = append(jobListRes, job)
			}
		}
	}

	runningJobs := make([]string, 0)
	failedJobs := make([]string, 0)
	completedJobs := make([]string, 0)

	jobList := make([]string, 0)

	for _, flow := range jobFlow.Spec.Flows {
		jobList = append(jobList, getJobName(jobFlow.Name, flow.Name))
	}

	jobFlowStatus := jobflowv1alpha1.JobFlowStatus{
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

	// 确认 jobFlow 狀態
	if jobFlow.DeletionTimestamp != nil {
		jobFlowStatus.State = jobflowv1alpha1.Terminating
	} else {
		if len(jobList) != len(completedJobs) {
			switch {
			case len(failedJobs) > 0:
				jobFlowStatus.State = jobflowv1alpha1.Failed
			case len(runningJobs) > 0 || len(completedJobs) > 0:
				jobFlowStatus.State = jobflowv1alpha1.Running
			default:
				jobFlowStatus.State = jobflowv1alpha1.Pending
			}
		} else {
			jobFlowStatus.State = jobflowv1alpha1.Succeed
		}
	}

	return &jobFlowStatus, nil
}

func (r *JobFlowController) OnUpdateJobHandlerByJobFlow(event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.ObjectNew.GetOwnerReferences() {
		if ref.Kind == jobflowv1alpha1.JobFlowKind && ref.APIVersion == jobflowv1alpha1.JobFlowApiVersion {
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ref.Name, Namespace: event.ObjectNew.GetNamespace(),
				},
			})
		}
	}
}

func (r *JobFlowController) OnDeleteJobHandlerByJobFlow(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.Object.GetOwnerReferences() {
		if ref.Kind == jobflowv1alpha1.JobFlowKind && ref.APIVersion == jobflowv1alpha1.JobFlowApiVersion {
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: ref.Name,
					Namespace: event.Object.GetNamespace()}})
		}
	}
}
