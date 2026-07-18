from __future__ import annotations

import asyncio


async def relay_bidirectional(
    client_reader: asyncio.StreamReader,
    client_writer: asyncio.StreamWriter,
    upstream_reader: asyncio.StreamReader,
    upstream_writer: asyncio.StreamWriter,
    *,
    idle_timeout: float,
) -> None:
    client_to_upstream = asyncio.create_task(
        copy_stream(client_reader, upstream_writer, idle_timeout=idle_timeout)
    )
    upstream_to_client = asyncio.create_task(
        copy_stream(upstream_reader, client_writer, idle_timeout=idle_timeout)
    )
    tasks = {client_to_upstream, upstream_to_client}
    done, pending = await asyncio.wait(tasks, return_when=asyncio.FIRST_COMPLETED)
    for task in pending:
        task.cancel()
    await asyncio.gather(*pending, return_exceptions=True)
    for task in done:
        task.result()


async def copy_stream(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    *,
    idle_timeout: float,
) -> None:
    while True:
        data = await asyncio.wait_for(reader.read(64 * 1024), timeout=idle_timeout)
        if not data:
            return
        writer.write(data)
        await writer.drain()


async def close_writer(writer: asyncio.StreamWriter | None) -> None:
    if writer is None:
        return
    writer.close()
    try:
        await writer.wait_closed()
    except (ConnectionError, OSError):
        pass
