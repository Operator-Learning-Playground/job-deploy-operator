package controller

import (
	"context"
	"fmt"
	jobflowv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobflow/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// deploy job by dependence order.
func (r *JobFlowController) deployJobFlow(ctx context.Context, jobFlow jobflowv1alpha1.JobFlow) error {
	// 启动 job
	for _, flow := range jobFlow.Spec.Flows {
		// job 对象
		job := &batchv1.Job{}
		jobName := getJobName(jobFlow.Name, flow.Name)
		namespacedNameJob := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      jobName,
		}

		// 设置 ownerReferences
		job.OwnerReferences = append(job.OwnerReferences, metav1.OwnerReference{
			APIVersion: jobFlow.APIVersion,
			Kind:       jobFlow.Kind,
			Name:       jobFlow.Name,
			UID:        jobFlow.UID,
		})

		// 如果没拿到这个 job
		if err := r.client.Get(ctx, namespacedNameJob, job); err != nil {
			if errors.IsNotFound(err) {
				// 判斷 job 是否有 Dependencies，
				// 如果沒有，直接創建，如果有，則要判斷 Dependencies 中的 job 是否已經成功
				if len(flow.Dependencies) == 0 {
					// 获取到 jobTemplate
					job.Name = jobName
					job.Namespace = jobFlow.Namespace
					job.Spec = flow.JobTemplate
					// 直接创建
					if err = r.client.Create(ctx, job); err != nil {
						if errors.IsAlreadyExists(err) {
							continue
						}
						return err
					}
					r.Recorder.Eventf(&jobFlow, v1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v without dependencies", job.Name))
				} else {
					// 如果有依赖的情况
					// query dependency meets the requirements
					flag := true
					// 查看依赖的 job 是否已经完成，
					for _, targetName := range flow.Dependencies {
						dependenciesJob := &batchv1.Job{}

						job.Name = jobName
						job.Namespace = jobFlow.Namespace
						job.Spec = flow.JobTemplate

						targetJobName := getJobName(jobFlow.Name, targetName)
						namespacedName := types.NamespacedName{
							Namespace: jobFlow.Namespace,
							Name:      targetJobName,
						}
						// 获取 job
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
					}
					// 如果已经完成，就进行下去
					if flag {
						if err = r.client.Create(ctx, job); err != nil {
							if errors.IsAlreadyExists(err) {
								break
							}
							return err
						}
						r.Recorder.Eventf(&jobFlow, v1.EventTypeNormal, "Created", fmt.Sprintf("create job named %v for next step", job.Name))
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
		klog.Error(err, "")
		return err
	}
	jobFlowStatus, err := getAllJobStatus(jobFlow, allJobList)
	if err != nil {
		return err
	}
	jobFlow.Status = *jobFlowStatus
	if err = r.client.Status().Update(ctx, jobFlow); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

const (
	JobFlow = "JobFlow"
)

// getAllJobStatus Get the information of all created jobs
func getAllJobStatus(jobFlow *jobflowv1alpha1.JobFlow, allJobList *batchv1.JobList) (*jobflowv1alpha1.JobFlowStatus, error) {
	// 过去掉只留 job flow 相关的
	jobListRes := make([]batchv1.Job, 0)
	for _, job := range allJobList.Items {
		for _, reference := range job.OwnerReferences {
			if reference.Kind == JobFlow && reference.Name == jobFlow.Name {
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
		a := fmt.Sprintf("%s/%s", job.Name, job.Namespace)
		jobFlowStatus.JobStatusList[a] = job.Status

		if job.Status.Succeeded == 1 {
			completedJobs = append(completedJobs, job.Name)
		} else if job.Status.Failed == 1 {
			failedJobs = append(failedJobs, job.Name)
		} else if job.Status.Active == 1 {
			runningJobs = append(runningJobs, job.Name)
		}
	}

	// 确认 jobflow 狀態
	if jobFlow.DeletionTimestamp != nil {
		jobFlowStatus.State = jobflowv1alpha1.Terminating
	} else {
		if len(jobList) != len(completedJobs) {
			if len(failedJobs) > 0 {
				jobFlowStatus.State = jobflowv1alpha1.Failed
			} else if len(runningJobs) > 0 || len(completedJobs) > 0 {
				jobFlowStatus.State = jobflowv1alpha1.Running
			} else {
				jobFlowStatus.State = jobflowv1alpha1.Pending
			}
		} else {
			jobFlowStatus.State = jobflowv1alpha1.Succeed
		}
	}

	return &jobFlowStatus, nil
}

func (r *JobFlowController) OnUpdateJobHandler(event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.ObjectNew.GetOwnerReferences() {
		fmt.Println("ccccccc")
		if ref.Kind == jobflowv1alpha1.JobFlowKind && ref.APIVersion == jobflowv1alpha1.JobFlowApiVersion {
			// 重新放入Reconcile调协方法
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ref.Name, Namespace: event.ObjectNew.GetNamespace(),
				},
			})
		}
	}
}

func (r *JobFlowController) OnDeleteJobHandler(event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {
	for _, ref := range event.Object.GetOwnerReferences() {
		fmt.Println("ddddddddd")
		if ref.Kind == jobflowv1alpha1.JobFlowKind && ref.APIVersion == jobflowv1alpha1.JobFlowApiVersion {
			// 重新入列，这样删除pod后，就会进入调和loop，发现ownerReference还在，会立即创建出新的pod。
			klog.Info("delete pod: ", event.Object.GetName(), event.Object.GetObjectKind())
			limitingInterface.Add(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: ref.Name,
					Namespace: event.Object.GetNamespace()}})
		}
	}
}
