#!/usr/bin/env bash
# bin/mk-release: do repositório privado ao público, com repositórios git
# temporários em $SBX. Nenhum comando sai de $SBX nem vai à rede: o destino é
# sempre um repositório bare local (MK_RELEASES).
. "$(dirname "$0")/../lib.sh"

git_de_verdade=$(PATH=/usr/bin:/bin command -v git) || skip "falta o git"

# O git do teste: o de verdade, mas só dentro de $SBX e só com caminhos locais.
# Um endereço de rede, ou um repositório fora de $SBX, reprova o arquivo.
export GIT_CONFIG_GLOBAL=$SBX/gitconfig GIT_CONFIG_SYSTEM=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_CEILING_DIRECTORIES=$SBX GIT_ALLOW_PROTOCOL=file GIT_TERMINAL_PROMPT=0
export GIT_AUTHOR_DATE="2026-11-20T12:00:00Z" GIT_COMMITTER_DATE="2026-11-20T12:00:00Z"
printf '[init]\n\tdefaultBranch = main\n' >"$GIT_CONFIG_GLOBAL"
stub git '
onde=$PWD ant=""
for a in "$@"; do
  case $a in *://* | git@*)
    echo "git com endereço de rede: $*" >>"$SBX/perigo"
    exit 97 ;;
  esac
  [[ $ant == -C ]] && onde=$a
  ant=$a
