# Free Proxy 重构文档：Go + sqlc + SQLite + Echo v5 + React

> 目标：把当前 Python（FastAPI + SQLAlchemy + asyncio）实现，重构为 **单一静态二进制**。
> 编译产物内嵌前端资源，落地即可运行；OpenVPN、iproute2 等系统依赖由二进制自身检测并安装。

本文是**实现级重构蓝图**，不是概念草案。所有表结构、枚举、配置项、API 契约、代理协议细节均对齐当前 Python 代码，直接照此实现即可保持行为一致。

---

## 0. 目标与不变量

**最终形态**

- 单个可执行文件 `free-proxy`（`CGO_ENABLED=0`，纯静态，无外部 `.so` 依赖）。
- 前端 React 构建产物 `embed` 进二进制，`/` 直出 SPA，无需单独部署静态资源。
- 数据库 SQLite，schema 迁移随二进制携带并在启动时自动执行（等价当前 `database-upgrade`）。
- 系统级依赖（openvpn、ip、sysctl、`/dev/net/tun`）由二进制 `doctor`/`preflight` 子命令检测，缺失时按发行版包管理器安装。

**必须保持不变的对外契约**（保证可平滑替换、前端/客户端零改动）

1. REST API 路径与语义：`/{secret_path}/api/v1/...`，26 个端点，长任务返回 `202 + Job` 后轮询。
2. 环境变量：`FREE_PROXY_` 前缀，键名与语义与现状一致。
3. 代理端口行为：单端口 `9527` 首字节识别 SOCKS5/HTTP，出站强制绑 `tun0`，DNS 走隧道。
4. 密码哈希格式：`scrypt$16384$8$1$<b64salt>$<b64digest>`（Go 与 Python 完全兼容，见 §12）。
5. 数据目录布局：`data_dir/` 下 `free-proxy.db`、`configs/`、`logs/`、`web-config.json`、`initial-admin-password`。

---

## 1. 技术栈映射总表

| 关注点 | 当前（Python） | 重构后（Go） | 说明 |
|---|---|---|---|
| 语言/运行时 | CPython 3.11 | Go 1.23+ | 静态编译、无 GIL、goroutine 并发 |
| Web 框架 | FastAPI + Starlette | **Echo v5** | 路由/中间件/绑定 |
| ASGI/HTTP 服务器 | uvicorn | `net/http`（Echo 内置） | 标准库 |
| ORM/数据访问 | SQLAlchemy async | **sqlc**（生成类型安全查询） | 手写 SQL + 代码生成 |
| DB 驱动 | aiosqlite | **modernc.org/sqlite**（纯 Go，无 CGO） | 单二进制的关键决策，§2.1 |
| 迁移 | Alembic | **goose**（embed 迁移） | §5.3 |
| 校验/DTO | Pydantic | 结构体 + `go-playground/validator` | §6 |
| 配置 | pydantic-settings | **caarlos0/env/v11** | `FREE_PROXY_` 前缀 |
| CLI | Typer | **spf13/cobra** | serve/discover/credentials/... |
| 日志 | 自研 JSON logger + LogStore | **log/slog** + 自定义 Handler | 可查询/按天滚动/3 天清理 |
| 密码哈希 | `hashlib.scrypt` | `golang.org/x/crypto/scrypt` | 格式互通 |
| 会话 | 内存 dict + asyncio.Lock | 内存 map + `sync.Mutex` | §12 |
| 并发模型 | asyncio 事件循环 | goroutine + `context.Context` | §4 |
| 前端 | React 19 + Vite + Tailwind v4 + Zustand | 不变 | 构建产物 embed |
| 前端交付 | `web/dist`（当前未构建，回落 Jinja） | `//go:embed` 进二进制 | §13 |
| 代理协议 | 手写 asyncio SOCKS5/HTTP | 手写 `net` SOCKS5/HTTP | §7 |
| 绑定网卡 | `SO_BINDTODEVICE`（手动 socket） | `net.Dialer.Control` + `x/sys/unix` | §7.3，更干净 |
| 首字节识别 | `StreamReader._buffer` 私有 hack | `bufio.Reader.Peek(1)` | §7.2，消除私有 API 依赖 |
| 进程管理 | asyncio subprocess | `os/exec` + `context` | §8 |
| 依赖安装 | 外部 `install.sh` | **内置 `doctor` 子命令** | §14 |
| 构建 | uv + uv_build | `go build` + Makefile | §15 |

---

## 2. 关键技术决策

### 2.1 SQLite 驱动：`modernc.org/sqlite`（无 CGO）—— 单二进制的基石

这是整份文档最关键的一个决定。

- **`mattn/go-sqlite3`**：性能好、生态成熟，但**依赖 CGO**。开启 CGO 后交叉编译困难、产物动态链接 libc，破坏"一个静态二进制到处跑"的目标。
- **`modernc.org/sqlite`**：SQLite 的**纯 Go 转译实现**，`CGO_ENABLED=0` 即可编译，交叉编译零成本，产物完全静态。

选择 **`modernc.org/sqlite`**。它通过 `database/sql` 暴露，驱动名为 `sqlite`，与 sqlc 生成的代码天然兼容。性能对本项目（低频管理操作 + 少量节点行）完全足够。

```go
import (
    "database/sql"
    _ "modernc.org/sqlite"
)

// dsn: file:/var/lib/free-proxy/free-proxy.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)
db, err := sql.Open("sqlite", dsn)
```

务必设置 `busy_timeout` 与 `journal_mode=WAL`：多 goroutine 并发读写时避免 `SQLITE_BUSY`，等价当前单写连接的安全性。写操作建议串行化（写用一个 `SetMaxOpenConns(1)` 的连接池，或依赖 WAL + busy_timeout 容忍并发）。

### 2.2 sqlc + goose：类型安全 SQL 与迁移的分工

`sqlc` **不做迁移**，只把手写 SQL 编译成类型安全的 Go 函数。迁移由 `goose` 负责。两者共享同一份 schema：

- `internal/store/migrations/*.sql`：goose 迁移文件（`-- +goose Up/Down`），也是 sqlc 读取 schema 的来源。
- `internal/store/queries/*.sql`：sqlc 的查询定义（`-- name: ListNodes :many` 等）。
- `sqlc generate` 产出 `internal/store/gen/`（`models.go` + `*.sql.go` + `querier.go`）。

这样 Alembic 的两个迁移（core_schema、job_probe_history）平移为两个 goose 迁移，且集成测试仍可做 up/down 往返验证。

### 2.3 关于 Echo v5

