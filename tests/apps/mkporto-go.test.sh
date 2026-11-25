#!/usr/bin/env bash
# apps/mkporto/src: o motor do MkPorto (o gerente de conexões SSH), em Go. Os
# testes dele são `go test` (com pastas ~/.ssh de mentira, um ssh-keygen de
# mentira e um listener TCP em 127.0.0.1); aqui eles entram na suíte do Mk,
# junto com o formato (gofmt), o go vet, a conferência de que o binário que
# vai no pacote é o do código que está no repositório e as regras que o código
# não pode quebrar (só roda o ssh sem conversa, nunca lê chave privada, nunca
# pede senha, só escreve no arquivo do app e na linha do Include).
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

app=$REPO/apps/mkporto
src=$app/src
bin=$app/bin/mkporto-motor
# O cache de compilação fica fora da caixa de areia, para não recompilar tudo
# a cada rodada; o resto (HOME, módulos) continua dentro dela.
export GOCACHE=${MK_GOCACHE:-/var/tmp/mk-gocache-$(id -u)}
# -buildvcs=false: sem isso o go chama o git (que aqui é um guarda) para carimbar o binário.
export GOFLAGS="-mod=mod -buildvcs=false" GOPROXY=off GOTOOLCHAIN=local
cd "$src" || exit 1

t "o módulo não tem dependência de fora: só a biblioteca padrão"
eq "" "$(grep -E '^\s*require|^\s+[a-z0-9.-]+\.[a-z]+/' go.mod || true)"
absent "$src/go.sum"

t "gofmt: todo arquivo está formatado"
eq "" "$(gofmt -l . 2>&1)"

t "go vet: nenhum aviso"
out=$(go vet ./... 2>&1) || _fail "$out"

t "go test: a suíte do motor passa"
out=$(go test -count=1 ./... 2>&1) || _fail "$(tail -40 <<<"$out")"
hasnt "$out" "FAIL"

t "todo pacote do motor tem teste"
while read -r pacote; do
  compgen -G "$src/$pacote"'/*_test.go' >/dev/null || _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mkporto$' | sed 's|^mkporto/||')
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

t "todo subcomando do contrato tem o seu teste de contrato em Go"
for c in Listar Sondar Entrar Guardar Remover Chaves Comando; do
  grep -qE "^func Test${c}_Contrato" "$src"/*_test.go || _fail "sem teste de contrato: $c"
done

t "os casos que não podem faltar estão na suíte"
for caso in TestHosts_IncludeComGlobRelativoAPastaDoSsh TestHosts_VariosApelidosNumaLinha TestHosts_PadroesComCuringaNaoSaoMaquinas \
  TestHosts_MatchEncerraOBloco TestGuardar_PreservaOConfigDaPessoaByteAByteENaoDuplicaOInclude TestGaranteInclude_NaoDuplica \
  TestGuardar_RecusaInjecaoDeOpcaoEDeLinhaSemEscreverNada TestValida_RecusaInjecaoDeOpcaoEDeLinha TestGrava_Permissoes600E700 \
  TestRemover_RecusaOQueNaoEDoApp TestSondar_Contrato TestComando_RecusaOQueNaoPodeIrParaUmaLinhaDeComando \
  TestListar_OsValoresVemDoSshG TestListar_SshGQueFalhaDemoraOuDizBobagemValeOLeitorProprio TestSondar_PortaQueAbreMasNaoESsh \
  TestSondar_PorSalto TestEntrar_Contrato TestComando_Gerar TestComando_CaminhoComEspacoAspasECifraoChegaInteiro; do
  grep -qhE "^func $caso\(" "$src"/*_test.go "$src"/internal/*/*_test.go || _fail "falta o caso: $caso"
done
# O sondar é testado contra um listener local e contra uma porta fechada.
ok grep -qF 'net.Listen("tcp", "127.0.0.1:0")' "$src/main_test.go"
ok grep -qF 'recusou a conexão' "$src/main_test.go"

