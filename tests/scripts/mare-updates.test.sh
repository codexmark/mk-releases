#!/usr/bin/env bash
# shell/mare/scripts/mare-updates: o JSON das atualizações (Arch, AUR, o canal
# do Mk, a última rodada, reinício pendente, migrações), com checkupdates, yay,
# git e uname falsos e a raiz do sistema em $SBX/root.
. "$(dirname "$0")/../lib.sh"

upd_sh=$REPO/shell/mare/scripts/mare-updates
root=$SBX/root
cfg=$XDG_CONFIG_HOME/mk
st=$XDG_STATE_HOME/mk
kernel=6.9.1-arch1-1
mkdir -p "$root/usr/share/mk/system/mk" "$root/usr/lib/modules/$kernel" "$root/var/log" "$cfg" "$st"

# checkupdates: o que estiver em cu.out, saindo com cu.rc (2 = nada a fazer).
stub checkupdates 'cat "$SBX/cu.out" 2>/dev/null; exit "$(cat "$SBX/cu.rc" 2>/dev/null || echo 2)"'
stub yay 'cat "$SBX/yay.out" 2>/dev/null'
stub uname 'echo 6.9.1-arch1-1'
# git: o canal é a pasta $SBX/canal; canal-fora faz o clone falhar.
stub git '
echo "prompt=${GIT_TERMINAL_PROMPT:-} askpass=${GIT_ASKPASS:-} ssh=${SSH_ASKPASS:-}" >>"$SBX/git.env"
case " $* " in
  *" clone "*)
    [[ ! -e $SBX/canal-fora ]] || exit 128
    for a; do d=$a; done
    mkdir -p "$d" ;;
  *" show HEAD:VERSION "*) cat "$SBX/canal/VERSION" 2>/dev/null ;;
  *" show HEAD:NOVIDADES "*) cat "$SBX/canal/NOVIDADES" 2>/dev/null ;;
  *" rev-parse "*) echo 0123456789ab ;;
esac'