按需求采用 **Echo v5**（`github.com/labstack/echo/v5`）。注意：截至重构时若 v5 尚未 GA，其 API 与 v4（`github.com/labstack/echo/v4`）高度一致，可作为无痛回退——本文所有 Echo 代码在两个大版本上语义相同（`e.Pre()` 预路由中间件、`e.Group()`、`echo.WrapHandler` 等均保留）。选型不影响架构，锁定版本在 `go.mod` 即可。

### 2.4 其余选型一句话理由

- **cobra**：子命令结构与 Typer 一一对应，成熟稳定。
- **caarlos0/env/v11**：结构体标签 + 前缀解析，最接近 pydantic-settings 的体验；范围校验用 `validator`。
- **log/slog**：标准库结构化日志，`JSONHandler` 直接产出与现状同构的 JSON；自定义 Handler 实现"可查询 + 按天文件 + 3 天清理"。
- **前端不动**：React/Vite/Tailwind/Zustand 保留，仅改交付方式（embed）。

---

## 3. 目标项目结构

```
free-proxy/
├── cmd/
│   └── free-proxy/
│       └── main.go                 # 入口：cobra root，装配依赖
├── internal/
│   ├── config/                     # env 解析（FREE_PROXY_ 前缀）
│   │   └── config.go
│   ├── domain/                     # 领域类型 + 枚举（对齐 enums.py / models.py）
│   │   ├── enums.go
│   │   └── models.go
│   ├── store/                      # 数据层
│   │   ├── sqlc.yaml
│   │   ├── migrations/             # goose，//go:embed
│   │   │   ├── 0001_core_schema.sql
│   │   │   └── 0002_job_probe_history.sql
│   │   ├── queries/                # sqlc 查询源
│   │   │   ├── nodes.sql  jobs.sql  probes.sql  settings.sql
│   │   ├── gen/                    # sqlc 生成（勿手改）
│   │   ├── db.go                   # Open + WAL + goose Up
│   │   └── repo.go                 # 在 gen 之上的仓储封装（对齐 4 个 Repository）
│   ├── services/                   # 用例编排（对齐 13 个 service）
│   │   ├── discovery.go  probe.go  gateway.go  pool.go
│   │   ├── maintenance.go  health.go  autoswitch.go
│   │   ├── jobs.go  settings.go  ipinfo.go
│   │   ├── diagnostics.go  operations.go   # operations = 网络操作互斥协调
│   ├── proxy/                      # 本地 SOCKS5/HTTP 网关
│   │   ├── unified.go  socks5.go  http.go
│   │   ├── connector.go  relay.go  dns.go  gateway.go
│   ├── tunnel/                     # OpenVPN 生命周期
│   │   ├── openvpn.go  command.go  process.go  logparse.go
│   ├── netx/                       # 系统网络操作
│   │   ├── routing.go  latency.go  upstream.go  tun.go  commands.go
│   ├── providers/
│   │   └── vpngate/                # client.go + parser.go
│   ├── ipinfo/                     # ip-api.com 批量分类
│   │   └── client.go
│   ├── api/                        # Echo 层
│   │   ├── server.go               # echo.New + 路由挂载 + 静态
│   │   ├── middleware.go           # secret-path + session 鉴权
│   │   ├── deps.go                 # 依赖容器（取代 app.state）
│   │   └── handlers/               # auth/gateway/jobs/logs/pool/proxies/settings/system
│   ├── security/                   # scrypt + session + admin config store
│   │   └── security.go
│   ├── platform/                   # 依赖检测与安装（doctor/preflight）
│   │   ├── detect.go  install.go  pkgmanager.go
│   ├── logging/                    # slog handler + 可查询 LogStore
│   │   └── logging.go
│   └── web/
│       ├── embed.go                # //go:embed dist
│       └── dist/                   # 前端构建产物（gitignore，构建时生成）
├── frontend/                       # React 源（保留现有）
│   ├── src/  index.html  package.json  vite.config.ts
├── deploy/                         # systemd / openrc / env.example（保留）
├── Makefile
├── go.mod
└── go.sum
```

依赖方向沿用当前项目已验证的单向规则：
`api → services → {store, proxy, tunnel, netx, providers, ipinfo, security} → domain`。
`domain` 零依赖；`proxy` 只依赖 `config` + `domain`，可独立测试。

---

## 4. 并发模型迁移：asyncio → goroutine + context

这是 Python→Go 概念差异最大的部分，先讲清楚，后续各模块都遵循它。

| asyncio 概念 | Go 对应 | 备注 |
|---|---|---|
| 事件循环 + `async def` | goroutine（`go f()`） | 无显式循环，运行时调度 |
| `await` | 阻塞调用（runtime 自动让出） | 代码写成同步风格 |
| `asyncio.Lock` | `sync.Mutex` | |
| `asyncio.Semaphore(n)` | 容量 n 的 `chan struct{}` | 见下 |
| `asyncio.wait_for(coro, t)` | `context.WithTimeout` | |
| `asyncio.create_task` + 取消 | goroutine + `ctx.Done()` | |
| lifespan 后台循环 | `ticker` + `for { select }` | 每个守护循环一个 goroutine |
| `app.state.*` DI | 一个 `Deps` 结构体显式注入 | 取代隐式全局 |

**信号量惯用法**（代理并发限制 `proxy_max_connections`）：

```go
type Semaphore chan struct{}
func (s Semaphore) TryAcquire() bool { select { case s <- struct{}{}: return true; default: return false } }
func (s Semaphore) Release()         { <-s }
```

**后台守护循环**（对齐当前的 maintenance / health / active-ping 三个后台任务）：

```go
func (m *MaintenanceService) Run(ctx context.Context) {
    t := time.NewTicker(m.cfg.MaintenanceInterval)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            if err := m.runOnce(ctx); err != nil {
                slog.Warn("maintenance failed", "err", err)
            }
        }
    }
}
```

主程序在 `serve` 里用一个根 `context`，收到 `SIGTERM/SIGINT` 时 `cancel()`，所有 goroutine 收敛后再退出（graceful shutdown）。这替代了 lifespan 的 startup/shutdown 钩子。

**网络操作互斥**（当前 `NetworkOperationCoordinator` + `OperationConflictError`）：用一个持有当前操作名的结构 + `sync.Mutex`，`TryBegin(name)` 失败即对应 API 返回 `409`。

---

## 5. 数据层：schema、sqlc、迁移

### 5.1 表清单（对齐当前 8 张表）