t "a suíte Go nunca fala com a rede de fora nem roda comando de verdade"
# Só 127.0.0.1 é discado; os outros endereços dos testes são de documentação
# (192.0.2.x, .example, .test) e nunca chegam a uma sonda sem salto.
eq "" "$(grep -ohE 'HostName [^ \\"]+' "$src/main_test.go" "$src/ssh_test.go" | cut -d' ' -f2 |
  grep -vE '^(127\.0\.0\.1|192\.0\.2\.[0-9]+|[a-z0-9.%-]+\.(example|test))$' || true)"
# Todo teste que roda um subcomando monta o mundo de mentira (HOME, ~/.ssh e PATH vazio).
awk '/^func Test/ { nome = $2; mundo = 0 }
     /mundo\(t, / { mundo = 1 }
     /(roda|quer|querErro)\(t, / { if (nome != "" && !mundo) print nome }' "$src/main_test.go" "$src/ssh_test.go" | sort -u >"$SBX/sem-mundo"
eq "TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON(t" "$(tr '\n' ' ' <"$SBX/sem-mundo" | sed 's/ $//')"
ok grep -qF 't.Setenv("PATH", bin)' "$src/main_test.go"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "o motor só roda o ssh sem conversa (-G, -W e true, com BatchMode) e o ssh-keygen -l -f de uma chave pública"
eq "4" "$(grep -hoE 'exec\.Command(Context)?\(' "${fontes[@]}" | wc -l)"
ok grep -qF 'exec.CommandContext(ctx, keygen, "-l", "-f", publica)' "$src/chaves.go"
ok grep -qF 'exec.LookPath(base.Comando("MKPORTO_KEYGEN", "ssh-keygen"))' "$src/chaves.go"
# Perguntar a configuração: só lê, não conecta.
ok grep -qF 'exec.CommandContext(ctx, ssh, "-G", "--", apelido)' "$src/efetivo.go"
# A sonda por salto e o teste de entrada: BatchMode (nada é perguntado, nenhuma
# chave de host é aceita sozinha), tempo limite, e o destino depois de --.
ok grep -qF 'argv := []string{ssh, "-o", "BatchMode=yes", "-o", "ConnectTimeout=4"}' "$src/sondar.go"
ok grep -qF 'return append(argv, "-W", net.JoinHostPort(host, strconv.Itoa(porta)), "--", salto), true' "$src/sondar.go"
ok grep -qF 'exec.CommandContext(ctx, argv[0], argv[1:]...)' "$src/sondar.go"
ok grep -qF 'exec.CommandContext(ctx, ssh, "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "NumberOfPasswordPrompts=0", "--", apelido, "true")' "$src/entrar.go"
# As duas linhas de comando que conectam (a sonda por salto e o entrar) levam BatchMode.
eq "2" "$(cat "${fontes[@]}" | grep -v '^\s*//' | grep -c '"BatchMode=yes"')"
# Todo comando externo tem prazo: nenhum roda sem contexto.
eq "" "$(grep -nE 'exec\.Command\(' "${fontes[@]}" || true)"
eq "4" "$(grep -hoE 'context\.WithTimeout\(' "${fontes[@]}" | wc -l)"
# Só um apelido seguro chega à linha de comando do ssh.
ok grep -qF 'if !sshcfg.ApelidoSeguro(hosts[i].Apelido) {' "$src/efetivo.go"
ok grep -qF 'if !sshcfg.ApelidoSeguro(apelido) {' "$src/entrar.go"
# Os LookPath: o ssh, o ssh-keygen e se um terminal conhecido existe.
eq "3" "$(grep -hoE 'exec\.LookPath\(' "${fontes[@]}" | wc -l)"
ok grep -qF 'exec.LookPath(base.Comando("MKPORTO_SSH", "ssh"))' "$src/efetivo.go"
ok grep -qF 'exec.LookPath(t); err == nil' "$src/comando.go"

t "nada passa por shell nem pede senha: não há sh -c, sudo, pkexec nem sshpass no motor"
eq "" "$(grep -nE '"(sh|bash|/bin/sh|/bin/bash|zsh|env|sudo|su|doas|run0|pkexec|sshpass|ssh-askpass|ssh-add)"|"-c"' "${fontes[@]}" || true)"
# (a única "password" do motor é a opção que proíbe a pergunta)
eq "" "$(grep -niE 'senha|password|passphrase|SSH_ASKPASS' "${fontes[@]}" | grep -v 'NumberOfPasswordPrompts=0' | grep -v '^[^:]*:[0-9]*:\s*//' || true)"

t "o motor só abre o config, os incluídos, os *.pub e os Ajustes do Mk: nunca uma chave privada"
# Toda leitura de arquivo do motor é uma destas quatro.
eq "$(printf '%s\n' \
  'chaves.go:	f, err := os.Open(caminho)' \
  'comando.go:	if b, err := os.ReadFile(base.Config("mk", "ajustes.json")); err == nil {' \
  'internal/sshcfg/gravar.go:	antes, err := os.ReadFile(config)' \
  'internal/sshcfg/ler.go:	f, err := os.Open(caminho)')" \
  "$(grep -HE 'os\.(Open|ReadFile|OpenFile)\(' "${fontes[@]}" | sed "s|^$src/||; s|:|:\t|; s|:\t\s*|:\t|" | sort)"
# E a de chaves.go recusa o que não termina em .pub antes de abrir.
like "$(<"$src/chaves.go")" 'func lePublica\(caminho string\) \(string, bool\) \{[[:space:]]+if !strings\.HasSuffix\(caminho, "\.pub"\) \{[[:space:]]+return "", false'
# Os nomes das chaves padrão só aparecem numa lista, para reconhecer o que o
# ssh -G diz por conta própria: nenhum deles é aberto.
eq "" "$(grep -nE 'id_(rsa|ed25519|ecdsa|dsa)"' "${fontes[@]}" | grep -vF 'efetivo.go:23:var chavesPadrao = []string{' || true)"

t "o motor só escreve por um caminho: o arquivo do app e a linha do Include, de forma atômica"
eq "internal/sshcfg/gravar.go" "$(grep -lE 'os\.(WriteFile|Create|CreateTemp|Rename|OpenFile|Remove|Mkdir|MkdirAll|Chmod)\(' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"
eq "" "$(grep -nE 'os\.(WriteFile|Create|OpenFile|MkdirAll|RemoveAll)\(' "${fontes[@]}" || true)"
ok grep -qF 'os.Rename(nome, caminho)' "$src/internal/sshcfg/gravar.go"
ok grep -qF 'escreveAtomico(ArquivoDoApp(dir), Texto(entradas), 0o600)' "$src/internal/sshcfg/gravar.go"
ok grep -qF 'os.Mkdir(caminho, 0o700)' "$src/internal/sshcfg/gravar.go"

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mkporto/pack depois de mexer em src/)"
# O pack anota a assinatura do código (um resumo de todos os .go e do go.mod)
# ao lado do binário. Comparar os bytes do binário dependeria da versão do Go
# de cada máquina.
eq "$("$app/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
like "$(jq -r .erro <<<"$out")" '^uso: mkporto-motor <'
contrato=$(<"$app/CONTRATO.md")
for c in listar sondar entrar guardar remover chaves comando; do
  has "$out" "$c" "subcomando do contrato"
  has "$contrato" "## \`mkporto-motor $c" "o contrato não descreve: $c"
done
eq 7 "$(jq -r .erro <<<"$out" | tr '|' '\n' | wc -l)"

# --- o binário de verdade numa casa de mentira ----------------------------------
ssh=$SBX/home/.ssh
mkdir -p "$ssh"
motor() { MKPORTO_SSH_DIR=$ssh "$bin" "$@"; }
# O ssh de mentira: diz como vê o atlas, não conhece os outros, e não deixa entrar.
stub ssh 'case " $* " in
  *" -G -- atlas "*) printf "hostname visto.oficina.example\nuser marina\nport 2200\nidentityfile ~/.ssh/id_rsa\n" ;;
  *" -G "*) exit 1 ;;
  *" true "*) echo "marina@192.0.2.5: Permission denied (publickey)." >&2; exit 255 ;;
  *) exit 9 ;;
