#!/usr/bin/env bash
# apps/mksom/src: o motor do MkSom (o equalizador e o teste de microfone), em
# Go. Os testes dele são `go test` (com pastas de mentira e dublês de
# systemctl, pw-dump, pw-cli, wpctl, pw-record e pw-play); aqui eles entram na
# suíte do Mk, junto com o formato (gofmt), o go vet, a conferência de que o
# binário que vai no pacote é o do código que está no repositório e as regras
# que o código não pode quebrar (todo comando vem por variável e com prazo,
# nada passa por shell, o motor não roda o pipewire, só escreve nos arquivos
# do app e nunca em ~/.config/pipewire, a gravação só vai para a memória).
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

app=$REPO/apps/mksom
src=$app/src
bin=$app/bin/mksom-motor
# Os comandos do PipeWire não estão entre os guardas do lib.sh: aqui eles são
# sempre de mentira, para o de verdade nunca rodar (nem por descuido).
for c in pw-dump pw-cli pw-record pw-play pipewire; do stub "$c" 'exit 1'; done
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
done < <(go list ./... 2>/dev/null | grep -v '^mksom$' | sed 's|^mksom/||')
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

t "todo subcomando do contrato tem o seu teste de contrato em Go"
for c in Estado EqLigar EqDesligar EqAjustar EqPredefinicao MicListar MicPadraoVolumeMudo MicNivel MicGravar MicOuvir; do
  grep -qE "^func Test${c}_Contrato" "$src"/*_test.go || _fail "sem teste de contrato: $c"
done

t "os casos que não podem faltar estão na suíte"
for caso in TestConf_EstruturaNomesEValores TestConf_SemAlvoOuComAlvoEstranhoNaoEscreveTargetObject TestMudancas_SoOQueMudou TestUnidade \
  TestEqLigar_Contrato TestEqLigar_EsperaOSinkAparecer TestEqLigar_DesfazQuandoOSinkNaoAparece TestEqLigar_DesfazQuandoOEnableOuOSetDefaultFalham \
  TestEqLigar_SemPipewireNaoEscreveNadaNemChamaNinguem TestEqLigar_JaNoArSoTornaPadraoENaoLembraOProprioEqualizador \
  TestEqDesligar_Contrato TestEqDesligar_AnteriorSumiuNaoInventa TestEqAjustar_Contrato TestEqAjustar_Validacoes \
  TestEqAjustar_PwCliQueFalhaNaoGravaOEstado TestEqPredefinicao_AsDeFabricaSaoRecusadas TestEqPredefinicao_NomesInvalidosEQueNaoExistem \
  TestMic_Validacoes TestMicNivel_ParaQuandoAEntradaFechaEMataOFilho TestMicNivel_FilhoQueIgnoraOSinalEMortoAForca \
  TestMicGravar_InterrompidoApagaOArquivo TestMicGravar_SemPastaEmMemoriaRecusaEErros TestMicOuvir_ErroEInterrupcaoTambemApagam \
  TestNadaEscreveNaConfiguracaoDoPipewire TestUmCicloInteiroSoTocaNasPastasDoApp TestEscreveAtomico_NuncaNaPastaDoPipewire \
  TestLe_Tolerante TestLe_NosPadraoVolumeEMudo TestDe_SilencioSenoideESaturado TestMedir_PulaUmCabecalhoWav TestValidaNome \
  TestEstado_SemPipewireAindaAbre TestEstado_LigadoEODoPipewireNaoODoArquivo; do
  grep -qhE "^func $caso\(" "$src"/*_test.go "$src"/internal/*/*_test.go || _fail "falta o caso: $caso"
done

