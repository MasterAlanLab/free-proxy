import subprocess
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]


def test_production_deployment_files_use_free_proxy_runtime() -> None:
    install_script = PROJECT_ROOT / "install.sh"
    systemd = PROJECT_ROOT / "deploy/free-proxy.service"
    openrc = PROJECT_ROOT / "deploy/free-proxy.openrc"
    privileged = PROJECT_ROOT / "tests/privileged/verify_linux.sh"

    subprocess.run(["sh", "-n", str(install_script)], check=True)
    subprocess.run(["sh", "-n", str(openrc)], check=True)
    subprocess.run(["sh", "-n", str(privileged)], check=True)

    script_text = install_script.read_text(encoding="utf-8")
    systemd_text = systemd.read_text(encoding="utf-8")
    openrc_text = openrc.read_text(encoding="utf-8")
    assert "uv sync --frozen --no-dev" in script_text
    assert "database-upgrade" in script_text
    assert "free-proxy\" preflight" in script_text
    assert "free-proxy-manage" in script_text
    assert "/opt/free-proxy/.venv/bin/free-proxy serve" in systemd_text
    assert "/opt/free-proxy/.venv/bin/free-proxy preflight" in systemd_text
    assert 'command="/opt/free-proxy/.venv/bin/free-proxy"' in openrc_text
    assert "ExecStart=/usr/bin/python3" not in script_text
    privileged_text = privileged.read_text(encoding="utf-8")
    assert "socks5://127.0.0.1" in privileged_text
    assert "ip route show default" in privileged_text
    assert "OpenVPN process remained after shutdown" in privileged_text
