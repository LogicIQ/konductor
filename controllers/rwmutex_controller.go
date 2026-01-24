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

// RWMutexReconciler reconciles a RWMutex object
type RWMutexReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=rwmutexes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=rwmutexes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=rwmutexes/finalizers,verbs=update

func (r *RWMutexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var rwmutex syncv1.RWMutex
	if err := r.Get(ctx, req.NamespacedName, &rwmutex); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	now := time.Now()
	updated := false

	// Check TTL expiration
	if rwmutex.Status.ExpiresAt != nil && rwmutex.Status.ExpiresAt.Time.Before(now) {
		rwmutex.Status.Phase = syncv1.RWMutexPhaseUnlocked
		rwmutex.Status.WriteHolder = ""
		rwmutex.Status.ReadHolders = nil
		rwmutex.Status.LockedAt = nil
		rwmutex.Status.ExpiresAt = nil
		updated = true
	}

	// Update phase based on holders
	if rwmutex.Status.WriteHolder == "" && len(rwmutex.Status.ReadHolders) == 0 {
		if rwmutex.Status.Phase != syncv1.RWMutexPhaseUnlocked {
			rwmutex.Status.Phase = syncv1.RWMutexPhaseUnlocked
			updated = true
		}
	} else if rwmutex.Status.WriteHolder != "" {
		if rwmutex.Status.Phase != syncv1.RWMutexPhaseWriteLocked {
			rwmutex.Status.Phase = syncv1.RWMutexPhaseWriteLocked
			updated = true
		}
	} else if len(rwmutex.Status.ReadHolders) > 0 {
		if rwmutex.Status.Phase != syncv1.RWMutexPhaseReadLocked {
			rwmutex.Status.Phase = syncv1.RWMutexPhaseReadLocked
			updated = true
		}
	}

	if updated {
		if err := r.Status().Update(ctx, &rwmutex); err != nil {
			log.Error(err, "unable to update RWMutex status")
			return ctrl.Result{}, err
		}
	}

	// Requeue if TTL is set and not expired
	if rwmutex.Status.ExpiresAt != nil && rwmutex.Status.ExpiresAt.Time.After(now) {
		return ctrl.Result{RequeueAfter: time.Until(rwmutex.Status.ExpiresAt.Time)}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RWMutexReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.RWMutex{}).
		Complete(r)
}
