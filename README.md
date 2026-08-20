**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Free Proxy — 自建免费代理池

> 在海外 Linux VPS 上运行安装脚本，从公开节点源（VPNGate）获取免费出口，完成连通性检测和延迟测试后选择可用线路，并提供 **SOCKS5 / HTTP 代理**。节点不可用时，服务会自动重连或切换。

> 🎥 视频演示： [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="一键部署" src="https://img.shields.io/badge/部署-一行命令-brightgreen">
  <img alt="Go 单二进制" src="https://img.shields.io/badge/Go-单二进制·零依赖-00ADD8">
  <img alt="免费" src="https://img.shields.io/badge/节点-免费·自动测速-orange">
</p>

**适用场景**

- 希望自行管理代理出口和运行环境。
- 已有（或准备购买）一台海外 VPS，希望将其作为代理网关。
- 希望通过安装脚本和网页后台完成部署及日常管理。

---

## 主要功能

- **脚本部署**：自动安装依赖、注册系统服务并配置开机启动。
- **节点发现与测速**：从公开源获取节点，检测连通性与延迟，并按策略选择线路。
- **故障切换**：节点不可用时自动重连或切换。
- **SOCKS5 / HTTP 同端口**：默认通过 `9527` 端口提供服务，并根据首字节识别协议。
- **网页后台**：集中查看节点池、网关状态、日志和策略。
- **单文件部署**：静态二进制内嵌前端与数据库，无需单独部署 Web 服务。

---

## 开始之前

### 1. 一台海外 Linux VPS

本工具需要运行在具有 **root 权限并支持 TUN** 的海外 Linux 服务器上。搬瓦工和 DMIT 都有较成熟的三网优化线路套餐，在线路质量、稳定性以及面向中国大陆的访问体验方面都有不错的方案，适合重视网络质量和长期稳定性的用户。

| 服务商 | 定位 | 选择时可关注 | 链接 |
|---|---|---|---|
| **搬瓦工 BandwagonHost** | 三网优化线路 VPS | 部分套餐提供 CN2 GIA 等优化线路，支持支付宝 | [查看当前套餐](https://cutt.ly/qywJNWzd) |
| **DMIT** | 三网优化线路 VPS | 部分套餐提供 CN2 GIA 等优化线路，支持支付宝 | [查看当前套餐](https://cutt.ly/YywJIzY0) |

> 两家都提供多种机房和线路方案，可通过上方链接查看当前套餐，结合主要访问地区、流量需求和预算进行选择。系统建议选择 **Ubuntu / Debian**（本文以此为例），并确认套餐采用 KVM 虚拟化且支持 TUN。

### 2. 可用的付款方式

不同服务商支持的付款方式有所区别，常见选项包括信用卡、PayPal 和支付宝。如确实需要虚拟信用卡，可参考下面的推广链接：

> [海外虚拟信用卡](https://cutt.ly/IyrMR4Mg)

---

## 三步部署

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

安装完成后，服务会在后台获取节点、测速并尝试建立连接。

---

## 🌐 怎么用代理 / 访问网页后台

服务默认监听 `0.0.0.0`,并内置 **「外网访问」开关**(后台「策略」页可随时切换,**即时生效、无需重启**)。本机与 SSH 隧道**始终可用**,不受开关影响；配置代理账号密码后，本机、SSH 隧道和外网代理连接统一要求认证。

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

之后可在本机应用里使用 `socks5://用户名:密码@127.0.0.1:9527`，外部设备使用 `socks5://用户名:密码@你的服务器IP:9527`。

> 🔒 最保守的用法(完全不开公网):后台关闭网页后台外网访问,改用 SSH 隧道——
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`,然后本地访问 `127.0.0.1`。

### 验证代理是否生效

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org

# 已配置代理账号密码时，本机连接同样携带凭据
curl --proxy socks5h://用户名:密码@127.0.0.1:9527 https://api.ipify.org
curl --proxy http://用户名:密码@127.0.0.1:9527   https://api.ipify.org
```

返回结果与 VPS 的公网 IP 不同时，说明代理正在通过 VPN 出口转发。

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
free-proxy status        # 查看配置和数据库状态
free-proxy logs --lines 100  # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

所有子命令、参数、输出和使用示例见 **[命令行使用指南](docs/cli.md)**。

**更新到最新版**:重新执行一次上面的「一行命令安装」即可。节点数据、设置、管理路径、账号和密码都会保留不变。

---

## ❓ 常见问题

- **连不上 / 暂时没有节点?** 免费节点(VPNGate)本身会波动,服务会自动重试与切换。多等一会,或在后台点一次「更新并检测节点」。
- **提示需要 root / TUN?** 请用 root 运行，并确认 VPS 开启了 TUN/TAP。选择 **[搬瓦工](https://cutt.ly/qywJNWzd)** 或 **[DMIT](https://cutt.ly/YywJIzY0)** 时，也应以具体套餐的虚拟化架构和 TUN 支持情况为准。
- **我的 VPS 是 ARM 架构?** 不用管,安装脚本会自动识别 amd64 / arm64。
- **能在自己电脑(macOS/Windows)上跑吗?** 可以编译和开发,但真实隧道与出口代理需要 Linux + root + TUN,请部署到 VPS。

---

## 资源推荐

- [Telegram 资源搜索机器人](https://cutt.ly/2yeh3GOE)
- 三网优化线路 VPS：[搬瓦工](https://cutt.ly/qywJNWzd) · [DMIT](https://cutt.ly/YywJIzY0)
- [海外虚拟信用卡](https://cutt.ly/IyrMR4Mg)

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

完整的命令和参数说明已整理到 **[命令行使用指南](docs/cli.md)**，其中包含权限要求、输出说明、配置加载规则、Shell 自动补全和常用示例。

### 配置

生产环境配置文件默认 `/etc/free-proxy/free-proxy.env`(由 `free-proxy install` 生成),只保留启动和机器相关配置。后台凭据、代理服务、节点发现、检测维护、DNS 与路由参数统一在网页后台管理并写入 SQLite。升级时旧环境变量和 `web-config.json` 会一次性迁移到数据库,随后移除旧文件和已迁移的环境项。

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

> 网页默认端口 `39527`,代理默认端口 `9527`,监听固定绑定 `0.0.0.0`;端口、凭据和外网访问均在后台配置。外网访问开关即时生效,其余运行参数保存后服务自动重启。

### 与 3x-ui 等其它面板共存

网卡名和策略路由表号属于**全系统共享的命名空间**。早期版本使用 `tun0` 和路由表 `100`——几乎每个隧道类程序(3x-ui 的 sing-box / Xray TUN 入站、WARP、tun2socks、其它 OpenVPN 实例)都默认占用这两个名字,于是在装了 3x-ui 的机器上会报 `TUN device is unavailable or not permitted`([#2](https://github.com/MasterAlanLab/free-proxy/issues/2))。

现在本项目只使用自己的私有命名空间:

| 资源 | 取值 | 说明 |
| --- | --- | --- |
| 活动隧道网卡 | `fpx0` | 离开公共的 `tunN` 池 |
| 探测网卡池 | `fpx1`–`fpx64` | 分配前会向内核确认该名字未被占用 |
| 策略路由表 | `9527` | 同时写入 `/etc/iproute2/rt_tables.d/free-proxy.conf`,方便其它工具看到该表已被占用 |

三条配套保证:

- **分配前校验**——探测网卡名在下发给 OpenVPN 之前会先检查内核中是否已存在同名接口,不会再抢别人的名字。
- **按归属清理**——断开和退出时只删除指向 `fpx*` 网卡的规则与路由,绝不执行 `ip rule del table N` / `ip route flush table N` 这类会连带清空邻居配置的操作。即使路由表号与其它程序重合,对方的条目也不会被动到。
- **提前诊断**——`free-proxy doctor` 和后台「系统诊断」会检查网卡名冲突与路由表占用,并直接给出该改哪个环境变量。

从旧版本升级时,`free-proxy install` 会自动把仍是旧默认值(`tun0` / `100`)的配置迁移到新命名空间;你手动改过的值会原样保留。若仍需避让,改 `FREE_PROXY_TUNNEL_INTERFACE`、`FREE_PROXY_PROBE_DEVICE_PREFIX` 或 `FREE_PROXY_POLICY_ROUTING_TABLE` 后重启服务即可。

> 网页端口 `39527` 与代理端口 `9527` 同样是共享资源。它们不与 3x-ui 的默认端口冲突,如遇占用可在后台修改。

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
