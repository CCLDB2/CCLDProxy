#!/bin/bash
#============================================================================
#  ccldproxy - Instalador
#  Servidor de tuneles SSH/OpenVPN/V2Ray (proxy) + menu (ccldproxy)
#  Uso personal / amigos. No incluye panel SSHPlus.
#
#  Instalacion por enlace:
#    bash <(curl -sL https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main/install.sh)
#============================================================================

set -e

CCLD_BIN="ccldproxy"
PROXY_BIN="proxy"
BASE_URL="${CCLD_URL:-https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main}"
INSTALL_DIR="/usr/bin"
LISTEN_PORT="${CCLD_PORT:-80}"

bold()   { echo -e "\033[1m$1\033[0m"; }
green()  { echo -e "\033[1;32m$1\033[0m"; }
yellow() { echo -e "\033[1;33m$1\033[0m"; }
red()    { echo -e "\033[1;31m$1\033[0m"; }

# --- Requiere root --------------------------------------------------------
if [[ "$(id -u)" != "0" ]]; then
  red "Este instalador debe ejecutarse como root."
  exit 1
fi

# --- Detectar distro ------------------------------------------------------
if   grep -qi 'ubuntu' /etc/os-release; then DISTRO="ubuntu"
elif grep -qi 'debian' /etc/os-release; then DISTRO="debian"
else
  red "Distro no soportada. Solo Ubuntu/Debian."
  exit 1
fi
CODENAME="$(grep -oP '(?<=VERSION_CODENAME=).*' /etc/os-release)"
green "Detectado: $DISTRO ($CODENAME)"
green "Leyendo checksums de version remota..."
REMOTE_VER="$(curl -fsSL "$BASE_URL/version" 2>/dev/null || echo 'dev')"
echo "  version remota: $REMOTE_VER"

# --- Instalar dependencias ------------------------------------------------
yellow "Instalando dependencias..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y >/dev/null 2>&1 || true
# OpenSSL 1.1 y curl: proxy enlaza contra libssl.so.1.1 / libcurl.so.4
if [[ "$DISTRO" == "ubuntu" && "$CODENAME" == "jammy" ]]; then
  # 22.04 no trae libssl1.1 en repos: bajar paquete de focal
  green "Ubuntu 22.04: instalando libssl1.1 desde focal..."
  curl -fsSL -o /tmp/libssl1.1.deb \
    "http://archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2.20_amd64.deb" \
    || yellow "  (no se pudo bajar libssl1.1, continuando)"
  dpkg -i /tmp/libssl1.1.deb >/dev/null 2>&1 || apt-get install -y -f >/dev/null 2>&1 || true
fi
apt-get install -y --no-install-recommends \
    openssl libcurl4 libstdc++6 curl ca-certificates net-tools ufw >/dev/null 2>&1 || true

# --- Descargar binarios desde el repo --------------------------------------
green "Descargando binarios desde el repo..."
TMP="$(mktemp -d)"
for BIN in "$PROXY_BIN" "$CCLD_BIN"; do
  echo "  $BIN -> $BASE_URL/bin/$BIN"
  curl -fsSL -o "$TMP/$BIN" "$BASE_URL/bin/$BIN" || { red "Fallo descargando $BIN"; exit 1; }
done

# Verificar que el proxy podra ejecutarse
ldd "$TMP/$PROXY_BIN" 2>/dev/null | grep -q "not found" && yellow "  (verifica libssl/libcurl con 'ldd proxy')" || true

# --- Instalar binarios ----------------------------------------------------
green "Instalando binarios en $INSTALL_DIR ..."
install -m 0755 "$TMP/$PROXY_BIN" "$INSTALL_DIR/$PROXY_BIN"
install -m 0755 "$TMP/$CCLD_BIN"  "$INSTALL_DIR/$CCLD_BIN"
rm -rf "$TMP"

# --- Configurar sshd (gestor ssh) ----------------------------------------
green "Configurando SSH..."
# Puerto SSH
if grep -q '^#Port 22' /etc/ssh/sshd_config; then
  sed -i 's/^#Port 22/Port 22/' /etc/ssh/sshd_config
fi
# Permitir login por password (para usuarios creados)
sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/' /etc/ssh/sshd_config
grep -q '^PasswordAuthentication' /etc/ssh/sshd_config || echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
# Permitir acceso root por password (opcional, descomenta si lo quieres)
# sed -i 's/^PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

systemctl restart ssh >/dev/null 2>&1 || service ssh restart >/dev/null 2>&1 || true

# --- Guardar IP publica ---------------------------------------------------
PUBLIC_IP="$(curl -fsSL ipv4.icanhazip.com 2>/dev/null || hostname -I | awk '{print $1}')"
echo "$PUBLIC_IP" > /etc/ccldproxy_ip 2>/dev/null || true
green "IP publica: $PUBLIC_IP"

# --- Abrir puerto en firewall --------------------------------------------
green "Abriendo puerto $LISTEN_PORT ..."
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$LISTEN_PORT/tcp" >/dev/null 2>&1 || true
  ufw allow 22/tcp >/dev/null 2>&1 || true
  ufw reload >/dev/null 2>&1 || true
fi

# --- Crear usuario(s) SSH -------------------------------------------------
green ""
green "El servidor de tuneles esta listo."
echo "  - Binario del menu:  $INSTALL_DIR/$CCLD_BIN"
echo "  - Motor proxy:       $INSTALL_DIR/$PROXY_BIN"
echo "  - Puerto de tunel:   $LISTEN_PORT"
echo "  - IP:                $PUBLIC_IP"
echo ""
if [[ "${CCLD_CREATE_USER:-yes}" == "yes" ]]; then
  yellow "Se va a crear un usuario SSH para conectarte."
  read -rp "Nombre de usuario: " SSH_USER
  read -rsp "Password para el usuario: " SSH_PASS; echo
  useradd -m -s /bin/bash "$SSH_USER" 2>/dev/null || true
  echo "$SSH_USER:$SSH_PASS" | chpasswd
  green "Usuario '$SSH_USER' creado."
  echo ""
  green "Para iniciar el proxy manualmente:"
  echo "  sudo $INSTALL_DIR/$PROXY_BIN --token CLAVE --port $LISTEN_PORT"
  echo "  (usa el menu: sudo $INSTALL_DIR/$CCLD_BIN)"
fi

bold ""
bold "Listo. Conectate con tu app a: $PUBLIC_IP:$LISTEN_PORT"
bold "SSH usuario/ip:  $SSH_USER@$PUBLIC_IP (puerto 22)"