t "a suíte Go nunca roda comando de verdade nem toca ou grava som"
# Todo teste que roda um subcomando monta o mundo de mentira antes (HOME, as
# pastas do app, PATH vazio e cada comando externo apontando para um dublê).
awk '/^func Test/ { nome = $2; mundo = 0 }
     /mundo\(t\)/ { mundo = 1 }
     /(roda|quer|querErro|fluxo)\(t, |rodar\(\[\]string/ { if (nome != "" && !mundo) print nome }' "$src/main_test.go" | sort -u >"$SBX/sem-mundo"
eq "" "$(<"$SBX/sem-mundo")"
ok grep -qF 't.Setenv("PATH", c.bin)' "$src/main_test.go"
ok grep -qF 't.Setenv(variavel, filepath.Join(c.bin, nome))' "$src/main_test.go"
for v in MKSOM_SYSTEMCTL MKSOM_PW_DUMP MKSOM_PW_CLI MKSOM_WPCTL MKSOM_PIPEWIRE MKSOM_PW_RECORD MKSOM_PW_PLAY; do
  ok grep -qF "\"$v\"" "$src/main_test.go"
done
# Os dublês são scripts de /bin/sh que só usam o que o shell tem dentro.
eq "" "$(grep -nE '/usr/bin|/bin/(cat|sleep|cp|head|dd)' "$src"/*_test.go "$src"/internal/*/*_test.go || true)"

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "todo comando externo vem por variável de ambiente, e são só os sete do contrato"
eq "$(printf '%s\n' \
  'MKSOM_PIPEWIRE pipewire' 'MKSOM_PW_CLI pw-cli' 'MKSOM_PW_DUMP pw-dump' 'MKSOM_PW_PLAY pw-play' \
  'MKSOM_PW_RECORD pw-record' 'MKSOM_SYSTEMCTL systemctl' 'MKSOM_WPCTL wpctl')" \
  "$(grep -ohE 'externo\{"MKSOM_[A-Z_]+", "[a-z-]+"\}' "$src/som.go" | sed -E 's/externo\{"([A-Z_]+)", "([a-z-]+)"\}/\1 \2/' | sort)"
contrato=$(<"$app/CONTRATO.md")
for v in MKSOM_SYSTEMCTL MKSOM_PW_DUMP MKSOM_PW_CLI MKSOM_WPCTL MKSOM_PIPEWIRE MKSOM_PW_RECORD MKSOM_PW_PLAY MKSOM_CONFIG MKSOM_UNIDADES; do
  has "$contrato" "\`$v\`" "o contrato não cita a variável $v"
done
# Nenhum nome de comando solto no código: só os da tabela acima.
eq "" "$(grep -nE '"(systemctl|pw-dump|pw-cli|wpctl|pw-record|pw-play|pw-cat|pactl|pw-metadata)"' "${fontes[@]}" | grep -v 'som.go:.*externo{' || true)"
# As pastas também.
ok grep -qF 'os.Getenv("MKSOM_CONFIG")' "$src/internal/base/base.go"
ok grep -qF 'os.Getenv("MKSOM_UNIDADES")' "$src/internal/base/base.go"

t "todo comando externo roda sem shell e com prazo (ou até a janela mandar parar)"
# Um só lugar acha o executável, e dois o rodam: o de uma resposta (com prazo)
# e o de um fluxo (que morre com o contexto, pelo sinal e depois à força).
eq "1" "$(grep -hoE 'exec\.LookPath\(' "${fontes[@]}" | wc -l)"
eq "" "$(grep -nE 'exec\.Command\(' "${fontes[@]}" || true)"
eq "2" "$(grep -hoE 'exec\.CommandContext\(' "${fontes[@]}" | wc -l)"
ok grep -qF 'ctx, cancela := context.WithTimeout(context.Background(), prazoDoComando)' "$src/som.go"
ok grep -qF 'cmd.WaitDelay = prazoDeMorrer' "$src/fluxo.go"
ok grep -qF 'cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}' "$src/fluxo.go"
eq "" "$(grep -nE '"(sh|bash|/bin/sh|/bin/bash|zsh|env|sudo|su|doas|run0|pkexec)"|"-c"' "${fontes[@]}" || true)"
# (o "-c" do pipewire é texto da unidade do systemd, não argumento de comando)
ok grep -qF '"ExecStart=" + pipewire + " -c \"" + r.Replace(conf) + "\"\n"' "$src/internal/eq/eq.go"

t "o motor não roda o pipewire: só escreve o nome dele na unidade"
eq "eq.go:		if err := base.EscreveAtomico(arquivoDaUnidade(), []byte(eq.Unidade(oPipewire.nome(), arquivoDoConf()))); err != nil {" \
  "$(grep -H 'oPipewire\.' "${fontes[@]}" | sed "s|^$src/||")"

t "os argumentos dos comandos são os do contrato"
ok grep -qF 'oSystemctl.roda(append([]string{"--user"}, args...)...)' "$src/eq.go"
ok grep -qF 'systemctl("daemon-reload")' "$src/eq.go"
ok grep -qF 'systemctl("enable", "--now", eq.NomeDaUnidade)' "$src/eq.go"
eq "2" "$(grep -c 'systemctl("disable", "--now", eq.NomeDaUnidade)' "$src/eq.go")"
ok grep -qF 'oPwCli.roda("set-param", strconv.Itoa(sink.ID), "Props", m.Props())' "$src/eq.go"
ok grep -qF '"--target", alvoDe(no), "--format", "s16", "--rate", strconv.Itoa(taxaDoMedidor), "--channels", "1", "-")' "$src/fluxo.go"
ok grep -qF '"--target", alvoDe(no), "--format", "s16", "--rate", strconv.Itoa(taxaDaGravacao), "--channels", "1", arquivo)' "$src/fluxo.go"
ok grep -qF 'wpctl("set-volume", strconv.Itoa(no.ID), fmt.Sprintf("%d.%02d", pct/100, pct%100))' "$src/mic.go"
# Só um id conferido (número, e de uma entrada do pw-dump) chega ao wpctl e ao pw-record.
eq "5" "$(grep -c 'no, codigo := entradaDe(' "$src/mic.go" "$src/fluxo.go" | awk -F: '{ n += $2 } END { print n }')"
ok grep -qF 'soNumero  = regexp.MustCompile(`^[0-9]{1,7}$`)' "$src/mic.go"

t "o motor só escreve por um caminho, atômico, e nunca na configuração do PipeWire"
eq "internal/base/base.go" "$(grep -lE 'os\.(WriteFile|Create|CreateTemp|Rename|OpenFile)\(' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"
ok grep -qF 'os.Rename(nome, caminho)' "$src/internal/base/base.go"
like "$(<"$src/internal/base/base.go")" 'func EscreveAtomico\(caminho string, dados \[\]byte\) error \{[[:space:]]+if DoSistemaDeSom\(caminho\) \{[[:space:]]+return'
# As pastas que o motor cria: a do app e a das unidades (pela escrita atômica) e a da gravação.
eq "fluxo.go internal/base/base.go" "$(grep -lE 'os\.(Mkdir|MkdirAll)\(' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"
# O único arquivo que o motor apaga é a gravação de teste (e o temporário da escrita atômica).
eq "" "$(grep -nE 'os\.(Remove|RemoveAll)\(' "${fontes[@]}" | grep -vE 'os\.Remove\((arquivo|nome)\)' || true)"
eq "" "$(grep -n 'RemoveAll' "${fontes[@]}" || true)"
# "pipewire" como pasta só aparece para ser recusado.
eq "internal/base/base.go" "$(grep -lE 'Config\("(pipewire|wireplumber)"\)|"\.config", "(pipewire|wireplumber)"' "${fontes[@]}" | sed "s|^$src/||" | tr '\n' ' ' | sed 's/ $//')"

t "a gravação de teste só mora em XDG_RUNTIME_DIR: nada de /tmp nem da home"
eq "" "$(grep -nE '"/tmp|os\.TempDir\(|"/var/tmp|"Música"|"Music"' "${fontes[@]}" || true)"
ok grep -qF 'd := os.Getenv("XDG_RUNTIME_DIR")' "$src/internal/base/base.go"
ok grep -qF 'defer os.Remove(arquivo)' "$src/fluxo.go"

t "o motor não fala com a rede"
eq "" "$(grep -nE '"net"|"net/http"|"net/url"' "${fontes[@]}" || true)"

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mksom/pack depois de mexer em src/)"
eq "$("$app/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário responde ao contrato: sem subcomando, o uso em JSON e saída 1"
out=$("$bin" 2>/dev/null)
eq 1 $?
eq "uso: mksom-motor <eq | estado | mic>" "$(jq -r .erro <<<"$out")"
for c in estado "eq ligar" "eq desligar" "eq ajustar" "eq predefinicao" "mic listar" "mic padrao" "mic nivel" "mic gravar"; do
  has "$contrato" "## \`mksom-motor $c" "o contrato não descreve: $c"
done
for c in "mic volume" "mic mudo" "mic ouvir" "mic limpar"; do
  has "$contrato" "\`$c" "o contrato não descreve: $c"
done

# --- o binário de verdade numa casa de mentira ----------------------------------
# O PipeWire de mentira: um pw-dump que mostra o que há em $SBX/dump.json e um
# systemctl que faz o sink do equalizador aparecer e sumir.
no() { printf '{"id": %s, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "%s", "node.description": "%s", "media.class": "%s", "object.serial": %s}, "params": {"Props": [{"mute": false, "channelVolumes": [0.125, 0.125]}]}}}' "$@"; }
dump() { # dump <com o equalizador? 0|1> <node.name da saída padrão>
  echo "["
  no 51 alsa_output.exemplo.analog-stereo "Alto-falantes" Audio/Sink 310
  echo ,
  no 52 alsa_input.exemplo.analog-stereo "Microfone interno" Audio/Source 311
  echo ,
  if [[ $1 == 1 ]]; then
    no 71 mksom.eq "Mk Equalizador" Audio/Sink 320
    echo ,
  fi
  printf '{"id": 36, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [{"key": "default.audio.sink", "value": {"name": "%s"}}, {"key": "default.audio.source", "value": {"name": "alsa_input.exemplo.analog-stereo"}}]}\n]\n' "$2"
}
dump 0 alsa_output.exemplo.analog-stereo >"$SBX/desligado.json"
dump 1 mksom.eq >"$SBX/ligado.json"
cp "$SBX/desligado.json" "$SBX/dump.json"
stub pw-dump 'cat "$SBX/dump.json"'
stub systemctl 'case "$*" in
  *"enable --now"*) cp "$SBX/ligado.json" "$SBX/dump.json" ;;
  *"disable --now"*) cp "$SBX/desligado.json" "$SBX/dump.json" ;;
