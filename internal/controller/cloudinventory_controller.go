package controller

import (
  "context"

   ctrl "sigs.k8s.io/controller-runtime"
   "k8s.io/apimachinery/pkg/runtime"
   "time"
   metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "sigs.k8s.io/controller-runtime/pkg/client"
  "sigs.k8s.io/controller-runtime/pkg/log"

  v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

// CloudInventoryReconciler reconciles a CloudInventory object
type CloudInventoryReconciler struct {
  client.Client
  Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=cloudinventories/status,verbs=get;update;patch

func (r *CloudInventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
  logger := log.FromContext(ctx).WithValues("cloudinventory", req.NamespacedName)

  // 1) fetch the resource
  var ci v1alpha1.CloudInventory
  if err := r.Get(ctx, req.NamespacedName, &ci); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
  }

  // 2) (placeholder) perform your inventory logic here,
  //    e.g. call out to cloud API, count items, etc.
  logger.Info("Running CloudInventory for region", "region", ci.Spec.Region)

  // 3) update status
  ci.Status.LastRunTime = metav1.Now()
  ci.Status.ItemCount = 42                    // <-- replace with real count
  ci.Status.Message = "Inventory complete"    // <-- or errors
  if err := r.Status().Update(ctx, &ci); err != nil {
    logger.Error(err, "unable to update CloudInventory status")
    return ctrl.Result{}, err
  }

  // requeue after X if you want periodic runs
  return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func (r *CloudInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
  return ctrl.NewControllerManagedBy(mgr).
    For(&v1alpha1.CloudInventory{}).
    Complete(r)
}
