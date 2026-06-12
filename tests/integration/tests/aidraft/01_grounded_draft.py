"""T1 acceptance: SOP-grounded AI draft generation via a mocked LLM.

The LLM HTTP API is mocked with WireMock (see conftest `llm_mock`); SigNoz's
LLM generator is pointed at it through DS_APM_LLM_ENDPOINT. These tests drive
the real Render -> claudeapi -> Parse -> grounding path end to end through the
`POST /api/v2/ds/ai/strategy/preview` endpoint.
"""

import json
from collections.abc import Callable
from http import HTTPStatus

import requests
from wiremock.resources.mappings import (
    HttpMethods,
    Mapping,
    MappingRequest,
    MappingResponse,
)

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD

PREVIEW_PATH = "/api/v2/ds/ai/strategy/preview"

# A bound SOP + alert payload reused by both cases. The SOP document is passed
# inline (the preview endpoint binds it from the request body), so no SOP store
# seeding is required.
REQUEST_BODY = {
    "incidentId": "INC-AIDRAFT-001",
    "alertFingerprint": "fp-aidraft-pay",
    "sopDocument": {
        "sopId": "SOP-PAY-001",
        "version": "2026-05-20.1",
        "title": "Payment API 5xx 대응 절차",
        "bodyMarkdown": "## 1단계\n- PG timeout 로그 확인\n- 결제 성공률 dashboard 확인",
    },
    "evidenceRefs": [
        {
            "refId": "metric:err:payment",
            "type": "metric",
            "observation": "5xx rate 15% for 5 minutes",
            "confidence": "high",
        }
    ],
}


def _mock_llm_response(
    make_http_mocks: Callable[[types.TestContainerDocker, list], None],
    llm_mock: types.TestContainerDocker,
    draft: dict,
) -> None:
    """Stub POST /v1/messages so the mocked LLM returns `draft` as its text body.

    The Anthropic Messages API wraps the model output in content[].text; the
    server concatenates those text blocks and parses the JSON draft out of them.
    """
    make_http_mocks(
        llm_mock,
        [
            Mapping(
                request=MappingRequest(method=HttpMethods.POST, url="/v1/messages"),
                response=MappingResponse(
                    status=200,
                    json_body={
                        "content": [
                            {"type": "text", "text": json.dumps(draft, ensure_ascii=False)}
                        ]
                    },
                ),
                persistent=False,
            )
        ],
    )


def _preview(signoz: types.SigNoz, token: str) -> requests.Response:
    return requests.post(
        signoz.self.host_configs["8080"].get(PREVIEW_PATH),
        json=REQUEST_BODY,
        headers={"Authorization": f"Bearer {token}"},
        timeout=15,
    )


def test_grounded_draft_includes_sop_citation_and_drafts(
    signoz: types.SigNoz,
    llm_mock: types.TestContainerDocker,
    create_user_admin: types.Operation,  # noqa: ARG001 — ensures the admin exists
    get_token: Callable[[str, str], str],
    make_http_mocks: Callable[[types.TestContainerDocker, list], None],
):
    # The mocked LLM returns a draft that cites the SOP step and fills the
    # customer/vendor communication drafts.
    draft = {
        "headline": "결제 API 5xx 급증 — PG timeout 우선 확인",
        "hypotheses": [
            {
                "rank": 1,
                "text": "외부 PG 응답 지연으로 5xx 증가",
                "confidence": "medium",
                "evidenceRefs": ["metric:err:payment"],
                "sopStepRefs": ["SOP-PAY-001#1"],
            }
        ],
        "firstActions": [
            {
                "text": "PG timeout 로그와 결제 성공률 dashboard 확인",
                "sopStepRef": "SOP-PAY-001#1",
                "requiresHumanApproval": True,
            }
        ],
        "customerUpdateDraft": "현재 결제 지연을 확인하여 SOP 기준 초동 분석 중입니다. 15분 내 업데이트 드리겠습니다.",
        "vendorRequestDraft": "PG사 측 응답 지연/장애 공지 여부를 확인 부탁드립니다.",
        "confidence": "medium",
        "status": "ready",
    }
    _mock_llm_response(make_http_mocks, llm_mock, draft)

    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    response = _preview(signoz, token)

    assert response.status_code == HTTPStatus.OK, response.text
    strategy = response.json()["data"]

    # Grounded draft: SOP citation survived (FR-CF2 SOP grounding).
    assert strategy["status"] == "ready"
    assert strategy["sopId"] == "SOP-PAY-001"
    assert strategy["firstActions"][0]["sopStepRef"] == "SOP-PAY-001#1"
    # Output-structure enrichment: customer/vendor drafts surfaced.
    assert strategy["customerUpdateDraft"]
    assert strategy["vendorRequestDraft"]
    assert strategy["confidence"] == "medium"
    # Audit records the v2 prompt template.
    assert strategy["audit"]["promptVersion"] == "ds-ir-ko-llm-v2"


def test_ungrounded_ready_draft_is_downgraded(
    signoz: types.SigNoz,
    llm_mock: types.TestContainerDocker,
    create_user_admin: types.Operation,  # noqa: ARG001 — ensures the admin exists
    get_token: Callable[[str, str], str],
    make_http_mocks: Callable[[types.TestContainerDocker, list], None],
):
    # The mocked LLM claims "ready" but cites no SOP step — only evidence refs.
    # With an SOP injected, the server must downgrade it (hallucination guard).
    draft = {
        "headline": "결제 API 5xx 급증",
        "hypotheses": [
            {
                "rank": 1,
                "text": "외부 PG 응답 지연으로 5xx 증가",
                "confidence": "medium",
                "evidenceRefs": ["metric:err:payment"],
            }
        ],
        "firstActions": [
            {
                "text": "결제 성공률 dashboard 확인",
                "evidenceRefs": ["metric:err:payment"],
                "requiresHumanApproval": True,
            }
        ],
        "confidence": "medium",
        "status": "ready",
    }
    _mock_llm_response(make_http_mocks, llm_mock, draft)

    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    response = _preview(signoz, token)

    assert response.status_code == HTTPStatus.OK, response.text
    strategy = response.json()["data"]

    assert strategy["status"] == "low_confidence", strategy
    assert strategy["confidence"] == "low"
    assert strategy["limitations"], "downgrade must explain the missing SOP grounding"
