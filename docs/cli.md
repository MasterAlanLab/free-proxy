# Free Proxy 命令行使用指南

本文档列出 `free-proxy` 当前支持的全部命令和命令行参数。文中的命令默认已将二进制安装为 `/usr/local/bin/free-proxy`；直接运行本地二进制时，可将 `free-proxy` 替换为对应文件路径。

## 基本格式

```text
free-proxy [命令] [参数]
```

查看总帮助或某个命令的帮助：

```bash
free-proxy --help
free-proxy help <命令>
free-proxy <命令> --help
```

所有命令都支持以下帮助参数：

| 参数 | 说明 |
| --- | --- |
| `-h`, `--help` | 显示当前命令的帮助并退出。 |

命令执行成功时退出码为 `0`；参数解析或运行过程发生错误时退出码为非 `0`，错误信息写入标准错误输出。

## 命令总览

| 命令 | 用途 | 主要权限要求 |
| --- | --- | --- |
| [`serve`](#free-proxy-serve) | 启动网页/API、代理网关和后台任务 | 生产环境通常使用 Linux `root` |
| [`discover`](#free-proxy-discover) | 从节点提供方拉取并保存节点 | 数据目录可写、可访问网络 |
| [`credentials`](#free-proxy-credentials) | 显示管理地址、用户名和初始密码状态 | 数据目录可读写 |
| [`status`](#free-proxy-status) | 显示本地配置和数据库表状态 | 数据目录可读写 |
| [`preflight`](#free-proxy-preflight) | 执行完整的启动前检查 | 检查本身无需提权 |
| [`logs`](#free-proxy-logs) | 输出最新日志文件中的最近记录 | 日志目录可读 |
| [`admin-config`](#free-proxy-admin-config) | 修改管理凭据和监听端口 | 数据目录可读写 |
| [`database-upgrade`](#free-proxy-database-upgrade) | 将数据库迁移到最新版本 | 数据目录可读写 |
| [`doctor`](#free-proxy-doctor) | 检查系统依赖，可选择安装缺失依赖 | 使用 `--fix` 安装时需要 `root` |
| [`install-deps`](#free-proxy-install-deps) | 安装全部运行依赖 | `root` |
| [`install`](#free-proxy-install) | 安装或更新二进制、依赖、配置和系统服务 | Linux `root` |
| [`uninstall`](#free-proxy-uninstall) | 删除系统服务、配置和二进制 | Linux `root` |
| [`version`](#free-proxy-version) | 显示版本 | 无 |
| [`completion`](#free-proxy-completion) | 生成 Shell 自动补全脚本 | 无 |
| [`help`](#free-proxy-help) | 显示命令帮助 | 无 |

## 业务和运维命令

### `free-proxy serve`

启动完整服务，包括网页管理界面、HTTP API、HTTP/SOCKS5 代理网关、节点健康检查、延迟检测和维护任务。启动过程中会创建数据目录、执行数据库迁移并加载数据库中的运行设置。收到 `SIGINT` 或 `SIGTERM` 后会关闭 HTTP 服务和当前 VPN 出口。

```text
free-proxy serve [--host <地址>] [--port <端口>]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--host <地址>` | 字符串 | `0.0.0.0` | 仅覆盖本次运行的网页监听地址。可使用 `127.0.0.1` 限制为本机访问。 |
| `--port <端口>` | 整数 | 数据库中的网页端口，首次使用为 `39527` | 仅覆盖本次运行的网页监听端口；传入 `0` 等同于不覆盖。 |

示例：

```bash
# 使用已保存的监听设置启动
sudo free-proxy serve

# 本次运行仅监听本机 8080 端口，不修改数据库设置
sudo free-proxy serve --host 127.0.0.1 --port 8080
```

> `--host` 和 `--port` 是进程级临时覆盖。需要持久修改端口时使用 `admin-config --port` 或网页后台。

### `free-proxy discover`

从当前配置的 VPNGate API 拉取节点，将结果写入数据库，并以格式化 JSON 输出发现统计。该命令只执行发现和入库，不负责逐个探测节点或切换当前出口。

```text
free-proxy discover
```

此命令没有专用参数。

```bash
free-proxy discover
```

### `free-proxy credentials`

显示当前网页管理地址和用户名。如果本次执行刚刚创建或迁移了初始明文密码，也会显示 `Password`；数据库平时只保存密码哈希，因此后续执行通常会显示密码已配置但不可回读。

```text
free-proxy credentials
```

此命令没有专用参数。输出格式如下：

```text
URL: http://0.0.0.0:39527/<管理路径>/
Username: <管理员用户名>
Password: <初始密码或状态提示>
```

忘记现有密码时，可以通过 [`admin-config --password`](#free-proxy-admin-config) 设置新密码。

### `free-proxy status`

创建或打开应用数据库、执行必要的迁移，然后以格式化 JSON 显示当前环境、数据目录、数据库连接地址、网页监听地址、代理监听地址和已有数据库表。

```text
free-proxy status
```

此命令没有专用参数。它显示的是本地配置与数据库结构，不表示 systemd/OpenRC 服务进程一定正在运行。

```bash
free-proxy status
```

### `free-proxy preflight`

执行 `serve` 启动前使用的完整环境诊断，并以格式化 JSON 输出总体健康状态和每项检查结果。检查内容包括：操作系统、`root` 身份、数据目录写入、`openvpn`/`ip`/`sysctl`、TUN 设备及访问权限、默认路由、IPv4 转发、反向路径过滤、节点提供方 DNS、代理端口可用性和代理绑定设置。

```text
free-proxy preflight
```

此命令没有专用参数。诊断项失败会体现在 JSON 的 `healthy` 和 `checks` 中；请同时检查输出内容，不要只依赖进程退出码。

```bash
free-proxy preflight
```

### `free-proxy logs`

按文件名排序找到日志目录中最新的 `.json` 文件，并输出该文件末尾的若干行。日志采用一行一个 JSON 对象的格式。日志目录不存在、目录内没有 JSON 日志时不输出内容。

```text
free-proxy logs [--lines <行数>]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--lines <行数>` | 整数 | `100` | 输出最新日志文件末尾的记录行数。 |

```bash
free-proxy logs
free-proxy logs --lines 200
```

### `free-proxy admin-config`

持久修改 SQLite 中的网页管理凭据、管理路径、网页端口或代理端口。没有传入的字段保持原值；保存后，监听器相关修改需要重启服务才会生效。

```text
free-proxy admin-config [参数]
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `--username <用户名>` | 字符串 | 设置网页后台管理员用户名。空字符串不更新原值。 |
| `--password <密码>` | 字符串 | 设置网页后台管理员密码；保存时使用 scrypt 哈希。空字符串不更新原值。为避免进入 Shell 历史，交互式使用时应注意终端的历史记录。 |
| `--secret-path <路径段>` | 字符串 | 设置管理页面 URL 中的秘密路径段。空字符串不更新原值。 |
| `--port <端口>` | 整数 | 设置网页监听端口；`0` 不更新原值。 |
| `--proxy-port <端口>` | 整数 | 设置 HTTP/SOCKS5 统一代理端口；`0` 不更新原值。 |

参数可以组合使用：

```bash
# 修改用户名和密码
free-proxy admin-config --username admin --password 'NEW_PASSWORD'

# 同时修改管理路径和两个端口
free-proxy admin-config \
  --secret-path NEW_SECRET_PATH \
  --port 39527 \
  --proxy-port 9527

# 使监听器修改生效（systemd）
sudo systemctl restart free-proxy
```

### `free-proxy database-upgrade`

打开当前配置指定的 SQLite 数据库并执行全部待执行迁移。服务正常启动时也会自动迁移数据库；该命令适合在维护窗口中提前完成迁移。

```text
free-proxy database-upgrade
```

此命令没有专用参数。

```bash
free-proxy database-upgrade
```

### `free-proxy doctor`

检查基础运行条件并逐项打印 `[OK]` 或 `[FAIL]`。检查项目包括 `openvpn`、`ip`、`sysctl`、`/dev/net/tun` 和当前用户身份。发现关键依赖缺失时返回非 `0` 退出码。

```text
free-proxy doctor [--fix]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--fix` | 布尔开关 | 关闭 | 安装检查中发现的、可由包管理器修复的缺失依赖。实际需要安装时必须以 `root` 运行。 |

```bash
# 只检查
free-proxy doctor

# 检查并安装缺失的软件包
sudo free-proxy doctor --fix
```

`--fix` 只安装缺失软件包，不负责创建 `/dev/net/tun`；VPS 缺少 TUN/TAP 时需要先在服务商控制面板中启用。

### `free-proxy install-deps`

通过当前系统中检测到的包管理器安装完整依赖集合：`openvpn`、`iproute2`、`procps` 和 `ca-certificates`。支持自动检测 `apt-get`、`apk`、`dnf` 或 `yum`，其中不同发行版的实际软件包名称会自动映射。

```text
free-proxy install-deps
```

此命令没有专用参数，必须以 `root` 运行。

```bash
sudo free-proxy install-deps
```

### `free-proxy install`

在 Linux 上安装或更新 Free Proxy：

1. 将当前可执行文件复制到 `/usr/local/bin/free-proxy`；
2. 尝试安装缺失的系统依赖；
3. 首次安装时创建 `/etc/free-proxy/free-proxy.env` 和 `/var/lib/free-proxy`；
4. 创建或迁移数据库设置；
5. 安装、启用并重启 systemd 或 OpenRC 服务。

再次执行安装会保留现有节点数据、管理路径、用户名和密码，除非明确使用 `--rotate-admin`。

```text
free-proxy install [--rotate-admin]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--rotate-admin` | 布尔开关 | 关闭 | 生成新的随机管理路径、管理员用户名和密码，并在命令输出中显示新值。 |

此命令要求 Linux、`root` 权限，以及 systemd 或 OpenRC。

```bash
# 首次安装或保留设置进行更新
sudo ./free-proxy install

# 安装或更新，同时轮换全部管理入口凭据
sudo ./free-proxy install --rotate-admin
```

### `free-proxy uninstall`

停止并删除 systemd/OpenRC 服务，删除 `/etc/free-proxy` 和 `/usr/local/bin/free-proxy`。默认保留 `/var/lib/free-proxy` 中的数据库、日志和节点配置，便于以后重新安装时继续使用。

```text
free-proxy uninstall [--purge-data]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--purge-data` | 布尔开关 | 关闭 | 同时删除 `/var/lib/free-proxy` 下的数据库、日志和配置。此操作会永久清除本机应用数据。 |

此命令要求 Linux `root` 权限。

```bash
# 卸载程序并保留数据
sudo free-proxy uninstall

# 卸载程序并清除数据
sudo free-proxy uninstall --purge-data
```

### `free-proxy version`

输出当前二进制版本。Release 构建会在编译时注入版本；未注入时显示 `dev`。

```text
free-proxy version
```

此命令没有专用参数。

## 辅助命令

### `free-proxy completion`

为 Bash、Zsh、Fish 或 PowerShell 生成自动补全脚本。

```text
free-proxy completion <bash|zsh|fish|powershell> [--no-descriptions]
```

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--no-descriptions` | 布尔开关 | 关闭 | 生成补全脚本时省略候选项说明。四种 Shell 子命令均支持。 |

各 Shell 的常用安装方式：

```bash
# Bash：当前会话
source <(free-proxy completion bash)

# Bash：Linux 全局安装（需要 bash-completion）
free-proxy completion bash | sudo tee /etc/bash_completion.d/free-proxy >/dev/null

# Zsh：当前会话
source <(free-proxy completion zsh)

# Zsh：安装到 fpath 中的第一个目录
free-proxy completion zsh > "${fpath[1]}/_free-proxy"

# Fish：当前会话
free-proxy completion fish | source

# Fish：持久安装
free-proxy completion fish > ~/.config/fish/completions/free-proxy.fish

# PowerShell：当前会话
free-proxy completion powershell | Out-String | Invoke-Expression
```

执行 `free-proxy completion <Shell> --help` 可查看该 Shell 对应的完整加载说明。

### `free-proxy help`

显示总帮助或指定命令路径的帮助。

```text
free-proxy help [命令 [子命令]]
```

```bash
free-proxy help
free-proxy help serve
free-proxy help completion zsh
```

## 配置加载与命令行覆盖

需要应用配置的命令会按以下优先级读取 `FREE_PROXY_*` 环境变量：

1. 当前进程已经设置的环境变量；
2. `FREE_PROXY_ENV_FILE` 指定的文件；未指定时读取 `/etc/free-proxy/free-proxy.env`；
3. 程序内置默认值。

环境文件只读取以 `FREE_PROXY_` 开头的 `KEY=VALUE` 行，且不会覆盖当前进程已经存在的同名变量。手动运行、尚未安装时，数据目录默认是当前工作目录下的 `free_proxy_data`；执行系统安装后默认是 `/var/lib/free-proxy`。

例如，临时使用另一个数据目录查询状态：

```bash
FREE_PROXY_DATA_DIR=/tmp/free-proxy-test free-proxy status
```

网页管理凭据、网页和代理端口、节点发现、检测维护、DNS 与路由等运行设置在首次迁移后以 SQLite 为准，可通过网页后台管理。`serve --host` 和 `serve --port` 的命令行值只影响当前进程，其中端口覆盖优先于数据库设置。
