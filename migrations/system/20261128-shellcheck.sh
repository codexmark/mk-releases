#!/usr/bin/env bash
# O lint dos testes do Mk (tests/run) usa o shellcheck, que não vinha: num Mk
# instalado essa checagem era pulada. Quem instala é o mk-update, junto com a
# atualização do sistema.
# pacotes: shellcheck
set -euo pipefail
pacman -Q shellcheck &>/dev/null ||
  echo "aviso: o shellcheck não foi instalado; o tests/run pula o lint de shell" >&2
