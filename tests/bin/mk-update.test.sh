#!/usr/bin/env bash
# bin/mk-update: a rodada inteira numa raiz falsa (MK_ROOT): a procura, a troca
# do Mk, as migrações, o pacman, os apps novos, o AUR, a trava, o resumo e o
# registro, a trava contra duas rodadas e a confirmação única. sudo, pacman, yay,
# git, pgrep e uname são stubs.
. "$(dirname "$0")/../lib.sh"

R=$SBX/raiz
share=$R/usr/share/mk
cfg=$XDG_CONFIG_HOME/mk
state=$XDG_STATE_HOME/mk
resumo=$state/ultima-atualizacao.json
vistos=$R/var/lib/mk/apps-oferecidos
trava=$XDG_RUNTIME_DIR/mk-update.trava
kernel=6.0-teste
velha="aaaaaaaaaaaa 2026-11-01"
nova="bbbbbbbbbbbb 2026-11-10"

# Uma árvore do Mk (o /usr/share/mk instalado, ou o que o canal entrega).
monta_mk() { # monta_mk <pasta> <"commit data">
  local d=$1 p
  mkdir -p "$d"/{bin,system/mk/boot,system/packages,shell/mare/scripts,migrations/system,migrations/user}
  cp "$REPO/bin/mk-update" "$d/bin/mk-update"
  for p in mk-session mk-hibernate; do printf '#!/bin/bash\n# %s de %s\n' "$p" "$2" >"$d/bin/$p"; done
  echo "gancho de $2" >"$d/system/mk/mk-os-release.hook"
  printf '#!/bin/bash\n# ajudante do DNS de %s\n' "$2" >"$d/system/mk/mk-dns"
  echo "canal-de-teste" >"$d/system/mk/channel"
  for p in config/hypr config/ghostty config/fastfetch config/bash shell/mare; do
    mkdir -p "$d/$p"
    echo "$2" >"$d/$p/versao"
  done
  # A procura: devolve o que o caso escreveu em $SBX/info.json. Com PRENDE no
  # ambiente, só responde depois que $SBX/solta existir (uma procura demorada).
  printf '#!/bin/bash\nwhile [[ -n ${PRENDE:-} && ! -e $SBX/solta ]]; do sleep 0.1; done\ncat "$SBX/info.json"\n' \
    >"$d/shell/mare/scripts/mare-updates"
  chmod +x "$d/shell/mare/scripts/mare-updates"
  printf '#!/bin/bash\necho limine-default >>"$SBX/ordem"\n' >"$d/system/mk/boot/limine-default"
  echo "$2" >"$d/VERSION"
}

# O que a procura responde: info <mk novo: true|false> <nº sistema> <nº AUR>.
info() {
  jq -nc --argjson av "$1" --argjson nsys "$2" --argjson naur "$3" '{
    system: [range($nsys) | {name: "pacote\(.)", from: "1", to: "2"}],
    aur: [range($naur) | {name: "aur\(.)", from: "1", to: "2"}],
    mk: {version: "aaaaaaaaaaaa", latest: "bbbbbbbbbbbb", channel: "canal-de-teste", available: $av},
    error: ""}' >"$SBX/info.json"
}

# Uma migração que anota em $SBX/ordem que rodou (e com que MK_SHARE).
migra() { # migra <system|user> <nome> [linha do cabeçalho] [código de saída]
  printf '#!/usr/bin/env bash\n%s\necho "%s/%s $MK_SHARE" >>"$SBX/ordem"\nexit %s\n' \
    "${3:-#}" "$1" "$2" "${4:-0}" >"$share/migrations/$1/$2"
}

# Máquina instalada na versão velha, tudo em dia, canal com a nova.
zera() {
  rm -rf "${R:?}" "${cfg:?}" "${state:?}" "${SBX:?}/canal" "${SBX:?}"/{instalados,sumiram,pacman-rc,yay-rc,git-falha,git-fecha,pacman-rodando,passou,pacman-plano,pacman-demora,solta}
  calls_reset
  : >"$SBX/ordem"
  monta_mk "$share" "$velha"
  monta_mk "$SBX/canal" "$nova"
  mkdir -p "$R/usr/bin" "$R/usr/lib/modules/$kernel" "$R/var/lib/mk" "$R/var/lib/pacman" "$R/etc" "$cfg"
  echo "$velha" >"$cfg/VERSION"
  local d
  for d in hypr shell ghostty fastfetch bash; do
    mkdir -p "$cfg/$d"
    echo "$velha" >"$cfg/$d/versao"
    touch "$cfg/$d/.instalado-pelo-mk"
  done
  info false 0 0
}

