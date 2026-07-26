# Free Proxy

Free Proxy 是一个自托管的免费代理出口聚合工具。它从公开 Provider（当前为 VPNGate）获取候选出口，执行真实
连通性探测，识别住宅、移动和机房 IP，并统一提供本地 SOCKS5 与 HTTP 代理。

本项目为 **Go 单二进制** 实现：一个 `CGO_ENABLED=0` 的静态可执行文件内嵌前端与数据库迁移，落地即可运行。
安装的全部环节——OpenVPN、iproute2 等系统依赖、环境文件、systemd/OpenRC 服务——都由二进制自身的
`install` / `doctor` / `install-deps` 子命令处理，不依赖外部安装脚本。

默认 SOCKS5 和 HTTP 共用 `127.0.0.1:9527`，通过首字节自动识别协议。只有显式使用该端口的应用流量会绑定到 `tun0` 出口。

## 技术栈

- **Go 1.23+**，Echo v5（Web/API）、sqlc + `modernc.org/sqlite`（纯 Go，无 CGO）、goose（内嵌迁移）、cobra（CLI）、log/slog（日志）。
- 前端 **React 19 + Vite + Tailwind v4 + Zustand**，构建产物经 `//go:embed` 内嵌进二进制。
- 密码 `scrypt` 哈希，随机安全路径 + 会话 Cookie 鉴权。

## 系统要求

- Linux VPS，root 权限，`/dev/net/tun` 可用，OpenVPN、`iproute2`、`sysctl`。
- macOS / 非特权环境可编译、跑测试、开发 API/前端，但无法完成真实 TUN、策略路由和出口代理。

## 快速安装

在 Linux VPS 上以 root 执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

`install.sh` 只做一件事：下载对应架构的二进制到 `/usr/local/bin/free-proxy`，然后执行 `free-proxy install`。
其余全部由二进制完成：安装系统依赖（openvpn/iproute2/procps）→ 写入 `/etc/free-proxy/free-proxy.env` →
注册并启动 systemd/OpenRC 服务。

也可以跳过脚本，手动下载二进制后执行：

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

查看首次管理地址与一次性密码：

```bash
free-proxy credentials
```

更新：重新执行上面任一安装命令即可（配置与数据保留）。卸载：`free-proxy uninstall`（加 `--purge-data` 同时删除数据）。

## 从源码构建

需要 Go 1.23+ 与 bun。

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

构建顺序：`bun run build`（前端 → `internal/web/dist`）→ `go build`（内嵌前端）。因为无 CGO，可在 macOS 上直接产出 Linux 二进制。
本地构建产物同样可以直接部署：把二进制拷到目标机器后执行 `sudo ./free-proxy install`。

## 发布 Release

`install.sh` 从 GitHub Releases 下载 `free-proxy-linux-amd64` / `free-proxy-linux-arm64`，这两个资产由
`.github/workflows/release.yml` 在**推送版本标签**时自动构建并发布（内部即 `make cross`，版本号取自标签）：

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action：构建前端 + 交叉编译 → 发布 Release（含 SHA256SUMS）
```

标签需以 `v` 开头（如 `v1.0.0`）。发布完成后，`install.sh` 的 `latest` 下载即可命中该二进制。未打过任何标签前，
安装脚本会因 Releases 为空而下载失败——首次使用务必先打一个版本标签。

开发时：

```bash
cd frontend && bun install && bun run dev   # 前端热更新（配合下方 serve）
go run ./cmd/free-proxy serve                # 后端（首次会生成随机管理地址与密码）
go run ./cmd/free-proxy credentials          # 查看管理地址与一次性密码
```

## 使用代理

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

如配置了 `FREE_PROXY_PROXY_USERNAME` / `FREE_PROXY_PROXY_PASSWORD`：`socks5://user:pass@127.0.0.1:9527`。

## 获取第一个出口

1. 登录 Web 管理页面（`http://<host>:8787/<安全路径>/`）。
2. 点击“更新并检测节点”，等待发现、测速与首次自动连接。
3. 在网关面板确认当前出口、活动延迟与出口 IP。
4. 将本机应用配置为使用 `127.0.0.1:9527`。

