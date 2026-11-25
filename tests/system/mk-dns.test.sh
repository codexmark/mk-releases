#!/usr/bin/env bash
# system/mk/mk-dns: o ajudante que roda como root (pelo pkexec) e grava o DNS
# global do NetworkManager. Só aceita endereços IP, escreve um arquivo só e
# recarrega o NetworkManager; aqui com uma raiz falsa (MK_ROOT) e o nmcli falso.
. "$(dirname "$0")/../lib.sh"

helper=$REPO/system/mk/mk-dns
R=$SBX/root
conf=$R/etc/NetworkManager/conf.d/90-mk-dns.conf
mkdir -p "$R/etc/NetworkManager/conf.d"
# Como o nmcli de verdade: o "reload" aceita uma coisa de cada vez (visto na VM:
# "argumento extra dns-rc", saída 2).
stub nmcli '[[ "$*" == "--wait 10 general reload conf" || "$*" == "--wait 10 general reload dns-rc" ]] || { echo "Erro: argumento extra" >&2; exit 2; }'
run() { MK_ROOT=$R bash "$helper" "$@"; }
# Tudo o que existe na raiz falsa, com modo e conteúdo.
retrato() { (cd "$R" && find . -mindepth 1 -printf '%p %y %m\n' | sort && find . -type f -print0 | sort -z | xargs -0 -r sha256sum); }

t "o ajudante tem o bit de execução (o pkexec o chama pelo caminho)"
ok test -x "$helper"

t "aplicar: grava a seção global com os servidores separados por vírgula"
out=$(run aplicar 9.9.9.9 149.112.112.112 2620:fe::fe 2620:fe::9 2>&1)
eq 0 $?
eq "" "$out"
exists "$conf"
eq '[global-dns-domain-*]' "$(grep -v '^#' "$conf" | sed -n 1p)"
eq 'servers=9.9.9.9,149.112.112.112,2620:fe::fe,2620:fe::9' "$(grep -v '^#' "$conf" | sed -n 2p)"
eq 2 "$(grep -vc '^#' "$conf")"

t "aplicar: o arquivo é 644 e não sobra temporário na pasta"
eq 644 "$(stat -c %a "$conf")"
eq "90-mk-dns.conf" "$(ls -A "$R/etc/NetworkManager/conf.d")"

t "aplicar: o cabeçalho diz que é do Mk e como desfazer"
head=$(grep '^#' "$conf")
has "$head" "Mk"
has "$head" "Ajustes"
has "$head" "da rede"
has "$head" "nmcli general reload conf"
has "$head" "nmcli general reload dns-rc"

t "aplicar: recarrega a configuração e depois o DNS do NetworkManager, em duas chamadas"
eq $'nmcli --wait 10 general reload conf\nnmcli --wait 10 general reload dns-rc' "$(grep '^nmcli' "$SBX/calls" | tail -2)"

t "aplicar de novo: o arquivo é regravado inteiro, não emendado"
ok run aplicar 10.0.0.53
eq 'servers=10.0.0.53' "$(grep -v '^#' "$conf" | sed -n 2p)"
eq 1 "$(grep -c '^servers=' "$conf")"
eq 1 "$(grep -c '^\[' "$conf")"

t "aplicar: IPv6 em maiúsculas é gravado em minúsculas"
ok run aplicar 2001:DB8::A
eq 'servers=2001:db8::a' "$(grep -v '^#' "$conf" | sed -n 2p)"

t "aplicar: cria a pasta conf.d se faltar"
rm -rf "${R:?}/etc/NetworkManager"
ok run aplicar 1.1.1.1
exists "$conf"

t "limpar: apaga o arquivo e recarrega"
calls_reset
ok run limpar
absent "$conf"
called "nmcli --wait 10 general reload conf"
called "nmcli --wait 10 general reload dns-rc"

