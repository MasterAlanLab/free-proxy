**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — ワンコマンドで自分専用の無料プロキシプールを構築

> 海外の格安 VPS 上で **1 行のコマンド** を実行するだけで、公開ノードソース(VPNGate)から数百の無料出口ノードを自動収集し、実際に速度を測定して最速の回線をスマートに選び、安定した **SOCKS5 / HTTP プロキシ** を提供します。ノードが切断されても自動で切り替わり、常に監視する必要はありません。

> 🎥 デモ動画: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="ワンコマンドデプロイ" src="https://img.shields.io/badge/%E3%83%87%E3%83%97%E3%83%AD%E3%82%A4-1%E8%A1%8C%E3%83%9E%E3%83%B3%E3%83%89-brightgreen">
  <img alt="Go 単一バイナリ" src="https://img.shields.io/badge/Go-%E5%8D%98%E4%B8%80%E3%83%90%E3%82%A4%E3%83%8A%E3%83%AA%C2%B7%E4%BE%9D%E5%AD%98%E3%81%AA%E3%81%97-00ADD8">
  <img alt="無料" src="https://img.shields.io/badge/%E3%83%8E%E3%83%BC%E3%83%89-%E7%84%A1%E6%96%99%C2%B7%E8%87%AA%E5%8B%95%E9%80%9F%E5%BA%A6%E6%B8%AC%E5%AE%9A-orange">
</p>

**こんな人におすすめ**

- 他人の VPN サービスに通信を預けるのではなく、**自分で管理できる**プロキシ出口が欲しい人。
- 海外 VPS を持っている(または購入予定の)人で、それを全自動のプロキシゲートウェイに変えたい人。
- 複雑な設定に手間をかけたくない人——**1 行のコマンドでインストール、ブラウザで数クリックすれば使える**。

---

## ✨ 主な特長

- 🔌 **1 行のコマンドでデプロイ**:依存関係、サービス、自動起動をすべて全自動で処理。初心者でも簡単。
- 🌍 **自動検出 + 実測速度**:公開ソースから数百のノードを収集し、接続性と遅延を実測して最速のものを自動選択。
- ♻️ **切断時の自動切り替え**:無料ノードが不安定?バックグラウンドで自動再接続・切り替えし、プロキシを常時オンラインに保ちます。
- 🧩 **SOCKS5 / HTTP 同一ポート**:1 つのポート `9527` で両対応、最初のバイトでプロトコルを自動判別。
- 🖥️ **シンプルな Web 管理画面**:ノードプール、ゲートウェイ状態、ログ、ポリシーを 1 画面で管理。
- 📦 **単一ファイル・依存なし**:1 つの静的バイナリにフロントエンドとデータベースを内蔵、配置すればすぐ動作。

---

## 🛒 始める前に:この 2 つを準備(初心者は必読)

### 1️⃣ 海外の Linux VPS(いわゆる「VPS」)

このツールは**海外の Linux サーバー**上で動作させる必要があります(root 権限が必要、TUN 対応)。初心者には以下の 2 社がおすすめで、いずれも **Alipay(支付宝)** での支払いに対応し、すぐに利用できます:

