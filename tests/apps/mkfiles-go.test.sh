#!/usr/bin/env bash
# apps/mkfiles/src: o motor do MkFiles (listar, tipos, operações, lixeira,
# busca, miniaturas, discos), em Go. Os testes dele são `go test` (tudo em
# pasta temporária, com bancos e comandos de mentira); aqui eles entram na
# suíte do Mk, junto com o formato, o go vet, as travas de segurança lidas no
# código, a conferência de que o binário é o do código que está no repositório
# e o binário de verdade respondendo ao contrato dentro da caixa de areia.
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

app=$REPO/apps/mkfiles
src=$app/src
bin=$app/bin/mkfiles-motor
contrato=$(<"$app/CONTRATO.md")
export GOCACHE=${MK_GOCACHE:-/var/tmp/mk-gocache-$(id -u)}
# -buildvcs=false: sem isso o go chama o git (que aqui é um guarda).
export GOFLAGS="-mod=mod -buildvcs=false" GOPROXY=off GOTOOLCHAIN=local
# O motor só enxerga a caixa de areia: os apps e os tipos são os daqui.
export XDG_DATA_DIRS=$SBX/dados XDG_CONFIG_DIRS=$SBX/xdg MKFILES_MIME=$SBX/mime
# Os compactadores de fora nunca rodam de verdade aqui.
export MKFILES_XZ=$SBX/nao-ha MKFILES_ZSTD=$SBX/nao-ha MKFILES_7Z=$SBX/nao-ha MKFILES_UNRAR=$SBX/nao-ha MKFILES_BSDTAR=$SBX/nao-ha
# Os remotos: a pasta do gvfs e a do ssh são as daqui, e o gio é sempre um stub (o de verdade é um guarda).
export MKFILES_GVFS=$SBX/gvfs MKFILES_SSH_DIR=$SBX/ssh MKFILES_REMOTO_MS=4000
mkdir -p "$SBX/gvfs" "$SBX/ssh"
export MKFILES_MOUNTINFO=$SBX/mountinfo MKFILES_ETC=$SBX/etc MKFILES_MINIATURIZADORES=$SBX/miniaturizadores
mkdir -p "$SBX"/{dados/applications,xdg,mime,etc,miniaturizadores}
echo "1 0 8:1 / / rw - ext4 /dev/raiz rw" >"$MKFILES_MOUNTINFO"
printf '50:text/plain:*.txt\n50:image/jpeg:*.jpg\n50:application/x-compressed-tar:*.tar.gz\n50:application/gzip:*.gz\n' >"$SBX/mime/globs2"
cd "$src" || exit 1

t "o módulo não tem dependência de fora: só a biblioteca padrão"
eq "" "$(grep -E '^\s*require|^\s+[a-z0-9.-]+\.[a-z]+/' go.mod || true)"
absent "$src/go.sum"

t "gofmt: todo arquivo está formatado"
eq "" "$(gofmt -l . 2>&1)"

t "go vet: nenhum aviso"
out=$(go vet ./... 2>&1) || _fail "$out"

t "go test: a suíte do motor passa"
out=$(go test ./... 2>&1) || _fail "$(tail -40 <<<"$out")"
hasnt "$out" "FAIL"

t "todo pacote do motor tem teste"
n=0
while read -r pacote; do
  n=$((n + 1))
  compgen -G "$src/$pacote"'/*_test.go' >/dev/null || _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mkfiles$' | sed 's|^mkfiles/||')
ok test "$n" -ge 17
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "nenhum teste do motor escreve fora de pasta temporária"
# Todo teste que mexe em disco passa pelo arnês (prova.Novo) ou por t.TempDir.
mapfile -t provas < <(find "$src" -name '*_test.go' | sort)
for f in "${provas[@]}"; do
  grep -qE 'prova\.Novo\(|Novo\(t,|t\.TempDir\(\)|motor\(t\)|func Test(DoConteudo|Permissoes|Familia|Normal|Casa|PedidoDe)' "$f" || _fail "teste sem caixa de areia: ${f#"$src"/}"
done
eq "" "$(grep -nE 'os\.(Remove|RemoveAll|WriteFile|Rename|Mkdir|MkdirAll)\("/' "${provas[@]}" || true)"

