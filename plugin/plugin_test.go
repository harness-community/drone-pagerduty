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
		panic("interface conversion failed")
	}
	return response, args.Error(1)
}

func (m *MockPagerDutyClient) CreateChangeEventWithContext(ctx context.Context, event pagerduty.ChangeEvent) (*pagerduty.ChangeEventResponse, error) {
	args := m.Called(ctx, event)
	response, ok := args.Get(0).(*pagerduty.ChangeEventResponse)
	if !ok && args.Get(0) != nil {
		panic("interface conversion failed")
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
		JobStatus:        "failure",
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
		CustomDetailsStr:  "{\"key1\": \"value1\"}",
	}

	// Set up expected change event.
	customDetailsMap := map[string]interface{}{"key1": "value1"}
	event := pagerduty.ChangeEvent{
		RoutingKey: args.RoutingKey,
		Payload: pagerduty.ChangeEventPayload{
			Summary:       args.IncidentSummary,
			Source:        args.IncidentSource,
			CustomDetails: customDetailsMap,
		},
	}

	mockClient.On("CreateChangeEventWithContext", ctx, event).Return(&pagerduty.ChangeEventResponse{}, nil)

	err := Exec(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestExecResolveIncidentAction tests the Exec function with Resolve set to true.
func TestExecResolveIncidentAction(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:      "testRoutingKey",
		IncidentSummary: "Test resolve summary",
		IncidentSource:  "Test source",
		DedupKey:        "testDedupKey",
		Resolve:         true,
		JobStatus:       "success",
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
	require.EqualError(t, err, "routingKey is required")
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
		JobStatus:        "failure",
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
	require.EqualError(t, err, "failed to trigger incident")
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
		CustomDetailsStr:  "invalid-json",
	}

	err := Exec(ctx, mockClient, args)
	require.EqualError(t, err, "failed to create change event")
}
