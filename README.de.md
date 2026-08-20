**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Baue mit einem Befehl deinen eigenen kostenlosen Proxy-Pool

> Führe auf einem kleinen Server im Ausland **einen einzigen Befehl** aus: Er zieht automatisch Hunderte kostenlose Ausgänge aus öffentlichen Knotenquellen (VPNGate), misst deren Geschwindigkeit in Echtzeit, wählt intelligent die schnellste Route und stellt nach außen einen stabilen **SOCKS5- / HTTP-Proxy** bereit. Fällt ein Knoten aus, wird automatisch umgeschaltet — du musst nichts überwachen.

> 🎥 Demo-Video: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="Bereitstellung mit einem Befehl" src="https://img.shields.io/badge/Bereitstellung-Ein%20Befehl-brightgreen">
  <img alt="Go Einzelbinary" src="https://img.shields.io/badge/Go-Einzelbinary%C2%B7ohne%20Abh%C3%A4ngigkeiten-00ADD8">
  <img alt="Kostenlos" src="https://img.shields.io/badge/Knoten-Kostenlos%C2%B7Auto-Speedtest-orange">
</p>

**Für wen ist das gedacht?**

- Für alle, die einen **eigenen, kontrollierbaren** Proxy-Ausgang wollen, statt ihren Traffic einem fremden Anbieter zu überlassen.
- Für alle, die einen VPS im Ausland haben (oder kaufen wollen) und ihn in ein voll automatisches Proxy-Gateway verwandeln möchten.
- Für alle, die keine komplizierte Konfiguration wollen — **mit einem Befehl installiert, mit ein paar Klicks im Web einsatzbereit**.

---

## ✨ Kern-Highlights

- 🔌 **Bereitstellung mit einem Befehl**: Abhängigkeiten, Dienst und Autostart werden vollautomatisch erledigt — auch für Einsteiger geeignet.
- 🌍 **Automatische Erkennung + echte Geschwindigkeitsmessung**: Zieht Hunderte Knoten aus öffentlichen Quellen, testet Konnektivität und Latenz real und wählt automatisch den schnellsten.
- ♻️ **Automatische Umschaltung bei Ausfall**: Kostenlose Knoten instabil? Im Hintergrund wird automatisch neu verbunden und umgeschaltet, sodass der Proxy dauerhaft online bleibt.
- 🧩 **SOCKS5 / HTTP auf demselben Port**: Ein einziger Port `9527` für alles, das Protokoll wird am ersten Byte automatisch erkannt.
- 🖥️ **Übersichtliches Web-Backend**: Knotenpool, Gateway-Status, Logs und Strategien auf einen Blick.
- 📦 **Eine Datei, keine Abhängigkeiten**: Ein statisches Binary mit eingebettetem Frontend und Datenbank — sofort lauffähig.

---

## 🛒 Bevor du beginnst: Diese zwei Dinge vorbereiten (Pflichtlektüre für Einsteiger)

### 1️⃣ Ein Linux-VPS im Ausland (umgangssprachlich „kleiner Server")

Dieses Tool muss auf einem **Linux-Server im Ausland** laufen (root nötig, TUN-Unterstützung erforderlich). Einsteigern empfehlen wir die folgenden zwei Anbieter — beide akzeptieren **Alipay** und sind sofort einsatzbereit:

| Empfehlung | Geeignet für | Merkmale | Zum Anbieter |
|---|---|---|---|
| **BandwagonHost (搬瓦工)** | 🔰 Einsteiger / Preis-Leistung | Etabliert und stabil, günstige Preise, Alipay-Unterstützung, optional hochwertige CN2-GIA-Routen, sofort einsatzbereit | **[Jetzt kaufen 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Für Geschwindigkeit / Premium | Erstklassige Optimierung für alle drei Netzbetreiber / CN2-GIA-Routen, geringe Latenz, hohe Geschwindigkeit, maximales Erlebnis | **[Jetzt kaufen 👉](https://cutt.ly/YywJIzY0)** |

> 💡 Begrenztes Budget, unkompliziert → wähle **[BandwagonHost](https://cutt.ly/qywJNWzd)**; für maximale Geschwindigkeit und Routenqualität → wähle **[DMIT](https://cutt.ly/YywJIzY0)**.
> Wähle als System **Ubuntu / Debian** (diese Anleitung verwendet sie als Beispiel) und als Paket KVM (unterstützt TUN standardmäßig).

### 2️⃣ Eine zahlungsfähige „Karte"

Die meisten ausländischen VPS benötigen eine Kreditkarte / PayPal. **Keine ausländische Kreditkarte?** Mit einer **ausländischen virtuellen Kreditkarte** ist in wenigen Minuten eine Karte eröffnet, mit der du problemlos verschiedenste ausländische Dienste abonnieren kannst (VPS, ChatGPT, Streaming, Abo-Software usw.):

> 💳 **[Ausländische virtuelle Kreditkarte · Schnelle Karteneröffnung 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Bereitstellung in drei Schritten (echte Einsteiger-Version)

Angenommen, du hast deinen VPS bereits gekauft und die **Server-IP** sowie das **root-Passwort** erhalten.

**Schritt 1 · Per SSH auf deinem VPS anmelden**

```bash
ssh root@你的服务器IP
```

**Schritt 2 · Installation mit einem Befehl**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

Das Skript erledigt automatisch: das passende Programm für deine Architektur herunterladen → Systemabhängigkeiten (openvpn usw.) installieren → den Autostart-Dienst registrieren → starten. Lass es einfach durchlaufen, es ist keinerlei Interaktion nötig.

**Schritt 3 · Verwaltungs-URL sowie Benutzername und Passwort notieren**

Nach der ersten Installation gibt das Skript den zufällig generierten Pfad, Benutzernamen und das Passwort **direkt aus**:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 Pfad, Benutzername und Passwort werden nur bei der **ersten Installation** zufällig generiert. Bitte sofort speichern, da das Passwort später nicht wiederhergestellt werden kann.
> 🔒 Bei späteren Updates bleiben Pfad, Benutzername und Passwort unverändert. Eine bewusste Änderung ist über das Dashboard oder mit `free-proxy install --rotate-admin` möglich.

✅ **Fertig!** Der Dienst zieht bereits im Hintergrund automatisch Knoten, misst die Geschwindigkeit und stellt Verbindungen her. Schauen wir uns als Nächstes die Nutzung an.

---

## 🌐 So nutzt du den Proxy / greifst auf das Web-Backend zu

Der Dienst lauscht standardmäßig auf `0.0.0.0` und verfügt über einen eingebauten **„Zugriff aus dem Internet"-Schalter** (jederzeit auf der Seite „Strategie" im Backend umschaltbar, **sofort wirksam, kein Neustart nötig**). Der lokale Zugriff und SSH-Tunnel sind **immer verfügbar** und vom Schalter unabhängig.

### Web-Backend: standardmäßig Zugriff aus dem Internet erlaubt ✅

Durch die doppelte Absicherung aus Login + zufälligem geheimen Pfad kannst du es nach der Installation direkt aus dem Internet öffnen. Rufe im Browser die von `free-proxy credentials` ausgegebene Adresse auf:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

Falls du keinen öffentlichen Zugriff brauchst, kannst du im Backend dessen Internet-Schalter deaktivieren oder stattdessen einen SSH-Tunnel verwenden (siehe unten).

### Proxy-Port: standardmäßig nur lokal 🔒

Um zu vermeiden, dass er zu einem für jeden nutzbaren **„offenen Proxy"** wird, dient der Proxy standardmäßig nur dem lokalen Rechner. Um ihn aus dem Internet zu nutzen, sind zwei Schritte nötig:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **Im Backend aktivieren**: Gehe ins Web-Backend zu „Strategie → Zugriff aus dem Internet", aktiviere „Zugriff auf den Proxy-Port aus dem Internet erlauben" und speichere.

Danach kannst du es in Anwendungen auf deinem Rechner verwenden: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 Die konservativste Nutzung (überhaupt kein öffentlicher Zugriff): Deaktiviere im Backend den Internet-Zugriff auf das Web-Backend und verwende stattdessen einen SSH-Tunnel —
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`, dann lokal `127.0.0.1` aufrufen.

### Prüfen, ob der Proxy funktioniert

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

Wenn du eine IP siehst, die sich von der deines VPS unterscheidet, bedeutet das, dass der Proxy den Traffic bereits über den VPN-Ausgang weiterleitet 🎉

---

## 🖱️ So bedienst du das Web-Backend

1. Öffne die Verwaltungs-URL und melde dich mit Benutzername und Passwort an.
2. Klicke auf **„Knoten aktualisieren und prüfen"** und warte kurz, bis er die Knoten erkennt, misst und automatisch mit dem schnellsten verbindet.
3. Im Panel **„Gateway"** siehst du den aktuellen Ausgangsknoten, die Latenz und die Ausgangs-IP.
4. Richte den Proxy deiner lokalen Anwendungen auf `127.0.0.1:9527` und schon geht's los.

---

## 🔧 Häufige Befehle

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**Auf die neueste Version aktualisieren**: Führe einfach den obigen „Installationsbefehl in einer Zeile" erneut aus. Knotendaten, Einstellungen, Verwaltungspfad, Benutzername und Passwort bleiben unverändert.

---

## ❓ Häufige Fragen

- **Keine Verbindung / vorübergehend keine Knoten?** Kostenlose Knoten (VPNGate) schwanken naturgemäß; der Dienst versucht es automatisch erneut und schaltet um. Warte etwas länger oder klicke im Backend einmal auf „Knoten aktualisieren und prüfen".
- **Meldung, dass root / TUN benötigt wird?** Führe es bitte als root aus und stelle sicher, dass der VPS TUN/TAP aktiviert hat. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** sind beide KVM-basiert, unterstützen es standardmäßig und sind sofort einsatzbereit.
- **Mein VPS hat ARM-Architektur?** Kein Problem, das Installationsskript erkennt amd64 / arm64 automatisch.
- **Kann ich es auf meinem eigenen Rechner (macOS/Windows) laufen lassen?** Kompilieren und Entwickeln ist möglich, aber der echte Tunnel und der Ausgangs-Proxy benötigen Linux + root + TUN — bitte auf einem VPS bereitstellen.

---

## 🧰 Empfohlene Tools und Ressourcen

- 🔎 **Der beste Telegram-Suchbot** — das perfekte Werkzeug, um Filme, Software, E-Books und alle Arten von Ressourcen zu finden, sofortige Treffer: 👉 **[Hier öffnen](https://cutt.ly/2yeh3GOE)**
- 🖥️ Noch keinen Server? **[BandwagonHost (Preis-Leistung für Einsteiger)](https://cutt.ly/qywJNWzd)** · **[DMIT (Premium-Routen)](https://cutt.ly/YywJIzY0)**
- 💳 Keine ausländische Karte zum Bezahlen? **[Ausländische virtuelle Kreditkarte](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Sicherheitsempfehlungen

- Der Dienst lauscht standardmäßig auf `0.0.0.0`; die Freigabe wird durch den „Zugriff aus dem Internet"-Schalter im Backend gesteuert: **Das Web-Backend ist standardmäßig offen** (mit Login- + geheimem-Pfad-Schutz), **der Proxy ist standardmäßig nur lokal**. Wenn du keinen öffentlichen Zugriff brauchst, kannst du im Backend den Internet-Zugriff auf das Web-Backend deaktivieren und stattdessen einen SSH-Tunnel verwenden.
- **Bevor du den Internet-Zugriff auf den Proxy aktivierst, musst du zwingend einen Proxy-Benutzernamen und ein Passwort setzen**, sonst wird er zu einem für jeden nutzbaren „offenen Proxy", der leicht missbraucht wird und dazu führen kann, dass dein VPS gesperrt wird; deshalb weist das System bei nicht gesetztem Passwort alle Proxy-Anfragen aus dem Internet ab.
- Ändere nach der ersten Anmeldung möglichst schnell Benutzername und Passwort des Verwaltungskontos. Tunnel, Policy-Routing und Abhängigkeitsinstallation benötigen root — aktiviere sie nur auf Servern, die du selbst kontrollierst.

---

## 🧑‍💻 Fortgeschrittene Nutzung (für Entwickler)

<details>
<summary>Zum Ausklappen klicken: Kommandozeile / Konfigurationsoptionen / API / Build aus dem Quellcode / Release / Projektstruktur</summary>

### Manuelle Installation (ohne Skript)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### Vollständige CLI

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

### Konfiguration

Die Konfigurationsdatei für die Produktionsumgebung ist standardmäßig `/etc/free-proxy/free-proxy.env` (von `free-proxy install` erzeugt), Umgebungsvariablen haben einheitlich das Präfix `FREE_PROXY_`. Alle Unterbefehle lesen diese Datei automatisch ein (Prozess-Umgebungsvariablen haben Vorrang; der Pfad kann mit `FREE_PROXY_ENV_FILE` überschrieben werden). Häufige Optionen:

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

Bei schwach ausgestatteten kleinen Servern (z. B. 1 Kern / 1 GB) kannst du die Prüflast verringern:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### API-Übersicht

Alle Endpunkte liegen unter dem geheimen Pfad-Präfix: `/{secret_path}/api/v1/...`. Lang laufende Operationen geben `202 + Job` zurück und werden per `GET /jobs/{id}` abgefragt.

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

### Technologie-Stack

- **Go 1.23+**, Echo v5 (Web/API), sqlc + `modernc.org/sqlite` (reines Go, ohne CGO), goose (eingebettete Migrationen), cobra (CLI), log/slog.
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, das Build-Ergebnis wird per `//go:embed` ins Binary eingebettet.
- Passwörter als `scrypt`-Hash, Authentifizierung über zufälligen geheimen Pfad + Session-Cookie.

### Aus dem Quellcode bauen

Benötigt Go 1.23+ und bun.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

Da kein CGO verwendet wird, kannst du unter macOS direkt Linux-Binaries erzeugen. Kopiere das lokal erstellte Build-Ergebnis auf die Zielmaschine und führe `sudo ./free-proxy install` aus, um es bereitzustellen.

Hot-Reload für die Entwicklung:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Release veröffentlichen

`install.sh` lädt `free-proxy-linux-{amd64,arm64}` aus den GitHub Releases herunter; sie werden von `.github/workflows/release.yml` beim **Pushen eines Versions-Tags** automatisch gebaut und veröffentlicht:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

Das Tag muss mit `v` beginnen. Nach Abschluss der Veröffentlichung trifft der `latest`-Download von `install.sh` genau dieses Binary.

### Projektstruktur

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

## 📄 Haftungsausschluss

- Dieses Projekt dient ausschließlich dem Lernen, dem Austausch und **legalen Zwecken**. Bitte halte dich an die Gesetze und Vorschriften deiner Region und nutze es keinesfalls für illegale Aktivitäten.
- Die kostenlosen Knoten werden von einem Dritten (VPNGate) bereitgestellt; ihre Verfügbarkeit und Sicherheit werden von diesem Projekt nicht garantiert. Bitte **übertrage keine sensiblen Informationen über kostenlose Knoten**.
- Die im Text genannten VPS-, virtuellen Kreditkarten-, Telegram-Bot-Links usw. sind Werbe- / Empfehlungslinks (Affiliate). Bestellungen über sie können dem Autor eine kleine Provision einbringen, **ohne dass dir zusätzliche Kosten entstehen**. Danke für deine Unterstützung ❤️

## 🙏 Danksagung und Referenzen

Dieses Projekt orientiert sich in Designideen und Umsetzung am Open-Source-Projekt **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**, wofür wir uns hiermit besonders bedanken 🙏

## License

Siehe [LICENSE](LICENSE).
