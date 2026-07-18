# Free Proxy

Free Proxy 是一个自托管的免费代理出口聚合工具。它从公开 Provider 获取候选出口，执行
真实连通性探测，识别住宅、移动和机房 IP，并统一提供本地 SOCKS5 与 HTTP 代理。

默认 SOCKS5 和 HTTP 共用 `127.0.0.1:9527`。普通系统流量不会切换到免费出口，只有
显式使用统一代理端口的应用流量会绑定到 `tun0`。

住宅 IP 类型来自第三方分类数据，并非绝对保证，应以当前探测结果为准。

## 工作流程

```text
公开节点源
    -> 节点发现和配置解析
    -> OpenVPN 握手与延迟探测
    -> IP 地理和网络类型分类
    -> 策略筛选与可用代理池
    -> 激活 tun0 出口
    -> 本地 SOCKS5 / HTTP 网关
```

## 核心能力

- VPNGate 节点发现、去重和 SQLite 持久化。
- HTTPS 校验回退、HTTP 回退及 HTTP/SOCKS5 上游代理。
- 并发 OpenVPN 探测、冷却黑名单和周期维护。
- 延迟优先的 `auto`、显式 `residential_first`、国家、固定节点、收藏和 IP 类型筛选。
- 出口健康检查、活动延迟刷新和失败自动轮换。
- SOCKS5、HTTP CONNECT、代理认证、并发限制和空闲超时。
- Web 管理页面、REST API、CLI、JSON 日志和探测历史。
- 管理密码 `scrypt` 哈希、随机安全路径和会话认证。
- Alembic 数据库迁移、systemd/OpenRC 部署。

## 系统要求

- Linux VPS，Python 3.11 或更高版本。
- root 权限。
- `/dev/net/tun` 可用。
- OpenVPN、`iproute2` 和 `sysctl`。
- systemd 或 OpenRC。

macOS 和非特权环境可以运行单元测试与 Web/API 开发，但无法完成真实 TUN、策略路由和
出口代理验收。

## 快速安装

在 Linux VPS 上以 root 执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

从本地仓库安装：

```bash
sudo ./install.sh install
```

安装器会执行：

```text
安装系统依赖
-> 安装 uv
-> uv sync --frozen --no-dev
-> free-proxy database-upgrade
-> 安装并启动 free-proxy.service 或 OpenRC 服务
```

查看首次管理地址和一次性密码：

```bash
sudo free-proxy-manage credentials
```

首次成功登录后，一次性明文密码文件会自动删除，长期配置只保存 `scrypt` 哈希。

## 开发环境

```bash
uv sync --dev
uv run free-proxy database-upgrade
uv run free-proxy serve --reload
```

管理台源码位于 `frontend/`，使用 React、Vite、Tailwind CSS v4 和 Zustand。修改页面后运行
`bun install && bun run build`，构建产物会写入 `src/free_proxy/web/dist`。

默认开发管理地址为 `http://127.0.0.1:8787/<安全路径>/`。运行以下命令查看实际地址：

```bash
uv run free-proxy credentials
```

## 获取第一个出口

1. 登录 Web 管理页面。
2. 点击“更新并检测节点”。
3. 等待节点发现、快速测试和首次自动连接完成。
4. 在状态面板确认当前出口、活动延迟和出口 IP。
5. 将本机应用配置为使用统一代理端口 `127.0.0.1:9527`。

也可以通过 API 触发：

```bash
curl -X POST http://127.0.0.1:8787/<安全路径>/api/v1/proxies/refresh
curl http://127.0.0.1:8787/<安全路径>/api/v1/gateway/status
```

启用管理认证时，API 调用需要先登录并携带会话 Cookie。

## 使用代理

SOCKS5：

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org
```

HTTP/HTTPS：

```bash
curl --proxy http://127.0.0.1:9527 https://api.ipify.org
```

Python `httpx`：

```python
import httpx

with httpx.Client(proxy="socks5://127.0.0.1:9527") as client:
    print(client.get("https://api.ipify.org").text)
