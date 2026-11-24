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
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
boot=${MK_BOOT:-$R/boot}
defaults=${MK_LIMINE_DEFAULTS:-$R/etc/default/limine}

if [[ -f $defaults ]] && grep -q '^MAX_SNAPSHOT_ENTRIES=5$' "$defaults"; then
  sed -i 's/^MAX_SNAPSHOT_ENTRIES=5$/MAX_SNAPSHOT_ENTRIES=auto/' "$defaults"
fi

conf=$boot/limine.conf
[[ -f $boot/EFI/limine/limine.conf ]] && conf=$boot/EFI/limine/limine.conf
command -v limine-update >/dev/null || exit 0
[[ -f $conf ]] || exit 0
# Só mexe se a entrada fixa do instalador está lá.
grep -q '^[[:space:]]*path: boot():/vmlinuz-linux[[:space:]]*$' "$conf" || exit 0

# Toda entrada aponta para arquivos que existem na ESP, e há kernel? (o "#resumo" no fim
# do caminho é da verificação do limine, não do nome)
conf_ok() {
  local file n=0
  while read -r file; do
    file=${file%%#*}
    [[ -f $boot/$file ]] || return 1
    n=$((n + 1))
  done < <(sed -n 's/^[[:space:]]*\(path\|module_path\|image_path\|kernel_path\): *boot():\/*//p' "$1")
  # E há pelo menos uma entrada de Linux (a "EFI fallback" sozinha não serve).
  ((n > 0)) && grep -q '^[[:space:]]*protocol: linux[[:space:]]*$' "$1"
}

cp -a "$conf" "$conf.mk-antes"
back() {
  cp -a "$conf.mk-antes" "$conf"
  echo "aviso: $1; o menu de boot ficou como estava ($conf)" >&2
  exit 0
}

# Dentro de "/Mk" e "/Mk (fallback)", tira as linhas da entrada fixa (as que
# vêm antes da primeira subentrada "//"). Se o grupo já tem subentradas dos
# ganchos, elas ficam; se não sobra nada, o título sai junto e o
# limine-update escreve o grupo. O resto do arquivo não muda.
awk '
  function flush() { if (head != "" && (!mine || kept)) printf "%s", head; printf "%s", body; head = ""; body = ""; kept = 0 }
  /^\/[^\/]/ { flush(); head = $0 "\n"; mine = ($0 ~ /^\/\+?Mk( \(fallback\))?[[:space:]]*$/); top = 1; next }
  head == "" { print; next }
  {
    if (mine && top && $0 ~ /^[[:space:]]*\/\//) top = 0
    if (mine && top) {
      if ($0 ~ /^[[:space:]]*(protocol|path|module_path|cmdline|kernel_cmdline|comment):/ || $0 ~ /^[[:space:]]*$/) next
    }
    body = body $0 "\n"
    if ($0 !~ /^[[:space:]]*$/) kept = 1
  }
  END { flush() }
' "$conf.mk-antes" >"$conf"

limine-update || back "o limine-update falhou"
conf_ok "$conf" || back "o menu gerado não aponta para um kernel que exista em $boot"
command -v limine-snapper-sync >/dev/null && { limine-snapper-sync || true; }
bash "$share/system/mk/boot/limine-default" "$conf" || true
conf_ok "$conf" || back "o menu ficou sem kernel depois dos ajustes"
