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
for c in Listar Sondar Entrar Guardar Remover Chaves Comando VmRequisitos VmCriar VmLigar VmSnapshot VmHost; do
  grep -qE "^func Test${c}_Contrato" "$src"/*_test.go || _fail "sem teste de contrato: $c"
done

t "os casos que não podem faltar estão na suíte"
for caso in TestHosts_IncludeComGlobRelativoAPastaDoSsh TestHosts_VariosApelidosNumaLinha TestHosts_PadroesComCuringaNaoSaoMaquinas \
  TestHosts_MatchEncerraOBloco TestGuardar_PreservaOConfigDaPessoaByteAByteENaoDuplicaOInclude TestGaranteInclude_NaoDuplica \
  TestGuardar_RecusaInjecaoDeOpcaoEDeLinhaSemEscreverNada TestValida_RecusaInjecaoDeOpcaoEDeLinha TestGrava_Permissoes600E700 \
  TestRemover_RecusaOQueNaoEDoApp TestSondar_Contrato TestComando_RecusaOQueNaoPodeIrParaUmaLinhaDeComando \
  TestListar_OsValoresVemDoSshG TestListar_SshGQueFalhaDemoraOuDizBobagemValeOLeitorProprio TestSondar_PortaQueAbreMasNaoESsh \
  TestSondar_PorSalto TestEntrar_Contrato TestComando_Gerar TestComando_CaminhoComEspacoAspasECifraoChegaInteiro \
  TestArgv_AMaquinaInteira TestArgv_VirgulaNoCaminhoVaiDobrada TestQMP_EventosNoMeioSaoIgnorados TestQMP_MudoSemSaudacaoOuSemSocket \
  TestLeSnapshots_AsSaidasDoQemuImg TestApaga_SoAPastaDaMaquinaENadaFora TestVmCriar_SeAlgoFalhaNoMeioAPastaEApagada \
  TestVmLigar_QemuQueMorreDevolveOFimDoLog TestVmDesligar_ComOQmpMudoForcarMandaSigtermSoAoQemuDaMaquina \
  TestSituacao_PidQueSobrouOuQueEDeOutroProgramaELimpoSemSinalizarNinguem TestVmRemover_SoComONomeRepetidoESoAPastaDela; do
  grep -qhE "^func $caso\(" "$src"/*_test.go "$src"/internal/*/*_test.go || _fail "falta o caso: $caso"
done
# O sondar é testado contra um listener local e contra uma porta fechada.
ok grep -qF 'net.Listen("tcp", "127.0.0.1:0")' "$src/main_test.go"
ok grep -qF 'recusou a conexão' "$src/main_test.go"

t "a suíte Go nunca fala com a rede de fora nem roda comando de verdade"
# Só 127.0.0.1 é discado; os outros endereços dos testes são de documentação
# (192.0.2.x, .example, .test) e nunca chegam a uma sonda sem salto.
eq "" "$(grep -ohE 'HostName [^ \\"]+' "$src/main_test.go" "$src/ssh_test.go" "$src/vm_test.go" | cut -d' ' -f2 |
  grep -vE '^(127\.0\.0\.1|192\.0\.2\.[0-9]+|[a-z0-9.%-]+\.(example|test))$' || true)"
