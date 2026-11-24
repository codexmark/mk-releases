#!/usr/bin/env bash
# shell/mare/scripts/mare-project: o estado de um projeto em JSON (git, mise,
# containers do compose, portas escutando), com git, mise, docker e ss por
# stub e um /proc falso.
. "$(dirname "$0")/../lib.sh"

proj_sh=$REPO/shell/mare/scripts/mare-project
proc=$SBX/proc
dir=$SBX/projetos/loja
mkdir -p "$dir/api" "$SBX/projetos/loja-antiga" "$SBX/projetos/outro" "$proc"

# git: git.branch e git.status dizem o que responder; sem git.branch, não é
# um repositório.
stub git '
[[ -e $SBX/git.branch ]] || { echo "fatal: not a git repository" >&2; exit 128; }
case "$*" in
  "branch --show-current") cat "$SBX/git.branch" ;;
  "status --porcelain") cat "$SBX/git.status" 2>/dev/null ;;
esac'
stub mise '[[ $1 == current ]] && cat "$SBX/mise.out" 2>/dev/null'
# docker: docker-ok diz que o serviço está no ar; docker.rows são as linhas
# do `docker ps` no formato pedido (serviço, estado, status, portas).
stub docker '
case $1 in
  info) [[ -e $SBX/docker-ok ]] ;;
  ps) cat "$SBX/docker.rows" 2>/dev/null ;;
esac'
stub ss 'cat "$SBX/ss.out" 2>/dev/null'

run() { PROC=$proc bash "$proj_sh" "$@"; }
# escuta <porta> <pid> <pasta do processo> [endereço]: uma linha do `ss -ltnpH`
# e a pasta de trabalho do processo em /proc.
escuta() {
  printf 'LISTEN 0      511    %21s      0.0.0.0:*    users:(("node",pid=%s,fd=23))\n' "${4:-127.0.0.1}:$1" "$2" >>"$SBX/ss.out"
  mkdir -p "$proc/$2"
  ln -sfn "$3" "$proc/$2/cwd"
}
zera() {
  local f
  for f in git.branch git.status mise.out docker-ok docker.rows ss.out; do rm -f "${SBX:?}/$f"; done
  rm -rf "${proc:?}" "$dir/mise.toml" "$dir/.mise.toml" "$dir/.tool-versions"
  mkdir -p "$proc"
  calls_reset
}
vazio='{"branch":"","dirty":0,"runtime":"","containers":[],"ports":[]}'

t "sem argumento, ou com uma pasta que não existe: o JSON vazio, sem chamar nada"
zera
eq "$vazio" "$(run)"
eq "$vazio" "$(run "$SBX/nao-existe")"
eq "$vazio" "$(run "$proj_sh")" "um arquivo não é pasta"
eq "" "$(<"$SBX/calls")"

t "pasta que não é repositório, sem docker e sem nada escutando: tudo vazio e JSON válido"
zera
out=$(run "$dir")
eq "$vazio" "$(jq -c . <<<"$out")"

t "git: o ramo atual e quantos arquivos têm mudança"
zera
echo "feat/checkout" >"$SBX/git.branch"
printf '%s\n' " M api/main.go" "?? notas.txt" "A  novo.go" >"$SBX/git.status"
out=$(run "$dir")
eq '["feat/checkout",3]' "$(jq -c '[.branch, .dirty]' <<<"$out")"

t "git: repositório limpo, dirty 0 (número, não texto)"
: >"$SBX/git.status"
eq '["feat/checkout",0,"number"]' "$(run "$dir" | jq -c '[.branch, .dirty, (.dirty | type)]')"

t "git: HEAD solto (sem ramo) vem com branch vazio"
: >"$SBX/git.branch"
eq '""' "$(run "$dir" | jq -c .branch)"

t "runtime: as versões do mise, 'ferramenta versão' separadas por ' · '"
zera
printf '%s\n' "go 1.25.0" "node 22.11.0" >"$SBX/mise.out"
: >"$dir/mise.toml"
eq "go 1.25.0 · node 22.11.0" "$(run "$dir" | jq -r .runtime)"

t "runtime: vale com mise.toml, .mise.toml ou .tool-versions"
for f in mise.toml .mise.toml .tool-versions; do
  rm -f "$dir/mise.toml" "$dir/.mise.toml" "$dir/.tool-versions"
  : >"$dir/$f"
  eq "go 1.25.0 · node 22.11.0" "$(run "$dir" | jq -r .runtime)" "$f"
done

t "runtime: projeto sem arquivo do mise não consulta o mise"
zera
echo "go 1.25.0" >"$SBX/mise.out"
eq "" "$(run "$dir" | jq -r .runtime)"
not_called "mise"

t "containers: os serviços do compose desta pasta, com nome, estado e se está rodando"
zera
: >"$SBX/docker-ok"
printf '%s\t%s\t%s\t%s\n' \
  web running "Up 2 hours" "0.0.0.0:8080->80/tcp, [::]:8080->80/tcp" \
  db exited "Exited (0) 3 days ago" "" \
  fila restarting "Restarting (1) 5 seconds ago" "" >"$SBX/docker.rows"
