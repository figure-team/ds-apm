"""T1 integration — SOP matching (grounding) + batch upload.

Covers the WT-grounding acceptance criteria end-to-end against the live
HTTP API (testcontainers stack):

  * multi-label priority ranking  -> resolution "label_match" + ranked candidates
  * fallback candidates           -> resolution "fallback" + warning, not bound
  * staleness exclusion           -> 90d+ SOP dropped from matching
  * batch partial failure         -> per-document aggregation incl. version conflict

Each test seeds its own SOP documents under a unique tenant scope so the
org-wide SOP store cannot leak documents between tests; the binding-preview
labels carry the matching project_id/environment so tenant filtering isolates
exactly the documents that test seeded.
"""

import hashlib
from collections.abc import Callable
from datetime import datetime, timedelta, timezone
from http import HTTPStatus

import requests

from fixtures import types
from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.logger import setup_logger

logger = setup_logger(__name__)

SOP_BATCH_PATH = "/api/v2/ds/sop/documents/batch"
SOP_BINDINGS_PREVIEW_PATH = "/api/v2/ds/sop/bindings/preview"


def _rfc3339(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _sop_document(
    sop_id: str,
    version: str,
    *,
    owner_team: str,
    tags: list[str],
    project_ids: list[str],
    environments: list[str],
    updated_at: str,
    body_markdown: str = "# Runbook\n\n1. inspect\n2. mitigate\n",
    approval_status: str = "approved",
) -> dict:
    """Build a valid ds.sop_document.v1 payload (mirrors the demo seed shape)."""
    checksum = "sha256:" + hashlib.sha256(body_markdown.encode("utf-8")).hexdigest()
    return {
        "contractVersion": "ds.sop_document.v1",
        "sopId": sop_id,
        "title": f"{sop_id} runbook",
        "version": version,
        "checksum": checksum,
        "source": {"type": "managed_markdown", "sourceId": "src-managed-markdown-default"},
        "bodyMarkdown": body_markdown,
        "displayUrl": "https://kb.example/sop/" + sop_id,
        "ownerTeam": owner_team,
        "approvalStatus": approval_status,
        "tenantScope": {"projectIds": project_ids, "environments": environments},
        "tags": tags,
        "updatedAt": updated_at,
        "securityContext": {
            "serviceAccountProfile": "ds-sop-reader",
            "secretRefVisible": False,
            "browserCredentialsUsed": False,
            "redactionApplied": True,
        },
    }


def _post_batch(signoz: types.SigNoz, token: str, documents: list[dict]) -> dict:
    response = requests.post(
        signoz.self.host_configs["8080"].get(SOP_BATCH_PATH),
        json={"contractVersion": "ds.sop_document_list.v1", "documents": documents},
        headers={"Authorization": f"Bearer {token}"},
        timeout=10,
    )
    assert response.status_code == HTTPStatus.OK, response.text
    return response.json()["data"]


def _preview_binding(signoz: types.SigNoz, token: str, labels: dict) -> dict:
    response = requests.post(
        signoz.self.host_configs["8080"].get(SOP_BINDINGS_PREVIEW_PATH),
        json={"labels": labels},
        headers={"Authorization": f"Bearer {token}"},
        timeout=10,
    )
    assert response.status_code == HTTPStatus.OK, response.text
    return response.json()["data"]


def test_batch_partial_failure_and_version_conflict(
    signoz: types.SigNoz,
    create_user_admin: types.Operation,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
):
    """Batch upload aggregates per-document results: ok, validation error, and
    an in-batch (sopId, version) duplicate reported as a version conflict."""
    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    project_ids, environments = ["customer-batch"], ["prod"]

    ok_doc = _sop_document(
        "SOP-BAT-001", "2026-06-01.1",
        owner_team="payments", tags=["pay-svc"],
        project_ids=project_ids, environments=environments, updated_at="2026-06-01T00:00:00Z",
    )
    duplicate_doc = _sop_document(
        "SOP-BAT-001", "2026-06-01.1",  # same (sopId, version) -> conflict
        owner_team="payments", tags=["pay-svc"],
        project_ids=project_ids, environments=environments, updated_at="2026-06-01T00:00:00Z",
    )
    invalid_doc = _sop_document(
        "SOP-BAT-002", "2026-06-01.1",
        owner_team="payments", tags=["pay-svc"],
        project_ids=project_ids, environments=environments, updated_at="2026-06-01T00:00:00Z",
        body_markdown="Rotate with access_token=hidden",  # secret-like -> validation error
    )
    other_ok_doc = _sop_document(
        "SOP-BAT-003", "2026-06-01.1",
        owner_team="payments", tags=["pay-svc"],
        project_ids=project_ids, environments=environments, updated_at="2026-06-01T00:00:00Z",
    )

    data = _post_batch(signoz, token, [ok_doc, duplicate_doc, invalid_doc, other_ok_doc])

    assert data["total"] == 4, data
    assert data["succeeded"] == 2, data
    assert data["failed"] == 2, data

    results = data["results"]
    assert len(results) == 4
    assert results[0]["status"] == "ok"
    assert results[1]["status"] == "error"
    assert "duplicate" in results[1]["error"].lower()
    assert results[2]["status"] == "error"
    assert results[2]["error"]
    assert results[3]["status"] == "ok"


def test_multi_label_ranking_binds_top_candidate(
    signoz: types.SigNoz,
    create_user_admin: types.Operation,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
):
    """An alert with no sop_id resolves via service/severity/team labels: the
    full-match SOP binds (resolution label_match) and candidates are ranked by
    matched-dimension count, with team outranking severity on ties."""
    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    project_ids, environments = ["customer-rank"], ["prod"]

    docs = [
        _sop_document("SOP-RANK-001", "2026-05-01.1", owner_team="payments",
                      tags=["pay-svc", "critical"], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 3
        _sop_document("SOP-RANK-002", "2026-05-02.1", owner_team="payments",
                      tags=["pay-svc"], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 2
        _sop_document("SOP-RANK-003", "2026-05-01.1", owner_team="payments",
                      tags=[], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 1 (team, prio 4)
        _sop_document("SOP-RANK-004", "2026-05-01.1", owner_team="infra",
                      tags=["critical"], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 1 (sev, prio 1)
    ]
    seeded = _post_batch(signoz, token, docs)
    assert seeded["succeeded"] == 4, seeded

    data = _preview_binding(signoz, token, {
        "project_id": "customer-rank",
        "environment": "prod",
        "service.name": "pay-svc",
        "severity": "critical",
        "owner_team": "payments",
    })

    assert data["status"] == "bound", data
    assert data["resolution"] == "label_match", data
    assert data["sopId"] == "SOP-RANK-001", data

    candidate_ids = [c["sopId"] for c in data["candidates"]]
    assert candidate_ids == ["SOP-RANK-001", "SOP-RANK-002", "SOP-RANK-003", "SOP-RANK-004"], candidate_ids
    assert data["candidates"][0]["score"] == 3
    assert data["candidates"][0]["matchedOn"] == ["owner_team", "service.name", "severity"]
    assert data["candidates"][1]["score"] == 2


def test_fallback_candidates_when_no_exact_match(
    signoz: types.SigNoz,
    create_user_admin: types.Operation,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
):
    """When no SOP matches every present label dimension, the response is not
    bound: it surfaces approximate candidates plus a no-exact-match warning
    (NF-5.5.1: no silent drop)."""
    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    project_ids, environments = ["customer-fb"], ["prod"]

    docs = [
        _sop_document("SOP-FB-001", "2026-05-02.1", owner_team="payments",
                      tags=["pay-svc"], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 2
        _sop_document("SOP-FB-002", "2026-05-01.1", owner_team="infra",
                      tags=["critical"], project_ids=project_ids,
                      environments=environments, updated_at="2026-06-01T00:00:00Z"),  # 1
    ]
    seeded = _post_batch(signoz, token, docs)
    assert seeded["succeeded"] == 2, seeded

    data = _preview_binding(signoz, token, {
        "project_id": "customer-fb",
        "environment": "prod",
        "service.name": "pay-svc",
        "severity": "critical",
        "owner_team": "payments",
    })

    assert data["status"] == "missing", data
    assert data["resolution"] == "fallback", data
    assert not data.get("sopId"), data  # fallback must not bind
    assert data["warnings"], data
    assert any("no exact" in w.lower() for w in data["warnings"]), data["warnings"]

    candidate_ids = [c["sopId"] for c in data["candidates"]]
    assert candidate_ids == ["SOP-FB-001", "SOP-FB-002"], candidate_ids


def test_staleness_excludes_old_sop_from_matching(
    signoz: types.SigNoz,
    create_user_admin: types.Operation,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
):
    """A SOP not updated within 90 days is excluded from matching even though it
    matches every dimension and carries a higher version; the fresh SOP binds."""
    token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)
    project_ids, environments = ["customer-stale"], ["prod"]

    now = datetime.now(timezone.utc)
    fresh_at = _rfc3339(now - timedelta(days=10))
    stale_at = _rfc3339(now - timedelta(days=200))

    docs = [
        _sop_document("SOP-ST-001", "2026-05-01.1", owner_team="payments",
                      tags=["pay-svc", "critical"], project_ids=project_ids,
                      environments=environments, updated_at=fresh_at),
        # Higher version + full match, but stale -> must be excluded.
        _sop_document("SOP-ST-009", "2026-12-31.9", owner_team="payments",
                      tags=["pay-svc", "critical"], project_ids=project_ids,
                      environments=environments, updated_at=stale_at),
    ]
    seeded = _post_batch(signoz, token, docs)
    assert seeded["succeeded"] == 2, seeded

    data = _preview_binding(signoz, token, {
        "project_id": "customer-stale",
        "environment": "prod",
        "service.name": "pay-svc",
        "severity": "critical",
        "owner_team": "payments",
    })

    assert data["status"] == "bound", data
    assert data["sopId"] == "SOP-ST-001", data
    candidate_ids = [c["sopId"] for c in data["candidates"]]
    assert "SOP-ST-009" not in candidate_ids, candidate_ids