t "o motor nunca passa texto por shell"
eq "" "$(grep -nE '"(sh|bash|dash|zsh|/bin/sh|/bin/bash|/usr/bin/env)"' "${fontes[@]}" || true)"
# Só estas áreas rodam programa de fora: miniaturizador, lsblk/udisksctl, wl-copy/wl-paste, os compactadores (xz, zstd, 7z, unrar) e o gio (remotos).
eq "area.go discos.go gio.go leitor.go miniatura.go tarefas.go" "$(grep -lE 'exec\.Command' "${fontes[@]}" | xargs -r -n1 basename | sort -u | tr '\n' ' ' | sed 's/ $//')"
# E nenhum deles sem prazo.
while read -r f; do
  grep -qE 'exec\.CommandContext|time\.AfterFunc|context\.WithTimeout|WaitDelay' "$f" || _fail "comando sem limite de tempo: ${f#"$src"/}"
done < <(grep -lE 'exec\.Command' "${fontes[@]}")

t "apagar: nenhum RemoveAll no motor, e a remoção recursiva passa pela recusa de segurança sem seguir links"
eq "" "$(grep -nE 'RemoveAll' "${fontes[@]}" || true)"
arvore=$(<"$src/internal/operar/arvore.go")
has "$arvore" "base.Protegido(caminho)"
has "$arvore" "os.Lstat(caminho)"
ok grep -q 'a raiz do sistema não pode ser mexida' "$src/internal/base/base.go"
ok grep -q 'a pasta pessoal inteira não pode ser mexida' "$src/internal/base/base.go"

t "o que o motor escreve fora das operações é atômico"
for f in internal/prefs/prefs.go internal/lugares/favoritos.go internal/abrir/desktop.go; do
  ok grep -q 'base.GravaAtomico' "$src/$f"
done

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkfiles/pack depois de mexer em src/)"
eq "$("$app/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "sem subcomando: o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkfiles-motor <'

t "sobre: todo pedido do motor está no contrato, e todo pedido do contrato existe no motor"
sobre=$("$bin" sobre)
mapfile -t ops < <(jq -r '.ops[]' <<<"$sobre")
ok test "${#ops[@]}" -ge 65
for op in "${ops[@]}"; do
  grep -qE "(\`|\"op\": \")$op(\`|\")" <<<"$contrato" || _fail "pedido fora do contrato: $op"
done
while read -r op; do
  [[ " ${ops[*]} " == *" $op "* ]] || _fail "o contrato tem, o motor não: $op"
done < <(grep -oE '"op": "[a-z-]+"' <<<"$contrato" | sed 's/.*: "//; s/"$//' | sort -u)
jeq '{"lote":true,"compactados":true,"previa":true,"remotos":true}' "$(jq -c .extensoes <<<"$sobre")"
# O lote, os compactados, a prévia e os remotos existem.
for op in renomear-lote compactar extrair previa compactado-listar remoto-montar remotos; do
  [[ " ${ops[*]} " == *" $op "* ]] || _fail "falta o pedido: $op"
done

t "as variáveis de ambiente do contrato são as que o código lê"
for v in MKFILES_MIME MKFILES_ETC MKFILES_MOUNTINFO MKFILES_MINIATURIZADORES MKFILES_LSBLK MKFILES_UDISKSCTL MKFILES_WLCOPY \
  MKFILES_WLPASTE MKFILES_TERMINAL MKFILES_LOTE MKFILES_DISCOS_MS MKFILES_GIO MKFILES_GVFS MKFILES_SSH_DIR MKFILES_REMOTO_MS MKFILES_RELER_MS; do
  has "$contrato" "\`$v\`"
  grep -qF "\"$v\"" "${fontes[@]}" || _fail "o código não lê $v"
done
while read -r v; do
  has "$contrato" "\`$v\`" "variável fora do contrato: $v"
done < <(grep -ohE '"MKFILES_[A-Z_]+"' "${fontes[@]}" | tr -d '"' | sort -u)

# --- o binário de verdade, dentro da caixa de areia -------------------------
p=$HOME/pasta
mkdir -p "$p/sub"
printf 'olá' >"$p/nota.txt"
printf 'x' >"$p/.oculto"
ln -s /nao/existe "$p/quebrado"

t "listar: os 16 campos do item, o tipo pelo nome e o link quebrado"
out=$("$bin" listar "$p")
eq 0 $?
eq "true 4" "$(jq -r '"\(.fim) \(.total)"' <<<"$out")"
eq "alvo alvo_tipo caminho dono grupo icone icone_reserva mime modo mtime nome oculto permissoes quebrado tamanho tipo" \
  "$(jq -r '.itens[0] | keys | join(" ")' <<<"$out")"