esac'

t "o binário guarda um host sem tocar no resto do config, com 600 e 700, e o lista de volta"
printf '# meu\nHost meu\n  HostName 192.0.2.5\n\tUser eu  \n' >"$ssh/config"
cp "$ssh/config" "$SBX/config.antes"
out=$(motor guardar <<<'{"apelido":"atlas","host":"atlas.oficina.example","usuario":"marina","porta":2200}')
eq 0 $?
jeq '{"ok":true,"apelido":"atlas","arquivo":"~/.ssh/config.d/mkporto.conf","include":"posto"}' "$out"
eq "Include config.d/mkporto.conf" "$(head -1 "$ssh/config")"
ok cmp -s <(tail -n +3 "$ssh/config") "$SBX/config.antes"
eq "600" "$(stat -c %a "$ssh/config.d/mkporto.conf")"
eq "700" "$(stat -c %a "$ssh/config.d")"
out=$(motor listar)
jeq '["atlas","meu"]' "$(jq -c '[.hosts[].apelido]' <<<"$out")"
jeq '["mkporto","config"]' "$(jq -c '[.hosts[].origem]' <<<"$out")"
# Os valores são os que o ssh diz (ssh -G); o que está escrito no arquivo do app vem à parte.
eq "marina@visto.oficina.example:2200" "$(jq -r '.hosts[0] | "\(.usuario)@\(.host):\(.porta)"' <<<"$out")"
jeq '[true,false]' "$(jq -c '[.hosts[].efetivo]' <<<"$out")"
jeq '{"host":"atlas.oficina.example","usuario":"marina","porta":2200,"chave":null,"salto":null}' "$(jq -c '.hosts[0].escrito' <<<"$out")"
eq "null" "$(jq -c '.hosts[1].escrito' <<<"$out")"
eq "eu@192.0.2.5" "$(jq -r '.hosts[1] | "\(.usuario)@\(.host)"' <<<"$out")"
called "ssh -G -- atlas"
called "ssh -G -- meu"

