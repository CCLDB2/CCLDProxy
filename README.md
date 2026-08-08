# ccldproxy

Servidor de túneles SSH/OpenVPN/V2Ray (motor `proxy`) con menú (`ccldproxy`) para uso personal / amigos.

## Contenido

- `bin/proxy` — motor de túneles (por defecto `/usr/bin/proxy`).
- `bin/ccldproxy` — menú (renombrado de `dtmenu`), usa `/usr/bin/proxy`.
- `install.sh` — instalador por enlace (dependencias, sshd, usuario SSH, abre puerto 80).
- `scripts/user.sh` — gestión de usuarios SSH (crear/listar/borrar/pass).
- `version` — versión del paquete.

## Instalación por enlace

```bash
bash <(curl -sL https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main/install.sh)
```

O bien:

```bash
curl -sL -o install.sh https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main/install.sh
bash install.sh
```

## Uso manual del motor

```bash
# abrir túnel en el puerto 80
sudo /usr/bin/proxy --token MI_CLAVE --port 80

# menú interactivo
sudo /usr/bin/ccldproxy
```

## Requisitos

- Ubuntu/Debian (probado en Ubuntu 20.04/22.04).
- Los binarios requieren `libssl.so.1.1`, `libcurl.so.4`, `libstdc++6` (los instala `install.sh`).
- Ejecutar como root.

## Notas

- El motor `proxy` acepta token de validación (modo crack local: acepta cualquier token, no valida contra servidor).
- El puerto por defecto es 80. Cambia `CCLD_PORT` si quieres otro.
- Sin límite por IP.
