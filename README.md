# wsone2sentinel

`wsone2sentinel` pulls device records from Workspace ONE and uploads them to Microsoft Sentinel through the Azure Monitor Logs Ingestion API.

## What It Does

- Authenticates to Workspace ONE with OAuth2 client credentials.
- Fetches device records from the WS1 `/mdm/devices/search` API.
- Normalizes selected fields before ingestion.
- Uploads the resulting records to Microsoft Sentinel.

## Requirements

- Go
- A Workspace ONE API client with access to the device search endpoint
- A Microsoft Entra application with permission to use your Sentinel ingestion pipeline
- An existing Azure Monitor Data Collection Rule and stream for the target Sentinel table

## Configuration

Configuration can be provided through a YAML file, environment variables, or both. Environment variables use the `CSS_` prefix.

### YAML Example

```yaml
log:
  level: DEBUG

microsoft:
  app_id: ""
  secret_key: ""
  tenant_id: ""
  subscription_id: ""
  resource_group: ""
  workspace_name: ""
  dcr:
    endpoint: ""
    rule_id: ""
    stream_name: ""

ws1:
  api_url: ""
  auth_location: ""
  client_id: ""
  client_secret: ""
```

## Run

Run with a config file:

```bash
go run ./cmd/... -config=config.yml
```

Run with the provided development config:

```bash
make run
```

## Build

Build the binary directly with Go:

```bash
go build -o wsone2sentinel ./cmd/...
```

Or use the existing release target:

```bash
make build
```