esac'
stub wpctl
stub pw-cli
cfg=$HOME/.config/mk/mksom
unidade=$HOME/.config/systemd/user/mksom-eq.service
motor() { "$bin" "$@"; }

t "o binário guarda os ganhos com o equalizador desligado, sem escrever conf nem chamar ninguém"
out=$(motor eq ajustar <<<'{"ganhos":[3,2.5,0,-1,0,0.5,0,-12,12,4],"pre":-4.5}')
eq 0 $?
jeq '{"ok":true,"ao_vivo":false,"aplicados":[],"predefinicao":null}' "$out"
eq "pw-dump " "$(cat "$SBX/calls")"
absent "$cfg/eq.conf"
eq "600" "$(stat -c %a "$cfg/estado.json")"
eq "700" "$(stat -c %a "$cfg")"

t "o binário liga: escreve o conf e a unidade, recarrega, sobe, espera o sink e o torna padrão"
calls_reset
out=$(motor eq ligar)
eq 0 $?
jeq '{"ok":true,"ligado":true,"na_saida":true,"aviso":null,"saida_anterior":{"nome":"alsa_output.exemplo.analog-stereo","descricao":"Alto-falantes","existe":true}}' "$out"
eq "pw-dump |systemctl --user daemon-reload|systemctl --user enable --now mksom-eq.service|pw-dump |wpctl set-default 71|pw-dump " "$(tr '\n' '|' <"$SBX/calls" | sed 's/|$//')"
conf=$(<"$cfg/eq.conf")
has "$conf" '{ name = libpipewire-module-filter-chain'
has "$conf" 'node.name = "mksom.eq"'
has "$conf" 'media.class = Audio/Sink'
has "$conf" 'target.object = "alsa_output.exemplo.analog-stereo"'
has "$conf" '{ type = builtin name = eq_L_0 label = bq_lowshelf control = { "Freq" = 31.0 "Q" = 1.0 "Gain" = 3.0 } }'
has "$conf" '{ type = builtin name = eq_R_7 label = bq_peaking control = { "Freq" = 4000.0 "Q" = 1.0 "Gain" = -12.0 } }'
has "$conf" '{ type = builtin name = eq_R_9 label = bq_highshelf control = { "Freq" = 16000.0 "Q" = 1.0 "Gain" = 4.0 } }'
has "$conf" '{ type = builtin name = pre_L label = linear control = { "Mult" = 0.595662 "Add" = 0.0 } }'
eq 22 "$(grep -c '{ type = builtin name = ' "$cfg/eq.conf")"
eq "$(grep -o '{' "$cfg/eq.conf" | wc -l)" "$(grep -o '}' "$cfg/eq.conf" | wc -l)"
has "$(<"$unidade")" "ExecStart=pipewire -c \"$cfg/eq.conf\""
has "$(<"$unidade")" "WantedBy=default.target"
eq "true alsa_output.exemplo.analog-stereo" "$(jq -r '"\(.ligado) \(.saida_anterior)"' "$cfg/estado.json")"
# Nada fora das pastas do app: nem ~/.config/pipewire, nem ~/.config/wireplumber.
absent "$HOME/.config/pipewire"
absent "$HOME/.config/wireplumber"
not_called "pipewire -c"

