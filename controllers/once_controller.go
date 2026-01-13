package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// OnceReconciler reconciles a Once object
type OnceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=onces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=onces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=onces/finalizers,verbs=update

func (r *OnceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var once syncv1.Once
	if err := r.Get(ctx, req.NamespacedName, &once); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize status if needed
	updateNeeded := false
	if once.Status.Phase == "" {
		once.Status.Phase = syncv1.OncePhasePending
		updateNeeded = true
	}
	if once.Status.Executed && once.Status.Phase != syncv1.OncePhaseExecuted {
		once.Status.Phase = syncv1.OncePhaseExecuted
		updateNeeded = true
	}

	if !updateNeeded {
		return ctrl.Result{}, nil
	}

	if err := r.Status().Update(ctx, &once); err != nil {
		log.Error(err, "unable to update Once status")
		if errors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	}

	return ctrl.Result{}, nil
}

func (r *OnceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Once{}).
		Complete(r)
}
