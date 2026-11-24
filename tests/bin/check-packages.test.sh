#!/usr/bin/env bash
# bin/check-packages: cada pacote das listas está onde devia (oficiais ou AUR),
# com listas falsas (PKG_DIR), o pacman -Si e o AUR (curl) por stub.
. "$(dirname "$0")/../lib.sh"

export PKG_DIR=$SBX/pacotes
mkdir -p "$PKG_DIR"
listas() {
  printf '# base\nbase\nlinux\n' >"$PKG_DIR/base.packages"
  printf 'hyprland\n' >"$PKG_DIR/desktop.packages"
  printf 'neovim\n' >"$PKG_DIR/dev.packages"
  printf 'chromium\n' >"$PKG_DIR/apps.packages"
  printf 'yay-bin\nlimine-snapper-sync\n' >"$PKG_DIR/aur.packages"
  printf '[amd]\nvulkan-radeon\n[intel]\nvulkan-intel\n[nvidia]\nnvidia-open-dkms\n[vm]\nqemu-guest-agent\n' >"$PKG_DIR/hardware.packages"
  # Onde cada pacote mora ("nome repositório"), e o que o AUR conhece.
  printf '%s\n' "base core" "linux core" "hyprland extra" "neovim extra" "chromium extra" \
    "vulkan-radeon extra" "vulkan-intel extra" "nvidia-open-dkms extra" "qemu-guest-agent extra" >"$SBX/repos"
  echo '{"results":[{"Name":"yay-bin","PackageBase":"yay-bin"},{"Name":"limine-snapper-sync","PackageBase":"limine-snapper-sync"}]}' >"$SBX/aur.json"
  rm -f "$SBX/rotulo"
}
listas

# pacman -Si <pacote>: a ficha, com o repositório, ou erro se não existe.
stub pacman '
[[ $1 == -Si ]] || { echo "pacman $*" >>"$SBX/perigo"; exit 97; }
repo=$(sed -n "s/^$2 //p" "$SBX/repos")
[[ -n $repo ]] || exit 1
printf "%-16s: %s\nName            : %s\n" "$(cat "$SBX/rotulo" 2>/dev/null || echo Repository)" "$repo" "$2"'
stub curl 'cat "$SBX/aur.json"'
confere() { bash "$REPO/bin/check-packages" 2>&1; }

t "tudo no lugar: diz que está tudo certo e sai com 0"
out=$(confere)
eq 0 $?
has "$out" "Tudo certo: todas as listas batem com os repositórios."
hasnt "$out" "FALTA"
hasnt "$out" "AVISO"

t "confere as cinco listas e as quatro seções de hardware"
for p in base linux hyprland neovim chromium vulkan-radeon vulkan-intel nvidia-open-dkms qemu-guest-agent; do
  called "pacman -Si $p"
done
for s in base desktop dev apps hardware aur; do has "$out" "== $s.packages"; done

t "o AUR é consultado uma vez, com todos os pacotes de aur.packages"
eq 1 "$(grep -c '^curl' "$SBX/calls")"
called "https://aur.archlinux.org/rpc/v5/info?arg[]=yay-bin&arg[]=limine-snapper-sync"

t "não instala nada: nem sudo, nem outro pacman que não o -Si"
not_called "sudo"
eq 0 "$(grep '^pacman' "$SBX/calls" | grep -cv '^pacman -Si ')"

t "rodar de novo dá o mesmo resultado"
eq "$out" "$(confere)"

t "pacote oficial que não existe: FALTA, com a lista dele, e sai com 1"
printf 'hyprland\nnao-existe-mais\n' >"$PKG_DIR/desktop.packages"
out=$(confere)
eq 1 $?
has "$out" "FALTA   nao-existe-mais (desktop): não existe nos repositórios configurados"
has "$out" "Há problemas nas listas."
hasnt "$out" "Tudo certo"
listas

t "pacote que vem de um repositório não oficial: AVISO e sai com 1"
echo "neovim chaotic-aur" >>"$SBX/repos"
sed -i '/^neovim extra$/d' "$SBX/repos"
out=$(confere)
eq 1 $?
has "$out" "AVISO   neovim (dev): vem de [chaotic-aur], não de um repositório oficial"
listas

