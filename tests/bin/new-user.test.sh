#!/usr/bin/env bash
# bin/new-user: cria um usuário já com o Mk, numa raiz falsa
# (MK_NEW_USER_ROOT) e a partir de uma cópia mínima do repositório em $SBX.
# pacman, id, getent, useradd, passwd, chown e install são stubs.
. "$(dirname "$0")/../lib.sh"

R=$SBX/raiz
falso=$SBX/repo
casa=$R/home/mare
marca=.instalado-pelo-mk

# Uma cópia mínima do repositório: o script e o que ele copia.
mkdir -p "$falso"/{bin,config/hypr,config/ghostty,config/fastfetch,config/bash,shell/mare,migrations/user}
cp "$REPO/bin/new-user" "$falso/bin/new-user"
echo "#!/bin/bash" >"$falso/bin/mk-session"
for d in config/hypr config/ghostty config/fastfetch config/bash shell/mare; do echo "v1" >"$falso/$d/arquivo"; done
echo "# migração" >"$falso/migrations/user/20260101-a.sh"
echo "# migração" >"$falso/migrations/user/20260202-b.sh"
echo "não é migração" >"$falso/migrations/user/LEIAME.md"

# Os usuários da máquina falsa ficam em $SBX/passwd; os grupos em $SBX/grupos.
stub getent '
case $1 in
  passwd) grep "^$2:" "$SBX/passwd" ;;
  group) grep -qx -- "$2" "$SBX/grupos" && echo "$2:x:1:" ;;
esac'
stub id '
u=${@: -1}
grep -q "^$u:" "$SBX/passwd" || exit 1
[[ $1 == -gn ]] && echo "grupo-$u"
exit 0'
stub useradd '
u=${@: -1}
echo "$u:x:1001:1001::$SBX/raiz/home/$u:/bin/bash" >>"$SBX/passwd"
mkdir -p "$SBX/raiz/home/$u"'
stub passwd
stub chown
# pacman -Qq falha para os pacotes de $SBX/faltam; o resto sai com $SBX/pacman-rc.
stub pacman '
if [[ $1 == -Qq ]]; then
  ! grep -qx -- "$2" "$SBX/faltam" 2>/dev/null
else
  exit "$(cat "$SBX/pacman-rc" 2>/dev/null || echo 0)"
fi'
# install: o de verdade, sem o dono e o grupo (que só o root pode dar).
stub install '
args=()
while (($#)); do
  case $1 in -o | -g) shift ;; *) args+=("$1") ;; esac
  shift
done
exec /usr/bin/install "${args[@]}"'

zera() {
  rm -rf "${R:?}"
  mkdir -p "$R/usr/share/wayland-sessions" "$R/home"
  : >"$SBX/passwd"
  printf 'video\naudio\ninput\nwheel\n' >"$SBX/grupos"
  rm -f "$SBX/faltam" "$SBX/pacman-rc"
  calls_reset
}
cria() { MK_NEW_USER_ROOT=$R bash "$falso/bin/new-user" "$@" 2>&1; }
arvore() { (cd "$R" && find . \( -type f -o -type d \) | sort && find . -type f -exec md5sum {} + | sort); }

# ----------------------------------------------------------------- recusas

if ((EUID != 0)); then
  t "sem root (e fora da raiz de teste): pede o sudo e sai com 1, sem fazer nada"
  zera
  out=$(bash "$falso/bin/new-user" mare 2>&1)
  eq 1 $?
  has "$out" "rode com sudo"
  eq "" "$(cat "$SBX/calls")"
fi

t "nome inválido: sai com 2 antes de tocar em qualquer coisa"
zera
for nome in "" Mare 1mare "ma re" "mare;id" "-mare" abcdefghijklmnopqrstuvwxyz0123456; do
  out=$(cria "$nome")
  eq 2 $? "nome '$nome'"
  has "$out" "nome inválido"
done
eq "" "$(cat "$SBX/calls")"
absent "$R/usr/local/bin/mk-session"

t "nome com 32 letras, com - e _: vale"
zera
cria a_b-cdefghijklmnopqrstuvwxyz0123 >/dev/null
eq 0 $?
called "useradd -m -s /bin/bash -G video,audio,input a_b-cdefghijklmnopqrstuvwxyz0123"

