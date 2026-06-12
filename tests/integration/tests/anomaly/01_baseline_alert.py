import json
import uuid
from collections.abc import Callable
from datetime import UTC, datetime, timedelta

import pytest
from wiremock.client import HttpMethods, Mapping, MappingRequest, MappingResponse

from fixtures import types
from fixtures.alerts import (
    update_rule_channel_name,
    verify_webhook_alert_expectation,
)
from fixtures.fs import get_testdata_file_path
from fixtures.logger import setup_logger

# Anomaly acceptance (FR-CF7.1): a metric whose latest value deviates beyond
# k·σ from the baseline of the preceding window must raise an early-warning
# alert. The data has a stable baseline (mean 10, σ≈1.41 over 10 points) then a
# newest spike of 20 (z≈7.07, well past the k=3 band), so the rule fires.
ANOMALY_BASELINE_DEVIATION = types.AlertTestCase(
    name="test_anomaly_fires_on_baseline_deviation",
    rule_path="anomaly/baseline_deviation/rule.json",
    alert_data=[
        types.AlertData(
            type="metrics",
            data_path="anomaly/baseline_deviation/alert_data.jsonl",
        ),
    ],
    alert_expectation=types.AlertExpectation(
        should_alert=True,
        wait_time_seconds=30,
        expected_alerts=[
            types.FiringAlert(
                labels={
                    "alertname": "anomaly_baseline_deviation",
                    "threshold.name": "anomaly",
                }
            ),
        ],
    ),
)

logger = setup_logger(__name__)


@pytest.mark.parametrize(
    "alert_test_case",
    [ANOMALY_BASELINE_DEVIATION],
    ids=lambda alert_test_case: alert_test_case.name,
)
def test_anomaly_baseline_alert(
    # Notification channel related fixtures
    notification_channel: types.TestContainerDocker,
    make_http_mocks: Callable[[types.TestContainerDocker, list[Mapping]], None],
    create_webhook_notification_channel: Callable[[str, str, dict, bool], str],
    # Alert rule related fixtures
    create_alert_rule: Callable[[dict], str],
    # Alert data insertion related fixtures
    insert_alert_data: Callable[[list[types.AlertData], datetime], None],
    alert_test_case: types.AlertTestCase,
):
    # Prepare notification channel name and webhook endpoint
    notification_channel_name = str(uuid.uuid4())
    webhook_endpoint_path = f"/alert/{notification_channel_name}"
    notification_url = notification_channel.container_configs["8080"].get(webhook_endpoint_path)

    # register the mock endpoint in notification channel
    make_http_mocks(
        notification_channel,
        [
            Mapping(
                request=MappingRequest(
                    method=HttpMethods.POST,
                    url=webhook_endpoint_path,
                ),
                response=MappingResponse(
                    status=200,
                    json_body={},
                ),
                persistent=False,
            )
        ],
    )

    # Create an alert channel using the given route
    create_webhook_notification_channel(
        channel_name=notification_channel_name,
        webhook_url=notification_url,
        http_config={},
        send_resolved=False,
    )

    # Insert the baseline + spike series. base_time anchors the earliest point,
    # so the newest (spike) datapoint lands ~1 minute before now, comfortably
    # inside the 12-minute evaluation window and as the latest evaluated point.
    insert_alert_data(
        alert_test_case.alert_data,
        base_time=datetime.now(tz=UTC) - timedelta(minutes=11),
    )

    # Create the anomaly rule
    rule_path = get_testdata_file_path(alert_test_case.rule_path)
    with open(rule_path, encoding="utf-8") as f:
        rule_data = json.loads(f.read())
    update_rule_channel_name(rule_data, notification_channel_name)
    rule_id = create_alert_rule(rule_data)
    logger.info(
        "anomaly rule created with id: %s",
        {"rule_id": rule_id, "rule_name": rule_data["alert"]},
    )

    # Verify the anomaly alert fires
    verify_webhook_alert_expectation(
        notification_channel,
        notification_channel_name,
        alert_test_case.alert_expectation,
    )
