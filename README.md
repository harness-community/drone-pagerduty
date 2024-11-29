# drone-pagerduty

## Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t plugins/pagerduty -f docker/Dockerfile .
```

## Testing

Execute the plugin from your current working directory:

## Pager Duty change event Env variables
```
docker run --rm \
  -e PLUGIN_ROUTING_KEY="your_routing_key_here" \
  -e PLUGIN_INCIDENT_SUMMARY="Incident summary here" \
  -e PLUGIN_INCIDENT_SOURCE="Incident source here" \
  -e PLUGIN_CREATE_CHANGE_EVENT=true \
  -e PLUGIN_CUSTOM_DETAILS="{\"build_state\": \"passed\", \"build_number\": \"2\", \"run_time\": \"1236s\"}" \
  -v $(pwd):$(pwd) \
  plugins/pagerduty
```

## Pager Duty Env variables
```
docker run --rm \
  -e PLUGIN_ROUTING_KEY="your_routing_key_here" \
  -e PLUGIN_INCIDENT_SUMMARY="Incident summary here" \
  -e PLUGIN_INCIDENT_SOURCE="Incident source here" \
  -e PLUGIN_INCIDENT_SEVERITY="info" \
  -e PLUGIN_DEDUP_KEY="your_dedup_key_here" \
  -e PLUGIN_RESOLVE_INCIDENT=true \
  -e PLUGIN_JOB_STATUS="success" \
  -v $(pwd):$(pwd) \
  plugins/pagerduty
```

## Plugin Settings
- `LOG_LEVEL` debug/info Level defines the plugin log level. Set this to debug to see the response from PagerDuty
- PLUGIN_ROUTING_KEY: The integration key for PagerDuty to route the event.
- PLUGIN_INCIDENT_SUMMARY: A summary of the incident being triggered.
- PLUGIN_INCIDENT_SOURCE: The source of the incident.
- PLUGIN_CREATE_CHANGE_EVENT: set to true to create change event
- PLUGIN_CUSTOM_DETAILS: Provide custom details for change event
- PLUGIN_INCIDENT_SEVERITY: Severity level specifies the severity level of the incident, e.g., 'critical', 'error', 'warning', 'info', 'unknown'.
- PLUGIN_DEDUP_KEY: Deduplication key for identifying and resolving incidents (optional).
- PLUGIN_RESOLVE_INCIDENT: Set to true to resolve an incident or false to trigger.
- PLUGIN_CREATE_CHANGE_EVENT: Set to true to create a change event.
- PLUGIN_JOB_STATUS: The job status is the condition of the job (success | failure | unstable | aborted)
	
