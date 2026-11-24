#!/usr/bin/env bash
# O MkPulso entra no Mk: o monitor de sistema (processos, desempenho, serviços,
# o que sobe no login) vem pronto em apps/mkpulso, com o coletor já compilado,
# e o mk-update o troca junto com o resto. Aqui ficam o comando mkpulso, o
# atalho no menu e o ícone (links para a pasta do Mk). Não pede pacote: usa o
# Quickshell e o jq, que o Mk já tem.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/mkpulso-links" "$R/"
[[ -x $share/apps/mkpulso/bin/mkpulso-coletor ]] ||
  echo "aviso: o coletor do MkPulso não veio neste lançamento; a janela não vai abrir" >&2
