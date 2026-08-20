**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Crea con un clic il tuo pool di proxy gratuiti personale

> Esegui **un solo comando** su un piccolo VPS all'estero: recupererà automaticamente centinaia di uscite gratuite da fonti pubbliche di nodi (VPNGate), con test di velocità reali e selezione intelligente della linea più veloce, offrendo verso l'esterno un **proxy SOCKS5 / HTTP** stabile. Quando un nodo cade, passa automaticamente a un altro, senza che tu debba mai stare a controllare.

> 🎥 Video dimostrativo: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="Distribuzione con un comando" src="https://img.shields.io/badge/Distribuzione-un%20comando-brightgreen">
  <img alt="Go binario unico" src="https://img.shields.io/badge/Go-binario%20unico·zero%20dipendenze-00ADD8">
  <img alt="Gratis" src="https://img.shields.io/badge/Nodi-gratis·test%20velocità%20automatico-orange">
</p>

**A chi è rivolto?**

- Vuoi un'uscita proxy **tua e sotto il tuo controllo**, invece di affidare il tuo traffico al servizio di qualcun altro.
- Hai (o stai per acquistare) un VPS all'estero e vuoi trasformarlo in un gateway proxy completamente automatico.
- Non vuoi perdere tempo con configurazioni complicate: **lo installi con un comando e lo usi con pochi clic nell'interfaccia web**.

---

## ✨ Caratteristiche principali

- 🔌 **Distribuzione con un comando**: dipendenze, servizio e avvio automatico all'accensione sono gestiti in modo completamente automatico, alla portata anche dei principianti.
- 🌍 **Scoperta automatica + test di velocità reali**: recupera centinaia di nodi da fonti pubbliche, testa realmente connettività e latenza e sceglie automaticamente il più veloce.
- ♻️ **Cambio automatico in caso di disconnessione**: i nodi gratuiti sono instabili? In background riconnette e cambia automaticamente, mantenendo il proxy sempre online.
- 🧩 **SOCKS5 / HTTP sulla stessa porta**: un'unica porta `9527` per tutto, con riconoscimento automatico del protocollo dal primo byte.
- 🖥️ **Pannello web essenziale**: pool di nodi, stato del gateway, log e strategie tutto in un'unica schermata.
- 📦 **File unico, zero dipendenze**: un singolo binario statico con frontend e database integrati, pronto all'uso appena installato.

---

## 🛒 Prima di iniziare: prepara queste due cose (da leggere se sei alle prime armi)

### 1️⃣ Un VPS Linux all'estero (in gergo "piccolo VPS")

Questo strumento deve girare su un **server Linux all'estero** (con root e supporto TUN). Per i principianti consigliamo i due provider qui sotto, entrambi supportano il pagamento con **Alipay** e sono pronti all'uso subito:

| Consigliato | Adatto a | Caratteristiche | Link |
|---|---|---|---|
| **BandwagonHost (搬瓦工)** | 🔰 Principianti / rapporto qualità-prezzo | Marchio storico e affidabile, prezzi accessibili, supporta Alipay, linee premium CN2 GIA opzionali, pronto all'uso | **[Acquista ora 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Massima velocità / fascia alta | Ottimizzazione top per i tre operatori cinesi / linee CN2 GIA, bassa latenza, alta velocità, esperienza al massimo | **[Acquista ora 👉](https://cutt.ly/YywJIzY0)** |

