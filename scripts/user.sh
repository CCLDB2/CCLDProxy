#!/bin/bash
#============================================================================
#  ccldproxy - Gestion de usuarios SSH
#  Uso:  user.sh [crear|listar|borrar|nombre|vencer]
#  Usado por el menu dtmenu (opcion 9).
#  Usuarios con fecha de expiracion se registran en /etc/ccldproxy/users.txt
#============================================================================

USERS_DB="/etc/ccldproxy/users.txt"
AUTO_ADD=yes   # al crear, pedir fecha de vencimiento

bold()   { echo -e "\033[1m$1\033[0m"; }
green()  { echo -e "\033[1;32m$1\033[0m"; }
yellow() { echo -e "\033[1;33m$1\033[0m"; }
red()    { echo -e "\033[1;31m$1\033[0m"; }
cyan()   { echo -e "\033[1;36m$1\033[0m"; }

[[ "$(id -u)" != "0" ]] && { red "Requiere root."; exit 1; }
mkdir -p "$(dirname "$USERS_DB")"
[[ -f "$USERS_DB" ]] || touch "$USERS_DB"

now() { date +%s; }
expir() { date -d "$1" +%s 2>/dev/null; }
fmt() { date -d "@$1" +%d/%m/%Y 2>/dev/null; }

# Almacenar: usuario:YYYY-MM-DD (fecha vencimiento) o usuario:0 (sin vencer)
set_expiry() { # user fecha(YYYY-MM-DD) o 0
  local u="$1" f="$2"
  grep -v "^$u:" "$USERS_DB" > "$USERS_DB.tmp" 2>/dev/null; mv "$USERS_DB.tmp" "$USERS_DB"
  echo "$u:$f" >> "$USERS_DB"
}
get_expiry() { # user -> fecha
  grep "^$1:" "$USERS_DB" 2>/dev/null | cut -d: -f2
}
user_expired() { # user -> 0 no expira / 1 expirado / 2 vigente
  local u="$1" f; f="$(get_expiry "$u")"
  [[ -z "$f" || "$f" = "0" ]] && { echo 0; return; }
  local e; e="$(expir "$f")"
  [[ -z "$e" ]] && { echo 0; return; }
  [[ "$(now)" -gt "$e" ]] && echo 1 || echo 2
}
lock_user() { usermod -L "$1" 2>/dev/null; }   # bloquear (deshabilitar login)
unlock_user() { usermod -U "$1" 2>/dev/null; }

# Si un usuario vencio, bloquearlo
apply_expiry() {
  local u s
  for u in $(awk -F: '$3>=1000 && $3!=65534 && $7~/bash/ {print $1}' /etc/passwd); do
    s="$(user_expired "$u")"
    [[ "$s" = "1" ]] && lock_user "$u"
    [[ "$s" = "2" ]] && unlock_user "$u"
  done
}
apply_expiry

read_date() { # prompt -> YYYY-MM-DD o 0
  local prompt="$1" d e
  while :; do
    read -rp "$prompt" d
    [[ -z "$d" ]] && d="0"
    [[ "$d" = "0" ]] && { echo 0; return; }
    e="$(expir "$d")"
    [[ -n "$e" ]] && { echo "$d"; return; }
    red "Formato invalido. Usa YYYY-MM-DD o 0 (sin vencer)."
  done
}

cmd="${1:-crear}"

case "$cmd" in
  crear|add)
    read -rp "Nombre de usuario: " u
    [[ -z "$u" ]] && { red "Nombre vacio."; exit 1; }
    id -u "$u" >/dev/null 2>&1 && { red "El usuario '$u' ya existe."; exit 1; }
    useradd -m -s /bin/bash "$u" 2>/dev/null || { red "No se pudo crear."; exit 1; }
    read -rsp "Password: " p; echo
    [[ -z "$p" ]] && { red "Password vacia."; userdel -r "$u" 2>/dev/null; exit 1; }
    echo "$u:$p" | chpasswd
    if [[ "$AUTO_ADD" = "yes" ]]; then
      f="$(read_date "Fecha de expiracion (YYYY-MM-DD) o 0: ")"; set_expiry "$u" "$f"
    fi
    green "Usuario '$u' creado." 
    ;;
  listar|list)
    green "=== Usuarios SSH ==="
    printf "%-20s %-14s %-12s\n" "USUARIO" "VENCE" "ESTADO"
    for u in $(awk -F: '$3>=1000 && $3!=65534 && $7~/bash/ {print $1}' /etc/passwd); do
      f="$(get_expiry "$u")"; s="$(user_expired "$u")"
      [[ -z "$f" || "$f" = "0" ]] && f="sin vencer"
      [[ -n "$(fmt "$(expir "$f")")" ]] && f="$(fmt "$(expir "$f")")"
      case "$s" in 1) st="EXPIRADO";; 2) st="vigente";; *) st="vigente";; esac
      printf "%-20s %-14s %-12s\n" "$u" "$f" "$st"
    done
    ;;
  borrar|del)
    read -rp "Usuario a borrar: " u
    id -u "$u" >/dev/null 2>&1 || { red "No existe."; exit 1; }
    read -rp "Borrar '$u'? (s/N): " ok
    [[ "$ok" =~ ^[sSyY]$ ]] || { yellow "Cancelado."; exit 0; }
    userdel -r "$u" 2>/dev/null && { sed -i "/^$u:/d" "$USERS_DB"; green "'$u' borrado."; } || red "No se pudo borrar."
    ;;
  pass)
    read -rp "Usuario: " u
    id -u "$u" >/dev/null 2>&1 || { red "No existe."; exit 1; }
    read -rsp "Nueva password: " p; echo
    echo "$u:$p" | chpasswd && green "Password actualizada." || red "Falló."
    ;;
  nombre|rename)
    read -rp "Usuario actual: " old
    id -u "$old" >/dev/null 2>&1 || { red "No existe."; exit 1; }
    read -rp "Nuevo nombre: " new
    [[ -z "$new" ]] && { red "Nombre vacio."; exit 1; }
    id -u "$new" >/dev/null 2>&1 && { red "Ya existe '$new'."; exit 1; }
    usermod -l "$new" "$old" 2>/dev/null && usermod -d "/home/$new" -m "$new" 2>/dev/null
    f="$(get_expiry "$old")"; grep -v "^$old:" "$USERS_DB" > "$USERS_DB.tmp" 2>/dev/null; mv "$USERS_DB.tmp" "$USERS_DB"
    [[ -n "$f" ]] && echo "$new:$f" >> "$USERS_DB"
    green "'$old' renombrado a '$new'."
    ;;
  vence|vencer|expiry)
    read -rp "Usuario: " u
    id -u "$u" >/dev/null 2>&1 || { red "No existe."; exit 1; }
    f="$(read_date "Nueva fecha (YYYY-MM-DD) o 0: ")"
    set_expiry "$u" "$f"; apply_expiry
    green "Fecha de '$u' actualizada."
    ;;
  *) echo "Uso: user.sh [crear|listar|borrar|pass|nombre|vence]";;
esac
