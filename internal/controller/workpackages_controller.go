package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openprojectv1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
	"github.com/shrapk2/openproject-operator/internal/configloader"
)

// WorkPackagesReconciler reconciles a WorkPackages object
type WorkPackageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=openproject.org,resources=workpackages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openproject.org,resources=workpackages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=openproject.org,resources=workpackages/finalizers,verbs=update

func (r *WorkPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var wp openprojectv1alpha1.WorkPackages
	if err := r.Get(ctx, req.NamespacedName, &wp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// config, err := configloader.LoadServerConfig(ctx, r.Client, req.Namespace)
	// if err != nil {
	// 	log.Error(err, "❌ Could not load ServerConfig")
	// 	return ctrl.Result{}, err
	// }
	var serverConfig openprojectv1alpha1.ServerConfig
	err := r.Get(ctx, types.NamespacedName{
		Name:      wp.Spec.ServerConfigRef.Name,
		Namespace: wp.Namespace, // or wp.Spec.ServerConfigRef.Namespace for cross-namespace
	}, &serverConfig)
	if err != nil {
		log.Error(err, "❌ Could not find referenced ServerConfig", "name", wp.Spec.ServerConfigRef.Name)
		return ctrl.Result{}, err
	}

	if wp.Status.NextRunTime == nil {
		log.Info("🆕 First-time run detected — setting initial status")

		now := metav1.Now()
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		cronSpec, err := parser.Parse(wp.Spec.Schedule)
		if err != nil {
			log.Error(err, "❌ Failed to parse cron schedule")
			return ctrl.Result{}, err
		}

		next := cronSpec.Next(now.Time)
		original := wp.DeepCopy()
		wp.Status.NextRunTime = &metav1.Time{Time: next}
		wp.Status.Status = "Scheduled"
		wp.Status.Message = "Next run scheduled"

		if err := r.Status().Patch(ctx, &wp, client.MergeFrom(original)); err != nil {
			log.Error(err, "❌ Failed to patch initial WorkPackage status")
		} else {
			log.Info("✅ Initial WorkPackage status set", "nextRunTime", next)
		}
	}

	if !shouldRunNow(wp.Spec.Schedule, wp.Status.LastRunTime, wp.CreationTimestamp) {
		log.Info("⏳ Not time to run yet based on schedule", "schedule", wp.Spec.Schedule)
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}

	log.Info("📡 Ready to send ticket", "server", serverConfig.Spec.Server)

	apiKey, err := configloader.LoadAPIKey(ctx, r.Client, &serverConfig)
	if err != nil {
		log.Error(err, "❌ Failed to load OpenProject API key")
		return ctrl.Result{}, err
	}

	log.Info("✅ Ready to use API key and send ticket", "server", serverConfig.Spec.Server)

	payload := map[string]interface{}{
		"subject": wp.Spec.Subject,
		"description": map[string]string{
			"format": "markdown",
			"raw":    wp.Spec.Description,
		},
		"_links": map[string]interface{}{
			"project": map[string]string{
				"href": fmt.Sprintf("/api/v3/projects/%d", wp.Spec.ProjectID),
			},
			"type": map[string]string{
				"href": fmt.Sprintf("/api/v3/types/%d", wp.Spec.TypeID),
			},
		},
	}

	if wp.Spec.EpicID > 0 {
		payload["_links"].(map[string]interface{})["parent"] = map[string]string{
			"href": fmt.Sprintf("/api/v3/work_packages/%d", wp.Spec.EpicID),
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Error(err, "❌ Failed to marshal WorkPackage payload")
		return ctrl.Result{}, err
	}

	reqURL := fmt.Sprintf("%s/api/v3/work_packages", serverConfig.Spec.Server)

	httpReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error(err, "❌ Failed to create HTTP request")
		return ctrl.Result{}, err
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("apikey:%s", apiKey)))
	httpReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", encoded))
	httpReq.Header.Set("Content-Type", "application/json")

	log.Info("🐞 Request JSON payload", "json", string(jsonData))
	log.Info("🐞 POST URL", "url", reqURL)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Error(err, "❌ Failed to send request to OpenProject")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Info("⚠️ Non-2xx status from OpenProject", "status", resp.StatusCode)
		return ctrl.Result{}, nil
	}

	now := metav1.Now()
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSpec, err := parser.Parse(wp.Spec.Schedule)
	if err != nil {
		log.Error(err, "❌ Failed to parse cron schedule")
		return ctrl.Result{}, err
	}
	next := cronSpec.Next(now.Time)
	original := wp.DeepCopy()

	wp.Status.LastRunTime = &now
	wp.Status.NextRunTime = &metav1.Time{Time: next}
	wp.Status.TicketID = extractIDFromResponse(resp)
	wp.Status.Status = "Created"
	wp.Status.Message = "Ticket successfully created"

	if err := r.Status().Patch(ctx, &wp, client.MergeFrom(original)); err != nil {
		log.Error(err, "❌ Failed to patch WorkPackage status")
	} else {
		log.Info("✅ Successfully updated WorkPackage status", "lastRunTime", now, "nextRunTime", next, "ticketID", wp.Status.TicketID)
	}

	return ctrl.Result{}, nil
}

func (r *WorkPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openprojectv1alpha1.WorkPackages{}).
		Complete(r)
}

func shouldRunNow(schedule string, lastRun *metav1.Time, creationTime metav1.Time) bool {
	if schedule == "" {
		return false
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(schedule)
	if err != nil {
		fmt.Println("❌ Error parsing cron schedule:", err)
		return false
	}

	now := time.Now()
	var last time.Time
	if lastRun == nil || lastRun.IsZero() {
		last = creationTime.Time
	} else {
		last = lastRun.Time
	}

	next := spec.Next(last)

	fmt.Printf("🕒 now: %s\n", now.Format(time.RFC3339))
	fmt.Printf("🕒 lastRunTime: %s\n", last.Format(time.RFC3339))
	fmt.Printf("🕒 next scheduled time: %s\n", next.Format(time.RFC3339))

	return now.After(next)
}

func extractIDFromResponse(resp *http.Response) string {
	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	if id, ok := result["id"]; ok {
		return fmt.Sprintf("%v", id)
	}

	return ""
}
