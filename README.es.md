**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Crea tu propio pool de proxies gratuito con un solo comando

> Ejecuta **una sola línea de comando** en un pequeño VPS en el extranjero y automáticamente obtiene cientos de salidas gratuitas desde fuentes de nodos públicos (VPNGate), mide su velocidad real, elige de forma inteligente la ruta más rápida y ofrece un **proxy SOCKS5 / HTTP** estable hacia el exterior. Si un nodo se cae, cambia automáticamente; no necesitas estar pendiente en ningún momento.

<p>
  <img alt="Despliegue con un comando" src="https://img.shields.io/badge/Despliegue-un%20comando-brightgreen">
  <img alt="Go binario único" src="https://img.shields.io/badge/Go-binario%20único·cero%20dependencias-00ADD8">
  <img alt="Gratis" src="https://img.shields.io/badge/Nodos-gratis·medición%20automática-orange">
</p>

**¿Para quién es?**

- Para quien quiere una salida de proxy **propia y bajo su control**, en lugar de entregar su tráfico al servicio de terceros.
- Para quien tiene (o va a comprar) un VPS en el extranjero y quiere convertirlo en una puerta de enlace de proxy totalmente automática.
- Para quien no quiere lidiar con configuraciones complejas: **se instala con una línea de comando y se usa con unos pocos clics en la web**.

---

## ✨ Aspectos destacados

- 🔌 **Despliegue con una línea de comando**: dependencias, servicio y arranque automático se resuelven solos; incluso los principiantes pueden hacerlo.
- 🌍 **Descubrimiento automático + medición real**: obtiene cientos de nodos desde fuentes públicas, prueba en la práctica la conectividad y la latencia, y elige automáticamente el más rápido.
- ♻️ **Cambio automático ante caídas**: ¿nodos gratuitos inestables? En segundo plano se reconecta y cambia automáticamente para mantener el proxy siempre en línea.
- 🧩 **SOCKS5 / HTTP en el mismo puerto**: un único puerto `9527` sirve para todo, con detección automática del protocolo por el primer byte.
- 🖥️ **Panel web sencillo**: pool de nodos, estado de la puerta de enlace, registros y estrategias, todo en una sola pantalla.
- 📦 **Archivo único sin dependencias**: un binario estático que integra el frontend y la base de datos; se ejecuta nada más ponerlo en marcha.

---

## 🛒 Antes de empezar: prepara estas dos cosas (imprescindible para principiantes)

### 1️⃣ Un VPS Linux en el extranjero (comúnmente llamado "小鸡")

Esta herramienta debe ejecutarse en un **servidor Linux en el extranjero** (requiere root y compatibilidad con TUN). Para principiantes recomendamos los dos proveedores siguientes; ambos admiten pago con **Alipay** y funcionan nada más arrancar:

| Recomendación | Ideal para | Características | Enlace |
|---|---|---|---|
| **BandwagonHost (搬瓦工)** | 🔰 Principiantes / relación calidad-precio | Marca veterana y estable, precio asequible, admite Alipay, con opción de rutas premium CN2 GIA, listo para usar | **[Comprar aquí 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Máxima velocidad / gama alta | Optimización de primer nivel para las tres operadoras / rutas CN2 GIA, baja latencia y alta velocidad, experiencia al máximo | **[Comprar aquí 👉](https://cutt.ly/YywJIzY0)** |

> 💡 ¿Presupuesto ajustado y buscas comodidad? → elige **[BandwagonHost](https://cutt.ly/qywJNWzd)**; ¿quieres velocidad extrema y calidad de ruta? → elige **[DMIT](https://cutt.ly/YywJIzY0)**.
> Para el sistema elige **Ubuntu / Debian** (este tutorial lo usa como ejemplo) y como plan elige KVM (que admite TUN por defecto).

### 2️⃣ Una "tarjeta" con la que puedas pagar

La mayoría de los VPS en el extranjero requieren tarjeta de crédito / PayPal. **¿No tienes una tarjeta de crédito internacional?** Con una **tarjeta de crédito virtual internacional** puedes abrir una en pocos minutos y suscribirte fácilmente a todo tipo de servicios extranjeros (VPS, ChatGPT, streaming, software por suscripción, etc.):

> 💳 **[Tarjeta de crédito virtual internacional · acceso rápido para abrir tu tarjeta 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Despliegue en tres pasos (versión realmente apta para principiantes)

Suponiendo que ya has comprado tu VPS y tienes la **IP del servidor** y la **contraseña de root**.

**Paso 1 · Inicia sesión en tu VPS por SSH**

```bash
ssh root@你的服务器IP
```

**Paso 2 · Instala con una línea de comando**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

El script hará automáticamente: descargar el programa para la arquitectura correspondiente → instalar las dependencias del sistema (openvpn, etc.) → registrar el servicio de arranque automático → iniciarlo. Solo tienes que esperar a que termine; no requiere interacción en ningún momento.

**Paso 3 · Obtén la dirección de administración y las credenciales**

```bash
free-proxy credentials
```

Mostrará algo similar a:

```text
URL: http://127.0.0.1:8787/xxxxxxxxxxxx/
Username: xxxxxxxx
Password: xxxxxxxx
```

✅ **¡Listo!** El servicio ya está obteniendo nodos, midiendo velocidades y conectándose automáticamente en segundo plano. Ahora veamos cómo usarlo.

---

## 🌐 Cómo usar el proxy / acceder al panel web

El servicio escucha por defecto en `0.0.0.0` e incluye un **interruptor de "acceso desde Internet"** (que puedes cambiar en cualquier momento en la página «Estrategias» del panel; **surte efecto de inmediato, sin necesidad de reiniciar**). El acceso local y el túnel SSH **están siempre disponibles**, sin verse afectados por ese interruptor.

### Panel web: acceso desde Internet permitido por defecto ✅

Cuenta con doble protección: inicio de sesión + ruta de clave aleatoria, así que puedes abrirlo desde Internet en cuanto lo instales. Abre en el navegador la dirección que imprime `free-proxy credentials`:

```text
http://你的服务器IP:8787/<你的安全路径>/
```

Si no necesitas acceso público, puedes desactivar su interruptor de acceso desde Internet en el panel, o usar un túnel SSH (ver más abajo).

### Puerto del proxy: solo local por defecto 🔒

Para evitar convertirlo en un **«proxy abierto»** que cualquiera pueda usar, el proxy solo sirve al equipo local por defecto. Para usarlo desde Internet, dos pasos:

1. **Establece una contraseña de proxy**: edita `/etc/free-proxy/free-proxy.env` y añade las dos líneas siguientes, luego ejecuta `systemctl restart free-proxy`:
   ```text
   FREE_PROXY_PROXY_USERNAME=自己设一个用户名
   FREE_PROXY_PROXY_PASSWORD=自己设一个强密码
   ```
2. **Actívalo en el panel**: entra en «Estrategias → Acceso desde Internet» del panel web, marca «Permitir acceso al puerto del proxy desde Internet» y guarda.

Después podrás usarlo en las aplicaciones de tu equipo con: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 El uso más conservador (sin abrir nada a Internet): desactiva en el panel el acceso desde Internet al panel web y usa un túnel SSH en su lugar —
> `ssh -L 8787:127.0.0.1:8787 -L 9527:127.0.0.1:9527 root@你的服务器IP`, y luego accede localmente a `127.0.0.1`.

### Verificar que el proxy funciona

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

Si ves una IP distinta a la de tu VPS, significa que el proxy ya está reenviando el tráfico a través de la salida VPN 🎉

---

## 🖱️ Cómo usar el panel web

1. Abre la dirección de administración e inicia sesión con tu usuario y contraseña.
2. Haz clic en **«Actualizar y comprobar nodos»** y espera a que descubra, mida las velocidades y se conecte automáticamente al nodo más rápido.
3. El panel **«Puerta de enlace»** muestra el nodo de salida actual, la latencia y la IP de salida.
4. Apunta el proxy de las aplicaciones de tu equipo a `127.0.0.1:9527` y ya puedes empezar a usarlo.

---

## 🔧 Comandos habituales

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**Actualizar a la última versión**: basta con volver a ejecutar la «instalación con una línea de comando» de arriba; se conservarán tanto la configuración como los datos.

---

## ❓ Preguntas frecuentes

- **¿No conecta / no hay nodos por el momento?** Los nodos gratuitos (VPNGate) fluctúan por naturaleza; el servicio reintenta y cambia automáticamente. Espera un poco más, o haz clic una vez en «Actualizar y comprobar nodos» en el panel.
- **¿Me indica que necesita root / TUN?** Ejecútalo como root y confirma que el VPS tiene TUN/TAP habilitado. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** son ambos de arquitectura KVM, lo admiten por defecto y funcionan nada más sacarlos de la caja.
- **¿Mi VPS es de arquitectura ARM?** No te preocupes, el script de instalación detecta automáticamente amd64 / arm64.
- **¿Puedo ejecutarlo en mi propio ordenador (macOS/Windows)?** Puedes compilarlo y desarrollar en él, pero el túnel real y el proxy de salida requieren Linux + root + TUN, así que despliégalo en un VPS.

---

## 🧰 Herramientas y recursos recomendados

- 🔎 **El mejor bot de búsqueda de Telegram** — la herramienta perfecta para encontrar películas, software, libros electrónicos y todo tipo de recursos, resultados al instante: 👉 **[Abrir](https://cutt.ly/2yeh3GOE)**
- 🖥️ ¿Aún no tienes servidor? **[BandwagonHost (relación calidad-precio para principiantes)](https://cutt.ly/qywJNWzd)** · **[DMIT (rutas de gama alta)](https://cutt.ly/YywJIzY0)**
- 💳 ¿No tienes una tarjeta internacional para pagar? **[Tarjeta de crédito virtual internacional](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Recomendaciones de seguridad

- El servicio escucha por defecto en `0.0.0.0`, y su exposición se controla mediante el interruptor de «acceso desde Internet» del panel: **el panel web está abierto por defecto** (con protección de inicio de sesión + ruta de clave) y **el proxy es solo local por defecto**. Cuando no necesites acceso público, puedes desactivar en el panel el acceso desde Internet al panel web y usar un túnel SSH en su lugar.
- **Antes de habilitar el acceso al proxy desde Internet debes establecer primero un usuario y una contraseña de proxy**, de lo contrario se convertiría en un «proxy abierto» que cualquiera podría usar, muy susceptible de ser abusado y provocar que bloqueen tu VPS; por ello, mientras no se haya definido una contraseña, el sistema rechazará toda solicitud de proxy desde Internet.
- Tras el primer inicio de sesión, cambia cuanto antes el usuario y la contraseña de administración. El túnel, el enrutamiento por estrategias y la instalación de dependencias requieren root, así que actívalo únicamente en servidores que controles tú mismo.

---

## 🧑‍💻 Uso avanzado (para desarrolladores)

<details>
<summary>Haz clic para desplegar: línea de comandos / opciones de configuración / API / compilación desde el código fuente / publicación / estructura del proyecto</summary>

### Instalación manual (sin usar el script)

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

### Configuración

El archivo de configuración de producción es por defecto `/etc/free-proxy/free-proxy.env` (generado por `free-proxy install`), y las variables de entorno usan de forma unificada el prefijo `FREE_PROXY_`. Todos los subcomandos leen automáticamente ese archivo (las variables de entorno del proceso tienen prioridad; la ruta se puede sobrescribir con `FREE_PROXY_ENV_FILE`). Opciones habituales:

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

> El servicio se enlaza de forma fija a `0.0.0.0`; que esté abierto a Internet lo controla el **interruptor de «acceso desde Internet» del panel** (el panel web abierto por defecto, el proxy cerrado por defecto), y surte efecto de inmediato en tiempo de ejecución, sin necesidad de reiniciar. Establecer `FREE_PROXY_PROXY_USERNAME` / `PASSWORD` es requisito previo para habilitar el acceso al proxy desde Internet.

En VPS de configuración débil (por ejemplo, 1 núcleo / 1 GB) puedes reducir la carga de sondeo:

```text
FREE_PROXY_MAX_PROBE_CONCURRENCY=2
FREE_PROXY_DISCOVERY_LIMIT=60
FREE_PROXY_INITIAL_CONNECT_TEST_LIMIT=5
```

### Resumen de la API

Todos los endpoints están bajo el prefijo de ruta segura: `/{secret_path}/api/v1/...`. Las operaciones de larga duración devuelven `202 + Job`, y se consultan mediante sondeo con `GET /jobs/{id}`.

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

### Pila tecnológica

- **Go 1.23+**, Echo v5 (Web/API), sqlc + `modernc.org/sqlite` (Go puro, sin CGO), goose (migraciones integradas), cobra (CLI), log/slog.
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, cuyos artefactos de compilación se integran en el binario mediante `//go:embed`.
- Contraseñas con hash `scrypt`, autenticación con ruta segura aleatoria + cookie de sesión.

### Compilación desde el código fuente

Requiere Go 1.23+ y bun.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

Como no usa CGO, puedes generar directamente el binario de Linux en macOS. Copia el artefacto compilado localmente a la máquina de destino y ejecuta `sudo ./free-proxy install` para desplegarlo.

Recarga en caliente para desarrollo:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Publicar una Release

`install.sh` descarga `free-proxy-linux-{amd64,arm64}` desde GitHub Releases, y `.github/workflows/release.yml` los compila y publica automáticamente al **enviar una etiqueta de versión (tag)**:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

La etiqueta debe empezar por `v`. Una vez completada la publicación, la descarga `latest` de `install.sh` apuntará a ese binario.

### Estructura del proyecto

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

## 📄 Aviso legal

- Este proyecto es únicamente para fines de aprendizaje e intercambio y para **usos legales**; respeta las leyes y normativas de tu región y no lo utilices para ninguna actividad ilegal.
- Los nodos gratuitos los proporciona un tercero (VPNGate); su disponibilidad y seguridad no están garantizadas por este proyecto, así que **no transmitas información sensible a través de nodos gratuitos**.
- Los enlaces del VPS, la tarjeta de crédito virtual, el bot de Telegram, etc., mencionados en el texto son enlaces de promoción / recomendación (afiliados); comprar a través de ellos puede reportar al autor una pequeña comisión, **sin coste adicional para ti**, gracias por tu apoyo ❤️

## 🙏 Agradecimientos y referencias

Este proyecto se ha inspirado, tanto en su enfoque de diseño como en su implementación, en el proyecto de código abierto **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**, al que damos un agradecimiento especial 🙏

## License

Consulta [LICENSE](LICENSE).
