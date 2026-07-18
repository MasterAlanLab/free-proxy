from __future__ import annotations

import asyncio
import ipaddress
import random
import socket
from typing import Protocol

from free_proxy.domain.exceptions import NetworkOperationError


class Resolver(Protocol):
    async def resolve(self, host: str) -> str: ...


class SystemResolver:
    async def resolve(self, host: str) -> str:
        return host


class TunDnsResolver:
    def __init__(
        self,
        *,
        interface: str,
        dns_server: str = "8.8.8.8",
        timeout: float = 3,
    ) -> None:
        self._interface = interface
        self._dns_server = dns_server
        self._timeout = timeout

    async def resolve(self, host: str) -> str:
        try:
            return str(ipaddress.ip_address(host))
        except ValueError:
            pass

        for query_type in (1, 28):
            result = await self._query(host, query_type)
            if result is not None:
                return result
        raise NetworkOperationError(f"Unable to resolve {host} through {self._interface}")

    async def _query(self, host: str, query_type: int) -> str | None:
        transaction_id = random.getrandbits(16)
        packet = build_dns_query(host, query_type, transaction_id)
        family = socket.AF_INET6 if ":" in self._dns_server else socket.AF_INET
        address: tuple[str, int] = (self._dns_server, 53)
        sock = socket.socket(family, socket.SOCK_DGRAM)
        sock.setblocking(False)
        try:
            bind_to_device(sock, self._interface)
            loop = asyncio.get_running_loop()
            await asyncio.wait_for(loop.sock_sendto(sock, packet, address), self._timeout)
            response, _ = await asyncio.wait_for(loop.sock_recvfrom(sock, 4096), self._timeout)
            return parse_dns_response(response, transaction_id, query_type)
        except (OSError, TimeoutError):
            return None
        finally:
            sock.close()


def bind_to_device(sock: socket.socket, interface: str | None) -> None:
    if not interface:
        return
    option = getattr(socket, "SO_BINDTODEVICE", None)
    if option is None:
        raise NetworkOperationError("SO_BINDTODEVICE is not supported on this platform")
    try:
        sock.setsockopt(socket.SOL_SOCKET, option, interface.encode("utf-8") + b"\0")
    except OSError as exc:
        raise NetworkOperationError(
            f"Unable to bind outbound socket to {interface}: {exc}"
        ) from exc


def build_dns_query(host: str, query_type: int, transaction_id: int) -> bytes:
    labels = host.rstrip(".").split(".")
    encoded_name = b""
    for label in labels:
        encoded = label.encode("idna")
        if not encoded or len(encoded) > 63:
            raise ValueError(f"Invalid DNS label in {host}")
        encoded_name += bytes([len(encoded)]) + encoded
    encoded_name += b"\0"
    header = transaction_id.to_bytes(2, "big") + b"\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00"
    return header + encoded_name + query_type.to_bytes(2, "big") + b"\x00\x01"


def parse_dns_response(response: bytes, transaction_id: int, query_type: int) -> str | None:
    if len(response) < 12 or int.from_bytes(response[:2], "big") != transaction_id:
        return None
    if response[3] & 0x0F:
        return None

    offset = 12
    question_count = int.from_bytes(response[4:6], "big")
    answer_count = int.from_bytes(response[6:8], "big")
    for _ in range(question_count):
        offset = skip_dns_name(response, offset)
        offset += 4

    for _ in range(answer_count):
        offset = skip_dns_name(response, offset)
        if offset + 10 > len(response):
            return None
        answer_type = int.from_bytes(response[offset : offset + 2], "big")
        answer_class = int.from_bytes(response[offset + 2 : offset + 4], "big")
        data_length = int.from_bytes(response[offset + 8 : offset + 10], "big")
        offset += 10
        data = response[offset : offset + data_length]
        offset += data_length
        if answer_class != 1 or answer_type != query_type:
            continue
        if query_type == 1 and len(data) == 4:
            return socket.inet_ntop(socket.AF_INET, data)
        if query_type == 28 and len(data) == 16:
            return socket.inet_ntop(socket.AF_INET6, data)
    return None


def skip_dns_name(packet: bytes, offset: int) -> int:
    while offset < len(packet):
        length = packet[offset]
        if length == 0:
            return offset + 1
        if length & 0xC0 == 0xC0:
            return offset + 2
        offset += length + 1
    return offset
