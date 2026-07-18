import socket

from free_proxy.proxy.connector import address_priority


def test_address_priority_prefers_ipv4_before_ipv6() -> None:
    ipv4 = (socket.AF_INET, socket.SOCK_STREAM, 6, "", ("192.0.2.10", 443))
    ipv6 = (socket.AF_INET6, socket.SOCK_STREAM, 6, "", ("2001:db8::10", 443, 0, 0))

    assert sorted([ipv6, ipv4], key=address_priority) == [ipv4, ipv6]