| 表 | 主键 | 用途 |
|---|---|---|
| `proxy_nodes` | `id` (TEXT) | 节点主表，唯一约束 `(provider, provider_identity)` |
| `ip_info_cache` | `ip_address` | ip-api 分类缓存（7 天） |
| `node_aliases` | `alias_id` | 节点去重别名映射 |
| `runtime_settings` | `id`(=1 单行) | 路由模式/国家/IP 类型/开关/固定节点 |
| `favorites` | `node_id` | 收藏 |
| `node_blacklist` | `node_id` | 失败退避黑名单（`expires_at`） |
| `jobs` | `id` (TEXT) | 异步任务（202 轮询） |
| `probe_results` | `id`(自增) | 探测历史 |

### 5.2 迁移 0001（核心 schema，goose + sqlc 共用）

SQLite 类型约定：时间统一存 `TEXT`（RFC3339 UTC 字符串），布尔存 `INTEGER`（0/1），Job/Probe 的 `result` 存 `TEXT`（JSON）。

```sql
-- internal/store/migrations/0001_core_schema.sql
-- +goose Up
CREATE TABLE proxy_nodes (
    id                 TEXT PRIMARY KEY,
    provider           TEXT NOT NULL DEFAULT '',
    provider_node_id   TEXT NOT NULL DEFAULT '',
    provider_identity  TEXT NOT NULL DEFAULT '',
    country            TEXT NOT NULL DEFAULT '',
    country_code       TEXT NOT NULL DEFAULT '',
    host_name          TEXT NOT NULL DEFAULT '',
    ip_address         TEXT NOT NULL DEFAULT '',
    remote_host        TEXT NOT NULL DEFAULT '',
    remote_port        INTEGER NOT NULL DEFAULT 0,
    transport          TEXT NOT NULL DEFAULT 'unknown',
    ip_type            TEXT NOT NULL DEFAULT 'unknown',
    owner              TEXT NOT NULL DEFAULT '',
    asn                TEXT NOT NULL DEFAULT '',
    as_name            TEXT NOT NULL DEFAULT '',
    location           TEXT NOT NULL DEFAULT '',
    quality            TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'discovered',
    source_score       INTEGER NOT NULL DEFAULT 0,
    source_ping_ms     INTEGER NOT NULL DEFAULT 0,
    source_speed_bps   INTEGER NOT NULL DEFAULT 0,
    source_sessions    INTEGER NOT NULL DEFAULT 0,
    latency_ms         INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    success_count      INTEGER NOT NULL DEFAULT 0,
    failure_count      INTEGER NOT NULL DEFAULT 0,
    config_text        TEXT NOT NULL DEFAULT '',
    fetched_at         TEXT NOT NULL,
    last_probed_at     TEXT,
    last_success_at    TEXT,
    ip_info_updated_at TEXT,
    cooldown_until     TEXT,
    last_seen_at       TEXT,
    source_present     INTEGER NOT NULL DEFAULT 1,
    UNIQUE (provider, provider_identity)
);
CREATE INDEX idx_nodes_provider ON proxy_nodes(provider);
CREATE INDEX idx_nodes_country  ON proxy_nodes(country);
CREATE INDEX idx_nodes_ip_type  ON proxy_nodes(ip_type);
CREATE INDEX idx_nodes_status   ON proxy_nodes(status);
CREATE INDEX idx_nodes_present  ON proxy_nodes(source_present);

CREATE TABLE ip_info_cache (
    ip_address TEXT PRIMARY KEY,
    owner TEXT NOT NULL DEFAULT '', asn TEXT NOT NULL DEFAULT '',
    as_name TEXT NOT NULL DEFAULT '', location TEXT NOT NULL DEFAULT '',
    ip_type TEXT NOT NULL DEFAULT 'unknown', quality TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);
CREATE TABLE node_aliases (
    alias_id TEXT PRIMARY KEY, node_id TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX idx_aliases_node ON node_aliases(node_id);
CREATE TABLE runtime_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    routing_mode TEXT NOT NULL DEFAULT 'auto',
    force_country TEXT NOT NULL DEFAULT '',
    routing_ip_type TEXT NOT NULL DEFAULT 'all',
    connection_enabled INTEGER NOT NULL DEFAULT 1,
    fixed_node_id TEXT
);
INSERT INTO runtime_settings (id) VALUES (1);
CREATE TABLE favorites (node_id TEXT PRIMARY KEY);
CREATE TABLE node_blacklist (
    node_id TEXT PRIMARY KEY, reason TEXT NOT NULL DEFAULT '',
    marked_at TEXT NOT NULL, expires_at TEXT NOT NULL
);
CREATE INDEX idx_blacklist_expires ON node_blacklist(expires_at);

-- +goose Down
DROP TABLE node_blacklist; DROP TABLE favorites; DROP TABLE runtime_settings;
DROP TABLE node_aliases; DROP TABLE ip_info_cache; DROP TABLE proxy_nodes;
```

迁移 0002 建 `jobs` 与 `probe_results`（`jobs.result` / `probe_results.result` 为 `TEXT` JSON，`probe_results.id` 自增），对齐 Alembic 第二个迁移。

### 5.3 sqlc 配置

```yaml
# internal/store/sqlc.yaml
version: "2"
sql:
  - engine: "sqlite"
    schema: "migrations"      # 直接用 goose 迁移当 schema 源
    queries: "queries"
    gen:
      go:
        package: "gen"
        out: "gen"
        emit_json_tags: true
        emit_interface: true          # 生成 Querier 接口，便于 mock
        emit_empty_slices: true
```

查询示例（`queries/nodes.sql`）：

```sql
-- name: ListNodes :many
SELECT * FROM proxy_nodes
WHERE (@ip_type = '' OR ip_type = @ip_type)
  AND (@status  = '' OR status  = @status)
  AND (@country = '' OR country = @country)
  AND (@current_only = 0 OR source_present = 1)
ORDER BY latency_ms ASC, source_score DESC
LIMIT @lim OFFSET @off;

-- name: GetNode :one
SELECT * FROM proxy_nodes WHERE id = ? LIMIT 1;

-- name: UpsertNode :exec
INSERT INTO proxy_nodes (...) VALUES (...) ON CONFLICT(id) DO UPDATE SET ...;
```

### 5.4 迁移执行（启动时自动 + CLI）

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *sql.DB) error {
    goose.SetBaseFS(migrationsFS)
    if err := goose.SetDialect("sqlite3"); err != nil { return err }
    return goose.Up(db, "migrations")
}
```

`serve` 启动前调用 `Migrate`；`database-upgrade` 子命令只调 `Migrate` 后退出，行为等价现状。

### 5.5 仓储层（`store/repo.go`）

在 sqlc 生成代码之上薄封装，对齐当前 4 个 Repository：`NodeRepository`、`SettingsRepository`、`JobRepository`、`ProbeResultRepository`。它们负责：DTO ↔ DB 行映射、时间 `time.Time`↔RFC3339 字符串、JSON 字段编解码、把多条 sqlc 调用组合成一个业务方法（如"upsert 节点并合并 IP 信息"）。服务层只依赖这四个仓储，不直接碰 sqlc 生成物。

---

## 6. 领域模型与枚举

枚举用 `string` 常量组，值与当前 `enums.py` 一一对应（DB、API 字符串完全不变）：

```go
// internal/domain/enums.go
package domain

