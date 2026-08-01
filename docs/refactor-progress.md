# Go 重构进度追踪

> 此文件用于 `/loop` 跨迭代记录进度。每次迭代开始先读它，结束时更新它。
> 蓝图见 `docs/refactor-go.md`。参考项目 `aimili-vpngate/` 不改动。

## 环境
- Go 1.26.4, bun 1.3.14 已就绪。
- sqlc v1.27.0 预编译二进制已装到 `$(go env GOPATH)/bin/sqlc`（从源码编译在本机 macOS 因 CGO `strchrnul` 冲突失败）。
- 模块路径：`github.com/masteralanlab/free-proxy`。

## 用户决策（2026-07-26）
- **前端：重建为完整精致 UI**（不是只 embed 原版）。P6 用 React 19 + Tailwind v4 重写成结构清晰、可维护的管理台：节点表格/筛选/分页、网关状态面板、策略配置、系统诊断、实时日志等完整页面，再 embed。视觉可参考 aimili 暗黑玻璃拟物风但用现代组件化实现。
- **循环：继续自动跑完** P2→P7 全部阶段，每阶段保持 `go build ./... && go vet ./...` 绿并尽量写测试。
- 跨平台注意：`SO_BINDTODEVICE` 仅 Linux；用 build tag 分 `bind_linux.go`/`bind_other.go`，darwin 上 no-op 以便本机编译与端到端代理测试。

## 阶段状态
- [x] **P0 脚手架** — go.mod、config、domain(enums+models)、logging(slog+store)、cobra 骨架（11 子命令，`--help` 可跑，`go build ./...` 绿）。
- [x] **P1 数据层** — 迁移 0001/0002（goose）、sqlc.yaml、5 个 queries 文件、`sqlc generate`→`gen/`、db.go(Open+WAL+Migrate+SchemaTables)、repo.go(Repos + Node/Settings/Job/ProbeResult/IPCache 5 个仓储)。`database-upgrade`/`status` 已接通并**冒烟通过**：迁移建 8 表 + goose_db_version，重复执行幂等。
- [x] **P2 代理网关** — proxy/ 9 文件（gateway 生命周期、unified 首字节分发 `Peek(1)`、socks5 RFC1928/1929、http CONNECT+转发+407、connector `SO_BINDTODEVICE`(build-tag linux/other)、dns tun 解析、relay 双向 64KB）。**5 个端到端测试全过**（SOCKS5 转发/认证、HTTP 转发/CONNECT/407）。build+vet+test 绿。用了 `golang.org/x/sys` 和测试用 `golang.org/x/net/proxy`。
- [x] **P3 隧道+网络+数据源** — netx(commands/tun/latency/routing)、tunnel(logparse/command/process/openvpn，os.Pipe+Setpgid 进程组管理)、ipinfo(批量分类)、providers/vpngate(parser CSV+去重/client 回退链)。**单元测试全过**（parser、logparse 分类、BuildArgs 版本分支、tun 分配）。build+vet+test 绿。约 1900 行。
- [x] **P4 服务层**（完成，~2135 行）
  - [x] **P4a** — repo 扩展、operations(Coordinator)、jobs、discovery、pool、domain(errors/countries)、`discover` CLI。测试全过。
  - [x] **P4b** — ipinfo(缓存富化)、probe(并发+TunAllocator+信号量)、gateway(激活/断开/状态/意外退出)、autoswitch(排除+黑名单退避)、settings(enforce active node)、health(SOCKS 出口检查+恢复)+HealthMonitor、active_latency Monitor、maintenance(discover→probe→autoconnect)+MaintenanceMonitor、diagnostics(系统检查+DNS 修复+provider 失败诊断)、netx.HealthChecker(SOCKS5 握手测出口 IP)、repo(ClearExpiredBlacklist/PurgeStaleNodes)、接通 `preflight` CLI。测试：diagnostics 辅助函数。build+vet+test 绿。
