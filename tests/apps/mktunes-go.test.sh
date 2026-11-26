#!/usr/bin/env bash
# apps/mktunes/src: o mktunesd, o daemon do MkTunes (o player de música do Mk),
# em Go. Os testes dele são `go test` (com a guarda em pastas temporárias e o
# mpv, o yt-dlp, o sqlite3, o ffprobe, o barramento e a rede de mentira); aqui
# eles entram na suíte do Mk, junto com o formato (gofmt), o go vet, a
# conferência de que o binário que vai no pacote é o do código que está no
# repositório, uma conversa de ponta a ponta com esse binário (uma janela de
# mentira no socket: search, play, toggle, like) e as regras que o código não
# pode quebrar (só a biblioteca padrão, todo comando por variável e sem shell,
# o id do vídeo conferido antes de ir para a linha de comando).
. "$(dirname "$0")/../lib.sh"

command -v go >/dev/null || skip "falta o go"

app=$REPO/apps/mktunes
src=$app/src
bin=$app/bin/mktunesd
contrato=$(<"$app/CONTRATO.md")
# Nenhum destes roda de verdade aqui: quem os chama é o daemon, e ele recebe o
# caminho de um dublê por variável. Se algum teste escapar, cai nestes.
for c in mpv yt-dlp ffprobe xdg-user-dir; do stub "$c" 'exit 1'; done
# O cache de compilação fica fora da caixa de areia, para não recompilar tudo
# a cada rodada; o resto (HOME, módulos) continua dentro dela.
export GOCACHE=${MK_GOCACHE:-/var/tmp/mk-gocache-$(id -u)}
# -buildvcs=false: sem isso o go chama o git (que aqui é um guarda) para carimbar o binário.
export GOFLAGS="-mod=mod -buildvcs=false" GOPROXY=off GOTOOLCHAIN=local
cd "$src" || exit 1

t "o módulo não tem dependência de fora: só a biblioteca padrão"
eq "" "$(grep -E '^\s*require|^\s+[a-z0-9.-]+\.[a-z]+/' go.mod || true)"
absent "$src/go.sum"
# Nenhum import de fora da biblioteca padrão (os do próprio módulo começam com mktunes/).
eq "" "$(grep -rhE '^\s*"[a-z0-9.-]+\.[a-z]+/' --include='*.go' . || true)"

t "gofmt: todo arquivo está formatado"
eq "" "$(gofmt -l . 2>&1)"

t "go vet: nenhum aviso"
out=$(go vet ./... 2>&1) || _fail "$out"

t "go test: a suíte do daemon passa (com o detector de corridas, quando há compilador de C)"
corrida=()
if command -v gcc >/dev/null || command -v cc >/dev/null; then corrida=(-race); fi
out=$(go test -count=1 "${corrida[@]}" ./... 2>&1) || _fail "$(tail -40 <<<"$out")"
hasnt "$out" "FAIL"

t "todo pacote do daemon tem teste"
while read -r pacote; do
  compgen -G "$src/$pacote"'/*_test.go' >/dev/null || _fail "sem teste: $pacote"
done < <(go list ./... 2>/dev/null | grep -v '^mktunes$' | sed 's|^mktunes/||')
compgen -G "$src/*_test.go" >/dev/null || _fail "sem teste: o pacote principal"

t "os casos que não podem faltar estão na suíte"
for caso in TestFila TestListasInteligentes TestResumoDeEscuta TestCaminhoDoDownload TestLetras TestNormalizacaoEAjustes \
  TestSessaoToca TestSessaoControles TestAutoplay TestRetomar TestSoneca TestHistoricoDeEscuta TestDownloads TestBackup \
  TestBibliotecaVoltaDepoisDeReabrir TestHistoricoEmLinhas TestArquivoEstragadoOuNovoDemais TestBackupNoFormatoDaFonte \
  TestMigraOBancoAntigoUmaVez TestMigracaoQueNaoDaCertoDeixaTudoComoEstava TestMigraComOSqliteDeVerdade \
  TestArgumentos TestTocador TestProcessoFalaComUmMpv \
  TestInicioEContinuacao TestFilaERelogio TestPaginas TestBuscas TestBuscaDeMusicasDeduzida TestCliente TestFeed TestVolumeMedido \
  TestResolvePeloYtdlp TestSoUmYtdlpPorFaixaDeCadaVez TestFalhasViramStreamIndisponivel TestSemYtdlpAvisaOPacoteUmaVez TestOCacheTemTeto \
  TestLetrasDoLrclib TestRadioBrowser TestArquivos TestPastaDeMusica TestEtiquetasDoFfprobe \
  TestPlayer TestBusca TestInicioDoApp TestBibliotecaDoApp TestDownloadsDoApp TestCompasso TestAjustesDoApp \
  TestAFilaEAPosicaoVoltamDepoisDeReiniciar TestServidorDoSocket TestComandoDeQuemJaFoiEmboraAindaChega TestMelhorResultado \
  TestBytesPelaEspecificacao TestTodosOsTiposVaoEVoltam TestRecusaOQueNaoCabe TestMensagemOla TestConversaComUmBarramentoDeMentira \
  TestSemBarramentoOuRecusado TestModelo TestServicoRespondeComoUmPlayer TestOsBotoesDoSistemaChegamComoComandos TestContaAsMudancasEOsPulos \
  TestConversaDePontaAPonta TestSemMpvODaemonNaoSobe; do
  grep -qhE "^func $caso\(" "$src"/*_test.go "$src"/internal/*/*_test.go || _fail "falta o caso: $caso"