type IpType string
const (
    IpResidential IpType = "residential"
    IpMobile      IpType = "mobile"
    IpHosting     IpType = "hosting"
    IpUnknown     IpType = "unknown"
)

type ProxyPolicyMode string
const (
    PolicyAuto             ProxyPolicyMode = "auto"
    PolicyResidentialFirst ProxyPolicyMode = "residential_first"
    PolicyCountry          ProxyPolicyMode = "country"
    PolicyFixed            ProxyPolicyMode = "fixed"
    PolicyFavorites        ProxyPolicyMode = "favorites"
)
// 同理：NodeStatus / TransportProtocol / JobStatus / RoutingIpType /
//       TunnelStatus / TunnelFailureCode，值全部对齐 enums.py
```

DTO 用普通结构体 + `json` 标签 + `validator` 标签替代 Pydantic：

```go
type ProxyNodePage struct {
    Items  []ProxyNodeRead `json:"items"`
    Total  int64           `json:"total"`
    Limit  int             `json:"limit"`
    Offset int             `json:"offset"`
}
type ProbeManyRequest struct {
    IDs []string `json:"ids" validate:"required,min=1,dive,required"`
}
```

Echo 的 `c.Bind(&req)` + `c.Validate(&req)`（注册 `go-playground/validator`）替代 FastAPI 的自动校验。

---

## 7. 代理网关（`internal/proxy/`）

这是重构的技术核心。行为对齐现状：单端口首字节识别、SOCKS5(RFC1928/1929)/HTTP(CONNECT + 普通转发)、出站绑 `tun0`、DNS 走隧道、双向中继带空闲超时、可选代理认证、连接数信号量。

### 7.1 网关组装（`gateway.go`）

```go
type Gateway struct {
    cfg       *config.Config
    connector *Connector          // 绑 tun0 的出站拨号器
    sem       Semaphore
}

func (g *Gateway) Serve(ctx context.Context) error {
    ln, err := net.Listen("tcp", net.JoinHostPort(g.cfg.ProxyHost, strconv.Itoa(g.cfg.ProxyPort)))
    if err != nil { return err }
    go func() { <-ctx.Done(); ln.Close() }()
    for {
        conn, err := ln.Accept()
        if err != nil { return err } // ctx 取消关闭 listener 后 Accept 出错退出
        go g.handle(ctx, conn)
    }
}
```

### 7.2 首字节识别（消除 Python 的私有 API hack）

Python 版靠改 `StreamReader._buffer` 回填探测字节，属实现细节依赖。Go 用 `bufio.Reader.Peek` 天然解决——peek 不消费缓冲，把这个 `*bufio.Reader` 传给下游即可：

```go
func (g *Gateway) handle(ctx context.Context, conn net.Conn) {
    defer conn.Close()
    if !g.sem.TryAcquire() { return }      // 超过 max_connections 直接关闭，对齐现状
    defer g.sem.Release()

    br := bufio.NewReader(conn)
    conn.SetReadDeadline(time.Now().Add(15 * time.Second))
    first, err := br.Peek(1)               // 不消费
    if err != nil { return }
    conn.SetReadDeadline(time.Time{})      // 清除

    switch {
    case first[0] == 0x05:
        g.serveSOCKS5(ctx, conn, br)
    case isHTTPMethodStart(first[0]):      // 字母开头
        g.serveHTTP(ctx, conn, br)
    default:
        // 未知协议，关闭
    }
}
```

> 注意：SOCKS5/HTTP 处理函数读取数据必须用 `br`（含已缓冲字节），写回用 `conn`。

### 7.3 出站拨号绑定 `tun0`（`connector.go`）

Python 手动建 socket 调 `SO_BINDTODEVICE`。Go 用 `net.Dialer.Control` 回调在 connect 前设置 socket 选项，更简洁也更健壮（自动处理 happy-eyeballs、超时、地址排序）：

```go
func newDialer(cfg *config.Config) *net.Dialer {
    return &net.Dialer{
        Timeout: cfg.ProxyConnectTimeout,
        Control: func(network, address string, c syscall.RawConn) error {
            if cfg.TunnelInterface == "" { return nil }
            var seterr error
            if err := c.Control(func(fd uintptr) {
                seterr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, cfg.TunnelInterface)
            }); err != nil { return err }
            return seterr
        },
    }
}
```

`SO_BINDTODEVICE` 需要 `CAP_NET_RAW`（root）；权限不足返回 `EPERM`，设备不存在返回 `ENODEV`——上层翻译为现状的错误码 3006/3004。

### 7.4 DNS 走隧道防泄漏（`dns.go`）

现状是手写 DNS 报文经 tun0 查 `8.8.8.8`。Go 用纯 Go 解析器（`PreferGo: true`）配合绑定 tun0 的自定义 Dial，即可让 A/AAAA 查询走隧道，无需手搓报文：

```go
func newResolver(cfg *config.Config) *net.Resolver {
    d := &net.Dialer{Control: bindToDevice(cfg.TunnelInterface)}
    return &net.Resolver{
        PreferGo: true,   // 强制使用 Go 内建解析器，不走 libc/系统 resolver
        Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
            return d.DialContext(ctx, "udp", net.JoinHostPort(cfg.ProxyDNSServer, "53"))
        },
    }
}
```

> 若需与现状 100% 一致（手写报文 + A/AAAA 并查），可换 `github.com/miekg/dns`；但纯 Go resolver 方案零额外依赖，推荐先用它。

### 7.5 双向中继（`relay.go`）

```go
func relay(a, b net.Conn, idle time.Duration) {
    done := make(chan struct{}, 2)
    cp := func(dst, src net.Conn) {
        buf := make([]byte, 64*1024)     // 对齐现状 64KB
        for {
            src.SetReadDeadline(time.Now().Add(idle))
            n, err := src.Read(buf)
            if n > 0 { dst.Write(buf[:n]) }
            if err != nil { break }
        }
        done <- struct{}{}
    }
    go cp(a, b)
    go cp(b, a)
    <-done            // 任一方向结束即收工（对齐 FIRST_COMPLETED）
    a.Close(); b.Close()
}
```

### 7.6 SOCKS5 / HTTP

- `socks5.go`：方法协商 → 若配置认证走 RFC1929 用户名/口令（method 0x02）→ 仅 CONNECT → 解析 IPv4/IPv6/域名 → 用 §7.3 connector 出站 → 回 reply code（异常映射到 SOCKS 失败码）。
- `http.go`：`CONNECT` 隧道 + 普通请求转发（重写 request-line、剥离 `Proxy-*` 头）；认证走 `Proxy-Authorization: Basic`，缺失返回 `407 + Proxy-Authenticate`。
- 认证比较用 `crypto/subtle.ConstantTimeCompare`（对齐 `secrets.compare_digest`）。

---

## 8. 隧道与 OpenVPN 管理（`internal/tunnel/`）

### 8.1 命令构造（`command.go`）

直接平移当前 `OpenVpnCommandBuilder.build`，参数逐一对齐：

```go
func BuildArgs(p BuildParams) []string {
    args := []string{
        "--config", p.ConfigFile, "--dev", p.Device, "--dev-type", "tun",
        "--pull-filter", "ignore", "route-ipv6",
        "--pull-filter", "ignore", "ifconfig-ipv6",
        "--route-delay", "2", "--connect-retry-max", "1", "--connect-timeout", "15",
        "--auth-user-pass", p.AuthFile, "--auth-nocache", "--verb", "3",
    }
    ciphers := "AES-128-CBC:AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305"
    if p.Version.GTE(2, 5) {
        args = append(args, "--data-ciphers", ciphers)   // ≥2.5
    } else {
        args = append(args, "--ncp-ciphers", ciphers)    // <2.5
    }
    if fileExists("/etc/ssl/certs") { args = append(args, "--capath", "/etc/ssl/certs") }
    if p.RouteNoPull { args = append(args, "--route-nopull") }
    if p.Upstream != nil {
        opt := "--http-proxy"; if p.Upstream.Kind == "socks" { opt = "--socks-proxy" }
        args = append(args, opt, p.Upstream.Host, strconv.Itoa(p.Upstream.Port))
        if p.UpstreamAuthFile != "" { args = append(args, p.UpstreamAuthFile) }
    }
    return args
}
```

`get_openvpn_version()` 平移为解析 `openvpn --version` 首行。auth 文件写 `0600`（`os.WriteFile(path, data, 0o600)`）。

### 8.2 进程生命周期（`process.go`）

```go
type Managed struct {
    cmd    *exec.Cmd
    cancel context.CancelFunc
    logs   chan string
}