eq "arquivo text/plain text-plain 4" "$(jq -r '.itens[] | select(.nome == "nota.txt") | "\(.tipo) \(.mime) \(.icone) \(.tamanho)"' <<<"$out")"
eq "link true /nao/existe" "$(jq -r '.itens[] | select(.nome == "quebrado") | "\(.tipo) \(.quebrado) \(.alvo)"' <<<"$out")"
eq "true" "$(jq -r '.itens[] | select(.nome == ".oculto") | .oculto' <<<"$out")"
eq "pasta folder" "$(jq -r '.itens[] | select(.nome == "sub") | "\(.tipo) \(.icone)"' <<<"$out")"

t "listar: o que não existe sai com 1 e o motivo"
out=$("$bin" listar "$HOME/nao-ha")
eq 1 $?
eq "nao-existe" "$(jq -r .motivo <<<"$out")"

t "validação: caminho relativo, com .., com // ou com byte nulo é recusado, não consertado"
pede() { printf '%s\n' "$1" | "$bin" servir; }
for c in 'pasta' "$p/../pasta" "$p//sub" "$p/"; do
  out=$(pede "$(jq -cn --arg c "$c" '{id: 1, op: "listar", caminho: $c}')")
  like "$(jq -r .erro <<<"$out")" 'caminho' "caminho torto: $c"
done
out=$(pede '{"id":1,"op":"listar","caminho":"/a\u0000b"}')
has "$(jq -r .erro <<<"$out")" "byte nulo"
for nome in 'a/b' '..' '.' ''; do
  out=$(pede "$(jq -cn --arg d "$p" --arg n "$nome" '{id: 1, op: "criar-pasta", destino: $d, nome: $n}')")
  ok jq -e '.erro != null' <<<"$out" >/dev/null
done
eq "nota.txt quebrado sub" "$(ls "$p" | tr '\n' ' ' | sed 's/ $//')"

t "segurança: apagar, lixeira, mover e renomear a raiz, a pasta pessoal ou o que a contém saem com 2 e não tocam em nada"
for alvo in / "$HOME" "$(dirname "$HOME")"; do
  for op in apagar lixeira; do
    out=$("$bin" "$op" "$alvo")
    eq 2 $? "$op $alvo"
    eq "true" "$(jq -r .recusado <<<"$out")" "$op $alvo"
  done
  out=$("$bin" mover "$alvo" "--destino=$p/sub")
  eq 2 $? "mover $alvo"
  out=$("$bin" renomear "$alvo" --nome=outro)
  eq 2 $? "renomear $alvo"
done
out=$(pede '{"id":1,"op":"apagar","origens":[""]}')
ok jq -e '.erro != null' <<<"$out" >/dev/null
out=$(pede '{"id":1,"op":"apagar","origens":[]}')
ok jq -e '.erro != null' <<<"$out" >/dev/null
exists "$p/nota.txt"
exists "$HOME"

t "copiar na mesma pasta cria «nome (2).ext», com a mesma regra do .js da janela"
if command -v node >/dev/null; then
  cat >"$SBX/livre.mjs" <<MJS
import { load } from "$TESTS/js/load.mjs";
const A = load("$app/qml/Arquivos.js");
const [modo, ...resto] = process.argv.slice(2);
if (modo === "livre")
    console.log(A.nomeLivre(resto[0], (n) => n === resto[0], resto[1] === "pasta"));
else
    console.log(A.uri(resto[0]));
MJS
  mkdir -p "$HOME/nomes"
  for nome in "foto.jpg" "pacote.tar.gz" "sem-extensao" ".oculto" "relatório (2).txt" "a.b.c.txt"; do
    printf 'x' >"$HOME/nomes/$nome"
    out=$("$bin" copiar "$HOME/nomes/$nome" "--destino=$HOME/nomes")
    criado=$(jq -rs '[.[] | select(.estado == "feita") | .criados[0]] | last' <<<"$out")
    eq "$HOME/nomes/$(node "$SBX/livre.mjs" livre "$nome" arquivo)" "$criado" "nome livre de: $nome"
    exists "$criado"
  done
  mkdir "$HOME/nomes/pasta.v1"
  out=$("$bin" copiar "$HOME/nomes/pasta.v1" "--destino=$HOME/nomes")
  eq "$HOME/nomes/$(node "$SBX/livre.mjs" livre "pasta.v1" pasta)" "$(jq -rs '[.[] | select(.estado == "feita") | .criados[0]] | last' <<<"$out")"
