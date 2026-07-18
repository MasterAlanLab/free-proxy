"""Create the core Free Proxy schema."""

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "20260717_0001"
down_revision: str | Sequence[str] | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "proxy_nodes",
        sa.Column("id", sa.String(length=96), primary_key=True),
        sa.Column("provider", sa.String(length=32), nullable=False),
        sa.Column("provider_node_id", sa.String(length=255), nullable=False, server_default=""),
        sa.Column("country", sa.String(length=128), nullable=False, server_default=""),
        sa.Column("country_code", sa.String(length=16), nullable=False, server_default=""),
        sa.Column("host_name", sa.String(length=255), nullable=False, server_default=""),
        sa.Column("ip_address", sa.String(length=64), nullable=False),
        sa.Column("remote_host", sa.String(length=255), nullable=False),
        sa.Column("remote_port", sa.Integer(), nullable=False),
        sa.Column("transport", sa.String(length=16), nullable=False, server_default="unknown"),
        sa.Column("ip_type", sa.String(length=16), nullable=False, server_default="unknown"),
        sa.Column("owner", sa.String(length=255), nullable=False, server_default=""),
        sa.Column("asn", sa.String(length=128), nullable=False, server_default=""),
        sa.Column("as_name", sa.String(length=255), nullable=False, server_default=""),
        sa.Column("location", sa.String(length=255), nullable=False, server_default=""),
        sa.Column("quality", sa.String(length=32), nullable=False, server_default=""),
        sa.Column("status", sa.String(length=16), nullable=False, server_default="discovered"),
        sa.Column("source_score", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("source_ping_ms", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("source_speed_bps", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("source_sessions", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("latency_ms", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("consecutive_failures", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("success_count", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("failure_count", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("config_text", sa.Text(), nullable=False),
        sa.Column("fetched_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("last_probed_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("last_success_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("ip_info_updated_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("cooldown_until", sa.DateTime(timezone=True), nullable=True),
    )
    for column in ("provider", "country", "country_code", "ip_address", "ip_type", "status"):
        op.create_index(f"ix_proxy_nodes_{column}", "proxy_nodes", [column])

    op.create_table(
        "runtime_settings",
        sa.Column("id", sa.Integer(), primary_key=True),
        sa.Column("routing_mode", sa.String(length=32), nullable=False, server_default="auto"),
        sa.Column("force_country", sa.String(length=128), nullable=False, server_default=""),
        sa.Column("routing_ip_type", sa.String(length=32), nullable=False, server_default="all"),
        sa.Column("connection_enabled", sa.Boolean(), nullable=False, server_default=sa.true()),
        sa.Column("fixed_node_id", sa.String(length=96), nullable=True),
    )
    op.create_table(
        "favorites",
        sa.Column("node_id", sa.String(length=96), primary_key=True),
    )
    op.create_table(
        "node_blacklist",
        sa.Column("node_id", sa.String(length=96), primary_key=True),
        sa.Column("reason", sa.Text(), nullable=False, server_default=""),
        sa.Column("marked_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("expires_at", sa.DateTime(timezone=True), nullable=False),
    )
    op.create_index("ix_node_blacklist_expires_at", "node_blacklist", ["expires_at"])


def downgrade() -> None:
    op.drop_index("ix_node_blacklist_expires_at", table_name="node_blacklist")
    op.drop_table("node_blacklist")
    op.drop_table("favorites")
    op.drop_table("runtime_settings")
    for column in ("status", "ip_type", "ip_address", "country_code", "country", "provider"):
        op.drop_index(f"ix_proxy_nodes_{column}", table_name="proxy_nodes")
    op.drop_table("proxy_nodes")
