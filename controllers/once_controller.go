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
		log.Error(err, "unable to fetch Once")
		return ctrl.Result{}, err
	}

	// Initialize status if needed
	if once.Status.Phase == "" {
		once.Status.Phase = syncv1.OncePhasePending
		once.Status.Executed = false
		if err := r.Status().Update(ctx, &once); err != nil {
			log.Error(err, "unable to initialize Once status")
			return ctrl.Result{RequeueAfter: time.Second}, err
		}
		log.Info("Initialized Once status", "name", once.Name)
		return ctrl.Result{}, nil
	}

	// If already executed, ensure phase is correct
	if once.Status.Executed {
		if once.Status.Phase != syncv1.OncePhaseExecuted {
			once.Status.Phase = syncv1.OncePhaseExecuted
			if err := r.Status().Update(ctx, &once); err != nil {
				log.Error(err, "unable to update Once phase")
				return ctrl.Result{RequeueAfter: time.Second}, err
			}
			log.Info("Updated Once phase to Executed", "name", once.Name)
		}
		return ctrl.Result{}, nil
	}

	// Once is pending and not executed - no action needed
	// External processes will mark it as executed when they complete
	return ctrl.Result{}, nil

}

func (r *OnceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Once{}).
		Complete(r)
}