t "limpar sem arquivo: sai com 0 e recarrega do mesmo jeito"
calls_reset
ok run limpar
called "nmcli --wait 10 general reload conf"
called "nmcli --wait 10 general reload dns-rc"

# Daqui em diante, o que o ajudante recusa. Antes de cada bloco há um arquivo
# bom gravado: a recusa não pode tocar nele.
ok run aplicar 9.9.9.9
calls_reset
antes=$(retrato)

# Sai com 2, diz por quê e não recarrega nada.
recusa() { # recusa <o que é> <args...>
  local oque=$1 err rc
  shift
  calls_reset
  err=$(run "$@" 2>&1 >/dev/null)
  rc=$?
  not_called "nmcli"
  [[ $rc == 2 ]] || _fail "$oque: devia sair com 2, saiu com $rc"
  [[ -n $err ]] || _fail "$oque: não disse por que recusou"
}

t "texto livre não entra: nome, seção, chave, quebra de linha, espaço, vírgula"
recusa "nome" aplicar dns.quad9.net
recusa "seção" aplicar '[main]'
recusa "chave" aplicar 'servers=1.1.1.1'
recusa "igual" aplicar '1.1.1.1=x'
recusa "colchete" aplicar '[2001:db8::1]'
recusa "quebra de linha" aplicar $'1.1.1.1\n[main]\ndns=none'
recusa "quebra no fim" aplicar $'1.1.1.1\n'
recusa "retorno de carro" aplicar $'1.1.1.1\r'
recusa "tab" aplicar $'1.1.1.1\t8.8.8.8'
recusa "espaço" aplicar '1.1.1.1 8.8.8.8'
recusa "espaço no fim" aplicar '1.1.1.1 '
recusa "vírgula" aplicar '1.1.1.1,8.8.8.8'
recusa "ponto e vírgula" aplicar '1.1.1.1;8.8.8.8'
recusa "comentário" aplicar '1.1.1.1#x'
recusa "URI" aplicar 'dns+tls://1.1.1.1'
recusa "vazio" aplicar ''
recusa "um bom e um ruim" aplicar 9.9.9.9 'x'
eq "$antes" "$(retrato)" "a recusa mexeu na raiz"

t "IPv4 que não é de verdade não entra"
for a in 256.1.1.1 1.1.1 1.1.1.1.1 01.1.1.1 1.1.1.01 1.1.1. .1.1.1 1.1.1.1/24 1.1.1.1:53 -1.1.1.1 0.0.0.0 0.1.2.3 \
  224.0.0.251 255.255.255.255 0x1.1.1.1 1.1.1.1e0; do
  recusa "$a" aplicar "$a"
done
eq "$antes" "$(retrato)" "a recusa mexeu na raiz"

t "IPv6 que não é de verdade não entra"
for a in : :: ::: 2001:db8 1:2:3:4:5:6:7 1:2:3:4:5:6:7:8:9 1::2::3 2001:db8::g 2001:db8::12345 :1:2:3:4:5:6:7 \
  1:2:3:4:5:6:7: 2001:db8:::1 1:2:3:4:5:6:7:8:: ::1:2:3:4:5:6:7:8 2001:db8::1/64 fe80::1%wlan0 ::ffff:1.2.3.4 ff02::fb FF02::1; do
  recusa "$a" aplicar "$a"
done
eq "$antes" "$(retrato)" "a recusa mexeu na raiz"

t "os endereços bons que a página aceita, o ajudante aceita também"
for a in 1.1.1.1 192.168.1.1 127.0.0.1 223.255.255.254 ::1 fd00::53 2001:db8:0:0:0:0:0:1 \
  2001:0db8:0000:0000:0000:0000:0000:0001 2001:db8:: 1:2:3:4:5:6:7:: ::2:3:4:5:6:7:8 ff::1 ff0::1; do
  run aplicar "$a" 2>/dev/null || _fail "recusou $a"
done
ok run aplicar 9.9.9.9
antes=$(retrato)