```

如配置了代理用户名和密码：

```text
socks5://username:password@127.0.0.1:9527
http://username:password@127.0.0.1:9527
```

## 筛选与轮换

支持以下路由模式：

- `auto`：按实际延迟升序、Provider score 降序选择可用节点，不隐式优先住宅 IP。
- `residential_first`：在延迟和 Provider score 排序前优先住宅/移动 IP。
- `country`：只选择指定国家或地区。
- `fixed`：锁定指定节点，失败后只重连原节点。
- `favorites`：只从收藏节点中选择。

IP 类型筛选支持 `all`、`residential` 和 `hosting`。`residential` 同时包含住宅和移动网络。

手动轮换：

```bash
curl -X POST http://127.0.0.1:8787/<安全路径>/api/v1/gateway/rotate
```

代理池统计：

```bash
curl http://127.0.0.1:8787/<安全路径>/api/v1/pool/statistics
```

## 配置

生产环境配置文件默认为 `/etc/free-proxy/free-proxy.env`，环境变量统一使用
`FREE_PROXY_` 前缀。

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
FREE_PROXY_DNS_REPAIR_SERVERS=1.1.1.1,8.8.8.8
```

HTTP/SOCKS5 上游代理示例：

```text
FREE_PROXY_UPSTREAM_PROXY_URL=socks5://user:password@127.0.0.1:1080
```

新安装默认 Web 和代理监听保持在 `127.0.0.1`，代理默认端口为 `9527`。管理密码使用 `scrypt` 哈希。
HTTP 和 SOCKS5 通过首字节自动识别，共用同一个监听端口。

自动 DNS 修复只支持使用 `resolvectl` 的 Linux 系统，默认关闭。启用后，节点发现遇到
明确 DNS 解析错误时会修复默认接口 DNS 并重试一次。

## 管理命令

应用 CLI：

```bash
free-proxy status
free-proxy credentials
free-proxy logs --lines 200
free-proxy discover
free-proxy database-upgrade
free-proxy admin-config --username admin --password 'new-password'
```

生产服务管理：

```bash
sudo free-proxy-manage status
sudo free-proxy-manage start
sudo free-proxy-manage stop
sudo free-proxy-manage restart
sudo free-proxy-manage logs
sudo free-proxy-manage update
sudo free-proxy-manage uninstall
```

默认卸载保留 `/var/lib/free-proxy`。同时删除数据：

```bash
sudo PURGE_DATA=1 free-proxy-manage uninstall
```

## API 摘要

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/proxies
POST   /api/v1/proxies/discover
POST   /api/v1/proxies/refresh
POST   /api/v1/proxies/probe
POST   /api/v1/proxies/{id}/probe
GET    /api/v1/proxies/{id}/probes
POST   /api/v1/proxies/{id}/activate
POST   /api/v1/proxies/{id}/favorite
GET    /api/v1/proxies/{id}/config
GET    /api/v1/gateway/status
POST   /api/v1/gateway/check
POST   /api/v1/gateway/rotate
DELETE /api/v1/gateway/current
GET    /api/v1/pool/statistics
GET    /api/v1/jobs/{id}
GET    /api/v1/settings
PUT    /api/v1/settings
GET    /api/v1/system/status
GET    /api/v1/system/diagnostics
POST   /api/v1/system/dns/repair
GET    /api/v1/logs
GET    /api/v1/logs/export
```

## 安全建议

- 保持 SOCKS5 和 HTTP 代理监听在 `127.0.0.1`，不要直接暴露到公网。
- 远程使用代理时优先通过 SSH 隧道或私有网络访问。
- Web 管理端口需要防火墙和安全组限制来源地址。
- 首次登录后立即修改管理账号和密码。
- 对公网开放代理前必须配置 `FREE_PROXY_PROXY_USERNAME` 和
  `FREE_PROXY_PROXY_PASSWORD`。
- DNS 自动修复和服务进程需要 root 权限，只在受控服务器启用。

## 故障排查

### 无法创建 TUN

确认 `/dev/net/tun` 存在，VPS 控制面板已启用 TUN/TAP，并以 root 运行服务。

```bash
ls -l /dev/net/tun
```

### OpenVPN 不可用

```bash
openvpn --version
free-proxy status
```

### 节点源 DNS 失败

先检查系统诊断：

```bash
curl http://127.0.0.1:8787/<安全路径>/api/v1/system/diagnostics
```

确认系统使用 systemd-resolved 后，可设置：

```text
FREE_PROXY_DNS_REPAIR_ENABLED=true
```

### 隧道已连接但代理无流量

检查 `tun0`、策略路由表和反向路径过滤：

```bash
ip address show tun0
ip rule show
ip route show table 100
sysctl net.ipv4.conf.all.rp_filter
```

### 查看日志

```bash
sudo free-proxy-manage logs
free-proxy logs --lines 200
```

## 测试

```bash
uv run pytest
uv run ruff check .
uv run mypy src/free_proxy
```

Linux 特权验收脚本位于 `tests/privileged/`，需要在具备 root、TUN 和 OpenVPN 的 Linux
VPS 上运行。

## License

本项目使用 [LICENSE](LICENSE) 中声明的许可证。
