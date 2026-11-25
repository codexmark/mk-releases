package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"mkporto/internal/base"
)

// mundo monta uma casa de mentira: HOME, o ~/.ssh com os arquivos dados e um
// PATH vazio (nenhum comando de verdade existe). Devolve a pasta do ssh e a
// pasta dos comandos falsos.
func mundo(t *testing.T, arquivos map[string]string) (dir, bin string) {
	t.Helper()
	casa := t.TempDir()
	dir = filepath.Join(casa, ".ssh")
	bin = filepath.Join(casa, "bin")
	for _, p := range []string{dir, bin} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", casa)
	t.Setenv("PATH", bin)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(casa, ".config"))
	t.Setenv("MKPORTO_SSH_DIR", dir)
	for _, v := range []string{"MKPORTO_SSH", "MKPORTO_COPY_ID", "MKPORTO_KEYGEN", "MKPORTO_TERMINAL"} {
		t.Setenv(v, "")
	}
	for nome, texto := range arquivos {
		escreve(t, filepath.Join(dir, nome), texto)
	}
	return dir, bin
}

func escreve(t *testing.T, caminho, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(texto), 0o600); err != nil {
		t.Fatal(err)
	}
}

func conteudo(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// falso põe um comando de mentira (um script de /bin/sh que só usa o que o
// shell tem dentro) na pasta dos comandos, e anota cada chamada.
func falso(t *testing.T, bin, nome, corpo string) string {
	t.Helper()
	caminho := filepath.Join(bin, nome)
	// A linha da chamada sai num printf só: dois dublês ao mesmo tempo não se misturam.
	script := "#!/bin/sh\nsep=$(printf '\\037'); chamada=; for a in \"$@\"; do chamada=\"$chamada$a$sep\"; done\n" +
		"printf '%s\\n' \"$chamada\" >>'" + caminho + ".chamadas'\n" + corpo + "\n"
	if err := os.WriteFile(caminho, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func chamadas(t *testing.T, bin, nome string) [][]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(bin, nome+".chamadas"))
	if err != nil {
		return nil
	}
	var out [][]string
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		out = append(out, strings.Split(strings.TrimSuffix(l, "\037"), "\037"))
	}
	return out
}

// roda executa um subcomando com a entrada dada e devolve o código e as
// linhas de JSON que ele escreveu.
func roda(t *testing.T, stdin string, args ...string) (int, []map[string]any) {
	t.Helper()
	var b bytes.Buffer
	antiga, antigaEntrada := base.Saida, entrada
	base.Saida, entrada = &b, strings.NewReader(stdin)
	defer func() { base.Saida, entrada = antiga, antigaEntrada }()
	codigo := rodar(args)
	var linhas []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		if l == "" {
			continue
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(l), &v); err != nil {
			t.Fatalf("linha que não é JSON: %q", l)
		}
		linhas = append(linhas, v)
	}
	return codigo, linhas
}

// quer roda e exige código 0 e uma linha só.
func quer(t *testing.T, stdin string, args ...string) map[string]any {
	t.Helper()
	codigo, linhas := roda(t, stdin, args...)
	if codigo != 0 || len(linhas) != 1 {
		t.Fatalf("%v: código %d, %v", args, codigo, linhas)
	}
	return linhas[0]
}

// querErro roda e exige o código dado e uma linha {"erro": …} com o trecho.
func querErro(t *testing.T, codigo int, trecho, stdin string, args ...string) map[string]any {
	t.Helper()
	obtido, linhas := roda(t, stdin, args...)
	if obtido != codigo || len(linhas) != 1 {
		t.Fatalf("%v: código %d (esperado %d), %v", args, obtido, codigo, linhas)
	}
	frase, _ := linhas[0]["erro"].(string)
	if frase == "" || !strings.Contains(frase, trecho) {
		t.Fatalf("%v: erro %q, esperado com %q", args, frase, trecho)
	}
	return linhas[0]
}

func campos(m map[string]any) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func hostDe(t *testing.T, resposta map[string]any, apelido string) map[string]any {
	t.Helper()
	for _, h := range resposta["hosts"].([]any) {
		if h.(map[string]any)["apelido"] == apelido {
			return h.(map[string]any)
		}
	}
	t.Fatalf("sem o host %q em %v", apelido, resposta["hosts"])
	return nil
}

func pedido(m map[string]any) string {
	b, _ := json.Marshal(m)
	return string(b)
}

const configDaPessoa = "# o meu config\nHost meu\n  HostName 192.0.2.5\n\tUser eu  \n\nHost *\n  ServerAliveInterval 30"

func TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON(t *testing.T) {
	for _, args := range [][]string{nil, {"banana"}} {
		linha := querErro(t, 1, "uso: mkporto-motor <", "", args...)
		for _, sub := range []string{"listar", "sondar", "guardar", "remover", "chaves", "comando"} {
			if !strings.Contains(linha["erro"].(string), sub) {
				t.Errorf("o uso não cita %s", sub)
			}
		}
	}
}

// --- listar -------------------------------------------------------------------

func TestListar_Contrato(t *testing.T) {
	dir, _ := mundo(t, map[string]string{
		"config":                "Include config.d/*.conf\nHost meu outro\n  User eu\nHost *.test web?\n  Port 2222\n",
		"config.d/mkporto.conf": "Host atlas\n  HostName atlas.oficina.example\n  User marina\n  Port 2200\n  IdentityFile \"~/.ssh/id_ed25519\"\n  ProxyJump meu\n",
	})
	r := quer(t, "", "listar")
	if !reflect.DeepEqual(campos(r), []string{"arquivo_do_app", "hosts", "pasta"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	if r["pasta"] != "~/.ssh" || r["arquivo_do_app"] != "~/.ssh/config.d/mkporto.conf" {
		t.Errorf("caminhos: %v %v", r["pasta"], r["arquivo_do_app"])
	}
	if n := len(r["hosts"].([]any)); n != 3 {
		t.Fatalf("hosts: %d", n)
	}
	atlas := hostDe(t, r, "atlas")
	esperado := map[string]any{"apelido": "atlas", "host": "atlas.oficina.example", "usuario": "marina", "porta": 2200.0,
		"chave": "~/.ssh/id_ed25519", "salto": "meu", "origem": "mkporto", "arquivo": "~/.ssh/config.d/mkporto.conf", "efetivo": false,
		"escrito": map[string]any{"host": "atlas.oficina.example", "usuario": "marina", "porta": 2200.0, "chave": "~/.ssh/id_ed25519", "salto": "meu"}}
	if !reflect.DeepEqual(atlas, esperado) {
		t.Errorf("atlas:\n%v\nesperado:\n%v", atlas, esperado)
	}
	meu := hostDe(t, r, "meu")
	esperado = map[string]any{"apelido": "meu", "host": "meu", "usuario": "eu", "porta": 22.0,
		"chave": nil, "salto": nil, "origem": "config", "arquivo": "~/.ssh/config", "efetivo": false, "escrito": nil}
	if !reflect.DeepEqual(meu, esperado) {
		t.Errorf("meu:\n%v\nesperado:\n%v", meu, esperado)
	}
	hostDe(t, r, "outro")
	_ = dir
}

func TestListar_SemConfigEUmaListaVazia(t *testing.T) {
	mundo(t, nil)
	r := quer(t, "", "listar")
	if hs, ok := r["hosts"].([]any); !ok || len(hs) != 0 {
		t.Errorf("hosts: %#v (tem de ser [], não null)", r["hosts"])
	}
	querErro(t, 1, "não conheço a opção", "", "listar", "--tudo")
}

// --- sondar -------------------------------------------------------------------

// escuta abre um listener em 127.0.0.1 numa porta livre e anota quantos
// bytes cada conexão mandou antes de fechar.
func escuta(t *testing.T, saudacao string) (porta int, recebido chan int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	recebido = make(chan int, 16)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				if saudacao != "" {
					_, _ = c.Write([]byte(saudacao))
				}
				_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
				n, _ := c.Read(make([]byte, 64))
				recebido <- n
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port, recebido
}

// portaFechada devolve uma porta de 127.0.0.1 em que ninguém escuta.
func portaFechada(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	porta := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return porta
}

func porApelido(linhas []map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, l := range linhas {
		out[l["apelido"].(string)] = l
	}
	return out
}

func TestSondar_Contrato(t *testing.T) {
	aberta, recebido := escuta(t, "SSH-2.0-Falso_1.0 de teste\r\n")
	fechada := portaFechada(t)
	mundo(t, map[string]string{"config": "Host vivo\n  HostName 127.0.0.1\n  Port " + strconv.Itoa(aberta) +
		"\nHost morto\n  HostName 127.0.0.1\n  Port " + strconv.Itoa(fechada) +
		"\nHost atras\n  HostName 192.0.2.77\n  ProxyJump vivo\n"})

	codigo, linhas := roda(t, "", "sondar", "--", "vivo", "morto", "atras", "fantasma")
	if codigo != 0 || len(linhas) != 4 {
		t.Fatalf("código %d, %v", codigo, linhas)
	}
	for _, l := range linhas {
		if !reflect.DeepEqual(campos(l), []string{"apelido", "motivo", "ms", "responde", "servidor", "via"}) {
			t.Errorf("campos: %v", campos(l))
		}
	}
	r := porApelido(linhas)
	if v := r["vivo"]; v["responde"] != true || v["motivo"] != nil || v["servidor"] != "Falso_1.0 de teste" || v["via"] != nil {
		t.Errorf("vivo: %v", v)
	} else if ms, ok := v["ms"].(float64); !ok || ms < 0 || ms > 2000 {
		t.Errorf("vivo, ms: %v", v["ms"])
	}
	if v := r["morto"]; v["responde"] != false || v["ms"] != nil || v["motivo"] != "recusou a conexão" {
		t.Errorf("morto: %v", v)
	}
	if v := r["atras"]; v["responde"] != nil || v["ms"] != nil || v["motivo"] != "por salto" {
		t.Errorf("atras: %v", v)
	}
	if v := r["fantasma"]; v["responde"] != nil || v["motivo"] != "não conheço esse apelido" {
		t.Errorf("fantasma: %v", v)
	}
	// Só abriu e fechou: nem uma letra do protocolo, nem tentativa de entrar.
	select {
	case n := <-recebido:
		if n != 0 {
			t.Errorf("a sonda mandou %d bytes; não devia falar nada", n)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("a sonda não fechou a conexão")
	}
}

func TestSondar_SemApelidoSondaTodos(t *testing.T) {
	aberta, _ := escuta(t, "SSH-2.0-Falso_1.0\r\n")
	mundo(t, map[string]string{"config": "Host a b\n  HostName 127.0.0.1\n  Port " + strconv.Itoa(aberta) + "\nHost *\n  User x\n"})
	codigo, linhas := roda(t, "", "sondar")
	if codigo != 0 || len(linhas) != 2 {
		t.Fatalf("código %d, %v", codigo, linhas)
	}
	r := porApelido(linhas)
	if r["a"]["responde"] != true || r["b"]["responde"] != true {
		t.Errorf("%v", linhas)
	}
}

func TestSondar_SemHostNenhumNaoDizNada(t *testing.T) {
	mundo(t, nil)
	if codigo, linhas := roda(t, "", "sondar"); codigo != 0 || len(linhas) != 0 {
		t.Errorf("código %d, %v", codigo, linhas)
	}
}

type erroDePrazo struct{}

func (erroDePrazo) Error() string   { return "i/o timeout" }
func (erroDePrazo) Timeout() bool   { return true }
func (erroDePrazo) Temporary() bool { return true }

func TestMotivo_AFraseDeCadaFalha(t *testing.T) {
	casos := map[string]error{
		"recusou a conexão":     &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)},
		"não respondeu em 2 s":  &net.OpError{Op: "dial", Err: erroDePrazo{}},
		"não achei esse nome":   &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host", Name: "x.invalid", IsNotFound: true}},
		"sem rota até lá":       &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.EHOSTUNREACH)},
		"não deu para conectar": errors.New("outra coisa"),
	}
	for quer, err := range casos {
		if obtido := motivo(err); obtido != quer {
			t.Errorf("motivo(%v) = %q; esperado %q", err, obtido, quer)
		}
	}
	if obtido := motivo(&net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.ENETUNREACH)}); obtido != "sem rota até lá" {
		t.Errorf("rede inalcançável: %q", obtido)
	}
}