else
  echo "  # node ausente: pulado" >&2
fi

t "favoritos: o arquivo é o gtk-3.0/bookmarks, e o URI sai igual ao do .js da janela"
mkdir -p "$HOME/Meus Projetos/ação #1"
out=$("$bin" favorito-por "$HOME/Meus Projetos/ação #1")
eq 0 $?
uri=$(jq -r '.favoritos[0].uri' <<<"$out")
like "$uri" '^file:///.*/Meus%20Projetos/a%C3%A7%C3%A3o%20%231$'
has "$(<"$XDG_CONFIG_HOME/gtk-3.0/bookmarks")" "$uri"
if command -v node >/dev/null; then
  eq "$uri" "$(node "$SBX/livre.mjs" uri "$HOME/Meus Projetos/ação #1")"
fi

t "lixeira: manda, lista de onde veio, restaura; a ficha segue o freedesktop"
printf 'fora' >"$p/vai embora.txt"
out=$("$bin" lixeira "$p/vai embora.txt")
eq 0 $?
absent "$p/vai embora.txt"
exists "$XDG_DATA_HOME/Trash/files/vai embora.txt"
ficha=$(<"$XDG_DATA_HOME/Trash/info/vai embora.txt.trashinfo")
has "$ficha" "[Trash Info]"
has "$ficha" "Path=$p/vai%20embora.txt"
like "$ficha" 'DeletionDate=[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}'
out=$("$bin" lixeira-listar)
eq "vai embora.txt|$p/vai embora.txt" "$(jq -r '.itens[0] | "\(.nome_original)|\(.origem)"' <<<"$out")"
eq 1 "$("$bin" lixeira-estado | jq -r .itens)"
ok "$bin" restaurar "$XDG_DATA_HOME/Trash/files/vai embora.txt" >/dev/null
eq "fora" "$(<"$p/vai embora.txt")"
eq 0 "$("$bin" lixeira-estado | jq -r .itens)"

t "apagar de vez: o link some, o alvo de fora fica"
mkdir -p "$HOME/fora" "$HOME/some"
printf 'fica' >"$HOME/fora/precioso.txt"
ln -s "$HOME/fora" "$HOME/some/atalho"
ok "$bin" apagar "$HOME/some" >/dev/null
absent "$HOME/some"
eq "fica" "$(<"$HOME/fora/precioso.txt")"

t "copiar uma pasta para dentro dela mesma é recusado"
out=$("$bin" copiar "$p" "--destino=$p/sub")
eq 1 $?
has "$(jq -r .erro <<<"$out")" "dentro dela mesma"
eq "" "$(ls "$p/sub")"

t "busca: sem diferenciar maiúsculas nem acentos, com curinga"
mkdir -p "$HOME/busca/fundo"
printf 'x' >"$HOME/busca/fundo/Relatório Final.TXT"
printf 'x' >"$HOME/busca/outra coisa.txt"
eq "Relatório Final.TXT" "$("$bin" buscar "$HOME/busca" --nome='relat*rio*' | jq -rs '[.[].itens[].nome] | join(",")')"
eq "Relatório Final.TXT" "$("$bin" buscar "$HOME/busca" --nome=RELATORIO | jq -rs '[.[].itens[].nome] | join(",")')"

t "abrir: o argv vem do .desktop, sem shell, com o arquivo inteiro num argumento só"
cat >"$SBX/dados/applications/visor.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Visor
Exec=visor --abrir %f
MimeType=text/plain;
DESKTOP
# shellcheck disable=SC2016 # é o texto, não a expansão, que tem de chegar
feio=$HOME/'um $(touch pwn); `x` "a".txt'
printf 'x' >"$feio"
out=$("$bin" abrir "$feio")
eq 0 $?
jeq "$(jq -cn --arg f "$feio" '["visor", "--abrir", $f]')" "$(jq -c '.lancar[0].argv' <<<"$out")"
eq "visor.desktop" "$(jq -r '.lancar[0].app' <<<"$out")"
absent "$src/pwn"
ok "$bin" definir-padrao --mime=text/plain --app=visor.desktop >/dev/null
has "$(<"$XDG_CONFIG_HOME/mimeapps.list")" "text/plain=visor.desktop"

