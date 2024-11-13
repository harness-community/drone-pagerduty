// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
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
	return args.Get(0).(*pagerduty.V2EventResponse), args.Error(1)
}

func (m *MockPagerDutyClient) CreateChangeEventWithContext(ctx context.Context, event pagerduty.ChangeEvent) (*pagerduty.ChangeEventResponse, error) {
	args := m.Called(ctx, event)
	return args.Get(0).(*pagerduty.ChangeEventResponse), args.Error(1)
}

// TestTriggerIncident tests the triggerIncident function.
func TestTriggerIncident(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:       "testRoutingKey",
		IncidentSummary:  "Test incident summary",
		IncidentSource:   "Test source",
		IncidentSeverity: "critical",
		DedupKey:         "testDedupKey",
	}

	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "trigger",
		Payload: &pagerduty.V2Payload{
			Summary:  args.IncidentSummary,
			Source:   args.IncidentSource,
			Severity: args.IncidentSeverity,
		},
		DedupKey: args.DedupKey,
	}

	mockClient.On("ManageEventWithContext", ctx, event).Return(&pagerduty.V2EventResponse{}, nil)

	err := triggerIncident(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestResolveIncidentAction tests the resolveIncidentAction function.
func TestResolveIncidentAction(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey: "testRoutingKey",
		DedupKey:   "testDedupKey",
	}

	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "resolve",
		DedupKey:   args.DedupKey,
	}

	mockClient.On("ManageEventWithContext", ctx, event).Return(&pagerduty.V2EventResponse{}, nil)

	err := resolveIncidentAction(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestCreateChangeEvent tests the createChangeEvent function.
func TestCreateChangeEvent(t *testing.T) {
	mockClient := new(MockPagerDutyClient)
	ctx := context.Background()
	args := Args{
		RoutingKey:      "testRoutingKey",
		IncidentSummary: "Test change event summary",
		IncidentSource:  "Test source",
		JobStatus:       "success",
	}

	event := pagerduty.ChangeEvent{
		RoutingKey: args.RoutingKey,
		Payload: pagerduty.ChangeEventPayload{
			Summary:       args.IncidentSummary,
			Source:        args.IncidentSource,
			CustomDetails: map[string]interface{}{"job_status": args.JobStatus},
		},
	}

	mockClient.On("CreateChangeEventWithContext", ctx, event).Return(&pagerduty.ChangeEventResponse{}, nil)

	err := createChangeEvent(ctx, mockClient, args)
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}
