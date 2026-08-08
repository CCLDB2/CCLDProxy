# ccldproxy

Servidor de túneles SSH/OpenVPN/V2Ray, reescrito en **Go** desde cero. Código 100% propio.

## Contenido

- `bin/ccldproxy` — motor de túneles (Go). Detecta protocolo por el payload inicial y reenvía raw TCP al backend (SSH:22, OpenVPN:1194, V2Ray:1080).
- `scripts/dtmenu` — menú de configuración (se ejecuta como `dtmenu` en root).
- `scripts/user.sh` — gestión de usuarios SSH.
- `src/` — código fuente Go del motor.
- `install.sh` — instalador por enlace.

## Instalación por enlace

```bash
bash <(curl -sL https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main/install.sh)
```

Luego, para configurar y arrancar:

```bash
dtmenu
```

## Uso manual del motor

```bash
# abrir túnel en el puerto 80
sudo ccldproxy --token MI_CLAVE --port 80

# con límite de 20 conexiones por IP
sudo ccldproxy --token MI_CLAVE --port 80 --max-per-ip 20

# HTTPS (con certificado y clave separados)
sudo ccldproxy --token MI_CLAVE --port 443 --https --cert cert.pem --key key.pem

# modo silencioso
sudo ccldproxy --token MI_CLAVE --port 80 --quiet
```

## Opciones del motor

| Flag | Descripción |
|------|-------------|
| `--token` | token de acceso |
| `--port` | puerto de escucha (default 80) |
| `--max-per-ip` | límite de conexiones por IP (0 = sin límite) |
| `--https` | usar HTTPS (con `--cert` y `--key`) |
| `--cert` / `--key` | certificado y clave separados |
| `--workers` | número de trabajadores (default 1000) |
| `--quiet` | modo silencioso |
| `--validate` | validar token y salir |

## Requisitos

- Ubuntu/Debian.
- El motor Go es **binario estático** (no necesita libssl/libcurl).
- Ejecutar como root.

## Mejoras vs. el binario original

- Certificado y clave **separados** (`--cert` + `--key`).
- **Timeout** de conexión al backend (no se cuelga).
- Límite opcional de conexiones por IP (`--max-per-ip`, anti-DoS).
- Logs con nivel y modo `--quiet`.
- Configuración por servicio systemd vía `dtmenu`.
