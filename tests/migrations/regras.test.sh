#!/usr/bin/env bash
# migrations/: as regras do migrations/README.md, conferidas em todas de uma
# vez (nome, cabeçalho, pacotes declarados e nunca instalados, raiz trocável).
. "$(dirname "$0")/../lib.sh"

dir=$REPO/migrations
mapfile -t all < <(find "$dir/system" "$dir/user" -maxdepth 1 -type f | sort)
mapfile -t sys < <(find "$dir/system" -maxdepth 1 -type f -name '*.sh' | sort)
mapfile -t usr < <(find "$dir/user" -maxdepth 1 -type f -name '*.sh' | sort)

# O corpo sem comentários (linhas inteiras e o resto da linha depois de " #").
corpo() { sed -e '/^[[:space:]]*#/d' -e 's/[[:space:]]#[[:space:]].*$//' "$1"; }
# Os nomes de uma lista de pacotes (sem comentários nem seções).
lista() { sed -e 's/#.*//' -e 's/[[:space:]]*$//' -e '/^\[/d' "$@" | grep -v '^[[:space:]]*$'; }

t "há migrações de sistema e de usuário"
((${#sys[@]} > 0)) || _fail "nenhuma em migrations/system"
((${#usr[@]} > 0)) || _fail "nenhuma em migrations/user"

t "em system/ e user/ só há AAAAMMDD-assunto.sh (o mk-update só enxerga *.sh)"
for f in "${all[@]}"; do
  like "$(basename "$f")" '^[0-9]{8}-[a-z0-9]+(-[a-z0-9]+)*\.sh$' "${f#"$dir"/}"
done

t "a data do nome existe no calendário"
for f in "${all[@]}"; do
  d=$(basename "$f")
  d=${d:0:8}
  [[ $(date -d "${d:0:4}-${d:4:2}-${d:6:2}" +%Y%m%d 2>/dev/null) == "$d" ]] || _fail "data inválida: ${f#"$dir"/}"
done

t "a ordem de nome é a ordem de data, em cada pasta"
for kind in system user; do
  nomes=$(find "$dir/$kind" -maxdepth 1 -name '*.sh' -printf '%f\n' | LC_ALL=C sort)
  datas=$(sed 's/^\(........\).*/\1/' <<<"$nomes")
  eq "$(sort -n <<<"$datas")" "$datas" "$kind"
done

t "cabeçalho: #!/usr/bin/env bash, um comentário dizendo o que muda e set -euo pipefail"
for f in "${all[@]}"; do
  eq '#!/usr/bin/env bash' "$(sed -n 1p "$f")" "${f#"$dir"/}: linha 1"
  like "$(sed -n 2p "$f")" '^# [^[:space:]]' "${f#"$dir"/}: linha 2"
  # O set vem antes de qualquer comando: é a primeira linha que não é comentário.
  eq 'set -euo pipefail' "$(sed -e 1d -e '/^#/d' -e '/^[[:space:]]*$/d' "$f" | head -1)" "${f#"$dir"/}: set"
done

t "nenhuma instala pacote por conta própria (pacman -S, yay, paru no corpo)"
for f in "${all[@]}"; do
  # (o texto de um aviso pode citar o comando: "depois: sudo pacman -S ...")
  ruim=$(corpo "$f" | sed 's/echo "[^"]*"/echo/g' | grep -nE '(^|[;&|({]|\b(then|do|else|sudo|exec))[[:space:]]*(pacman[[:space:]]+(-[A-Za-z]*S|--sync)|yay|paru|makepkg)\b' || true)
  eq "" "$ruim" "${f#"$dir"/} instala pacote"
done

t "a linha '# pacotes:' só tem nomes que o mk-update aceita"
for f in "${sys[@]}"; do
  while read -r p; do
    [[ -z $p ]] || like "$p" '^[a-z0-9@._+-]+$' "${f#"$dir"/}: $p"
  done < <(sed -n 's/^# pacotes:[[:space:]]*//p' "$f" | tr -s '[:space:]' '\n')
done

t "'pacotes:' escrito de outro jeito (recuado, sem espaço, maiúscula) o mk-update não lê"
for f in "${all[@]}"; do
  ruim=$(grep -niE '^[[:space:]]*#[[:space:]]*pacotes?[[:space:]]*:' "$f" | grep -vE '^[0-9]+:# pacotes: ' || true)
  eq "" "$ruim" "${f#"$dir"/}"
done

t "cada pacote declarado está nas listas de system/packages (e não só na do AUR)"
oficiais=$(lista "$REPO"/system/packages/{base,desktop,dev,apps,hardware}.packages)
for f in "${sys[@]}"; do
  while read -r p; do
    [[ -n $p ]] || continue
    grep -qx -- "$p" <<<"$oficiais" || _fail "${f#"$dir"/}: '$p' não está em system/packages (o instalador não o traz)"
  done < <(sed -n 's/^# pacotes:[[:space:]]*//p' "$f" | tr -s '[:space:]' '\n')
done

t "o que vai para a fila do AUR (aur-pendentes.txt) está em aur.packages"
aur=$(lista "$REPO/system/packages/aur.packages")
for f in "${sys[@]}"; do
  while read -r p; do
    [[ -n $p ]] || continue
    grep -qx -- "$p" <<<"$aur" || _fail "${f#"$dir"/}: '$p' não está em aur.packages"
  done < <(corpo "$f" | sed -n 's/.*echo \([a-z0-9@._+-]*\) >>"\$\(pend\|share\/aur-pendentes\.txt\)".*/\1/p')
done

t "migração de usuário não declara pacotes (a linha é ignorada ali)"
for f in "${usr[@]}"; do
  eq "" "$(grep -n '^# pacotes:' "$f" || true)" "${f#"$dir"/}"
done

t "o Mk instalado é lido por \${MK_SHARE:-/usr/share/mk}, nunca pelo caminho fixo"
for f in "${all[@]}"; do
  ruim=$(corpo "$f" | sed 's|\${MK_SHARE:-/usr/share/mk}||g' | grep -n '/usr/share/mk\b' || true)
  eq "" "$ruim" "${f#"$dir"/}"
done

t "todo caminho do sistema passa pela raiz \$R (MK_ROOT), para rodar numa caixa de areia"
for f in "${all[@]}"; do
  limpo=$(corpo "$f" | sed -e 's|\${MK_SHARE:-/usr/share/mk}||g' -e 's|\$R/[A-Za-z0-9_./-]*||g' -e 's|\${R}/[A-Za-z0-9_./-]*||g')
  ruim=$(grep -nE '(^|[^A-Za-z0-9_.~$}/-])/(etc|usr|var|boot|opt|root|home|srv|lib|bin|sbin)(/|\b)' <<<"$limpo" | grep -vE '/dev/null' || true)
  eq "" "$ruim" "${f#"$dir"/}"
  if grep -q '\$R\b' <<<"$(corpo "$f")"; then
    eq 1 "$(grep -c '^R=\${MK_ROOT:-}' "$f")" "${f#"$dir"/}: R=\${MK_ROOT:-}"
  fi
done

t "migração de sistema não mexe em home (\$HOME e \$USER ali são os do root)"
for f in "${sys[@]}"; do
  eq "" "$(corpo "$f" | grep -nE '\$\{?(HOME|USER|XDG_[A-Z_]+)\b|~/' | grep -v "printf '" || true)" "${f#"$dir"/}"
done

t "migração de usuário não pede root"
for f in "${usr[@]}"; do
  eq "" "$(corpo "$f" | grep -nE '\b(sudo|pkexec|su -)\b' || true)" "${f#"$dir"/}"
done

t "nenhuma faz pergunta (read, gum, select)"
for f in "${all[@]}"; do
  eq "" "$(corpo "$f" | grep -nE '(^|[;&|]|then|do)[[:space:]]*(read[[:space:]]+-[a-z]*p|select[[:space:]]|gum[[:space:]]+(confirm|input|choose))' || true)" "${f#"$dir"/}"
done

t "cada migração tem o seu teste em tests/migrations/"
for f in "${all[@]}"; do
  rel=${f#"$dir"/}
  exists "$TESTS/migrations/${rel%.sh}.test.sh"
done

t "cada teste de tests/migrations/ é de uma migração que existe"
while read -r f; do
  rel=${f#"$TESTS/migrations/"}
  exists "$dir/${rel%.test.sh}.sh"
done < <(find "$TESTS/migrations/system" "$TESTS/migrations/user" -name '*.test.sh')

t "o README documenta a raiz MK_ROOT"
has "$(cat "$dir/README.md")" 'MK_ROOT'
