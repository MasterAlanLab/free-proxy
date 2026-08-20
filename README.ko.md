**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — 명령어 한 줄로 나만의 무료 프록시 풀 구축

> 해외 VPS 한 대에서 **명령어 한 줄**만 실행하면, 공개 노드 소스(VPNGate)에서 수백 개의 무료 출구를 자동으로 수집하고, 실제로 속도를 측정한 뒤, 가장 빠른 회선을 지능적으로 선택해 안정적인 **SOCKS5 / HTTP 프록시**를 제공합니다. 노드가 끊기면 자동으로 전환되며, 계속 지켜볼 필요가 전혀 없습니다.

> 🎥 데모 영상: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="원클릭 배포" src="https://img.shields.io/badge/배포-명령어_한_줄-brightgreen">
  <img alt="Go 단일 바이너리" src="https://img.shields.io/badge/Go-단일_바이너리·의존성_제로-00ADD8">
  <img alt="무료" src="https://img.shields.io/badge/노드-무료·자동_속도측정-orange">
</p>

**누구에게 적합한가요?**

- 남의 VPN 서비스에 트래픽을 맡기는 대신, **자신만의, 통제 가능한** 프록시 출구를 원하는 분.
- 이미 해외 VPS가 있거나(또는 구매 예정) 이를 완전 자동 프록시 게이트웨이로 만들고 싶은 분.
- 복잡한 설정으로 고생하기 싫은 분 —— **명령어 한 줄로 설치하고, 웹에서 몇 번 클릭하면 바로 사용**.

---

## ✨ 핵심 특징

- 🔌 **명령어 한 줄로 배포**: 의존성, 서비스, 부팅 시 자동 시작까지 전부 자동 처리, 초보자도 쉽게 시작.
- 🌍 **자동 발견 + 실제 속도 측정**: 공개 소스에서 수백 개 노드를 수집하고, 연결성과 지연 시간을 실측해 가장 빠른 것을 자동 선택.
- ♻️ **끊기면 자동 전환**: 무료 노드가 불안정한가요? 백그라운드에서 자동으로 재연결·전환하여 프록시를 항상 온라인 상태로 유지.
- 🧩 **SOCKS5 / HTTP 동일 포트**: 하나의 포트 `9527`로 모두 처리, 첫 바이트로 프로토콜을 자동 인식.
- 🖥️ **간결한 웹 관리 콘솔**: 노드 풀, 게이트웨이 상태, 로그, 정책을 한 화면에서 처리.
- 📦 **단일 파일·의존성 제로**: 프런트엔드와 데이터베이스를 내장한 하나의 정적 바이너리, 배치하면 바로 실행.

---

## 🛒 시작하기 전에: 이 두 가지를 먼저 준비하세요 (초보자 필독)

### 1️⃣ 해외 Linux VPS 한 대 (흔히 말하는 "샤오지")

이 도구는 **해외 Linux 서버**에서 실행되어야 합니다(root 권한과 TUN 지원 필요). 초보자에게는 아래 두 곳을 추천하며, 모두 **알리페이** 결제를 지원하고 켜면 바로 사용할 수 있습니다:

