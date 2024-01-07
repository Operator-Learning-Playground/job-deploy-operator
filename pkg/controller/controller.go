package controller

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type JobFlowController struct {
	client client.Client
	Scheme *runtime.Scheme
	log    logr.Logger
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

	return reconcile.Result{}, nil
}
