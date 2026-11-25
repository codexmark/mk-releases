package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sshFalso põe um ssh de mentira (shell, só com o que o shell tem dentro) e
// faz dele o MKPORTO_SSH. O corpo vê os argumentos como o ssh os veria.
func sshFalso(t *testing.T, bin, corpo string) {
	t.Helper()
	t.Setenv("MKPORTO_SSH", falso(t, bin, "ssh", corpo))
}

// chamadasCom devolve as chamadas ao ssh falso que têm este argumento.
func chamadasCom(t *testing.T, bin, argumento string) [][]string {
	t.Helper()
	var out [][]string
	for _, c := range chamadas(t, bin, "ssh") {
		for _, a := range c {
			if a == argumento {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// encurta troca um prazo do motor durante o teste.
func encurta(t *testing.T, prazo *time.Duration, novo time.Duration) {
	t.Helper()
	antigo := *prazo
	*prazo = novo
	t.Cleanup(func() { *prazo = antigo })
}

// --- listar com os valores do ssh -G --------------------------------------------

const sshG = `[ "$1" = -G ] || exit 9
case "$3" in
  atlas) printf 'host atlas\nhostname real.oficina.example\nuser marina\nport 2200\nidentityfile ~/.ssh/trabalho\nidentityfile ~/.ssh/outra\nproxyjump farol\n' ;;
  meu) printf 'hostname 192.0.2.9\nuser local\nport 22\nidentityfile ~/.ssh/id_rsa\nidentityfile ~/.ssh/id_ed25519\nproxyjump none\n' ;;
  comchave) printf 'hostname comchave\nuser local\nport 22\nidentityfile ~/.ssh/id_rsa\n' ;;
  *) exit 255 ;;
esac`

func TestListar_OsValoresVemDoSshG(t *testing.T) {
	_, bin := mundo(t, map[string]string{
		"config": "Include config.d/mkporto.conf\nHost atlas\n  HostName escrito.test\nHost comchave\n  IdentityFile ~/.ssh/id_rsa\n" +
			"Host semresposta\n  User ana\nHost -oProxyCommand=x\nHost a;b\n",
		"config.d/mkporto.conf": "Host meu\n  HostName 192.0.2.9\n",
	})
	sshFalso(t, bin, sshG)
	r := quer(t, "", "listar")

	atlas := hostDe(t, r, "atlas")
	if atlas["host"] != "real.oficina.example" || atlas["usuario"] != "marina" || atlas["porta"] != 2200.0 ||
		atlas["chave"] != "~/.ssh/trabalho" || atlas["salto"] != "farol" || atlas["efetivo"] != true {
		t.Errorf("atlas: %v", atlas)
	}
	if atlas["origem"] != "config" || atlas["arquivo"] != "~/.ssh/config" || atlas["escrito"] != nil {
		t.Errorf("de onde vem continua sendo do leitor próprio: %v", atlas)
	}

	// O usuário local e a chave padrão que o ssh -G sempre diz: o usuário
	// aparece; a chave padrão não é uma escolha do config.
	meu := hostDe(t, r, "meu")
	if meu["usuario"] != "local" || meu["chave"] != nil || meu["salto"] != nil || meu["efetivo"] != true {
		t.Errorf("meu: %v", meu)
	}
	// E o que está escrito no arquivo do app não se mistura com isso.
	escrito := map[string]any{"host": "192.0.2.9", "usuario": nil, "porta": 22.0, "chave": nil, "salto": nil}
	if !reflect.DeepEqual(meu["escrito"], escrito) {
		t.Errorf("escrito: %v", meu["escrito"])
	}
	// Uma chave padrão que ESTÁ no config aparece.
	if c := hostDe(t, r, "comchave"); c["chave"] != "~/.ssh/id_rsa" {
		t.Errorf("comchave: %v", c)
	}
	// O ssh não respondeu: valem os do leitor próprio.
	if s := hostDe(t, r, "semresposta"); s["efetivo"] != false || s["usuario"] != "ana" || s["host"] != "semresposta" {
		t.Errorf("semresposta: %v", s)
	}
	// Apelido que não é seguro nunca chega à linha de comando do ssh.
	for _, apelido := range []string{"-oProxyCommand=x", "a;b"} {
		if h := hostDe(t, r, apelido); h["efetivo"] != false {
			t.Errorf("%s: %v", apelido, h)
		}
	}
	vistos := []string{}
	for _, c := range chamadas(t, bin, "ssh") {
		if len(c) != 3 || c[0] != "-G" || c[1] != "--" {
			t.Errorf("chamada ao ssh: %q", c)
			continue
		}
		vistos = append(vistos, c[2])
	}
	if strings.Join(sortidas(vistos), " ") != "atlas comchave meu semresposta" {
		t.Errorf("o ssh -G foi chamado para: %v", vistos)
	}
}

func sortidas(l []string) []string {
	out := append([]string{}, l...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestListar_SshGQueFalhaDemoraOuDizBobagemValeOLeitorProprio(t *testing.T) {
	for nome, corpo := range map[string]string{
		"falha":        "exit 1",
		"bobagem":      "printf 'isto não é a saída do ssh -G\\n'",
		"sem hostname": "printf 'user x\\nport 22\\n'",
		"demora":       "while :; do :; done",
	} {
		_, bin := mundo(t, map[string]string{"config": "Host atlas\n  HostName escrito.test\n  User ana\n  Port 2222\n"})
		sshFalso(t, bin, corpo)
		encurta(t, &prazoDoG, 300*time.Millisecond)
		inicio := time.Now()
		h := hostDe(t, quer(t, "", "listar"), "atlas")
		if h["efetivo"] != false || h["host"] != "escrito.test" || h["usuario"] != "ana" || h["porta"] != 2222.0 {
			t.Errorf("%s: %v", nome, h)
		}
		if time.Since(inicio) > 5*time.Second {
			t.Errorf("%s: demorou %v", nome, time.Since(inicio))
		}
	}
}

func TestAplicaG_SoOQueFazSentido(t *testing.T) {
	_, _ = mundo(t, nil)
	h, ok := aplicaG("a", "", "Hostname  x.test \nUSER ana\nport 99999\nidentityfile ~/.ssh/k\nproxyjump NONE\nlixo\n\nhostname outro\n")
	if !ok || h.Host != "x.test" || h.Usuario != "ana" || h.Porta != 22 || h.Chave != "~/.ssh/k" || h.Salto != "" {
		t.Errorf("%+v %v", h, ok)
	}
	// A chave padrão, por extenso na home, também não conta.
	casa, _ := os.UserHomeDir()
	h, _ = aplicaG("a", "", "hostname x\nidentityfile "+filepath.Join(casa, ".ssh", "id_ed25519")+"\n")
	if h.Chave != "" {
		t.Errorf("chave padrão por extenso: %q", h.Chave)
	}
	h, _ = aplicaG("a", "~/.ssh/id_ed25519", "hostname x\nidentityfile ~/.ssh/id_ed25519\n")
	if h.Chave != "~/.ssh/id_ed25519" {
		t.Errorf("chave padrão que está no config: %q", h.Chave)
	}
	if _, ok := aplicaG("a", "", "user x\n"); ok {
		t.Errorf("sem hostname não é a saída do ssh -G")
	}
}

// --- sondar de verdade ----------------------------------------------------------

func TestSondar_PortaQueAbreMasNaoESsh(t *testing.T) {
	web, _ := escuta(t, "HTTP/1.1 400 Bad Request\r\n\r\n")
	muda, _ := escuta(t, "")
	conversa, _ := escuta(t, "bem-vindo\r\naviso: manutenção às 22h\r\nSSH-2.0-OpenSSH_10.0 "+strings.Repeat("x", 100)+"\x07\r\n")
	antigo, _ := escuta(t, "SSH-1.99-Antigo\n")
	mundo(t, map[string]string{"config": "Host *\n  HostName 127.0.0.1\nHost web\n  Port " + strconv.Itoa(web) +
		"\nHost muda\n  Port " + strconv.Itoa(muda) + "\nHost conversa\n  Port " + strconv.Itoa(conversa) +
		"\nHost antigo\n  Port " + strconv.Itoa(antigo) + "\n"})
	encurta(t, &prazoDaSaudacao, 400*time.Millisecond)

	_, linhas := roda(t, "", "sondar")
	r := porApelido(linhas)
	for _, apelido := range []string{"web", "muda"} {
		if v := r[apelido]; v["responde"] != false || v["motivo"] != "a porta abre, mas não é um servidor ssh" || v["servidor"] != nil {
			t.Errorf("%s: %v", apelido, v)
		}
	}
	// O servidor pode dizer outras linhas antes da saudação; o nome dele vem
	// cortado em 64 e só com o que dá para imprimir.
	v := r["conversa"]
	servidor, _ := v["servidor"].(string)
	if v["responde"] != true || len(servidor) != 64 || !strings.HasPrefix(servidor, "OpenSSH_10.0 xxx") {
		t.Errorf("conversa: %v", v)
	}
	if v := r["antigo"]; v["responde"] != true || v["servidor"] != "Antigo" {
		t.Errorf("antigo: %v", v)
	}
}

const sshW = `case " $* " in
  *" -G "*) exit 1 ;;
  *" -W "*) ;;
  *) exit 9 ;;
esac
for ultimo; do :; done
case "$ultimo" in
  farol|ana@farol) printf 'SSH-2.0-OpenSSH_9.9 Debian\r\n'; read espera ;;
  fechado) printf 'channel 0: open failed: administratively prohibited: open failed\nstdio forwarding failed\n' >&2; exit 255 ;;
  outracoisa) printf '220 mail.test ESMTP\r\n'; read espera ;;
  mudo) while :; do :; done ;;
esac`

func TestSondar_PorSalto(t *testing.T) {
	_, bin := mundo(t, map[string]string{"config": `
Host cofre
  HostName 192.0.2.77
  ProxyJump farol
Host comporta
  HostName cofre.interno.test
  Port 2200
  ProxyJump ana@farol:2222
Host barrado
  HostName 192.0.2.78
  ProxyJump fechado
Host errado
  HostName 192.0.2.79
  ProxyJump outracoisa
Host calado
  HostName 192.0.2.80
  ProxyJump mudo
Host cadeia
  HostName 192.0.2.81
  ProxyJump farol,outro
Host v6
  HostName 192.0.2.82
  ProxyJump [2001:db8::1]:22
`})
	sshFalso(t, bin, sshW)
	encurta(t, &prazoDoSalto, 500*time.Millisecond)
	encurta(t, &prazoDoG, 300*time.Millisecond)

	codigo, linhas := roda(t, "", "sondar")
	if codigo != 0 || len(linhas) != 7 {
		t.Fatalf("código %d, %v", codigo, linhas)
	}
	r := porApelido(linhas)
	if v := r["cofre"]; v["responde"] != true || v["via"] != "farol" || v["servidor"] != "OpenSSH_9.9 Debian" || v["motivo"] != nil {
		t.Errorf("cofre: %v", v)
	} else if ms, ok := v["ms"].(float64); !ok || ms < 0 {
		t.Errorf("cofre, ms: %v", v["ms"])
	}
	if v := r["comporta"]; v["responde"] != true || v["via"] != "ana@farol:2222" {
		t.Errorf("comporta: %v", v)
	}
	if v := r["barrado"]; v["responde"] != false || v["motivo"] != "o salto não deixou passar: stdio forwarding failed" || v["ms"] != nil {
		t.Errorf("barrado: %v", v)
	}
	for _, apelido := range []string{"errado", "calado"} {
		if v := r[apelido]; v["responde"] != false || v["motivo"] != "o salto não deixou passar" {
			t.Errorf("%s: %v", apelido, v)
		}
	}
	// O que não dá para sondar sem conversa continua "por salto".
	for _, apelido := range []string{"cadeia", "v6"} {
		if v := r[apelido]; v["responde"] != nil || v["motivo"] != "por salto" {
			t.Errorf("%s: %v", apelido, v)
		}
	}

	// As linhas de comando: BatchMode sempre, o destino depois de --.
	porDestino := map[string][]string{}
	for _, c := range chamadasCom(t, bin, "-W") {
		porDestino[c[len(c)-1]] = c
	}
	if len(porDestino) != 5 {
		t.Errorf("chamadas com -W: %v", porDestino)
	}
	if c := porDestino["farol"]; !reflect.DeepEqual(c, []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=4", "-W", "192.0.2.77:22", "--", "farol"}) {
		t.Errorf("cofre: %q", c)
	}
	if c := porDestino["ana@farol"]; !reflect.DeepEqual(c, []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=4", "-p", "2222", "-W", "cofre.interno.test:2200", "--", "ana@farol"}) {
		t.Errorf("comporta: %q", c)
	}
}

func TestSondar_PorSaltoSemOSshContinuaPorSalto(t *testing.T) {
	mundo(t, map[string]string{"config": "Host cofre\n  HostName 192.0.2.77\n  ProxyJump farol\n"})
	_, linhas := roda(t, "", "sondar")
	if v := linhas[0]; v["responde"] != nil || v["motivo"] != "por salto" || v["via"] != nil {
		t.Errorf("%v", v)
	}
}

func TestArgvDoSalto(t *testing.T) {
	casos := []struct {
		salto, host string
		porta       int
		quer        []string
	}{
		{"farol", "x.test", 22, []string{"S", "-o", "BatchMode=yes", "-o", "ConnectTimeout=4", "-W", "x.test:22", "--", "farol"}},
		{"ana@farol:2222", "2001:db8::1", 2200, []string{"S", "-o", "BatchMode=yes", "-o", "ConnectTimeout=4", "-p", "2222", "-W", "[2001:db8::1]:2200", "--", "ana@farol"}},
		{"a,b", "x.test", 22, nil},
		{"[2001:db8::1]:22", "x.test", 22, nil},
		{"ssh://farol", "x.test", 22, nil},
		{"-oProxyCommand=x", "x.test", 22, nil},
		{"farol", "-x", 22, nil},
		{"farol", "x y", 22, nil},
		{"farol:99999", "x.test", 22, nil},
		{"", "x.test", 22, nil},
	}
	for _, c := range casos {
		obtido, ok := argvDoSalto("S", c.salto, c.host, c.porta)
		if ok != (c.quer != nil) || !reflect.DeepEqual(obtido, c.quer) {
			t.Errorf("argvDoSalto(%q, %q) = %q %v", c.salto, c.host, obtido, ok)
		}
	}
}

func TestNomeDoServidor(t *testing.T) {
	for linha, quer := range map[string]string{
		"SSH-2.0-OpenSSH_10.0":               "OpenSSH_10.0",
		"SSH-2.0-OpenSSH_9.9 Debian-3\r":     "OpenSSH_9.9 Debian-3",
		"SSH-1.99-Antigo":                    "Antigo",
		"SSH-2.0-":                           "",
		"SSH-2.0":                            "",
		"SSH-2.0-a\x1b[31mb\x00c":            "a[31mbc",
		"SSH-2.0-" + strings.Repeat("é", 80): "",
	} {
		if obtido := nomeDoServidor(linha); obtido != quer {
			t.Errorf("nomeDoServidor(%q) = %q", linha, obtido)
		}
	}
}

// --- entrar ---------------------------------------------------------------------

const sshEntra = `case " $* " in *" -G "*) exit 1 ;; esac
for a; do penultimo=$ultimo; ultimo=$a; done
[ "$ultimo" = true ] || exit 9
case "$penultimo" in
  atlas) exit 0 ;;
  semchave) printf 'ana@192.0.2.5: Permission denied (publickey,password).\n' >&2; exit 255 ;;
  novo) printf 'Host key verification failed.\n' >&2; exit 255 ;;
  mudou) printf '@@@@@@@@@@\n@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\nHost key verification failed.\n' >&2; exit 255 ;;
  longe) printf 'ssh: connect to host 192.0.2.5 port 22: Connection timed out\n' >&2; exit 255 ;;
  fechado) printf 'ssh: connect to host 192.0.2.5 port 22: Connection refused\n' >&2; exit 255 ;;
  semnome) printf 'ssh: Could not resolve hostname x.test: Name or service not known\n' >&2; exit 255 ;;
  estranho) printf 'primeira\n\nkex_exchange_identification: Connection closed by remote host\n\n' >&2; exit 255 ;;
  quieto) exit 3 ;;
  preso) while :; do :; done ;;
esac`

func TestEntrar_Contrato(t *testing.T) {
	_, bin := mundo(t, map[string]string{"config": "Host atlas semchave novo mudou longe fechado semnome estranho quieto preso\nHost -x\nHost a;b\n"})
	sshFalso(t, bin, sshEntra)
	encurta(t, &prazoDaEntrada, 400*time.Millisecond)

	r := quer(t, "", "entrar", "--", "atlas")
	if !reflect.DeepEqual(campos(r), []string{"apelido", "entra", "motivo", "ms"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	if r["apelido"] != "atlas" || r["entra"] != true || r["motivo"] != nil {
		t.Errorf("%v", r)
	}
	if ms, ok := r["ms"].(float64); !ok || ms < 0 {
		t.Errorf("ms: %v", r["ms"])
	}
	// A linha de comando: sem conversa, sem pergunta de senha, e só `true` lá.
	c := chamadasCom(t, bin, "true")
	quer1 := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "NumberOfPasswordPrompts=0", "--", "atlas", "true"}
	if len(c) != 1 || !reflect.DeepEqual(c[0], quer1) {
		t.Errorf("chamada: %q", c)
	}

	for apelido, motivo := range map[string]string{
		"semchave": "a máquina não aceitou nenhuma das suas chaves (envie uma com enviar chave)",
		"novo":     "a chave do host ainda não foi aceita: conecte uma vez pelo terminal",
		"mudou":    "a chave do host mudou: confira antes de conectar",
		"longe":    "a máquina não respondeu a tempo",
		"fechado":  "a máquina recusou a conexão",
		"semnome":  "não achei esse nome",
		"estranho": "kex_exchange_identification: Connection closed by remote host",
		"quieto":   "o ssh saiu com o código 3",
		"preso":    "a máquina não respondeu a tempo",
	} {
		r := quer(t, "", "entrar", apelido)
		if r["entra"] != false || r["motivo"] != motivo {
			t.Errorf("%s: %v", apelido, r)
		}
	}

	antes := len(chamadas(t, bin, "ssh"))
	querErro(t, 1, "fantasma", "", "entrar", "fantasma")
	querErro(t, 2, "linha de comando", "", "entrar", "--", "-x")
	querErro(t, 2, "linha de comando", "", "entrar", "--", "a;b")
	querErro(t, 1, "uso:", "", "entrar")
	querErro(t, 1, "uso:", "", "entrar", "atlas", "true")
	querErro(t, 1, "não conheço a opção", "", "entrar", "-oProxyCommand=x")
	if len(chamadas(t, bin, "ssh")) != antes {
		t.Errorf("uma recusa não roda o ssh")
	}
}

func TestEntrar_SemOSshEErro(t *testing.T) {
	mundo(t, map[string]string{"config": "Host atlas\n"})
	querErro(t, 1, "ssh", "", "entrar", "atlas")
}

func TestMotivoDaEntrada_CortaOQueEComprido(t *testing.T) {
	m := motivoDaEntrada(strings.Repeat("é", 500)+"\n", 255, false)
	if n := len([]rune(m)); n != 161 || !strings.HasSuffix(m, "…") {
		t.Errorf("%d runas", n)
	}
	if m := motivoDaEntrada("a\x1b[31mb\x00\n", 255, false); m != "a[31mb" {
		t.Errorf("controle: %q", m)
	}
	if m := motivoDaEntrada("ssh: connect to host x port 22: No route to host\n", 255, false); m != "sem rota até lá" {
		t.Errorf("%q", m)
	}
}

// --- gerar chave ----------------------------------------------------------------

func TestComando_Gerar(t *testing.T) {
	dir, bin := mundo(t, map[string]string{"id_ed25519.pub": pubEd + "\n", "sozinha": "privada sem pública\n"})
	falso(t, bin, "ssh-keygen", "exit 9")
	falso(t, bin, "foot", "")
	r := quer(t, "", "comando", "gerar", "--", "trabalho", "ana no trabalho")
	if !reflect.DeepEqual(campos(r), []string{"acao", "apelido", "argv", "aviso", "terminal", "texto"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	if r["acao"] != "gerar" || r["apelido"] != nil || r["texto"] != nil {
		t.Errorf("%v", r)
	}
	argv := []string{"ssh-keygen", "-t", "ed25519", "-f", filepath.Join(dir, "trabalho"), "-C", "ana no trabalho"}
	if obtido := lista(r["argv"]); !reflect.DeepEqual(obtido, argv) {
		t.Errorf("argv: %q", obtido)
	}
	if term := lista(r["terminal"]); !reflect.DeepEqual(term, append([]string{"foot", "--title=ssh-keygen · trabalho", "--hold"}, argv...)) {
		t.Errorf("terminal: %q", term)
	}
	// Sem comentário: usuario@maquina daqui.
	argv = lista(quer(t, "", "comando", "gerar", "nova")["argv"])
	if len(argv) != 7 || !regexp.MustCompile(`^[^\s@]+@[^\s@]+$`).MatchString(argv[6]) {
		t.Errorf("comentário padrão: %q", argv)
	}
	t.Setenv("MKPORTO_KEYGEN", "/opt/teste/keygen")
	if argv := lista(quer(t, "", "comando", "gerar", "nova")["argv"]); argv[0] != "/opt/teste/keygen" {
		t.Errorf("argv: %q", argv)
	}

	for _, nome := range []string{"", "-x", ".oculta", "a b", "a/b", "../x", "chave.pub", "máquina", "a\nb", strings.Repeat("a", 65)} {
		codigo, linhas := roda(t, "", "comando", "gerar", "--", nome)
		if codigo != 1 || len(linhas) != 1 || linhas[0]["campo"] != "nome" {
			t.Errorf("nome %q: código %d, %v", nome, codigo, linhas)
		}
	}
	// Nunca por cima de uma chave que já existe (a pública ou a privada).
	for _, nome := range []string{"id_ed25519", "sozinha"} {
		linha := querErro(t, 1, "já existe", "", "comando", "gerar", nome)
		if linha["campo"] != "nome" {
			t.Errorf("%s: %v", nome, linha)
		}
	}
	for _, comentario := range []string{"-x", "a\nb", "a\rb", "a\x00b", "a\tb", strings.Repeat("é", 129)} {
		codigo, linhas := roda(t, "", "comando", "gerar", "--", "boa", comentario)
		if codigo != 1 || len(linhas) != 1 || linhas[0]["campo"] != "comentario" {
			t.Errorf("comentário %q: código %d, %v", comentario, codigo, linhas)
		}
	}
	quer(t, "", "comando", "gerar", "--", "boa", strings.Repeat("é", 128))
	querErro(t, 1, "uso:", "", "comando", "gerar")
	querErro(t, 1, "uso:", "", "comando", "gerar", "a", "b", "c")
	// O motor só disse o que abrir: nem o ssh-keygen nem o terminal rodaram.
	if chamadas(t, bin, "ssh-keygen") != nil || chamadas(t, bin, "foot") != nil {
		t.Errorf("o motor rodou alguma coisa")
	}
	itens, _ := os.ReadDir(dir)
	if len(itens) != 2 {
		t.Errorf("o motor criou arquivo: %v", itens)
	}
}

// --- caminhos com espaço --------------------------------------------------------

func TestComando_CaminhoComEspacoAspasECifraoChegaInteiro(t *testing.T) {
	_, bin := mundo(t, map[string]string{"config": "Host atlas\n"})
	// A pasta do ssh numa home com espaço, aspas simples, aspas duplas e $.
	dir := filepath.Join(t.TempDir(), `casa de 'ana' "a" $HOME`, ".ssh")
	t.Setenv("MKPORTO_SSH_DIR", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(filepath.Dir(dir), ".config"))
	escreve(t, filepath.Join(dir, "config"), "Host atlas\n")
	escreve(t, filepath.Join(dir, "minha chave $1.pub"), pubEd+"\n")
	for _, nome := range terminais {
		falso(t, bin, nome, "")
	}
	pub := filepath.Join(dir, "minha chave $1.pub")
	for _, nome := range terminais {
		t.Setenv("MKPORTO_TERMINAL", nome)
		r := quer(t, "", "comando", "chave", "--", "atlas", "minha chave $1.pub")
		argv := []string{"ssh-copy-id", "-i", pub, "--", "atlas"}
		if obtido := lista(r["argv"]); !reflect.DeepEqual(obtido, argv) {
			t.Errorf("%s, argv: %q", nome, obtido)
		}
		// No fim da linha do terminal está o argv, um argumento por item: nada
		// juntado numa linha, nada passando por shell.
		term := lista(r["terminal"])
		if term[0] != nome || !reflect.DeepEqual(term[len(term)-len(argv):], argv) {
			t.Errorf("%s, terminal: %q", nome, term)
		}
		for _, a := range term {
			if a == "sh" || a == "-c" || a == "bash" {
				t.Errorf("%s: a linha passa por shell: %q", nome, term)
			}
		}

		r = quer(t, "", "comando", "gerar", "--", "nova", "ana d'água $USER")
		gerar := []string{"ssh-keygen", "-t", "ed25519", "-f", filepath.Join(dir, "nova"), "-C", "ana d'água $USER"}
		term = lista(r["terminal"])
		if !reflect.DeepEqual(term[len(term)-len(gerar):], gerar) {
			t.Errorf("%s, gerar: %q", nome, term)
		}
	}
}
