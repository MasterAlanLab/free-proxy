import base64

from free_proxy.domain.enums import TransportProtocol
from free_proxy.providers.vpngate.parser import parse_remote, parse_vpngate_response


def test_parse_remote_reads_openvpn_directives() -> None:
    config = """
    client
    proto tcp-client
    remote 203.0.113.10 443 tcp
    """

    host, port, transport = parse_remote(config, "198.51.100.4")

    assert host == "203.0.113.10"
    assert port == 443
    assert transport is TransportProtocol.TCP


def test_parse_vpngate_response_decodes_and_deduplicates_nodes() -> None:
    config = "client\nproto udp\nremote 198.51.100.8 1194 udp\n"
    encoded = base64.b64encode(config.encode()).decode()
    header = (
        "#HostName,IP,Score,Ping,Speed,CountryLong,CountryShort,NumVpnSessions,"
        "OpenVPN_ConfigData_Base64"
    )
    row = f"vpn1,198.51.100.8,1000,12,9000000,Japan,JP,3,{encoded}"

    nodes = parse_vpngate_response(f"{header}\n{row}\n{row}\n")

    assert len(nodes) == 1
    node = nodes[0]
    assert node.provider == "vpngate"
    assert node.country == "Japan"
    assert node.country_code == "JP"
    assert node.remote_port == 1194
    assert node.transport is TransportProtocol.UDP
    assert node.config_text == config