t "nenhum endereço, ou mais de quatro, não entra"
recusa "nenhum" aplicar
recusa "cinco" aplicar 10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5
eq "$antes" "$(retrato)" "a recusa mexeu na raiz"

t "o repetido entra uma vez só"
ok run aplicar 9.9.9.9 9.9.9.9 2620:FE::FE 2620:fe::fe
eq 'servers=9.9.9.9,2620:fe::fe' "$(grep -v '^#' "$conf" | sed -n 2p)"
ok run aplicar 9.9.9.9
antes=$(retrato)

t "não há opção nem outro verbo: caminho, nome de arquivo e -- são recusados"
recusa "sem verbo"
recusa "verbo desconhecido" gravar 1.1.1.1
recusa "caminho" aplicar /etc/passwd
recusa "volta de pasta" aplicar ../../etc/passwd
recusa "opção" aplicar --arquivo=/etc/shadow 1.1.1.1
recusa "limpar com argumento" limpar /etc/passwd
recusa "limpar com endereço" limpar 1.1.1.1
eq "$antes" "$(retrato)" "a recusa mexeu na raiz"

t "só aquele arquivo: um vizinho em conf.d e o NetworkManager.conf ficam como estavam"
echo '[main]' >"$R/etc/NetworkManager/NetworkManager.conf"
echo '[connection]' >"$R/etc/NetworkManager/conf.d/10-outro.conf"
viz=$(sha256sum "$R/etc/NetworkManager/NetworkManager.conf" "$R/etc/NetworkManager/conf.d/10-outro.conf")
ok run aplicar 8.8.8.8
ok run limpar
eq "$viz" "$(sha256sum "$R/etc/NetworkManager/NetworkManager.conf" "$R/etc/NetworkManager/conf.d/10-outro.conf")"
eq "10-outro.conf" "$(ls -A "$R/etc/NetworkManager/conf.d")"

t "um link no lugar do arquivo não é seguido: o alvo fica intacto"
echo "segredo" >"$R/alvo"
ln -s "$R/alvo" "$conf"
ok run aplicar 8.8.8.8
eq "segredo" "$(cat "$R/alvo")"
ok test ! -L "$conf"
eq 'servers=8.8.8.8' "$(grep -v '^#' "$conf" | sed -n 2p)"
ln -sf "$R/alvo" "$conf"
ok run limpar
absent "$conf"
eq "segredo" "$(cat "$R/alvo")"

t "o NetworkManager não recarrega: o arquivo fica gravado e o ajudante sai com 4"
stub nmcli 'echo "Error: NetworkManager is not running." >&2; exit 8'
err=$(run aplicar 9.9.9.9 2>&1 >/dev/null)
eq 4 $?
has "$err" "NetworkManager"
eq 'servers=9.9.9.9' "$(grep -v '^#' "$conf" | sed -n 2p)"
run limpar 2>/dev/null
eq 4 $?
absent "$conf"

t "sem o nmcli: grava e sai com 4"
PATH=$(path_sem nmcli) MK_ROOT=$R bash "$helper" aplicar 9.9.9.9 2>/dev/null
eq 4 $?
exists "$conf"

t "não dá para gravar (a pasta não aceita escrita): sai com 3 e não recarrega"
stub nmcli
calls_reset
rm -f "$conf"
chmod 555 "$R/etc/NetworkManager/conf.d"
run aplicar 9.9.9.9 2>/dev/null
eq 3 $?
absent "$conf"
not_called "nmcli"
chmod 755 "$R/etc/NetworkManager/conf.d"

t "o ajudante não lê nada do ambiente além da raiz de teste, nem chama shell com o que recebe"
src=$(sed -e '/^[[:space:]]*#/d' "$helper")
hasnt "$src" "eval"
hasnt "$src" "HOME"
hasnt "$src" "sudo"
hasnt "$src" "pkexec"
