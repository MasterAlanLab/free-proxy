"""Add persistent jobs and probe history."""

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "20260717_0002"
down_revision: str | Sequence[str] | None = "20260717_0001"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    inspector = sa.inspect(op.get_bind())
    tables = set(inspector.get_table_names())
    columns = {column["name"] for column in inspector.get_columns("proxy_nodes")}
    for name, column in (
        ("provider_identity", sa.String(length=255)),
        ("last_seen_at", sa.DateTime(timezone=True)),
        ("source_present", sa.Boolean()),
    ):
        if name not in columns:
            op.add_column("proxy_nodes", sa.Column(name, column, nullable=True))
    op.execute(
        sa.text(
            "UPDATE proxy_nodes SET provider_identity = provider || ':' || ip_address "
            "WHERE provider_identity IS NULL OR provider_identity = ''"
        )
    )
    op.execute(sa.text("UPDATE proxy_nodes SET source_present = 1 WHERE source_present IS NULL"))
    op.execute(
        sa.text("UPDATE proxy_nodes SET last_seen_at = fetched_at WHERE last_seen_at IS NULL")
    )
    if "ip_info_cache" not in tables:
        op.create_table(
            "ip_info_cache",
            sa.Column("ip_address", sa.String(length=64), primary_key=True),
            sa.Column("owner", sa.String(length=255), nullable=False, server_default=""),
            sa.Column("asn", sa.String(length=128), nullable=False, server_default=""),
            sa.Column("as_name", sa.String(length=255), nullable=False, server_default=""),
            sa.Column("location", sa.String(length=255), nullable=False, server_default=""),
            sa.Column("ip_type", sa.String(length=16), nullable=False, server_default="unknown"),
            sa.Column("quality", sa.String(length=32), nullable=False, server_default=""),
            sa.Column("updated_at", sa.DateTime(timezone=True), nullable=False),
        )
    if "node_aliases" not in tables:
        op.create_table(
            "node_aliases",
            sa.Column("alias_id", sa.String(length=96), primary_key=True),
            sa.Column("node_id", sa.String(length=96), nullable=False),
            sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
        )
    if "jobs" not in tables:
        op.create_table(
            "jobs",
            sa.Column("id", sa.String(length=64), primary_key=True),
            sa.Column("name", sa.String(length=128), nullable=False),
            sa.Column("status", sa.String(length=32), nullable=False),
            sa.Column("created_at", sa.DateTime(timezone=True), nullable=False),
            sa.Column("started_at", sa.DateTime(timezone=True), nullable=True),
            sa.Column("finished_at", sa.DateTime(timezone=True), nullable=True),
            sa.Column("result", sa.JSON(), nullable=True),
            sa.Column("error", sa.Text(), nullable=True),
        )
        op.create_index("ix_jobs_name", "jobs", ["name"])
        op.create_index("ix_jobs_status", "jobs", ["status"])
        op.create_index("ix_jobs_created_at", "jobs", ["created_at"])
    if "probe_results" not in tables:
        op.create_table(
            "probe_results",
            sa.Column("id", sa.Integer(), primary_key=True, autoincrement=True),
            sa.Column("node_id", sa.String(length=96), nullable=False),
            sa.Column("available", sa.Boolean(), nullable=False),
            sa.Column("latency_ms", sa.Integer(), nullable=False),
            sa.Column("probed_at", sa.DateTime(timezone=True), nullable=False),
            sa.Column("result", sa.JSON(), nullable=False),
        )
        op.create_index("ix_probe_results_node_id", "probe_results", ["node_id"])
        op.create_index("ix_probe_results_available", "probe_results", ["available"])
        op.create_index("ix_probe_results_probed_at", "probe_results", ["probed_at"])


def downgrade() -> None:
    inspector = sa.inspect(op.get_bind())
    tables = set(inspector.get_table_names())
    if "probe_results" in tables:
        op.drop_index("ix_probe_results_probed_at", table_name="probe_results")
        op.drop_index("ix_probe_results_available", table_name="probe_results")
        op.drop_index("ix_probe_results_node_id", table_name="probe_results")
        op.drop_table("probe_results")
    if "jobs" in tables:
        op.drop_index("ix_jobs_created_at", table_name="jobs")
        op.drop_index("ix_jobs_status", table_name="jobs")
        op.drop_index("ix_jobs_name", table_name="jobs")
        op.drop_table("jobs")
