package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

type CloudInventoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories/status,verbs=get;update;patch

func (r *CloudInventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cloudinventory", req.NamespacedName)

	var ci v1alpha1.CloudInventory
	if err := r.Get(ctx, req.NamespacedName, &ci); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	switch ci.Spec.Mode {
	case "aws":
		return r.reconcileAWS(ctx, &ci, log)
	case "kubernetes":
		return r.reconcileKubernetes(ctx, &ci, log)
	default:
		err := fmt.Errorf("unsupported inventory mode: %s", ci.Spec.Mode)
		log.Error(err, "invalid mode")
		return ctrl.Result{}, err
	}
}

func (r *CloudInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CloudInventory{}).
		Complete(r)
}