// --- guardar ------------------------------------------------------------------

func atlas() map[string]any {
	return map[string]any{"apelido": "atlas", "host": "atlas.oficina.example", "usuario": "marina", "porta": 2200,
		"chave": "~/.ssh/id_ed25519", "salto": ""}
}

func TestGuardar_Contrato(t *testing.T) {
	dir, _ := mundo(t, nil)
	r := quer(t, pedido(atlas()), "guardar")
	esperado := map[string]any{"ok": true, "apelido": "atlas", "arquivo": "~/.ssh/config.d/mkporto.conf", "include": "posto"}
	if !reflect.DeepEqual(r, esperado) {
		t.Errorf("resposta: %v", r)
	}
	if texto := conteudo(t, filepath.Join(dir, "config")); texto != "Include config.d/mkporto.conf\n" {
		t.Errorf("config: %q", texto)
	}
	for caminho, quer := range map[string]os.FileMode{
		filepath.Join(dir, "config"):                   0o600,
		filepath.Join(dir, "config.d", "mkporto.conf"): 0o600,
		filepath.Join(dir, "config.d"):                 0o700,
	} {
		info, err := os.Stat(caminho)
		if err != nil || info.Mode().Perm() != quer {
			t.Errorf("%s: %v %v (esperado %o)", caminho, info.Mode().Perm(), err, quer)
		}
	}
	h := hostDe(t, quer(t, "", "listar"), "atlas")
	if h["origem"] != "mkporto" || h["host"] != "atlas.oficina.example" || h["usuario"] != "marina" || h["porta"] != 2200.0 ||
		h["chave"] != "~/.ssh/id_ed25519" || h["salto"] != nil {
		t.Errorf("listar depois de guardar: %v", h)
	}

	// O segundo host: o Include já está lá.
	r = quer(t, `{"apelido":"farol","host":"192.0.2.10"}`, "guardar")
	if r["include"] != "ja_havia" {
		t.Errorf("include: %v", r["include"])
	}
	if h := hostDe(t, quer(t, "", "listar"), "farol"); h["porta"] != 22.0 || h["usuario"] != nil {
		t.Errorf("farol (porta que falta é 22): %v", h)
	}
}

