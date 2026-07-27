**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — 나만의 무료 프록시 풀을 명령어 하나로 직접 구축

> 해외 소형 서버에서 **명령어 한 줄**만 실행하면, 공개 노드 소스(VPNGate)에서 수백 개의 무료 출구를 자동으로 수집하고, 실제 속도를 측정해 가장 빠른 회선을 똑똑하게 골라 안정적인 **SOCKS5 / HTTP 프록시**를 외부에 제공합니다. 노드가 끊기면 자동으로 전환되므로, 계속 지켜볼 필요가 없습니다.

<p>
  <img alt="원클릭 배포" src="https://img.shields.io/badge/배포-명령어%20한%20줄-brightgreen">
  <img alt="Go 단일 바이너리" src="https://img.shields.io/badge/Go-단일%20바이너리·의존성%20제로-00ADD8">
  <img alt="무료" src="https://img.shields.io/badge/노드-무료·자동%20속도측정-orange">
</p>

**누구에게 적합할까요?**

- 남의 기계(VPN 서비스)에 트래픽을 맡기지 않고, **직접 소유하고 통제할 수 있는** 프록시 출구를 원하는 분.
- 해외 VPS를 가지고 있거나(혹은 구매할 예정) 이를 완전 자동화된 프록시 게이트웨이로 만들고 싶은 분.
- 복잡한 설정으로 씨름하고 싶지 않은 분——**명령어 한 줄로 설치하고, 웹에서 몇 번 클릭하면 바로 사용**.

---

## ✨ 핵심 특징

- 🔌 **명령어 한 줄로 배포**: 의존성, 서비스, 부팅 자동 시작까지 전부 자동으로 처리되어, 초보자도 손쉽게 시작할 수 있습니다.
- 🌍 **자동 발견 + 실제 속도 측정**: 공개 소스에서 수백 개의 노드를 수집하고, 연결성과 지연 시간을 실제로 측정해 가장 빠른 것을 자동으로 선택합니다.
- ♻️ **끊기면 자동 전환**: 무료 노드가 불안정하다고요? 백그라운드에서 자동으로 재연결하고 전환해 프록시를 항상 온라인으로 유지합니다.
- 🧩 **SOCKS5 / HTTP 동일 포트**: 하나의 포트 `9527`로 모두 처리하며, 첫 바이트로 프로토콜을 자동 인식합니다.
- 🖥️ **간결한 웹 관리 화면**: 노드 풀, 게이트웨이 상태, 로그, 정책을 한 화면에서 관리합니다.
- 📦 **단일 파일, 의존성 제로**: 프런트엔드와 데이터베이스를 내장한 하나의 정적 바이너리로, 올려놓기만 하면 바로 실행됩니다.

---

## 🛒 시작하기 전에: 먼저 이 두 가지를 준비하세요 (초보자 필독)

### 1️⃣ 해외 Linux VPS 한 대(흔히 "소형 서버"라고 부름)

이 도구는 **해외 Linux 서버**에서 실행해야 합니다(root 권한과 TUN 지원 필요). 초보자에게는 아래 두 곳을 추천하며, 둘 다 **알리페이(支付宝)** 결제를 지원하고 켜자마자 바로 사용할 수 있습니다.