func Start(parent context.Context, bin string, args []string) (*Managed, error) {
    ctx, cancel := context.WithCancel(parent)
    cmd := exec.CommandContext(ctx, bin, args...)
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    if err := cmd.Start(); err != nil { cancel(); return nil, err }
    m := &Managed{cmd: cmd, cancel: cancel, logs: make(chan string, 256)}
    go m.scan(stdout); go m.scan(stderr)     // bufio.Scanner 逐行送 logs + logparse
    return m, nil
}
func (m *Managed) Stop() { m.cancel(); _ = m.cmd.Wait() }
```

- 等待握手：读 `logs` 通道，`logparse` 匹配 `Initialization Sequence Completed` 判成功，匹配 `AUTH_FAILED`/`TLS`/`RESOLVE` 等分类为 `TunnelFailureCode`（对齐 `log_parser.py`）。用 `context.WithTimeout(openvpn_connect_timeout)` 兜底。
- 清理残留：`process.go` 提供 `KillStray()`，扫描 `/proc/*/cmdline` 找本项目起的 openvpn（按 `--config` 指向 data_dir 识别），只杀自己的，对齐现状"只清理本项目残留进程"。

### 8.3 探测用临时网卡池（`tun2..tun99`）

`ProbeService` 并发拨号测速时，从 `test_tun_start..test_tun_end` 领取独占 tun 设备（`netx.TunAllocator`，一个带 mutex 的空闲索引集合），每个探测用 `--route-nopull` 起临时 OpenVPN，超时约 `openvpn_test_timeout`。并发上限 `max_probe_concurrency`（信号量）。

---

## 9. 网络层（`internal/netx/`）

全部是"拼 `ip`/`sysctl` 命令 + 解析输出"，用 `exec.CommandContext` 平移，逻辑与现状一致：

- `routing.go`（`PolicyRouter`）：`ip route add default dev tun0 table 100` → `ip rule add oif tun0 table 100` → 读取并设 `net.ipv4.conf.{all,default,tun0}.rp_filter=2`（记录原值以便 `cleanup` 还原）。带重试（`routing_setup_retries` / `routing_retry_interval`）与 `routing_strict_rp_filter` 严格模式。仅 Linux。
- `latency.go`：`ping` 延迟 + TCP 连接延迟回退，可绑定指定网卡。
- `upstream.go`：解析 `upstream_proxy_url` / `http_proxy` 等环境变量为 `(kind, host, port, user, pass)`。
- `tun.go`：`TunAllocator`（见 §8.3）。
- `commands.go`：`CommandRunner` 接口（`Run(ctx, args) (stdout, stderr, code)`），生产用 `exec`，测试注入假实现——对齐当前用 `Protocol` 抽象命令层的做法。

**IP 情报**（`internal/ipinfo/client.go`）：批量 POST `ip-api.com/batch`（每批 100），解析 `hosting`/`mobile`/`proxy` 推导 `IpType`，结果写 `ip_info_cache`（7 天）。用标准库 `net/http` + `encoding/json`。

---

## 10. 服务层（`internal/services/`）

13 个服务平移为 13 个结构体，构造函数注入依赖（仓储、netx、tunnel、config、logger）。要点：

| 服务 | 职责 | Go 关注点 |
|---|---|---|
| `DiscoveryService` | VPNGate 拉取+解析+入库 | HTTPS→免校验→HTTP 回退，走上游代理/直连 |
| `ProbeService` | 并发真实拨号测速 | 信号量 + TunAllocator + `errgroup` |
| `GatewayService` | 激活/断开出口、组装隧道+路由+代理 | 长任务，经 Job 执行 |
| `ProxyPoolService` | 按策略筛选可用节点 | 纯函数排序：`auto` 按 latency↑,score↓ |
| `MaintenanceService` | 周期拉取+测速+必要时切换 | 后台 goroutine（§4） |
| `HealthService` | 30s 经本地代理测真实出口 IP | 后台 goroutine |
| `ActiveLatencyMonitor` | 10s 刷新当前节点延迟 | 后台 goroutine |
| `AutoSwitchService` | 失败自动换节点 | 由 health/进程退出触发 |
| `JobService` | 异步任务提交+状态持久化 | goroutine 执行 + jobs 表 |
| `SettingsService` | 运行时设置/收藏 | |
| `IpInfoService` | 调用 ipinfo 客户端富化 | |
| `DiagnosticsService` | 系统诊断（最大，294 行） | 汇总各子系统健康 + 心跳 |
| `NetworkOperationCoordinator` | 网络操作互斥 | mutex + 当前操作名 → 冲突返回 409 |

**Job 系统**（对齐 202 轮询）：

```go
func (s *JobService) Submit(name string, fn func(ctx context.Context) (any, error)) (domain.JobRead, error) {
    id := newID()
    s.repo.Create(ctx, id, name, "pending")
    go func() {
        jobCtx := context.WithoutCancel(s.rootCtx) // 不随请求取消，但随进程根 ctx 取消
        s.repo.MarkRunning(jobCtx, id)
        res, err := fn(jobCtx)
        if err != nil { s.repo.MarkFailed(jobCtx, id, err.Error()) } else { s.repo.MarkSucceeded(jobCtx, id, res) }
    }()
    return s.repo.Get(ctx, id)
}
```

启动时把上次遗留的 `running`/`pending` 任务标记为 `cancelled`（对齐现状"取消未完成任务"）。

---

## 11. API 层（Echo v5）

### 11.1 路由清单（26 端点，路径/方法/状态码不变）

```
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/auth/config      PUT /api/v1/auth/config     (凭据/端口)
GET    /api/v1/auth/credentials
GET    /api/v1/proxies          (分页 + ip_type/status/country/search/include_history)
POST   /api/v1/proxies/discover      -> 202 Job
POST   /api/v1/proxies/refresh       -> 202 Job (409 若有并发操作)
POST   /api/v1/proxies/probe         -> 202 Job (最多 manual_test_node_limit 个)
POST   /api/v1/proxies/{id}/probe    -> 202 Job
GET    /api/v1/proxies/{id}/probes
POST   /api/v1/proxies/{id}/activate -> 202 Job
POST   /api/v1/proxies/{id}/favorite
GET    /api/v1/proxies/{id}/config   (下载 .ovpn)
GET    /api/v1/gateway/status
POST   /api/v1/gateway/check
POST   /api/v1/gateway/rotate
DELETE /api/v1/gateway/current
GET    /api/v1/pool/statistics
GET    /api/v1/jobs/{id}
GET    /api/v1/settings   PUT /api/v1/settings
GET    /api/v1/system/status
GET    /api/v1/system/diagnostics
POST   /api/v1/system/dns/repair
GET    /api/v1/logs       GET /api/v1/logs/export
```

### 11.2 依赖容器（取代 `app.state` + `dependencies.py`）

```go
type Deps struct {
    Cfg         *config.Config
    Repos       *store.Repos
    Discovery   *services.DiscoveryService
    Probe       *services.ProbeService
    Gateway     *services.GatewayService
    Maintenance *services.MaintenanceService
    Jobs        *services.JobService
    Pool        *services.ProxyPoolService
    Settings    *services.SettingsService
    Health      *services.HealthService
    Diag        *services.DiagnosticsService
    Coordinator *services.NetworkOperationCoordinator
    Auth        *security.AuthService
    Logs        *logging.Store
}
```

`main.go` 手工装配（等价 lifespan 的显式装配，无 DI 魔法），handler 通过 `*Deps` 闭包访问。

### 11.3 handler 示例

```go
func (h *Handlers) RefreshProxies(c echo.Context) error {
    if op := h.Deps.Coordinator.Current(); op != "" {
        return echo.NewHTTPError(http.StatusConflict, "Another network operation is running")
    }
    job, err := h.Deps.Jobs.Submit("refresh-proxies", h.Deps.Maintenance.RunJob)
    if err != nil { return err }
    return c.JSON(http.StatusAccepted, job)
}
```

---

## 12. 安全（`internal/security/`）

### 12.1 密码哈希（与 Python 完全互通）

`golang.org/x/crypto/scrypt` 与 `hashlib.scrypt` 同算法同参数，输出可互相校验，因此**现有 `web-config.json` 里的哈希无需重置**：

```go
func HashPassword(pw string) (string, error) {
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil { return "", err }
    dk, err := scrypt.Key([]byte(pw), salt, 1<<14, 8, 1, 32) // n=16384,r=8,p=1,dklen=32
    if err != nil { return "", err }
    return fmt.Sprintf("scrypt$16384$8$1$%s$%s",
        base64.URLEncoding.EncodeToString(salt),
        base64.URLEncoding.EncodeToString(dk)), nil
}
func VerifyPassword(pw, encoded string) bool {
    p := strings.SplitN(encoded, "$", 6)
    if len(p) != 6 || p[0] != "scrypt" { return false }
    n, _ := strconv.Atoi(p[1]); r, _ := strconv.Atoi(p[2]); pp, _ := strconv.Atoi(p[3])
    salt, _ := base64.URLEncoding.DecodeString(p[4])
    want, _ := base64.URLEncoding.DecodeString(p[5])
    got, err := scrypt.Key([]byte(pw), salt, n, r, pp, len(want))
    if err != nil { return false }
    return subtle.ConstantTimeCompare(got, want) == 1
}
```

### 12.2 会话与 admin 配置

- `SessionManager`：内存 `map[string]time.Time` + `sync.Mutex`，`token = hex(32 bytes)`，TTL `session_ttl_seconds`。Cookie `session`，`HttpOnly; SameSite=Lax; Path=/<secret>`。与现状同样是内存态（重启失效）——如需持久化可后续加一张 `sessions` 表，不属本次范围。
- `AdminConfigStore`：读写 `web-config.json`（`0600`，tmp+rename 原子写），生成 `initial-admin-password` 一次性文件，首次登录成功后删除。`random_credential()` 平移（首字符字母 + 大小写数字齐全）。

### 12.3 secret-path 中间件（Echo `Pre`）

对齐当前 ASGI 中间件：不匹配前缀返回 **404**（不暴露服务存在），匹配后剥前缀再交给路由；白名单 `/`、`/api/v1/auth/login`、`/static/*` 免鉴权，其余校验 session。用 `e.Pre()`（预路由）改写 `c.Request().URL.Path`：

```go
func SecretPath(auth *security.AuthService) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !auth.Cfg.AdminAuthEnabled { return next(c) }
            prefix := "/" + auth.Store.Config().SecretPath
            p := c.Request().URL.Path
            if p == prefix { return c.Redirect(http.StatusTemporaryRedirect, prefix+"/") }
            if !strings.HasPrefix(p, prefix+"/") {
                return echo.NewHTTPError(http.StatusNotFound, "Not found")
            }
            c.Request().URL.Path = strings.TrimPrefix(p, prefix)
            authed := auth.Sessions.Valid(readSessionCookie(c))
            c.Set("authorized", authed)
            if !authed && !isPublic(c.Request().URL.Path) {
                return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
            }
            return next(c)
        }
    }
}
```

---

## 13. 前端集成与 embed

前端源码 `frontend/` **保持不变**（React 19 + Vite + Tailwind v4 + Zustand，bun 构建）。只改两点：

1. Vite `build.outDir` 指向 `internal/web/dist`。
2. Go 用 `embed` 打进二进制，Echo 提供静态 + SPA 回退：

```go
// internal/web/embed.go
package web

import "embed"
//go:embed all:dist
var Dist embed.FS
```

```go
// api/server.go —— 挂在 secret-path 之后
sub, _ := fs.Sub(web.Dist, "dist")
e.GET("/*", echo.WrapHandler(spaHandler(http.FS(sub)))) // 命中文件返回文件，未命中回 index.html
```

> `all:dist` 前缀确保包含 Vite 产物里以 `_`/`.` 开头的文件。构建顺序：先 `bun run build` 生成 `internal/web/dist`，再 `go build`。`dist/` 加入 `.gitignore`，但为保证未构建前端时裸 `go build` 不报错，提交一个占位 `dist/index.html`；CI/发布流水线必须先构建前端。

这样彻底告别当前"React 未构建、回落 Jinja + vanilla JS"的半成品状态——二进制里只有一套前端。

---

## 14. 依赖自检与安装（`internal/platform/`）

这是本次重构的一个显式目标：**由二进制自身检测并安装系统依赖**，取代外部 `install.sh` 的相应职责。

### 14.1 检测（`detect.go`）

`doctor` 子命令逐项检查并输出人话诊断（对齐当前诊断错误码风格）：

| 检查项 | 方法 | 缺失后果 |
|---|---|---|
| `openvpn` | `exec.LookPath("openvpn")` + `--version` 取版本 | 无法建隧道 |
| `ip`(iproute2) | `exec.LookPath("ip")` | 无法配策略路由 |
| `sysctl` | `exec.LookPath("sysctl")` | 无法调 rp_filter |
| `/dev/net/tun` | `os.Stat("/dev/net/tun")` | 无法创建 TUN（LXC/OpenVZ 常见） |
| root 权限 | `os.Geteuid() == 0` | 绑网卡/改路由失败 |
| 发行版 | 解析 `/etc/os-release` 的 `ID`/`ID_LIKE` | 决定包管理器 |

```go
type Check struct { Name string; OK bool; Detail string; Fixable bool }
func RunChecks() []Check { /* openvpn / ip / sysctl / tun / root ... */ }
```

### 14.2 包管理器识别与安装（`pkgmanager.go` / `install.go`）

按 `LookPath` 探测可用包管理器（覆盖 aimili install.sh 支持的发行版矩阵）：

| 包管理器 | 发行版 | 安装命令 |
|---|---|---|
| `apt-get` | Debian/Ubuntu | `apt-get update && apt-get install -y openvpn iproute2 procps ca-certificates` |
| `apk` | Alpine | `apk add --no-cache openvpn iproute2 procps ca-certificates` |
| `dnf` | Fedora/RHEL9/Rocky/Alma | `dnf install -y openvpn iproute procps-ng ca-certificates` |
| `yum` | CentOS7/RHEL7/Amazon/Oracle | `yum install -y openvpn iproute procps-ng ca-certificates` |

```go
func Install(ctx context.Context, pkgs []string) error {
    pm := detectPkgManager() // apt-get / apk / dnf / yum
    if pm == nil { return errors.New("no supported package manager found") }
    for _, step := range pm.InstallSteps(pkgs) {
        if err := runVisible(ctx, step); err != nil {
            return fmt.Errorf("install %v failed: %w", pkgs, err)
        }
    }
    return nil
}
```

### 14.3 子命令与安装期行为

- `free-proxy doctor`：只检测并打印报告，退出码非 0 表示有缺失。
- `free-proxy doctor --fix` / `free-proxy install-deps`：检测 + 自动安装缺失项（需 root）。
- `free-proxy preflight`：`serve` 前置检查（对齐现状 systemd `ExecStartPre`），缺关键依赖直接失败并给修复建议。
- `serve` 启动时可选调用 preflight（配置 `FREE_PROXY_PREFLIGHT_STRICT`），缺 openvpn 时提示运行 `install-deps`。

> 设计原则：**默认只检测不擅自安装**（安装是有副作用的系统变更）；安装需显式 `--fix`/`install-deps` 或安装脚本调用。二进制不再依赖外部 `install.sh` 才能补齐依赖，但仍保留一个瘦 `install.sh` 用于下载二进制 + 装 systemd 单元（见 §16）。

---

## 15. 构建与打包（单静态二进制）

### 15.1 关键构建参数

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
  -o dist/free-proxy ./cmd/free-proxy
```

