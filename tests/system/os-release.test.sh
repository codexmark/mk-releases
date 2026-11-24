#!/usr/bin/env bash
# system/mk/os-release, issue e mk-os-release.hook: o nome do sistema (Mk) e o
# gancho do pacman que o repõe. O gancho nunca roda aqui (ele apaga
# /etc/os-release): é conferido por leitura, e o comando dele numa cópia com
# os caminhos trocados para dentro de $SBX.
. "$(dirname "$0")/../lib.sh"

d=$REPO/system/mk
campo() { sed -n "s/^$1=//p" "$d/os-release" | tr -d '"'; }

t "os-release: o sistema se chama Mk, é da família do Arch e é rolling"
eq Mk "$(campo NAME)"
eq Mk "$(campo PRETTY_NAME)"
eq mk "$(campo ID)"
eq arch "$(campo ID_LIKE)"
eq rolling "$(campo BUILD_ID)"

t "os-release: é shell válido, uma CHAVE=valor por linha (o formato do os-release(5))"
ok bash -n "$d/os-release"
eq "" "$(grep -v -E '^[A-Z][A-Z0-9_]*=("[^"]*"|[A-Za-z0-9._;:/-]+)$' "$d/os-release")"
eq "" "$(cut -d= -f1 "$d/os-release" | sort | uniq -d)"

t "os-release: ID em minúsculas sem espaço; a cor é o cobre do Mk"
like "$(campo ID)" '^[a-z0-9._-]+$'
eq "38;2;206;94;50" "$(campo ANSI_COLOR)"

t "issue: Mk no console de login, com os escapes do agetty"
eq 'Mk \r (\l)' "$(head -1 "$d/issue")"
hasnt "$(<"$d/issue")" "Arch"

t "gancho: dispara na instalação e na atualização do pacote filesystem, depois da transação"
hook=$(<"$d/mk-os-release.hook")
ok grep -qx 'Operation = Install' <<<"$hook"
ok grep -qx 'Operation = Upgrade' <<<"$hook"
ok grep -qx 'Type = Package' <<<"$hook"
ok grep -qx 'Target = filesystem' <<<"$hook"
ok grep -qx 'When = PostTransaction' <<<"$hook"
fails grep -qx 'Operation = Remove' <<<"$hook"

t "gancho: tem as duas seções e só chaves que o alpm-hooks conhece"
ok grep -qx '\[Trigger\]' <<<"$hook"
ok grep -qx '\[Action\]' <<<"$hook"
eq "" "$(grep -v -e '^#' -e '^$' -e '^\[' <<<"$hook" | grep -v -E '^(Operation|Type|Target|Description|When|Depends|Exec|AbortOnFail|NeedsTargets) = ')"
eq 1 "$(grep -c '^Exec = ' <<<"$hook")"

exec_=$(sed -n 's/^Exec = //p' <<<"$hook")

t "gancho: o comando usa caminhos inteiros e os arquivos que ele copia existem no repositório"
like "$exec_" '^/usr/bin/sh -c '
while IFS= read -r c; do
  exists "$REPO/${c#/usr/share/mk/}"
done < <(grep -o '/usr/share/mk/[A-Za-z0-9_./-]*' <<<"$exec_")
eq 2 "$(grep -o '/usr/share/mk/[A-Za-z0-9_./-]*' <<<"$exec_" | sort -u | wc -l)"

t "gancho: o comando, com os caminhos trocados, deixa o os-release e o issue do Mk"
mkdir -p "$SBX/etc" "$SBX/usr/lib"
echo 'NAME="Arch Linux"' >"$SBX/usr/lib/os-release"
ln -s ../usr/lib/os-release "$SBX/etc/os-release"
echo 'Arch Linux \r (\l)' >"$SBX/etc/issue"
cmd=${exec_#/usr/bin/sh -c }
cmd=${cmd#\'}
cmd=${cmd%\'}
cmd=${cmd//\/usr\/share\/mk\//$REPO/}
cmd=${cmd//\/etc\//$SBX/etc/}
# Só roda se não sobrou nenhum caminho do sistema de verdade.
if grep -q -E "(^|[ =])/(etc|usr|boot|var)/" <<<"${cmd//$REPO/}"; then
  _fail "sobrou caminho de fora de \$SBX: $cmd"
else
  ok sh -c "$cmd"
  ok cmp -s "$d/os-release" "$SBX/etc/os-release"
  ok cmp -s "$d/issue" "$SBX/etc/issue"
fi

t "gancho: o os-release do pacote filesystem não é escrito por cima (o link é trocado por um arquivo)"
fails test -L "$SBX/etc/os-release"
eq 'NAME="Arch Linux"' "$(<"$SBX/usr/lib/os-release")"

t "gancho: sem os arquivos do Mk (a instalação ainda não os copiou) não faz nada e não acusa erro"
# Visto no log da instalação: o gancho do live roda no pacstrap, antes de o
# instalador copiar /usr/share/mk para o disco. O cp falhava ("cannot stat",
# "command failed to execute correctly") e o /etc/os-release já tinha sido
# apagado.
mkdir -p "$SBX/vazio/etc"
echo 'NAME="Arch Linux"' >"$SBX/vazio/etc/os-release"
echo 'Arch Linux' >"$SBX/vazio/etc/issue"
semmk=${exec_#/usr/bin/sh -c }
semmk=${semmk#\'}
semmk=${semmk%\'}
semmk=${semmk//\/usr\/share\/mk\//$SBX/nao-existe/}
semmk=${semmk//\/etc\//$SBX/vazio/etc/}
if grep -q -E "(^|[ =])/(etc|usr|boot|var)/" <<<"$semmk"; then
  _fail "sobrou caminho de fora de \$SBX: $semmk"
else
  saida=$(sh -c "$semmk" 2>&1)
  eq 0 $?
  eq "" "$saida"
  eq 'NAME="Arch Linux"' "$(<"$SBX/vazio/etc/os-release")"
  eq 'Arch Linux' "$(<"$SBX/vazio/etc/issue")"
fi

t "gancho: o instalador e o iso/build o instalam em /etc/pacman.d/hooks"
has "$(<"$REPO/installer/mk-install")" 'etc/pacman.d/hooks/mk-os-release.hook'
has "$(<"$REPO/iso/build")" 'etc/pacman.d/hooks/mk-os-release.hook'

t "channel: um endereço só, https e de git; o resto é comentário"
canal=$(grep -v -e '^#' -e '^$' "$d/channel")
eq 1 "$(grep -c . <<<"$canal")"
like "$canal" '^https://[A-Za-z0-9./_-]+\.git$'