t "discos: a lista vem do lsblk (aqui, de mentira), e montar chama o udisksctl sem privilégio"
stub lsblk 'cat <<JSON
{"blockdevices":[{"name":"sdb","path":"/dev/sdb","label":null,"fstype":null,"size":16000000000,"mountpoints":[null],"rm":true,"hotplug":true,"type":"disk","uuid":null,
 "children":[{"name":"sdb1","path":"/dev/sdb1","label":"PENDRIVE","fstype":"vfat","size":15999000000,"mountpoints":[null],"rm":true,"hotplug":true,"type":"part","uuid":"AB12-CD34"}]},
 {"name":"sda","path":"/dev/sda","label":null,"fstype":"ext4","size":500000000000,"mountpoints":["/"],"rm":false,"hotplug":false,"type":"disk","uuid":"x"}]}
JSON'
stub udisksctl 'echo "Mounted /dev/sdb1 at /run/media/ana/PENDRIVE"'
out=$("$bin" discos)
eq 0 $?
eq "/dev/sdb1 PENDRIVE vfat true" "$(jq -r '.discos[] | "\(.dispositivo) \(.nome) \(.sistema) \(.removivel)"' <<<"$out")"
called "lsblk --json"
ok "$bin" montar --dispositivo=/dev/sdb1 >/dev/null
called "udisksctl mount -b /dev/sdb1"
calls_reset
out=$("$bin" montar '--dispositivo=/dev/sdb1; rm -rf x')
eq 1 $?
out=$("$bin" montar --dispositivo=/dev/sdz9)
eq 1 $?
not_called "udisksctl"

t "área de transferência: o texto vai pelo stdin do wl-copy, com o tipo certo"
stub wl-copy 'cat >"$SBX/copiado"'
ok "$bin" area-por --tipo=x-special/gnome-copied-files "--texto=cut
file:///home/ana/a.txt" >/dev/null
called "wl-copy --type x-special/gnome-copied-files"
eq "cut
file:///home/ana/a.txt" "$(<"$SBX/copiado")"
out=$("$bin" area-por --tipo=application/x-qualquer --texto=x)
eq 1 $?

t "prefs: o padrão do contrato, e a gravação fica em ~/.config/mk/mkfiles/preferencias.json"
jeq '{"visao":"icones","visao_por_pasta":{},"ordem":{"campo":"nome","inversa":false},"ocultos":false,"icone_px":64,"pastas_antes":true,"confirmar_lixeira":false,"clique":"duplo","colunas":{"tamanho":90,"tipo":150,"mtime":150,"permissoes":100},"miniaturas_remotas":true,"miniatura_remota_mb":5}' \
  "$("$bin" prefs | jq -c .prefs)"
ok "$bin" prefs-gravar '--prefs={"visao":"lista","icone_px":9999,"inventada":1}' >/dev/null
eq "lista 64 null" "$(jq -r '"\(.visao) \(.icone_px) \(.inventada)"' "$XDG_CONFIG_HOME/mk/mkfiles/preferencias.json")"

t "renomear em lote: troca em ciclo numa tarefa só, e um conflito recusa tudo antes de mexer"
mkdir -p "$HOME/lote"
printf 'A' >"$HOME/lote/a.txt"
printf 'B' >"$HOME/lote/b.txt"
printf 'C' >"$HOME/lote/c.txt"
out=$("$bin" renomear-lote "--pares=$(jq -cn --arg p "$HOME/lote" '[[$p + "/a.txt", "b.txt"], [$p + "/b.txt", "a.txt"]]')")
eq 0 $?
eq "B A" "$(<"$HOME/lote/a.txt") $(<"$HOME/lote/b.txt")"
eq "a.txt b.txt c.txt" "$(ls -A "$HOME/lote" | tr '\n' ' ' | sed 's/ $//')"
out=$("$bin" renomear-lote "--pares=$(jq -cn --arg p "$HOME/lote" '[[$p + "/a.txt", "c.txt"]]')")
eq 1 $?
eq "existe" "$(jq -r '.conflitos[0].motivo' <<<"$out")"
eq "B C" "$(<"$HOME/lote/a.txt") $(<"$HOME/lote/c.txt")"

