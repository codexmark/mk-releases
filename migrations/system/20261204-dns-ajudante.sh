#!/usr/bin/env bash
# A página DNS dos Ajustes troca o servidor de nomes de todas as redes por um
# arquivo em /etc/NetworkManager/conf.d. Quem escreve lá é um ajudante que roda
# como root, pelo pkexec, e por isso precisa morar numa pasta do root:
# /usr/lib/mk/mk-dns. (O /usr/share/mk não serve: a troca do mk-update o deixa
# no nome de quem atualizou.) Aqui só o ajudante é copiado; o DNS de ninguém
# muda.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

src=$share/system/mk/mk-dns
if [[ ! -f $src ]]; then
  echo "aviso: $src não existe; a página DNS dos Ajustes fica sem o ajudante" >&2
  exit 0
fi
dst=$R/usr/lib/mk/mk-dns
if [[ -L $dst ]] || ! cmp -s "$src" "$dst" || [[ $(stat -c %a "$dst") != 755 ]]; then
  rm -f "$dst"
  install -Dm755 "$src" "$dst"
fi
