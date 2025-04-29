package controller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

// CloudInventoryReconciler reconciles a CloudInventory object
type CloudInventoryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventoryreports,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list

// Reconcile entry point only run when called by WorkPackage
func (r *CloudInventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("CloudInventory", req.NamespacedName)

	var ci v1alpha1.CloudInventory
	if err := r.Get(ctx, req.NamespacedName, &ci); err != nil {
		log.Error(err, "unable to fetch CloudInventory")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager wires up the controller to the manager
func (r *CloudInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CloudInventory{}).
		Complete(r)
}
