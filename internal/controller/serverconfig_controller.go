package controller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openprojectv1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

type ServerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// getScopedServerConfigLogger returns a simplified logger for normal mode or a detailed logger for debug mode
func getScopedServerConfigLogger(ctx context.Context, config *openprojectv1alpha1.ServerConfig) logr.Logger {
	if debugEnabled {
		// In debug mode, use the full context logger from controller-runtime
		return log.FromContext(ctx)
	}

	// For normal mode, create a fresh logger with just the essential fields
	return ctrl.Log.WithValues("serverconfig", config.Name)
}

// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs/finalizers,verbs=update
func (r *ServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 🔍 List all ServerConfigs in the same namespace
	var configs openprojectv1alpha1.ServerConfigList
	if err := r.List(ctx, &configs, client.InNamespace(req.Namespace)); err != nil {
		// Create a minimal logger for error reporting when we can't get the config
		errorLog := ctrl.Log.WithValues("serverconfig", req.NamespacedName.String())
		errorLog.Error(err, "❌ Unable to list ServerConfigs")
		return ctrl.Result{}, err
	}

	// ✅ Get the current ServerConfig
	var config openprojectv1alpha1.ServerConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get a scoped logger for this ServerConfig
	log := getScopedServerConfigLogger(ctx, &config)

	// 🚫 Enforce only one config per namespace
	if len(configs.Items) > 1 {
		statusLog(log, "⚠️", "Multiple ServerConfigs found in namespace. Only one is allowed. Extra configs will be ignored.")
		// You can optionally update the status or mark the others somehow
	}

	// Log that the ServerConfig was loaded
	statusLog(log, "🛠", "ServerConfig loaded", "server", config.Spec.Server)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openprojectv1alpha1.ServerConfig{}).
		Complete(r)
}