| おすすめ | 向いている人 | 特徴 | リンク |
|---|---|---|---|
| **BandwagonHost(搬瓦工)** | 🔰 初心者 / コスパ重視 | 老舗で安定、手頃な価格、Alipay 対応、CN2 GIA 優良回線も選択可、すぐ使える | **[今すぐ購入 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 速度重視 / ハイエンド | トップクラスの三大キャリア最適化 / CN2 GIA 回線、低遅延・高速で快適さ抜群 | **[今すぐ購入 👉](https://cutt.ly/YywJIzY0)** |

> 💡 予算が限られていて手軽に済ませたい → **[BandwagonHost（搬瓦工）](https://cutt.ly/qywJNWzd)** を選択;究極の速度と回線品質が欲しい → **[DMIT](https://cutt.ly/YywJIzY0)** を選択。
> OS は **Ubuntu / Debian** を選んでください(本チュートリアルはこれを例に説明します)。プランは KVM(デフォルトで TUN 対応)を選択してください。

### 2️⃣ 支払いできる「カード」1 枚

海外 VPS のほとんどはクレジットカード / PayPal が必要です。**海外クレジットカードをお持ちでない?** **海外バーチャルクレジットカード**なら数分で 1 枚発行でき、さまざまな海外サービス(VPS、ChatGPT、ストリーミング、サブスク型ソフトなど)に手軽に登録できます:

> 💳 **[海外バーチャルクレジットカード · かんたん発行はこちら 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ 3 ステップでデプロイ(超・初心者版)

すでに VPS を購入済みで、**サーバー IP** と **root パスワード** を入手済みだと仮定します。

**ステップ 1 · SSH で VPS にログイン**

```bash
ssh root@你的服务器IP
```

**ステップ 2 · 1 行のコマンドでインストール**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

スクリプトが自動的に:対応アーキテクチャのプログラムをダウンロード → システム依存関係(openvpn など)をインストール → 自動起動サービスを登録 → 起動、まで行います。完了するまで待つだけで、途中の操作は一切不要です。

**ステップ 3 · 管理 URL とアカウント・パスワードを控える**

初回インストールの完了時に、ランダム生成されたパス、アカウント、パスワードが**そのまま表示されます**:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 パス、アカウント、パスワードは**初回インストール時のみ**ランダム生成されます。パスワードは後から復元できないため、その場で保存してください。
> 🔒 以後の更新では既存のパス、アカウント、パスワードが保持されます。明示的に変更する場合は管理画面または `free-proxy install --rotate-admin` を使用します。

✅ **完了!** サービスはすでにバックグラウンドで自動的にノード収集・速度測定・接続を行っています。続いて使い方を見ていきましょう。

---

## 🌐 プロキシの使い方 / Web 管理画面へのアクセス

サービスはデフォルトで `0.0.0.0` をリッスンし、**「外部アクセス」スイッチ**を内蔵しています(管理画面の「ポリシー」ページでいつでも切り替え可能、**即時反映・再起動不要**)。ローカルおよび SSH トンネルは**常に利用可能**で、このスイッチの影響を受けません。

### Web 管理画面:デフォルトで外部アクセス許可 ✅

ログイン + ランダムなシークレットパスの二重保護があり、インストール後すぐ外部から開けます。ブラウザで `free-proxy credentials` に表示されたアドレスにアクセスしてください:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

公開アクセスが不要な場合は、管理画面でその外部アクセススイッチをオフにするか、SSH トンネル(下記参照)に切り替えられます。

### プロキシポート:デフォルトはローカルのみ 🔒

誰でも使える**「オープンプロキシ」**になってしまうのを防ぐため、プロキシはデフォルトでローカルのみに提供されます。外部から使いたい場合は 2 ステップ:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **管理画面で有効化**:Web 管理画面の「ポリシー → 外部アクセス」に入り、「プロキシポートの外部アクセスを許可」にチェックを入れて保存します。

その後、ローカルのアプリで使用できます:`socks5://用户名:密码@你的服务器IP:9527`。

> 🔒 最も安全な使い方(公開アクセスを一切開かない):管理画面で Web 管理画面の外部アクセスをオフにし、SSH トンネルに切り替えます——
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP` を実行し、ローカルで `127.0.0.1` にアクセスします。

### プロキシが有効か検証する

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

VPS とは異なる IP が表示されれば、プロキシがすでに VPN 出口経由で転送していることを意味します 🎉

---

## 🖱️ Web 管理画面の使い方

1. 管理 URL を開き、アカウントとパスワードでログインします。
2. **「ノードを更新して検出」** をクリックし、ノードの発見・速度測定・最速ノードへの自動接続が終わるまで少し待ちます。
3. **「ゲートウェイ」** パネルで現在の出口ノード、遅延、出口 IP を確認できます。
4. ローカルのアプリのプロキシを `127.0.0.1:9527` に向ければ、すぐに使い始められます。

---

## 🔧 よく使うコマンド

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**最新版へ更新**:上記の「1 行のコマンドでインストール」をもう一度実行するだけです。ノードデータ、設定、管理パス、アカウント、パスワードはすべて保持されます。

---

## ❓ よくある質問

- **接続できない / 一時的にノードがない?** 無料ノード(VPNGate)自体に変動があり、サービスは自動で再試行と切り替えを行います。しばらく待つか、管理画面で「ノードを更新して検出」を一度クリックしてください。
- **root / TUN が必要と表示される?** root で実行し、VPS で TUN/TAP が有効になっていることを確認してください。**[BandwagonHost（搬瓦工）](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** はいずれも KVM アーキテクチャで、デフォルトで対応しており、すぐに使えます。
- **私の VPS は ARM アーキテクチャです?** 気にする必要はありません。インストールスクリプトが amd64 / arm64 を自動判別します。
- **自分のパソコン(macOS/Windows)で動かせますか?** ビルドや開発は可能ですが、実際のトンネルと出口プロキシには Linux + root + TUN が必要なので、VPS にデプロイしてください。

---

## 🧰 おすすめツールとリソース

- 🔎 **Telegram 最強検索ボット** —— 映画、ソフト、電子書籍、各種リソースを探すのに最適、検索すればすぐ見つかる:👉 **[開く](https://cutt.ly/2yeh3GOE)**
- 🖥️ サーバーをまだお持ちでない? **[BandwagonHost（搬瓦工／初心者向けコスパ）](https://cutt.ly/qywJNWzd)** · **[DMIT（ハイエンド回線）](https://cutt.ly/YywJIzY0)**
- 💳 海外カードで支払えない? **[海外バーチャルクレジットカード](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ セキュリティの推奨事項

- サービスはデフォルトで `0.0.0.0` をリッスンし、公開範囲は管理画面の「外部アクセス」スイッチで制御されます:**Web 管理画面はデフォルトで公開**(ログイン + シークレットパス保護あり)、**プロキシはデフォルトでローカルのみ**。公開アクセスが不要なときは、管理画面で Web 管理画面の外部アクセスをオフにし、SSH トンネルに切り替えられます。
- **プロキシの外部アクセスを有効にする前に、必ずプロキシのユーザー名とパスワードを設定してください**。さもないと誰でも使える「オープンプロキシ」になり、悪用されて VPS が BAN される恐れが非常に高くなります。そのためシステムは、パスワード未設定の状態ではすべての外部プロキシリクエストを拒否します。
- 初回ログイン後は、できるだけ早く管理アカウントのパスワードを変更してください。トンネル、ポリシールーティング、依存関係のインストールには root が必要なので、自分で管理できるサーバーでのみ有効化してください。

---

## 🧑‍💻 上級者向けの使い方(開発者向け)

<details>
<summary>クリックして展開:コマンドライン / 設定項目 / API / ソースビルド / リリース / プロジェクト構成</summary>

### 手動インストール(スクリプトを使わない)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### 完全な CLI

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

本番環境の設定ファイルはデフォルトで `/etc/free-proxy/free-proxy.env`(`free-proxy install` により生成)、環境変数は統一して `FREE_PROXY_` 接頭辞を使います。すべてのサブコマンドがこのファイルを自動的に読み込みます(プロセス環境変数が優先;パスは `FREE_PROXY_ENV_FILE` で上書き可能)。よく使う項目:

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

低スペックの VPS(1 コア / 1G など)ではプローブ負荷を下げられます:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### API 概要

すべてのエンドポイントはシークレットパス接頭辞の下にあります:`/{secret_path}/api/v1/...`。長時間かかる操作は `202 + Job` を返し、`GET /jobs/{id}` でポーリングします。

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

- **Go 1.23+**、Echo v5(Web/API)、sqlc + `modernc.org/sqlite`(純 Go、CGO なし)、goose(内蔵マイグレーション)、cobra(CLI)、log/slog。
- フロントエンドは **React 19 + Vite + Tailwind v4 + Zustand**、ビルド成果物は `//go:embed` でバイナリに内蔵。
- パスワードは `scrypt` ハッシュ、ランダムなシークレットパス + セッション Cookie 認証。

### ソースからビルド

Go 1.23+ と bun が必要です。

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

CGO がないため、macOS 上で直接 Linux バイナリを生成できます。ローカルのビルド成果物をターゲットマシンにコピーし、`sudo ./free-proxy install` を実行すればデプロイできます。

開発時のホットリロード:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### リリースの公開

`install.sh` は GitHub Releases から `free-proxy-linux-{amd64,arm64}` をダウンロードします。これは `.github/workflows/release.yml` により、**バージョンタグをプッシュした**ときに自動でビルド・公開されます:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

タグは `v` で始める必要があります。公開が完了すると、`install.sh` の `latest` ダウンロードがそのバイナリにヒットします。

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

- 本プロジェクトは学習・交流および**合法的な用途**のみを目的としています。お住まいの地域の法律・法規を遵守し、いかなる違法行為にも使用しないでください。
- 無料ノードは第三者(VPNGate)により提供され、その可用性と安全性は本プロジェクトが保証するものではありません。**無料ノードを通じて機密情報を送信しないでください**。
- 本文中の VPS、バーチャルクレジットカード、Telegram ボットなどはプロモーション / 推奨(アフィリエイト)リンクです。それらを通じて申し込むと作者にわずかな紹介料が入る場合がありますが、**あなたの費用が追加でかかることはありません**。ご支援に感謝します ❤️

## 🙏 謝辞と参考

本プロジェクトは、設計思想と実装においてオープンソースプロジェクト **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)** を参考にしました。ここに特別な感謝を捧げます 🙏

## License

[LICENSE](LICENSE) を参照してください。