run() { MK_ROOT=$root bash "$upd_sh"; }
# Um PATH só com o básico do sistema, os stubs e os guardas, sem os comandos
# dados: para os ramos "falta a ferramenta".
path_sem() {
  local d c p
  d=$(mktemp -d "$SBX/sem.XXXXXX")
  mkdir "$d/s" "$d/g"
  for c in bash sh env jq awk sed grep head tail cut tr sort wc cat mktemp rm mv mkdir dirname basename date timeout sleep readlink touch ls; do
    p=$(PATH=/usr/bin:/bin command -v "$c") && ln -s "$p" "$d/s/$c"
  done
  cp "$SBX"/guard/* "$d/g/"
  for c; do rm -f "$d/g/$c"; done
  echo "$SBX/bin:$d/g:$d/s"
}
# Recomeça do zero: sem pacotes, sem versão, sem canal, sem histórico.
zera() {
  local f
  for f in cu.out cu.rc yay.out canal canal-fora git.env; do rm -rf "${SBX:?}/$f"; done
  rm -rf "${cfg:?}" "${st:?}" "${root:?}/usr/share/mk" "${root:?}/var"
  mkdir -p "$cfg" "$st" "$root/usr/share/mk/system/mk" "$root/var/log"
  calls_reset
}
canal() { # canal <versão> <data> [novidades]
  mkdir -p "$SBX/canal"
  echo "$1 $2" >"$SBX/canal/VERSION"
  [[ -z ${3:-} ]] || printf '%s' "$3" >"$SBX/canal/NOVIDADES"
  echo "https://exemplo.test/mk.git" >"$cfg/channel"
}

t "sem nada para atualizar: JSON completo, tudo vazio e sem erro"
zera
antes=$(date +%s)
out=$(run)
ok jq -e . <<<"$out" >/dev/null
eq '{"system":[],"aur":[],"mk":{"version":"","date":"","channel":"","latest":"","available":false},"migrations":0,"error":"","last":null,"applied":0,"reboot":false,"news":[]}' \
  "$(jq -c 'del(.checked)' <<<"$out")"
eq 1 "$(wc -l <<<"$out")" "uma linha só"

t "checked é a hora da consulta, em segundos inteiros"
checked=$(jq -r .checked <<<"$out")
like "$checked" '^[0-9]+$'
((checked >= antes && checked <= antes + 30)) || _fail "checked=$checked, antes=$antes"

t "system: cada linha do checkupdates vira {name, from, to}"
zera
printf '%s\n' "linux 6.9.1.arch1-1 -> 6.9.2.arch1-1" "mesa 1:24.1.0-1 -> 1:24.1.1-1" >"$SBX/cu.out"
echo 0 >"$SBX/cu.rc"
out=$(run)
eq '[{"name":"linux","from":"6.9.1.arch1-1","to":"6.9.2.arch1-1"},{"name":"mesa","from":"1:24.1.0-1","to":"1:24.1.1-1"}]' \
  "$(jq -c .system <<<"$out")"
eq '""' "$(jq -c .error <<<"$out")"

t "aur: as linhas do yay -Qua, no mesmo formato"
zera
echo "yay-bin 12.3.0-1 -> 12.3.5-1" >"$SBX/yay.out"
out=$(run)
eq '[{"name":"yay-bin","from":"12.3.0-1","to":"12.3.5-1"}]' "$(jq -c .aur <<<"$out")"
eq '[]' "$(jq -c .system <<<"$out")"
called "yay -Qua"

t "não usa sudo nem pacman: só consulta"
not_called "sudo"
not_called "pacman"

t "linha que não é 'nome velha -> nova' (aviso, linha em branco) fica de fora"
zera
printf '%s\n' ":: aviso qualquer" "" "erro" "bash 5.2-1 -> 5.3-1" >"$SBX/cu.out"
echo 0 >"$SBX/cu.rc"
eq '["bash"]' "$(run | jq -c '.system | map(.name)')"

t "nome de pacote com aspas, barra invertida ou cifrão não quebra o JSON"
zera
printf '%s\n' 'pac"ote\n$(id) 1.0"-1 -> 2.0`x`-1' >"$SBX/cu.out"
echo 0 >"$SBX/cu.rc"
out=$(run)
eq 'pac"ote\n$(id)' "$(jq -r '.system[0].name' <<<"$out")"
eq '2.0`x`-1' "$(jq -r '.system[0].to' <<<"$out")"

t "checkupdates saindo com 1 (sem internet): o erro avisa, e o resto do JSON continua"
zera
echo 1 >"$SBX/cu.rc"
out=$(run)
has "$(jq -r .error <<<"$out")" "não deu para consultar os espelhos"
eq '[]' "$(jq -c .system <<<"$out")"

t "checkupdates saindo com 2 (nada a atualizar) não é erro"
zera
echo 2 >"$SBX/cu.rc"
eq "" "$(run | jq -r .error)"

t "sem o checkupdates instalado: o erro diz o pacote que falta"
zera
mv "$SBX/bin/checkupdates" "$SBX/checkupdates.fora"
out=$(PATH=$(path_sem checkupdates) MK_ROOT=$root bash "$upd_sh")
has "$(jq -r .error <<<"$out")" "falta o pacman-contrib"
eq '[]' "$(jq -c .system <<<"$out")"
mv "$SBX/checkupdates.fora" "$SBX/bin/checkupdates"

t "sem o yay instalado: aur vazio, sem erro"
zera
mv "$SBX/bin/yay" "$SBX/yay.fora"
out=$(PATH=$(path_sem yay) MK_ROOT=$root bash "$upd_sh")
eq '[]' "$(jq -c .aur <<<"$out")"
eq "" "$(jq -r .error <<<"$out")"
mv "$SBX/yay.fora" "$SBX/bin/yay"

# ---------------------------------------------------------------- a versão do Mk

t "mk.version: os 12 primeiros caracteres do commit do VERSION, e a data"
zera
echo "0123456789abcdef0123 2025-03-01" >"$root/usr/share/mk/VERSION"
eq '{"version":"0123456789ab","date":"2025-03-01","channel":"","latest":"","available":false}' "$(run | jq -c .mk)"

t "mk.version: o VERSION de ~/.config/mk vale sobre o de /usr/share/mk"
echo "aaaaaaaaaaaa 2025-04-01" >"$cfg/VERSION"
eq '["aaaaaaaaaaaa","2025-04-01"]' "$(run | jq -c '[.mk.version, .mk.date]')"

t "mk.version: um VERSION vazio em ~/.config/mk não vale: vale o de /usr/share/mk"
# Visto na VM: a máquina foi desligada à força logo depois de uma atualização
# e o arquivo ficou com 0 bytes. A versão vinha vazia, e com versão vazia o Mk
# nunca mais achava versão nova ("em dia ()").
: >"$cfg/VERSION"
eq '["0123456789ab","2025-03-01"]' "$(run | jq -c '[.mk.version, .mk.date]')"
echo "   " >"$cfg/VERSION"
eq '["0123456789ab","2025-03-01"]' "$(run | jq -c '[.mk.version, .mk.date]')"

t "mk.version: os dois VERSION vazios é o mesmo que não ter versão"
: >"$cfg/VERSION"
: >"$root/usr/share/mk/VERSION"
eq '["",""]' "$(run | jq -c '[.mk.version, .mk.date]')"
rm -f "$cfg/VERSION"
echo "0123456789abcdef0123 2025-03-01" >"$root/usr/share/mk/VERSION"
calls_reset

t "sem canal configurado, o git nem é chamado"
not_called "git"

t "mk.channel: a primeira linha que não é comentário; o do usuário vale sobre o do sistema"
zera
mkdir -p "$SBX/canal"
printf '%s\n' "# o canal de lançamentos" "https://sistema.test/mk.git" >"$root/usr/share/mk/system/mk/channel"
eq "https://sistema.test/mk.git" "$(run | jq -r .mk.channel)"
printf '%s\n' "# meu canal" "https://meu.test/mk.git" "https://outro.test/mk.git" >"$cfg/channel"
eq "https://meu.test/mk.git" "$(run | jq -r .mk.channel)"

t "mk.channel: um arquivo do usuário só com comentários deixa valer o do sistema"
echo "# nada aqui" >"$cfg/channel"
eq "https://sistema.test/mk.git" "$(run | jq -r .mk.channel)"

t "canal: clone raso, sem baixar arquivos, e nunca pede senha"
zera
canal bbbbbbbbbbbb 2025-05-01
run >/dev/null
called "clone -q --depth 1 --filter=blob:none --no-checkout https://exemplo.test/mk.git"
called "-c credential.interactive=never clone"
eq "prompt=0 askpass=/bin/false ssh=/bin/false" "$(head -1 "$SBX/git.env")"

t "canal: a pasta temporária do clone não fica para trás"
eq "" "$(ls -A "$TMPDIR")"

t "versão nova no canal: latest e available"
zera
echo "aaaaaaaaaaaa 2025-04-01" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01
eq '["bbbbbbbbbbbb",true]' "$(run | jq -c '[.mk.latest, .mk.available]')"

t "mesma versão no canal (pelos 12 primeiros caracteres): nada disponível"
zera
echo "aaaaaaaaaaaa1111 2025-04-01" >"$cfg/VERSION"
canal aaaaaaaaaaaa2222 2025-04-01
eq false "$(run | jq -c .mk.available)"

t "um Mk mais novo que o último lançamento não 'volta': nada disponível"
zera
echo "cccccccccccc 2025-06-15" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01
eq '["bbbbbbbbbbbb",false]' "$(run | jq -c '[.mk.latest, .mk.available]')"

t "lançamento no mesmo dia, com outro commit: disponível"
zera
echo "aaaaaaaaaaaa 2025-05-01" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01
eq true "$(run | jq -c .mk.available)"

t "sem data de um dos lados, outra versão no canal conta como disponível"
zera
echo "aaaaaaaaaaaa" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01
eq true "$(run | jq -c .mk.available)"

t "sem versão instalada conhecida, nada é oferecido"
zera
canal bbbbbbbbbbbb 2025-05-01
eq '["","bbbbbbbbbbbb",false]' "$(run | jq -c '[.mk.version, .mk.latest, .mk.available]')"

t "canal sem arquivo VERSION: latest é o commit do canal"
zera
echo "aaaaaaaaaaaa 2025-04-01" >"$cfg/VERSION"
canal x y
rm -f "$SBX/canal/VERSION"
eq '["0123456789ab",true]' "$(run | jq -c '[.mk.latest, .mk.available]')"
called "rev-parse --short=12 HEAD"

t "canal fora do ar: o erro diz qual canal, e nada fica disponível"
zera
echo "aaaaaaaaaaaa 2025-04-01" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01
: >"$SBX/canal-fora"
out=$(run)
eq "sem acesso ao canal do Mk (https://exemplo.test/mk.git)" "$(jq -r .error <<<"$out")"
eq '["",false]' "$(jq -c '[.mk.latest, .mk.available]' <<<"$out")"

t "canal fora do ar e espelhos fora do ar: fica o erro dos espelhos"
echo 1 >"$SBX/cu.rc"
has "$(run | jq -r .error)" "espelhos"

t "news: as NOVIDADES do lançamento, sem linhas em branco, quando há versão nova"
zera
echo "aaaaaaaaaaaa 2025-04-01" >"$cfg/VERSION"
canal bbbbbbbbbbbb 2025-05-01 $'Monitores: taxas extras\n\n   \nEmoji "novo" com \\ barra\n'
eq '["Monitores: taxas extras","Emoji \"novo\" com \\ barra"]' "$(run | jq -c .news)"

t "news: no máximo as 12 primeiras linhas"
canal bbbbbbbbbbbb 2025-05-01 "$(seq 1 30)"
eq 12 "$(run | jq '.news | length')"

t "news: sem versão nova para instalar, vem vazio"
echo "bbbbbbbbbbbb 2025-05-01" >"$cfg/VERSION"
eq '[]' "$(run | jq -c .news)"

# ---------------------------------------------------------------- a última rodada

t "last: o resumo da última rodada, e applied é a data dele (inteira)"
zera
echo "log" >"$st/rodada.log"
jq -n --arg log "$st/rodada.log" '{applied: 1700000000.9, ok: true, log: $log}' >"$st/ultima-atualizacao.json"
out=$(run)
eq "$st/rodada.log" "$(jq -r .last.log <<<"$out")"
eq true "$(jq -c .last.ok <<<"$out")"
eq 1700000000 "$(jq -c .applied <<<"$out")"

t "last: registro que já saiu da pasta vem com log vazio"
rm -f "$st/rodada.log"
eq '""' "$(run | jq -c .last.log)"

t "last: resumo ilegível, ou que não é um objeto, vira null e applied 0"
for lixo in '{"applied": 17' '[1,2,3]' '"texto"' ''; do
  printf '%s' "$lixo" >"$st/ultima-atualizacao.json"
  out=$(run)
  eq '[null,0]' "$(jq -c '[.last, .applied]' <<<"$out")" "resumo: $lixo"
done

t "last: applied que não é número positivo vira 0, sem quebrar o JSON"
for v in '"ontem"' '-5' 'null' '"1; rm -rf"'; do
  echo "{\"applied\": $v}" >"$st/ultima-atualizacao.json"
  eq 0 "$(run | jq -c .applied)" "applied: $v"
done

t "applied: um -Syu à mão mais novo que o resumo (pacman.log) vale"
zera
echo '{"applied": 1700000000}' >"$st/ultima-atualizacao.json"
cat >"$root/var/log/pacman.log" <<'LOG'
[2024-01-01T10:00:00+0000] [PACMAN] starting full system upgrade
[2025-03-10T09:00:00-0300] [PACMAN] Running 'pacman -Syu'
[2025-03-10T09:00:00-0300] [PACMAN] starting full system upgrade
[2025-03-10T09:05:00-0300] [ALPM] upgraded linux (6.9.1 -> 6.9.2)
LOG
eq 1741608000 "$(run | jq -c .applied)"

t "applied: um pacman.log mais antigo que o resumo não ganha"
echo '{"applied": 1800000000}' >"$st/ultima-atualizacao.json"
eq 1800000000 "$(run | jq -c .applied)"

t "applied: sem resumo, vale o pacman.log; com data ilegível no log, 0"
rm -f "$st/ultima-atualizacao.json"
eq 1741608000 "$(run | jq -c .applied)"
echo '[data torta] [PACMAN] starting full system upgrade' >"$root/var/log/pacman.log"
eq 0 "$(run | jq -c .applied)"

# ---------------------------------------------------------------- reinício e migrações

t "reboot: falso com os módulos do kernel em uso no disco, verdadeiro se já saíram"
zera
eq false "$(run | jq -c .reboot)"
mv "$root/usr/lib/modules/$kernel" "$root/usr/lib/modules/6.9.2-arch1-1"
eq true "$(run | jq -c .reboot)"
mv "$root/usr/lib/modules/6.9.2-arch1-1" "$root/usr/lib/modules/$kernel"

t "migrations: conta as que ainda não rodaram, do sistema e do usuário"
zera
mkdir -p "$root/usr/share/mk/migrations/system" "$root/usr/share/mk/migrations/user" \
  "$root/var/lib/mk/migrations" "$st/migrations"
touch "$root/usr/share/mk/migrations/system/"{001-a.sh,002-b.sh,003-c.sh,LEIAME.md} \
  "$root/usr/share/mk/migrations/user/"{001-u.sh,002-v.sh}
eq 5 "$(run | jq -c .migrations)"
touch "$root/var/lib/mk/migrations/001-a.sh" "$root/var/lib/mk/migrations/002-b.sh" "$st/migrations/001-u.sh"
eq 2 "$(run | jq -c .migrations)"

t "migrations: a marca de uma do sistema não vale para a do usuário de mesmo nome"
zera
mkdir -p "$root/usr/share/mk/migrations/system" "$root/usr/share/mk/migrations/user" "$root/var/lib/mk/migrations"
touch "$root/usr/share/mk/migrations/system/001-x.sh" "$root/usr/share/mk/migrations/user/001-x.sh" \
  "$root/var/lib/mk/migrations/001-x.sh"
eq 1 "$(run | jq -c .migrations)"