done

t "as respostas gravadas do InnerTube estão lá, e a deduzida diz que é deduzida"
for f in home home-continuation next album playlist artist search-albums search-artists search-playlists search-suggestions player-vr; do
  ok jq -e . "$src/internal/innertube/testdata/$f.json" >/dev/null
done
ok jq -e . "$src/internal/innertube/testdata/search-songs.DEDUZIDO.json" >/dev/null

# O código que vai no binário (sem os testes).
mapfile -t fontes < <(find "$src" -name '*.go' ! -name '*_test.go' | sort)

t "todo arquivo Go traz o cabeçalho SPDX da licença"
for f in $(find "$src" -name '*.go' | sort); do
  eq "// SPDX-License-Identifier: GPL-3.0-or-later" "$(head -1 "$f")" "$f"
done

t "todo comando externo vem por variável de ambiente, e são os do contrato"
for v in MKTUNES_MPV MKTUNES_YTDLP MKTUNES_SQLITE3 MKTUNES_FFPROBE MKTUNES_LAUNCHER; do
  ok grep -qF "ambiente(\"$v\"," "$src/main.go"
  has "$contrato" "\`$v\`" "o contrato não cita a variável $v"
done
for v in MKTUNES_MPV_ARGS MKTUNES_MUSIC_DIR MKTUNES_MPRIS MKTUNES_INNERTUBE_URL MKTUNES_LRCLIB_URL MKTUNES_RADIO_URL; do
  ok grep -qF "os.Getenv(\"$v\")" "$src/main.go"
  has "$contrato" "\`$v\`" "o contrato não cita a variável $v"
done
# Fora do main, nenhum nome de comando solto: os pacotes recebem o comando de quem os monta.
eq "" "$(grep -nE '"(mpv|yt-dlp|sqlite3|ffprobe|ffmpeg|xdg-user-dir|setsid|busctl|dbus-send|playerctl)"' "${fontes[@]}" | grep -v "^$src/main.go:" || true)"

t "nenhum comando passa por shell, e todo o que pode demorar tem prazo"
# ("su" fica de fora da lista: é uma assinatura de tipos do D-Bus, texto e uint32.)
eq "" "$(grep -nE '"(sh|bash|/bin/sh|/bin/bash|zsh|env|sudo|doas|run0|pkexec)"|"-c"' "${fontes[@]}" || true)"
# Os dois que ficam no ar (o mpv e a janela que o Raise abre) são os únicos sem prazo.
eq "$(printf '%s\n' "internal/mpv/mpv.go" "main.go")" "$(grep -lE 'exec\.Command\(' "${fontes[@]}" | sed "s|^$src/||" | sort)"
eq "3" "$(grep -hoE 'exec\.CommandContext\(' "${fontes[@]}" | wc -l)"
ok grep -qF 'context.WithTimeout(ctx, prazoDoSqlite)' "$src/internal/guarda/migra.go"
ok grep -qF 'context.WithTimeout(ctx, prazoFfprobe)' "$src/internal/local/local.go"
ok grep -qF 'context.WithTimeout(context.Background(), prazo)' "$src/internal/ytdlp/ytdlp.go"

t "o yt-dlp e o mpv recebem os argumentos do contrato, e o id do vídeo é conferido antes"
ok grep -qF 'return []string{"-f", seletor, "-g", "--no-playlist", "--no-warnings", "--", "https://music.youtube.com/watch?v=" + id}' "$src/internal/ytdlp/ytdlp.go"
ok grep -qF 'idDeVideo = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)' "$src/internal/ytdlp/ytdlp.go"
ok grep -qF 'var hostsDoYouTube = []string{"googlevideo.com", "youtube.com"}' "$src/internal/ytdlp/ytdlp.go"
# A conferência do id vem antes de qualquer outra coisa no Resolve.
like "$(grep -n -A8 '^func (r \*Resolvedor) Resolve' "$src/internal/ytdlp/ytdlp.go")" 'IDValido\(f\.ID\)'
ok grep -qF '"--config=no", "--load-scripts=no", "--idle=yes", "--no-video", "--no-terminal",' "$src/internal/mpv/mpv.go"
ok grep -qF '"--audio-display=no", "--input-ipc-server=" + socket, "--keep-open=no",' "$src/internal/mpv/mpv.go"
ok grep -qF 'exec.CommandContext(ctx, sqlite3, "-readonly", "-json", filepath.Clean(banco), sql)' "$src/internal/guarda/migra.go"

