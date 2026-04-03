# nuon-ext-webhook-server

Simple webhook receiver extension for validating Nuon flow step lifecycle webhook delivery.

## Install

```bash
nuon ext install nuonco/nuon-ext-webhook-server
```

## Run

```bash
nuon webhook-server serve --addr :8787
```

By default, the server exposes:

1. `POST /webhooks/flow/step-lifecycle` for lifecycle webhooks.
1. `GET /healthz` for health checks.

## Configure ctl-api webhook URL

Point ctl-api step lifecycle webhooks at your receiver endpoint, for example:

```text
http://localhost:8787/webhooks/flow/step-lifecycle
```

If your ctl-api process runs in Docker and the extension runs on the host, use host networking aliases such as `host.docker.internal` as needed.

## Optional shared secret

Require `X-Nuon-Webhook-Secret` header either by flag or environment variable:

```bash
nuon webhook-server serve --secret my-shared-secret
```

```bash
NUON_WEBHOOK_SECRET=my-shared-secret nuon webhook-server serve
```

## Example test request

```bash
curl -sS -X POST http://localhost:8787/webhooks/flow/step-lifecycle \
  -H 'content-type: application/json' \
  -d '{
    "version": "v1",
    "transition": "after",
    "sent_at": "2026-03-17T00:00:00Z",
    "event": {
      "workflow_id": "wf_123",
      "workflow_type": "install_deploy",
      "org_id": "org_123",
      "install_id": "ins_123",
      "plan_only": false,
      "step_id": "step_123",
      "step_name": "plan",
      "phase": "plan-policy-check",
      "descriptor": {
        "step_idx": 1,
        "group_idx": 0,
        "group_retry_idx": 0,
        "execution_type": "approval",
        "signal_type": "execute_deploy_component_sync_and_plan",
        "step_target_type": "install_deploys",
        "step_target_id": "dep_123",
        "kind": "component-plan"
      }
    },
    "outcome": {
      "status": "success"
    },
    "source": {
      "service": "ctl-api",
      "kind": "step_lifecycle_hook"
    }
  }'
```

## Logged fields

The receiver logs structured JSON events including:

1. Workflow context (`workflow_id`, `workflow_type`, `org_id`, `install_id`).
1. Step semantics (`step_id`, `step_name`, `phase`, `execution_type`, `signal_type`, `step_target_type`, `step_target_id`, `kind`).
1. Optional descriptor context (`component_id`, `action_trigger_type`).
1. Outcome context when present (`status`, `err_message`, `duration`, metadata count).
1. Request metadata (`remote_addr`, `user_agent`, transition).
