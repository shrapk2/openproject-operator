package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/shrapk2/openproject-operator/api/v1alpha1"
)

// Store the original HTTP client
var originalHTTPClient *http.Client

// Create a mock HTTP client for testing API calls
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if resp, ok := args.Get(0).(*http.Response); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

// Setup function to initialize and restore original client
func setupTestHTTPClient() func() {
	originalHTTPClient = httpClient
	return func() {
		httpClient = originalHTTPClient
	}
}

// Setup common test objects
func setupTestWithWorkPackages() (*runtime.Scheme, v1alpha1.WorkPackages, v1alpha1.ServerConfig, *corev1.Secret) {
	// Create scheme and add types
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// Create a WorkPackage
	wp := v1alpha1.WorkPackages{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wp",
			Namespace: "default",
			CreationTimestamp: metav1.Time{
				Time: time.Now().Add(-time.Hour), // Created an hour ago
			},
		},
		Spec: v1alpha1.WorkPackagesSpec{
			Subject:     "Test Ticket",
			Description: "Test description",
			ProjectID:   123,
			TypeID:      456,
			EpicID:      789,
			Schedule:    "*/5 * * * *", // Every 5 minutes
			ServerConfigRef: corev1.LocalObjectReference{
				Name: "test-server",
			},
		},
	}

	// Create a ServerConfig - use the actual fields from your CRD
	config := v1alpha1.ServerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: v1alpha1.ServerConfigSpec{
			Server: "https://test-openproject.example.com",
		},
	}

	// Create a Secret with API key
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	return scheme, wp, config, &secret
}

// Helper to create a fake successful HTTP response
func createSuccessResponse() *http.Response {
	responseBody := map[string]interface{}{
		"id":      12345,
		"subject": "Test Ticket",
	}
	jsonBody, _ := json.Marshal(responseBody)

	return &http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(bytes.NewReader(jsonBody)),
		Header:     http.Header{},
	}
}

// Helper to create a fake error HTTP response
func createErrorResponse() *http.Response {
	responseBody := map[string]interface{}{
		"error": "Invalid request",
	}
	jsonBody, _ := json.Marshal(responseBody)

	return &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(bytes.NewReader(jsonBody)),
		Header:     http.Header{},
	}
}

// Test parseSchedule function
func TestParseSchedule(t *testing.T) {
	tests := []struct {
		name        string
		schedule    string
		shouldError bool
	}{
		{
			name:        "Valid Schedule",
			schedule:    "*/5 * * * *",
			shouldError: false,
		},
		{
			name:        "Invalid Schedule",
			schedule:    "invalid",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSchedule(tt.schedule)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test calculateNextRunTime function
func TestCalculateNextRunTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		schedule    string
		from        time.Time
		shouldError bool
	}{
		{
			name:        "Valid Schedule",
			schedule:    "*/5 * * * *",
			from:        now,
			shouldError: false,
		},
		{
			name:        "Invalid Schedule",
			schedule:    "invalid",
			from:        now,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := calculateNextRunTime(tt.schedule, tt.from)

			if tt.shouldError {
				assert.Error(t, err)
				assert.True(t, next.IsZero())
			} else {
				assert.NoError(t, err)
				assert.True(t, next.After(tt.from))

				// Check that the difference is approximately 5 minutes
				diff := next.Sub(tt.from)
				expectedMinutes := 5 * time.Minute
				// Allow a small buffer for calculation time
				assert.True(t, diff >= expectedMinutes && diff < expectedMinutes+time.Second*10)
			}
		})
	}
}

// Test buildTicketPayload function
func TestBuildTicketPayload(t *testing.T) {
	_, wp, _, _ := setupTestWithWorkPackages()

	payload := buildTicketPayload(&wp)

	// Check the subject
	assert.Equal(t, wp.Spec.Subject, payload["subject"])

	// Check the description
	description, ok := payload["description"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "markdown", description["format"])
	assert.Equal(t, wp.Spec.Description, description["raw"])

	// Check the links
	links, ok := payload["_links"].(map[string]interface{})
	assert.True(t, ok)

	// Check project link
	project, ok := links["project"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("/api/v3/projects/%d", wp.Spec.ProjectID), project["href"])

	// Check type link
	typeLink, ok := links["type"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("/api/v3/types/%d", wp.Spec.TypeID), typeLink["href"])

	// Check parent link (epicID)
	parent, ok := links["parent"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("/api/v3/work_packages/%d", wp.Spec.EpicID), parent["href"])

	// Test without epicID
	wpNoEpic := wp.DeepCopy()
	wpNoEpic.Spec.EpicID = 0

	payloadNoEpic := buildTicketPayload(wpNoEpic)
	linksNoEpic := payloadNoEpic["_links"].(map[string]interface{})
	_, hasParent := linksNoEpic["parent"]
	assert.False(t, hasParent)
}