done
case $onde/ in "$SBX"/*) ;; *)
  echo "git fora do SBX (em $onde): $*" >>"$SBX/perigo"
  exit 97 ;;
esac
exec '"$git_de_verdade"' "$@"'

priv=$SBX/priv
pub=$SBX/pub.git
trabalho=$XDG_CACHE_HOME/mk-release
noreply=16628997+codexmark@users.noreply.github.com

# Uma checagem falsa: anota que rodou e sai com o código de $SBX/<nome>-rc.
checagem() { # checagem <caminho> <nome>
  printf '#!/bin/bash\necho %s >>"$SBX/checagens"\nexit "$(cat "$SBX/%s-rc" 2>/dev/null || echo 0)"\n' "$2" "$2" >"$1"
  chmod +x "$1"
}
# O repositório privado, com um commit.
novo_privado() {
  rm -rf "${priv:?}"
  mkdir -p "$priv"/{bin,tests,config}
  cp "$REPO/bin/mk-release" "$priv/bin/mk-release"
  checagem "$priv/bin/check-keybinds" check-keybinds
  checagem "$priv/bin/check-shell" check-shell
  checagem "$priv/tests/run" tests-run
  printf 'segredo.env\ncache/\n' >"$priv/.gitignore"
  echo "conteúdo" >"$priv/arquivo.txt"
  echo "cor = cobre" >"$priv/config/tema.conf"
  git -C "$priv" init -q
  git -C "$priv" config user.name "Fulana de Tal"
  git -C "$priv" config user.email "fulana@privado.example"
  git -C "$priv" add -A
  git -C "$priv" commit -q -m "primeiro commit"
}
# Tudo do zero: o privado com um commit e o público vazio.
novo() {
  rm -rf "${pub:?}" "${trabalho:?}" "${SBX:?}/outro.git" "${SBX:?}/origem.git"
  rm -f "${SBX:?}"/checagens "${SBX:?}"/*-rc
  novo_privado
  git -C "$SBX" init -q --bare "$pub"
  calls_reset
}
commit() { # commit <assunto>
  echo "$1" >>"$priv/arquivo.txt"
  git -C "$priv" commit -q -a -m "$1"
}
lanca() { MK_RELEASES=${DESTINO:-$pub} bash "$priv/bin/mk-release" "$@" 2>&1; }
no_pub() { git -C "$pub" "$@" 2>/dev/null; }
lancamentos() { no_pub rev-list --count main || echo 0; }
src() { git -C "$priv" rev-parse --short=12 HEAD; }

t "o endereço padrão dos lançamentos é o canal que os Mk instalados leem"
canal=$(grep -v '^#' "$REPO/system/mk/channel" | head -1)
has "$(grep '^releases=' "$REPO/bin/mk-release")" "MK_RELEASES:-$canal}"

# ----------------------------------------------------------------- recusas

t "mudança sem commit em arquivo rastreado: recusa, sem checar nem publicar"
novo
echo "mexi" >>"$priv/arquivo.txt"
out=$(lanca)
eq 1 $?
has "$out" "há mudanças sem commit"
absent "$SBX/checagens"
not_called "clone"
not_called "push"
eq 0 "$(lancamentos)"

t "mudança já no 'git add', mas sem commit: recusa também"
novo
echo "mexi" >>"$priv/arquivo.txt"
git -C "$priv" add arquivo.txt
lanca >/dev/null
eq 1 $?
eq 0 "$(lancamentos)"

t "check-keybinds reprova: não lança"
novo
echo 1 >"$SBX/check-keybinds-rc"
lanca >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "clone"
not_called "push"
eq 0 "$(lancamentos)"

t "check-shell reprova: não lança"
novo
echo 1 >"$SBX/check-shell-rc"
lanca >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "push"
eq 0 "$(lancamentos)"

t "tests/run reprova: não lança"
novo
echo 1 >"$SBX/tests-run-rc"
lanca >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "clone"
not_called "push"
eq 0 "$(lancamentos)"

# ------------------------------------------------------ o primeiro lançamento

t "roda check-keybinds, check-shell e os testes, nessa ordem, antes de publicar"
novo
out=$(lanca)
eq 0 $?
eq "check-keybinds
check-shell
tests-run" "$(cat "$SBX/checagens")"

t "VERSION: uma linha 'commit data', com as 12 letras do HEAD e a data dele"
eq "$(src) 2026-11-20" "$(no_pub show main:VERSION)"
like "$(no_pub show main:VERSION)" '^[0-9a-f]{12} [0-9]{4}-[0-9]{2}-[0-9]{2}$'

t "primeiro lançamento: NOVIDADES diz 'Primeiro lançamento.'"
eq "Primeiro lançamento." "$(no_pub show main:NOVIDADES)"

t "o commit público se chama 'Mk <data> (<commit>)' e traz as novidades no corpo"
eq "Mk 2026-11-20 ($(src))" "$(no_pub log -1 --format=%s main)"
eq "Primeiro lançamento." "$(no_pub log -1 --format=%b main)"
has "$out" "lançado: Mk 2026-11-20 ($(src)) em $SBX/pub"

t "o commit público usa o seu nome e o e-mail noreply, nunca o e-mail do repositório privado"
eq "Fulana de Tal" "$(no_pub log -1 --format=%an main)"
eq "$noreply $noreply" "$(no_pub log -1 --format='%ae %ce' main)"
hasnt "$(no_pub log --format='%ae %ce %b' main)" "privado.example"

t "leva os arquivos rastreados, com o modo de execução, e sem .git"
arquivos=$(no_pub ls-tree -r --name-only main)
eq ".gitignore
NOVIDADES
VERSION
arquivo.txt
bin/check-keybinds
bin/check-shell
bin/mk-release
config/tema.conf
tests/run" "$arquivos"
has "$(no_pub ls-tree main bin/check-shell)" "100755"
eq "cor = cobre" "$(no_pub show main:config/tema.conf)"

t "o repositório privado fica como estava (sem mudança, sem commit novo)"
eq "" "$(git -C "$priv" status --porcelain)"
eq 1 "$(git -C "$priv" rev-list --count HEAD)"

t "lançar o mesmo commit de novo: diz que já está lançado e não publica outra vez"
calls_reset
out=$(lanca)
eq 0 $?
has "$out" "o Mk $(src) já está lançado"
eq 1 "$(lancamentos)"
not_called "push"
not_called "commit"

# ------------------------------------------- o que não pode ir a público

t "arquivo ignorado e arquivo novo sem 'git add' não impedem e não são publicados"
novo
echo "SENHA=123" >"$priv/segredo.env"
mkdir -p "$priv/cache"
echo "lixo" >"$priv/cache/x"
echo "rascunho" >"$priv/ainda-nao-rastreado.txt"
out=$(lanca)
eq 0 $?
arquivos=$(no_pub ls-tree -r --name-only main)
hasnt "$arquivos" "segredo.env"
hasnt "$arquivos" "cache/"
hasnt "$arquivos" "ainda-nao-rastreado.txt"
has "$arquivos" "arquivo.txt"
eq "" "$(no_pub grep -l SENHA main)"

t "arquivo ignorado que sobrou na pasta de trabalho de um lançamento anterior não é publicado"
echo "SENHA=456" >"$trabalho/mk-releases/segredo.env"
commit "outro commit"
lanca >/dev/null
eq 0 $?
hasnt "$(no_pub ls-tree -r --name-only main)" "segredo.env"

t "arquivo apagado no privado some do público"
novo
lanca >/dev/null
git -C "$priv" rm -q config/tema.conf
git -C "$priv" commit -q -m "tira o tema"
lanca >/dev/null
eq 0 $?
hasnt "$(no_pub ls-tree -r --name-only main)" "config/tema.conf"
eq 2 "$(lancamentos)"

# ---------------------------------------------------------------- NOVIDADES

t "NOVIDADES: os assuntos dos commits desde o lançamento anterior, um por linha, sem o '- '"
novo
lanca >/dev/null
commit "ilha: mostra a bateria"
commit "ajustes: página de impressoras"
commit "corrige o relógio"
out=$(lanca)
eq 0 $?
eq "corrige o relógio
ajustes: página de impressoras
ilha: mostra a bateria" "$(no_pub show main:NOVIDADES)"

t "NOVIDADES: o corpo do commit público lista os mesmos assuntos"
eq "- corrige o relógio
- ajustes: página de impressoras
- ilha: mostra a bateria" "$(no_pub log -1 --format=%b main)"
eq "$(src) 2026-11-20" "$(no_pub show main:VERSION)"

t "NOVIDADES: 12 commits cabem inteiros"
novo
lanca >/dev/null
for i in 01 02 03 04 05 06 07 08 09 10 11 12; do commit "mudança $i"; done
lanca >/dev/null
news=$(no_pub show main:NOVIDADES)
eq 12 "$(grep -c '' <<<"$news")"
hasnt "$news" "e mais"
eq "mudança 12" "$(head -1 <<<"$news")"
eq "mudança 01" "$(tail -1 <<<"$news")"

t "NOVIDADES: com 15 commits ficam os 11 mais novos e '… e mais 4' (12 linhas)"
for i in 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27; do commit "mudança $i"; done
lanca >/dev/null
news=$(no_pub show main:NOVIDADES)
eq 12 "$(grep -c '' <<<"$news")"
eq "mudança 27" "$(head -1 <<<"$news")"
eq "mudança 17" "$(sed -n 11p <<<"$news")"
eq "… e mais 4" "$(tail -1 <<<"$news")"

t "NOVIDADES: a sua frase entra no lugar da lista de commits (no arquivo e no commit)"
commit "assunto que não é para leitura pública"
out=$(lanca "Impressoras nos Ajustes e a ilha mais calma.")
eq 0 $?
eq "Impressoras nos Ajustes e a ilha mais calma." "$(no_pub show main:NOVIDADES)"
eq "Impressoras nos Ajustes e a ilha mais calma." "$(no_pub log -1 --format=%b main)"
hasnt "$(no_pub log -1 --format=%B main)" "leitura pública"

t "NOVIDADES: frase de várias linhas perde as linhas vazias e respeita as 12"
commit "mais um"
lanca "$(printf 'linha %s\n\n' 1 2 3 4 5 6 7 8 9 10 11 12 13)" >/dev/null
news=$(no_pub show main:NOVIDADES)
eq 12 "$(grep -c '' <<<"$news")"
eq "linha 1" "$(head -1 <<<"$news")"
eq "… e mais 2" "$(tail -1 <<<"$news")"

t "NOVIDADES: o commit do lançamento anterior não existe neste clone: 'Primeiro lançamento.'"
novo
commit "só no histórico antigo"
lanca >/dev/null
novo_privado # outro histórico: o commit lançado não existe aqui
commit "histórico reescrito"
out=$(lanca)
eq 0 $?
eq "Primeiro lançamento." "$(no_pub show main:NOVIDADES)"
eq "$(src) 2026-11-20" "$(no_pub show main:VERSION)"
eq 2 "$(lancamentos)"

# ------------------------------------------------------ e-mail, aviso, destino

t "MK_RELEASE_EMAIL troca o endereço do commit público"
novo
MK_RELEASE_EMAIL=outra@exemplo.test lanca >/dev/null
eq "outra@exemplo.test outra@exemplo.test" "$(no_pub log -1 --format='%ae %ce' main)"

t "commits que ainda não foram para o repositório privado: avisa e lança assim mesmo"
novo
git -C "$SBX" init -q --bare "$SBX/origem.git"
git -C "$priv" remote add origin "$SBX/origem.git"
git -C "$priv" push -q -u origin HEAD:main 2>/dev/null
out=$(lanca)
hasnt "$out" "aviso:"
commit "ainda só aqui"
out=$(lanca)
eq 0 $?
has "$out" "aviso: há commits que ainda não foram para o repositório privado"
eq 2 "$(lancamentos)"
eq 1 "$(git -C "$SBX/origem.git" rev-list --count main)"

t "sem repositório remoto no privado: lança sem aviso"
novo
out=$(lanca)
eq 0 $?
hasnt "$out" "aviso:"

t "MK_RELEASES trocado com a pasta de trabalho de outro destino: publica no destino pedido, não no antigo"
novo
lanca >/dev/null
git -C "$SBX" init -q --bare "$SBX/outro.git"
commit "segunda versão"
out=$(DESTINO=$SBX/outro.git lanca)
eq 0 $?
eq 1 "$(lancamentos)"
eq "$(src) 2026-11-20" "$(git -C "$SBX/outro.git" show main:VERSION 2>/dev/null)"
eq 1 "$(git -C "$SBX/outro.git" rev-list --count main 2>/dev/null)"
has "$out" "em $SBX/outro"

t "o público andou por fora (outro clone lançou): a pasta de trabalho acompanha antes de publicar"
novo
lanca >/dev/null
cp -r "$trabalho" "$SBX/guardada"
commit "segunda"
lanca >/dev/null
rm -rf "${trabalho:?}"
mv "$SBX/guardada" "$trabalho"
commit "terceira"
out=$(lanca)
eq 0 $?
eq 3 "$(lancamentos)"
eq "terceira" "$(no_pub show main:NOVIDADES)"