t "compactados: compactar, listar por dentro e extrair dão a mesma árvore, pela biblioteca padrão"
mkdir -p "$HOME/zip/origem/sub" "$HOME/zip/saida"
printf 'um' >"$HOME/zip/origem/a.txt"
printf 'dois' >"$HOME/zip/origem/sub/b.txt"
ok "$bin" compactar "$HOME/zip/origem" "--destino=$HOME/zip" --nome=pacote.zip --formato=zip >/dev/null
exists "$HOME/zip/pacote.zip"
eq "zip true" "$("$bin" compactado-info "$HOME/zip/pacote.zip" | jq -r '"\(.formato) \(.disponivel)"')"
eq "origem" "$("$bin" compactado-listar "$HOME/zip/pacote.zip" | jq -rs '[.[].itens[].interno] | join(",")')"
eq "origem/a.txt,origem/sub" "$("$bin" compactado-listar "$HOME/zip/pacote.zip" --dentro=origem | jq -rs '[.[].itens[].interno] | sort | join(",")')"
ok "$bin" extrair "$HOME/zip/pacote.zip" "--destino=$HOME/zip/saida" --modo=aqui >/dev/null
eq "um dois" "$(<"$HOME/zip/saida/origem/a.txt") $(<"$HOME/zip/saida/origem/sub/b.txt")"
eq "" "$(find "$HOME/zip" -name '.mkfiles-*')"

t "compactados: sem a ferramenta de fora, o motor diz o pacote que falta"
printf '\xfd7zXZ\x00\x00' >"$HOME/zip/x.tar.xz"
out=$("$bin" compactado-info "$HOME/zip/x.tar.xz")
eq "tar.xz false xz" "$(jq -r '"\(.formato) \(.disponivel) \(.falta)"' <<<"$out")"

t "prévia: o texto vem cortado em 200 KiB, e um binário não vem"
printf 'olá, prévia\n' >"$HOME/previa.txt"
eq "texto false olá, prévia" "$("$bin" previa "$HOME/previa.txt" | jq -r '"\(.classe) \(.binario) \(.texto)"' | head -1)"
printf 'a\x00b' >"$HOME/nulo.txt"
eq "true null" "$("$bin" previa "$HOME/nulo.txt" | jq -r '"\(.binario) \(.texto)"')"

t "remotos: o gio (aqui, um stub) monta, a senha vai só pelo stdin e o remoto vira uma pasta comum"
# shellcheck disable=SC2016 # o corpo do stub é expandido quando ele roda
stub gio 'if [[ $1 == mount && $2 != -u ]]; then
  printf "User [ana]: "; read -r u; printf "Password: "; read -r s
  [[ $s == "segredo do teste" ]] || { echo "Permission denied"; exit 2; }
  mkdir -p "$MKFILES_GVFS/sftp:host=servidor,user=$u/docs"
  printf x >"$MKFILES_GVFS/sftp:host=servidor,user=$u/docs/remoto.txt"
elif [[ $1 == mount ]]; then
  rm -r "${MKFILES_GVFS:?}/sftp:host=servidor,user=ana"
fi'
eq "$SBX/gvfs" "$("$bin" sobre | jq -r .gvfs)"
coproc REMOTO { "$bin" servir 2>"$SBX/remoto.err"; }
exec {dele}<&"${REMOTO[0]}"
ler() {
  local linha
  while read -r -t 10 linha <&"$dele"; do
    printf '%s\n' "$linha" >>"$SBX/remoto.out"
    if jq -e "$1" <<<"$linha" >/dev/null 2>&1; then
      printf '%s\n' "$linha"
      return 0
    fi
  done
  return 1
}
echo '{"id":1,"op":"remoto-montar","uri":"sftp://servidor/docs","anonimo":false}' >&"${REMOTO[1]}"
p=$(ler '.evento == "pergunta-remota"')
eq "usuario ana 1" "$(jq -r '"\(.tipo) \(.padrao) \(.pedido)"' <<<"$p")"
echo '{"id":2,"op":"remoto-responder","pedido":1,"pergunta":1,"resposta":"ana"}' >&"${REMOTO[1]}"
p=$(ler '.evento == "pergunta-remota"')
eq "senha false" "$(jq -r '"\(.tipo) \(.de_novo)"' <<<"$p")"
echo '{"id":3,"op":"remoto-responder","pedido":1,"pergunta":2,"resposta":"segredo do teste"}' >&"${REMOTO[1]}"
r=$(ler '.id == 1')
raiz=$SBX/gvfs/sftp:host=servidor,user=ana
eq "$raiz/docs servidor" "$(jq -r '"\(.caminho) \(.nome)"' <<<"$r")"
# Dali em diante é uma pasta: listar funciona sem nada de rede.
jq -cn --arg c "$raiz/docs" '{id: 4, op: "listar", caminho: $c}' >&"${REMOTO[1]}"
eq "remoto.txt" "$(ler '.id == 4 and .fim' | jq -r '.itens[0].nome')"
# A senha não aparece em nada que o motor escreveu, nem nos argumentos do gio.
hasnt "$(<"$SBX/remoto.out")$(<"$SBX/remoto.err")" "segredo do teste"
hasnt "$(<"$SBX/calls")" "segredo do teste"
called "gio mount sftp://servidor/docs"
entrada=${REMOTO[1]}
exec {entrada}>&-
wait "$REMOTO_PID"