t "o binário testa a entrada sem conversa e traduz o que o ssh reclamou"
calls_reset
out=$(motor entrar -- atlas)
eq 0 $?
eq "false" "$(jq -r .entra <<<"$out")"
eq "a máquina não aceitou nenhuma das suas chaves (envie uma com enviar chave)" "$(jq -r .motivo <<<"$out")"
eq "ssh -o BatchMode=yes -o ConnectTimeout=5 -o NumberOfPasswordPrompts=0 -- atlas true" "$(cat "$SBX/calls")"
out=$(motor entrar fantasma)
eq 1 $?

t "o binário recusa a injeção de opção e de linha (código 1, com o campo) e não escreve nada"
cp "$ssh/config.d/mkporto.conf" "$SBX/app.antes"
cp "$ssh/config" "$SBX/config.depois"
while IFS= read -r pedido; do
  out=$(motor guardar <<<"$pedido")
  eq 1 $? "$pedido"
  ok jq -e 'has("erro") and has("campo")' <<<"$out" >/dev/null
done <<'PEDIDOS'
{"apelido":"-oProxyCommand=touch pwn","host":"x.test"}
{"apelido":"a\nHost *\n  ProxyCommand touch pwn","host":"x.test"}
{"apelido":"a","host":"-oProxyCommand=touch pwn"}
{"apelido":"a","host":"x.test\nProxyCommand touch pwn"}
{"apelido":"a","host":"x.test","usuario":"eu\nProxyCommand touch pwn"}
{"apelido":"a","host":"x.test","chave":"k\nProxyCommand touch pwn"}
{"apelido":"a","host":"x.test","salto":"-oProxyCommand=touch pwn"}
{"apelido":"a","host":"x.test","porta":0}
PEDIDOS
ok cmp -s "$ssh/config.d/mkporto.conf" "$SBX/app.antes"
ok cmp -s "$ssh/config" "$SBX/config.depois"
hasnt "$(cat "$ssh/config" "$ssh/config.d/mkporto.conf")" "ProxyCommand"

t "o binário recusa (código 2) mexer no que é do config da pessoa"
out=$(motor guardar <<<'{"apelido":"meu","host":"x.test"}')
eq 2 $?
# shellcheck disable=SC2088 # é o texto "~/.ssh/config" da frase, não um caminho
has "$(jq -r .erro <<<"$out")" "~/.ssh/config"
out=$(motor remover meu)
eq 2 $?
has "$(jq -r .erro <<<"$out")" "é do seu ~/.ssh/config"
ok cmp -s "$ssh/config" "$SBX/config.depois"