t "opção desconhecida: diz qual e sai com 2, sem criar nem instalar nada"
zera
out=$(cria mare --root)
eq 2 $?
has "$out" "opção desconhecida: --root"
eq "" "$(cat "$SBX/calls")"

# -------------------------------------------------------------- usuário novo

t "usuário novo: cria com home, bash e os grupos video, audio e input"
zera
out=$(cria mare)
eq 0 $?
called "useradd -m -s /bin/bash -G video,audio,input mare"
has "$out" "criando mare (grupos: video audio input)"
has "$out" "Pronto."

t "usuário novo: com todos os pacotes instalados, não chama o pacman -S"
called "pacman -Qq quickshell"
called "pacman -Qq uwsm"
not_called "pacman -S"

t "usuário novo: instala a sessão Mk na tela de login"
ok cmp -s "$falso/bin/mk-session" "$R/usr/local/bin/mk-session"
ok test -x "$R/usr/local/bin/mk-session"
eq "[Desktop Entry]
Name=Mk
Comment=Hyprland com o Maré
Exec=/usr/local/bin/mk-session
Type=Application" "$(cat "$R/usr/share/wayland-sessions/mk.desktop")"

t "usuário novo: a config do Mk vai para a home dele, com a marca"
for d in hypr ghostty shell fastfetch bash; do
  eq v1 "$(cat "$casa/.config/mk/$d/arquivo" 2>/dev/null)" "$d"
  exists "$casa/.config/mk/$d/$marca"
done
exists "$casa/Work"
exists "$casa/Pictures/Wallpapers"

t "usuário novo: o .bashrc passa a carregar o mk.bash"
eq 1 "$(grep -c 'source ~/.config/mk/bash/mk.bash' "$casa/.bashrc")"

t "usuário novo: as migrações de usuário nascem marcadas como feitas, sem rodar, e são dele"
exists "$casa/.local/state/mk/migrations/20260101-a.sh"
exists "$casa/.local/state/mk/migrations/20260202-b.sh"
absent "$casa/.local/state/mk/migrations/LEIAME.md"
eq 0 "$(stat -c %s "$casa/.local/state/mk/migrations/20260101-a.sh")"
called "install -d -o mare -g grupo-mare $casa/.local $casa/.local/state $casa/.local/state/mk $casa/.local/state/mk/migrations"
called "install -m644 -o mare -g grupo-mare /dev/null $casa/.local/state/mk/migrations/20260101-a.sh"

t "usuário novo: tudo o que foi criado na home passa a ser dele, e a senha é pedida"
called "chown -R mare:grupo-mare $casa/.config/mk $casa/Work $casa/Pictures"
called "passwd mare"

t "usuário novo: sem --admin não entra no wheel"
not_called "wheel"

# ---------------------------------------- rodar de novo (usuário que já existe)

t "rodar de novo: não cria de novo, não pede senha e não marca migrações"
echo "# migração nova" >"$falso/migrations/user/20260303-c.sh"
antes=$(arvore)
calls_reset
out=$(cria mare)
eq 0 $?
has "$out" "mare já existe: só atualizo a config"
not_called "useradd"
eq 0 "$(grep -c '^passwd' "$SBX/calls")"
absent "$casa/.local/state/mk/migrations/20260303-c.sh"
rm "$falso/migrations/user/20260303-c.sh"

t "rodar de novo: a home fica igual (o .bashrc não ganha a linha duas vezes)"
eq "$antes" "$(arvore)"
eq 1 "$(grep -c 'mk/bash/mk.bash' "$casa/.bashrc")"

t "rodar de novo: a config que o Mk criou é atualizada, e o que saiu do Mk some"
echo "v2" >"$falso/config/hypr/arquivo"
echo "resto" >"$casa/.config/mk/hypr/arquivo-antigo"
cria mare >/dev/null
eq 0 $?
eq v2 "$(cat "$casa/.config/mk/hypr/arquivo")"
absent "$casa/.config/mk/hypr/arquivo-antigo"
exists "$casa/.config/mk/hypr/$marca"
echo "v1" >"$falso/config/hypr/arquivo"

