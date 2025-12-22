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

// LeaseReconciler reconciles a Lease object
type LeaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sync.konductor.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.konductor.io,resources=leases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=leases/finalizers,verbs=update
//+kubebuilder:rbac:groups=sync.konductor.io,resources=leaserequests,verbs=get;list;watch
//+kubebuilder:rbac:groups=sync.konductor.io,resources=leaserequests/status,verbs=get;update;patch

func (r *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling Lease", "name", req.Name, "namespace", req.Namespace)

	var lease syncv1.Lease
	if err := r.Get(ctx, req.NamespacedName, &lease); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Lease not found, likely deleted", "name", req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Lease")
		return ctrl.Result{}, err
	}

	log.Info("Found Lease", "name", lease.Name, "currentHolder", lease.Status.Holder, "currentPhase", lease.Status.Phase)

	now := time.Now()

	if lease.Status.ExpiresAt != nil && lease.Status.ExpiresAt.Time.Before(now) {
		lease.Status.Phase = syncv1.LeasePhaseExpired
		lease.Status.Holder = ""
		lease.Status.AcquiredAt = nil
		lease.Status.ExpiresAt = nil
	}

	if lease.Status.Holder == "" {
		lease.Status.Phase = syncv1.LeasePhaseAvailable
	}

	requests := &syncv1.LeaseRequestList{}
	if err := r.List(ctx, requests, client.InNamespace(req.Namespace),
		client.MatchingLabels{"lease": lease.Name}); err != nil {
		log.Error(err, "unable to list lease requests")
		return ctrl.Result{}, err
	}

	log.Info("Found lease requests", "count", len(requests.Items), "lease", lease.Name)

	if lease.Status.Phase == syncv1.LeasePhaseAvailable && len(requests.Items) > 0 {
		var bestRequest *syncv1.LeaseRequest
		var highestPriority int32 = -1

		for i := range requests.Items {
			req := &requests.Items[i]
			priority := int32(0)
			if req.Spec.Priority != nil {
				priority = *req.Spec.Priority
			}
			if priority > highestPriority {
				highestPriority = priority
				bestRequest = req
			}
		}

		if bestRequest != nil {
			lease.Status.Holder = bestRequest.Spec.Holder
			lease.Status.Phase = syncv1.LeasePhaseHeld
			acquiredAt := metav1.Now()
			lease.Status.AcquiredAt = &acquiredAt
			expiresAt := metav1.NewTime(now.Add(lease.Spec.TTL.Duration))
			lease.Status.ExpiresAt = &expiresAt
			lease.Status.RenewCount = 0

			bestRequest.Status.Phase = syncv1.LeaseRequestPhaseGranted
			if err := r.Status().Update(ctx, bestRequest); err != nil {
				log.Error(err, "unable to update lease request status")
			}
		}
	}

	if err := r.Status().Update(ctx, &lease); err != nil {
		log.Error(err, "unable to update Lease status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated Lease status", "name", lease.Name, "holder", lease.Status.Holder, "phase", lease.Status.Phase)

	if lease.Status.ExpiresAt != nil {
		return ctrl.Result{RequeueAfter: time.Until(lease.Status.ExpiresAt.Time)}, nil
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *LeaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1.Lease{}).
		Complete(r)
}
