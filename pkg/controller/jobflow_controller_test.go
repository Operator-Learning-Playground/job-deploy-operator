package controller

import (
	"context"
	jobflowv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobflow/v1alpha1"
	. "github.com/smartystreets/goconvey/convey"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

func createJobFlowController(initObjs ...client.Object) *JobFlowController {
	scheme := runtime.NewScheme()
	// 加入 scheme
	utilruntime.Must(jobflowv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: "jobflow-controller"})

	logf.SetLogger(zap.New())
	log := logf.Log.WithName("JobFlow operator")
	return NewJobFlowController(fakeClient, log, scheme, recorder)
}

func TestJobFlowController_Reconcile(t *testing.T) {
	Convey("Test JobFlow Reconcile", t, func() {
		jobflow := createJobFlow("jobflow-test")
		reconcileController := createJobFlowController(jobflow)
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "jobflow-test",
				Namespace: "default",
			},
		}
		_, err := reconcileController.Reconcile(context.TODO(), request)
		So(err, ShouldBeNil)

		job1 := newJob(jobflow, "jobflow-test-jobflow-test1")
		// 过两秒后更新 job1 状态
		select {
		case <-time.After(time.Second * 2):
			job1.Status = batchv1.JobStatus{
				Succeeded: 1,
			}
			reconcileController.client.Status().Update(context.TODO(), job1)
		}
		_, err = reconcileController.Reconcile(context.TODO(), request)
		So(err, ShouldBeNil)
	})
}

func createJobFlow(jobName string) *jobflowv1alpha1.JobFlow {
	flows := make([]jobflowv1alpha1.Flow, 0)
	flowTest1 := jobflowv1alpha1.Flow{
		Name: "jobflow-test1",
		JobTemplate: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: "test-images",
						},
					},
				},
			},
		},
	}
	flowTest2 := jobflowv1alpha1.Flow{
		Name: "jobflow-test2",
		JobTemplate: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: "test-images",
						},
					},
				},
			},
		},
		Dependencies: []string{"jobflow-test1"},
	}
	flows = append(flows, flowTest1)
	flows = append(flows, flowTest2)

	job1 := &jobflowv1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},
		Spec: jobflowv1alpha1.JobFlowSpec{
			Flows: flows,
		},
	}
	return job1
}

func createPod(job1 *jobflowv1alpha1.JobFlow) *v1.Pod {
	job1Pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job1.Name,
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(job1, jobflowv1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test",
					Image: "test-images",
				},
			},
		},
	}
	return job1Pod
}

func newJob(job1 *jobflowv1alpha1.JobFlow, jobName string) *batchv1.Job {
	job1Pod := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(job1, jobflowv1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: "test-image",
						},
					},
				},
			},
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	return job1Pod
}
