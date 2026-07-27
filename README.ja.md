**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — コマンド1行であなた専用の無料プロキシプールを自前構築

> 海外の小さなVPS上で **たった1行のコマンド** を実行するだけ。公開ノードソース(VPNGate)から数百もの無料出口を自動で取得し、実測で速度を計測、最速の回線を賢く選び、外部に安定した **SOCKS5 / HTTP プロキシ** を提供します。ノードが落ちても自動で切り替わり、あなたが張り付いて監視する必要は一切ありません。

<p>
  <img alt="ワンコマンドデプロイ" src="https://img.shields.io/badge/%E3%83%87%E3%83%97%E3%83%AD%E3%82%A4-%E3%82%B3%E3%83%9E%E3%83%B3%E3%83%891%E8%A1%8C-brightgreen">
  <img alt="Go 単一バイナリ" src="https://img.shields.io/badge/Go-%E5%8D%98%E4%B8%80%E3%83%90%E3%82%A4%E3%83%8A%E3%83%AA%E3%83%BB%E4%BE%9D%E5%AD%98%E3%82%BC%E3%83%AD-00ADD8">
  <img alt="無料" src="https://img.shields.io/badge/%E3%83%8E%E3%83%BC%E3%83%89-%E7%84%A1%E6%96%99%E3%83%BB%E8%87%AA%E5%8B%95%E9%80%9F%E5%BA%A6%E6%B8%AC%E5%AE%9A-orange">
</p>

**どんな人に向いている?**

- 他人の機場(サービス)に自分のトラフィックを預けるのではなく、**自分自身でコントロールできる** プロキシ出口が欲しい人。
- 海外VPSを持っている(または購入予定の)人で、それを全自動のプロキシゲートウェイに変えたい人。
- 複雑な設定に手を焼きたくない人——**コマンド1行でインストールでき、ウェブ画面を数回クリックするだけで使えます**。

---

## ✨ 主な特長

- 🔌 **コマンド1行でデプロイ**:依存関係、サービス、自動起動まで全自動でセットアップ。初心者でもすぐに使えます。
- 🌍 **自動発見 + 実測速度計測**:公開ソースから数百のノードを取得し、接続性と遅延を実測して、自動で最速のものを選択。
- ♻️ **切断時の自動切り替え**:無料ノードは不安定?バックグラウンドが自動で再接続・切り替えを行い、プロキシを常時オンラインに保ちます。
- 🧩 **SOCKS5 / HTTP を同一ポートで**:1つのポート `9527` ですべてに対応。先頭バイトからプロトコルを自動判別します。
- 🖥️ **シンプルなウェブ管理画面**:ノードプール、ゲートウェイの状態、ログ、ポリシーを1画面で管理。
- 📦 **単一ファイル・依存ゼロ**:1つの静的バイナリにフロントエンドとデータベースを内蔵。置くだけで動きます。

---

## 🛒 始める前に:まずこの2つを準備(初心者は必読)

### 1️⃣ 海外のLinux VPS(いわゆる「小鶏(VPS)」)

本ツールは **海外のLinuxサーバー** 上で動作させる必要があります(rootが必要、TUNをサポートしていること)。初心者には以下の2社がおすすめで、いずれも **Alipay(支付宝)** での支払いに対応し、すぐに利用できます:

| おすすめ | 向いている人 | 特徴 | リンク |
|---|---|---|---|
| **搬瓦工 BandwagonHost** | 🔰 初心者 / コストパフォーマンス | 老舗で安定、手頃な価格、Alipay対応、CN2 GIA の優良回線を選択可能、箱を開けてすぐ使える | **[今すぐ購入 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 速度重視 / ハイエンド | トップクラスの三網最適化 / CN2 GIA 回線、低遅延・高速で体験は最高 | **[今すぐ購入 👉](https://cutt.ly/YywJIzY0)** |

> 💡 予算が限られていて手間をかけたくない → **[搬瓦工](https://cutt.ly/qywJNWzd)** を選択;究極の速度と回線品質が欲しい → **[DMIT](https://cutt.ly/YywJIzY0)** を選択。
> システムは **Ubuntu / Debian** を選んでください(本チュートリアルはこれを例にします)。プランは KVM(デフォルトで TUN 対応)を選びましょう。

### 2️⃣ 支払いに使える「カード」1枚

海外VPSの多くはクレジットカード / PayPal を必要とします。**海外のクレジットカードを持っていない?** **海外向けバーチャルクレジットカード** を使えば数分で1枚発行でき、各種の海外サービス(VPS、ChatGPT、ストリーミング、サブスクリプション型ソフトなど)を手軽に契約できます:

> 💳 **[海外バーチャルクレジットカード · スピード発行入口 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ 3ステップでデプロイ(真の初心者版)

VPSを購入済みで、**サーバーIP** と **rootパスワード** を入手していると仮定します。

**ステップ 1 · SSHでVPSにログイン**

```bash
ssh root@你的服务器IP
```

**ステップ 2 · コマンド1行でインストール**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

スクリプトが自動で実行します:対応するアーキテクチャのプログラムをダウンロード → システム依存(openvpn など)をインストール → 自動起動サービスを登録 → 起動。完了するまで待つだけで、途中の操作は一切不要です。

**ステップ 3 · 管理URLとアカウント・パスワードを取得**

```bash
free-proxy credentials
```

以下のような内容が表示されます:

```text
URL: http://127.0.0.1:8787/xxxxxxxxxxxx/
Username: xxxxxxxx
Password: xxxxxxxx
```

✅ **完了!** サービスはすでにバックグラウンドで自動的にノードの取得・速度計測・接続を行っています。次は使い方を見てみましょう。

---

## 🌐 プロキシの使い方 / ウェブ管理画面へのアクセス

サービスはデフォルトで `0.0.0.0` をリッスンし、**「外部アクセス」スイッチ** を内蔵しています(管理画面の「ポリシー」ページでいつでも切り替え可能、**即時反映、再起動不要**)。ローカルおよびSSHトンネルは **常に利用可能** で、スイッチの影響を受けません。

### ウェブ管理画面:デフォルトで外部アクセスを許可 ✅

ログイン + ランダムな秘密パス という二重の保護があり、インストール後すぐに外部から開けます。ブラウザで `free-proxy credentials` が表示したアドレスにアクセスします:

```text
http://你的服务器IP:8787/<你的安全路径>/
```

外部アクセスが不要なら、管理画面でその外部アクセススイッチをオフにするか、SSHトンネルに切り替えてください(下記参照)。

### プロキシポート:デフォルトはローカルのみ 🔒

誰でも使える **「オープンプロキシ」** になってしまうのを防ぐため、プロキシはデフォルトでローカルのみに提供されます。外部から使いたい場合は2ステップ:

1. **プロキシパスワードを設定**:`/etc/free-proxy/free-proxy.env` を編集して以下の2行を追加し、`systemctl restart free-proxy` を実行:
   ```text
   FREE_PROXY_PROXY_USERNAME=自己设一个用户名
   FREE_PROXY_PROXY_PASSWORD=自己设一个强密码
   ```
2. **管理画面で有効化**:ウェブ管理画面の「ポリシー → 外部アクセス」に進み、「プロキシポートの外部アクセスを許可」にチェックを入れて保存します。

その後、ローカルのアプリで次のように使用できます:`socks5://用户名:密码@你的服务器IP:9527`。

> 🔒 最も保守的な使い方(公開を一切しない):管理画面でウェブ管理画面の外部アクセスをオフにし、SSHトンネルに切り替えます——
> `ssh -L 8787:127.0.0.1:8787 -L 9527:127.0.0.1:9527 root@你的服务器IP` を実行し、ローカルで `127.0.0.1` にアクセスします。

### プロキシが有効か検証する

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

あなたのVPSとは異なるIPが表示されれば、プロキシがVPN出口を経由して転送していることを意味します 🎉

---

## 🖱️ ウェブ管理画面の使い方

1. 管理URLを開き、アカウントとパスワードでログインします。
2. **「ノードを更新して検出」** をクリックし、発見・速度計測を待つと、自動で最速のノードに接続されます。
3. **「ゲートウェイ」** パネルで、現在の出口ノード、遅延、出口IPを確認できます。
4. ローカルのアプリのプロキシを `127.0.0.1:9527` に向ければ、使い始められます。

---

## 🔧 よく使うコマンド

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**最新版へ更新**:上記の「コマンド1行でインストール」をもう一度実行するだけです。設定とデータはすべて保持されます。

---

## ❓ よくある質問

- **接続できない / 一時的にノードがない?** 無料ノード(VPNGate)自体に変動があり、サービスは自動でリトライと切り替えを行います。しばらく待つか、管理画面で一度「ノードを更新して検出」をクリックしてください。
- **root / TUN が必要と表示される?** rootで実行し、VPSでTUN/TAPが有効になっていることを確認してください。**[搬瓦工](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** はいずれもKVMアーキテクチャで、デフォルトで対応しており、箱を開けてすぐ使えます。
- **私のVPSはARMアーキテクチャ?** 気にする必要はありません。インストールスクリプトが amd64 / arm64 を自動で判別します。
- **自分のパソコン(macOS/Windows)で動かせる?** ビルドや開発は可能ですが、実際のトンネルと出口プロキシには Linux + root + TUN が必要なので、VPSにデプロイしてください。

---

## 🧰 おすすめのツールとリソース

- 🔎 **Telegram 最強の検索ボット** —— 映画、ソフト、電子書籍、各種リソースを探すのに便利な神ツール。検索すればすぐ見つかる:👉 **[開く](https://cutt.ly/2yeh3GOE)**
- 🖥️ まだサーバーがない? **[搬瓦工(初心者向けコスパ)](https://cutt.ly/qywJNWzd)** · **[DMIT(ハイエンド回線)](https://cutt.ly/YywJIzY0)**
- 💳 海外カードで支払えない? **[海外バーチャルクレジットカード](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ セキュリティに関する推奨事項

- サービスはデフォルトで `0.0.0.0` をリッスンし、公開の可否は管理画面の「外部アクセス」スイッチで制御します:**ウェブ管理画面はデフォルトで開放**(ログイン + 秘密パスで保護)、**プロキシはデフォルトでローカルのみ**。公開アクセスが不要なときは、管理画面でウェブ管理画面の外部アクセスをオフにし、SSHトンネルに切り替えられます。
- **プロキシの外部アクセスを有効にする前に、必ずプロキシのユーザー名とパスワードを設定してください**。そうしないと誰でも使える「オープンプロキシ」になってしまい、悪用されてVPSがBANされる原因になりやすいです。そのため、システムはパスワード未設定の状態ではあらゆる外部プロキシリクエストを拒否します。
- 初回ログイン後は、できるだけ早く管理アカウントのパスワードを変更してください。トンネル、ポリシールーティング、依存関係のインストールにはrootが必要なので、あなた自身が管理できるサーバーでのみ有効にしてください。

---

## 🧑‍💻 上級者向けの使い方(開発者向け)

<details>
<summary>クリックして展開:コマンドライン / 設定項目 / API / ソースからのビルド / リリース / プロジェクト構成</summary>

### 手動インストール(スクリプトを使わない)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### 完全なCLI

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

### 設定

本番環境の設定ファイルはデフォルトで `/etc/free-proxy/free-proxy.env`(`free-proxy install` によって生成)、環境変数はすべて `FREE_PROXY_` プレフィックスで統一されています。すべてのサブコマンドはこのファイルを自動的に読み込みます(プロセスの環境変数が優先;パスは `FREE_PROXY_ENV_FILE` で上書き可能)。よく使う項目:

```text
FREE_PROXY_DATA_DIR=/var/lib/free-proxy
FREE_PROXY_WEB_PORT=8787
FREE_PROXY_PROXY_PORT=9527
FREE_PROXY_PROXY_ENABLED=true
FREE_PROXY_PROXY_USERNAME=
FREE_PROXY_PROXY_PASSWORD=
FREE_PROXY_OPENVPN_COMMAND=openvpn
FREE_PROXY_TUNNEL_INTERFACE=tun0
FREE_PROXY_UPSTREAM_PROXY_URL=
FREE_PROXY_DNS_REPAIR_ENABLED=false
```

> リッスンは `0.0.0.0` に固定でバインドされます;外部に公開するかどうかは **管理画面の「外部アクセス」スイッチ** で制御され(ウェブ管理画面はデフォルトでオン、プロキシはデフォルトでオフ)、実行時に即時反映され、再起動は不要です。`FREE_PROXY_PROXY_USERNAME` / `PASSWORD` の設定は、プロキシの外部アクセスを有効にするための前提条件です。

低スペックのVPS(1コア / 1GB など)では、探査(プローブ)の負荷を下げられます:

```text
FREE_PROXY_MAX_PROBE_CONCURRENCY=2
FREE_PROXY_DISCOVERY_LIMIT=60
FREE_PROXY_INITIAL_CONNECT_TEST_LIMIT=5
```

### API 概要

すべてのエンドポイントは秘密パスのプレフィックス配下にあります:`/{secret_path}/api/v1/...`。長時間かかる処理は `202 + Job` を返し、`GET /jobs/{id}` でポーリングします。

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

### 技術スタック

- **Go 1.23+**、Echo v5(Web/API)、sqlc + `modernc.org/sqlite`(純Go、CGOなし)、goose(内蔵マイグレーション)、cobra(CLI)、log/slog。
- フロントエンドは **React 19 + Vite + Tailwind v4 + Zustand**、ビルド成果物は `//go:embed` でバイナリに内蔵されます。
- パスワードは `scrypt` でハッシュ化、ランダムな秘密パス + セッションCookieで認証。

### ソースからのビルド

Go 1.23+ と bun が必要です。

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

CGOがないため、macOS上で直接Linuxバイナリを生成できます。ローカルのビルド成果物をターゲットマシンにコピーしたら、`sudo ./free-proxy install` を実行すればデプロイできます。

開発時のホットリロード:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### リリースの公開

`install.sh` は GitHub Releases から `free-proxy-linux-{amd64,arm64}` をダウンロードします。これは `.github/workflows/release.yml` によって、**バージョンタグをプッシュした** ときに自動でビルド・公開されます:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

タグは `v` で始まる必要があります。公開が完了すると、`install.sh` の `latest` ダウンロードがそのバイナリを取得します。

### プロジェクト構成

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

## 📄 免責事項

- 本プロジェクトは学習・交流および **合法的な用途** にのみ提供されます。あなたの居住する地域の法律・規制を遵守し、いかなる違法行為にも使用しないでください。
- 無料ノードは第三者(VPNGate)によって提供されており、その可用性と安全性は本プロジェクトが保証するものではありません。**無料ノードを通じて機密情報を送信しないでください**。
- 本文中のVPS、バーチャルクレジットカード、Telegramボットなどはプロモーション / 推奨(アフィリエイト)リンクです。これらを通じて注文すると作者にわずかな報酬が入る場合がありますが、**あなたの費用が余分に増えることはありません**。ご支援に感謝します ❤️

## 🙏 謝辞と参考

本プロジェクトは、設計思想と実装の面でオープンソースプロジェクト **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)** を参考にしました。ここに特別な感謝を捧げます 🙏

## License

[LICENSE](LICENSE) を参照してください。
