#!/bin/bash
#============================================================================
#  ccldproxy - Instalador
#  Servidor de tuneles SSH/OpenVPN/V2Ray (proxy Go propio) + menu (dtmenu)
#  Uso personal / amigos. Codigo 100% propio.
#
#  Instalacion por enlace:
#    bash <(curl -sL https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main/install.sh)
#============================================================================

set -e
set +e

CCLD_BIN="ccldproxy"
DTMENU_BIN="dtmenu"
BASE_URL="${CCLD_URL:-https://raw.githubusercontent.com/CCLDB2/CCLDProxy/main}"
INSTALL_DIR="/usr/bin"
CONF_DIR="/etc/ccldproxy"
SERVICE="ccldproxy"

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

# --- Instalar dependencias ------------------------------------------------
yellow "Instalando dependencias..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y >/dev/null 2>&1 || true
apt-get install -y --no-install-recommends \
    openssl curl ca-certificates net-tools ufw openssh-server systemd >/dev/null 2>&1 || true

# --- Descargar binarios y scripts desde el repo ---------------------------
green "Descargando archivos desde el repo..."
TMP="$(mktemp -d)"
curl -fsSL -o "$TMP/$CCLD_BIN" "$BASE_URL/bin/$CCLD_BIN"  || { red "Fallo descargando $CCLD_BIN"; exit 1; }
curl -fsSL -o "$TMP/$DTMENU_BIN" "$BASE_URL/scripts/$DTMENU_BIN" || { red "Fallo descargando $DTMENU_BIN"; exit 1; }
curl -fsSL -o "$TMP/user.sh" "$BASE_URL/scripts/user.sh" || true

# --- Instalar -------------------------------------------------------------
green "Instalando en $INSTALL_DIR ..."
install -m 0755 "$TMP/$CCLD_BIN"  "$INSTALL_DIR/$CCLD_BIN"
install -m 0755 "$TMP/$DTMENU_BIN" "$INSTALL_DIR/$DTMENU_BIN"
install -m 0755 "$TMP/user.sh"    "$INSTALL_DIR/user.sh" 2>/dev/null || true
rm -rf "$TMP"

# --- Config de ejemplo ----------------------------------------------------
mkdir -p "$CONF_DIR"
if [[ ! -f "$CONF_DIR/config" ]]; then
  cat > "$CONF_DIR/config" <<EOF
TOKEN=
PORT=80
MAXIP=0
QUIET=no
EOF
  chmod 600 "$CONF_DIR/config"
fi

# --- Instalar OpenSSH server y configurar ---------------------------------
green "Instalando/configurando OpenSSH..."
apt-get install -y --no-install-recommends openssh-server >/dev/null 2>&1 || true
SSHD_CFG="/etc/ssh/sshd_config"
grep -q '^Port 22' "$SSHD_CFG" 2>/dev/null || sed -i 's/^#Port 22/Port 22/' "$SSHD_CFG" 2>/dev/null || true
sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/' "$SSHD_CFG" 2>/dev/null || true
grep -q '^PasswordAuthentication' "$SSHD_CFG" 2>/dev/null || echo "PasswordAuthentication yes" >> "$SSHD_CFG"
systemctl restart ssh >/dev/null 2>&1 || service ssh restart >/dev/null 2>&1 || true

# --- Firewall -------------------------------------------------------------
green "Abriendo puerto 80 ..."
ufw allow 80/tcp >/dev/null 2>&1 || true
ufw allow 22/tcp >/dev/null 2>&1 || true
ufw reload >/dev/null 2>&1 || true

# --- Resumen --------------------------------------------------------------
PUBLIC_IP="$(curl -fsSL ipv4.icanhazip.com 2>/dev/null || hostname -I | awk '{print $1}')"
green ""
green "Instalacion completa."
echo "  - Motor:        $INSTALL_DIR/$CCLD_BIN"
echo "  - Menu:         $INSTALL_DIR/$DTMENU_BIN"
echo "  - Config:       $CONF_DIR/config"
echo "  - IP:           $PUBLIC_IP"
echo ""
green "Para configurar y arrancar, ejecuta:"
bold   "  dtmenu"
echo ""
echo "  (abre el menu: token, puerto, limite por IP, arrancar servicio)"
