#!/usr/bin/env bash
# O painel de emoji digita o escolhido na janela em foco: precisa do wtype.
# Quem instala é o mk-update, junto com a atualização do sistema.
# pacotes: wtype
set -euo pipefail
pacman -Q wtype &>/dev/null || echo "aviso: o wtype não foi instalado; o painel de emoji só copia" >&2