t "a migração não apaga o banco antigo: só o renomeia depois de gravar tudo"
eq "" "$(grep -nE 'os\.Remove(All)?\(' "$src/internal/guarda/migra.go" || true)"
ok grep -qF 'destino := banco + ".migrado"' "$src/internal/guarda/migra.go"
# Em Go, nenhum RemoveAll no que vai no binário.
eq "" "$(grep -nE 'RemoveAll\(' "${fontes[@]}" || true)"

t "a memória tem teto: os caches e as filas dizem o limite"
ok grep -qE '^\s*guardados = 32$' "$src/internal/ytdlp/ytdlp.go"
ok grep -qE '^const lembradas = 16$' "$src/internal/app/casos.go"
ok grep -qE '^const abertasGuardadas = 32$' "$src/internal/daemon/inicio.go"
ok grep -qE '^const avulsasNaFila = 64$' "$src/internal/daemon/quadro.go"
ok grep -qE '^const linhaMax = 8 << 20$' "$src/internal/daemon/servidor.go"
ok grep -qE '^\s*mensagemMax\s+= 16 << 20$' "$src/internal/dbus/fio.go"

t "todo comando do protocolo que a janela manda está no contrato e é tratado pelo daemon"
mapfile -t pedidos < <(grep -ohE 'cmd: *"[A-Za-z]+"' "$app"/qml/*.qml | grep -oE '"[A-Za-z]+"' | tr -d '"' | sort -u)
ok test "${#pedidos[@]}" -ge 45
for cmd in "${pedidos[@]}"; do
  has "$contrato" "\`$cmd\`" "comando fora do contrato: $cmd"
  grep -qhE "case .*\"$cmd\"|cmd != \"$cmd\"" "$src"/internal/daemon/*.go || _fail "o daemon não trata: $cmd"
done
# E todo estado que a janela lê é publicado.
for tipo in now progress search home library downloads radio lyrics settings volume local; do
  ok grep -qE "case \"$tipo\":" "$app/qml/Daemon.qml"
  grep -qhF "Estado(\"$tipo\"," "$src"/internal/daemon/*.go || _fail "o daemon não publica: $tipo"
  has "$contrato" "| \`$tipo\` |"
done

t "o binário do pacote existe, é estático e executável"
exists "$bin"
ok test -x "$bin"
like "$(file -b "$bin")" 'statically linked'

t "o binário do pacote saiu do código que está no repositório (rode apps/mktunes/pack depois de mexer em src/)"
eq "$("$app/assinatura")" "$(head -1 "$bin.de" 2>/dev/null)"

t "o binário diz a versão do arquivo VERSION, e recusa opção desconhecida com saída 2"
versao=$("$bin" --versao)
eq 0 $?
eq "$(cut -d' ' -f1 "$app/VERSION")" "$versao"
"$bin" --inventada 2>/dev/null
eq 2 $?

t "sem daemon no socket, --ping sai com 1 e --encerra com 0 (não há quem encerrar)"
"$bin" --ping >/dev/null 2>&1
eq 1 $?
ok "$bin" --encerra

t "ponta a ponta: o binário do pacote conversa com uma janela de mentira (search, play, toggle, like)"
out=$(MKTUNESD_BIN=$bin go test -count=1 -run 'TestConversaDePontaAPonta|TestSemMpvODaemonNaoSobe' -v . 2>&1) || _fail "$(tail -40 <<<"$out")"
has "$out" "--- PASS: TestConversaDePontaAPonta"
has "$out" "--- PASS: TestSemMpvODaemonNaoSobe"
# A conversa usa mesmo o binário do pacote quando MKTUNESD_BIN aponta para ele.
ok grep -qF 'binario := os.Getenv("MKTUNESD_BIN")' "$src/main_test.go"
for cmd in search play toggle like; do
  ok grep -qF "{\"cmd\":\"$cmd\"" "$src/main_test.go"
done

t "nenhum teste Go fala com a rede de verdade nem com o barramento de verdade sem pedirem"
# Nenhum teste usa o cliente HTTP padrão: a rede é sempre um httptest, e os
# adaptadores recebem o endereço dele.
eq "" "$(grep -nE 'http\.(DefaultClient|Get|Post|Head)\b' "$src"/*_test.go "$src"/internal/*/*_test.go || true)"
eq "" "$(grep -nE '(Lrclib|RadioBrowser)\{' "$src"/internal/*/*_test.go | grep -vE 'Base: |Servidores: |= (Lrclib|RadioBrowser)\{\}' || true)"
eq "" "$(grep -nE '\&Cliente\{' "$src"/internal/*/*_test.go | grep -v 'Base: s.URL' || true)"
ok grep -qF 'if os.Getenv("MKTUNES_BARRAMENTO_DE_VERDADE") != "1" {' "$src/internal/dbus/verdade_test.go"
ok grep -qF '"DBUS_SESSION_BUS_ADDRESS=", "MKTUNES_PAPEL=daemon"' "$src/main_test.go"