func TestGuardar_PreservaOConfigDaPessoaByteAByteENaoDuplicaOInclude(t *testing.T) {
	dir, _ := mundo(t, map[string]string{"config": configDaPessoa})
	quer(t, pedido(atlas()), "guardar")
	depois := conteudo(t, filepath.Join(dir, "config"))
	if depois != "Include config.d/mkporto.conf\n\n"+configDaPessoa {
		t.Fatalf("config:\n%q", depois)
	}
	for _, apelido := range []string{"b", "c"} {
		quer(t, `{"apelido":"`+apelido+`","host":"x.test"}`, "guardar")
	}
	quer(t, "", "remover", "b")
	if conteudo(t, filepath.Join(dir, "config")) != depois {
		t.Errorf("o config mudou de novo:\n%q", conteudo(t, filepath.Join(dir, "config")))
	}
	if n := strings.Count(conteudo(t, filepath.Join(dir, "config")), "Include"); n != 1 {
		t.Errorf("%d linhas de Include", n)
	}
}

func TestGuardar_ComUmIncludeQueJaCobreOConfigNaoETocado(t *testing.T) {
	antes := "Include config.d/*\n" + configDaPessoa
	dir, _ := mundo(t, map[string]string{"config": antes})
	if r := quer(t, pedido(atlas()), "guardar"); r["include"] != "ja_havia" {
		t.Errorf("include: %v", r["include"])
	}
	if conteudo(t, filepath.Join(dir, "config")) != antes {
		t.Errorf("o config mudou")
	}
	hostDe(t, quer(t, "", "listar"), "atlas")
}

func TestGuardar_RecusaInjecaoDeOpcaoEDeLinhaSemEscreverNada(t *testing.T) {
	dir, _ := mundo(t, map[string]string{"config": configDaPessoa})
	com := func(campo string, valor any) string {
		m := atlas()
		m[campo] = valor
		return pedido(m)
	}
	casos := []struct{ campo, stdin string }{
		{"apelido", com("apelido", "-oProxyCommand=touch pwn")},
		{"apelido", com("apelido", "-atlas")},
		{"apelido", com("apelido", "atlas\nHost *\n  ProxyCommand touch pwn")},
		{"apelido", com("apelido", "a b")},
		{"apelido", com("apelido", "web*")},
		{"apelido", com("apelido", "")},
		{"host", com("host", "-oProxyCommand=touch pwn")},
		{"host", com("host", "x.test\nProxyCommand touch pwn")},
		{"host", com("host", "x.test\r\nProxyCommand touch pwn")},
		{"host", com("host", "x test")},
		{"host", com("host", "")},
		{"usuario", com("usuario", "eu\nProxyCommand touch pwn")},
		{"usuario", com("usuario", "-l")},
		{"porta", com("porta", 0)},
		{"porta", com("porta", 70000)},
		{"chave", com("chave", "~/.ssh/k\nProxyCommand touch pwn")},
		{"chave", com("chave", `~/.ssh/k" ProxyCommand "x`)},
		{"salto", com("salto", "-oProxyCommand=x")},
		{"salto", com("salto", "a\nb")},
	}
	for _, c := range casos {
		codigo, linhas := roda(t, c.stdin, "guardar")
		if codigo != 1 || len(linhas) != 1 || linhas[0]["campo"] != c.campo || linhas[0]["erro"] == nil {
			t.Errorf("%s: código %d, %v", c.stdin, codigo, linhas)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "config.d")); err == nil {
		t.Errorf("uma recusa não cria nada")
	}
	if conteudo(t, filepath.Join(dir, "config")) != configDaPessoa {
		t.Errorf("uma recusa não toca no config")
	}
}

func TestGuardar_EntradaQueNaoEOPedidoEErro(t *testing.T) {
	dir, _ := mundo(t, nil)
	for _, stdin := range []string{"", "banana", "[]", `{"apelido":"a","host":"x.test","porta":"22"}`,
		`{"apelido":"a","host":"x.test","porta":22.5}`, `{"apelido":"a","host":"x.test","comando":"rm"}`,
		`{"apelido":"a","host":"x.test"} {"apelido":"b","host":"y.test"}`, `{"apelido":["a"],"host":"x.test"}`} {
		querErro(t, 1, "pedido", stdin, "guardar")
	}
	if _, err := os.Stat(filepath.Join(dir, "config")); err == nil {
		t.Errorf("um pedido torto não cria nada")
	}
}