t "remotos: sem lixeira (a janela pergunta), a raiz do remoto não se apaga, e desmontar é pelo gio"
out=$("$bin" lixeira "$raiz/docs/remoto.txt")
eq "$raiz/docs/remoto.txt" "$(jq -rs '[.[] | select(.sem_lixeira != null) | .sem_lixeira[]] | last' <<<"$out")"
exists "$raiz/docs/remoto.txt"
out=$("$bin" apagar "$raiz")
eq 2 $?
out=$("$bin" apagar "$SBX/gvfs")
eq 2 $?
exists "$raiz/docs/remoto.txt"
eq "sftp://ana@servidor/ sftp" "$("$bin" remotos | jq -r '.remotos[0] | "\(.uri) \(.protocolo)"')"
ok "$bin" remoto-desmontar --uri=sftp://ana@servidor/ >/dev/null
called "gio mount -u sftp://ana@servidor/"
absent "$raiz"
calls_reset
out=$("$bin" remoto-montar '--uri=--help')
eq 1 $?
eq "endereco" "$(jq -r .motivo <<<"$out")"
out=$("$bin" remoto-montar '--uri=sftp://ana:senha-no-endereco@servidor/')
eq "endereco" "$(jq -r .motivo <<<"$out")"
hasnt "$out" "senha-no-endereco"
not_called "gio"

t "remotos: os hosts do ssh config viram destinos de sftp"
printf 'Host casa\n  HostName 192.168.0.10\n  User ana\nHost *\n  User todos\n' >"$SBX/ssh/config"
eq "casa 192.168.0.10 ana null" "$("$bin" remoto-hosts | jq -r '.hosts[] | "\(.apelido) \(.host) \(.usuario) \(.porta)"')"

t "servir: responde cada pedido na hora, solta os eventos sozinho e sai com 0 quando o stdin fecha"
coproc MOTOR { "$bin" servir; }
le() {
  # Lê linhas até uma que case com o filtro do jq (ou estourar o tempo).
  local linha
  while read -r -t 10 linha <&"$saida"; do
    if jq -e "$1" <<<"$linha" >/dev/null 2>&1; then
      printf '%s\n' "$linha"
      return 0
    fi
  done
  return 1
}
# O descritor do coproc não chega a um subshell: uma cópia chega.
exec {saida}<&"${MOTOR[0]}"
mkdir -p "$HOME/vigiada"
jq -cn --arg c "$HOME/vigiada" '{id: 1, op: "vigiar", caminho: $c}' >&"${MOTOR[1]}"
eq "true" "$(le '.id == 1' | jq -r .vigiando)"
jq -cn --arg c "$HOME/vigiada" '{id: 2, op: "criar-pasta", destino: $c, nome: "Nova pasta"}' >&"${MOTOR[1]}"
eq "1" "$(le '.id == 2' | jq -r .tarefa)"
evento=$(le '.evento == "pasta"')
eq "Nova pasta pasta" "$(jq -r '.mudou[0] | "\(.nome) \(.tipo)"' <<<"$evento")"
echo '{"id":3,"op":"desfazer"}' >&"${MOTOR[1]}"
ok le '.evento == "pasta" and (.saiu | index("Nova pasta") != null)' >/dev/null
absent "$HOME/vigiada/Nova pasta"
entrada=${MOTOR[1]}
exec {entrada}>&-
wait "$MOTOR_PID"
eq 0 $?