- `CGO_ENABLED=0`：纯静态（`modernc.org/sqlite` 让这成为可能）。
- `-s -w`：去符号表/调试信息，减小体积。
- `-trimpath`：去构建机路径。

### 15.2 交叉编译

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ... -o dist/free-proxy-linux-amd64 ...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ... -o dist/free-proxy-linux-arm64 ...
```

因为无 CGO，交叉编译零工具链成本，可在 macOS 上直接产出 Linux 二进制。

### 15.3 Makefile（构建顺序：前端 → sqlc → go build）

```makefile
.PHONY: frontend gen build all
frontend:
	cd frontend && bun install && bun run build   # 输出到 internal/web/dist
gen:
	sqlc generate -f internal/store/sqlc.yaml
build: frontend gen
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/free-proxy ./cmd/free-proxy
all: build
```

CI 里 `sqlc generate` 的产物应提交（或在 CI 校验无 diff），保证可复现。

---

## 16. 部署（systemd / OpenRC）

保留 `deploy/` 的两套单元，但简化——不再需要 `database-upgrade`/`preflight` 两个 `ExecStartPre`，因为迁移在 `serve` 内自动执行；preflight 可作为 `serve` 内前置或保留一个 ExecStartPre：

```ini
# deploy/free-proxy.service
[Service]
Type=simple
ExecStartPre=/opt/free-proxy/free-proxy doctor          # 缺依赖则启动失败并给提示
ExecStart=/opt/free-proxy/free-proxy serve
Restart=on-failure
NoNewPrivileges=true
PrivateTmp=true
EnvironmentFile=-/etc/free-proxy/free-proxy.env
```

`install.sh` 瘦身为：探测架构 → 下载对应二进制到 `/opt/free-proxy/` → `free-proxy install-deps` → 写单元 → 启动。相比当前 Python 版省去了装 uv、`uv sync`、建 venv 的全部步骤。

> 关于配置变更后自重启：当前 Python 版用 `os._exit(0)` 配 `Restart=on-failure` 存在语义冲突（退出码 0 不触发重启）。Go 版应改为**退出码非 0** 触发 systemd 重启，或用 `systemctl` 明确重启，避免同样的坑。

---

## 17. 测试策略

对齐并尽量超过当前 42 个测试的覆盖：

- **单元测试**（`*_test.go`）：VPNGate 解析/去重、上游 URL 解析、scrypt 互通（用 Python 生成的哈希做黄金用例验证兼容）、OpenVPN 命令构造（版本分支）、日志失败分类、策略路由命令拼接（注入假 `CommandRunner`）、代理池排序、config 校验。
- **集成测试**：SQLite 用临时文件 DB，跑 goose up/down 往返；API 用 `httptest.NewServer` + Echo，覆盖登录/登出/secret-path 404/401、Job 提交与轮询、日志过滤/导出。
- **代理端到端**（对齐现状 4 个）：本地起 `Gateway`，用回环 TCP 目标验证 SOCKS5 转发、HTTP CONNECT、缺认证被拒、unified 单端口分流。绑 tun0 的部分用可注入的 `Dialer` 在非 Linux/非特权环境下跳过。
- **特权验收**：保留 `tests/privileged/verify_linux.sh`（真机 root + TUN + openvpn），几乎无需改动。
- 工具：标准 `testing` + `testify/require`；mock 用 sqlc 生成的 `Querier` 接口。

---

## 18. 分阶段实施路线图

建议按依赖自底向上、每阶段可编译可测：

| 阶段 | 内容 | 产出 |
|---|---|---|
| **P0 脚手架** | go.mod、目录、config、domain 枚举、slog、cobra 空壳 | `free-proxy --help` 可跑 |
| **P1 数据层** | 迁移 0001/0002、sqlc、repo、`database-upgrade`/`doctor` | DB 建表 + 依赖检测可用 |
| **P2 代理网关** | proxy/*（unified/socks5/http/connector/relay/dns）| 端到端代理测试通过（可先直连不绑 tun） |
| **P3 隧道+网络** | tunnel/*、netx/*、providers/vpngate、ipinfo | `discover`、真机建隧道+策略路由 |
| **P4 服务层** | 13 个 service + Job 系统 + 后台循环 | 拉取/测速/激活/自动切换闭环 |
| **P5 API+安全** | Echo 路由、secret-path、session、scrypt、26 端点 | API 契约对齐，旧前端可直连联调 |
| **P6 前端 embed** | Vite outDir、embed、SPA 回退 | 单二进制含前端 |
| **P7 打包部署** | Makefile、交叉编译、瘦 install.sh、systemd/openrc | 发布物 |

P5 完成即可用现有 React 前端（改 API base 到 secret-path）联调，验证契约无回归后再做 P6。

---

## 19. 风险与注意事项

1. **root/CAP 依赖**：`SO_BINDTODEVICE`、`ip rule/route`、`sysctl`、创建 tun 都需 root（或 `CAP_NET_ADMIN`/`CAP_NET_RAW`）。Go 版行为与现状一致，非特权环境只能跑 API/前端，跑不了真实出口。
2. **SQLite 并发写**：Go 多 goroutine 并发写需 `WAL` + `busy_timeout`，或写连接 `SetMaxOpenConns(1)` 串行化。这是相对 aiosqlite 单连接模型需要显式处理的点。
3. **Echo v5 成熟度**：若 v5 未 GA，锁 v4，架构不变（§2.3）。
4. **DNS 方案取舍**：纯 Go resolver（`PreferGo`）零依赖但行为与手写报文略有差异；若需与现状逐字节一致，引入 `miekg/dns`。先用前者，按需升级。
5. **前端构建是 embed 前置**：`go build` 前必须先 `bun run build`，CI 顺序不能反；仓库放占位 `dist/index.html` 防止裸 `go build` 失败。
6. **scrypt 内存**：`n=16384,r=8,p=1` 约 16MB/次，登录属低频，无碍；但别在无节流的登录接口被刷（当前也无登录限流，可顺带补一个简单限流中间件）。
7. **时间与时区**：DB 统一存 UTC RFC3339，Go 侧 `time.Time` 显式 `.UTC()`，避免与 Python timezone-aware 值不一致。
8. **自重启语义**：见 §16，务必用非 0 退出码触发 systemd 重启，别重蹈 `os._exit(0)` 的坑。

---

## 20. 附：Python 模块 → Go 包对照

| Python | Go |
|---|---|
| `config.py` | `internal/config/config.go` |
| `domain/{enums,models,countries,exceptions}.py` | `internal/domain/*.go` |
| `security.py` | `internal/security/security.go` |
| `middleware.py` | `internal/api/middleware.go` |
| `main.py` / `lifespan.py` | `cmd/free-proxy/main.go` + `internal/api/server.go` |
| `cli.py` | `cmd/free-proxy/main.go`（cobra 子命令） |
| `logging.py` | `internal/logging/logging.go` |
| `infrastructure/database/{models,repositories,connection}.py` | `internal/store/{migrations,queries,gen,repo.go,db.go}` |
| `infrastructure/tunnel/{openvpn,process,log_parser,command}.py` | `internal/tunnel/*.go` |
| `infrastructure/network/{routing,latency,upstream,commands,tun}.py` | `internal/netx/*.go` |
| `infrastructure/ipinfo/client.py` | `internal/ipinfo/client.go` |
| `proxy/{unified,socks5,http,connector,relay,dns,gateway}.py` | `internal/proxy/*.go` |
| `providers/vpngate/{client,parser}.py` | `internal/providers/vpngate/*.go` |
| `services/*.py`（13 个） | `internal/services/*.go`（13 个） |
| `api/dependencies.py` + `api/routers/*.py` | `internal/api/deps.go` + `internal/api/handlers/*.go` |
| `install.sh`（依赖安装部分） | `internal/platform/*.go`（`doctor`/`install-deps`） |
| `frontend/`（React） | 不变，构建产物 embed 至 `internal/web/dist` |
| `alembic/` | `internal/store/migrations/`（goose） |
| `deploy/` | `deploy/`（简化，见 §16） |

---

*本文对齐的源码快照：`free-proxy` 分层实现（8 表 / 26 端点 / 13 服务 / 7 代理模块）。实现时以各 §的行为约定为准，对外契约（API、env、代理端口、哈希格式、数据目录）保持逐项不变，即可与现有前端和客户端无缝替换。*