func TestGuardar_ApelidoQueJaEstaNoConfigDaPessoaERecusado(t *testing.T) {
	arquivos := map[string]string{
		"config":           "Include outros/*\n" + configDaPessoa,
		"outros/trab.conf": "Host trabalho vpn\n  HostName 192.0.2.30\n",
	}
	dir, _ := mundo(t, arquivos)
	for _, apelido := range []string{"meu", "trabalho", "vpn"} {
		m := atlas()
		m["apelido"] = apelido
		linha := querErro(t, 2, "~/.ssh/", pedido(m), "guardar")
		if !strings.Contains(linha["erro"].(string), apelido) {
			t.Errorf("a frase não diz qual apelido: %v", linha["erro"])
		}
	}
	for nome, antes := range arquivos {
		if conteudo(t, filepath.Join(dir, nome)) != antes {
			t.Errorf("%s mudou", nome)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "config.d")); err == nil {
		t.Errorf("uma recusa não cria nada")
	}
}

func TestGuardar_OMesmoApelidoSubstituiOHostNoLugar(t *testing.T) {
	dir, _ := mundo(t, nil)
	for _, apelido := range []string{"a", "atlas", "z"} {
		m := atlas()
		m["apelido"] = apelido
		quer(t, pedido(m), "guardar")
	}
	m := atlas()
	m["host"], m["usuario"], m["chave"] = "192.0.2.99", "", ""
	quer(t, pedido(m), "guardar")
	r := quer(t, "", "listar")
	var ordem []string
	for _, h := range r["hosts"].([]any) {
		ordem = append(ordem, h.(map[string]any)["apelido"].(string))
	}
	if !reflect.DeepEqual(ordem, []string{"a", "atlas", "z"}) {
		t.Errorf("ordem: %v", ordem)
	}
	if h := hostDe(t, r, "atlas"); h["host"] != "192.0.2.99" || h["usuario"] != nil || h["chave"] != nil {
		t.Errorf("atlas: %v", h)
	}
	if n := strings.Count(conteudo(t, filepath.Join(dir, "config.d", "mkporto.conf")), "Host atlas"); n != 1 {
		t.Errorf("%d blocos do atlas", n)
	}
}

func TestGuardar_EdicaoComTrocaDeApelido(t *testing.T) {
	mundo(t, map[string]string{"config": configDaPessoa})
	for _, apelido := range []string{"a", "atlas", "z"} {
		m := atlas()
		m["apelido"] = apelido
		quer(t, pedido(m), "guardar")
	}
	m := atlas()
	m["apelido"], m["anterior"] = "titan", "atlas"
	if r := quer(t, pedido(m), "guardar"); r["apelido"] != "titan" {
		t.Errorf("resposta: %v", r)
	}
	var ordem []string
	for _, h := range quer(t, "", "listar")["hosts"].([]any) {
		ordem = append(ordem, h.(map[string]any)["apelido"].(string))
	}
	if !reflect.DeepEqual(ordem, []string{"a", "titan", "z", "meu"}) {
		t.Errorf("o novo apelido fica no lugar do antigo: %v", ordem)
	}

	// Trocar para um apelido que já é de outro host do app: erro, nada muda.
	m["apelido"], m["anterior"] = "z", "titan"
	querErro(t, 1, "já existe", pedido(m), "guardar")
	// O anterior que não existe no app.
	m["apelido"], m["anterior"] = "novo", "fantasma"
	querErro(t, 1, "fantasma", pedido(m), "guardar")
	// O anterior que é da pessoa: recusa.
	m["apelido"], m["anterior"] = "novo", "meu"
	querErro(t, 2, "~/.ssh/config", pedido(m), "guardar")
	// O anterior também é validado: não vira nada estranho.
	m["apelido"], m["anterior"] = "novo", "-x\ny"
	querErro(t, 1, "apelido", pedido(m), "guardar")
	// Anterior igual ao apelido é a edição sem troca.
	m["apelido"], m["anterior"], m["host"] = "titan", "titan", "192.0.2.50"
	quer(t, pedido(m), "guardar")
	if h := hostDe(t, quer(t, "", "listar"), "titan"); h["host"] != "192.0.2.50" {
		t.Errorf("titan: %v", h)
	}
}

// --- remover ------------------------------------------------------------------