> 💡 Budget limitato e voglia di semplicità → scegli **[BandwagonHost](https://cutt.ly/qywJNWzd)**; se vuoi velocità e qualità di linea al massimo → scegli **[DMIT](https://cutt.ly/YywJIzY0)**.
> Come sistema operativo scegli **Ubuntu / Debian** (questo tutorial li usa come esempio), come piano scegli KVM (supporta TUN di default).

### 2️⃣ Una "carta" con cui poter pagare

La maggior parte dei VPS all'estero richiede una carta di credito / PayPal. **Non hai una carta di credito internazionale?** Con una **carta di credito virtuale internazionale** ne apri una in pochi minuti e sottoscrivi facilmente ogni tipo di servizio estero (VPS, ChatGPT, streaming, software in abbonamento, ecc.):

> 💳 **[Carta di credito virtuale internazionale · Apertura rapida 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Distribuzione in tre passaggi (versione davvero per principianti)

Supponiamo che tu abbia già acquistato il VPS e ottenuto l'**IP del server** e la **password di root**.

**Passaggio 1 · Accedi al tuo VPS via SSH**

```bash
ssh root@你的服务器IP
```

**Passaggio 2 · Installazione con un comando**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

Lo script eseguirà automaticamente: download del programma per l'architettura corrispondente → installazione delle dipendenze di sistema (openvpn ecc.) → registrazione del servizio con avvio automatico → avvio. Basta attendere che finisca, senza alcuna interazione.

**Passaggio 3 · Prendi nota dell'indirizzo di amministrazione e delle credenziali**

Dopo la prima installazione, lo script **stampa direttamente** il percorso, il nome utente e la password generati casualmente:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 Percorso, nome utente e password vengono generati casualmente solo alla **prima installazione**. Salvali subito perché la password non è recuperabile in seguito.
> 🔒 Gli aggiornamenti successivi mantengono invariati percorso, nome utente e password. Per cambiarli esplicitamente, usa il pannello o esegui `free-proxy install --rotate-admin`.

✅ **Fatto!** Il servizio sta già recuperando nodi, testando la velocità e connettendosi automaticamente in background. Ora vediamo come usarlo.

---

## 🌐 Come usare il proxy / accedere al pannello web

Il servizio è in ascolto su `0.0.0.0` di default e include un **interruttore "Accesso da Internet"** (modificabile in qualsiasi momento nella pagina "Strategie" del pannello, **con effetto immediato, senza riavvio**). L'accesso locale e tramite tunnel SSH è **sempre disponibile**, indipendentemente dall'interruttore.

### Pannello web: accesso da Internet consentito di default ✅

Con doppia protezione tramite login + percorso con chiave casuale, una volta installato puoi aprirlo da Internet. Nel browser vai all'indirizzo stampato da `free-proxy credentials`:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

Se non ti serve l'accesso pubblico, puoi disattivare il suo interruttore di accesso da Internet nel pannello, oppure usare un tunnel SSH (vedi sotto).

### Porta del proxy: solo locale di default 🔒

Per evitare di trasformarsi in un **"proxy aperto"** utilizzabile da chiunque, il proxy serve solo la macchina locale di default. Per usarlo da Internet, due passaggi:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **Attiva nel pannello**: entra nel pannello web in "Strategie → Accesso da Internet", spunta "Consenti accesso da Internet alla porta del proxy" e salva.

Dopodiché potrai usarlo nelle applicazioni locali: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 L'uso più prudente (senza aprire nulla a Internet): disattiva l'accesso da Internet al pannello web e usa un tunnel SSH —
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`, poi accedi in locale a `127.0.0.1`.

### Verifica che il proxy funzioni

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

Se vedi un IP diverso da quello del tuo VPS, significa che il proxy sta già inoltrando il traffico attraverso l'uscita VPN 🎉

---

## 🖱️ Come usare il pannello web

1. Apri l'indirizzo di amministrazione e accedi con nome utente e password.
2. Clicca su **"Aggiorna e verifica i nodi"** e attendi che li scopra, ne testi la velocità e si connetta automaticamente al più veloce.
3. Il pannello **"Gateway"** mostra il nodo di uscita corrente, la latenza e l'IP di uscita.
4. Imposta il proxy delle tue applicazioni locali su `127.0.0.1:9527` e puoi iniziare a usarlo.

---

## 🔧 Comandi comuni

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**Aggiornare all'ultima versione**: basta rieseguire il comando di installazione. Dati, impostazioni, percorso di amministrazione, nome utente e password restano invariati.

---

## ❓ Domande frequenti

- **Non riesci a connetterti / nessun nodo al momento?** I nodi gratuiti (VPNGate) sono per natura instabili; il servizio riprova e cambia automaticamente. Attendi ancora un po', oppure clicca una volta su "Aggiorna e verifica i nodi" nel pannello.
- **Ti viene richiesto root / TUN?** Esegui come root e verifica che il VPS abbia TUN/TAP abilitato. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** usano entrambi architettura KVM, la supportano di default e sono pronti all'uso.
- **Il mio VPS è di architettura ARM?** Non preoccuparti, lo script di installazione riconosce automaticamente amd64 / arm64.
- **Posso eseguirlo sul mio computer (macOS/Windows)?** Puoi compilarlo e svilupparlo, ma il tunnel reale e il proxy di uscita richiedono Linux + root + TUN, quindi distribuiscilo su un VPS.

---

## 🧰 Strumenti e risorse consigliati

- 🔎 **Il miglior bot di ricerca per Telegram** — lo strumento perfetto per trovare film, software, ebook e ogni tipo di risorsa, li trovi con una sola ricerca: 👉 **[Apri](https://cutt.ly/2yeh3GOE)**
- 🖥️ Non hai ancora un server? **[BandwagonHost (rapporto qualità-prezzo per principianti)](https://cutt.ly/qywJNWzd)** · **[DMIT (linee di fascia alta)](https://cutt.ly/YywJIzY0)**
- 💳 Non hai una carta internazionale per pagare? **[Carta di credito virtuale internazionale](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Consigli sulla sicurezza

- Il servizio è in ascolto su `0.0.0.0` di default e la sua esposizione è controllata dall'interruttore "Accesso da Internet" del pannello: **il pannello web è aperto di default** (protetto da login + percorso con chiave), **il proxy è solo locale di default**. Quando non serve l'accesso pubblico, puoi disattivare l'accesso da Internet al pannello web e usare un tunnel SSH.
- **Prima di abilitare l'accesso da Internet al proxy devi obbligatoriamente impostare nome utente e password del proxy**, altrimenti diventerebbe un "proxy aperto" utilizzabile da chiunque, facilmente soggetto ad abusi che potrebbero portare al blocco del VPS; per questo, in assenza di password, il sistema rifiuta qualsiasi richiesta di proxy proveniente da Internet.
- Dopo il primo accesso, cambia al più presto le credenziali di amministrazione. Il tunnel, il routing basato su strategie e l'installazione delle dipendenze richiedono root: abilitalo solo su server che controlli personalmente.

---

## 🧑‍💻 Uso avanzato (per sviluppatori)

<details>
<summary>Clicca per espandere: riga di comando / opzioni di configurazione / API / build dal sorgente / pubblicazione / struttura del progetto</summary>

### Installazione manuale (senza script)

```bash
curl -fL https://github.com/masteralanlab/free-proxy/releases/latest/download/free-proxy-linux-amd64 -o free-proxy
chmod +x free-proxy && sudo ./free-proxy install
```

### CLI completa

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

### Configurazione

Il file di configurazione per l'ambiente di produzione è `/etc/free-proxy/free-proxy.env` di default (generato da `free-proxy install`); le variabili d'ambiente usano tutte il prefisso `FREE_PROXY_`. Tutti i sottocomandi leggono automaticamente questo file (le variabili d'ambiente del processo hanno la precedenza; il percorso può essere sovrascritto con `FREE_PROXY_ENV_FILE`). Opzioni comuni:

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

Su VPS a bassa potenza (es. 1 core / 1 GB) puoi ridurre il carico di probing:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### Riepilogo delle API

Tutti gli endpoint sono sotto il prefisso del percorso sicuro: `/{secret_path}/api/v1/...`. Le operazioni di lunga durata restituiscono `202 + Job`, che si interroga tramite `GET /jobs/{id}`.

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

### Stack tecnologico

- **Go 1.23+**, Echo v5 (Web/API), sqlc + `modernc.org/sqlite` (puro Go, senza CGO), goose (migrazioni integrate), cobra (CLI), log/slog.
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, con gli artefatti di build integrati nel binario tramite `//go:embed`.
- Password con hash `scrypt`, autenticazione tramite percorso sicuro casuale + cookie di sessione.

### Build dal sorgente

Richiede Go 1.23+ e bun.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

Poiché non usa CGO, puoi produrre binari Linux direttamente su macOS. Dopo aver copiato l'artefatto di build locale sulla macchina di destinazione, esegui `sudo ./free-proxy install` per distribuirlo.

Hot reload in fase di sviluppo:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Pubblicazione di una Release

`install.sh` scarica `free-proxy-linux-{amd64,arm64}` dalle GitHub Releases, che vengono costruite e pubblicate automaticamente da `.github/workflows/release.yml` al **push di un tag di versione**:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

Il tag deve iniziare con `v`. Al termine della pubblicazione, il download `latest` di `install.sh` punterà a quel binario.

### Struttura del progetto

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

## 📄 Esclusione di responsabilità

- Questo progetto è destinato esclusivamente all'apprendimento, allo scambio e a **usi leciti**; rispetta le leggi e i regolamenti della tua area e non usarlo per alcuna attività illegale.
- I nodi gratuiti sono forniti da terze parti (VPNGate); la loro disponibilità e sicurezza non sono garantite da questo progetto, quindi **non trasmettere informazioni sensibili attraverso i nodi gratuiti**.
- I link a VPS, carte di credito virtuali, bot Telegram, ecc. presenti nel testo sono link promozionali / di affiliazione (affiliate); effettuando un ordine tramite essi l'autore potrebbe ricevere una piccola commissione, **senza costi aggiuntivi per te**, grazie del supporto ❤️

## 🙏 Ringraziamenti e riferimenti

Nella progettazione e nell'implementazione questo progetto ha preso spunto dal progetto open source **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**, al quale va un ringraziamento speciale 🙏

## License

Consulta il file [LICENSE](LICENSE).