| 추천 | 적합 대상 | 특징 | 바로가기 |
|---|---|---|---|
| **BandwagonHost 반와공** | 🔰 초보자 / 가성비 | 오래된 안정적인 업체, 저렴한 가격, 알리페이 지원, CN2 GIA 고품질 회선 선택 가능, 즉시 사용 가능 | **[바로 구매 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 속도 추구 / 고급형 | 최상급 3망 최적화 / CN2 GIA 회선, 낮은 지연, 빠른 속도, 최고의 경험 | **[바로 구매 👉](https://cutt.ly/YywJIzY0)** |

> 💡 예산이 제한적이고 간편함을 원한다면 → **[반와공](https://cutt.ly/qywJNWzd)** 선택; 극한의 속도와 회선 품질을 원한다면 → **[DMIT](https://cutt.ly/YywJIzY0)** 선택.
> 시스템은 **Ubuntu / Debian**을 선택하세요(본 가이드는 이를 예로 듭니다), 플랜은 KVM(기본적으로 TUN 지원)을 선택하세요.

### 2️⃣ 결제 가능한 "카드" 한 장

해외 VPS는 대부분 신용카드 / PayPal이 필요합니다. **해외 신용카드가 없나요?** **해외 가상 신용카드**를 사용하면 몇 분 안에 한 장 발급할 수 있으며, 각종 해외 서비스(VPS, ChatGPT, 스트리밍, 구독형 소프트웨어 등)를 손쉽게 구독할 수 있습니다:

> 💳 **[해외 가상 신용카드 · 빠른 발급 안내 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ 3단계 배포 (진짜 초보자용)

이미 VPS를 구매하여 **서버 IP**와 **root 비밀번호**를 받았다고 가정합니다.

**1단계 · SSH로 VPS에 로그인**

```bash
ssh root@你的服务器IP
```

**2단계 · 명령어 한 줄로 설치**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

스크립트는 자동으로: 해당 아키텍처의 프로그램 다운로드 → 시스템 의존성 설치(openvpn 등) → 부팅 시 자동 시작 서비스 등록 → 시작합니다. 완료될 때까지 기다리기만 하면 되며, 전 과정에 별도 조작이 필요 없습니다.

**3단계 · 관리 주소와 계정·비밀번호 기록**

첫 설치가 완료되면 무작위로 생성된 경로, 계정, 비밀번호를 **바로 출력**합니다:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 경로, 계정, 비밀번호는 **첫 설치 시에만** 무작위로 생성됩니다. 비밀번호는 나중에 복구할 수 없으므로 즉시 저장하세요.
> 🔒 이후 업데이트에서는 기존 경로, 계정, 비밀번호가 그대로 유지됩니다. 명시적으로 변경하려면 관리 화면이나 `free-proxy install --rotate-admin`을 사용하세요.

✅ **완료!** 서비스는 이미 백그라운드에서 자동으로 노드를 수집하고, 속도를 측정하고, 연결하고 있습니다. 이제 사용 방법을 살펴봅시다.

---

## 🌐 프록시 사용법 / 웹 관리 콘솔 접속

서비스는 기본적으로 `0.0.0.0`을 리스닝하며, **「외부 네트워크 접근」스위치**를 내장하고 있습니다(관리 콘솔의 「정책」페이지에서 언제든 전환 가능, **즉시 적용, 재시작 불필요**). 로컬 및 SSH 터널은 **항상 사용 가능**하며, 이 스위치의 영향을 받지 않습니다.

### 웹 관리 콘솔: 기본적으로 외부 네트워크 접근 허용 ✅

로그인 + 무작위 시크릿 경로의 이중 보호가 있어, 설치 후 바로 외부 네트워크에서 열 수 있습니다. 브라우저에서 `free-proxy credentials`가 출력한 주소로 접속하세요:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

공인망 접근이 필요 없다면, 관리 콘솔에서 외부 네트워크 스위치를 끄거나 SSH 터널로 바꿔 사용할 수 있습니다(아래 참조).

### 프록시 포트: 기본적으로 로컬만 🔒

누구나 사용할 수 있는 **「개방 프록시」**가 되는 것을 방지하기 위해, 프록시는 기본적으로 로컬에만 서비스합니다. 외부 네트워크에서 사용하려면 두 단계:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **관리 콘솔에서 활성화**: 웹 관리 콘솔의 「정책 → 외부 네트워크 접근」으로 들어가 「프록시 포트 외부 네트워크 접근 허용」을 체크하고 저장.

이후 로컬 애플리케이션에서 사용할 수 있습니다: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 가장 보수적인 사용법(공인망을 전혀 열지 않음): 관리 콘솔에서 웹 관리 콘솔 외부 네트워크 접근을 끄고, SSH 터널로 바꿔 사용 ——
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`, 그런 다음 로컬에서 `127.0.0.1`로 접속.

### 프록시가 작동하는지 검증

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

VPS와 다른 IP가 보이면, 프록시가 이미 VPN 출구를 통해 전달하고 있다는 뜻입니다 🎉

---

## 🖱️ 웹 관리 콘솔 사용법

1. 관리 주소를 열고, 계정과 비밀번호로 로그인합니다.
2. **「노드 업데이트 및 검사」**를 클릭하고, 노드를 발견·속도 측정하여 가장 빠른 노드에 자동으로 연결될 때까지 잠시 기다립니다.
3. **「게이트웨이」** 패널에서 현재 출구 노드, 지연 시간, 출구 IP를 볼 수 있습니다.
4. 로컬 애플리케이션의 프록시를 `127.0.0.1:9527`로 지정하면 바로 사용을 시작할 수 있습니다.

---

## 🔧 자주 쓰는 명령어

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**최신 버전으로 업데이트**: 위의 「명령어 한 줄 설치」를 다시 실행하면 됩니다. 노드 데이터, 설정, 관리 경로, 계정, 비밀번호가 모두 유지됩니다.

---

## ❓ 자주 묻는 질문

- **연결이 안 되나요 / 일시적으로 노드가 없나요?** 무료 노드(VPNGate) 자체가 변동이 있으며, 서비스가 자동으로 재시도하고 전환합니다. 조금 더 기다리거나, 관리 콘솔에서 「노드 업데이트 및 검사」를 한 번 클릭하세요.
- **root / TUN이 필요하다는 안내가 나오나요?** root로 실행하고, VPS에서 TUN/TAP이 켜져 있는지 확인하세요. **[반와공](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)**는 모두 KVM 아키텍처로 기본 지원되며, 즉시 사용 가능합니다.
- **제 VPS가 ARM 아키텍처인가요?** 신경 쓸 필요 없습니다, 설치 스크립트가 amd64 / arm64를 자동으로 인식합니다.
- **제 컴퓨터(macOS/Windows)에서 실행할 수 있나요?** 컴파일과 개발은 가능하지만, 실제 터널과 출구 프록시는 Linux + root + TUN이 필요하니 VPS에 배포하세요.

---

## 🧰 추천 도구 및 리소스

- 🔎 **텔레그램 최강 검색 봇** —— 영화, 소프트웨어, 전자책, 각종 리소스를 찾는 필수품, 검색 한 번이면 바로 찾기: 👉 **[열기](https://cutt.ly/2yeh3GOE)**
- 🖥️ 아직 서버가 없나요? **[반와공(초보자 가성비)](https://cutt.ly/qywJNWzd)** · **[DMIT(고급 회선)](https://cutt.ly/YywJIzY0)**
- 💳 해외 카드로 결제할 수 없나요? **[해외 가상 신용카드](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ 보안 권장 사항

- 서비스는 기본적으로 `0.0.0.0`을 리스닝하며, 노출은 관리 콘솔의 「외부 네트워크 접근」스위치로 제어됩니다: **웹 관리 콘솔은 기본 개방**(로그인 + 시크릿 경로 보호 있음), **프록시는 기본적으로 로컬만**. 공인망 접근이 필요 없을 때는 관리 콘솔에서 웹 관리 콘솔 외부 네트워크 접근을 끄고 SSH 터널로 바꿔 사용할 수 있습니다.
- **프록시 외부 네트워크 접근을 켜기 전에 반드시 프록시 사용자 이름과 비밀번호를 먼저 설정해야 합니다.** 그렇지 않으면 누구나 사용할 수 있는 「개방 프록시」가 되어, 악용되기 쉽고 VPS가 차단될 수 있습니다. 이를 위해 시스템은 비밀번호가 설정되지 않은 경우 모든 외부 네트워크 프록시 요청을 거부합니다.
- 최초 로그인 후에는 가능한 한 빨리 관리 계정 비밀번호를 변경하세요. 터널, 정책 라우팅, 의존성 설치에는 root가 필요하니, 본인이 통제 가능한 서버에서만 활성화하세요.

---

## 🧑‍💻 고급 사용 (개발자용)

<details>
<summary>클릭하여 펼치기: 커맨드라인 / 설정 항목 / API / 소스 빌드 / 릴리스 / 프로젝트 구조</summary>

### 수동 설치 (스크립트를 거치지 않음)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### 전체 CLI

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

### 설정

프로덕션 환경 설정 파일은 기본적으로 `/etc/free-proxy/free-proxy.env`(`free-proxy install`이 생성)이며, 환경 변수는 모두 `FREE_PROXY_` 접두사로 통일됩니다. 모든 하위 명령은 이 파일을 자동으로 읽습니다(프로세스 환경 변수가 우선; 경로는 `FREE_PROXY_ENV_FILE`로 덮어쓸 수 있음). 자주 쓰는 항목:

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

사양이 낮은 VPS(예: 1코어 / 1G)는 탐지 부하를 낮출 수 있습니다:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### API 요약

모든 엔드포인트는 시크릿 경로 접두사 아래에 있습니다: `/{secret_path}/api/v1/...`. 오래 걸리는 작업은 `202 + Job`을 반환하며, `GET /jobs/{id}`로 폴링합니다.

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

### 기술 스택

- **Go 1.23+**, Echo v5(Web/API), sqlc + `modernc.org/sqlite`(순수 Go, CGO 없음), goose(내장 마이그레이션), cobra(CLI), log/slog.
- 프런트엔드는 **React 19 + Vite + Tailwind v4 + Zustand**, 빌드 산출물은 `//go:embed`를 통해 바이너리에 내장됩니다.
- 비밀번호는 `scrypt` 해시, 무작위 보안 경로 + 세션 쿠키 인증.

### 소스에서 빌드

Go 1.23+와 bun이 필요합니다.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

CGO가 없으므로 macOS에서 직접 Linux 바이너리를 산출할 수 있습니다. 로컬 빌드 산출물을 대상 머신으로 복사한 뒤 `sudo ./free-proxy install`을 실행하면 배포됩니다.

개발용 핫 리로드:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Release 배포

`install.sh`는 GitHub Releases에서 `free-proxy-linux-{amd64,arm64}`를 다운로드하며, 이는 `.github/workflows/release.yml`이 **버전 태그를 푸시**할 때 자동으로 빌드·배포합니다:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

태그는 `v`로 시작해야 합니다. 배포가 완료되면 `install.sh`의 `latest` 다운로드가 해당 바이너리를 가리킵니다.

### 프로젝트 구조

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

## 📄 면책 조항

- 본 프로젝트는 학습 교류 및 **합법적 용도**로만 제공되니, 거주 지역의 법률과 규정을 준수하고, 어떠한 불법 활동에도 사용하지 마세요.
- 무료 노드는 제3자(VPNGate)가 제공하며, 그 가용성과 안전성은 본 프로젝트가 보장하지 않으니, **무료 노드를 통해 민감한 정보를 전송하지 마세요**.
- 본문의 VPS, 가상 신용카드, 텔레그램 봇 등은 홍보 / 추천(affiliate) 링크이며, 이를 통해 주문하면 작성자에게 소액의 수수료가 발생할 수 있으나, **당신의 비용이 추가로 늘어나지는 않습니다**. 지원해 주셔서 감사합니다 ❤️

## 🙏 감사 및 참고

본 프로젝트는 설계 사상과 구현에서 오픈소스 프로젝트 **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**를 참고했으며, 이에 특별히 감사드립니다 🙏

## License

[LICENSE](LICENSE)를 참조하세요.
