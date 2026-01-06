package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// WaitGroupReconciler reconciles a WaitGroup object
type WaitGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=waitgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=waitgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=waitgroups/finalizers,verbs=update

func (r *WaitGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var wg syncv1.WaitGroup
	if err := r.Get(ctx, req.NamespacedName, &wg); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update phase based on counter
	if wg.Status.Counter <= 0 {
		wg.Status.Phase = syncv1.WaitGroupPhaseDone
	} else {
		wg.Status.Phase = syncv1.WaitGroupPhaseWaiting
	}

	if err := r.Status().Update(ctx, &wg); err != nil {
		log.Error(err, "unable to update WaitGroup status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WaitGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.WaitGroup{}).
		Complete(r)
}
