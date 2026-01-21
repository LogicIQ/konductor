package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

// BarrierReconciler reconciles a Barrier object
type BarrierReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=barriers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=barriers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=barriers/finalizers,verbs=update
//+kubebuilder:rbac:groups=sync.konductor.io,resources=arrivals,verbs=get;list;watch

func (r *BarrierReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling Barrier", "name", req.Name, "namespace", req.Namespace)

	var barrier syncv1.Barrier
	if err := r.Get(ctx, req.NamespacedName, &barrier); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Barrier not found, likely deleted", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Barrier")
		return ctrl.Result{}, err
	}

	log.Info("Found Barrier", "name", barrier.Name, "expected", barrier.Spec.Expected, "currentArrived", barrier.Status.Arrived)

	arrivals := &syncv1.ArrivalList{}
	if err := r.List(ctx, arrivals, client.InNamespace(req.Namespace),
		client.MatchingLabels{"barrier": barrier.Name}); err != nil {
		log.Error(err, "unable to list arrivals")
		return ctrl.Result{}, err
	}

	log.Info("Found arrivals", "count", len(arrivals.Items), "barrier", barrier.Name)

	barrier.Status.Arrived = int32(len(arrivals.Items))
	barrier.Status.Arrivals = make([]string, len(arrivals.Items))
	for i, arrival := range arrivals.Items {
		barrier.Status.Arrivals[i] = arrival.Spec.Holder
	}

	requiredArrivals := barrier.Spec.Expected
	if barrier.Spec.Quorum != nil {
		requiredArrivals = *barrier.Spec.Quorum
	}

	var newPhase syncv1.BarrierPhase
	if barrier.Spec.Timeout != nil && barrier.CreationTimestamp.Add(barrier.Spec.Timeout.Duration).Before(time.Now()) {
		if barrier.Status.Arrived < requiredArrivals {
			newPhase = syncv1.BarrierPhaseFailed
		} else {
			newPhase = barrier.Status.Phase
		}
	} else if barrier.Status.Arrived >= requiredArrivals {
		newPhase = syncv1.BarrierPhaseOpen
		if barrier.Status.OpenedAt == nil {
			now := metav1.Now()
			barrier.Status.OpenedAt = &now
		}
	} else {
		newPhase = syncv1.BarrierPhaseWaiting
	}

	if barrier.Status.Phase != newPhase {
		barrier.Status.Phase = newPhase
		if err := r.Status().Update(ctx, &barrier); err != nil {
			log.Error(err, "unable to update Barrier status")
			return ctrl.Result{}, err
		}
		log.Info("Successfully updated Barrier status", "name", barrier.Name, "arrived", barrier.Status.Arrived, "phase", barrier.Status.Phase)
	}

	if barrier.Spec.Timeout != nil && barrier.Status.Phase == syncv1.BarrierPhaseWaiting {
		timeoutAt := barrier.CreationTimestamp.Add(barrier.Spec.Timeout.Duration)
		requeueAfter := time.Until(timeoutAt)
		if requeueAfter > time.Minute {
			requeueAfter = time.Minute
		} else if requeueAfter < 0 {
			requeueAfter = 0
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *BarrierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Barrier{}).
		Owns(&syncv1.Arrival{}).
		Complete(r)
}
