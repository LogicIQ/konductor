package controllers

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// GateReconciler reconciles a Gate object
type GateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=gates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=gates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=gates/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch

func (r *GateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling Gate", "name", req.Name, "namespace", req.Namespace)

	var gate syncv1.Gate
	if err := r.Get(ctx, req.NamespacedName, &gate); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Gate not found, likely deleted", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Gate")
		return ctrl.Result{}, err
	}

	log.Info("Found Gate", "name", gate.Name, "conditions", len(gate.Spec.Conditions), "currentPhase", gate.Status.Phase)

	allMet := true
	conditionStatuses := make([]syncv1.GateConditionStatus, len(gate.Spec.Conditions))

	for i, condition := range gate.Spec.Conditions {
		status := syncv1.GateConditionStatus{
			Type: condition.Type,
			Name: condition.Name,
			Met:  false,
		}

		namespace := condition.Namespace
		if namespace == "" {
			namespace = gate.Namespace
		}

		switch condition.Type {
		case "Job":
			var job batchv1.Job
			if err := r.Get(ctx, client.ObjectKey{Name: condition.Name, Namespace: namespace}, &job); err != nil {
				if errors.IsNotFound(err) {
					log.V(1).Info("Job not found for gate condition", "job", condition.Name, "namespace", namespace)
					status.Message = "Job not found"
				} else {
					log.Error(err, "Failed to get Job for gate condition", "job", condition.Name, "namespace", namespace)
					status.Message = "Failed to get Job"
				}
				allMet = false
			} else {
				if condition.State == "Complete" && job.Status.Succeeded > 0 {
					status.Met = true
					status.Message = "Job completed successfully"
				} else {
					status.Message = "Job not in required state"
					allMet = false
				}
			}

		case "Semaphore":
			var semaphore syncv1.Semaphore
			if err := r.Get(ctx, client.ObjectKey{Name: condition.Name, Namespace: namespace}, &semaphore); err != nil {
				status.Message = "Semaphore not found"
				allMet = false
			} else {
				if condition.Value != nil && semaphore.Status.Available >= *condition.Value {
					status.Met = true
					status.Message = "Semaphore has required permits"
				} else {
					status.Message = "Semaphore does not have required permits"
					allMet = false
				}
			}

		case "Barrier":
			var barrier syncv1.Barrier
			if err := r.Get(ctx, client.ObjectKey{Name: condition.Name, Namespace: namespace}, &barrier); err != nil {
				status.Message = "Barrier not found"
				allMet = false
			} else {
				if condition.State == "Open" && barrier.Status.Phase == syncv1.BarrierPhaseOpen {
					status.Met = true
					status.Message = "Barrier is open"
				} else {
					status.Message = "Barrier is not open"
					allMet = false
				}
			}

		case "Lease":
			var lease syncv1.Lease
			if err := r.Get(ctx, client.ObjectKey{Name: condition.Name, Namespace: namespace}, &lease); err != nil {
				status.Message = "Lease not found"
				allMet = false
			} else {
				if condition.State == "Available" && lease.Status.Phase == syncv1.LeasePhaseAvailable {
					status.Met = true
					status.Message = "Lease is available"
				} else {
					status.Message = "Lease is not available"
					allMet = false
				}
			}

		default:
			status.Message = "Unknown condition type"
			allMet = false
		}

		conditionStatuses[i] = status
	}

	gate.Status.ConditionStatuses = conditionStatuses

	if allMet {
		gate.Status.Phase = syncv1.GatePhaseOpen
		if gate.Status.OpenedAt == nil {
			now := metav1.Now()
			gate.Status.OpenedAt = &now
		}
	} else {
		if gate.Spec.Timeout != nil && gate.CreationTimestamp.Add(gate.Spec.Timeout.Duration).Before(time.Now()) {
			gate.Status.Phase = syncv1.GatePhaseFailed
		} else {
			gate.Status.Phase = syncv1.GatePhaseWaiting
		}
	}

	if err := r.Status().Update(ctx, &gate); err != nil {
		log.Error(err, "unable to update Gate status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated Gate status", "name", gate.Name, "phase", gate.Status.Phase, "allMet", allMet)

	if gate.Status.Phase == syncv1.GatePhaseWaiting {
		requeueAfter := 10 * time.Second
		if gate.Spec.Timeout != nil {
			remaining := time.Until(gate.CreationTimestamp.Add(gate.Spec.Timeout.Duration))
			if remaining > 0 {
				// Use 10% of remaining time, bounded between 1s and 30s
				requeueAfter = remaining / 10
				if requeueAfter < time.Second {
					requeueAfter = time.Second
				} else if requeueAfter > 30*time.Second {
					requeueAfter = 30 * time.Second
				}
			}
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *GateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Gate{}).
		Complete(r)
}