out=$(run "$dir")
eq '[{"name":"web","running":true,"status":"Up 2 hours"},{"name":"db","running":false,"status":"Exited (0) 3 days ago"},{"name":"fila","running":false,"status":"Restarting (1) 5 seconds ago"}]' \
  "$(jq -c .containers <<<"$out")"

t "containers: o filtro do docker é a pasta de trabalho do compose igual a este projeto"
called "docker ps -a --filter label=com.docker.compose.project.working_dir=$dir --format"

t "containers: as portas publicadas entram em ports, sem repetir IPv4 e IPv6"
eq '[":8080"]' "$(jq -c .ports <<<"$out")"

t "containers: várias portas publicadas pelo mesmo serviço"
printf '%s\t%s\t%s\t%s\n' web running "Up 1 minute" "0.0.0.0:8443->443/tcp, 0.0.0.0:8080->80/tcp, 127.0.0.1:5432->5432/tcp, 9000/tcp" >"$SBX/docker.rows"
eq '[":5432",":8080",":8443"]' "$(run "$dir" | jq -c .ports)"

t "containers: com o serviço do docker parado, lista vazia e nenhum docker ps"
zera
printf '%s\t%s\t%s\t%s\n' web running "Up" "0.0.0.0:8080->80/tcp" >"$SBX/docker.rows"
out=$(run "$dir")
eq '[[],[]]' "$(jq -c '[.containers, .ports]' <<<"$out")"
not_called "docker ps"

t "ports: quem escuta de dentro da pasta do projeto, ou de uma subpasta"
zera
escuta 3000 4001 "$dir"
escuta 5173 4002 "$dir/api"
eq '[":3000",":5173"]' "$(run "$dir" | jq -c .ports)"
called "ss -ltnpH"

t "ports: processo de outra pasta não conta, nem de uma vizinha com o mesmo começo de nome"
zera
escuta 3000 4001 "$dir"
escuta 4000 4003 "$SBX/projetos/outro"
escuta 4100 4004 "$SBX/projetos/loja-antiga"
escuta 4200 4005 "$SBX/projetos"
eq '[":3000"]' "$(run "$dir" | jq -c .ports)"

t "ports: processo sem pasta legível (de outro usuário) ou linha sem pid ficam de fora"
zera
escuta 3000 4001 "$dir"
printf 'LISTEN 0      128          0.0.0.0:22         0.0.0.0:*    users:(("sshd",pid=999,fd=3))\n' >>"$SBX/ss.out"
printf 'LISTEN 0      128          0.0.0.0:631        0.0.0.0:*\n' >>"$SBX/ss.out"
eq '[":3000"]' "$(run "$dir" | jq -c .ports)"

t "ports: escutando em IPv6 ([::]:porta) ou em todos os endereços (*:porta)"
zera
escuta 8000 4001 "$dir" "[::]"
escuta 9000 4002 "$dir" "*"
eq '[":8000",":9000"]' "$(run "$dir" | jq -c .ports)"

t "ports: em ordem numérica e sem repetição, juntando docker e processos"
zera
: >"$SBX/docker-ok"
printf '%s\t%s\t%s\t%s\n' web running "Up" "0.0.0.0:8080->80/tcp, 0.0.0.0:3000->3000/tcp" >"$SBX/docker.rows"
escuta 10000 4001 "$dir"
escuta 3000 4002 "$dir"
escuta 80 4003 "$dir"
escuta 80 4003 "$dir" "[::]"
eq '[":80",":3000",":8080",":10000"]' "$(run "$dir" | jq -c .ports)"

t "projeto completo: as cinco chaves, uma vez cada"
echo "main" >"$SBX/git.branch"
echo " M x" >"$SBX/git.status"
echo "node 22.11.0" >"$SBX/mise.out"
: >"$dir/mise.toml"
out=$(run "$dir")
eq '["branch","containers","dirty","ports","runtime"]' "$(jq -cS keys <<<"$out")"
eq '["main",1,"node 22.11.0",1,4]' "$(jq -c '[.branch, .dirty, .runtime, (.containers | length), (.ports | length)]' <<<"$out")"

t "pasta e ramo com espaço, aspas e cifrão não quebram o JSON nem o filtro"
zera
hostil="$SBX/projetos/meu \"projeto\" \$(id) 'novo'"
mkdir -p "$hostil"
echo 'ramo "com" \barra' >"$SBX/git.branch"
: >"$SBX/docker-ok"
escuta 3000 4001 "$hostil"
out=$(run "$hostil")
ok jq -e . <<<"$out" >/dev/null
eq 'ramo "com" \barra' "$(jq -r .branch <<<"$out")"
eq '[":3000"]' "$(jq -c .ports <<<"$out")"
called "label=com.docker.compose.project.working_dir=$hostil --format"

t "serviço com aspas no nome ou no estado não quebra o JSON"
zera
: >"$SBX/docker-ok"
printf '%s\t%s\t%s\t%s\n' 'we"b' running 'Up "2" \hours' "" >"$SBX/docker.rows"
out=$(run "$dir")
eq '[{"name":"we\"b","running":true,"status":"Up \"2\" \\hours"}]' "$(jq -c .containers <<<"$out")"