// Test extractID function
func TestExtractID(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		expectedID string
	}{
		{
			name:       "Valid Response",
			response:   createSuccessResponse(),
			expectedID: "12345",
		},
		{
			name: "Response Without ID",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status": "ok"}`))),
				Header:     http.Header{},
			},
			expectedID: "",
		},
		{
			name: "Invalid JSON Response",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`not json`))),
				Header:     http.Header{},
			},
			expectedID: "",
		},
		{
			name: "Error Reading Body",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(&errorReader{}),
				Header:     http.Header{},
			},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := extractID(tt.response)
			assert.Equal(t, tt.expectedID, id)
		})
	}
}

// Mock reader that always fails
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

// Test shouldRunNow function
func TestShouldRunNow(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		schedule     string
		lastRun      *metav1.Time
		creationTime metav1.Time
		expected     bool
	}{
		{
			name:     "No LastRun, Should Run Based on Creation Time",
			schedule: "*/5 * * * *",
			lastRun:  nil,
			creationTime: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			expected: true, // 10 minutes since creation > 5 minute schedule
		},
		{
			name:     "LastRun Too Recent",
			schedule: "*/5 * * * *",
			lastRun: &metav1.Time{
				Time: now.Add(-2 * time.Minute),
			},
			creationTime: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			expected: false, // 2 minutes since last run < 5 minute schedule
		},
		{
			name:     "LastRun Old Enough",
			schedule: "*/5 * * * *",
			lastRun: &metav1.Time{
				Time: now.Add(-6 * time.Minute),
			},
			creationTime: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			expected: true, // 6 minutes since last run > 5 minute schedule
		},
		{
			name:     "Zero LastRun, Should Run",
			schedule: "*/5 * * * *",
			lastRun: &metav1.Time{
				Time: time.Time{}, // Zero time
			},
			creationTime: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			expected: true, // Zero lastRun means initialized but waiting for first run
		},
		{
			name:     "Invalid Schedule",
			schedule: "invalid",
			lastRun:  nil,
			creationTime: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			expected: false, // Invalid schedule should not run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRunNow(tt.schedule, tt.lastRun, tt.creationTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test handleInitialization function
func TestHandleInitialization(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	loggerCtx := log.IntoContext(context.Background(), testLogger)

	// Call function
	result, err := r.handleInitialization(loggerCtx, &wp, testLogger)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Check that status was updated
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusScheduled, updatedWP.Status.Status)
	assert.Equal(t, "Next run scheduled", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
	assert.NotNil(t, updatedWP.Status.LastRunTime)
	assert.True(t, updatedWP.Status.LastRunTime.IsZero()) // Should be a zero time
}

// Test updateFailedStatus function
func TestUpdateFailedStatus(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	loggerCtx := log.IntoContext(context.Background(), testLogger)

	// Call function
	r.updateFailedStatus(loggerCtx, &wp, testLogger)

	// Check that status was updated
	var updatedWP v1alpha1.WorkPackages
	err := client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusFailed, updatedWP.Status.Status)
	assert.Equal(t, "Ticket creation failed", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
}

// RoundTripFunc is an adapter to allow using a function as RoundTripper
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Test handleCreateTicket function with success
func TestHandleCreateTicketSuccess(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	loggerCtx := log.IntoContext(context.Background(), testLogger)

	// Setup mock client - the correct way for an http.Client
	cleanup := setupTestHTTPClient()
	defer cleanup()

	// Create a mock client using RoundTripFunc
	mockClient := new(MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(createSuccessResponse(), nil)

	// Replace the global httpClient with our mock
	httpClient = &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return createSuccessResponse(), nil
		}),
	}

	// Call function
	result, err := r.handleCreateTicket(loggerCtx, &wp, &config, "test-api-key", testLogger)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Check that status was updated
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusCreated, updatedWP.Status.Status)
	assert.Equal(t, "Ticket successfully created", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
	assert.NotNil(t, updatedWP.Status.LastRunTime)
	assert.Equal(t, "12345", updatedWP.Status.TicketID)
}

// Test handleCreateTicket function with error
func TestHandleCreateTicketError(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	loggerCtx := log.IntoContext(context.Background(), testLogger)

	// Setup mock client - the correct way for an http.Client
	cleanup := setupTestHTTPClient()
	defer cleanup()

	// Replace the global httpClient with our mock
	httpClient = &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return createErrorResponse(), nil
		}),
	}

	// Call function
	result, err := r.handleCreateTicket(loggerCtx, &wp, &config, "test-api-key", testLogger)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Check that status was updated
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusFailed, updatedWP.Status.Status)
	assert.Equal(t, "Ticket creation failed", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
}

