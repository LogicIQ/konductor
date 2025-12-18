package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// SemaphoreReconciler reconciles a Semaphore object
type SemaphoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores/finalizers,verbs=update
//+kubebuilder:rbac:groups=sync.konductor.io,resources=permits,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *SemaphoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Semaphore instance
	var semaphore syncv1.Semaphore
	if err := r.Get(ctx, req.NamespacedName, &semaphore); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Semaphore")
		return ctrl.Result{}, err
	}

	// Count active permits by looking for Permit CRs
	permits := &syncv1.PermitList{}
	if err := r.List(ctx, permits, client.InNamespace(req.Namespace), 
		client.MatchingLabels{"semaphore": semaphore.Name}); err != nil {
		log.Error(err, "unable to list permits")
		return ctrl.Result{}, err
	}

	// Count valid (non-expired) permits
	validPermits := 0
	now := time.Now()
	for _, permit := range permits.Items {
		if permit.Status.ExpiresAt != nil && permit.Status.ExpiresAt.Time.After(now) {
			validPermits++
		} else if permit.Status.ExpiresAt == nil {
			validPermits++ // No expiration set
		}
	}

	// Update status
	semaphore.Status.InUse = int32(validPermits)
	semaphore.Status.Available = semaphore.Spec.Permits - int32(validPermits)
	
	if semaphore.Status.Available > 0 {
		semaphore.Status.Phase = syncv1.SemaphorePhaseReady
	} else {
		semaphore.Status.Phase = syncv1.SemaphorePhaseFull
	}

	// Update the status
	if err := r.Status().Update(ctx, &semaphore); err != nil {
		log.Error(err, "unable to update Semaphore status")
		return ctrl.Result{}, err
	}

	// Requeue to check for expired permits
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SemaphoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Semaphore{}).
		Complete(r)
}