t "o binário ajusta ao vivo: um pw-cli por banda que mudou, sem reiniciar nada"
calls_reset
out=$(motor eq ajustar <<<'{"ganhos":[3,2.5,0,4,0,0.5,0,-12,12,4],"pre":-6}')
eq 0 $?
jeq '{"ok":true,"ao_vivo":true,"aplicados":["pre_L:Mult","pre_R:Mult","eq_L_3:Gain","eq_R_3:Gain"],"predefinicao":null}' "$out"
eq 'pw-dump |pw-cli set-param 71 Props { params = [ "pre_L:Mult" 0.501187 "pre_R:Mult" 0.501187 ] }|pw-cli set-param 71 Props { params = [ "eq_L_3:Gain" 4.0 "eq_R_3:Gain" 4.0 ] }' \
  "$(tr '\n' '|' <"$SBX/calls" | sed 's/|$//')"
# O conf fica pronto para a próxima subida.
has "$(<"$cfg/eq.conf")" 'name = eq_L_3 label = bq_peaking control = { "Freq" = 250.0 "Q" = 1.0 "Gain" = 4.0 }'
has "$(<"$cfg/eq.conf")" '"Mult" = 0.501187'

t "o binário diz o estado de verdade (o do pw-dump) e as predefinições"
out=$(motor estado)
eq 0 $?
eq "true true -6 71" "$(jq -r '"\(.eq.ligado) \(.eq.na_saida) \(.eq.pre) \(.saida_padrao.id)"' <<<"$out")"
jeq '["plano","graves","agudos","voz","noite","fone pequeno"]' "$(jq -c '[.eq.predefinicoes[].nome]' <<<"$out")"
jeq '[{"id":52,"nome":"alsa_input.exemplo.analog-stereo","descricao":"Microfone interno","padrao":true,"volume":50,"mudo":false}]' "$(jq -c .entradas <<<"$out")"

