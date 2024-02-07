package controller

import (
	"context"
	daemonjobv1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/daemonjob/v1alpha1"
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

func createDaemonJobController(initObjs ...client.Object) *DaemonJobController {
	scheme := runtime.NewScheme()
	// 加入 scheme
	utilruntime.Must(daemonjobv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: "daemonjob-controller"})
	logf.SetLogger(zap.New())
	log := logf.Log.WithName("JobFlow operator")
	return NewDaemonJobController(fakeClient, log, scheme, recorder)
}

func TestDaemonJobController_Reconcile(t *testing.T) {
	Convey("Test DaemonJob Reconcile", t, func() {
		node1 := createNode("node1")
		node2 := createNode("node2")
		node3 := createNode("node3")
		daemonjob := createDaemonJob("daemonjob-test")
		reconcileController := createDaemonJobController(daemonjob, node1, node2, node3)
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "daemonjob-test",
				Namespace: "default",
			},
		}
		_, err := reconcileController.Reconcile(context.TODO(), request)
		So(err, ShouldBeNil)

		job1 := newJobFromDaemonJob(daemonjob, "daemonjob-test-node1")
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

func createDaemonJob(jobName string) *daemonjobv1alpha1.DaemonJob {

	daemonjob := &daemonjobv1alpha1.DaemonJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			UID:       "12345",
		},
		Spec: daemonjobv1alpha1.DaemonJobSpec{
			GlobalParams: daemonjobv1alpha1.GlobalParams{
				Labels: map[string]string{
					"key": "value",
				},
				Annotations: map[string]string{
					"key": "value",
				},
				Env: []v1.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
			ExcludeNodeList: "node3",
			JobSpec: batchv1.JobSpec{
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
		},
	}
	return daemonjob
}

func newJobFromDaemonJob(daemonjob *daemonjobv1alpha1.DaemonJob, jobName string) *batchv1.Job {
	job1Pod := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(daemonjob, jobflowv1alpha1.SchemeGroupVersion.WithKind("DaemonJob")),
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

func createNode(nodeName string) *v1.Node {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: "default",
		},
		Spec: v1.NodeSpec{},
		Status: v1.NodeStatus{
			Phase: v1.NodeRunning,
		},
	}
	return node
}
