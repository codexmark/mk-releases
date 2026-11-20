# Funções compartilhadas para ler as listas de pacotes. Use com `source`.

PKG_DIR="${PKG_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../system/packages" && pwd)}"

# Lista os pacotes de um arquivo, sem comentários e linhas em branco.
read_list() {
  sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$1" | grep -v '^[[:space:]]*$'
}

# Lista os pacotes de uma ou mais seções de hardware.packages.
# Uso: read_hardware amd nvidia
read_hardware() {
  local wanted=" $* "
  local section=""
  while IFS= read -r line; do
    line="${line%%#*}"
    line="${line//[[:space:]]/}"
    [[ -z $line ]] && continue
    if [[ $line =~ ^\[(.+)\]$ ]]; then
      section="${BASH_REMATCH[1]}"
    elif [[ $wanted == *" $section "* ]]; then
      echo "$line"
    fi
  done <"$PKG_DIR/hardware.packages"
}
