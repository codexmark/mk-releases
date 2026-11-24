#!/usr/bin/env bash
# bin/lib-packages.sh: a leitura das listas de pacotes (read_list) e das
# seções de hardware.packages (read_hardware), com listas falsas em $SBX.
. "$(dirname "$0")/../lib.sh"

export PKG_DIR=$SBX/pacotes
mkdir -p "$PKG_DIR"
# shellcheck source=../../bin/lib-packages.sh
source "$REPO/bin/lib-packages.sh"

# ---------------------------------------------------------------- read_list

t "read_list: um pacote por linha, na ordem do arquivo"
printf 'hyprland\nquickshell\nghostty\n' >"$PKG_DIR/a.packages"
eq "hyprland
quickshell
ghostty" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: comentários, linhas vazias e linhas só de espaços ficam de fora"
printf '# Sessão\nhyprland\n\n   \n\t\n# quickshell\nghostty\n' >"$PKG_DIR/a.packages"
eq "hyprland
ghostty" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: o comentário no fim da linha sai, com os espaços antes dele"
printf 'hyprland   # o compositor\nghostty\t# terminal\ncava#sem espaço\n' >"$PKG_DIR/a.packages"
eq "hyprland
ghostty
cava" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: espaços no começo e no fim da linha não vão no nome"
printf '  hyprland\n\tghostty  \n' >"$PKG_DIR/a.packages"
eq "hyprland
ghostty" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: nomes com +, ., @ e números passam inteiros"
printf 'gtk4\nlibc++\nqt6-5compat\nfoo.bar\nsystemd@x\n' >"$PKG_DIR/a.packages"
eq "gtk4
libc++
qt6-5compat
foo.bar
systemd@x" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: arquivo sem a quebra de linha no fim não perde o último pacote"
printf 'hyprland\nghostty' >"$PKG_DIR/a.packages"
eq "hyprland
ghostty" "$(read_list "$PKG_DIR/a.packages")"

t "read_list: lista só de comentários é uma lista vazia, não um erro"
printf '# nada por enquanto\n\n' >"$PKG_DIR/a.packages"
out=$(read_list "$PKG_DIR/a.packages")
eq 0 $?
eq "" "$out"

t "read_list: arquivo vazio também"
: >"$PKG_DIR/a.packages"
out=$(read_list "$PKG_DIR/a.packages")
eq 0 $?
eq "" "$out"

t "read_list: arquivo que não existe é erro"
fails read_list "$PKG_DIR/nao-existe.packages" 2>/dev/null

# ------------------------------------------------------------ read_hardware

cat >"$PKG_DIR/hardware.packages" <<'LISTA'
# Drivers por perfil.
solto-antes-de-secao

[amd]
mesa            # OpenGL
vulkan-radeon

[intel]
mesa
  vulkan-intel

# a NVIDIA
[nvidia]
nvidia-open-dkms
# nvidia-settings

[vm]
LISTA

t "read_hardware: só os pacotes da seção pedida"
eq "mesa
vulkan-radeon" "$(read_hardware amd)"

t "read_hardware: várias seções, na ordem do arquivo"
eq "mesa
vulkan-radeon
nvidia-open-dkms" "$(read_hardware nvidia amd)"

t "read_hardware: comentários, linhas vazias e espaços em volta não entram"
eq "mesa
vulkan-intel" "$(read_hardware intel)"
hasnt "$(read_hardware amd intel nvidia vm)" "#"
hasnt "$(read_hardware amd intel nvidia vm)" "nvidia-settings"

t "read_hardware: seção vazia e perfil que não existe não listam nada"
eq "" "$(read_hardware vm)"
eq "" "$(read_hardware arm)"

t "read_hardware: o que vem antes da primeira seção não é de perfil nenhum"
hasnt "$(read_hardware amd intel nvidia vm)" "solto-antes-de-secao"

t "read_hardware: 'amd' não casa com um pedaço de outro nome de seção"
printf '[amd64]\nerrado\n[amd]\ncerto\n' >"$PKG_DIR/hardware.packages"
eq "certo" "$(read_hardware amd)"

# ------------------------------------------------------------------ PKG_DIR

t "PKG_DIR: sem a variável, as listas são as de system/packages do repositório"
eq "$REPO/system/packages" "$(unset PKG_DIR && source "$REPO/bin/lib-packages.sh" && echo "$PKG_DIR")"

t "PKG_DIR: quem já definiu a variável fica com a sua"
eq "$SBX/outra" "$(PKG_DIR=$SBX/outra && source "$REPO/bin/lib-packages.sh" && echo "$PKG_DIR")"

t "as listas de verdade: cada linha lida é um nome de pacote, sem espaço nem comentário"
for lista in base desktop dev apps aur; do
  lidos=$(read_list "$REPO/system/packages/$lista.packages")
  [[ -n $lidos ]] || _fail "$lista.packages saiu vazia"
  ruins=$(grep -vE '^[a-z0-9@._+-]+$' <<<"$lidos" || true)
  eq "" "$ruins" "$lista.packages"
done
ruins=$(PKG_DIR=$REPO/system/packages read_hardware amd intel nvidia vm | grep -vE '^[a-z0-9@._+-]+$' || true)
eq "" "$ruins" "hardware.packages"
