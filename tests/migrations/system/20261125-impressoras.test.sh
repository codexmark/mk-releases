#!/usr/bin/env bash
# migrations/system/20261125-impressoras.sh: o grupo wheel administra o CUPS e
# o cups.socket fica ligado; os pacotes quem instala é o mk-update.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261125-impressoras.sh

conf=$R/etc/cups/cups-files.conf
stub systemctl
cups_conf() { mkdir -p "$R/etc/cups" && printf '%s\n' "$@" >"$conf"; }

t "declara cups, ipp-usb e jq no cabeçalho"
eq "cups ipp-usb jq" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "sem o cups (não veio): avisa, não mexe em nada e sai com 0"
cups_conf "SystemGroup sys root"
sem_mudanca mig
has "$(cat "$ERR")" "aviso"
not_called "systemctl"
not_called "pacman -S"

t "com o cups: o wheel entra no SystemGroup, o serviço reinicia e o soquete é ligado"
instalados cups ipp-usb jq
cups_conf "# cups-files.conf" "SystemGroup sys root" "AccessLog syslog"
ok mig
eq $'# cups-files.conf\nSystemGroup sys root wheel\nAccessLog syslog' "$(cat "$conf")"
called "systemctl try-restart cups.service"
called "systemctl enable --now cups.socket"
not_called "pacman -S"

t "de novo: o wheel não é repetido e o serviço não reinicia outra vez"
calls_reset
sem_mudanca mig
eq 1 "$(grep -o wheel "$conf" | wc -l)"
not_called "try-restart"
called "systemctl enable --now cups.socket"

t "um SystemGroup que já tem o wheel (no meio da linha) fica como está"
calls_reset
cups_conf "SystemGroup wheel sys root"
sem_mudanca mig
not_called "try-restart"

t "sem linha SystemGroup: acrescenta uma com o wheel"
cups_conf "AccessLog syslog"
ok mig
eq $'AccessLog syslog\nSystemGroup sys root wheel' "$(cat "$conf")"

t "um grupo com wheel só no nome (wheelbarrow) não conta como wheel"
cups_conf "SystemGroup wheelbarrow"
ok mig
eq "SystemGroup wheelbarrow wheel" "$(cat "$conf")"

t "PeerCred off: avisa que vai pedir senha, e segue"
cups_conf "SystemGroup sys root wheel" "PeerCred off"
ok mig
has "$(cat "$ERR")" "PeerCred off"
called "systemctl enable --now cups.socket"

t "com o cups mas sem o cups-files.conf: só liga o soquete"
calls_reset
rm -rf "$R/etc/cups"
ok mig
absent "$conf"
not_called "try-restart"
called "systemctl enable --now cups.socket"

t "se o systemctl falhar, a migração falha (fica pendente para a próxima)"
stub systemctl 'exit 1'
fails mig