## CLI

```bash
free-proxy serve                 # 运行控制台 + 代理网关 + 后台任务
free-proxy install               # 一键安装：二进制 + 依赖 + 环境文件 + 服务（需 root）
free-proxy uninstall             # 卸载服务与二进制，--purge-data 同时删数据（需 root）
free-proxy credentials           # 打印管理地址与一次性密码
free-proxy discover              # 拉取并存储节点
free-proxy status                # 打印配置与数据库表
free-proxy preflight             # 启动前环境检查
free-proxy doctor [--fix]        # 检查（并可安装）系统依赖
free-proxy install-deps          # 仅安装 openvpn / iproute2 / procps（需 root）
free-proxy database-upgrade      # 执行数据库迁移
free-proxy admin-config ...      # 修改管理凭据与监听
free-proxy logs --lines 200      # 打印最近日志
```

## 配置

生产环境配置文件默认 `/etc/free-proxy/free-proxy.env`（由 `free-proxy install` 生成），环境变量统一
`FREE_PROXY_` 前缀。所有子命令都会自动读取该文件（进程环境变量优先；路径可用 `FREE_PROXY_ENV_FILE` 覆盖），
因此 CLI 与服务看到的配置始终一致。常用项：

```text
FREE_PROXY_DATA_DIR=/var/lib/free-proxy
FREE_PROXY_WEB_HOST=127.0.0.1
FREE_PROXY_WEB_PORT=8787
FREE_PROXY_PROXY_HOST=127.0.0.1
FREE_PROXY_PROXY_PORT=9527
FREE_PROXY_PROXY_ENABLED=true
FREE_PROXY_PROXY_USERNAME=
FREE_PROXY_PROXY_PASSWORD=
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_TUNNEL_INTERFACE=tun0
FREE_PROXY_UPSTREAM_PROXY_URL=
FREE_PROXY_DNS_REPAIR_ENABLED=false
```

## API 摘要

所有端点在安全路径前缀下：`/{secret_path}/api/v1/...`。长耗时操作返回 `202 + Job`，通过 `GET /jobs/{id}` 轮询。

```text
POST   /api/v1/auth/login        POST /api/v1/auth/logout
GET    /api/v1/auth/config       PUT  /api/v1/auth/credentials
GET    /api/v1/proxies           POST /api/v1/proxies/discover|refresh|probe
POST   /api/v1/proxies/{id}/probe|activate|favorite
GET    /api/v1/proxies/{id}/probes|config
GET    /api/v1/gateway/status    POST /api/v1/gateway/check|rotate    DELETE /api/v1/gateway/current
GET    /api/v1/pool/statistics   GET  /api/v1/jobs/{id}
GET    /api/v1/settings          PUT  /api/v1/settings
GET    /api/v1/system/status|diagnostics   POST /api/v1/system/dns/repair
GET    /api/v1/logs              GET  /api/v1/logs/export
```

## 项目结构

```text
cmd/free-proxy      # 入口 + cobra 子命令 + serve 装配
internal/
  config domain logging security store        # 基础层
  proxy tunnel netx providers ipinfo          # 代理/隧道/网络/数据源
  services                                    # 13 个用例服务 + 后台监控
  api web                                     # Echo 服务 + 内嵌前端
frontend/           # React 源码（构建到 internal/web/dist）
install.sh          # 引导脚本：仅下载二进制并执行 free-proxy install
docs/               # 重构文档
```

## 安全建议

- SOCKS5/HTTP 代理与 Web 管理默认监听 `127.0.0.1`，不要直接暴露公网；远程优先 SSH 隧道。
- 首次登录后立即修改管理账号密码；对公网开放代理前必须配置代理认证。
- 隧道、策略路由与依赖安装需 root，只在受控服务器启用。

## 测试

```bash
go test ./...
go vet ./...
```

Linux 特权验收脚本位于 `tests/privileged/verify_linux.sh`，需在具备 root、TUN 和 OpenVPN 的 Linux VPS 上运行。

## License

见 [LICENSE](LICENSE)。