t "o binário guarda, aplica e recusa predefinições"
out=$(motor eq predefinicao guardar -- "Fone do trabalho")
eq 0 $?
eq "Fone do trabalho" "$(jq -r .predefinicao <<<"$out")"
eq "600" "$(stat -c %a "$cfg/predefinicoes.json")"
calls_reset
out=$(motor eq predefinicao aplicar -- plano)
eq 0 $?
eq "true plano" "$(jq -r '"\(.ao_vivo) \(.predefinicao)"' <<<"$out")"
called 'pw-cli set-param 71 Props { params = [ "pre_L:Mult" 1.000000 "pre_R:Mult" 1.000000 ] }'
called 'pw-cli set-param 71 Props { params = [ "eq_L_7:Gain" 0.0 "eq_R_7:Gain" 0.0 ] }'
not_called '"eq_L_2:Gain"'
motor eq predefinicao apagar -- graves >/dev/null
eq 2 $?
motor eq predefinicao guardar -- 'a/b' >/dev/null
eq 1 $?
motor eq predefinicao guardar -- "$(printf 'a\nb')" >/dev/null
eq 1 $?
jeq '["Fone do trabalho"]' "$(jq -c '[.[].nome]' "$cfg/predefinicoes.json")"

t "o binário recusa ganhos fora do contrato e não aplica nada"
calls_reset
for pedido in '{"ganhos":[13,0,0,0,0,0,0,0,0,0],"pre":0}' '{"ganhos":[0,0,0],"pre":0}' '{"ganhos":[0.3,0,0,0,0,0,0,0,0,0],"pre":0}' \
  '{"ganhos":[0,0,0,0,0,0,0,0,0,0],"pre":1}' '{"ganhos":[0,0,0,0,0,0,0,0,0,0]}' 'x'; do
  out=$(motor eq ajustar <<<"$pedido")
  eq 1 $? "$pedido"
  ok jq -e 'has("erro")' <<<"$out" >/dev/null
