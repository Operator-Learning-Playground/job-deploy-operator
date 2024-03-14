package controller

import (
	"context"
	jobtemplatev1alpha1 "github.com/myoperator/jobflowoperator/pkg/apis/jobTemplate/v1alpha1"
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

func TestJobTemplateController_Reconcile(t *testing.T) {
	Convey("Test JobFlow Reconcile", t, func() {
		jobtemplate := createJobTemplate("jobtemplate-test")
		reconcileController := createJobTemplateController(jobtemplate)
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "jobtemplate-test",
				Namespace: "default",
			},
		}
		_, err := reconcileController.Reconcile(context.TODO(), request)
		So(err, ShouldBeNil)

		// 过两秒后更新 job1 状态
		select {
		case <-time.After(time.Second * 2):
			jobtemplate.Status = jobtemplatev1alpha1.JobTemplateStatus{}
			reconcileController.client.Status().Update(context.TODO(), jobtemplate)
		}
		_, err = reconcileController.Reconcile(context.TODO(), request)
		So(err, ShouldBeNil)
	})
}

func createJobTemplateController(initObjs ...client.Object) *JobTemplateController {
	scheme := runtime.NewScheme()
	// 加入 scheme
	utilruntime.Must(jobtemplatev1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme, v1.EventSource{Component: "jobtemplate-controller"})

	logf.SetLogger(zap.New())
	log := logf.Log.WithName("JobTemplate operator")
	return NewJobTemplateController(fakeClient, log, scheme, recorder)
}

func createJobTemplate(jobTemplateName string) *jobtemplatev1alpha1.JobTemplate {

	jobTemplate := &jobtemplatev1alpha1.JobTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobTemplateName,
			Namespace: "default",
			UID:       "12345",
		},
		Spec: jobtemplatev1alpha1.JobTemplateSpec{
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
		},
	}
	return jobTemplate
}
