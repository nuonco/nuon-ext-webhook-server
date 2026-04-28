# nuon-ext-webhook-server

Simple webhook receiver extension for validating Nuon operation lifecycle webhook delivery.

## Install

```bash
nuon ext install nuonco/nuon-ext-webhook-server
```

## Run

```bash
nuon webhook-server serve --addr :8787
```

By default, the server exposes:

1. `POST /webhooks/operation/lifecycle` for lifecycle webhooks.
1. `GET /healthz` for health checks.

## Configure ctl-api webhook URL

Point ctl-api operation lifecycle webhooks at your receiver endpoint, for example:

```text
http://localhost:8787/webhooks/operation/lifecycle
```

If your ctl-api process runs in Docker and the extension runs on the host, use host networking aliases such as `host.docker.internal` as needed.

## Optional shared secret

Require an `X-Nuon-Signature` header (HMAC-SHA256 hex digest of the raw request body, keyed with the shared secret) either by flag or environment variable:

```bash
nuon webhook-server serve --secret my-shared-secret
```

```bash
NUON_WEBHOOK_SECRET=my-shared-secret nuon webhook-server serve
```

ctl-api computes the signature automatically when a webhook is configured with a secret.

## Payload format

The receiver validates a [CloudEvents v1.0](https://github.com/cloudevents/spec) envelope of type `com.nuon.operation.lifecycle.v1` with Nuon-specific extension attributes (`nuonorgid`, `nuonoperation`, `nuonstage`, `nuonstatus`) and a `data` object describing the operation lifecycle event.

### Event names

`data.event` is one of:

- `plan.started`, `plan.finished`
- `apply.started`, `apply.finished`
- `operation.started`, `operation.finished`

Multi-phase operations (`component-deploy`, `component-teardown`, `sandbox-provision`, `sandbox-deprovision`, `sandbox-reprovision`) emit `plan.*` and `apply.*` events. Single-phase operations (`runner-provision`, `runner-reprovision`, `install-created`, `install-updated`, `install-restart`, `action-workflow-run`) emit `operation.*` events.

### Status values

`data.status` is one of: `started`, `succeeded`, `failed`, `canceled`.

When `data.status` is `failed`, `data.failure_reason` may be set (for example, `validation_failed`).

### Example test request

```bash
curl -sS -X POST http://localhost:8787/webhooks/operation/lifecycle \
  -H 'content-type: application/cloudevents+json; charset=utf-8' \
  -d '{
    "specversion": "1.0",
    "id": "5f1c5b0e-2c8e-4f8b-9b4d-2f6c1f3a4d5e",
    "type": "com.nuon.operation.lifecycle.v1",
    "source": "//nuon.co/ctl-api",
    "time": "2026-04-28T00:00:00Z",
    "subject": "org_123/component-deploy/plan/plan.finished",
    "datacontenttype": "application/json",
    "nuonorgid": "org_123",
    "nuonoperation": "component-deploy",
    "nuonstage": "plan",
    "nuonstatus": "succeeded",
    "data": {
      "event": "plan.finished",
      "operation": "component-deploy",
      "stage": "plan",
      "status": "succeeded",
      "duration_ms": 1234,
      "context": {
        "org_id": "org_123",
        "install_id": "ins_123",
        "component_id": "cmp_123"
      }
    }
  }'
```

The receiver validates that:

1. `specversion` is `"1.0"`.
1. `id`, `type`, and `source` are non-empty.
1. `type` equals `"com.nuon.operation.lifecycle.v1"`.
1. `data` is present, `data.event` is a known event name, `data.operation` is non-empty, `data.status` is one of `started|succeeded|failed|canceled`, and `data.context.org_id` is non-empty.

## Logged fields

For each accepted webhook the receiver:

1. Logs a structured `received operation lifecycle webhook` event with `remote_addr`, `user_agent`, `event`, `operation`, `stage`, and `status`.
1. Pretty-prints the full CloudEvents envelope to stdout for inspection.
1. Returns a JSON ack containing `ok`, `received_at`, `event_id`, `event`, `operation`, `stage`, and `status`.

Rejected requests are logged as warnings with the failure reason (`invalid JSON payload`, `missing signature`, `invalid signature`, or a validation error such as `data.event is required`).