| 추천 | 적합한 대상 | 특징 | 바로가기 |
|---|---|---|---|
| **BandwagonHost(반와공)** | 🔰 초보자 / 가성비 | 오래된 안정성, 저렴한 가격, 알리페이 지원, CN2 GIA 프리미엄 회선 선택 가능, 개봉 즉시 사용 | **[바로 구매 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 속도 추구 / 고급형 | 최고급 3망 최적화 / CN2 GIA 회선, 낮은 지연, 빠른 속도로 완벽한 경험 | **[바로 구매 👉](https://cutt.ly/YywJIzY0)** |

> 💡 예산이 빠듯하고 편하게 쓰고 싶다면 → **[BandwagonHost](https://cutt.ly/qywJNWzd)** 선택; 극한의 속도와 회선 품질을 원한다면 → **[DMIT](https://cutt.ly/YywJIzY0)** 선택.
> 운영체제는 **Ubuntu / Debian**(이 가이드는 이를 기준으로 함)을 선택하고, 플랜은 KVM(기본적으로 TUN 지원)을 선택하세요.

### 2️⃣ 결제 가능한 "카드" 한 장

해외 VPS는 대부분 신용카드 / PayPal이 필요합니다. **해외 신용카드가 없다고요?** **해외 가상 신용카드**를 이용하면 몇 분 만에 한 장 발급받아 각종 해외 서비스(VPS, ChatGPT, 스트리밍, 구독형 소프트웨어 등)를 손쉽게 구독할 수 있습니다.

> 💳 **[해외 가상 신용카드 · 빠른 발급 입구 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ 3단계 배포 (진짜 초보자 버전)

이미 VPS를 구매해 **서버 IP**와 **root 비밀번호**를 받았다고 가정합니다.

**1단계 · SSH로 VPS에 로그인**

```bash
ssh root@你的服务器IP
```

**2단계 · 명령어 한 줄로 설치**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

스크립트가 자동으로: 해당 아키텍처의 프로그램 다운로드 → 시스템 의존성(openvpn 등) 설치 → 부팅 자동 시작 서비스 등록 → 시작. 실행이 끝날 때까지 기다리기만 하면 되며, 중간에 어떤 조작도 필요 없습니다.

**3단계 · 관리 주소와 계정/비밀번호 받기**

```bash
free-proxy credentials
```

다음과 비슷하게 출력됩니다:

```text
URL: http://127.0.0.1:8787/xxxxxxxxxxxx/
Username: xxxxxxxx
Password: xxxxxxxx
```

✅ **완료!** 서비스가 이미 백그라운드에서 노드를 자동으로 수집하고, 속도를 측정하고, 연결하고 있습니다. 이제 사용 방법을 알아봅시다.

---

## 🌐 프록시 사용 방법 / 웹 관리 화면 접속 방법

서비스는 기본적으로 `0.0.0.0`에서 수신 대기하며, **"외부 접속" 스위치**를 내장하고 있습니다(관리 화면의 "정책" 페이지에서 언제든 전환 가능, **즉시 적용되며 재시작 불필요**). 로컬 및 SSH 터널은 **항상 사용 가능**하며 이 스위치의 영향을 받지 않습니다.

### 웹 관리 화면: 기본적으로 외부 접속 허용 ✅

로그인 + 랜덤 비밀 경로의 이중 보호가 적용되어, 설치 후 바로 외부에서 열 수 있습니다. 브라우저에서 `free-proxy credentials`가 출력한 주소로 접속하세요:

```text
http://你的服务器IP:8787/<你的安全路径>/
```

공개망 접속이 필요 없다면, 관리 화면에서 외부 접속 스위치를 끄거나 SSH 터널로 전환하면 됩니다(아래 참고).

### 프록시 포트: 기본적으로 로컬 전용 🔒

누구나 사용할 수 있는 **"오픈 프록시"**가 되는 것을 방지하기 위해, 프록시는 기본적으로 로컬에만 서비스를 제공합니다. 외부에서 사용하려면 두 단계를 거치세요:

1. **프록시 비밀번호 설정**: `/etc/free-proxy/free-proxy.env`를 편집해 아래 두 줄을 추가한 뒤 `systemctl restart free-proxy` 실행:
   ```text
   FREE_PROXY_PROXY_USERNAME=자신이 정한 사용자 이름
   FREE_PROXY_PROXY_PASSWORD=자신이 정한 강력한 비밀번호
   ```
2. **관리 화면에서 활성화**: 웹 관리 화면의 "정책 → 외부 접속"으로 들어가 "프록시 포트 외부 접속 허용"을 체크하고 저장.

이후 로컬 애플리케이션에서 다음과 같이 사용할 수 있습니다: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 가장 보수적인 사용법(공개망을 전혀 열지 않음): 관리 화면에서 웹 관리 화면 외부 접속을 끄고 SSH 터널로 전환——
> `ssh -L 8787:127.0.0.1:8787 -L 9527:127.0.0.1:9527 root@你的服务器IP`, 그런 다음 로컬에서 `127.0.0.1`로 접속.

### 프록시가 정상 작동하는지 확인

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 반환되는 것은 "VPN 출구 IP"여야 하며, VPS 자신의 IP가 아니어야 합니다
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

VPS와 다른 IP가 보인다면, 프록시가 VPN 출구를 통해 전달하고 있다는 뜻입니다 🎉

---

## 🖱️ 웹 관리 화면 사용법

1. 관리 주소를 열고 계정/비밀번호로 로그인합니다.
2. **"노드 업데이트 및 검사"**를 클릭하고, 발견·속도 측정 후 가장 빠른 노드에 자동으로 연결될 때까지 잠시 기다립니다.
3. **"게이트웨이"** 패널에서 현재 출구 노드, 지연 시간, 출구 IP를 확인할 수 있습니다.
4. 로컬 애플리케이션의 프록시를 `127.0.0.1:9527`로 지정하면 바로 사용할 수 있습니다.

---

## 🔧 자주 쓰는 명령어

```bash
free-proxy credentials   # 관리 주소와 계정/비밀번호 확인
free-proxy status        # 실행 상태 확인
free-proxy logs -n 100   # 최근 로그 확인
free-proxy uninstall     # 제거(--purge-data 를 붙이면 데이터까지 함께 삭제)
```

**최신 버전으로 업데이트**: 위의 "명령어 한 줄 설치"를 다시 한 번 실행하면 되며, 설정과 데이터는 모두 유지됩니다.

---

## ❓ 자주 묻는 질문

- **연결이 안 됨 / 일시적으로 노드가 없음?** 무료 노드(VPNGate) 자체가 변동이 있으며, 서비스가 자동으로 재시도하고 전환합니다. 잠시 더 기다리거나, 관리 화면에서 "노드 업데이트 및 검사"를 한 번 클릭하세요.
- **root / TUN이 필요하다고 나옴?** root로 실행하고, VPS에서 TUN/TAP이 활성화되어 있는지 확인하세요. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** 는 모두 KVM 아키텍처로 기본 지원되며, 개봉 즉시 사용할 수 있습니다.
- **내 VPS가 ARM 아키텍처인데요?** 신경 쓸 필요 없습니다. 설치 스크립트가 amd64 / arm64 를 자동으로 인식합니다.
- **내 컴퓨터(macOS/Windows)에서 실행할 수 있나요?** 컴파일과 개발은 가능하지만, 실제 터널과 출구 프록시는 Linux + root + TUN이 필요하므로 VPS에 배포하세요.

---

## 🧰 추천 도구 및 리소스

- 🔎 **텔레그램 최강 검색 봇** —— 영화, 소프트웨어, 전자책, 각종 자료를 찾는 마법의 도구, 검색 한 번으로 바로 획득: 👉 **[열기](https://cutt.ly/2yeh3GOE)**
- 🖥️ 아직 서버가 없다면? **[BandwagonHost(초보자 가성비)](https://cutt.ly/qywJNWzd)** · **[DMIT(고급 회선)](https://cutt.ly/YywJIzY0)**
- 💳 해외 카드로 결제할 수 없다면? **[해외 가상 신용카드](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ 보안 권장 사항

- 서비스는 기본적으로 `0.0.0.0`에서 수신 대기하며, 노출 여부는 관리 화면의 "외부 접속" 스위치로 제어합니다: **웹 관리 화면은 기본 개방**(로그인 + 비밀 경로 보호), **프록시는 기본 로컬 전용**. 공개망 접속이 필요 없을 때는 관리 화면에서 웹 관리 화면 외부 접속을 끄고 SSH 터널로 전환하세요.
- **프록시 외부 접속을 켜기 전에 반드시 프록시 사용자 이름과 비밀번호를 먼저 설정해야 합니다.** 그러지 않으면 누구나 사용할 수 있는 "오픈 프록시"가 되어 악용되기 쉽고 VPS가 차단될 수 있습니다. 이를 위해 시스템은 비밀번호가 설정되지 않은 상태에서는 모든 외부 프록시 요청을 거부합니다.
- 최초 로그인 후에는 가능한 한 빨리 관리 계정 비밀번호를 변경하세요. 터널, 정책 라우팅, 의존성 설치에는 root 권한이 필요하므로, 본인이 통제할 수 있는 서버에서만 활성화하세요.

---

## 🧑‍💻 고급 사용법 (개발자용)

<details>
<summary>클릭하여 펼치기: 커맨드라인 / 설정 항목 / API / 소스 빌드 / 릴리스 / 프로젝트 구조</summary>

### 수동 설치 (스크립트 없이)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### 전체 CLI

```bash
free-proxy serve                 # 콘솔 + 프록시 게이트웨이 + 백그라운드 작업 실행
free-proxy install               # 원클릭 설치: 바이너리 + 의존성 + 환경 파일 + 서비스(root 필요)
free-proxy uninstall             # 서비스와 바이너리 제거, --purge-data 로 데이터도 함께 삭제(root 필요)
free-proxy credentials           # 관리 주소와 일회성 비밀번호 출력
free-proxy discover              # 노드 수집 및 저장
free-proxy status                # 설정과 데이터베이스 테이블 출력
free-proxy preflight             # 시작 전 환경 점검
free-proxy doctor [--fix]        # 시스템 의존성 점검(및 설치 가능)
free-proxy install-deps          # openvpn / iproute2 / procps 만 설치(root 필요)
free-proxy database-upgrade      # 데이터베이스 마이그레이션 실행
free-proxy admin-config ...      # 관리 자격 증명과 수신 대기 설정 변경
free-proxy logs --lines 200      # 최근 로그 출력
```

### 설정

프로덕션 환경 설정 파일은 기본적으로 `/etc/free-proxy/free-proxy.env`(`free-proxy install`이 생성)이며, 환경 변수는 모두 `FREE_PROXY_` 접두사로 통일됩니다. 모든 서브 커맨드가 이 파일을 자동으로 읽습니다(프로세스 환경 변수 우선; 경로는 `FREE_PROXY_ENV_FILE` 로 재정의 가능). 자주 쓰는 항목:

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

> 수신 대기는 `0.0.0.0`에 고정 바인딩됩니다; 외부에 개방할지 여부는 **관리 화면의 "외부 접속" 스위치**로 제어하며(웹 관리 화면은 기본 켜짐, 프록시는 기본 꺼짐), 런타임에 즉시 적용되어 재시작이 필요 없습니다. `FREE_PROXY_PROXY_USERNAME` / `PASSWORD` 설정은 프록시 외부 접속을 켜기 위한 전제 조건입니다.

사양이 낮은 소형 서버(예: 1코어 / 1G)에서는 탐지 부하를 낮출 수 있습니다:

```text
FREE_PROXY_MAX_PROBE_CONCURRENCY=2
FREE_PROXY_DISCOVERY_LIMIT=60
FREE_PROXY_INITIAL_CONNECT_TEST_LIMIT=5
```

### API 요약

모든 엔드포인트는 비밀 경로 접두사 아래에 있습니다: `/{secret_path}/api/v1/...`. 장시간 소요되는 작업은 `202 + Job`을 반환하며, `GET /jobs/{id}`로 폴링합니다.

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
- 프런트엔드는 **React 19 + Vite + Tailwind v4 + Zustand**, 빌드 산출물은 `//go:embed`로 바이너리에 내장됩니다.
- 비밀번호는 `scrypt` 해시, 랜덤 비밀 경로 + 세션 쿠키 인증.

### 소스에서 빌드

Go 1.23+ 와 bun 이 필요합니다.

```bash
make build        # 프런트엔드 + 정적 바이너리를 dist/free-proxy 로 빌드
make cross        # linux amd64 / arm64 크로스 컴파일
make test         # Go 테스트 실행
```

CGO가 없으므로 macOS에서 바로 Linux 바이너리를 생성할 수 있습니다. 로컬 빌드 산출물을 대상 머신에 복사한 뒤 `sudo ./free-proxy install`을 실행하면 배포됩니다.

개발 핫 리로드:

```bash
cd frontend && bun install && bun run dev   # 프런트엔드 핫 리로드(아래 serve 와 함께 사용)
go run ./cmd/free-proxy serve                # 백엔드(최초 실행 시 랜덤 관리 주소와 비밀번호 생성)
```

### 릴리스 배포

`install.sh`는 GitHub Releases에서 `free-proxy-linux-{amd64,arm64}`를 다운로드하며, `.github/workflows/release.yml`이 **버전 태그 푸시** 시 자동으로 빌드하고 배포합니다:

```bash
git tag v1.0.0
git push origin v1.0.0      # Action 트리거: 프런트엔드 빌드 + 크로스 컴파일 → 릴리스 배포(SHA256SUMS 포함)
```

태그는 `v`로 시작해야 합니다. 배포가 완료되면 `install.sh`의 `latest` 다운로드가 해당 바이너리를 가리킵니다.

### 프로젝트 구조

```text
cmd/free-proxy      # 진입점 + cobra 서브 커맨드 + serve 조립
internal/
  config domain logging security store        # 기반 계층
  proxy tunnel netx providers ipinfo          # 프록시/터널/네트워크/데이터 소스
  services                                    # 유스케이스 서비스 + 백그라운드 모니터링
  api web                                     # Echo 서버 + 내장 프런트엔드
frontend/           # React 소스 코드(internal/web/dist 로 빌드)
install.sh          # 부트스트랩 스크립트: 바이너리 다운로드 후 free-proxy install 실행
```

</details>

---

## 📄 면책 조항

- 본 프로젝트는 학습 및 교류와 **합법적 용도**로만 제공됩니다. 거주 지역의 법률과 규정을 준수하고, 어떠한 불법 활동에도 사용하지 마세요.
- 무료 노드는 제3자(VPNGate)가 제공하며, 그 가용성과 안전성은 본 프로젝트가 보장하지 않습니다. **무료 노드를 통해 민감한 정보를 전송하지 마세요.**
- 본문의 VPS, 가상 신용카드, 텔레그램 봇 등은 홍보 / 추천(affiliate) 링크로, 이를 통해 주문하면 작성자에게 소액의 리베이트가 발생할 수 있으나 **여러분의 비용이 추가로 늘어나지는 않습니다.** 지원해 주셔서 감사합니다 ❤️

## 🙏 감사의 말과 참고

본 프로젝트는 설계 아이디어와 구현에서 오픈소스 프로젝트 **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**를 참고했으며, 이에 특별히 감사드립니다 🙏

## License

[LICENSE](LICENSE)를 참고하세요.
