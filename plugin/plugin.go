package plugin

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/sirupsen/logrus"
)

// Severity levels as constants for reuse and readability.
const (
	SeverityCritical = "critical"
	SeverityError    = "error"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
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
	ResolveIncident   bool   `envconfig:"PLUGIN_RESOLVE_INCIDENT"`
	JobStatus         string `envconfig:"PLUGIN_JOB_STATUS"`
	CustomDetailsStr  string `envconfig:"PLUGIN_CUSTOM_DETAILS"` // Intermediate string to receive JSON
	CustomDetails     map[string]interface{}
}

// PagerDutyClient defines the methods used from the PagerDuty API.
type PagerDutyClient interface {
	ManageEventWithContext(ctx context.Context, event *pagerduty.V2Event) (*pagerduty.V2EventResponse, error)
	CreateChangeEventWithContext(ctx context.Context, event pagerduty.ChangeEvent) (*pagerduty.ChangeEventResponse, error)
}

// validateSeverity validates PagerDuty's allowed severity values.
func validateSeverity(severity string) error {
	switch severity {
	case SeverityCritical, SeverityError, SeverityWarning, SeverityInfo:
		return nil
	default:
		return errors.New("invalid severity value; allowed values are 'critical', 'error', 'warning', 'info'")
	}
}

// Exec executes the plugin.
func Exec(ctx context.Context, client PagerDutyClient, args Args) error {
	logger := logrus.WithFields(logrus.Fields{
		"PLUGIN_ROUTING_KEY":         string("XXXXXXXXXXXXXXXXXXXXXXXX"),
		"PLUGIN_INCIDENT_SUMMARY":    args.IncidentSummary,
		"PLUGIN_INCIDENT_SOURCE":     args.IncidentSource,
		"PLUGIN_INCIDENT_SEVERITY":   args.IncidentSeverity,
		"PLUGIN_CREATE_CHANGE_EVENT": args.CreateChangeEvent,
		"PLUGIN_JOB_STATUS":          args.JobStatus,
	})

	logger.Info("Starting plugin execution")

	if args.RoutingKey == "" {
		return errors.New("missing required parameter: routingKey")
	}

	if !args.CreateChangeEvent {
		if args.DedupKey == "" {
			return errors.New("missing required parameter: dedupKey when not creating a change event")
		}
		if args.JobStatus == "" {
			return errors.New("missing required parameter: jobStatus when not creating a change event")
		}
		if args.IncidentSummary == "" {
			return errors.New("missing required parameter: incidentSummary")
		}
		if args.IncidentSource == "" {
			return errors.New("missing required parameter: incidentSource")
		}
	}

	// Validate severity value if not creating a change event
	if !args.CreateChangeEvent {
		if err := validateSeverity(args.IncidentSeverity); err != nil {
			return err
		}
	}

	if args.JobStatus == "" {
		logger.Warn("Job status is empty, exiting execution")
	}

	if args.CreateChangeEvent {
		logger.Info("Creating change event")
		if err := createChangeEvent(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to create change event")
			return errors.New("failed to create change event: " + err.Error())
		}
		logger.Info("Change event created Successfully")
		return nil
	}

	// Handle job status and decide whether to trigger or resolve incidents
	var resolveIncident bool
	var summary = args.IncidentSummary

	switch args.JobStatus {
	case "SUCCESS":
		resolveIncident = args.ResolveIncident || bool(true)
		summary = "Job succeeded: " + summary
		logger.Info("Job succeeded, deciding on resolving incident")
	case "FAILED":
		resolveIncident = args.ResolveIncident
		summary = "Job failed: " + summary
		logger.Info("Job failed, deciding on triggering or resolving incident")
	case "RUNNING":
		resolveIncident = args.ResolveIncident || bool(true)
		summary = "Job is unstable: " + summary
		logger.Info("Job is running, deciding on triggering or resolving incident")
	case "ABORTED":
		resolveIncident = args.ResolveIncident
		summary = "Job was aborted: " + summary
		logger.Info("Job was aborted, deciding on triggering or resolving incident")
	case "EXPIRED":
		resolveIncident = args.ResolveIncident
		summary = "Job was aborted: " + summary
		logger.Info("Job was expired, deciding on triggering or resolving incident")
	default:
		summary = "Job status unknown: " + summary
		resolveIncident = bool(false) // Unknown status, do not resolve by default
		logger.Warn("Unknown job status, no action taken")
		return nil
	}

	args.IncidentSummary = summary

	if resolveIncident {
		if err := resolveIncidentAction(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to resolve incident: " + err.Error())
			return errors.New("failed to resolve incident: " + err.Error())
		}
	} else {
		if err := triggerIncidentAction(ctx, client, args); err != nil {
			logger.WithError(err).Error("Failed to trigger incident")
			return errors.New("failed to trigger incident: " + err.Error())
		}
	}

	logger.Info("Plugin execution completed successfully")
	return nil
}

// triggerIncident triggers an incident in PagerDuty.
func triggerIncidentAction(ctx context.Context, client PagerDutyClient, args Args) error {
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
	if args.CustomDetailsStr != "" {
		var customDetailsMap map[string]interface{}
		err := json.Unmarshal([]byte(args.CustomDetailsStr), &customDetailsMap)
		if err != nil {
			logrus.WithError(err).Error("Failed to parse custom details JSON" + err.Error())
			return errors.New("failed to parse custom details JSON: " + err.Error())
		}
		args.CustomDetails = customDetailsMap
	}

	event := pagerduty.ChangeEvent{
		RoutingKey: args.RoutingKey,
		Payload: pagerduty.ChangeEventPayload{
			Summary:       args.IncidentSummary,
			Source:        args.IncidentSource,
			CustomDetails: args.CustomDetails,
		},
	}

	_, err := client.CreateChangeEventWithContext(ctx, event)
	if err != nil {
		return errors.New("failed to create change event: " + err.Error())
	}
	return nil
}