t "pasta de config que não foi o Mk que criou: recusa, sai com 1 e não sobrescreve"
rm "$casa/.config/mk/ghostty/$marca"
echo "meu" >"$casa/.config/mk/ghostty/arquivo"
out=$(cria mare)
eq 1 $?
has "$out" "ERRO: $casa/.config/mk/ghostty já existe e não foi criado pelo Mk"
eq meu "$(cat "$casa/.config/mk/ghostty/arquivo")"

# ---------------------------------------------------------- grupos e pacotes

t "--admin: o usuário novo entra também no wheel"
zera
cria ana --admin >/dev/null
eq 0 $?
called "useradd -m -s /bin/bash -G video,audio,input,wheel ana"

t "grupo que não existe na máquina fica de fora"
zera
printf 'video\n' >"$SBX/grupos"
cria ana >/dev/null
called "useradd -m -s /bin/bash -G video ana"

t "sem nenhum dos grupos: cria sem -G"
zera
: >"$SBX/grupos"
out=$(cria ana)
eq 0 $?
called "useradd -m -s /bin/bash ana"
not_called " -G "
has "$out" "grupos: nenhum extra"

t "pacotes que faltam: instala só eles, com --needed, antes de criar o usuário"
zera
printf 'quickshell\nmatugen\n' >"$SBX/faltam"
out=$(cria ana)
eq 0 $?
called "pacman -S --needed quickshell matugen"
has "$out" "instalando: quickshell matugen"
ok test "$(grep -n '^pacman -S ' "$SBX/calls" | cut -d: -f1)" -lt "$(grep -n '^useradd' "$SBX/calls" | cut -d: -f1)"

t "o pacman falha: para ali, sem criar o usuário"
zera
echo quickshell >"$SBX/faltam"
echo 1 >"$SBX/pacman-rc"
cria ana >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "useradd"

# ------------------------------------------------------------- sobras antigas

t "o Maré antigo em ~/.config/quickshell/mare (com a marca) é apagado"
zera
cria mare >/dev/null
mkdir -p "$casa/.config/quickshell/mare"
touch "$casa/.config/quickshell/mare/$marca"
cria mare >/dev/null
absent "$casa/.config/quickshell/mare"

t "um ~/.config/quickshell/mare que não é do Mk fica onde está"
mkdir -p "$casa/.config/quickshell/mare"
echo "meu" >"$casa/.config/quickshell/mare/shell.qml"
cria mare >/dev/null
eq meu "$(cat "$casa/.config/quickshell/mare/shell.qml")"

# ------------------------------------------------------------ papéis de parede

t "--wallpapers: copia as imagens de quem chamou o sudo, sem sobrescrever as dele"
zera
echo "dona:x:1000:1000::$R/home/dona:/bin/bash" >>"$SBX/passwd"
mkdir -p "$R/home/dona/Pictures/Wallpapers/sub"
for f in a.jpg b.JPEG c.png d.webp notas.txt; do echo "de dona" >"$R/home/dona/Pictures/Wallpapers/$f"; done
echo "fundo" >"$R/home/dona/Pictures/Wallpapers/sub/e.png"
cria mare >/dev/null
echo "do mare" >"$casa/Pictures/Wallpapers/a.jpg"
out=$(SUDO_USER=dona cria mare --wallpapers)
eq 0 $?
has "$out" "copiando papéis de parede de dona"
eq "a.jpg
b.JPEG
c.png
d.webp" "$(ls "$casa/Pictures/Wallpapers")"
eq "do mare" "$(cat "$casa/Pictures/Wallpapers/a.jpg")"

t "--wallpapers sem SUDO_USER: não copia nada e termina bem"
zera
out=$(cria mare --wallpapers)
eq 0 $?
hasnt "$out" "copiando papéis de parede"
eq "" "$(ls "$casa/Pictures/Wallpapers")"

t "sem --wallpapers: os papéis de parede de quem chamou não são copiados"
zera
echo "dona:x:1000:1000::$R/home/dona:/bin/bash" >>"$SBX/passwd"
mkdir -p "$R/home/dona/Pictures/Wallpapers"
echo "de dona" >"$R/home/dona/Pictures/Wallpapers/a.jpg"
SUDO_USER=dona cria mare >/dev/null
eq "" "$(ls "$casa/Pictures/Wallpapers")"