# Todo teste que roda um subcomando monta o mundo de mentira (HOME, ~/.ssh e PATH vazio).
awk '/^func Test/ { nome = $2; mundo = 0 }
     /mundo(DasMaquinas)?\(t[,)]/ { mundo = 1 }
     /(roda|quer|querErro)\(t, / { if (nome != "" && !mundo) print nome }' "$src/main_test.go" "$src/ssh_test.go" "$src/vm_test.go" | sort -u >"$SBX/sem-mundo"
eq "TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON(t" "$(tr '\n' ' ' <"$SBX/sem-mundo" | sed 's/ $//')"
ok grep -qF 't.Setenv("PATH", bin)' "$src/main_test.go"
# O QEMU dos testes é o próprio binário de teste (MKPORTO_QEMU_FALSO), o
# qemu-img é um script, e nenhum deles sobrevive ao teste.
ok grep -qF 't.Setenv("MKPORTO_QEMU", eu)' "$src/vm_test.go"
ok grep -qF 't.Setenv("MKPORTO_QEMU_IMG", falso(t, bin, "qemu-img", imgFalso))' "$src/vm_test.go"
ok grep -qF '_ = syscall.Kill(pid, syscall.SIGKILL)' "$src/vm_test.go"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "o motor só roda o ssh sem conversa (-G, -W e true, com BatchMode) e o ssh-keygen -l -f de uma chave pública"
# (mais o qemu-img, duas vezes, e o QEMU de uma máquina: veja as máquinas, abaixo)
eq "7" "$(grep -hoE 'exec\.Command(Context)?\(' "${fontes[@]}" | wc -l)"
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
# Todo comando externo tem prazo, menos um: o QEMU de uma máquina, que tem de
# continuar depois que o motor sai.
eq "vm_ligar.go:	cmd := exec.Command(argv[0], argv[1:]...)" "$(grep -HE 'exec\.Command\(' "${fontes[@]}" | sed "s|^$src/||; s|:\s*|:\t|")"
eq "6" "$(grep -hoE 'context\.WithTimeout\(' "${fontes[@]}" | wc -l)"
# Só um apelido seguro chega à linha de comando do ssh.
ok grep -qF 'if !sshcfg.ApelidoSeguro(hosts[i].Apelido) {' "$src/efetivo.go"
ok grep -qF 'if !sshcfg.ApelidoSeguro(apelido) {' "$src/entrar.go"
# Os LookPath: o ssh, o ssh-keygen, se um terminal conhecido existe e os
# comandos do QEMU.
eq "4" "$(grep -hoE 'exec\.LookPath\(' "${fontes[@]}" | wc -l)"
ok grep -qF 'if caminho, err := exec.LookPath(nome); err == nil {' "$src/vm.go"
ok grep -qF 'exec.LookPath(base.Comando("MKPORTO_SSH", "ssh"))' "$src/efetivo.go"
ok grep -qF 'exec.LookPath(t); err == nil' "$src/comando.go"

t "nada passa por shell nem pede senha: não há sh -c, pkexec nem sshpass no motor, e o sudo só aparece no argv que a janela abre num terminal"
# (o "-c" que há é a opção de criar snapshot do qemu-img, não um shell)
eq "$(printf '%s\n' \
  'vm.go:	argv := []string{"sudo", "pacman", "-S", "--needed", "qemu-desktop", "edk2-ovmf"}' \
  'vm.go:	opcao := map[string]string{"listar": "-l", "criar": "-c", "restaurar": "-a", "apagar": "-d"}')" \
  "$(grep -HE '"(sh|bash|/bin/sh|/bin/bash|zsh|env|sudo|su|doas|run0|pkexec|sshpass|ssh-askpass|ssh-add)"|"-c"' "${fontes[@]}" | sed "s|^$src/||; s|:\s*|:\t|")"
# O argv do sudo só vai para a resposta (instalar), nunca para um exec do
# motor: o único exec de vm.go é o do qemu-img.
ok grep -qF 'instalar := map[string]any{"argv": argv, "terminal": nil, "aviso": nil}' "$src/vm.go"
eq 2 "$(grep -c 'exec\.CommandContext(ctx, img\.Caminho, ' "$src/vm.go")"
eq 2 "$(grep -c 'exec\.Command' "$src/vm.go")"
# (a única "password" do motor é a opção que proíbe a pergunta)
eq "" "$(grep -niE 'senha|password|passphrase|SSH_ASKPASS' "${fontes[@]}" | grep -v 'NumberOfPasswordPrompts=0' | grep -v '^[^:]*:[0-9]*:\s*//' || true)"

t "o motor só abre o config, os incluídos, os *.pub, os Ajustes do Mk e o que é das máquinas: nunca uma chave privada"
# Toda leitura de arquivo do motor é uma destas: quatro do lado do ssh, e as
# das máquinas (o maquina.json, o OVMF_VARS, o registro, o /dev/kvm, o pid e o /proc).
eq "$(printf '%s\n' \
  'chaves.go:	f, err := os.Open(caminho)' \
  'comando.go:	if b, err := os.ReadFile(base.Config("mk", "ajustes.json")); err == nil {' \
  'internal/sshcfg/gravar.go:	antes, err := os.ReadFile(config)' \
  'internal/sshcfg/ler.go:	f, err := os.Open(caminho)' \
  'internal/vm/loja.go:	b, err := os.ReadFile(l.Config(nome))' \
  'internal/vm/loja.go:	b, err := os.ReadFile(l.Log(nome))' \
  'internal/vm/loja.go:	de, err := os.Open(modelo)' \
  'internal/vm/loja.go:	f, err := os.OpenFile(caminho, os.O_RDWR, 0)' \
  'internal/vm/loja.go:	para, err := os.OpenFile(l.Vars(nome), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)' \
  'internal/vm/loja.go:	return os.OpenFile(l.Log(nome), os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0o600)' \
  'internal/vm/proc.go:	b, err := os.ReadFile(arquivo)' \
  'internal/vm/proc.go:	b, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "cmdline"))' \
  'internal/vm/proc.go:	stat, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "stat"))' \
  'internal/vm/proc.go:	status, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "status"))')" \
  "$(grep -HE 'os\.(Open|ReadFile|OpenFile)\(' "${fontes[@]}" | sed "s|^$src/||; s|:|:\t|; s|:\t\s*|:\t|" | sort)"
# E a de chaves.go recusa o que não termina em .pub antes de abrir.
like "$(<"$src/chaves.go")" 'func lePublica\(caminho string\) \(string, bool\) \{[[:space:]]+if !strings\.HasSuffix\(caminho, "\.pub"\) \{[[:space:]]+return "", false'
# Os nomes das chaves padrão só aparecem numa lista, para reconhecer o que o
# ssh -G diz por conta própria: nenhum deles é aberto.
eq "" "$(grep -nE 'id_(rsa|ed25519|ecdsa|dsa)"' "${fontes[@]}" | grep -vF 'efetivo.go:23:var chavesPadrao = []string{' || true)"

t "o motor só escreve em dois lugares: o arquivo do app (com a linha do Include) e as pastas das máquinas"
eq "internal/sshcfg/gravar.go internal/vm/loja.go" "$(grep -lE 'os\.(WriteFile|Create|CreateTemp|Rename|OpenFile|Remove|RemoveAll|Mkdir|MkdirAll|Chmod)\(' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"
eq "" "$(grep -nE 'os\.(WriteFile|Create|OpenFile|MkdirAll|RemoveAll)\(' "${fontes[@]}" | grep -v '/internal/vm/loja.go:' || true)"
# Apagar uma máquina: só dois RemoveAll, os dois de um caminho que o filhoDireto
# conferiu (um filho direto da raiz, nunca um atalho, nunca a própria raiz).
eq 2 "$(grep -c 'os\.RemoveAll(' "$src/internal/vm/loja.go")"
like "$(<"$src/internal/vm/loja.go")" 'if err := filhoDireto\(l\.Raiz, l\.Pasta\(nome\)\); err != nil \{[[:space:]]+return fmt\.Errorf\("não apago: %v", err\)'
like "$(<"$src/internal/vm/loja.go")" 'if err := filhoDireto\(l\.Execucao, l\.Corrida\(nome\)\); err == nil \{[[:space:]]+_ = os\.RemoveAll\(l\.Corrida\(nome\)\)'
# O único sinal que o motor manda é o SIGTERM do forçar, depois de conferir
# no /proc que o processo é o QEMU daquela máquina.
eq "vm_ligar.go" "$(grep -lE 'syscall\.Kill\(' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"
eq 1 "$(grep -c 'syscall\.Kill(' "$src/vm_ligar.go")"
like "$(<"$src/vm_ligar.go")" 'if !vm\.EDaMaquina\(oProc\(\), s\.Pid, m\.Nome, l\.Pid\(m\.Nome\)\) \{[^}]*\}[[:space:]]+if err := syscall\.Kill\(s\.Pid, syscall\.SIGTERM\)'
# O QEMU nasce num grupo próprio, sem shell e sem daemonize.
ok grep -qF 'cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}' "$src/vm_ligar.go"
eq "" "$(grep -n 'daemonize' "$src/internal/vm/argv.go" || true)"
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
for c in listar sondar entrar guardar remover chaves comando vm; do
  has "$out" "$c" "subcomando do contrato"
  has "$contrato" "## \`mkporto-motor $c" "o contrato não descreve: $c"
done
eq 8 "$(jq -r .erro <<<"$out" | tr '|' '\n' | wc -l)"

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

# --- as máquinas, com o binário de verdade e nenhum QEMU --------------------------
# (qemu-system-x86_64 e qemu-img são guardas do arnês: se o motor rodasse um
# deles sem ser o dublê, o arquivo reprovava.)
mq=$SBX/maquinas
vmotor() { MKPORTO_SSH_DIR=$ssh MKPORTO_MAQUINAS=$mq MKPORTO_EXECUCAO=$SBX/run/mq MKPORTO_OVMF=$SBX/ovmf MKPORTO_KVM=$SBX/kvm "$bin" vm "$@"; }

t "o binário diz o que falta para ter máquinas e o que abrir para instalar, sem rodar nada"
calls_reset
out=$(MKPORTO_QEMU=qemu-que-nao-existe MKPORTO_TERMINAL=foot vmotor requisitos)
eq 0 $?
jeq 'false' "$(jq -c '.pronto' <<<"$out")"
jeq '[false,false,false]' "$(jq -c '[.qemu.tem, .ovmf.tem, .kvm.tem]' <<<"$out")"
jeq '["sudo","pacman","-S","--needed","qemu-desktop","edk2-ovmf"]' "$(jq -c '.instalar.argv' <<<"$out")"
jeq '["foot","--title=instalar o QEMU","--hold","sudo","pacman","-S","--needed","qemu-desktop","edk2-ovmf"]' "$(jq -c '.instalar.terminal' <<<"$out")"
eq "" "$(cat "$SBX/calls")"
out=$(vmotor listar)
jeq '[]' "$(jq -c .maquinas <<<"$out")"

t "o binário cria uma máquina (pasta 700, disco pelo qemu-img, maquina.json) e a lista desligada"
# shellcheck disable=SC2016 # o $4 é do dublê, não daqui
stub qemu-img '[[ $1 == create ]] && : >"$4"'
out=$(vmotor criar <<<'{"nome":"oficina","memoria_mb":2048,"cpus":2,"disco_gb":20,"iso":"","uefi":false,"tres_d":false}')
eq 0 $?
eq "true" "$(jq -r .ok <<<"$out")"
called "qemu-img create -f qcow2 $mq/oficina/disco.qcow2 20G"
eq "700" "$(stat -c %a "$mq/oficina")"
jeq '{"nome":"oficina","memoria_mb":2048,"cpus":2,"disco_gb":20,"iso":"","uefi":false,"tres_d":false}' \
  "$(jq -c 'del(.porta_ssh, .criada_em)' "$mq/oficina/maquina.json")"
ok test "$(jq .porta_ssh "$mq/oficina/maquina.json")" -ge 2222
jeq '["oficina","desligada",null]' "$(vmotor listar | jq -c '.maquinas[0] | [.nome, .estado, .pid]')"

t "o binário recusa uma ISO com vírgula e um nome com cara de opção, sem criar nada"
: >"$SBX/a,b.iso"
out=$(vmotor criar <<<"{\"nome\":\"outra\",\"memoria_mb\":2048,\"cpus\":2,\"disco_gb\":20,\"iso\":\"$SBX/a,b.iso\",\"uefi\":false,\"tres_d\":false}")
eq 1 $?
eq "iso" "$(jq -r .campo <<<"$out")"
has "$(jq -r .erro <<<"$out")" "vírgula"
out=$(vmotor criar <<<'{"nome":"-enable-kvm","memoria_mb":2048,"cpus":2,"disco_gb":20,"iso":"","uefi":false,"tres_d":false}')
eq 1 $?
eq "nome" "$(jq -r .campo <<<"$out")"
out=$(vmotor criar <<<'{"nome":"comuefi","memoria_mb":2048,"cpus":2,"disco_gb":20,"iso":"","uefi":true,"tres_d":false}')
eq 1 $?
has "$(jq -r .erro <<<"$out")" "edk2-ovmf"
eq "oficina" "$(ls -A "$mq")"

t "sem o QEMU o binário não liga e diz o pacote; sem onde pôr o socket, recusa"
out=$(MKPORTO_QEMU=qemu-que-nao-existe vmotor ligar oficina)
eq 1 $?
has "$(jq -r .erro <<<"$out")" "qemu-desktop"
out=$(MKPORTO_EXECUCAO='' XDG_RUNTIME_DIR='' MKPORTO_SSH_DIR=$ssh MKPORTO_MAQUINAS=$mq "$bin" vm ligar oficina)
eq 2 $?
has "$(jq -r .erro <<<"$out")" "XDG_RUNTIME_DIR"

t "um pid que sobrou (de outro programa) não faz a máquina parecer ligada, e ninguém é sinalizado"
mkdir -p "$SBX/run/mq/oficina"
echo $$ >"$SBX/run/mq/oficina/pid"
jeq '"desligada"' "$(vmotor listar | jq -c '.maquinas[0].estado')"
absent "$SBX/run/mq/oficina/pid"
out=$(vmotor desligar --forcar oficina)
eq 1 $?
ok kill -0 $$

t "o binário põe a máquina na área Hosts e só a remove com o nome repetido"
out=$(vmotor host oficina)
eq 0 $?
ok grep -q 'HostName 127.0.0.1' "$ssh/config.d/mkporto.conf"
ok grep -q '^Host oficina$' "$ssh/config.d/mkporto.conf"
out=$(vmotor remover oficina)
eq 2 $?
out=$(vmotor remover --confirmo outra -- oficina)
eq 2 $?
exists "$mq/oficina/maquina.json"
out=$(vmotor remover --confirmo oficina -- oficina)
eq 0 $?
absent "$mq/oficina"
exists "$mq"