func TestRemover_Contrato(t *testing.T) {
	dir, _ := mundo(t, map[string]string{"config": configDaPessoa})
	quer(t, pedido(atlas()), "guardar")
	quer(t, `{"apelido":"farol","host":"192.0.2.10"}`, "guardar")
	r := quer(t, "", "remover", "--", "atlas")
	if !reflect.DeepEqual(r, map[string]any{"ok": true, "apelido": "atlas"}) {
		t.Errorf("resposta: %v", r)
	}
	app := conteudo(t, filepath.Join(dir, "config.d", "mkporto.conf"))
	if strings.Contains(app, "atlas") || !strings.Contains(app, "Host farol") {
		t.Errorf("arquivo do app:\n%s", app)
	}
	info, _ := os.Stat(filepath.Join(dir, "config.d", "mkporto.conf"))
	if info.Mode().Perm() != 0o600 {
		t.Errorf("modo: %o", info.Mode().Perm())
	}
	querErro(t, 1, "atlas", "", "remover", "atlas")
}

func TestRemover_RecusaOQueNaoEDoApp(t *testing.T) {
	dir, _ := mundo(t, map[string]string{"config": configDaPessoa})
	quer(t, pedido(atlas()), "guardar")
	config := conteudo(t, filepath.Join(dir, "config"))
	app := conteudo(t, filepath.Join(dir, "config.d", "mkporto.conf"))

	querErro(t, 2, "é do seu ~/.ssh/config", "", "remover", "meu")
	querErro(t, 1, "fantasma", "", "remover", "fantasma")
	querErro(t, 1, "uso:", "", "remover")
	querErro(t, 1, "uso:", "", "remover", "a", "b")
	querErro(t, 1, "não conheço a opção", "", "remover", "-x")

	if conteudo(t, filepath.Join(dir, "config")) != config || conteudo(t, filepath.Join(dir, "config.d", "mkporto.conf")) != app {
		t.Errorf("uma recusa não muda arquivo nenhum")
	}
}

// --- chaves -------------------------------------------------------------------

// Chaves públicas de mentira: o conteúdo não é uma chave de verdade, só tem a
// forma (tipo, base64, comentário).
const (
	pubEd  = "ssh-ed25519 QUFBQWV4ZW1wbG8tZWQyNTUxOQ== marina@oficina.example"
	pubRsa = "ssh-rsa QUFBQWV4ZW1wbG8tcnNh chave de trabalho"
)

func TestChaves_Contrato(t *testing.T) {
	dir, bin := mundo(t, map[string]string{
		"id_ed25519.pub":      pubEd + "\n",
		"trabalho.pub":        pubRsa + "\n",
		"id_ed25519":          "-----BEGIN OPENSSH PRIVATE KEY-----\nsegredo\n",
		"id_ed25519-cert.pub": "ssh-ed25519-cert-v01@openssh.com QUFBQQ== cert\n",
		"vazio.pub":           "",
		"lixo.pub":            "isto não é uma chave\n",
		"known_hosts":         "x.test ssh-ed25519 QUFBQQ==\n",
		"sub/outra.pub":       pubEd + "\n",
	})
	keygen := falso(t, bin, "ssh-keygen", `case "$3" in
  */id_ed25519.pub) printf '256 SHA256:exemploDeImpressaoEd marina@oficina.example (ED25519)\n' ;;
  */trabalho.pub) printf '4096 SHA256:exemploDeImpressaoRsa chave de trabalho (RSA)\n' ;;
  *) exit 1 ;;
esac`)
	t.Setenv("MKPORTO_KEYGEN", keygen)

	r := quer(t, "", "chaves")
	chaves := r["chaves"].([]any)
	if len(chaves) != 2 {
		t.Fatalf("chaves: %v", chaves)
	}
	esperado := map[string]any{"nome": "id_ed25519.pub", "arquivo": "~/.ssh/id_ed25519.pub", "tipo": "ed25519", "bits": 256.0,
		"impressao": "SHA256:exemploDeImpressaoEd", "comentario": "marina@oficina.example", "publica": pubEd}
	if !reflect.DeepEqual(chaves[0], esperado) {
		t.Errorf("ed25519:\n%v\nesperado:\n%v", chaves[0], esperado)
	}
	rsa := chaves[1].(map[string]any)
	if rsa["tipo"] != "rsa" || rsa["bits"] != 4096.0 || rsa["comentario"] != "chave de trabalho" || rsa["publica"] != pubRsa {
		t.Errorf("rsa: %v", rsa)
	}

	// O ssh-keygen só viu arquivos .pub, com -l -f e o caminho inteiro.
	for _, c := range chamadas(t, bin, "ssh-keygen") {
		if len(c) != 3 || c[0] != "-l" || c[1] != "-f" || !strings.HasSuffix(c[2], ".pub") || filepath.Dir(c[2]) != dir {
			t.Errorf("chamada ao ssh-keygen: %q", c)
		}
	}
	if len(chamadas(t, bin, "ssh-keygen")) != 2 {
		t.Errorf("chamadas: %v", chamadas(t, bin, "ssh-keygen"))
	}
}

