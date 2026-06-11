"""T1 acceptance for SCOPE row 3 (WT-pii).

Given an alert whose labels and annotations carry PII (email, phone, opaque
token), when it fires to a webhook notification channel, then the external
payload captured at the webhook must contain ZERO raw PII — across both the
derived ``incident`` block and the embedded Alertmanager template data
(``commonLabels`` / ``commonAnnotations`` / ``alerts[].labels|annotations``).

The ``owner_email`` label below is deliberately NOT one of the recognized
incident keys, so it is only scrubbed by the full-payload sanitization
(``SanitizeTemplateData``) — it is the regression guard for the embedded
``*template.Data`` leak.

Metric exposure note: SCOPE row 3 also calls for the redaction counter
(``signoz_alertmanager_incident_redactions_total``) to be exposed. That counter
is registered on the scrape registry in ``alertmanagerserver`` and its counting
is covered by the hermetic T0 unit test ``TestRedactionMetric``. It is NOT
asserted here because the Prometheus endpoint binds an internal port (9090) that
the shared ``signoz`` test fixture does not publish, and that fixture is frozen
for parallel work (TESTING.md §4). See SCOPE.md memo.
"""

import base64
import json
import time
import uuid
from collections.abc import Callable
from datetime import UTC, datetime, timedelta

from wiremock.client import HttpMethods, Mapping, MappingRequest, MappingResponse

from fixtures import types
from fixtures.alerts import update_rule_channel_name
from fixtures.fs import get_testdata_file_path
from fixtures.logger import setup_logger

logger = setup_logger(__name__)

# A known-good firing scenario; only its labels/annotations are overridden here.
_FIRING_SCENARIO = "alerts/test_scenarios/threshold_above_at_least_once"

# Raw PII that must never appear in the outbound webhook payload.
_LEAKED_EMAIL = "chulsoo@example.co.kr"
_LEAKED_OWNER_EMAIL = "oncall@example.com"  # non-incident label -> data.* leak guard
_LEAKED_PHONE = "010-1234-5678"
_LEAKED_TOKEN = "ghp_aBcDeF0123456789abcdef0123456789abcd"
_ALL_LEAKED = [_LEAKED_EMAIL, _LEAKED_OWNER_EMAIL, _LEAKED_PHONE, _LEAKED_TOKEN]


def _collect_outbound_bodies(
    notification_channel: types.TestContainerDocker, webhook_path: str
) -> list[dict]:
    """Return the JSON bodies of every request captured at the webhook path."""
    import requests

    res = requests.post(
        notification_channel.host_configs["8080"].get("__admin/requests/find"),
        json={"method": "POST", "url": webhook_path},
        timeout=5,
    )
    res.raise_for_status()
    bodies = []
    for req in res.json()["requests"]:
        raw = base64.b64decode(req["bodyAsBase64"]).decode("utf-8")
        bodies.append(json.loads(raw))
    return bodies


def test_masked_outbound_payload(
    notification_channel: types.TestContainerDocker,
    make_http_mocks: Callable[[types.TestContainerDocker, list[Mapping]], None],
    create_webhook_notification_channel: Callable[[str, str, dict, bool], str],
    create_alert_rule: Callable[[dict], str],
    insert_alert_data: Callable[[list[types.AlertData], datetime], None],
) -> None:
    channel_name = f"pii-{uuid.uuid4()}"
    webhook_path = f"/alert/{channel_name}"
    webhook_url = notification_channel.container_configs["8080"].get(webhook_path)

    make_http_mocks(
        notification_channel,
        [
            Mapping(
                request=MappingRequest(method=HttpMethods.POST, url=webhook_path),
                response=MappingResponse(status=200, json_body={}),
                persistent=False,
            )
        ],
    )

    create_webhook_notification_channel(
        channel_name=channel_name,
        webhook_url=webhook_url,
        http_config={},
        send_resolved=False,
    )

    # Insert the known-good firing data for this scenario's metric.
    insert_alert_data(
        [types.AlertData(type="metrics", data_path=f"{_FIRING_SCENARIO}/alert_data.jsonl")],
        base_time=datetime.now(tz=UTC) - timedelta(minutes=5),
    )

    # Load the proven rule, then inject PII into its labels and annotations.
    with open(get_testdata_file_path(f"{_FIRING_SCENARIO}/rule.json"), encoding="utf-8") as f:
        rule_data = json.load(f)
    rule_data["alert"] = f"pii_redaction_{uuid.uuid4().hex[:8]}"
    rule_data["labels"] = {
        "service.name": "checkout-api",
        "severity": "critical",
        "owner_email": _LEAKED_OWNER_EMAIL,
    }
    rule_data["annotations"] = {
        "impact_summary": f"call {_LEAKED_PHONE} or email {_LEAKED_EMAIL}",
        "customer_update": f"leaked token={_LEAKED_TOKEN}",
        "next_action": "investigate p99 latency",
    }
    update_rule_channel_name(rule_data, channel_name)
    create_alert_rule(rule_data)

    # Wait for the alert to reach the webhook.
    deadline = datetime.now() + timedelta(seconds=45)
    bodies: list[dict] = []
    while datetime.now() < deadline:
        bodies = _collect_outbound_bodies(notification_channel, webhook_path)
        if bodies:
            break
        time.sleep(1)

    assert bodies, "expected the PII alert to fire to the webhook, but no request was captured"

    for body in bodies:
        blob = json.dumps(body)
        for leaked in _ALL_LEAKED:
            assert leaked not in blob, (
                f"raw PII {leaked!r} leaked in outbound webhook payload: {blob}"
            )

        # Sanity: the alert really carried the (now masked) incident context, and
        # innocuous metadata is preserved (we mask PII, we don't blank everything).
        assert "incident" in body, f"expected an incident block in payload: {blob}"
        assert "investigate p99 latency" in blob, (
            f"innocuous annotation should be preserved, got: {blob}"
        )
