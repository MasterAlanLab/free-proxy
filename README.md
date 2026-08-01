**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — 一键自建你的专属免费代理池

> 在一台海外小鸡上跑 **一行命令**,自动从公开节点源(VPNGate)抓取上百个免费出口、真实测速、智能挑选最快的线路,对外提供一个稳定的 **SOCKS5 / HTTP 代理**。节点掉线自动切换,全程无需你盯着。

<p>
  <img alt="一键部署" src="https://img.shields.io/badge/部署-一行命令-brightgreen">
  <img alt="Go 单二进制" src="https://img.shields.io/badge/Go-单二进制·零依赖-00ADD8">
  <img alt="免费" src="https://img.shields.io/badge/节点-免费·自动测速-orange">
</p>

**它适合谁?**

- 想要一个**自己的、可控的**代理出口,而不是把流量交给别人的机场。
- 手上有(或准备买)一台海外 VPS,想把它变成一个全自动的代理网关。
- 不想折腾复杂配置——**一行命令装好,网页点几下就能用**。

---

## ✨ 核心亮点

- 🔌 **一行命令部署**:依赖、服务、开机自启全自动搞定,小白也能上手。
- 🌍 **自动发现 + 真实测速**:从公开源抓取上百节点,实测连通性与延迟,自动选最快的。
- ♻️ **掉线自动切换**:免费节点不稳?后台自动重连、切换,保持代理常在线。
- 🧩 **SOCKS5 / HTTP 同端口**:一个端口 `9527` 通吃,首字节自动识别协议。
- 🖥️ **简洁网页后台**:节点池、网关状态、日志、策略一屏搞定。
- 📦 **单文件零依赖**:一个静态二进制,内嵌前端与数据库,落地即跑。

---

## 🛒 开始之前:先准备这两样(小白必看)

### 1️⃣ 一台海外 Linux VPS(俗称"小鸡")

本工具需要跑在一台**海外的 Linux 服务器**上(要 root、支持 TUN)。新手推荐下面两家,均支持**支付宝**付款、开机即用:

| 推荐 | 适合 | 特点 | 传送门 |
|---|---|---|---|
| **搬瓦工 BandwagonHost** | 🔰 新手 / 性价比 | 老牌稳定、价格亲民、支持支付宝,可选 CN2 GIA 优质线路,开箱即用 | **[点此选购 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 追求速度 / 高端 | 顶级三网优化 / CN2 GIA 线路,延迟低、速度快,体验拉满 | **[点此选购 👉](https://cutt.ly/YywJIzY0)** |

> 💡 预算有限、图省心 → 选 **[搬瓦工](https://cutt.ly/qywJNWzd)**;想要极致速度与线路质量 → 选 **[DMIT](https://cutt.ly/YywJIzY0)**。
> 系统请选 **Ubuntu / Debian**(本教程以此为例),套餐选 KVM(默认支持 TUN)。

### 2️⃣ 一张能付款的"卡"

海外 VPS 大多需要信用卡 / PayPal。**没有海外信用卡?** 用**海外虚拟信用卡**几分钟就能开一张,轻松订阅各类海外服务(VPS、ChatGPT、流媒体、订阅制软件等):

> 💳 **[海外虚拟信用卡 · 快速开卡入口 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ 三步部署(真·小白版)

假设你已经买好 VPS,拿到了 **服务器 IP** 和 **root 密码**。

**第 1 步 · SSH 登录你的 VPS**

```bash
ssh root@你的服务器IP
```

**第 2 步 · 一行命令安装**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

脚本会自动:下载对应架构的程序 → 安装系统依赖(openvpn 等)→ 注册开机自启服务 → 启动。等它跑完即可,全程无需交互。

**第 3 步 · 记下管理网址和账号密码**

首次安装完成时，脚本会**直接打印**随机生成的路径、账号、密码:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 路径、账号、密码仅在**首次安装**时随机生成，没有任何默认值，请当场保存（密码事后无法找回）。
> 🔒 后续重新运行安装进行更新时会保留原有路径、账号和密码。如需主动更换，可使用后台设置或执行 `free-proxy install --rotate-admin`。

✅ **搞定!** 服务已经在后台自动抓节点、测速、连接了。接下来看看怎么用。

---

## 🌐 怎么用代理 / 访问网页后台

服务默认监听 `0.0.0.0`,并内置 **「外网访问」开关**(后台「策略」页可随时切换,**即时生效、无需重启**)。本机与 SSH 隧道**始终可用**,不受开关影响。

### 网页后台:默认允许外网访问 ✅

有登录 + 随机密钥路径双重保护,装好即可从外网打开。浏览器访问 `free-proxy credentials` 打印的地址:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

如无需公网访问,可在后台关闭它的外网开关,或改用 SSH 隧道(见下)。

### 代理端口:默认仅本机 🔒

为避免变成任何人可用的 **「开放代理」**,代理默认只服务本机。想从外网使用,两步:

1. **设置代理凭据**:进入网页后台「策略 → 后台与代理服务」填写代理用户名和新密码。
2. **后台开启**:勾选「允许代理端口外网访问」并保存。配置写入 SQLite,密码只保存 scrypt 哈希。

之后即可在本机应用里使用:`socks5://用户名:密码@你的服务器IP:9527`。

> 🔒 最保守的用法(完全不开公网):后台关闭网页后台外网访问,改用 SSH 隧道——
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`,然后本地访问 `127.0.0.1`。

### 验证代理是否生效

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

看到一个和你 VPS 不同的 IP,就说明代理已经在通过 VPN 出口转发了 🎉

---

## 🖱️ 网页后台怎么用

1. 打开管理网址,用账号密码登录。
2. 点 **「更新并检测节点」**,稍等它发现、测速并自动连上最快的节点。
3. **「网关」** 面板可看到当前出口节点、延迟和出口 IP。
4. 把本机应用的代理指向 `127.0.0.1:9527` 即可开始使用。

---

## 🔧 常用命令

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**更新到最新版**:重新执行一次上面的「一行命令安装」即可。节点数据、设置、管理路径、账号和密码都会保留不变。

---

## ❓ 常见问题

- **连不上 / 暂时没有节点?** 免费节点(VPNGate)本身会波动,服务会自动重试与切换。多等一会,或在后台点一次「更新并检测节点」。
- **提示需要 root / TUN?** 请用 root 运行,并确认 VPS 开启了 TUN/TAP。**[搬瓦工](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** 均为 KVM 架构,默认支持,开箱即用。
- **我的 VPS 是 ARM 架构?** 不用管,安装脚本会自动识别 amd64 / arm64。
- **能在自己电脑(macOS/Windows)上跑吗?** 可以编译和开发,但真实隧道与出口代理需要 Linux + root + TUN,请部署到 VPS。

---

## 🧰 推荐工具与资源

- 🔎 **Telegram 最强搜索机器人** —— 找电影、软件、电子书、各类资源的神器,一搜即得:👉 **[点此使用](https://cutt.ly/2yeh3GOE)**
- 🖥️ 还没有服务器? **[搬瓦工(新手性价比)](https://cutt.ly/qywJNWzd)** · **[DMIT(高端线路)](https://cutt.ly/YywJIzY0)**
- 💳 没有海外卡付款? **[海外虚拟信用卡](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ 安全建议

- 服务默认监听 `0.0.0.0`,由后台「外网访问」开关控制暴露:**网页后台默认开放**(有登录 + 密钥路径保护),**代理默认仅本机**。无需公网访问时,可在后台关闭网页后台外网访问,改用 SSH 隧道。
- **开启代理外网访问前必须先设置代理用户名密码**,否则会变成任何人可用的「开放代理」,极易被滥用导致 VPS 被封;为此系统在未设密码时会拒绝一切外网代理请求。
- 首次登录后请尽快修改管理账号密码。隧道、策略路由与依赖安装需要 root,请只在你自己可控的服务器上启用。

---

## 🧑‍💻 进阶使用(开发者向)

<details>
<summary>点击展开:命令行 / 配置项 / API / 源码构建 / 发布 / 项目结构</summary>

### 手动安装(不走脚本)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### 完整 CLI

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

### 配置

生产环境配置文件默认 `/etc/free-proxy/free-proxy.env`(由 `free-proxy install` 生成),只保留启动和机器相关配置。后台凭据、代理服务、节点发现、检测维护、DNS 与路由参数统一在网页后台管理并写入 SQLite。升级时旧环境变量和 `web-config.json` 会一次性迁移到数据库,随后移除旧文件和已迁移的环境项。

```text
FREE_PROXY_DATA_DIR=/var/lib/free-proxy
FREE_PROXY_DATABASE_URL=
FREE_PROXY_SQL_ECHO=false
FREE_PROXY_ALLOW_PROCESS_RESTART=true
FREE_PROXY_PREFLIGHT_STRICT=false
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_OPENVPN_USERNAME=vpn
FREE_PROXY_OPENVPN_PASSWORD=vpn
FREE_PROXY_TUNNEL_INTERFACE=tun0
FREE_PROXY_TEST_TUN_START=2
FREE_PROXY_TEST_TUN_END=99
FREE_PROXY_POLICY_ROUTING_TABLE=100
```

> 网页默认端口 `39527`,代理默认端口 `9527`,监听固定绑定 `0.0.0.0`;端口、凭据和外网访问均在后台配置。外网访问开关即时生效,其余运行参数保存后服务自动重启。

弱配置小鸡(如 1 核 / 1G)可在后台调低「探测并发数」「每次发现节点上限」「首次连接检测数」。

### API 摘要

所有端点在安全路径前缀下:`/{secret_path}/api/v1/...`。长耗时操作返回 `202 + Job`,通过 `GET /jobs/{id}` 轮询。

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

### 技术栈

- **Go 1.23+**、Echo v5(Web/API)、sqlc + `modernc.org/sqlite`(纯 Go,无 CGO)、goose(内嵌迁移)、cobra(CLI)、log/slog。
- 前端 **React 19 + Vite + Tailwind v4 + Zustand**,构建产物经 `//go:embed` 内嵌进二进制。
- 密码 `scrypt` 哈希,随机安全路径 + 会话 Cookie 鉴权。

### 从源码构建

需要 Go 1.23+ 与 bun。

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

因无 CGO,可在 macOS 上直接产出 Linux 二进制。本地构建产物拷到目标机器后执行 `sudo ./free-proxy install` 即可部署。

开发热更新:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### 发布 Release

`install.sh` 从 GitHub Releases 下载 `free-proxy-linux-{amd64,arm64}`,由 `.github/workflows/release.yml` 在**推送版本标签**时自动构建发布:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

标签需以 `v` 开头。发布完成后,`install.sh` 的 `latest` 下载即命中该二进制。

### 项目结构

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

## 📄 免责声明

- 本项目仅供学习交流与**合法用途**,请遵守你所在地区的法律法规,切勿用于任何非法活动。
- 免费节点由第三方(VPNGate)提供,其可用性与安全性不由本项目保证,请**勿通过免费节点传输敏感信息**。
- 文中的 VPS、虚拟信用卡、Telegram 机器人等为推广 / 推荐(affiliate)链接,通过它们下单可能为作者带来少量返佣,**不会额外增加你的花费**,感谢支持 ❤️

## 🙏 致谢与参考

本项目在设计思路与实现上参考了开源项目 **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**,在此特别致谢 🙏

## License

见 [LICENSE](LICENSE)。
