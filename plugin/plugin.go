package plugin

import (
	"context"
	"errors"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/sirupsen/logrus"
)

// Args provides plugin execution arguments.
type Args struct {
	Level             string `envconfig:"PLUGIN_LOG_LEVEL"`
	RoutingKey        string `envconfig:"PLUGIN_ROUTING_KEY"`
	IncidentSummary   string `envconfig:"PLUGIN_INCIDENT_SUMMARY"`
	IncidentSource    string `envconfig:"PLUGIN_INCIDENT_SOURCE"`
	IncidentSeverity  string `envconfig:"PLUGIN_INCIDENT_SEVERITY"`
	DedupKey          string `envconfig:"PLUGIN_DEDUP_KEY"`
	CreateChangeEvent bool   `envconfig:"PLUGIN_CREATE_CHANGE_EVENT"`
	Resolve           bool   `envconfig:"PLUGIN_RESOLVE"`
	JobStatus         string `envconfig:"PLUGIN_JOB_STATUS"`
}

// PagerDutyClient defines the methods used from the PagerDuty API.
type PagerDutyClient interface {
	ManageEventWithContext(ctx context.Context, event *pagerduty.V2Event) (*pagerduty.V2EventResponse, error)
	CreateChangeEventWithContext(ctx context.Context, event pagerduty.ChangeEvent) (*pagerduty.ChangeEventResponse, error)
}

// Exec executes the plugin.
func Exec(ctx context.Context, args Args) error {
	client := pagerduty.NewClient(args.RoutingKey)
	logger := logrus.
		WithField("PLUGIN_ROUTING_KEY", args.RoutingKey).
		WithField("PLUGIN_INCIDENT_SUMMARY", args.IncidentSummary).
		WithField("PLUGIN_INCIDENT_SOURCE", args.IncidentSource).
		WithField("PLUGIN_INCIDENT_SEVERITY", args.IncidentSeverity).
		WithField("PLUGIN_CREATE_CHANGE_EVENT", args.CreateChangeEvent).
		WithField("PLUGIN_JOB_STATUS", args.JobStatus)

	logger.Info("Starting plugin execution")

	// Validate required fields
	if args.RoutingKey == "" {
		return errors.New("routingKey is required")
	}
	if args.IncidentSummary == "" {
		return errors.New("incidentSummary is required")
	}
	if args.DedupKey == "" && args.Resolve {
		return errors.New("dedupKey is required for resolving an incident")
	}
	if args.JobStatus == "" {
		logger.Warn("Job status is empty")
	}

	if args.CreateChangeEvent {
		logger.Info("Creating change event")
		if err := createChangeEvent(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to create change event")
			return errors.New("failed to create change event")
		}
		// If job status is empty, skip further incident processing
		if args.JobStatus == "" {
			logger.Warn("Skipping incident logic as job status is empty")
			return nil
		}
	}

	// Handle job status and decide whether to trigger or resolve incidents
	var resolveIncident bool
	var severity = args.IncidentSeverity
	var summary = args.IncidentSummary

	switch args.JobStatus {
	case "success":
		resolveIncident = args.Resolve
		summary = "Job succeeded: " + summary
		logger.Info("Job succeeded, deciding on resolving incident")
	case "failure":
		resolveIncident = false
		severity = "critical"
		summary = "Job failed: " + summary
		logger.Info("Job failed, deciding on triggering or resolving incident")
	case "unstable":
		resolveIncident = false
		severity = "warning"
		summary = "Job is unstable: " + summary
		logger.Info("Job is unstable, deciding on triggering or resolving incident")
	case "aborted":
		resolveIncident = false
		severity = "warning"
		summary = "Job was aborted: " + summary
		logger.Info("Job was aborted, deciding on triggering or resolving incident")
	default:
		logger.Warn("Unknown job status, no action taken")
		return nil
	}

	args.IncidentSeverity = severity
	args.IncidentSummary = summary

	if resolveIncident {
		if err := resolveIncidentAction(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to resolve incident")
			return errors.New("failed to resolve incident")
		}
	} else {
		if err := triggerIncident(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to trigger incident")
			return errors.New("failed to trigger incident")
		}
	}

	logger.Info("Plugin execution completed successfully")
	return nil
}

// triggerIncident triggers an incident in PagerDuty.
func triggerIncident(ctx context.Context, client PagerDutyClient, args Args) error {
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

	_, err := client.ManageEventWithContext(ctx, event)
	if err != nil {
		return errors.New("failed to trigger incident: " + err.Error())
	}
	return nil
}

// resolveIncidentAction resolves an incident in PagerDuty.
func resolveIncidentAction(ctx context.Context, client PagerDutyClient, args Args) error {
	event := &pagerduty.V2Event{
		RoutingKey: args.RoutingKey,
		Action:     "resolve",
		DedupKey:   args.DedupKey,
	}

	_, err := client.ManageEventWithContext(ctx, event)
	if err != nil {
		return errors.New("failed to resolve incident: " + err.Error())
	}
	return nil
}

// createChangeEvent creates a change event in PagerDuty.
func createChangeEvent(ctx context.Context, client PagerDutyClient, args Args) error {
	event := pagerduty.ChangeEvent{
		RoutingKey: args.RoutingKey,
		Payload: pagerduty.ChangeEventPayload{
			Summary:       args.IncidentSummary,
			Source:        args.IncidentSource,
			CustomDetails: map[string]interface{}{"job_status": args.JobStatus},
		},
	}

	_, err := client.CreateChangeEventWithContext(ctx, event)
	if err != nil {
		return errors.New("failed to create change event: " + err.Error())
	}
	return nil
}
