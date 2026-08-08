#!/bin/bash
#============================================================================
#  ccldproxy - Gestion de usuarios SSH
#  Uso:  bash user.sh [crear|listar|borrar|pass]
#============================================================================

bold()   { echo -e "\033[1m$1\033[0m"; }
green()  { echo -e "\033[1;32m$1\033[0m"; }
yellow() { echo -e "\033[1;33m$1\033[0m"; }
red()    { echo -e "\033[1;31m$1\033[0m"; }

[[ "$(id -u)" != "0" ]] && { red "Requiere root."; exit 1; }

cmd="${1:-crear}"

case "$cmd" in
  crear|add)
    read -rp "Nombre de usuario: " u
    id -u "$u" >/dev/null 2>&1 && { yellow "El usuario '$u' ya existe."; exit 0; }
    useradd -m -s /bin/bash "$u"
    read -rsp "Password: " p; echo
    echo "$u:$p" | chpasswd
    green "Usuario '$u' creado."
    ;;
  listar|list)
    green "Usuarios del sistema con shell bash:"
    awk -F: '$3>=1000 && $3!=65534 && $7~/bash/ {print $1}' /etc/passwd
    ;;
  borrar|del)
    read -rp "Usuario a borrar: " u
    userdel -r "$u" 2>/dev/null && green "'$u' borrado." || red "No se pudo borrar '$u'."
    ;;
  pass)
    read -rp "Usuario: " u
    read -rsp "Nueva password: " p; echo
    echo "$u:$p" | chpasswd && green "Password actualizada." || red "Falló."
    ;;
  *) echo "Uso: bash user.sh [crear|listar|borrar|pass]";;
esac
