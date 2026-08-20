**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Spin Up Your Own Free Proxy Pool With One Command

> Run **a single command** on an overseas VPS and it automatically pulls hundreds of free exit nodes from public sources (VPNGate), runs real speed tests, intelligently picks the fastest routes, and exposes a stable **SOCKS5 / HTTP proxy**. When a node drops, it switches automatically — no babysitting required.

> 🎥 Demo video: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="One-command deploy" src="https://img.shields.io/badge/Deploy-One%20Command-brightgreen">
  <img alt="Go single binary" src="https://img.shields.io/badge/Go-Single%20Binary·Zero%20Deps-00ADD8">
  <img alt="Free" src="https://img.shields.io/badge/Nodes-Free·Auto%20Speed%20Test-orange">
</p>

**Who is it for?**

- Anyone who wants a **proxy exit of their own, fully under their control**, instead of handing traffic to someone else's commercial service.
- Anyone who has (or plans to buy) an overseas VPS and wants to turn it into a fully automated proxy gateway.
- Anyone who doesn't want to fuss with complex configuration — **one command to install, a few clicks in the web UI to use**.

---

## ✨ Highlights

- 🔌 **One-command deploy**: dependencies, service, and auto-start on boot are all handled automatically — even beginners can do it.
- 🌍 **Automatic discovery + real speed tests**: pulls hundreds of nodes from public sources, measures connectivity and latency for real, and picks the fastest automatically.
- ♻️ **Auto-switch on drop**: free nodes flaky? It reconnects and switches in the background to keep the proxy always online.
- 🧩 **SOCKS5 / HTTP on one port**: a single port `9527` handles both, auto-detecting the protocol from the first byte.
- 🖥️ **Clean web dashboard**: node pool, gateway status, logs, and policy all on one screen.
- 📦 **Single file, zero dependencies**: one static binary with the frontend and database embedded — drop it in and it runs.

---

## 🛒 Before You Start: Get These Two Things Ready (Read This If You're New)

### 1️⃣ An Overseas Linux VPS

This tool needs to run on an **overseas Linux server** (with root and TUN support). For newcomers, the two providers below are recommended — both accept **Alipay** and work right out of the box:

| Recommendation | Best for | Highlights | Link |
|---|---|---|---|
| **BandwagonHost** | 🔰 Beginners / value | Long-established and stable, wallet-friendly pricing, supports Alipay, optional premium CN2 GIA routes, ready to use out of the box | **[Buy now 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Speed / premium | Top-tier three-network optimization / CN2 GIA routes, low latency and high speed, a maxed-out experience | **[Buy now 👉](https://cutt.ly/YywJIzY0)** |

> 💡 On a budget and want a hassle-free setup → pick **[BandwagonHost](https://cutt.ly/qywJNWzd)**; want ultimate speed and route quality → pick **[DMIT](https://cutt.ly/YywJIzY0)**.
> Choose **Ubuntu / Debian** as the OS (this guide uses it as the example), and pick a KVM plan (TUN supported by default).

### 2️⃣ A Card You Can Pay With

Most overseas VPS providers require a credit card / PayPal. **No overseas credit card?** You can open an **overseas virtual credit card** in a few minutes and easily subscribe to all kinds of overseas services (VPS, ChatGPT, streaming, subscription software, etc.):

> 💳 **[Overseas Virtual Credit Card · Fast Sign-Up 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Deploy in Three Steps (Truly Beginner-Friendly)

Assuming you've already bought a VPS and have its **server IP** and **root password**.

**Step 1 · SSH into your VPS**

```bash
ssh root@你的服务器IP
```

**Step 2 · Install with one command**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

The script automatically: downloads the build for your architecture → installs system dependencies (openvpn, etc.) → registers an auto-start service → launches it. Just wait for it to finish — no interaction needed.

**Step 3 · Note down the admin URL and login credentials**

After the first install, the script **prints directly** the randomly generated path, username, and password:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 The path, username, and password are randomly generated only on the **first install**, with no defaults. Save them immediately because the password cannot be recovered later.
> 🔒 Re-running the installer for an update preserves the existing path, username, and password. To change them explicitly, use the dashboard or run `free-proxy install --rotate-admin`.

✅ **Done!** The service is already fetching nodes, running speed tests, and connecting in the background. Next, let's see how to use it.

---

## 🌐 How to Use the Proxy / Access the Web Dashboard

The service listens on `0.0.0.0` by default and includes a built-in **"external access" toggle** (switch it anytime on the "Policy" page in the dashboard — **effective instantly, no restart needed**). Local access and SSH tunnels **always work**, regardless of the toggle.

### Web dashboard: external access allowed by default ✅

Protected by both login and a random secret path, so you can open it from the internet right after install. Visit the address printed by `free-proxy credentials` in your browser:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

If you don't need public access, you can turn off its external-access toggle in the dashboard, or use an SSH tunnel instead (see below).

### Proxy port: local-only by default 🔒

To avoid becoming an **"open proxy"** anyone can use, the proxy serves only the local machine by default. To use it from the internet, two steps:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **Enable it in the dashboard**: go to "Policy → External access", check "Allow external access to the proxy port", and save.

After that you can use it in apps on your local machine: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 The most conservative approach (no public access at all): turn off external access to the web dashboard in the settings and use an SSH tunnel instead —
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`, then access `127.0.0.1` locally.

### Verify the proxy is working

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # should return the "VPN exit IP", not your VPS's own IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

If you see an IP different from your VPS, the proxy is already forwarding through the VPN exit 🎉

---

## 🖱️ How to Use the Web Dashboard

1. Open the admin URL and log in with your username and password.
2. Click **"Update and check nodes"** and wait a moment while it discovers, speed-tests, and automatically connects to the fastest node.
3. The **"Gateway"** panel shows the current exit node, latency, and exit IP.
4. Point your local app's proxy at `127.0.0.1:9527` and you're good to go.

---

## 🔧 Common Commands

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**Update to the latest version**: just run the "one-command install" above again. Node data, settings, admin path, username, and password are all preserved.

---

## ❓ FAQ

- **Can't connect / no nodes for now?** Free nodes (VPNGate) fluctuate by nature; the service retries and switches automatically. Wait a bit, or click "Update and check nodes" once in the dashboard.
- **It says root / TUN is required?** Run it as root and make sure your VPS has TUN/TAP enabled. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** are both KVM-based, support it by default, and work out of the box.
- **My VPS is ARM architecture?** No worries — the install script automatically detects amd64 / arm64.
- **Can I run it on my own computer (macOS/Windows)?** You can build and develop there, but the real tunnel and exit proxy require Linux + root + TUN, so deploy to a VPS.

---

## 🧰 Recommended Tools and Resources

- 🔎 **The most powerful Telegram search bot** — a magic tool for finding movies, software, e-books, and all kinds of resources, with instant results: 👉 **[Open it](https://cutt.ly/2yeh3GOE)**
- 🖥️ Don't have a server yet? **[BandwagonHost (best value for beginners)](https://cutt.ly/qywJNWzd)** · **[DMIT (premium routes)](https://cutt.ly/YywJIzY0)**
- 💳 No overseas card to pay with? **[Overseas Virtual Credit Card](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Security Recommendations

- The service listens on `0.0.0.0` by default, and exposure is controlled by the dashboard's "external access" toggle: **the web dashboard is open by default** (protected by login + secret path), while **the proxy is local-only by default**. When you don't need public access, turn off external access to the web dashboard and use an SSH tunnel instead.
- **You must set a proxy username and password before enabling external proxy access**, otherwise it becomes an "open proxy" anyone can use — highly prone to abuse and likely to get your VPS banned. For this reason, the system rejects all external proxy requests when no password is set.
- Change the admin username and password as soon as possible after your first login. Tunnels, policy routing, and dependency installation require root, so only enable them on servers you fully control yourself.

---

## 🧑‍💻 Advanced Usage (For Developers)

<details>
<summary>Click to expand: CLI / configuration / API / building from source / releases / project structure</summary>

### Manual install (without the script)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### Full CLI

```bash
free-proxy serve                 # 运行控制台 + 代理网关 + 后台任务
free-proxy install               # 一键安装:二进制 + 依赖 + 环境文件 + 服务(需 root)
free-proxy uninstall             # 卸载服务与二进制,--purge-data 同时删数据(需 root)
free-proxy credentials           # 打印管理地址与一次性密码
free-proxy discover              # 拉取并存储节点
free-proxy status                # 打印配置与数据库表
free-proxy preflight             # 启动前环境检查
free-proxy doctor [--fix]        # 检查(并可安装)系统依赖
free-proxy install-deps          # 仅安装 openvpn / iproute2 / procps(需 root)
free-proxy database-upgrade      # 执行数据库迁移
free-proxy admin-config ...      # 修改管理凭据与监听
free-proxy logs --lines 200      # 打印最近日志
```

### Configuration

The production config file defaults to `/etc/free-proxy/free-proxy.env` (generated by `free-proxy install`), and environment variables share the `FREE_PROXY_` prefix. Every subcommand reads this file automatically (process environment variables take precedence; the path can be overridden with `FREE_PROXY_ENV_FILE`). Common options:

```text
FREE_PROXY_DATA_DIR=/var/lib/free-proxy
FREE_PROXY_DATABASE_URL=
FREE_PROXY_SQL_ECHO=false
FREE_PROXY_ALLOW_PROCESS_RESTART=true
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_OPENVPN_USERNAME=vpn
FREE_PROXY_OPENVPN_PASSWORD=vpn
FREE_PROXY_TUNNEL_INTERFACE=fpx0
FREE_PROXY_PROBE_DEVICE_PREFIX=fpx
FREE_PROXY_TEST_TUN_START=1
FREE_PROXY_TEST_TUN_END=64
FREE_PROXY_POLICY_ROUTING_TABLE=9527
```

> Web port, proxy port, credentials, discovery, maintenance, DNS, routing, and external-access options are managed in the dashboard and stored in SQLite.

### Coexisting with 3x-ui and other panels

Interface names and policy routing table ids are **host-global namespaces**. Earlier releases claimed `tun0` and table `100` — the defaults that nearly every other tunnel program also picks (3x-ui's sing-box/Xray TUN inbound, WARP, tun2socks, another OpenVPN instance) — so on a machine already running 3x-ui the service failed with `TUN device is unavailable or not permitted` ([#2](https://github.com/MasterAlanLab/free-proxy/issues/2)).

The project now allocates only from its own namespace:

| Resource | Value | Notes |
| --- | --- | --- |
| Active tunnel device | `fpx0` | Outside the shared `tunN` pool |
| Probe device pool | `fpx1`–`fpx64` | Each name is verified free against the kernel before use |
| Policy routing table | `9527` | Also registered in `/etc/iproute2/rt_tables.d/free-proxy.conf` so other tools can see the id is taken |

Three guarantees back that up:

- **Verified before use** — a probe device name is checked against the host's existing interfaces before it is handed to OpenVPN, so we never race another program for a name.
- **Attribution-based teardown** — disconnect and shutdown remove only rules and routes pointing at an `fpx*` device. The blanket `ip rule del table N` / `ip route flush table N` is gone, so even when the table id is shared, a neighbour's entries are left untouched.
- **Diagnosed early** — `free-proxy doctor` and the dashboard's system diagnostics report device-name and routing-table conflicts, naming the setting that moves us out of the way.

Upgrades are migrated automatically: `free-proxy install` rewrites configuration still holding the old defaults (`tun0` / `100`) and leaves any value you chose yourself alone. To move further, set `FREE_PROXY_TUNNEL_INTERFACE`, `FREE_PROXY_PROBE_DEVICE_PREFIX`, or `FREE_PROXY_POLICY_ROUTING_TABLE` and restart the service.

> Web port `39527` and proxy port `9527` are shared resources too. They do not collide with 3x-ui's defaults; change them in the dashboard if something else holds them.

On low-spec VPSes (e.g. 1 core / 1 GB) you can lower the probe load:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### API Overview

All endpoints live under the secret path prefix: `/{secret_path}/api/v1/...`. Long-running operations return `202 + Job`, which you poll via `GET /jobs/{id}`.

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

### Tech Stack

- **Go 1.23+**, Echo v5 (Web/API), sqlc + `modernc.org/sqlite` (pure Go, no CGO), goose (embedded migrations), cobra (CLI), log/slog.
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, with the build output embedded into the binary via `//go:embed`.
- Passwords hashed with `scrypt`; authentication via a random secret path + session cookie.

### Building From Source

Requires Go 1.23+ and bun.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

Because there's no CGO, you can produce Linux binaries directly on macOS. Copy the locally built artifact to the target machine and run `sudo ./free-proxy install` to deploy.

Hot-reload development:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Publishing a Release

`install.sh` downloads `free-proxy-linux-{amd64,arm64}` from GitHub Releases, built and published automatically by `.github/workflows/release.yml` **when a version tag is pushed**:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

Tags must start with `v`. Once the release is published, the `latest` download in `install.sh` will resolve to that binary.

### Project Structure

```text
cmd/free-proxy      # 入口 + cobra 子命令 + serve 装配
internal/
  config domain logging security store        # 基础层
  proxy tunnel netx providers ipinfo          # 代理/隧道/网络/数据源
  services                                    # 用例服务 + 后台监控
  api web                                     # Echo 服务 + 内嵌前端
frontend/           # React 源码(构建到 internal/web/dist)
install.sh          # 引导脚本:下载二进制并执行 free-proxy install
```

</details>

---

## 📄 Disclaimer

- This project is for learning, exchange, and **lawful use** only. Please comply with the laws and regulations of your region and never use it for any illegal activity.
- The free nodes are provided by a third party (VPNGate); their availability and security are not guaranteed by this project, so **do not transmit sensitive information through free nodes**.
- The VPS, virtual credit card, Telegram bot, and other links in this document are promotional / affiliate links. Ordering through them may earn the author a small commission at **no extra cost to you** — thanks for your support ❤️

## 🙏 Acknowledgements and References

This project drew on the open-source project **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)** for its design ideas and implementation, and we give special thanks here 🙏

## License

See [LICENSE](LICENSE).
