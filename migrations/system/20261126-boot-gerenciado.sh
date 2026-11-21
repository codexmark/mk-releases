#!/usr/bin/env bash
# O instalador até a versão de 20/11 descartava o menu de boot que os ganchos
# do limine geram e gravava entradas fixas (/Mk → /boot/vmlinuz-linux): o
# submenu de snapshots nunca entrava e, num kernel novo, o menu podia ficar
# no antigo. Aqui a entrada fixa sai e o limine-update escreve a dele. O
# arquivo anterior fica em limine.conf.mk-antes e volta se algo não fechar.
# De carona, o limite de snapshots no menu passa a "auto" (o número fixo
# gerava um aviso a cada sincronização).
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
boot=${MK_BOOT:-/boot}
defaults=${MK_LIMINE_DEFAULTS:-/etc/default/limine}

if [[ -f $defaults ]] && grep -q '^MAX_SNAPSHOT_ENTRIES=5$' "$defaults"; then
  sed -i 's/^MAX_SNAPSHOT_ENTRIES=5$/MAX_SNAPSHOT_ENTRIES=auto/' "$defaults"
fi

conf=$boot/limine.conf
[[ -f $boot/EFI/limine/limine.conf ]] && conf=$boot/EFI/limine/limine.conf
command -v limine-update >/dev/null || exit 0
[[ -f $conf ]] || exit 0
# Só mexe se a entrada fixa do instalador está lá.
grep -q '^[[:space:]]*path: boot():/vmlinuz-linux[[:space:]]*$' "$conf" || exit 0

# Toda entrada aponta para arquivos que existem na ESP? (o "#resumo" no fim
# do caminho é da verificação do limine, não do nome)
conf_ok() {
  local file n=0
  while read -r file; do
    file=${file%%#*}
    [[ -f $boot/$file ]] || return 1
    n=$((n + 1))
  done < <(sed -n 's/^[[:space:]]*\(path\|module_path\|image_path\|kernel_path\): *boot():\/*//p' "$1")
  ((n > 0))
}

cp -a "$conf" "$conf.mk-antes"
back() {
  cp -a "$conf.mk-antes" "$conf"
  echo "aviso: $1; o menu de boot ficou como estava ($conf)" >&2
  exit 0
}

# Tira as entradas de primeiro nível "/Mk" e "/Mk (fallback)" que apontam
# para /vmlinuz-linux; o resto do arquivo (cabeçalho, outras entradas) fica.
awk '
  function flush() { if (held != "" && !drop) printf "%s", held; held = ""; drop = 0 }
  /^\/[^\/]/ { flush(); inblock = 1; mine = ($0 ~ /^\/\+?Mk( \(fallback\))?[[:space:]]*$/) }
  /^\/\// { }
  {
    if (inblock) {
      held = held $0 "\n"
      if (mine && $0 ~ /^[[:space:]]*path: boot\(\):\/vmlinuz-linux[[:space:]]*$/) drop = 1
    } else {
      printf "%s\n", $0
    }
  }
  END { flush() }
' "$conf.mk-antes" >"$conf"

limine-update || back "o limine-update falhou"
conf_ok "$conf" || back "o menu gerado não aponta para um kernel que exista em $boot"
command -v limine-snapper-sync >/dev/null && { limine-snapper-sync || true; }
bash "$share/system/mk/boot/limine-default" "$conf" || true
conf_ok "$conf" || back "o menu ficou sem kernel depois dos ajustes"