t "o binário diz o comando de conectar sem rodar nada, e recusa um apelido com cara de opção"
stub foot
out=$(MKPORTO_TERMINAL=foot motor comando conectar -- atlas)
eq 0 $?
jeq '["ssh","--","atlas"]' "$(jq -c .argv <<<"$out")"
jeq '["foot","--title=ssh · atlas","ssh","--","atlas"]' "$(jq -c .terminal <<<"$out")"
eq "ssh atlas" "$(jq -r .texto <<<"$out")"
out=$(motor comando conectar -oProxyCommand=x)
eq 1 $?
printf 'Host -oProxyCommand=x\n' >>"$ssh/config"
out=$(motor comando conectar -- -oProxyCommand=x)
eq 2 $?
not_called "foot"

t "o binário lista as chaves públicas e só mostra ao ssh-keygen os .pub"
printf 'ssh-ed25519 QUFBQWV4ZW1wbG8tZWQyNTUxOQ== marina@oficina.example\n' >"$ssh/id_ed25519.pub"
printf 'segredo\n' >"$ssh/id_ed25519"
stub ssh-keygen 'echo "256 SHA256:exemplo marina@oficina.example (ED25519)"'
out=$(motor chaves)
eq 0 $?
jeq '[{"nome":"id_ed25519.pub","arquivo":"~/.ssh/id_ed25519.pub","tipo":"ed25519","bits":256,"impressao":"SHA256:exemplo","comentario":"marina@oficina.example","publica":"ssh-ed25519 QUFBQWV4ZW1wbG8tZWQyNTUxOQ== marina@oficina.example"}]' "$(jq -c .chaves <<<"$out")"
called "ssh-keygen -l -f $ssh/id_ed25519.pub"
eq 1 "$(grep -c '^ssh-keygen ' "$SBX/calls")"
hasnt "$out" "segredo"

t "o binário remove o host do app e o config continua como estava"
cp "$ssh/config" "$SBX/config.final"
out=$(motor remover -- atlas)
eq 0 $?
jeq '{"ok":true,"apelido":"atlas"}' "$out"
hasnt "$(<"$ssh/config.d/mkporto.conf")" "atlas"
ok cmp -s "$ssh/config" "$SBX/config.final"
eq "600" "$(stat -c %a "$ssh/config.d/mkporto.conf")"

t "a suíte Go do motor é rápida (poucos segundos)"
inicio=$(date +%s)
out=$(go test -count=1 ./... 2>&1) || _fail "$(tail -20 <<<"$out")"
ok test $(($(date +%s) - inicio)) -le 30

t "o binário diz o comando da chave nova, com o caminho inteiro num argumento só, e não gera nada"
casa="$SBX/casa de 'ana' \$HOME/.ssh"
mkdir -p "$casa"
printf 'ssh-ed25519 QUFBQWV4ZW1wbG8tZWQyNTUxOQ== ana@oficina.example\n' >"$casa/velha.pub"
calls_reset
out=$(MKPORTO_SSH_DIR=$casa MKPORTO_TERMINAL=foot "$bin" comando gerar -- nova "ana d'água")
eq 0 $?
jeq "$(jq -cn --arg f "$casa/nova" '["ssh-keygen","-t","ed25519","-f",$f,"-C","ana d\u0027água"]')" "$(jq -c .argv <<<"$out")"
jeq "$(jq -cn --arg f "$casa/nova" '["foot","--title=ssh-keygen · nova","--hold","ssh-keygen","-t","ed25519","-f",$f,"-C","ana d\u0027água"]')" "$(jq -c .terminal <<<"$out")"
out=$(MKPORTO_SSH_DIR=$casa "$bin" comando gerar -- velha)
eq 1 $?
eq "nome" "$(jq -r .campo <<<"$out")"
out=$(MKPORTO_SSH_DIR=$casa "$bin" comando gerar -- -oProxyCommand=x)
eq 1 $?
eq "velha.pub" "$(ls -A "$casa")"
not_called "ssh-keygen"
not_called "foot"