- [x] **P5 API+安全** — security(scrypt 与 RFC7914 向量校验、AdminConfigStore 原子写+迁移+一次性密码、SessionManager、AuthService)、api(deps 容器、secret-path Echo `Pre` 中间件、26 端点 handlers、structValidator、HTTPErrorHandler 映射 domain 错误、SPA 占位)、serve 全装配(logging→db→migrate→repos→security→13 services+monitors→proxy→echo，SIGTERM graceful)、接通 credentials/admin-config/logs CLI。**运行时冒烟全过**：404/200/401、scrypt 登录、gateway/pool/settings JSON、discover 202+Job。用 Echo v5.3.1(Context 为指针结构体)。CGO_ENABLED=0 静态二进制 22MB。
- [x] **P6 前端重建 + embed** — 用 React 19 + Vite + Tailwind v4 + Zustand 重写为完整暗黑玻璃拟物风管理台：登录页、节点表格(筛选/分页/激活/探测/收藏/下载 config)、网关面板(检测/切换/断开)、策略配置、系统诊断+运行状态、实时日志(过滤/导出)、统计卡片、Toast、8s 轮询、secret-path 相对 API base、401→登录。Vite outDir→`internal/web/dist`；`internal/web/embed.go` `//go:embed all:dist`；`api/frontend.go` 服务 embed + SPA 回退。**冒烟通过**：单二进制(23MB)直出真实 SPA、资源正确 content-type、未知路由回 index。前端源 ~13 文件。
- [x] **P7 打包部署 + 收尾** — `internal/platform`(doctor 依赖检测 + install-deps 按 apt/apk/dnf/yum 安装 openvpn/iproute2/procps，含单测)、接通 `doctor`/`install-deps` CLI；`Makefile`(frontend→build/build-go/cross/test/vet)；瘦身 `install.sh`(下载/复制二进制→install-deps→内联 systemd/openrc 单元→启动)；简化 `deploy/*.service|.openrc`(ExecStartPre=doctor)；重写 `README.md` 为 Go 版；`.gitignore` 修正(`/dist/` 根级，保留 embed index.html)；**移除 Python**：src/、alembic/、alembic.ini、pyproject.toml、uv.lock、tests/{unit,integration}（保留 tests/privileged/verify_linux.sh、aimili-vpngate/）。`make build` 产出 15.8MB 静态二进制；最终 serve 冒烟全过（SPA/资源/401/登录/system status）。

## ✅ 重构完成（P0–P7 全部达成）
- 单一 `CGO_ENABLED=0` 静态二进制，内嵌前端 + 迁移；`make build`/`make cross` 交叉编译 linux amd64/arm64。
- 对外契约保持：`/{secret}/api/v1` 26 端点、`FREE_PROXY_` env、9527 单端口 SOCKS5/HTTP、scrypt 哈希格式、data_dir 布局。
- 依赖自检安装内置（doctor/install-deps）。参考项目 `aimili-vpngate/` 全程未改动。
- 校验：`go build ./... && go vet ./... && go test ./...` 全绿；serve 运行时冒烟通过。

## 已建文件
- `go.mod`
- `internal/config/config.go`
- `internal/domain/enums.go`, `internal/domain/models.go`
- `internal/logging/logging.go`
- `cmd/free-proxy/main.go`, `cmd/free-proxy/commands.go`（子命令目前为 P0 占位，后续阶段填实）

## 下一步（P7 打包 + 收尾）—— 最后阶段
1. `Makefile`：`frontend`(cd frontend && bun install && bun run build)、`gen`(sqlc generate)、`build`(CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=…")、`test`、`all`；`cross`(linux amd64+arm64)。
2. 提交占位 `internal/web/dist/index.html`（简单占位，非 hash 版）保证裸 `go build` 不报错；真实构建由 Makefile 保证。
3. 瘦身 `install.sh`：探测架构→下载/复制二进制到 /opt/free-proxy→`free-proxy install-deps`→写 systemd/openrc 单元→启动。去掉 uv/venv/uv sync。
4. 简化 `deploy/free-proxy.service` 与 `.openrc`：`ExecStartPre=free-proxy doctor`、`ExecStart=free-proxy serve`、`Restart=on-failure`、`EnvironmentFile=-/etc/free-proxy/free-proxy.env`；更新 `deploy/free-proxy.env.example`。
5. 实现 `platform`(doctor/install-deps) 若尚未：`internal/platform/{detect,pkgmanager,install}.go`，接通 `doctor`/`install-deps` CLI（当前是占位桩）。**注意：P0-P6 未做 platform 包，doctor/install-deps 仍是 errPending 桩——P7 必须补。**
6. 移除 Python：`src/`、`alembic/`、`alembic.ini`、`pyproject.toml`、`uv.lock`、`tests/` 中 py、`src/free_proxy/web`（旧前端）。更新 `README.md` 为 Go 版说明。保留 `aimili-vpngate/` 不动。
7. 全量校验：`go build ./... && go vet ./... && go test ./...` 绿；`make build` 产出单二进制；最终 serve 冒烟。

## 备注
- 校验命令：`go build ./... && go vet ./...`。
- 每阶段完成后更新本文件的勾选与"下一步"。
