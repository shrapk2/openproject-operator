/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openprojectv1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

// ServerConfigReconciler reconciles a ServerConfig object
type ServerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openproject.org,resources=serverconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServerConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	//USER LOGIC
	log := log.FromContext(ctx)
	// 🔍 List all ServerConfigs in the same namespace
	var configs openprojectv1alpha1.ServerConfigList
	if err := r.List(ctx, &configs, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "unable to list ServerConfigs")
		return ctrl.Result{}, err
	}

	// 🚫 Enforce only one config per namespace
	if len(configs.Items) > 1 {
		log.Info("⚠️ Multiple ServerConfigs found in namespace. Only one is allowed. Extra configs will be ignored.")
		// You can optionally update the status or mark the others somehow
	}

	// ✅ Get the current ServerConfig
	var config openprojectv1alpha1.ServerConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//USER LOGIC
	log.Info("✅ ServerConfig loaded", "server", config.Spec.Server)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openprojectv1alpha1.ServerConfig{}).
		Complete(r)
}
