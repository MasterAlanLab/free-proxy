import time
from pathlib import Path

from fastapi.testclient import TestClient

from free_proxy.config import Settings
from free_proxy.main import create_app


def test_system_status_and_empty_proxy_pool(tmp_path: Path) -> None:
    settings = Settings(
        data_dir=tmp_path,
        database_url=f"sqlite+aiosqlite:///{tmp_path / 'test.db'}",
        proxy_enabled=False,
        maintenance_enabled=False,
        admin_auth_enabled=False,
        allow_process_restart=False,
    )

    with TestClient(create_app(settings)) as client:
        status_response = client.get("/api/v1/system/status")
        proxy_response = client.get("/api/v1/proxies")
        gateway_response = client.get("/api/v1/gateway/status")
        pool_response = client.get("/api/v1/pool/statistics")
        rotate_response = client.post("/api/v1/gateway/rotate")
        rotate_job = wait_for_job(client, rotate_response.json()["id"])
        excessive_probe = client.post(
            "/api/v1/proxies/probe",
            json={"ids": [f"node-{index}" for index in range(6)]},
        )
        settings_response = client.get("/api/v1/settings")
        update_settings_response = client.put(
            "/api/v1/settings",
            json={
                "routing_mode": "country",
                "force_country": "Japan",
                "routing_ip_type": "residential",
                "connection_enabled": True,
                "fixed_node_id": None,
            },
        )
        home_response = client.get("/")

    assert status_response.status_code == 200
    assert status_response.json()["name"] == "Free Proxy"
    assert status_response.json()["nodes"] == 0
    assert proxy_response.status_code == 200
    assert proxy_response.json() == {"items": [], "total": 0, "limit": 100, "offset": 0}
    assert gateway_response.status_code == 200
    assert gateway_response.json()["running"] is False
    assert status_response.json()["monitors"] == {
        "maintenance": False,
        "active_latency": True,
        "health": True,
    }
    assert pool_response.status_code == 200
    assert pool_response.json() == {
        "total": 0,
        "ready": 0,
        "discovered": 0,
        "unavailable": 0,
        "cooldown": 0,
        "residential": 0,
        "mobile": 0,
        "hosting": 0,
        "unknown": 0,
        "favorites": 0,
        "blacklisted": 0,
        "countries": 0,
    }
    assert rotate_response.status_code == 202
    assert rotate_job["status"] == "succeeded"
    assert rotate_job["result"]["connected"] is False
    assert excessive_probe.status_code == 400
    assert excessive_probe.json()["detail"] == "At most 5 nodes can be tested at once"
    assert settings_response.status_code == 200
    assert settings_response.json()["routing_mode"] == "auto"
    assert update_settings_response.status_code == 200
    assert update_settings_response.json()["force_country"] == "Japan"
    assert home_response.status_code == 200
    assert "Free Proxy" in home_response.text


def wait_for_job(client: TestClient, job_id: str) -> dict[str, object]:
    for _ in range(50):
        response = client.get(f"/api/v1/jobs/{job_id}")
        payload = response.json()
        if payload["status"] in {"succeeded", "failed", "cancelled"}:
            return payload
        time.sleep(0.01)
    raise AssertionError(f"Job did not finish: {job_id}")