t "multilib conta como oficial"
sed -i 's/^vulkan-radeon extra$/vulkan-radeon multilib/' "$SBX/repos"
out=$(confere)
eq 0 $?
listas

t "pacman em português ('Repositório'): lê o repositório do mesmo jeito"
echo "Repositório" >"$SBX/rotulo"
out=$(confere)
eq 0 $?
has "$out" "Tudo certo"
listas

t "pacote de hardware que não existe: FALTA (hardware)"
sed -i '/^nvidia-open-dkms /d' "$SBX/repos"
out=$(confere)
eq 1 $?
has "$out" "FALTA   nvidia-open-dkms (hardware)"
listas

t "pacote de aur.packages que o AUR não conhece: FALTA (aur) e sai com 1"
echo '{"results":[{"Name":"yay-bin","PackageBase":"yay-bin"}]}' >"$SBX/aur.json"
out=$(confere)
eq 1 $?
has "$out" "FALTA   limine-snapper-sync (aur): não encontrado no AUR"
hasnt "$out" "yay-bin (aur)"
listas

t "pacote do AUR achado pelo PackageBase (o nome do pacote é outro) vale"
echo '{"results":[{"Name":"yay-bin","PackageBase":"yay-bin"},{"Name":"limine-snapper-sync-git","PackageBase":"limine-snapper-sync"}]}' >"$SBX/aur.json"
out=$(confere)
eq 0 $?
listas

t "pacote de aur.packages que também existe nos oficiais: AVISO para mover de lista"
echo "yay-bin extra" >>"$SBX/repos"
out=$(confere)
eq 1 $?
has "$out" "AVISO   yay-bin (aur): também existe nos oficiais, mova para outra lista"
listas

t "sem rede (o curl falha): diz que não alcançou o AUR e sai com 1, sem acusar pacote nenhum"
# Antes, sem rede, todo pacote de aur.packages saía como "não encontrado no AUR".
stub curl 'echo "curl: (6) Could not resolve host: aur.archlinux.org" >&2; exit 6'
out=$(confere)
eq 1 $?
has "$out" "não alcancei o AUR"
has "$out" "Não deu para conferir o AUR"
hasnt "$out" "não encontrado no AUR"
hasnt "$out" "FALTA"
hasnt "$out" "Tudo certo"
hasnt "$out" "Há problemas nas listas."

t "sem rede: as outras listas são conferidas do mesmo jeito, e o problema delas aparece"
printf 'hyprland\nnao-existe-mais\n' >"$PKG_DIR/desktop.packages"
out=$(confere)
eq 1 $?
has "$out" "FALTA   nao-existe-mais (desktop)"
has "$out" "Há problemas nas listas."
has "$out" "não alcancei o AUR"
hasnt "$out" "(aur): não encontrado"
listas

t "o AUR responde com erro (limite de consultas): diz o erro, não que os pacotes faltam"
stub curl 'echo "{\"type\":\"error\",\"error\":\"Rate limit reached\",\"results\":[]}"'
out=$(confere)
eq 1 $?
has "$out" "o AUR não devolveu a lista de pacotes (Rate limit reached)"
hasnt "$out" "não encontrado no AUR"
hasnt "$out" "Tudo certo"

t "a resposta não é do AUR (página de portal cativo): também não acusa os pacotes"
stub curl 'echo "<html>entre com a sua senha</html>"'
out=$(confere)
eq 1 $?
has "$out" "o AUR não devolveu a lista de pacotes"
hasnt "$out" "não encontrado no AUR"
stub curl 'cat "$SBX/aur.json"'

t "o AUR alcançado e sem nenhum dos pacotes: aí sim é FALTA, um por um"
echo '{"type":"multiinfo","resultcount":0,"results":[]}' >"$SBX/aur.json"
out=$(confere)
eq 1 $?
has "$out" "FALTA   yay-bin (aur): não encontrado no AUR"
has "$out" "FALTA   limine-snapper-sync (aur): não encontrado no AUR"
hasnt "$out" "não alcancei"
listas

t "vários problemas: lista todos antes de sair"
printf 'neovim\nsumiu-a\nsumiu-b\n' >"$PKG_DIR/dev.packages"
out=$(confere)
eq 1 $?
has "$out" "FALTA   sumiu-a (dev)"
has "$out" "FALTA   sumiu-b (dev)"
has "$out" "== aur.packages"
listas
