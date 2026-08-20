**🌐 Languages:** [中文](README.md) · [English](README.en.md) · [Deutsch](README.de.md) · [Español](README.es.md) · [العربية](README.ar.md) · [Italiano](README.it.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# 🚀 Free Proxy — Crea tu propio pool de proxies gratis con un solo comando

> Ejecuta **un solo comando** en un pequeño VPS en el extranjero y automáticamente obtiene cientos de salidas gratuitas desde fuentes públicas de nodos (VPNGate), mide su velocidad de forma real, elige de manera inteligente la ruta más rápida y ofrece hacia el exterior un **proxy SOCKS5 / HTTP** estable. Si un nodo se cae, cambia solo, sin que tengas que estar pendiente.

> 🎥 Video de demostración: [YouTube](https://youtu.be/0uf9St0cBM8)

<p>
  <img alt="Despliegue con un comando" src="https://img.shields.io/badge/Despliegue-un%20comando-brightgreen">
  <img alt="Go binario único" src="https://img.shields.io/badge/Go-binario%20único·sin%20dependencias-00ADD8">
  <img alt="Gratis" src="https://img.shields.io/badge/Nodos-gratis·medición%20automática-orange">
</p>

**¿Para quién es?**

- Para quien quiere una salida de proxy **propia y bajo su control**, en lugar de entregar su tráfico al panel de otra persona.
- Para quien tiene (o piensa comprar) un VPS en el extranjero y quiere convertirlo en una pasarela de proxy totalmente automática.
- Para quien no quiere lidiar con configuraciones complejas: **se instala con un comando y se usa con unos pocos clics en la web**.

---

## ✨ Puntos clave

- 🔌 **Despliegue con un solo comando**: dependencias, servicio y arranque automático se resuelven solos; hasta un principiante puede hacerlo.
- 🌍 **Descubrimiento automático + medición real**: obtiene cientos de nodos desde fuentes públicas, prueba de verdad la conectividad y la latencia, y elige el más rápido automáticamente.
- ♻️ **Cambio automático al caerse**: ¿nodos gratis inestables? Reconecta y cambia solo en segundo plano para mantener el proxy siempre en línea.
- 🧩 **SOCKS5 / HTTP en el mismo puerto**: un único puerto `9527` sirve para todo, reconociendo el protocolo por el primer byte.
- 🖥️ **Panel web sencillo**: pool de nodos, estado de la pasarela, registros y estrategia, todo en una sola pantalla.
- 📦 **Un solo archivo, sin dependencias**: un único binario estático con el frontend y la base de datos embebidos, listo para correr al instante.

---

## 🛒 Antes de empezar: prepara estas dos cosas (imprescindible para principiantes)

### 1️⃣ Un VPS Linux en el extranjero (el llamado "pollito")

Esta herramienta necesita correr en un **servidor Linux en el extranjero** (requiere root y soporte para TUN). Para novatos se recomiendan estos dos proveedores, ambos aceptan **Alipay** y funcionan nada más contratarlos:

| Recomendación | Ideal para | Características | Enlace |
|---|---|---|---|
| **BandwagonHost (搬瓦工)** | 🔰 Principiantes / relación calidad-precio | Marca veterana y estable, precios accesibles, acepta Alipay, con opción de línea premium CN2 GIA, listo para usar | **[Comprar aquí 👉](https://cutt.ly/qywJNWzd)** |
| **DMIT** | 🚀 Los que buscan velocidad / gama alta | Optimización de primer nivel para las tres operadoras / línea CN2 GIA, baja latencia y alta velocidad, experiencia al máximo | **[Comprar aquí 👉](https://cutt.ly/YywJIzY0)** |

> 💡 Presupuesto ajustado y sin complicaciones → elige **[BandwagonHost](https://cutt.ly/qywJNWzd)**; si quieres máxima velocidad y calidad de línea → elige **[DMIT](https://cutt.ly/YywJIzY0)**.
> Como sistema elige **Ubuntu / Debian** (este tutorial usa ese ejemplo) y como plan elige KVM (que soporta TUN por defecto).

### 2️⃣ Una "tarjeta" con la que puedas pagar

La mayoría de los VPS en el extranjero requieren tarjeta de crédito / PayPal. **¿No tienes una tarjeta de crédito internacional?** Con una **tarjeta de crédito virtual internacional** puedes abrir una en pocos minutos y suscribirte fácilmente a todo tipo de servicios en el extranjero (VPS, ChatGPT, streaming, software por suscripción, etc.):

> 💳 **[Tarjeta de crédito virtual internacional · acceso rápido para abrir una 👉](https://cutt.ly/IyrMR4Mg)**

---

## ⚡ Despliegue en tres pasos (versión de verdad para principiantes)

Supongamos que ya compraste el VPS y tienes la **IP del servidor** y la **contraseña de root**.

**Paso 1 · Conéctate por SSH a tu VPS**

```bash
ssh root@你的服务器IP
```

**Paso 2 · Instala con un solo comando**

```bash
bash <(curl -Ls https://raw.githubusercontent.com/masteralanlab/free-proxy/main/install.sh)
```

El script hará automáticamente: descargar el programa para la arquitectura correspondiente → instalar las dependencias del sistema (openvpn, etc.) → registrar el servicio de arranque automático → iniciarlo. Solo espera a que termine, sin ninguna interacción de tu parte.

**Paso 3 · Anota la dirección de administración y el usuario/contraseña**

Después de la primera instalación, el script **imprimirá directamente** la ruta, el usuario y la contraseña generados aleatoriamente:

```text
URL:       http://<你的服务器IP>:39527/xxxxxxxxxxxx/
Username:  xxxxxxxx
Password:  xxxxxxxx
```

> 🔑 La ruta, el usuario y la contraseña se generan aleatoriamente solo en la **primera instalación**. Guárdalos de inmediato porque la contraseña no se puede recuperar después.
> 🔒 Las actualizaciones posteriores conservan la ruta, el usuario y la contraseña. Para cambiarlos expresamente, usa el panel o ejecuta `free-proxy install --rotate-admin`.

✅ **¡Listo!** El servicio ya está en segundo plano obteniendo nodos, midiendo velocidad y conectándose automáticamente. Ahora veamos cómo usarlo.

---

## 🌐 Cómo usar el proxy / acceder al panel web

El servicio escucha por defecto en `0.0.0.0` e incluye un **interruptor de "acceso desde internet"** (se puede cambiar en cualquier momento en la página «Estrategia» del panel, con **efecto inmediato y sin reiniciar**). El acceso local y por túnel SSH **está siempre disponible** y no se ve afectado por el interruptor.

### Panel web: acceso desde internet permitido por defecto ✅

Cuenta con doble protección: inicio de sesión + ruta con clave aleatoria, así que puedes abrirlo desde internet nada más instalarlo. Accede en el navegador a la dirección que imprime `free-proxy credentials`:

```text
http://你的服务器IP:39527/<你的安全路径>/
```

Si no necesitas acceso público, puedes desactivar su interruptor de acceso desde internet en el panel, o usar un túnel SSH (ver más abajo).

### Puerto del proxy: solo local por defecto 🔒

Para evitar convertirse en un **«proxy abierto»** que cualquiera pueda usar, el proxy solo sirve al equipo local por defecto. Para usarlo desde internet, dos pasos:

1. **Configure proxy credentials in the dashboard**: open “Policy → Web and proxy service”, then enter a proxy username and a new password.
2. **Actívalo en el panel**: entra en «Estrategia → Acceso desde internet» del panel web, marca «Permitir acceso al puerto del proxy desde internet» y guarda.

Después ya puedes usarlo en las aplicaciones de tu equipo local: `socks5://用户名:密码@你的服务器IP:9527`.

> 🔒 El uso más conservador (sin abrir nada al público): desactiva en el panel el acceso desde internet del panel web y usa un túnel SSH en su lugar:
> `ssh -L 39527:127.0.0.1:39527 -L 9527:127.0.0.1:9527 root@你的服务器IP`, y luego accede localmente a `127.0.0.1`.

### Verificar si el proxy funciona

```bash
curl --proxy socks5h://127.0.0.1:9527 https://api.ipify.org   # 返回的应是"VPN 出口 IP",而不是你 VPS 自己的 IP
curl --proxy http://127.0.0.1:9527   https://api.ipify.org
```

Si ves una IP distinta a la de tu VPS, significa que el proxy ya está reenviando el tráfico a través de la salida de la VPN 🎉

---

## 🖱️ Cómo usar el panel web

1. Abre la dirección de administración e inicia sesión con tu usuario y contraseña.
2. Haz clic en **«Actualizar y comprobar nodos»** y espera un momento a que descubra, mida la velocidad y se conecte automáticamente al nodo más rápido.
3. El panel **«Pasarela»** muestra el nodo de salida actual, la latencia y la IP de salida.
4. Apunta el proxy de las aplicaciones de tu equipo a `127.0.0.1:9527` y ya puedes empezar a usarlo.

---

## 🔧 Comandos habituales

```bash
free-proxy credentials   # 查看管理网址与账号密码
free-proxy status        # 查看运行状态
free-proxy logs -n 100   # 查看最近日志
free-proxy uninstall     # 卸载(加 --purge-data 连数据一起删除)
```

**Actualizar a la última versión**: basta con volver a ejecutar el «comando único de instalación» de arriba. Se conservan los datos, la configuración, la ruta de administración, el usuario y la contraseña.

---

## ❓ Preguntas frecuentes

- **¿No conecta / temporalmente sin nodos?** Los nodos gratuitos (VPNGate) fluctúan por naturaleza; el servicio reintentará y cambiará automáticamente. Espera un poco más, o haz clic una vez en «Actualizar y comprobar nodos» en el panel.
- **¿Avisa de que necesita root / TUN?** Ejecútalo como root y confirma que el VPS tiene TUN/TAP activado. **[BandwagonHost](https://cutt.ly/qywJNWzd)** / **[DMIT](https://cutt.ly/YywJIzY0)** son ambos de arquitectura KVM, lo soportan por defecto y funcionan de inmediato.
- **¿Mi VPS es de arquitectura ARM?** No te preocupes, el script de instalación detecta automáticamente amd64 / arm64.
- **¿Puedo ejecutarlo en mi propio ordenador (macOS/Windows)?** Puedes compilarlo y desarrollar, pero el túnel real y el proxy de salida requieren Linux + root + TUN, así que despliégalo en un VPS.

---

## 🧰 Herramientas y recursos recomendados

- 🔎 **El mejor bot de búsqueda de Telegram** — una maravilla para encontrar películas, software, libros electrónicos y todo tipo de recursos, con una sola búsqueda: 👉 **[Abrir](https://cutt.ly/2yeh3GOE)**
- 🖥️ ¿Todavía no tienes servidor? **[BandwagonHost (calidad-precio para principiantes)](https://cutt.ly/qywJNWzd)** · **[DMIT (líneas de gama alta)](https://cutt.ly/YywJIzY0)**
- 💳 ¿No tienes tarjeta internacional para pagar? **[Tarjeta de crédito virtual internacional](https://cutt.ly/IyrMR4Mg)**

---

## 🛡️ Recomendaciones de seguridad

- El servicio escucha por defecto en `0.0.0.0`, y su exposición se controla mediante el interruptor de «acceso desde internet» del panel: **el panel web está abierto por defecto** (con protección de inicio de sesión + ruta con clave), y **el proxy es solo local por defecto**. Cuando no necesites acceso público, puedes desactivar el acceso desde internet del panel web y usar un túnel SSH en su lugar.
- **Antes de habilitar el acceso al proxy desde internet debes configurar primero el usuario y la contraseña del proxy**, de lo contrario se convertirá en un «proxy abierto» que cualquiera puede usar, muy fácil de abusar y que puede provocar el baneo del VPS; por eso el sistema rechaza toda petición de proxy desde internet mientras no haya contraseña configurada.
- Después del primer inicio de sesión, cambia cuanto antes el usuario y la contraseña de administración. El túnel, el enrutamiento por estrategia y la instalación de dependencias requieren root, así que actívalos solo en servidores que tú mismo controles.

---

## 🧑‍💻 Uso avanzado (para desarrolladores)

<details>
<summary>Haz clic para expandir: línea de comandos / opciones de configuración / API / compilación desde el código fuente / publicación / estructura del proyecto</summary>

### Instalación manual (sin el script)

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

En producción, el archivo de configuración es por defecto `/etc/free-proxy/free-proxy.env` (generado por `free-proxy install`), y las variables de entorno usan de forma uniforme el prefijo `FREE_PROXY_`. Todos los subcomandos leen ese archivo automáticamente (las variables de entorno del proceso tienen prioridad; la ruta se puede sobrescribir con `FREE_PROXY_ENV_FILE`). Opciones habituales:

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

En un "pollito" de configuración débil (por ejemplo 1 núcleo / 1 GB) puedes reducir la carga de sondeo:

Use the dashboard to lower probe concurrency, discovery limit, and initial test count.

### Resumen de la API

Todos los endpoints están bajo el prefijo de la ruta segura: `/{secret_path}/api/v1/...`. Las operaciones de larga duración devuelven `202 + Job`, que se consulta mediante `GET /jobs/{id}`.

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

### Stack tecnológico

- **Go 1.23+**, Echo v5 (Web/API), sqlc + `modernc.org/sqlite` (Go puro, sin CGO), goose (migraciones embebidas), cobra (CLI), log/slog.
- Frontend **React 19 + Vite + Tailwind v4 + Zustand**, cuyo resultado de compilación se embebe en el binario mediante `//go:embed`.
- Contraseñas con hash `scrypt`, autenticación mediante ruta segura aleatoria + cookie de sesión.

### Compilar desde el código fuente

Requiere Go 1.23+ y bun.

```bash
make build        # 构建前端 + 静态二进制到 dist/free-proxy
make cross        # 交叉编译 linux amd64 / arm64
make test         # 运行 Go 测试
```

Como no hay CGO, puedes producir directamente un binario de Linux en macOS. Copia el resultado de la compilación local a la máquina de destino y ejecuta `sudo ./free-proxy install` para desplegarlo.

Recarga en caliente durante el desarrollo:

```bash
cd frontend && bun install && bun run dev   # 前端热更新(配合下方 serve)
go run ./cmd/free-proxy serve                # 后端(首次会生成随机管理地址与密码)
```

### Publicar una Release

`install.sh` descarga `free-proxy-linux-{amd64,arm64}` desde GitHub Releases, y `.github/workflows/release.yml` la construye y publica automáticamente al **empujar una etiqueta de versión**:

```bash
git tag v1.0.0
git push origin v1.0.0      # 触发 Action:构建前端 + 交叉编译 → 发布 Release(含 SHA256SUMS)
```

La etiqueta debe empezar por `v`. Una vez publicada la Release, la descarga `latest` de `install.sh` apuntará a ese binario.

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

- Este proyecto es solo para aprendizaje, intercambio y **usos legales**; respeta las leyes y normativas de tu región y no lo utilices para ninguna actividad ilegal.
- Los nodos gratuitos los proporciona un tercero (VPNGate); su disponibilidad y seguridad no están garantizadas por este proyecto, así que **no transmitas información sensible a través de nodos gratuitos**.
- Los enlaces de VPS, tarjetas de crédito virtuales, bot de Telegram, etc. mencionados en el texto son enlaces de promoción / recomendación (afiliados); realizar un pedido a través de ellos puede reportar al autor una pequeña comisión, **sin coste adicional para ti**, gracias por tu apoyo ❤️

## 🙏 Agradecimientos y referencias

Este proyecto se inspiró, en su enfoque de diseño e implementación, en el proyecto de código abierto **[aimili-vpngate](https://github.com/baoweise-bot/aimili-vpngate)**; nuestro especial agradecimiento 🙏

## License

Consulta [LICENSE](LICENSE).
