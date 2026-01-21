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

// MutexReconciler reconciles a Mutex object
type MutexReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=mutexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=mutexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=mutexes/finalizers,verbs=update

func (r *MutexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var mutex syncv1.Mutex
	if err := r.Get(ctx, req.NamespacedName, &mutex); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	now := time.Now()
	updated := false

	// Check TTL expiration
	if mutex.Status.ExpiresAt != nil && mutex.Status.ExpiresAt.Time.Before(now) {
		mutex.Status.Phase = syncv1.MutexPhaseUnlocked
		mutex.Status.Holder = ""
		mutex.Status.LockedAt = nil
		mutex.Status.ExpiresAt = nil
		updated = true
	}

	if mutex.Status.Holder == "" && mutex.Status.Phase != syncv1.MutexPhaseUnlocked {
		mutex.Status.Phase = syncv1.MutexPhaseUnlocked
		updated = true
	}

	if updated {
		if err := r.Status().Update(ctx, &mutex); err != nil {
			log.Error(err, "unable to update Mutex status")
			return ctrl.Result{}, err
		}
	}

	// Requeue if TTL is set
	if mutex.Status.ExpiresAt != nil && mutex.Status.ExpiresAt.Time.After(now) {
		return ctrl.Result{RequeueAfter: time.Until(mutex.Status.ExpiresAt.Time)}, nil
	}

	return ctrl.Result{}, nil
}

func (r *MutexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Mutex{}).
		Complete(r)
}
