"""Fixtures for the AI-draft (CF-2) integration tests.

These tests exercise the LLM-backed AI strategy generator end to end. The
real Anthropic/OpenAI endpoint is replaced by a dedicated WireMock container
(`llm_mock`); SigNoz is pointed at it via DS_APM_LLM_ENDPOINT so no external
network call is made.
"""

import docker
import docker.errors
import pytest
from testcontainers.core.container import Network
from wiremock.testing.testcontainer import WireMockContainer

from fixtures import reuse, types
from fixtures.logger import setup_logger
from fixtures.signoz import create_signoz

logger = setup_logger(__name__)


@pytest.fixture(name="llm_mock", scope="package")
def llm_mock(
    network: Network,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
) -> types.TestContainerDocker:
    """A dedicated WireMock container standing in for the LLM HTTP API.

    Kept separate from `zeus`/`gateway` so mapping resets never disturb the
    license/gateway stubs other fixtures rely on.
    """

    def create() -> types.TestContainerDocker:
        container = WireMockContainer(image="wiremock/wiremock:2.35.1-1", secure=False)
        container.with_exposed_ports(8080)
        container.with_network(network)
        container.start()

        return types.TestContainerDocker(
            id=container.get_wrapped_container().id,
            host_configs={
                "8080": types.TestContainerUrlConfig(
                    "http",
                    container.get_container_host_ip(),
                    container.get_exposed_port(8080),
                )
            },
            container_configs={
                "8080": types.TestContainerUrlConfig(
                    "http",
                    container.get_wrapped_container().name,
                    8080,
                )
            },
        )

    def delete(container: types.TestContainerDocker) -> None:
        client = docker.from_env()
        try:
            client.containers.get(container_id=container.id).stop()
            client.containers.get(container_id=container.id).remove(v=True)
        except docker.errors.NotFound:
            logger.info("Skipping removal of llm_mock(%s), not found.", container.id)

    def restore(cache: dict) -> types.TestContainerDocker:
        return types.TestContainerDocker.from_cache(cache)

    return reuse.wrap(
        request,
        pytestconfig,
        "llm_mock",
        lambda: types.TestContainerDocker(id="", host_configs={}, container_configs={}),
        create,
        delete,
        restore,
    )


@pytest.fixture(name="signoz", scope="package")
def signoz_aidraft(  # pylint: disable=too-many-arguments,too-many-positional-arguments
    network: Network,
    zeus: types.TestContainerDocker,
    gateway: types.TestContainerDocker,
    sqlstore: types.TestContainerSQL,
    clickhouse: types.TestContainerClickhouse,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
    llm_mock: types.TestContainerDocker,  # pylint: disable=redefined-outer-name
) -> types.SigNoz:
    """SigNoz configured with the LLM generator pointed at the WireMock mock."""
    return create_signoz(
        network=network,
        zeus=zeus,
        gateway=gateway,
        sqlstore=sqlstore,
        clickhouse=clickhouse,
        request=request,
        pytestconfig=pytestconfig,
        cache_key="signoz-aidraft",
        env_overrides={
            "DS_APM_AI_GENERATOR": "llm",
            "DS_APM_LLM_PROVIDER": "claude",
            "DS_APM_LLM_TRANSPORT": "api",
            # claudeapi requires a non-empty key; the value is never validated
            # by WireMock.
            "ANTHROPIC_API_KEY": "test-key-dummy",
            # in-network address of the WireMock container (by container name).
            "DS_APM_LLM_ENDPOINT": llm_mock.container_configs["8080"].get("/v1/messages"),
            "DS_APM_LLM_TIMEOUT_SECONDS": "10",
        },
    )