done
not_called "pw-cli"

t "o binário desliga: devolve a saída de antes e derruba a unidade, nessa ordem"
calls_reset
out=$(motor eq desligar)
eq 0 $?
eq "false false" "$(jq -r '"\(.ligado) \(.na_saida)"' <<<"$out")"
eq "pw-dump |wpctl set-default 51|systemctl --user disable --now mksom-eq.service" "$(tr '\n' '|' <"$SBX/calls" | sed 's/|$//')"
eq "false" "$(jq -r .ligado "$cfg/estado.json")"

t "o binário desfaz quando o sink não aparece (a unidade não fica ligada pela metade)"
calls_reset
stub systemctl
out=$(motor eq ligar)
eq 1 $?
has "$(jq -r .erro <<<"$out")" "não apareceu no PipeWire"
eq "systemctl --user daemon-reload|systemctl --user enable --now mksom-eq.service|systemctl --user disable --now mksom-eq.service" \
  "$(grep -v '^pw-dump' "$SBX/calls" | tr '\n' '|' | sed 's/|$//')"
not_called "wpctl"
eq "false" "$(jq -r .ligado "$cfg/estado.json")"

t "o binário diz o erro quando o PipeWire não responde, sem chamar mais ninguém"
calls_reset
stub pw-dump 'echo "pw-dump: can not connect" >&2; exit 1'
out=$(motor eq ligar)
eq 1 $?
has "$(jq -r .erro <<<"$out")" "o PipeWire não respondeu"
eq "pw-dump " "$(cat "$SBX/calls")"
out=$(motor estado)
eq 0 $?
eq "false" "$(jq -r .som.responde <<<"$out")"
stub pw-dump 'cat "$SBX/dump.json"'

t "o binário mexe na entrada só com id e valores conferidos"
calls_reset
jeq '{"ok":true}' "$(motor mic volume 52 80)"
jeq '{"ok":true}' "$(motor mic mudo 52 sim)"
jeq '{"ok":true}' "$(motor mic padrao 52)"
eq "wpctl set-volume 52 0.80|wpctl set-mute 52 1|wpctl set-default 52" "$(grep '^wpctl' "$SBX/calls" | tr '\n' '|' | sed 's/|$//')"
calls_reset
for args in "volume 52 151" "volume 52 abc" "volume x 50" "mudo 52 talvez" "padrao 52;reboot" "padrao @DEFAULT_SOURCE@" "nivel x" "gravar x"; do
  # shellcheck disable=SC2086 # é para separar: são os argumentos
  motor mic $args >/dev/null
  eq 1 $? "$args"
