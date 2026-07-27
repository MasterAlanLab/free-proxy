**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Baue deinen eigenen kostenlosen Proxy-Pool mit einem einzigen Befehl

> Führe auf einem kleinen ausländischen Server **einen einzigen Befehl** aus: Automatisch werden aus öffentlichen Knotenquellen (VPNGate) hunderte kostenlose Ausgänge abgerufen, real per Geschwindigkeit getestet, die schnellste Route intelligent ausgewählt und nach außen ein stabiler **SOCKS5-/HTTP-Proxy** bereitgestellt. Fällt ein Knoten aus, wird automatisch umgeschaltet – du musst nichts überwachen.

<p>
  <img alt="Ein-Klick-Bereitstellung" src="https://img.shields.io/badge/Bereitstellung-Ein%20Befehl-brightgreen">
  <img alt="Go Einzelbinary" src="https://img.shields.io/badge/Go-Einzelbinary·ohne%20Abhängigkeiten-00ADD8">
  <img alt="kostenlos" src="https://img.shields.io/badge/Knoten-kostenlos·automatischer%20Speedtest-orange">
</p>

**Für wen ist das gedacht?**

- Für alle, die einen **eigenen, kontrollierbaren** Proxy-Ausgang wollen, statt ihren Datenverkehr fremden Anbietern zu überlassen.
- Für alle, die (bereits oder demnächst) einen ausländischen VPS besitzen und ihn in ein vollautomatisches Proxy-Gateway verwandeln möchten.
- Für alle, die keine komplizierte Konfiguration möchten – **mit einem Befehl installiert, mit ein paar Klicks im Web startklar**.

---

## ✨ Kernfunktionen

- 🔌 **Bereitstellung mit einem Befehl**: Abhängigkeiten, Dienst und Autostart werden vollautomatisch eingerichtet – auch für Einsteiger geeignet.
- 🌍 **Automatische Erkennung + realer Speedtest**: Hunderte Knoten werden aus öffentlichen Quellen abgerufen, Erreichbarkeit und Latenz real gemessen und der schnellste automatisch ausgewählt.
- ♻️ **Automatisches Umschalten bei Ausfall**: Kostenlose Knoten sind instabil? Im Hintergrund wird automatisch neu verbunden und umgeschaltet, sodass der Proxy dauerhaft online bleibt.
- 🧩 **SOCKS5 / HTTP auf demselben Port**: Ein einziger Port `9527` für alles, das Protokoll wird anhand des ersten Bytes automatisch erkannt.
- 🖥️ **Schlichtes Web-Backend**: Knotenpool, Gateway-Status, Logs und Strategien auf einen Blick.
- 📦 **Einzeldatei ohne Abhängigkeiten**: Eine statische Binary mit eingebettetem Frontend und eingebetteter Datenbank – sofort einsatzbereit.

---

## 🛒 Bevor du loslegst: diese beiden Dinge vorbereiten (Pflichtlektüre für Einsteiger)

### 1️⃣ Ein ausländischer Linux-VPS (umgangssprachlich „kleiner Server")

Dieses Tool muss auf einem **ausländischen Linux-Server** laufen (mit root, TUN-Unterstützung erforderlich). Für Einsteiger empfehlen sich die folgenden beiden Anbieter, die beide **Alipay** als Zahlungsmethode unterstützen und sofort startklar sind:

| Empfehlung | Geeignet für | Merkmale | Link |
|---|---|---|---|
| **BandwagonHost (搬瓦工)** | 🔰 Einsteiger / Preis-Leistung | Etabliert und stabil, günstiger Preis, Alipay-Unterstützung, wählbare hochwertige CN2-GIA-Route, sofort einsatzbereit | **[Jetzt kaufen 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Für höchste Geschwindigkeit / High-End | Erstklassige Optimierung für alle drei Netze / CN2-GIA-Route, niedrige Latenz, hohe Geschwindigkeit, maximales Erlebnis | **[Jetzt kaufen 👉](https://cutt.ly/YywJIzY0)** |

> 💡 Begrenztes Budget und unkompliziert → wähle **[BandwagonHost](https://cutt.ly/qywJNWzd)**; für maximale Geschwindigkeit und Routenqualität → wähle **[DMIT](https://cutt.ly/YywJIzY0)**.
> Wähle als System bitte **Ubuntu / Debian** (dieses Tutorial verwendet diese als Beispiel) und als Paket KVM (unterstützt TUN standardmäßig).

### 2️⃣ Eine „Karte", mit der du bezahlen kannst

Die meisten ausländischen VPS erfordern eine Kreditkarte / PayPal. **Keine ausländische Kreditkarte?** Mit einer **virtuellen ausländischen Kreditkarte** eröffnest du in wenigen Minuten eine Karte und abonnierst mühelos allerlei ausländische Dienste (VPS, ChatGPT, Streaming, Abo-Software usw.):

> 💳 **[Virtuelle ausländische Kreditkarte · Schnelleinstieg zur Kartenerstellung 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Bereitstellung in drei Schritten (die echte Einsteiger-Version)

Angenommen, du hast bereits einen VPS gekauft und die **Server-IP** sowie das **root-Passwort** erhalten.

**Schritt 1 · Per SSH auf deinem VPS anmelden**

```bash
ssh root@你的服务器IP
```

**Schritt 2 · Installation mit einem Befehl**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

Das Skript erledigt automatisch: das Programm für die passende Architektur herunterladen → Systemabhängigkeiten (openvpn usw.) installieren → den Autostart-Dienst registrieren → starten. Warte einfach, bis es durchgelaufen ist – der gesamte Vorgang läuft ohne Interaktion.

**Schritt 3 · Verwaltungs-URL sowie Benutzername und Passwort abrufen**

```bash
free-proxy credentials
```

Ausgegeben wird etwa:

```text
URL: http://127.0.0.1:8787/xxxxxxxxxxxx/
Username: xxxxxxxx
Password: xxxxxxxx
```

✅ **Fertig!** Der Dienst ruft im Hintergrund bereits automatisch Knoten ab, testet die Geschwindigkeit und stellt die Verbindung her. Als Nächstes sehen wir uns die Nutzung an.

---

## 🌐 So nutzt du den Proxy / greifst auf das Web-Backend zu

Der Dienst lauscht standardmäßig auf `0.0.0.0` und bringt einen integrierten **„Externer-Zugriff"-Schalter** mit (jederzeit auf der Seite „Strategie" im Backend umschaltbar, **wirkt sofort, ohne Neustart**). Lokaler Zugriff und SSH-Tunnel sind **immer verfügbar** und vom Schalter unabhängig.

### Web-Backend: externer Zugriff standardmäßig erlaubt ✅

Durch die doppelte Absicherung mit Login + zufälligem Schlüsselpfad kannst du es direkt nach der Installation von außen öffnen. Rufe im Browser die von `free-proxy credentials` ausgegebene Adresse auf:

```text
http://你的服务器IP:8787/<你的安全路径>/
```

Falls kein öffentlicher Zugriff nötig ist, kannst du den Externer-Zugriff-Schalter im Backend deaktivieren oder stattdessen einen SSH-Tunnel verwenden (siehe unten).

### Proxy-Port: standardmäßig nur lokal 🔒

Um zu vermeiden, dass daraus ein für jedermann nutzbarer **„offener Proxy"** wird, bedient der Proxy standardmäßig nur den lokalen Rechner. Um ihn von außen zu nutzen, sind zwei Schritte nötig:

1. **Proxy-Passwort setzen**: Bearbeite `/etc/free-proxy/free-proxy.env`, füge die folgenden beiden Zeilen hinzu und führe anschließend `systemctl restart free-proxy` aus:
   ```text
   FREE_PROXY_PROXY_USERNAME=自己设一个用户名
   FREE_PROXY_PROXY_PASSWORD=自己设一个强密码
   ```
2. **Im Backend aktivieren**: Gehe im Web-Backend zu „Strategie → Externer Zugriff", aktiviere „Externen Zugriff auf den Proxy-Port erlauben" und speichere.

Danach kannst du ihn in Anwendungen auf deinem Rechner nutzen: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 Die konservativste Nutzung (gar kein öffentlicher Zugriff): Deaktiviere im Backend den externen Zugriff auf das Web-Backend und verwende stattdessen einen SSH-Tunnel –
> `ssh -L 8787:127.0.0.1:8787 -L 9527:127.0.0.1:9527 root@你的服务器IP`, und greife dann lokal über `127.0.0.1` zu.

### Prüfen, ob der Proxy funktioniert

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # Zurückgegeben werden sollte die "VPN-Ausgangs-IP", nicht die IP deines eigenen VPS
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

Wenn du eine andere IP als die deines VPS siehst, bedeutet das, dass der Proxy den Datenverkehr bereits über den VPN-Ausgang weiterleitet 🎉

---

## 🖱️ So nutzt du das Web-Backend

1. Öffne die Verwaltungs-URL und melde dich mit Benutzername und Passwort an.
2. Klicke auf **„Knoten aktualisieren und prüfen"** und warte kurz, bis das System Knoten erkennt, die Geschwindigkeit testet und sich automatisch mit dem schnellsten verbindet.
3. Im Panel **„Gateway"** siehst du den aktuellen Ausgangsknoten, die Latenz und die Ausgangs-IP.
4. Richte den Proxy deiner lokalen Anwendungen auf `127.0.0.1:9527` und du kannst loslegen.

---

## 🔧 Häufige Befehle

```bash
free-proxy credentials   # Verwaltungs-URL sowie Benutzername und Passwort anzeigen
free-proxy status        # Betriebsstatus anzeigen
free-proxy logs -n 100   # Die letzten Logs anzeigen
free-proxy uninstall     # Deinstallieren (mit --purge-data werden auch die Daten gelöscht)
```

**Auf die neueste Version aktualisieren**: Führe einfach den obigen „Ein-Befehl-Installer" erneut aus – Konfiguration und Daten bleiben erhalten.

---

## ❓ Häufige Fragen

- **Keine Verbindung / vorübergehend keine Knoten?** Kostenlose Knoten (VPNGate) schwanken naturgemäß; der Dienst versucht es automatisch erneut und schaltet um. Warte etwas länger oder klicke im Backend einmal auf „Knoten aktualisieren und prüfen".
- **Meldung, dass root / TUN benötigt wird?** Führe es bitte als root aus und stelle sicher, dass auf dem VPS TUN/TAP aktiviert ist. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** basieren beide auf KVM, unterstützen es standardmäßig und sind sofort einsatzbereit.
- **Mein VPS hat eine ARM-Architektur?** Kein Problem, das Installationsskript erkennt amd64 / arm64 automatisch.
- **Kann ich es auf meinem eigenen Rechner (macOS/Windows) laufen lassen?** Kompilieren und Entwickeln ist möglich, aber der reale Tunnel und der Ausgangs-Proxy erfordern Linux + root + TUN – bitte auf einem VPS bereitstellen.

---

## 🧰 Empfohlene Tools und Ressourcen

- 🔎 **Der beste Telegram-Suchbot** –– ein Wundertool zum Finden von Filmen, Software, E-Books und allerlei Ressourcen, sofort auffindbar: 👉 **[Hier öffnen](https://cutt.ly/2yeh3GOE)**
- 🖥️ Noch keinen Server? **[BandwagonHost (Preis-Leistung für Einsteiger)](https://cutt.ly/qywJNWzd)** · **[DMIT (High-End-Route)](https://cutt.ly/YywJIzY0)**
- 💳 Keine ausländische Karte zum Bezahlen? **[Virtuelle ausländische Kreditkarte](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Sicherheitsempfehlungen

- Der Dienst lauscht standardmäßig auf `0.0.0.0`; die Freigabe nach außen wird über den „Externer-Zugriff"-Schalter im Backend gesteuert: **Das Web-Backend ist standardmäßig offen** (mit Login- und Schlüsselpfad-Schutz), **der Proxy standardmäßig nur lokal**. Wenn kein öffentlicher Zugriff nötig ist, kannst du im Backend den externen Zugriff auf das Web-Backend deaktivieren und stattdessen einen SSH-Tunnel verwenden.
- **Bevor du den externen Zugriff auf den Proxy aktivierst, musst du zwingend Benutzername und Passwort für den Proxy setzen**, sonst wird daraus ein für jedermann nutzbarer „offener Proxy", der leicht missbraucht wird und dazu führen kann, dass dein VPS gesperrt wird; deshalb weist das System ohne gesetztes Passwort alle externen Proxy-Anfragen ab.
- Ändere nach der ersten Anmeldung möglichst rasch Benutzername und Passwort des Verwaltungskontos. Tunnel, Policy-Routing und die Installation von Abhängigkeiten erfordern root – aktiviere sie nur auf Servern, die du selbst kontrollierst.

---

## 🧑‍💻 Fortgeschrittene Nutzung (für Entwickler)

<details>
<summary>Zum Ausklappen anklicken: Kommandozeile / Konfigurationsoptionen / API / Build aus dem Quellcode / Release / Projektstruktur</summary>

### Manuelle Installation (ohne Skript)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### Vollständige CLI

```bash
free-proxy serve                 # Konsole + Proxy-Gateway + Hintergrundaufgaben ausführen
free-proxy install               # Ein-Klick-Installation: Binary + Abhängigkeiten + Umgebungsdatei + Dienst (erfordert root)
free-proxy uninstall             # Dienst und Binary deinstallieren, --purge-data löscht zugleich die Daten (erfordert root)
free-proxy credentials           # Verwaltungsadresse und Einmalpasswort ausgeben
free-proxy discover              # Knoten abrufen und speichern
free-proxy status                # Konfiguration und Datenbanktabellen ausgeben
free-proxy preflight             # Umgebungsprüfung vor dem Start
free-proxy doctor [--fix]        # Systemabhängigkeiten prüfen (und ggf. installieren)
free-proxy install-deps          # Nur openvpn / iproute2 / procps installieren (erfordert root)
free-proxy database-upgrade      # Datenbankmigration ausführen
free-proxy admin-config ...      # Verwaltungs-Anmeldedaten und Listener ändern
free-proxy logs --lines 200      # Die letzten Logs ausgeben
```

### Konfiguration

Die Produktions-Konfigurationsdatei ist standardmäßig `/etc/free-proxy/free-proxy.env` (wird von `free-proxy install` erzeugt); Umgebungsvariablen tragen einheitlich das Präfix `FREE_PROXY_`. Alle Unterbefehle lesen diese Datei automatisch (Prozess-Umgebungsvariablen haben Vorrang; der Pfad lässt sich über `FREE_PROXY_ENV_FILE` überschreiben). Gebräuchliche Optionen:

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

> Der Listener bindet fest an `0.0.0.0`; ob nach außen freigegeben wird, steuert der **„Externer-Zugriff"-Schalter im Backend** (Web-Backend standardmäßig an, Proxy standardmäßig aus), wirkt zur Laufzeit sofort, ohne Neustart. Das Setzen von `FREE_PROXY_PROXY_USERNAME` / `PASSWORD` ist Voraussetzung, um den externen Zugriff auf den Proxy zu aktivieren.

Bei schwach ausgestatteten Servern (z. B. 1 Kern / 1 GB) kannst du die Prüflast reduzieren:

```text
FREE_PROXY_MAX_PROBE_CONCURRENCY=2
FREE_PROXY_DISCOVERY_LIMIT=60
FREE_PROXY_INITIAL_CONNECT_TEST_LIMIT=5
```

### API-Übersicht

Alle Endpunkte liegen unter dem sicheren Pfad-Präfix: `/{secret_path}/api/v1/...`. Langlaufende Operationen liefern `202 + Job` und werden über `GET /jobs/{id}` abgefragt.

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
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, das Build-Ergebnis wird per `//go:embed` in die Binary eingebettet.
- Passwörter werden mit `scrypt` gehasht, Authentifizierung über zufälligen sicheren Pfad + Session-Cookie.

### Aus dem Quellcode bauen

Erforderlich sind Go 1.23+ und bun.

```bash
make build        # Frontend + statische Binary nach dist/free-proxy bauen
make cross        # Cross-Compile für Linux amd64 / arm64
make test         # Go-Tests ausführen
```

Da kein CGO verwendet wird, lässt sich auf macOS direkt eine Linux-Binary erzeugen. Kopiere das lokale Build-Ergebnis auf den Zielrechner und führe `sudo ./free-proxy install` aus, um es bereitzustellen.

Hot-Reload für die Entwicklung:

```bash
cd frontend && bun install && bun run dev   # Frontend-Hot-Reload (zusammen mit dem serve unten)
go run ./cmd/free-proxy serve                # Backend (beim ersten Mal werden zufällige Verwaltungsadresse und Passwort erzeugt)
```

### Ein Release veröffentlichen

`install.sh` lädt `free-proxy-linux-{amd64,arm64}` aus den GitHub Releases; diese werden von `.github/workflows/release.yml` beim **Pushen eines Versions-Tags** automatisch gebaut und veröffentlicht:

```bash
git tag v1.0.0
git push origin v1.0.0      # Löst die Action aus: Frontend bauen + Cross-Compile → Release veröffentlichen (inkl. SHA256SUMS)
```

Der Tag muss mit `v` beginnen. Nach abgeschlossener Veröffentlichung trifft der `latest`-Download von `install.sh` genau diese Binary.

### Projektstruktur

```text
cmd/free-proxy      # Einstiegspunkt + cobra-Unterbefehle + serve-Zusammenbau
internal/
  config domain logging security store        # Basisschicht
  proxy tunnel netx providers ipinfo          # Proxy/Tunnel/Netzwerk/Datenquellen
  services                                    # Use-Case-Services + Hintergrundüberwachung
  api web                                     # Echo-Dienst + eingebettetes Frontend
frontend/           # React-Quellcode (Build nach internal/web/dist)
install.sh          # Bootstrap-Skript: Binary herunterladen und free-proxy install ausführen
```

</details>

---

## 📄 Haftungsausschluss

- Dieses Projekt dient ausschließlich dem Lernen, dem Austausch und **legalen Zwecken**; bitte halte dich an die Gesetze und Vorschriften deiner Region und verwende es keinesfalls für illegale Aktivitäten.
- Die kostenlosen Knoten werden von Dritten (VPNGate) bereitgestellt; deren Verfügbarkeit und Sicherheit werden von diesem Projekt nicht garantiert – **übertrage bitte keine sensiblen Informationen über kostenlose Knoten**.
- Die im Text genannten Links zu VPS, virtuellen Kreditkarten, Telegram-Bots usw. sind Werbe- / Empfehlungslinks (Affiliate); eine Bestellung über sie kann dem Autor eine kleine Provision einbringen, **verursacht dir jedoch keine zusätzlichen Kosten**. Danke für deine Unterstützung ❤️

## 🙏 Danksagung und Referenzen

Dieses Projekt hat sich in Konzept und Umsetzung am Open-Source-Projekt **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)** orientiert; ihm gilt ein besonderer Dank 🙏

## License

Siehe [LICENSE](LICENSE).
