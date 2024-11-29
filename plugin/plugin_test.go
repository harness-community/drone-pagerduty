package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPagerDutyClient is a mock client for PagerDuty API calls.
type MockPagerDutyClient struct {
	mock.Mock
}

func (m *MockPagerDutyClient) ManageEventWithContext(ctx context.Context, event *pagerduty.V2Event) (*pagerduty.V2EventResponse, error) {
	args := m.Called(ctx, event)
	response, ok := args.Get(0).(*pagerduty.V2EventResponse)
	if !ok && args.Get(0) != nil {
		return nil, errors.New("failed to convert interface to *pagerduty.V2EventResponse")
	}
	return response, args.Error(1)
}

func (m *MockPagerDutyClient) CreateChangeEventWithContext(ctx context.Context, event pagerduty.ChangeEvent) (*pagerduty.ChangeEventResponse, error) {
	args := m.Called(ctx, event)
	response, ok := args.Get(0).(*pagerduty.ChangeEventResponse)
	if !ok && args.Get(0) != nil {
		return nil, errors.New("failed to convert interface to *pagerduty.ChangeEventResponse")
	}
	return response, args.Error(1)
}

// TestExec tests the Exec function.
func TestExec(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "critical",
		DedupKey:         "testDedupKey",
		JobStatus:        "failed",
	}

	// Set up expected event for failure scenario.
	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Summary:  "Job failed: " + args.IncidentSummary,
			Source:   args.IncidentSource,
			Severity: args.IncidentSeverity,
		},
		DedupKey: args.DedupKey,
	}

	mockClient.On("ManageEventWithContext", ctx, event).Return(&pagerduty.V2EventResponse{}, nil)

	err := Exec(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestExecCreateChangeEvent tests the Exec function with CreateChangeEvent.
func TestExecCreateChangeEvent(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:        "testRoutingKey",
		IncidentSummary:   "Test change event summary",
		IncidentSource:    "Test source",
		CreateChangeEvent: true,
		CustomDetailsStr:  "{\"key1\": \"value1\"}", // Valid JSON
	}

	// Define mock expectations
	mockClient.On("CreateChangeEventWithContext", mock.Anything, mock.MatchedBy(func(event pagerduty.ChangeEvent) bool {
		// Ensure fields match
		return event.RoutingKey == args.RoutingKey &&
			event.Payload.Summary == args.IncidentSummary &&
			event.Payload.Source == args.IncidentSource &&
			len(event.Payload.CustomDetails) == 1 // Check key-value pairs
	})).Return(&pagerduty.ChangeEventResponse{}, nil)

	// Execute the function
	err := Exec(ctx, mockClient, args)

	// Assert no error
	require.NoError(t, err, "Expected no error but got: %v", err)

	// Assert mock expectations
	mockClient.AssertExpectations(t)
}

// TestExecResolveIncidentAction tests the Exec function with Resolve set to true.
func TestExecResolveIncidentAction(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test resolve summary",
		IncidentSource:   "Test source",
		DedupKey:         "testDedupKey",
		Resolve:          true,
		JobStatus:        "success",
		IncidentSeverity: "info",
	}

	// Set up expected resolve event.
	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "resolve",
		DedupKey:   args.DedupKey,
	}

	mockClient.On("ManageEventWithContext", ctx, event).Return(&pagerduty.V2EventResponse{}, nil)

	err := Exec(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestExecMissingRoutingKey tests the Exec function with a missing RoutingKey.
func TestExecMissingRoutingKey(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "critical",
		DedupKey:         "testDedupKey",
		JobStatus:        "failure",
	}

	err := Exec(ctx, mockClient, args)
	require.EqualError(t, err, "missing required parameter: routingKey")
}

// TestExecAPICallFailure tests the Exec function with an API call failure.
func TestExecAPICallFailure(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "critical",
		DedupKey:         "testDedupKey",
		JobStatus:        "failed",
	}

	// Set up expected event for failure scenario.
	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Summary:  "Job failed: " + args.IncidentSummary,
			Source:   args.IncidentSource,
			Severity: args.IncidentSeverity,
		},
		DedupKey: args.DedupKey,
	}

	mockClient.On("ManageEventWithContext", ctx, event).Return(nil, errors.New("API call failed"))

	err := Exec(ctx, mockClient, args)
	require.EqualError(t, err, "failed to trigger incident: failed to trigger incident: API call failed")
	mockClient.AssertExpectations(t)
}

// TestExecInvalidCustomDetails tests the Exec function with invalid CustomDetailsStr.
func TestExecInvalidCustomDetails(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:        "testRoutingKey",
		IncidentSummary:   "Test change event summary",
		IncidentSource:    "Test source",
		CreateChangeEvent: true,
		CustomDetailsStr:  "invalid-json", // Invalid JSON
	}

	// Execute the function
	err := Exec(ctx, mockClient, args)

	// Define the expected error
	expectedErr := "failed to create change event: failed to parse custom details JSON: invalid character 'i' looking for beginning of value"

	// Assert the error matches the expected value
	require.EqualError(t, err, expectedErr, "Expected: %q, but got: %v", expectedErr, err)
}

// TestExecInvalidSeverity tests the Exec function with an invalid severity value.
func TestExecInvalidSeverity(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "invalid-severity",
		DedupKey:         "testDedupKey",
		JobStatus:        "failed",
	}

	err := Exec(ctx, mockClient, args)
	require.EqualError(t, err, "invalid severity value; allowed values are 'critical', 'error', 'warning', 'info'")
}

// TestExecUnknownJobStatus tests the Exec function with an unknown JobStatus.
func TestExecUnknownJobStatus(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "info",
		DedupKey:         "testDedupKey",
		JobStatus:        "unknown-status",
	}

	// No API call should be made since the job status is unknown.
	err := Exec(ctx, mockClient, args)
	// Plugin should gracefully handle the unknown status without errors.
	require.NoError(t, err)
	mockClient.AssertNotCalled(t, "ManageEventWithContext")
	mockClient.AssertNotCalled(t, "CreateChangeEventWithContext")
}