done
motor mic padrao 51 >/dev/null # 51 é uma saída
eq 2 $?
motor mic volume 999 50 >/dev/null
eq 2 $?
not_called "wpctl"

t "o binário mede o nível: PCM conhecido vira dBFS, e o alvo é o serial do nó"
calls_reset
stub pw-record 'head -c 2132 /dev/zero; printf "\377\177%.0s" $(seq 1066)'
out=$(motor mic nivel 52 < <(sleep 3))
eq 0 $?
eq '{"pico":-90,"rms":-90}|{"pico":0,"rms":0}|{"fim":true}' "$(tr '\n' '|' <<<"$out" | sed 's/|$//')"
called "pw-record --raw --target 311 --format s16 --rate 16000 --channels 1 -"

t "o binário para o medidor quando a entrada padrão fecha, e o pw-record morre junto"
stub pw-record 'echo $$ >"$SBX/pid"; while :; do head -c 2132 /dev/zero || exit 0; done'
inicio=$(date +%s)
out=$(motor mic nivel 52 < <(sleep 0.4))
eq 0 $?
ok test $(($(date +%s) - inicio)) -le 4
eq '{"fim":true}' "$(tail -1 <<<"$out")"
ok test "$(grep -c pico <<<"$out")" -ge 1
fails kill -0 "$(<"$SBX/pid")" 2>/dev/null

t "o binário só grava na memória (XDG_RUNTIME_DIR), e apaga o que foi interrompido"
calls_reset
stub pw-record 'for a in "$@"; do f=$a; done; printf RIFF >"$f"; trap "exit 0" INT TERM; while :; do sleep 0.05; done'
out=$(motor mic gravar 52 < <(sleep 0.3))
eq 0 $?
eq '{"fim":true}' "$(tail -1 <<<"$out")"
called "pw-record --target 311 --format s16 --rate 48000 --channels 1 $XDG_RUNTIME_DIR/mksom/teste.wav"
absent "$XDG_RUNTIME_DIR/mksom/teste.wav"
eq "700" "$(stat -c %a "$XDG_RUNTIME_DIR/mksom")"
out=$(env -u XDG_RUNTIME_DIR "$bin" mic gravar 52)
eq 2 $?
has "$(jq -r .erro <<<"$out")" "não vai para disco"

t "o binário toca a gravação e a apaga; sem gravação, recusa"
calls_reset
stub pw-play
out=$(motor mic ouvir)
eq 2 $?
not_called "pw-play"
printf RIFF >"$XDG_RUNTIME_DIR/mksom/teste.wav"
out=$(motor mic ouvir < <(sleep 2))
eq 0 $?
jeq '{"ok":true}' "$out"
called "pw-play $XDG_RUNTIME_DIR/mksom/teste.wav"
absent "$XDG_RUNTIME_DIR/mksom/teste.wav"
printf RIFF >"$XDG_RUNTIME_DIR/mksom/teste.wav"
jeq '{"ok":true}' "$(motor mic limpar)"
absent "$XDG_RUNTIME_DIR/mksom/teste.wav"

t "depois de tudo, o motor só deixou arquivos nas pastas do app"
absent "$HOME/.config/pipewire"
absent "$HOME/.config/wireplumber"
eq "mk/mksom/eq.conf|mk/mksom/estado.json|mk/mksom/predefinicoes.json|systemd/user/mksom-eq.service" \
  "$(cd "$HOME/.config" && find mk systemd -type f | sort | tr '\n' '|' | sed 's/|$//')"

t "a suíte Go do motor é rápida (poucos segundos)"
inicio=$(date +%s)
out=$(go test -count=1 ./... 2>&1) || _fail "$(tail -20 <<<"$out")"
ok test $(($(date +%s) - inicio)) -le 30