# O sudo falso executa o comando, mas só aceita caminhos dentro de $SBX: um
# caminho do sistema de verdade reprova o arquivo (o MK_ROOT não cobriu algo).
# "sudo -v" (pedir a senha) e "sudo -n -v" (renovar) só ficam anotados.
stub sudo '
[[ $* == -v || $* == "-n -v" ]] && exit 0
for a in "$@"; do
  case $a in /*) p=$a ;; *=/*) p=${a#*=} ;; *) continue ;; esac
  case $p in "$SBX"/* | /dev/null) ;; *)
    echo "sudo fora do SBX: $*" >>"$SBX/perigo"
    exit 97 ;;
  esac
done
exec "$@"'
# pacman: -T devolve os que faltam (os que não estão em $SBX/instalados), -Si
# falha para os de $SBX/sumiram, e a transação sai com o código de $SBX/pacman-rc
# (depois de $SBX/pacman-demora segundos, se houver). O ensaio (--print) não
# entra em $SBX/ordem e responde o que estiver em $SBX/pacman-plano.
stub pacman '
case " $* " in *" --print "*) cat "$SBX/pacman-plano" 2>/dev/null; exit 0 ;; esac
case $1 in
  -T) shift; for p in "$@"; do grep -qxF -- "$p" "$SBX/instalados" 2>/dev/null || echo "$p"; done ;;
  -Si) ! grep -qxF -- "$2" "$SBX/sumiram" 2>/dev/null ;;
  -Sy) echo "pacman $*" >>"$SBX/ordem" ;;
  -S*) echo "pacman $*" >>"$SBX/ordem"; sleep "$(cat "$SBX/pacman-demora" 2>/dev/null || echo 0)"; exit "$(cat "$SBX/pacman-rc" 2>/dev/null || echo 0)" ;;
esac'
stub yay 'echo "yay $*" >>"$SBX/ordem"; exit "$(cat "$SBX/yay-rc" 2>/dev/null || echo 0)"'
# git: o clone copia $SBX/canal (com um .git, para ver que ele sai). Com
# $SBX/git-falha, morre deixando um clone pela metade; com $SBX/git-fecha, a
# janela é fechada (TERM) no meio dele.
stub git '
case " $* " in
  *" clone "*)
    if [[ -e $SBX/git-falha || -e $SBX/git-fecha ]]; then
      mkdir -p "${@: -1}/.git" && echo metade >"${@: -1}/.git/pack"
      [[ -e $SBX/git-fecha ]] && kill -TERM "$PPID"
      exit 128
    fi
    cp -a "$SBX/canal" "${@: -1}" && mkdir -p "${@: -1}/.git" ;;
  *" show HEAD:"*) f=${*: -1}; cat "$2/${f#HEAD:}" 2>/dev/null ;;
  *rev-parse*) echo abcdef123456 ;;
  *" log "*) echo 2026-11-20 ;;
esac'
stub pgrep '[[ -e $SBX/pacman-rodando ]]'
stub uname "echo $kernel"

run() { MK_ROOT=$R bash "$REPO/bin/mk-update" "$@" 2>&1; }
# Para os casos "sem o yay": um PATH só com os stubs e as ferramentas inofensivas
# de que o mk-update precisa. Nada do /usr/bin de verdade fica ao alcance.
mkdir -p "$SBX/min"
for c in bash env cat jq date mktemp rm cp mv mkdir touch sed grep tr head basename install tee; do
  ln -s "$(PATH=/usr/bin:/bin command -v "$c")" "$SBX/min/$c"
done
sem_yay() {
  rm -f "$SBX/bin/yay"
  PATH=$SBX/bin:$SBX/min run
  local rc=$?
  stub yay 'echo "yay $*" >>"$SBX/ordem"; exit "$(cat "$SBX/yay-rc" 2>/dev/null || echo 0)"'
  return "$rc"
}
campo() { jq -r ".$1" "$resumo"; }

# ---------------------------------------------------------------- a procura

t "--check: mostra o que há, não pergunta, não aplica e não grava resumo"
zera
info true 2 1
migra system 20260101-a.sh
out=$(run --check </dev/null)
eq 0 $?
has "$out" "nova versão (bbbbbbbbbbbb)"
has "$out" "sistema    2 pacotes"
has "$out" "AUR        1 pacotes"
has "$out" "ajustes    1 mudanças do Mk no sistema"
has "$out" "pacote0  1 → 2"
has "$out" "aur0  1 → 2"
hasnt "$out" "Atualizar agora?"
not_called "sudo"
not_called "pacman -S"
not_called "git"
absent "$resumo"
eq "" "$(cat "$SBX/ordem")"
eq "$velha" "$(cat "$share/VERSION")"

t "sem nada a fazer: 'tudo em dia', sem perguntar e sem resumo"
zera
out=$(run </dev/null)
eq 0 $?
has "$out" "em dia (aaaaaaaaaaaa)"
has "$out" "tudo em dia"
hasnt "$out" "Atualizar agora?"
not_called "sudo"
absent "$resumo"

t "sem canal: diz 'sem canal de atualização' e o Arch atualiza do mesmo jeito"
zera
info false 1 0
jq -c '.mk.channel = ""' "$SBX/info.json" >"$SBX/info.novo" && mv "$SBX/info.novo" "$SBX/info.json"
out=$(run <<<"s")
eq 0 $?
has "$out" "sem canal de atualização"
called "sudo pacman -Su --noconfirm --needed"
not_called "git"

t "responder 'n' não aplica nada e não grava resumo"
zera
info true 3 1
migra system 20260101-a.sh
out=$(run <<<"n")
eq 0 $?
not_called "sudo"
not_called "git"
not_called "yay"
absent "$resumo"
eq "" "$(cat "$SBX/ordem")"

t "sem resposta (janela fechada na pergunta) é não"
zera
info false 1 0
run </dev/null >/dev/null
eq 0 $?
not_called "sudo"
absent "$resumo"

t "Enter sozinho é sim"
zera
info false 1 0
run <<<"" >/dev/null
called "sudo pacman -Su --noconfirm --needed"

t "mais de 40 pacotes: lista 40 e diz quantos ficaram de fora"
zera
info false 43 2
out=$(run --check)
eq 40 "$(grep -c ' → ' <<<"$out")"
has "$out" "… e mais 5"

t "um aviso da procura (sem internet) aparece e não impede a rodada"
zera
info false 1 0
jq -c '.error = "não deu para consultar os espelhos"' "$SBX/info.json" >"$SBX/info.novo" && mv "$SBX/info.novo" "$SBX/info.json"
out=$(run <<<"s")
eq 0 $?
has "$out" "aviso: não deu para consultar os espelhos"

t "a procura quebrada (sem JSON) para antes de perguntar, sem resumo"
zera
printf '#!/bin/bash\nexit 1\n' >"$share/shell/mare/scripts/mare-updates"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "sudo"
absent "$resumo"

# ------------------------------------------------------------ a troca do Mk

t "Mk novo: /usr/share/mk passa a ser o conteúdo do canal, sem .git nem sobras"
zera
info true 0 0
echo "sobra" >"$share/arquivo-que-saiu.txt"
out=$(run <<<"s")
eq 0 $?
eq "$nova" "$(cat "$share/VERSION")"
absent "$share/.git"
absent "$share/arquivo-que-saiu.txt"
absent "$share.novo"
called "clone -q --depth 1 canal-de-teste"
has "$out" "Mk em $nova"
not_called "hyprctl"

t "Mk novo: reinstala /usr/bin/mk-session, mk-update, mk-hibernate e o gancho do pacman"
has "$(cat "$R/usr/bin/mk-session")" "mk-session de $nova"
has "$(cat "$R/usr/bin/mk-hibernate")" "mk-hibernate de $nova"
ok cmp -s "$R/usr/bin/mk-update" "$SBX/canal/bin/mk-update"
ok test -x "$R/usr/bin/mk-update"
eq "gancho de $nova" "$(cat "$R/etc/pacman.d/hooks/mk-os-release.hook")"

t "Mk novo: o ajudante do DNS vai para /usr/lib/mk, fora da pasta que a troca deixa no nome de quem atualiza"
has "$(cat "$R/usr/lib/mk/mk-dns")" "ajudante do DNS de $nova"
eq 755 "$(stat -c %a "$R/usr/lib/mk/mk-dns")"
called "install -Dm755 $R/usr/share/mk/system/mk/mk-dns $R/usr/lib/mk/mk-dns"

t "Mk novo: troca as pastas de ~/.config/mk e grava o VERSION"
for d in hypr shell ghostty fastfetch bash; do
  eq "$nova" "$(cat "$cfg/$d/versao")" "$d"
  exists "$cfg/$d/.instalado-pelo-mk"
  absent "$cfg/$d.novo"
  absent "$cfg/$d.velho"
done
eq "$nova" "$(cat "$cfg/VERSION")"

t "Mk novo: o VERSION de ~/.config/mk é gravado inteiro e de uma vez (arquivo ao lado, sync, troca)"
# Um desligamento à força logo depois da atualização deixava o arquivo com 0
# bytes (visto na VM): gravar por cima sem sync não garante o conteúdo.
absent "$cfg/VERSION.novo"
has "$(sed -n '/install_bins$/,/after-mk$/p' "$REPO/bin/mk-update")" 'mv "$cfg/VERSION.novo" "$cfg/VERSION"'
has "$(sed -n '/install_bins$/,/after-mk$/p' "$REPO/bin/mk-update")" 'sync'

t "Mk novo: o resumo diz de qual versão para qual"
eq true "$(campo ok)"
eq aaaaaaaaaaaa "$(campo from)"
eq bbbbbbbbbbbb "$(campo to)"

t "Mk novo: a pasta que perdeu a marca fica como está, com aviso; os seus ajustes ficam"
zera
info true 0 0
rm "$cfg/hypr/.instalado-pelo-mk"
echo '{"meu":1}' >"$cfg/ajustes.json"
out=$(run <<<"s")
eq 0 $?
has "$out" "mantido (não é do Mk): $cfg/hypr"
eq "$velha" "$(cat "$cfg/hypr/versao")"
absent "$cfg/hypr/.instalado-pelo-mk"
eq "$nova" "$(cat "$cfg/shell/versao")"
eq '{"meu":1}' "$(cat "$cfg/ajustes.json")"

t "Mk novo: a pasta que não existe em ~/.config/mk é criada, com a marca"
zera
info true 0 0
rm -rf "$cfg/fastfetch"
run <<<"s" >/dev/null
eq "$nova" "$(cat "$cfg/fastfetch/versao")"
exists "$cfg/fastfetch/.instalado-pelo-mk"

t "Mk novo: o aur-pendentes.txt de /usr/share/mk é preservado"
zera
info true 0 0
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
sem_yay <<<"s" >/dev/null # sem yay, a oferta do AUR não entra nesta rodada
eq "$nova" "$(cat "$share/VERSION")"
eq "limine-snapper-sync" "$(cat "$share/aur-pendentes.txt" 2>/dev/null)"

t "Mk novo: passa a vez ao /usr/bin/mk-update recém-instalado, com --after-mk e a versão de antes"
zera
info true 0 0
printf '#!/bin/bash\necho "$* from=$MK_UPDATE_FROM" >"$SBX/passou"\n' >"$SBX/canal/bin/mk-update"
run <<<"s" >/dev/null
eq 0 $?
eq "--after-mk from=aaaaaaaaaaaa" "$(cat "$SBX/passou" 2>/dev/null)"

t "Mk novo: canal sem VERSION usa o commit e a data do clone"
zera
info true 0 0
rm "$SBX/canal/VERSION"
run <<<"s" >/dev/null
eq 0 $?
eq "abcdef123456 2026-11-20" "$(cat "$cfg/VERSION")"
eq "abcdef123456 2026-11-20" "$(cat "$share/VERSION")"

t "Mk novo dentro do Hyprland, com o qs: recarrega a config e reinicia o Maré"
zera
info true 0 0
stub hyprctl
stub qs
HYPRLAND_INSTANCE_SIGNATURE=teste run <<<"s" >/dev/null
eq 0 $?
called "hyprctl reload"
called "hyprctl dispatch hl.dsp.exec_cmd(\"bash '$cfg/shell/scripts/mare-start' --restart\")"
rm -f "$SBX/bin/hyprctl" "$SBX/bin/qs"

t "Mk novo vindo de um mk-update antigo: o que só a versão nova instala chega na mesma rodada"
# Visto na VM: um Mk de antes do mk-hibernate atualizou e o /usr/bin/mk-hibernate
# não veio, porque quem copia os binários é o mk-update velho, que não o conhecia.
# A versão nova, ao assumir (--after-mk), confere os binários de novo.
zera
info true 0 0
# O "antigo": o de agora sem o mk-hibernate e o gancho na lista, e sem a
# conferência ao assumir.
sed -e 's/for b in mk-session mk-update mk-hibernate; do/for b in mk-session mk-update; do/' \
  -e '/^if \[\[ \$mode == --after-mk \]\]; then install_bins; fi$/d' \
  -e '/mk-os-release.hook/d' -e '/sudo install -Dm644 "\$src" "\$dst"/d' \
  "$REPO/bin/mk-update" >"$SBX/mk-update-antigo"
fails grep -qE 'mk-hibernate; do|mk-os-release|then install_bins' "$SBX/mk-update-antigo"
ok bash -n "$SBX/mk-update-antigo"
out=$(MK_ROOT=$R bash "$SBX/mk-update-antigo" <<<"s" 2>&1)
eq 0 $?
has "$(cat "$R/usr/bin/mk-hibernate" 2>/dev/null)" "mk-hibernate de $nova"
eq "gancho de $nova" "$(cat "$R/etc/pacman.d/hooks/mk-os-release.hook" 2>/dev/null)"
ok test -x "$R/usr/bin/mk-hibernate"

t "--after-mk com os binários já iguais não reinstala nada"
calls_reset
out=$(MK_ROOT=$R bash "$REPO/bin/mk-update" --after-mk 2>&1 <<<"")
not_called "install -m755"
not_called "install -Dm644"

t "o clone do canal falha: para em 'mk' e nada é trocado"
zera
info true 1 0
touch "$SBX/git-falha"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq false "$(campo ok)"
eq mk "$(campo step)"
eq aaaaaaaaaaaa "$(campo to)"
eq "$velha" "$(cat "$share/VERSION")"
eq "$velha" "$(cat "$cfg/VERSION")"
eq "$velha" "$(cat "$cfg/hypr/versao")"
not_called "sudo"
not_called "pacman -S"

t "depois do clone falho, rodar de novo refaz a troca inteira"
rm "$SBX/git-falha"
run <<<"s" >/dev/null
eq 0 $?
eq "$nova" "$(cat "$share/VERSION")"
eq true "$(campo ok)"
eq "" "$(campo step)"
called "sudo pacman -Su --noconfirm --needed"

t "o mk-update novo quebra logo depois da troca: o resumo fica 'parou em: mk', já na versão nova"
zera
info true 1 0
printf '#!/bin/bash\nexit 1\n' >"$SBX/canal/shell/mare/scripts/mare-updates"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq false "$(campo ok)"
eq mk "$(campo step)"
eq aaaaaaaaaaaa "$(campo from)"
eq bbbbbbbbbbbb "$(campo to)"
not_called "pacman -S"

# --------------------------------------------------------------- migrações

t "migrações: as de sistema antes das de usuário, em ordem de nome, com MK_SHARE"
zera
migra user 20260101-u.sh
migra system 20260202-b.sh
migra system 20260101-a.sh
migra user 20260303-v.sh
out=$(run <<<"s")
eq 0 $?
has "$out" "ajustes    4 mudanças do Mk no sistema"
eq "system/20260101-a.sh $share
system/20260202-b.sh $share
user/20260101-u.sh $share
user/20260303-v.sh $share" "$(cat "$SBX/ordem")"

t "migrações: a de sistema roda com sudo, a de usuário sem"
called "sudo env MK_SHARE=$share bash $share/migrations/system/20260101-a.sh"
not_called "bash $share/migrations/user"

t "migrações: cada uma fica anotada como feita (sistema na máquina, usuário no estado dele)"
exists "$R/var/lib/mk/migrations/20260101-a.sh"
exists "$R/var/lib/mk/migrations/20260202-b.sh"
exists "$state/migrations/20260101-u.sh"
exists "$state/migrations/20260303-v.sh"
absent "$state/migrations/20260101-a.sh"

t "migrações: só migração pendente não chama o pacman"
not_called "pacman -S"

t "migrações: rodar de novo não repete nenhuma (tudo em dia)"
out=$(run </dev/null)
eq 0 $?
has "$out" "tudo em dia"
hasnt "$out" "ajustes "
eq 4 "$(grep -c '' "$SBX/ordem")"

t "migrações: uma nova entra sozinha; as já feitas não rodam"
: >"$SBX/ordem"
migra system 20260404-c.sh
run <<<"s" >/dev/null
eq "system/20260404-c.sh $share" "$(cat "$SBX/ordem")"

t "migrações: arquivo que não é .sh é ignorado"
zera
echo "echo nota >>\"\$SBX/ordem\"" >"$share/migrations/system/LEIAME.md"
out=$(run </dev/null)
has "$out" "tudo em dia"

t "migração que falha: a rodada para em 'ajustes' e ela continua pendente"
zera
info false 2 1
migra system 20260101-a.sh
migra system 20260202-quebra.sh "#" 3
migra system 20260303-c.sh
migra user 20260101-u.sh
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq false "$(campo ok)"
eq ajustes "$(campo step)"
exists "$R/var/lib/mk/migrations/20260101-a.sh"
absent "$R/var/lib/mk/migrations/20260202-quebra.sh"
absent "$R/var/lib/mk/migrations/20260303-c.sh"
absent "$state/migrations/20260101-u.sh"

t "migração que falha: o que vinha depois (outras migrações, pacman, AUR) não roda"
eq "system/20260101-a.sh $share
system/20260202-quebra.sh $share" "$(cat "$SBX/ordem")"
not_called "pacman -S"
not_called "yay"

t "migração consertada: na rodada seguinte roda ela e as que faltavam, não a que já foi"
: >"$SBX/ordem"
migra system 20260202-quebra.sh
run <<<"s" >/dev/null
eq 0 $?
eq "system/20260202-quebra.sh $share
system/20260303-c.sh $share
user/20260101-u.sh $share
pacman -Sy
pacman -Su --noconfirm --needed
yay -Sua --noconfirm --answerclean None --answerdiff None" "$(cat "$SBX/ordem")"
eq true "$(campo ok)"

t "# pacotes: entram na transação (-Sy, depois -Su --needed) e a migração roda depois dela"
zera
migra system 20260101-antes.sh
migra system 20260202-pede.sh "# pacotes: wtype  cups"
migra user 20260101-u.sh
out=$(run <<<"s")
eq 0 $?
eq "system/20260101-antes.sh $share
user/20260101-u.sh $share
pacman -Sy
pacman -Su --noconfirm --needed wtype cups
system/20260202-pede.sh $share" "$(cat "$SBX/ordem")"
exists "$R/var/lib/mk/migrations/20260202-pede.sh"
has "$out" "ajustes do Mk que dependiam dos pacotes"

t "# pacotes: o que já está instalado não entra no pedido"
zera
migra system 20260202-pede.sh "# pacotes: wtype cups"
echo cups >"$SBX/instalados"
run <<<"s" >/dev/null
eq "pacman -Sy
pacman -Su --noconfirm --needed wtype
system/20260202-pede.sh $share" "$(cat "$SBX/ordem")"

t "# pacotes: com tudo já instalado e nada a atualizar, o pacman é pulado e a migração roda"
zera
migra system 20260202-pede.sh "# pacotes: wtype cups"
printf 'wtype\ncups\n' >"$SBX/instalados"
run <<<"s" >/dev/null
eq 0 $?
eq "system/20260202-pede.sh $share" "$(cat "$SBX/ordem")"

t "# pacotes: nome que saiu dos repositórios fica de fora com aviso, e a migração roda assim mesmo"
zera
migra system 20260202-pede.sh "# pacotes: wtype sumido"
echo sumido >"$SBX/sumiram"
out=$(run <<<"s")
eq 0 $?
has "$out" "sumido não está nos repositórios: fica de fora"
eq "pacman -Sy
pacman -Su --noconfirm --needed wtype
system/20260202-pede.sh $share" "$(cat "$SBX/ordem")"

t "# pacotes: nome fora de ^[a-z0-9@._+-]+\$ é descartado em silêncio"
zera
migra system 20260202-pede.sh '# pacotes: wtype Maiusculo $(reboot) a;b'
run <<<"s" >/dev/null
eq 0 $?
called "pacman -Su --noconfirm --needed wtype"
not_called "Maiusculo"
not_called "reboot"

t "# pacotes: em user/ a linha é ignorada (roda antes, sem pacman)"
zera
migra user 20260101-u.sh "# pacotes: wtype"
run <<<"s" >/dev/null
eq "user/20260101-u.sh $share" "$(cat "$SBX/ordem")"
not_called "pacman -S"

t "# pacotes: o pacman falha, a migração que esperava não roda e fica pendente ('sistema')"
zera
migra system 20260202-pede.sh "# pacotes: wtype"
echo 1 >"$SBX/pacman-rc"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq sistema "$(campo step)"
absent "$R/var/lib/mk/migrations/20260202-pede.sh"
hasnt "$(cat "$SBX/ordem")" "system/20260202-pede.sh"

t "# pacotes: a migração que falha depois do pacman para em 'ajustes', e o AUR fica por fazer"
zera
info false 0 1
migra system 20260202-pede.sh "# pacotes: wtype" 1
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq ajustes "$(campo step)"
called "pacman -Su --noconfirm --needed wtype"
not_called "yay"

# ------------------------------------------------------------------ o Arch

t "pacotes do sistema: lista nova (-Sy) e uma transação só (-Su --noconfirm --needed)"
zera
info false 3 0
out=$(run <<<"s")
eq 0 $?
eq "pacman -Sy
pacman -Su --noconfirm --needed" "$(cat "$SBX/ordem")"
has "$out" "pronto"

t "nada do Arch a atualizar (só AUR): o pacman é pulado"
zera
info false 0 2
run <<<"s" >/dev/null
eq 0 $?
not_called "pacman -S"
called "yay -Sua"

t "o pacman falha: para em 'sistema' e o AUR não roda"
zera
info false 2 2
echo 1 >"$SBX/pacman-rc"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq false "$(campo ok)"
eq sistema "$(campo step)"
not_called "yay"

# ------------------------------------------------------------- apps novos

apps() { printf '%s\n' "$@" >"$share/system/packages/apps.packages"; }

t "app novo em apps.packages: é oferecido e entra na mesma transação da atualização"
zera
info false 1 0
apps chromium nautilus loupe
printf 'chromium\nnautilus\n' >"$vistos"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "apps       1 novos do Mk"
has "$out" "apps novos do Mk"
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe" "$(cat "$SBX/ordem")"

t "app novo: depois da rodada fica anotado em apps-oferecidos, uma vez"
eq "chromium
nautilus
loupe" "$(cat "$vistos")"

t "app novo: na rodada seguinte não é oferecido de novo"
out=$(run <<<"s")
hasnt "$out" "novos do Mk"
eq 1 "$(grep -cx loupe "$vistos")"

t "só app novo, sem mais nada a atualizar: a oferta aparece mesmo assim"
zera
apps chromium loupe
echo chromium >"$vistos"
out=$(run <<<$'s\ns')
eq 0 $?
hasnt "$out" "tudo em dia"
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe" "$(cat "$SBX/ordem")"

t "app recusado: não instala, mas fica anotado para não voltar"
zera
info false 1 0
apps chromium loupe
echo chromium >"$vistos"
run <<<$'s\nn' >/dev/null
eq 0 $?
eq "pacman -Sy
pacman -Su --noconfirm --needed" "$(cat "$SBX/ordem")"
eq 1 "$(grep -cx loupe "$vistos")"

t "app recusado sem pacotes a atualizar: o pacman é pulado e a oferta fica anotada"
zera
apps chromium loupe
echo chromium >"$vistos"
run <<<$'s\nn' >/dev/null
eq 0 $?
not_called "pacman -S"
eq 1 "$(grep -cx loupe "$vistos")"

t "app novo que já está instalado não é oferecido"
zera
info false 1 0
apps chromium loupe
echo chromium >"$vistos"
echo loupe >"$SBX/instalados"
out=$(run <<<"s")
hasnt "$out" "novos do Mk"
hasnt "$out" "Instalar junto"
eq "pacman -Sy
pacman -Su --noconfirm --needed" "$(cat "$SBX/ordem")"

t "comentários, linhas vazias e comentário no fim da linha de apps.packages não viram oferta"
zera
info false 1 0
printf '# Navegador\nchromium\n\n   \nloupe   # visor de imagens\n# gimp\n' >"$share/system/packages/apps.packages"
echo chromium >"$vistos"
run <<<$'s\ns' >/dev/null
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe" "$(cat "$SBX/ordem")"
eq "chromium
loupe" "$(cat "$vistos")"

t "apps-oferecidos ausente: nada é oferecido"
zera
info false 1 0
apps chromium loupe
out=$(run <<<"s")
hasnt "$out" "novos do Mk"
eq "pacman -Sy
pacman -Su --noconfirm --needed" "$(cat "$SBX/ordem")"
absent "$vistos"

t "apps-oferecidos vazio: nada é oferecido nunca"
zera
info false 1 0
apps chromium loupe
: >"$vistos"
out=$(run <<<"s")
hasnt "$out" "novos do Mk"
eq "" "$(cat "$vistos")"

t "uma migração que cria o apps-oferecidos nesta rodada já faz a oferta valer"
zera
apps chromium loupe
printf '#!/usr/bin/env bash\necho chromium >"%s"\n' "$vistos" >"$share/migrations/system/20260101-apps.sh"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "apps novos do Mk"
called "sudo pacman -Su --noconfirm --needed loupe"

t "o pacman falha com app aceito: a oferta não é anotada e volta na próxima"
zera
info false 1 0
apps chromium loupe
echo chromium >"$vistos"
echo 1 >"$SBX/pacman-rc"
run <<<$'s\ns' >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq chromium "$(cat "$vistos")"
rm "$SBX/pacman-rc"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "apps       1 novos do Mk"
eq 1 "$(grep -cx loupe "$vistos")"

t "LibreOffice: pergunta à parte; recusado, sai do pedido e os outros ficam"
zera
apps chromium loupe libreoffice-fresh
echo chromium >"$vistos"
out=$(run <<<$'s\ns\nn')
eq 0 $?
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe" "$(cat "$SBX/ordem")"
eq 1 "$(grep -cx libreoffice-fresh "$vistos")"

t "LibreOffice aceito num sistema em pt_BR: entra junto o libreoffice-fresh-pt-br"
zera
apps chromium libreoffice-fresh
echo chromium >"$vistos"
echo "LANG=pt_BR.UTF-8" >"$R/etc/locale.conf"
run <<<$'s\ns\ns' >/dev/null
eq "pacman -Sy
pacman -Su --noconfirm --needed libreoffice-fresh libreoffice-fresh-pt-br" "$(cat "$SBX/ordem")"

t "LibreOffice aceito noutro idioma: sem o pacote de pt-br"
zera
apps chromium libreoffice-fresh
echo chromium >"$vistos"
echo "LANG=en_US.UTF-8" >"$R/etc/locale.conf"
run <<<$'s\ns\ns' >/dev/null
eq "pacman -Sy
pacman -Su --noconfirm --needed libreoffice-fresh" "$(cat "$SBX/ordem")"

t "apps novos e pacotes de migração vão na mesma transação"
zera
apps chromium loupe
echo chromium >"$vistos"
migra system 20260202-pede.sh "# pacotes: wtype"
run <<<$'s\ns' >/dev/null
eq "pacman -Sy
pacman -Su --noconfirm --needed wtype loupe
system/20260202-pede.sh $share" "$(cat "$SBX/ordem")"

# -------------------------------------------------------------------- AUR

t "AUR: yay -Sua sem perguntas, depois do pacman"
zera
info false 1 2
run <<<"s" >/dev/null
eq 0 $?
eq "pacman -Sy
pacman -Su --noconfirm --needed
yay -Sua --noconfirm --answerclean None --answerdiff None" "$(cat "$SBX/ordem")"
not_called "sudo yay"

t "AUR: o yay falha, a rodada para em 'aur'"
zera
info false 0 1
echo 1 >"$SBX/yay-rc"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq false "$(campo ok)"
eq aur "$(campo step)"

t "AUR sem o yay: o passo é pulado em silêncio e a rodada termina bem"
zera
info false 1 2
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
out=$(sem_yay <<<"s")
eq 0 $?
has "$out" "pronto"
hasnt "$out" "AUR que ficou de fora"
eq "pacman -Sy
pacman -Su --noconfirm --needed" "$(cat "$SBX/ordem")"
eq true "$(campo ok)"
exists "$share/aur-pendentes.txt"

t "aur-pendentes: aceito e o yay dá certo, instala, apaga o arquivo e acerta o menu de boot"
zera
info false 1 0
printf 'limine-mkinitcpio-hook\nlimine-snapper-sync\n' >"$share/aur-pendentes.txt"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "AUR que ficou de fora da instalação: limine-mkinitcpio-hook limine-snapper-sync"
eq "pacman -Sy
pacman -Su --noconfirm --needed
yay -S --needed --noconfirm --answerclean None --answerdiff None --removemake limine-mkinitcpio-hook limine-snapper-sync
limine-default" "$(cat "$SBX/ordem")"
absent "$share/aur-pendentes.txt"
called "sudo bash $share/system/mk/boot/limine-default"

t "aur-pendentes: o yay falha, o arquivo fica para a próxima e a rodada termina 'pronto'"
zera
info false 1 0
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
echo 1 >"$SBX/yay-rc"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "pronto"
exists "$share/aur-pendentes.txt"
has "$(cat "$SBX/ordem")" "limine-default"
eq true "$(campo ok)"

t "aur-pendentes recusado: não chama o yay e o arquivo fica"
zera
info false 1 0
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
run <<<$'s\nn' >/dev/null
eq 0 $?
not_called "yay"
exists "$share/aur-pendentes.txt"
hasnt "$(cat "$SBX/ordem")" "limine-default"

t "aur-pendentes vazio: nada é perguntado"
zera
info false 1 0
: >"$share/aur-pendentes.txt"
out=$(run <<<"s")
hasnt "$out" "AUR que ficou de fora"
not_called "yay"

# ---------------------------------------------------------- trava do pacman

t "trava do pacman sem ninguém rodando: oferece e, aceito, remove"
zera
info false 1 0
touch "$R/var/lib/pacman/db.lck"
out=$(run <<<$'s\ns')
eq 0 $?
has "$out" "o pacman ficou travado por uma atualização interrompida"
absent "$R/var/lib/pacman/db.lck"
called "sudo rm -f $R/var/lib/pacman/db.lck"
called "sudo pacman -Su --noconfirm"

t "trava do pacman recusada: fica onde está"
zera
info false 1 0
touch "$R/var/lib/pacman/db.lck"
run <<<$'s\nn' >/dev/null
exists "$R/var/lib/pacman/db.lck"

t "trava do pacman com um pacman rodando: não oferece tirar"
zera
info false 1 0
touch "$R/var/lib/pacman/db.lck" "$SBX/pacman-rodando"
out=$(run <<<"s")
hasnt "$out" "Destravar"
exists "$R/var/lib/pacman/db.lck"
called "pgrep -x pacman|yay|makepkg|pamac"

t "sem trava: nem pergunta"
zera
info false 1 0
out=$(run <<<"s")
hasnt "$out" "Destravar"

# ------------------------------------------------------ uma confirmação só

t "as decisões (trava, apps, AUR que ficou de fora) vêm todas antes da transação"
# Visto em uso: a pessoa respondeu a primeira pergunta, saiu, e a rodada ficou
# nove minutos parada na pergunta seguinte, no meio.
zera
info false 1 1
touch "$R/var/lib/pacman/db.lck"
apps chromium loupe
echo chromium >"$vistos"
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
out=$(run <<<$'s\ns\ns\ns')
eq 0 $?
antes=${out%%"sistema (sudo pede"*}
has "$antes" "o pacman ficou travado"
has "$antes" "apps novos do Mk"
has "$antes" "AUR que ficou de fora da instalação: limine-snapper-sync"
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe
yay -Sua --noconfirm --answerclean None --answerdiff None
yay -S --needed --noconfirm --answerclean None --answerdiff None --removemake limine-snapper-sync
limine-default" "$(cat "$SBX/ordem")"

t "a ordem das decisões: a trava, os apps, o AUR que ficou de fora"
zera
info false 1 0
touch "$R/var/lib/pacman/db.lck"
apps chromium loupe
echo chromium >"$vistos"
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
run <<<$'s\nn\ns\nn' >/dev/null
eq 0 $?
exists "$R/var/lib/pacman/db.lck"
eq "pacman -Sy
pacman -Su --noconfirm --needed loupe" "$(cat "$SBX/ordem")"
exists "$share/aur-pendentes.txt"

t "as decisões vêm antes também das migrações"
zera
info false 1 0
migra system 20260101-a.sh
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
out=$(run <<<$'s\ns')
eq 0 $?
has "${out%%"ajustes do Mk no sistema"*}" "AUR que ficou de fora"
eq "system/20260101-a.sh $share
pacman -Sy
pacman -Su --noconfirm --needed
yay -S --needed --noconfirm --answerclean None --answerdiff None --removemake limine-snapper-sync
limine-default" "$(cat "$SBX/ordem")"

t "depois da última decisão ninguém lê mais nada do terminal"
# Uma resposta a mais do que as perguntas: tem de sobrar, sem ninguém para ler.
zera
info false 1 1
echo "limine-snapper-sync" >"$share/aur-pendentes.txt"
sobra=$({ printf 's\ns\n'; echo SOBRA; } | { run >/dev/null; cat; })
eq SOBRA "$sobra"

t "o pacman teria de escolher um provedor: para em 'sistema' com a escolha na tela, sem transação"
# Com --noconfirm o pacman pegaria o primeiro da lista sozinho. O ensaio
# (--print, sem sudo) mostra a escolha antes, e a rodada para.
zera
info false 1 0
apps chromium loupe
echo chromium >"$vistos"
printf ':: There are 2 providers available for jack:\n:: Repository extra\n   1) jack2  2) pipewire-jack\n\nEnter a number (default=1): \n' >"$SBX/pacman-plano"
out=$(run <<<$'s\ns')
[[ $? -ne 0 ]] || _fail "devia sair com erro"
has "$out" "o pacman precisa de uma escolha sua"
has "$out" "There are 2 providers available for jack:"
has "$out" "1) jack2  2) pipewire-jack"
has "$out" "sudo pacman -Syu --needed loupe"
eq "pacman -Sy" "$(cat "$SBX/ordem")"
eq false "$(campo ok)"
eq sistema "$(campo step)"
eq chromium "$(cat "$vistos")"

t "o ensaio do pacman roda sem sudo, com os mesmos pacotes da transação"
called "pacman -Su --print --noconfirm --needed loupe"
not_called "sudo pacman -Su --print"

t "o ensaio que não pede escolha nenhuma deixa a transação seguir"
zera
info false 1 0
printf 'https://espelho/pacote0-2.pkg.tar.zst\n' >"$SBX/pacman-plano"
out=$(run <<<"s")
eq 0 $?
hasnt "$out" "escolha sua"
called "sudo pacman -Su --noconfirm --needed"

t "o pacman falha na transação: diz que ele rodou sem perguntar e como responder à mão"
zera
info false 1 0
echo 1 >"$SBX/pacman-rc"
out=$(run <<<"s")
eq 1 $?
has "$out" "o pacman parou"
has "$out" "sudo pacman -Syu"
eq sistema "$(campo step)"

t "a senha do sudo é pedida uma vez, antes da transação, e renovada enquanto ela roda"
zera
info false 1 1
echo 1 >"$SBX/pacman-demora"
MK_UPDATE_SUDO_CADA=0.2 run <<<"s" >/dev/null
eq 0 $?
eq "sudo -v" "$(grep '^sudo' "$SBX/calls" | head -1)"
called "sudo -n -v"

t "terminada a rodada, ninguém continua renovando o sudo"
n=$(grep -c '^sudo -n -v' "$SBX/calls")
sleep 0.7
eq "$n" "$(grep -c '^sudo -n -v' "$SBX/calls")"

t "só AUR a atualizar: a senha é pedida antes de o yay começar a compilar"
zera
info false 0 1
run <<<"s" >/dev/null
eq 0 $?
eq "sudo -v
yay -Sua --noconfirm --answerclean None --answerdiff None" "$(grep -E '^(sudo|yay)' "$SBX/calls")"

t "rodada que não precisa de root (só migração de usuário) não chama o sudo"
zera
migra user 20260101-u.sh
run <<<"s" >/dev/null
eq 0 $?
not_called "sudo"

# ------------------------------------------------- duas rodadas ao mesmo tempo

t "com uma rodada em curso, a segunda diz 'já há uma atualização rodando' e sai com erro, sem tocar em nada"
zera
info false 1 0
exec 8>"$trava"
ok flock -n 8
out=$(run <<<"s")
[[ $? -ne 0 ]] || _fail "devia sair com erro"
has "$out" "já há uma atualização rodando"
hasnt "$out" "procurando"
not_called "sudo"
absent "$resumo"

t "--check não é travado por uma rodada em curso"
out=$(run --check)
eq 0 $?
has "$out" "sistema    1 pacotes"
hasnt "$out" "já há uma atualização"
exec 8>&-

t "terminada a outra, a rodada roda"
out=$(run <<<"s")
eq 0 $?
hasnt "$out" "já há uma atualização"
called "sudo pacman -Su --noconfirm --needed"

t "--check não trava: uma rodada começa com um --check ainda procurando"
zera
info false 1 0
PRENDE=1 run --check >/dev/null &
pid=$!
out=$(run <<<"s")
eq 0 $?
hasnt "$out" "já há uma atualização"
touch "$SBX/solta"
wait "$pid"

t "duas rodadas de verdade: a que chega depois é recusada enquanto a primeira está no pacman"
zera
info false 1 0
echo 30 >"$SBX/pacman-demora"
MK_ROOT=$R setsid bash "$REPO/bin/mk-update" <<<"s" >/dev/null 2>&1 &
pid=$!
for _ in $(seq 100); do grep -q 'pacman -Su' "$SBX/ordem" && break; sleep 0.1; done
has "$(cat "$SBX/ordem")" "pacman -Su --noconfirm"
rm "$SBX/pacman-demora"
out=$(run <<<"s")
[[ $? -ne 0 ]] || _fail "devia sair com erro"
has "$out" "já há uma atualização rodando"
eq 1 "$(grep -c 'pacman -Su' "$SBX/ordem")"

t "a trava some sozinha quando a rodada morre (kill -9)"
kill -KILL -- "-$pid" 2>/dev/null
wait "$pid" 2>/dev/null
for _ in $(seq 50); do flock -n "$trava" true && break; sleep 0.1; done
out=$(run <<<"s")
eq 0 $?
hasnt "$out" "já há uma atualização"
eq 2 "$(grep -c 'pacman -Su' "$SBX/ordem")"

t "sem o flock instalado, a rodada roda do mesmo jeito (sem trava)"
zera
info false 1 0
out=$(PATH=$(path_sem flock) run <<<"s")
eq 0 $?
called "sudo pacman -Su --noconfirm --needed"

# ------------------------------------------------------ a pasta temporária

t "o clone do canal falha: a pasta temporária não fica para trás"
zera
info true 0 0
rm -rf "${TMPDIR:?}"/*
echo meu >"$TMPDIR/alheio"
touch "$SBX/git-falha"
run <<<"s" >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq "$TMPDIR/alheio" "$(find "$TMPDIR" -mindepth 1 -maxdepth 1)"

t "janela fechada no meio do clone (TERM): também limpa, e o resumo fica em 'mk'"
zera
info true 0 0
touch "$SBX/git-fecha"
run <<<"s" >/dev/null
eq 143 $?
eq "$TMPDIR/alheio" "$(find "$TMPDIR" -mindepth 1 -maxdepth 1)"
eq mk "$(campo step)"

t "a troca do Mk que dá certo não deixa pasta temporária nem apaga o que não é dela"
zera
info true 0 0
run <<<"s" >/dev/null
eq 0 $?
eq "$TMPDIR/alheio" "$(find "$TMPDIR" -mindepth 1 -maxdepth 1)"
eq meu "$(cat "$TMPDIR/alheio")"
rm -f "$TMPDIR/alheio"

# ----------------------------------------------------------------- o resumo

t "resumo: rodada que deu certo grava todos os campos"
zera
info false 1 0
antes=$(date +%s)
run <<<"s" >/dev/null
eq 0 $?
eq '["applied","from","log","ok","reboot","started","step","to","when"]' "$(jq -c 'keys' "$resumo")"
eq true "$(campo ok)"
eq "" "$(campo step)"
eq false "$(campo reboot)"
eq "$(campo when)" "$(campo applied)"
eq aaaaaaaaaaaa "$(campo from)"
eq aaaaaaaaaaaa "$(campo to)"
eq "" "$(campo log)"
ok test "$(campo started)" -ge "$antes"
ok test "$(campo when)" -ge "$(campo started)"
absent "$resumo.novo"

t "resumo: o caminho do registro (MK_UPDATE_LOG) vai no campo log"
zera
info false 1 0
MK_UPDATE_LOG=$state/atualizacao-x.log run <<<"s" >/dev/null
eq "$state/atualizacao-x.log" "$(campo log)"

t "resumo: rodada que falha mantém o 'applied' da última que deu certo"
zera
info false 1 0
mkdir -p "$state"
echo '{"applied": 1700000000, "ok": true}' >"$resumo"
echo 1 >"$SBX/pacman-rc"
run <<<"s" >/dev/null
eq false "$(campo ok)"
eq 1700000000 "$(campo applied)"
eq sistema "$(campo step)"

t "resumo: um resumo anterior estragado não derruba a rodada"
zera
info false 1 0
mkdir -p "$state"
echo 'isto não é json' >"$resumo"
run <<<"s" >/dev/null
eq 0 $?
eq true "$(campo ok)"

t "kernel em uso já saiu do disco: avisa para reiniciar e o resumo marca reboot"
zera
info false 1 0
rmdir "$R/usr/lib/modules/$kernel"
out=$(run <<<"s")
eq 0 $?
has "$out" "o kernel foi atualizado: reinicie quando puder"
hasnt "$out" "pronto"
eq true "$(campo reboot)"
eq true "$(campo ok)"

t "janela fechada no meio (TERM): grava o resumo com o passo em que estava"
zera
migra system 20260101-a.sh
printf '#!/usr/bin/env bash\nkill -TERM "$PPID"\n' >"$share/migrations/user/20260101-fecha.sh"
run <<<"s" >/dev/null
eq 143 $?
eq false "$(campo ok)"
eq ajustes "$(campo step)"
absent "$state/migrations/20260101-fecha.sh"

# ------------------------------------- com o scripts/mare-updates de verdade

t "canal com data anterior à instalada: o Mk não é oferecido nem trocado"
zera
cp "$REPO/shell/mare/scripts/mare-updates" "$share/shell/mare/scripts/mare-updates"
stub checkupdates 'exit 2'
stub timeout 'shift; exec "$@"'
echo "aaaaaaaaaaaa 2026-11-05" >"$cfg/VERSION"
echo "bbbbbbbbbbbb 2026-11-04" >"$SBX/canal/VERSION"
out=$(run </dev/null)
eq 0 $?
has "$out" "em dia (aaaaaaaaaaaa)"
has "$out" "tudo em dia"
eq "$velha" "$(cat "$share/VERSION")"

t "canal com o mesmo commit: em dia"
echo "aaaaaaaaaaaa 2026-11-05" >"$SBX/canal/VERSION"
out=$(run </dev/null)
has "$out" "tudo em dia"

t "canal com outro commit e a mesma data: há versão nova, e ela é instalada"
echo "bbbbbbbbbbbb 2026-11-05" >"$SBX/canal/VERSION"
cp "$REPO/shell/mare/scripts/mare-updates" "$SBX/canal/shell/mare/scripts/mare-updates"
out=$(run <<<"s")
eq 0 $?
has "$out" "nova versão (bbbbbbbbbbbb)"
eq "bbbbbbbbbbbb 2026-11-05" "$(cat "$cfg/VERSION")"
eq "bbbbbbbbbbbb 2026-11-05" "$(cat "$share/VERSION")"
eq true "$(campo ok)"

t "depois de instalada, a mesma versão do canal não é oferecida de novo"
out=$(run </dev/null)
eq 0 $?
has "$out" "em dia (bbbbbbbbbbbb)"
has "$out" "tudo em dia"
rm -f "$SBX/bin/timeout" "$SBX/bin/checkupdates"

# ------------------------------------------------- o registro (num terminal)

if command -v script >/dev/null; then
  # Roda num pseudo-terminal, como aberto pelos Ajustes; as respostas vêm de stdin.
  no_terminal() {
    timeout 60 script -qec "MK_ROOT='$R' bash '$REPO/bin/mk-update' $*" /dev/null
  }

  t "num terminal: a rodada fica em atualizacao-*.log, limpo para ler, e o resumo aponta para ele"
  zera
  info false 1 0
  printf 's\n\n' | no_terminal >/dev/null 2>&1
  logs=("$state"/atualizacao-*.log)
  eq 1 "${#logs[@]}"
  eq "${logs[0]}" "$(campo log)"
  eq true "$(campo ok)"
  has "$(cat "${logs[0]}" 2>/dev/null)" "sistema (sudo pede a sua senha)"
  hasnt "$(cat "${logs[0]}" 2>/dev/null)" $'\r'
  hasnt "$(cat "${logs[0]}" 2>/dev/null)" "COMMAND="
  called "sudo pacman -Su --noconfirm --needed"

  t "num terminal, com Mk novo: uma pergunta só, e o mk-update novo segue no mesmo registro"
  zera
  info true 1 0
  out=$(printf 's\n\n' | no_terminal 2>&1)
  eq 1 "$(grep -c 'Atualizar agora?' <<<"$out")"
  logs=("$state"/atualizacao-*.log)
  eq 1 "${#logs[@]}"
  eq "${logs[0]}" "$(campo log)"
  eq bbbbbbbbbbbb "$(campo to)"
  has "$(cat "${logs[0]}" 2>/dev/null)" "Mk em $nova"
  has "$(cat "${logs[0]}" 2>/dev/null)" "pronto"

  t "num terminal: ficam só os 5 últimos registros"
  zera
  info false 1 0
  mkdir -p "$state"
  for i in 1 2 3 4 5 6 7; do echo velho >"$state/atualizacao-2020010$i-000000.log"; done
  printf 's\n\n' | no_terminal >/dev/null 2>&1
  logs=("$state"/atualizacao-*.log)
  eq 5 "${#logs[@]}"
  absent "$state/atualizacao-20200101-000000.log"
  absent "$state/atualizacao-20200103-000000.log"
  exists "$state/atualizacao-20200104-000000.log"
  exists "$(campo log)"

  t "num terminal: rodada em que você disse não não deixa registro"
  zera
  info false 1 0
  printf 'n\n\n' | no_terminal >/dev/null 2>&1
  eq 0 "$(find "$state" -name 'atualizacao-*.log' 2>/dev/null | wc -l)"
  not_called "sudo"

  t "num terminal: --check não cria registro"
  zera
  info false 1 0
  printf '\n' | no_terminal --check >/dev/null 2>&1
  eq 0 "$(find "$state" -name 'atualizacao-*.log' 2>/dev/null | wc -l)"

  t "num terminal: a rodada que falha também deixa registro, e sai com o erro"
  zera
  info false 1 0
  echo 1 >"$SBX/pacman-rc"
  printf 's\n\n' | no_terminal >/dev/null 2>&1
  [[ $? -ne 0 ]] || _fail "devia sair com erro"
  eq sistema "$(campo step)"
  exists "$(campo log)"
fi

t "um VERSION vazio em ~/.config/mk (desligamento à força) não esconde a versão instalada"
zera
info false 0 0
: >"$cfg/VERSION"
MK_ROOT=$R bash "$REPO/bin/mk-update" --check >/dev/null 2>&1 </dev/null
# O resumo de uma rodada parte da versão de /usr/share/mk.
info true 0 0
out=$(run <<<"s")
eq aaaaaaaaaaaa "$(campo from)"
eq "$nova" "$(cat "$cfg/VERSION")"

t "rodado de dentro de /usr/share/mk (que a rodada troca): sai de lá antes, sem erros de getcwd"
# Visto na VM: de dentro de uma pasta de /usr/share/mk, depois da troca cada
# comando do script reclamava "job-working-directory: getcwd" dezenas de vezes.
zera
info true 0 0
mkdir -p "$share/apps/x"
out=$(cd "$share/apps/x" && MK_ROOT=$R bash "$REPO/bin/mk-update" 2>&1 <<<"s")
eq 0 $?
hasnt "$out" "getcwd"
hasnt "$out" "job-working-directory"
eq "$nova" "$(cat "$cfg/VERSION")"