// Test loadConfig function
func TestLoadConfig(t *testing.T) {
	// For this test, we need to mock the configloader.LoadAPIKey function
	// We'll do this by setting up a mock implementation for this test

	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	loggerCtx := log.IntoContext(context.Background(), testLogger)

	// We'll need to mock the loadConfig function or use an actual implementation
	// For simplicity, we'll skip the actual API key loading logic and just verify
	// the ServerConfig is loaded correctly
	resultConfig, _, err := r.loadConfig(loggerCtx, &wp, testLogger)

	// Check result
	require.NoError(t, err)
	assert.NotNil(t, resultConfig)
	assert.Equal(t, config.Spec.Server, resultConfig.Spec.Server)
}

// Test Reconcile function with initialization
func TestReconcileInitialization(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	ctx := log.IntoContext(context.Background(), testLogger)

	// Create request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      wp.Name,
			Namespace: wp.Namespace,
		},
	}

	// Call reconcile
	result, err := r.Reconcile(ctx, req)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Check that status was initialized
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusScheduled, updatedWP.Status.Status)
	assert.Equal(t, "Next run scheduled", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
	assert.NotNil(t, updatedWP.Status.LastRunTime)
	assert.True(t, updatedWP.Status.LastRunTime.IsZero()) // Should be a zero time
}

// Test Reconcile function when it's time to create ticket
func TestReconcileCreateTicket(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Setup wp with lastRun from 10 minutes ago
	now := time.Now()
	wp.Status.LastRunTime = &metav1.Time{Time: now.Add(-10 * time.Minute)}
	wp.Status.NextRunTime = &metav1.Time{Time: now.Add(-5 * time.Minute)} // NextRun time in past
	wp.Status.Status = StatusCreated

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	ctx := log.IntoContext(context.Background(), testLogger)

	// Setup mock client
	cleanup := setupTestHTTPClient()
	defer cleanup()

	// Replace the global httpClient with our mock
	httpClient = &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return createSuccessResponse(), nil
		}),
	}

	// Create request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      wp.Name,
			Namespace: wp.Namespace,
		},
	}

	// Call reconcile
	result, err := r.Reconcile(ctx, req)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Check that ticket was created
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, StatusCreated, updatedWP.Status.Status)
	assert.Equal(t, "Ticket successfully created", updatedWP.Status.Message)
	assert.NotNil(t, updatedWP.Status.NextRunTime)
	assert.NotEqual(t, wp.Status.NextRunTime.Time, updatedWP.Status.NextRunTime.Time) // Should be updated
	assert.NotNil(t, updatedWP.Status.LastRunTime)
	assert.NotEqual(t, wp.Status.LastRunTime.Time, updatedWP.Status.LastRunTime.Time) // Should be updated
	assert.Equal(t, "12345", updatedWP.Status.TicketID)
}

// Test Reconcile function when it's not time to create ticket
func TestReconcileNotTimeToRun(t *testing.T) {
	scheme, wp, config, secret := setupTestWithWorkPackages()

	// Setup wp with lastRun from 1 minute ago
	now := time.Now()
	wp.Status.LastRunTime = &metav1.Time{Time: now.Add(-1 * time.Minute)}
	wp.Status.NextRunTime = &metav1.Time{Time: now.Add(4 * time.Minute)} // NextRun time in future
	wp.Status.Status = StatusCreated

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&wp, &config, secret).
		Build()

	// Create reconciler
	r := &WorkPackageReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create logger
	testLogger := testr.New(t)
	ctx := log.IntoContext(context.Background(), testLogger)

	// Create request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      wp.Name,
			Namespace: wp.Namespace,
		},
	}

	// Call reconcile
	result, err := r.Reconcile(ctx, req)

	// Check result
	require.NoError(t, err)
	assert.Equal(t, DefaultRequeueTime, result.RequeueAfter)

	// Verify that wp was not changed
	var updatedWP v1alpha1.WorkPackages
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      wp.Name,
		Namespace: wp.Namespace,
	}, &updatedWP)
	require.NoError(t, err)

	assert.Equal(t, wp.Status.LastRunTime.Time.Unix(), updatedWP.Status.LastRunTime.Time.Unix())
	assert.Equal(t, wp.Status.NextRunTime.Time.Unix(), updatedWP.Status.NextRunTime.Time.Unix())
	assert.Equal(t, wp.Status.Status, updatedWP.Status.Status)
}

// Test SetupWithManager function
func TestSetupWithManager(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	r := &WorkPackageReconciler{
		Client: nil, // Not used in this test
		Scheme: scheme,
	}

	// Create a proper mock manager instead of using the problematic mockManager
	mgr, err := ctrl.NewManager(
		&rest.Config{Host: "localhost"},
		ctrl.Options{Scheme: scheme},
	)
	require.NoError(t, err)

	err = r.SetupWithManager(mgr)
	require.NoError(t, err)
}
