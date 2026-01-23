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

// SemaphoreReconciler reconciles a Semaphore object
type SemaphoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=semaphores/finalizers,verbs=update
//+kubebuilder:rbac:groups=sync.konductor.io,resources=permits,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=permits/status,verbs=get;update;patch

func (r *SemaphoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling Semaphore", "name", req.Name, "namespace", req.Namespace)

	var semaphore syncv1.Semaphore
	if err := r.Get(ctx, req.NamespacedName, &semaphore); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Semaphore not found, likely deleted", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Semaphore")
		return ctrl.Result{}, err
	}

	log.Info("Found Semaphore", "name", semaphore.Name, "permits", semaphore.Spec.Permits, "currentAvailable", semaphore.Status.Available)

	if semaphore.Status.Available == 0 && semaphore.Status.InUse == 0 && semaphore.Status.Phase == "" {
		semaphore.Status.Available = semaphore.Spec.Permits
		semaphore.Status.InUse = 0
		semaphore.Status.Phase = syncv1.SemaphorePhaseReady
		if err := r.Status().Update(ctx, &semaphore); err != nil {
			log.Error(err, "unable to initialize Semaphore status")
			return ctrl.Result{}, err
		}
		log.Info("Initialized Semaphore status", "name", semaphore.Name)
		return ctrl.Result{}, nil
	}

	permits := &syncv1.PermitList{}
	if err := r.List(ctx, permits, client.InNamespace(req.Namespace),
		client.MatchingLabels{"semaphore": semaphore.Name}); err != nil {
		log.Error(err, "unable to list permits")
		return ctrl.Result{}, err
	}

	log.Info("Found permits", "count", len(permits.Items), "semaphore", semaphore.Name)

	validPermits := 0
	now := time.Now()
	for i, permit := range permits.Items {
		isValid := permit.Status.ExpiresAt == nil || permit.Status.ExpiresAt.Time.After(now)
		if isValid {
			if permit.Status.Phase != syncv1.PermitPhaseGranted {
				permits.Items[i].Status.Phase = syncv1.PermitPhaseGranted
				if err := r.Status().Update(ctx, &permits.Items[i]); err != nil {
					log.Error(err, "failed to update permit status", "permit", permit.Name)
					return ctrl.Result{}, err
				}
			}
			validPermits++
		}
	}

	oldInUse := semaphore.Status.InUse
	oldAvailable := semaphore.Status.Available
	oldPhase := semaphore.Status.Phase

	semaphore.Status.InUse = int32(validPermits)
	semaphore.Status.Available = semaphore.Spec.Permits - int32(validPermits)

	if semaphore.Status.Available > 0 {
		semaphore.Status.Phase = syncv1.SemaphorePhaseReady
	} else {
		semaphore.Status.Phase = syncv1.SemaphorePhaseFull
	}

	log.Info("Status update", "semaphore", semaphore.Name,
		"validPermits", validPermits,
		"oldInUse", oldInUse, "newInUse", semaphore.Status.InUse,
		"oldAvailable", oldAvailable, "newAvailable", semaphore.Status.Available,
		"oldPhase", oldPhase, "newPhase", semaphore.Status.Phase)

	if err := r.Status().Update(ctx, &semaphore); err != nil {
		log.Error(err, "unable to update Semaphore status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated Semaphore status", "name", semaphore.Name)

	// Use adaptive requeue interval based on activity
	requeueAfter := time.Minute
	if oldInUse != semaphore.Status.InUse || oldAvailable != semaphore.Status.Available {
		// Active changes detected, requeue sooner
		requeueAfter = 10 * time.Second
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *SemaphoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Semaphore{}).
		Owns(&syncv1.Permit{}).
		Complete(r)
}