func TestChaves_SemOSshKeygenAImpressaoECalculadaAqui(t *testing.T) {
	mundo(t, map[string]string{"id_ed25519.pub": pubEd + "\n"})
	r := quer(t, "", "chaves")
	c := r["chaves"].([]any)[0].(map[string]any)
	// SHA256 de "AAAAexemplo-ed25519", em base64 sem o "=" do fim.
	if c["impressao"] != "SHA256:UJlja5834xQgYKkPPzTC3KIi8kbZUtxRBdIgBTR2efs" {
		t.Errorf("impressão: %v", c["impressao"])
	}
	if c["bits"] != nil || c["tipo"] != "ed25519" {
		t.Errorf("%v", c)
	}
}

func TestChaves_SemPastaOuSemChaveEUmaListaVazia(t *testing.T) {
	dir, _ := mundo(t, nil)
	if c, ok := quer(t, "", "chaves")["chaves"].([]any); !ok || len(c) != 0 {
		t.Errorf("chaves: %#v", c)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if c, ok := quer(t, "", "chaves")["chaves"].([]any); !ok || len(c) != 0 {
		t.Errorf("sem a pasta: %#v", c)
	}
}

func TestTipoDaChave(t *testing.T) {
	for algoritmo, quer := range map[string]string{
		"ssh-ed25519":                        "ed25519",
		"ssh-rsa":                            "rsa",
		"ssh-dss":                            "dsa",
		"ecdsa-sha2-nistp256":                "ecdsa",
		"ecdsa-sha2-nistp521":                "ecdsa",
		"sk-ssh-ed25519@openssh.com":         "ed25519-sk",
		"sk-ecdsa-sha2-nistp256@openssh.com": "ecdsa-sk",
		"ssh-novidade":                       "ssh-novidade",
	} {
		if obtido := tipoDaChave(algoritmo); obtido != quer {
			t.Errorf("tipoDaChave(%q) = %q", algoritmo, obtido)
		}
	}
}

// --- comando ------------------------------------------------------------------

func lista(v any) []string {
	out := []string{}
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestComando_Contrato(t *testing.T) {
	dir, bin := mundo(t, map[string]string{
		"config":         "Host atlas\n  HostName atlas.oficina.example\n",
		"id_ed25519.pub": pubEd + "\n",
	})
	falso(t, bin, "foot", "")
	r := quer(t, "", "comando", "conectar", "--", "atlas")
	if !reflect.DeepEqual(campos(r), []string{"acao", "apelido", "argv", "aviso", "terminal", "texto"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	if r["acao"] != "conectar" || r["apelido"] != "atlas" || r["texto"] != "ssh atlas" || r["aviso"] != nil {
		t.Errorf("%v", r)
	}
	if argv := lista(r["argv"]); !reflect.DeepEqual(argv, []string{"ssh", "--", "atlas"}) {
		t.Errorf("argv: %q", argv)
	}
	if term := lista(r["terminal"]); !reflect.DeepEqual(term, []string{"foot", "--title=ssh · atlas", "ssh", "--", "atlas"}) {
		t.Errorf("terminal: %q", term)
	}

	r = quer(t, "", "comando", "chave", "--", "atlas", "id_ed25519.pub")
	pub := filepath.Join(dir, "id_ed25519.pub")
	if argv := lista(r["argv"]); !reflect.DeepEqual(argv, []string{"ssh-copy-id", "-i", pub, "--", "atlas"}) {
		t.Errorf("argv: %q", argv)
	}
	if term := lista(r["terminal"]); !reflect.DeepEqual(term, []string{"foot", "--title=ssh · atlas", "--hold", "ssh-copy-id", "-i", pub, "--", "atlas"}) {
		t.Errorf("terminal: %q", term)
	}
	if r["acao"] != "chave" {
		t.Errorf("%v", r)
	}

	// Nenhum comando rodou: o motor só diz o que abrir.
	if c := chamadas(t, bin, "foot"); c != nil {
		t.Errorf("o motor abriu o terminal: %v", c)
	}
}

func TestComando_OsComandosSaoTrocaveis(t *testing.T) {
	mundo(t, map[string]string{"config": "Host atlas\n", "k.pub": pubEd + "\n"})
	t.Setenv("MKPORTO_SSH", "/opt/teste/ssh")
	t.Setenv("MKPORTO_COPY_ID", "/opt/teste/copia")
	if argv := lista(quer(t, "", "comando", "conectar", "atlas")["argv"]); argv[0] != "/opt/teste/ssh" {
		t.Errorf("argv: %q", argv)
	}
	if argv := lista(quer(t, "", "comando", "chave", "atlas", "k.pub")["argv"]); argv[0] != "/opt/teste/copia" {
		t.Errorf("argv: %q", argv)
	}
}

func TestComando_SemTerminalOTextoContinuaServindo(t *testing.T) {
	mundo(t, map[string]string{"config": "Host atlas\n"})
	r := quer(t, "", "comando", "conectar", "atlas")
	if r["terminal"] != nil || r["texto"] != "ssh atlas" {
		t.Errorf("%v", r)
	}
	if aviso, _ := r["aviso"].(string); !strings.Contains(aviso, "terminal") {
		t.Errorf("aviso: %v", r["aviso"])
	}
}

func TestComando_OTerminalEOPedidoOuOPrimeiroQueHouver(t *testing.T) {
	_, bin := mundo(t, map[string]string{"config": "Host atlas\n"})
	for _, nome := range []string{"alacritty", "kitty", "wezterm", "ghostty"} {
		falso(t, bin, nome, "")
	}
	primeiro := func() []string {
		return lista(quer(t, "", "comando", "conectar", "atlas")["terminal"])
	}
	if term := primeiro(); term[0] != "ghostty" || !reflect.DeepEqual(term[len(term)-4:], []string{"-e", "ssh", "--", "atlas"}) {
		t.Errorf("sem pedido, o primeiro da lista: %q", term)
	}
	t.Setenv("MKPORTO_TERMINAL", "alacritty")
	if term := primeiro(); !reflect.DeepEqual(term, []string{"alacritty", "--title", "ssh · atlas", "-e", "ssh", "--", "atlas"}) {
		t.Errorf("alacritty: %q", term)
	}
	t.Setenv("MKPORTO_TERMINAL", "wezterm")
	if term := primeiro(); !reflect.DeepEqual(term, []string{"wezterm", "start", "--always-new-process", "--", "ssh", "--", "atlas"}) {
		t.Errorf("wezterm: %q", term)
	}
	// Um terminal que o Mk não conhece não entra: nada de rodar o que vier na variável.
	t.Setenv("MKPORTO_TERMINAL", "/tmp/qualquer-coisa")
	if term := primeiro(); term[0] != "ghostty" {
		t.Errorf("terminal desconhecido: %q", term)
	}
	// O dos Ajustes do Mk.
	t.Setenv("MKPORTO_TERMINAL", "")
	escreve(t, base.Config("mk", "ajustes.json"), `{"apps.terminal":"kitty"}`)
	if term := primeiro(); !reflect.DeepEqual(term, []string{"kitty", "--title=ssh · atlas", "ssh", "--", "atlas"}) {
		t.Errorf("dos ajustes: %q", term)
	}
}

func TestComando_RecusaOQueNaoPodeIrParaUmaLinhaDeComando(t *testing.T) {
	_, bin := mundo(t, map[string]string{
		"config": "Host atlas\nHost -oProxyCommand=x\nHost a;b\nHost $(x)\n",
		"k.pub":  pubEd + "\n",
		"-x.pub": pubEd + "\n",
	})
	falso(t, bin, "foot", "")
	// Do config da pessoa, mas com cara de opção ou de shell: recusa.
	for _, apelido := range []string{"-oProxyCommand=x", "a;b", "$(x)"} {
		querErro(t, 2, "linha de comando", "", "comando", "conectar", "--", apelido)
		querErro(t, 2, "linha de comando", "", "comando", "chave", "--", apelido, "k.pub")
	}
	querErro(t, 1, "fantasma", "", "comando", "conectar", "fantasma")
	querErro(t, 1, "uso:", "", "comando")
	querErro(t, 1, "uso:", "", "comando", "apagar", "atlas")
	querErro(t, 1, "uso:", "", "comando", "conectar")
	querErro(t, 1, "uso:", "", "comando", "conectar", "atlas", "sobra")
	querErro(t, 1, "uso:", "", "comando", "chave", "atlas")
	querErro(t, 1, "não conheço a opção", "", "comando", "conectar", "-oProxyCommand=x")
	// A chave tem de ser uma das listadas, pelo nome: nada de caminho.
	for _, nome := range []string{"nao-existe.pub", "../k.pub", "/etc/passwd", "config", "id_ed25519", "sub/k.pub"} {
		querErro(t, 1, "chave", "", "comando", "chave", "--", "atlas", nome)
	}
	// Um .pub cujo nome começa com "-" vai como caminho inteiro: não vira opção.
	argv := lista(quer(t, "", "comando", "chave", "--", "atlas", "-x.pub")["argv"])
	if !filepath.IsAbs(argv[2]) || filepath.Base(argv[2]) != "-x.pub" {
		t.Errorf("argv: %q", argv)
	}
}
