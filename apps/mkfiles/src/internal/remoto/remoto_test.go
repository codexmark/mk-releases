package remoto

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/prova"
)

// motor monta a caixa de areia com a pasta do gvfs de mentira e um gio que
// não existe (cada teste põe o seu).
func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	t.Setenv("MKFILES_GVFS", m.Pasta("gvfs"))
	t.Setenv("MKFILES_GIO", m.Casa+"/nao-ha")
	t.Setenv("MKFILES_SSH_DIR", m.Pasta(".ssh"))
	t.Setenv("MKFILES_REMOTO_MS", "3000")
	return m
}

// gio escreve um gio de mentira: um script que anota os argumentos e o
// ambiente em gio.log e faz o que o corpo mandar.
func gio(t *testing.T, m *prova.Motor, corpo string) {
	t.Helper()
	caminho := filepath.Join(m.Casa, "gio-de-mentira")
	script := "#!/bin/sh\n{ echo \"ARGS: $*\"; env; } >>\"" + m.Casa + "/gio.log\"\nG=\"$MKFILES_GVFS\"\n" + corpo + "\n"
	if err := os.WriteFile(caminho, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKFILES_GIO", caminho)
}

func diario(m *prova.Motor) string {
	b, _ := os.ReadFile(m.Casa + "/gio.log")
	return string(b)
}

func TestTradutor(t *testing.T) {
	casos := []struct{ uri, pasta, dentro, volta string }{
		{"sftp://ana@Servidor.local:2222/home/ana", "sftp:host=servidor.local,port=2222,user=ana", "/home/ana", "sftp://ana@servidor.local:2222/"},
		{"sftp://nas/", "sftp:host=nas", "", "sftp://nas/"},
		{"smb://nas/Meus%20Arquivos/sub/x", "smb-share:server=nas,share=meus%20arquivos", "/sub/x", "smb://nas/meus%20arquivos"},
		{"smb://CASA;ana@nas/docs", "smb-share:domain=CASA,server=nas,share=docs,user=ana", "", "smb://CASA;ana@nas/docs"},
		{"ftp://ftp.exemplo.org/pub", "ftp:host=ftp.exemplo.org", "/pub", "ftp://ftp.exemplo.org/"},
		{"davs://nuvem.exemplo.org/remote.php/dav", "dav:host=nuvem.exemplo.org,ssl=true,prefix=%2Fremote.php%2Fdav", "", "davs://nuvem.exemplo.org/remote.php/dav"},
		{"dav://ana@caixa:8080/", "dav:host=caixa,port=8080,ssl=false,user=ana", "", "dav://ana@caixa:8080/"},
		{"mtp://%5Busb%3A001,004%5D/", "mtp:host=[usb:001%2C004]", "", "mtp://%5Busb:001%2C004%5D/"},
	}
	for _, c := range casos {
		e, err := Abre(c.uri)
		if err != nil {
			t.Errorf("Abre(%q): %v", c.uri, err)
			continue
		}
		pasta, dentro := e.Pasta()
		if pasta != c.pasta || dentro != c.dentro {
			t.Errorf("%s → pasta %q, dentro %q; esperado %q, %q", c.uri, pasta, dentro, c.pasta, c.dentro)
		}
		// A pasta que nós mesmos escrevemos casa com o endereço, e o que sobra é o "dentro".
		if d, casa := e.Casa(c.pasta); !casa || d != c.dentro {
			t.Errorf("%s não casa com %q (dentro %q)", c.uri, c.pasta, d)
		}
		da, ok := DaPasta(c.pasta)
		if !ok {
			t.Errorf("DaPasta(%q) não entendeu", c.pasta)
			continue
		}
		if c.volta != "" && da.URI() != c.volta {
			t.Errorf("DaPasta(%q).URI() = %q; esperado %q", c.pasta, da.URI(), c.volta)
		}
	}
	// A ordem das chaves do gvfs de verdade pode ser outra, e pode haver chaves a mais.
	e, _ := Abre("sftp://ana@servidor/docs")
	if d, casa := e.Casa("sftp:user=ana,host=servidor,extra=1"); !casa || d != "/docs" {
		t.Errorf("outra ordem de chaves: %q %v", d, casa)
	}
	// Sem usuário no endereço, casa com a pasta de qualquer usuário; com outro, não.
	e, _ = Abre("sftp://servidor/")
	if _, casa := e.Casa("sftp:host=servidor,user=bia"); !casa {
		t.Error("endereço sem usuário casa com a pasta que tem")
	}
	e, _ = Abre("sftp://ana@servidor/")
	for _, outra := range []string{"sftp:host=servidor,user=bia", "sftp:host=outro,user=ana", "ftp:host=servidor,user=ana", "sftp:host=servidor,user=ana,port=2222x", "estranho"} {
		if _, casa := e.Casa(outra); casa && outra != "sftp:host=servidor,user=ana,port=2222x" {
			t.Errorf("não devia casar com %q", outra)
		}
	}
	if _, ok := DaPasta("google-drive:host=gmail.com,user=ana"); ok {
		t.Error("protocolo desconhecido não vira endereço")
	}
}

func TestEnderecosRecusados(t *testing.T) {
	for _, ruim := range []string{"", "servidor", "/home/ana", "file:///etc", "http://x/", "--help", "-u sftp://x/", "sftp://", "sftp:///pasta",
		"sftp://ana:segredo@servidor/", "sftp://servidor/a\nb", "sftp://servi\x00dor/", "smb://\x1b[31mx/"} {
		if _, err := Abre(ruim); err == nil {
			t.Errorf("Abre(%q) devia recusar", ruim)
		}
	}
	// A recusa da senha no endereço não repete a senha.
	if _, err := Abre("sftp://ana:segredo@servidor/"); strings.Contains(err.Error(), "segredo") {
		t.Error("o erro não pode trazer a senha")
	}
}

func TestClassifica(t *testing.T) {
	casos := []struct {
		texto, tipo, padrao string
		pergunta            bool
	}{
		{"User [ana]: ", pUsuario, "ana", true},
		{"Password: ", pSenha, "", true},
		{"Domain [WORKGROUP]: ", pDominio, "WORKGROUP", true},
		{"Usuário [ana]: ", pUsuario, "ana", true},
		{"Senha: ", pSenha, "", true},
		{"Domínio [CASA]: ", pDominio, "CASA", true},
		{"Enter password for user ana on servidor: ", pSenha, "", true},
		{"Palavra-passe:", pSenha, "", true},
		{"Authentication Required\nEnter user and password for share on nas:\nUser [ana]: ", pUsuario, "ana", true},
		{"Can't verify the identity of “servidor”.\n[1] Log In Anyway\n[2] Cancel Login\nChoice: ", pConfirmar, "", true},
		{"Não foi possível verificar a identidade.\n[1] Continuar\n[2] Cancelar\nEscolha: ", pConfirmar, "", true},
		{"Montando…\n", "", "", false},
		{"Connecting to server", "", "", false},
		{"", "", "", false},
	}
	for _, c := range casos {
		tipo, texto, padrao, ok := classifica(c.texto, false)
		if ok != c.pergunta || tipo != c.tipo || padrao != c.padrao {
			t.Errorf("classifica(%q) = %q, padrão %q, %v; esperado %q, %q, %v", c.texto, tipo, padrao, ok, c.tipo, c.padrao, c.pergunta)
		}
		if c.tipo == pConfirmar && !strings.Contains(texto, "[1]") {
			t.Errorf("a confirmação leva as escolhas: %q", texto)
		}
	}
	// Uma pergunta que o motor não conhece, parada sem quebra de linha: "outro".
	if tipo, texto, _, ok := classifica("Algo estranho\nCódigo do autenticador: ", true); !ok || tipo != pOutro || !strings.Contains(texto, "Algo estranho") {
		t.Errorf("pergunta desconhecida: %q %q %v", tipo, texto, ok)
	}
	if _, _, _, ok := classifica("texto solto sem dois pontos", true); ok {
		t.Error("um resto qualquer não é pergunta")
	}
}

func TestHostsDoSsh(t *testing.T) {
	m := motor(t)
	m.Grava(".ssh/config", `# comentário
Include config.d/*.conf "outro arquivo"
Include /nao/existe

Host casa  nas-casa
    HostName 192.168.0.10
    User ana
    Port 2222
    User ignorado

Host *.interno !ruim
    User todos

Host=trabalho
    HostName=srv.exemplo.org

Match host x
    User nao-conta

Host -torto
Host casa
    User repetido
`)
	m.Grava(".ssh/config.d/a.conf", "Host incluido\n  HostName 10.0.0.1\n  Include ../config\n")
	m.Grava(".ssh/outro arquivo", "Host \"com aspas\"\n")
	m.Grava(".ssh/id_ed25519", "CHAVE PRIVADA: nunca lida")
	r := m.Pede(`{"op":"remoto-hosts"}`)
	var linhas []string
	for _, h := range r["hosts"].([]any) {
		c := h.(map[string]any)
		linhas = append(linhas, fmt.Sprintf("%v|%v|%v|%v", c["apelido"], c["host"], c["usuario"], c["porta"]))
	}
	esperado := "incluido|10.0.0.1|<nil>|<nil> com aspas|<nil>|<nil>|<nil> casa|192.168.0.10|ana|2222 nas-casa|192.168.0.10|ana|2222 trabalho|srv.exemplo.org|<nil>|<nil>"
	if strings.Join(linhas, " ") != esperado {
		t.Errorf("hosts:\n%s\nesperado:\n%s", strings.Join(linhas, " "), esperado)
	}
	// Sem pasta do ssh, lista vazia.
	t.Setenv("MKFILES_SSH_DIR", m.Casa+"/nao-ha")
	if r := m.Pede(`{"op":"remoto-hosts"}`); len(r["hosts"].([]any)) != 0 {
		t.Errorf("sem config: %v", r)
	}
}

func TestMontarSemPerguntas(t *testing.T) {
	m := motor(t)
	gio(t, m, `mkdir -p "$G/sftp:host=servidor,user=ana/home/ana"; echo ok`)
	r := m.Pede(`{"op":"remoto-montar","uri":"sftp://ana@servidor/home/ana","anonimo":false}`)
	raiz := m.Casa + "/gvfs/sftp:host=servidor,user=ana"
	if r["erro"] != nil || r["raiz"] != raiz || r["caminho"] != raiz+"/home/ana" || r["nome"] != "servidor" || r["uri"] != "sftp://ana@servidor/home/ana" {
		t.Fatalf("montar: %v", r)
	}
	if !strings.Contains(diario(m), "ARGS: mount sftp://ana@servidor/home/ana\n") {
		t.Errorf("o gio recebeu: %s", diario(m))
	}
	if e := m.Evento("remotos", nil); len(e["remotos"].([]any)) != 1 {
		t.Errorf("evento remotos: %v", e)
	}
	// Já montado: responde o lugar sem chamar o gio de novo; pasta que não
	// existe lá dentro cai na raiz.
	os.Remove(m.Casa + "/gio.log")
	r = m.Pede(`{"op":"remoto-montar","uri":"sftp://ana@servidor/nao/ha"}`)
	if r["caminho"] != raiz || diario(m) != "" {
		t.Errorf("já montado: %v (gio: %q)", r, diario(m))
	}
	// A lista dos montados e o desmontar.
	l := m.Pede(`{"op":"remotos","rede":false}`)
	um := l["remotos"].([]any)[0].(map[string]any)
	if um["uri"] != "sftp://ana@servidor/" || um["raiz"] != raiz || um["nome"] != "servidor" || um["protocolo"] != "sftp" || um["host"] != "servidor" || l["rede"] != nil {
		t.Errorf("remotos: %v", l)
	}
	gio(t, m, `[ "$1 $2" = "mount -u" ] && rmdir -p "$G/sftp:host=servidor,user=ana/home/ana" 2>/dev/null; rm -rf "${G:?}/sftp:host=servidor,user=ana"`)
	if r := m.Pede(`{"op":"remoto-desmontar","uri":"sftp://outro/"}`); r["erro"] == nil {
		t.Errorf("desmontar o que não está montado: %v", r)
	}
	r = m.Pede(`{"op":"remoto-desmontar","uri":"sftp://ana@servidor/"}`)
	if r["erro"] != nil || len(r["remotos"].([]any)) != 0 || !strings.Contains(diario(m), "ARGS: mount -u sftp://ana@servidor/\n") {
		t.Errorf("desmontar: %v\n%s", r, diario(m))
	}
}

// conversa manda montar e responde às perguntas na ordem dada; devolve as
// perguntas que vieram e a resposta final.
func conversa(t *testing.T, m *prova.Motor, uri string, respostas ...any) ([]prova.Linha, prova.Linha) {
	t.Helper()
	id := m.Manda(`{"op":"remoto-montar","uri":` + prova.J(uri) + `}`)
	var perguntas []prova.Linha
	for _, resposta := range respostas {
		e := m.Evento("pergunta-remota", nil)
		perguntas = append(perguntas, e)
		if e["pedido"] != float64(id) {
			t.Fatalf("a pergunta diz de que pedido é: %v", e)
		}
		r := m.Pede(fmt.Sprintf(`{"op":"remoto-responder","pedido":%d,"pergunta":%v,"resposta":%s}`, id, e["pergunta"], prova.J(resposta)))
		if r["respondida"] != true {
			t.Fatalf("responder: %v", r)
		}
	}
	return perguntas, m.Resposta(id)
}

const segredo = "S3nh4-muito_secreta"

func TestMontarComUsuarioESenha(t *testing.T) {
	m := motor(t)
	gio(t, m, `printf 'Authentication Required\nEnter user and password for share on nas:\n'
printf 'User [%s]: ' "$USER"; read u
printf 'Domain [WORKGROUP]: '; read d
printf 'Password: '; read s
echo "LIDO: $u|$d|$s" >>"`+m.Casa+`/lido.log"
[ "$s" = "`+segredo+`" ] || { echo "erro: senha recusada"; exit 2; }
mkdir -p "$G/smb-share:server=nas,share=docs,user=$u"`)
	perguntas, r := conversa(t, m, "smb://nas/docs", "ana", "", segredo)
	tipos := ""
	for _, p := range perguntas {
		tipos += p["tipo"].(string) + " "
	}
	if tipos != "usuario dominio senha " || perguntas[1]["padrao"] != "WORKGROUP" || perguntas[2]["de_novo"] != false {
		t.Errorf("perguntas: %v", perguntas)
	}
	if r["erro"] != nil || r["raiz"] != m.Casa+"/gvfs/smb-share:server=nas,share=docs,user=ana" {
		t.Fatalf("montar: %v", r)
	}
	if lido, _ := os.ReadFile(m.Casa + "/lido.log"); string(lido) != "LIDO: ana||"+segredo+"\n" {
		t.Errorf("o gio leu do stdin: %q", lido)
	}
	// A senha foi só pelo stdin: nem nos argumentos nem no ambiente do gio.
	if strings.Contains(diario(m), segredo) {
		t.Error("a senha apareceu nos argumentos ou no ambiente do gio")
	}
}

func TestSenhaErradaPerguntaDeNovoENaoVaza(t *testing.T) {
	m := motor(t)
	gio(t, m, `n=0
while [ $n -lt 3 ]; do
  printf 'Password: '; read s
  [ "$s" = "certa" ] && { mkdir -p "$G/sftp:host=servidor"; exit 0; }
  echo "Permission denied, senha recebida: $s"
  n=$((n+1))
done
echo "gio: sftp://servidor/: Permission denied"; exit 2`)
	perguntas, r := conversa(t, m, "sftp://servidor/", segredo, segredo+"2", segredo+"3")
	if perguntas[0]["de_novo"] != false || perguntas[1]["de_novo"] != true || perguntas[2]["de_novo"] != true {
		t.Errorf("de_novo: %v", perguntas)
	}
	if r["motivo"] != "senha" || r["erro"] == nil {
		t.Fatalf("senha errada: %v", r)
	}
	// Nem o gio ecoando a senha a faz chegar à janela: nem na saída, nem nas perguntas.
	tudo := fmt.Sprint(r, perguntas)
	if strings.Contains(tudo, segredo) || !strings.Contains(fmt.Sprint(r["saida"]), "•••") {
		t.Errorf("a senha vazou: %s", tudo)
	}
	// Na segunda tentativa, certa, monta.
	_, r = conversa(t, m, "sftp://servidor/", "errada", "certa")
	if r["erro"] != nil || r["nome"] != "servidor" {
		t.Errorf("depois de errar uma vez: %v", r)
	}
}

func TestConfirmarOHostEDesistir(t *testing.T) {
	m := motor(t)
	gio(t, m, `printf 'Não foi possível verificar a identidade de “servidor”.\nImpressão digital: ab:cd\n[1] Entrar assim mesmo\n[2] Cancelar\nEscolha: '; read c
[ "$c" = "1" ] || { echo cancelado; exit 2; }
mkdir -p "$G/sftp:host=servidor"`)
	perguntas, r := conversa(t, m, "sftp://servidor/", "1")
	if perguntas[0]["tipo"] != "confirmar" || !strings.Contains(perguntas[0]["texto"].(string), "Impressão digital: ab:cd") || r["erro"] != nil {
		t.Errorf("confirmar: %v → %v", perguntas, r)
	}
	// Desistir (resposta null) cancela a montagem: o gio é morto e nada é montado.
	os.Remove(m.Casa + "/gvfs/sftp:host=servidor")
	_, r = conversa(t, m, "sftp://servidor/", nil)
	if r["motivo"] != "recusado" {
		t.Errorf("desistir: %v", r)
	}
	if _, err := os.Stat(m.Casa + "/gvfs/sftp:host=servidor"); err == nil {
		t.Error("desistiu, mas montou")
	}
}

func TestPerguntaDesconhecidaDemoraEFalha(t *testing.T) {
	m := motor(t)
	// Uma pergunta que o motor não conhece vira "outro" depois de a saída parar.
	antes := sossego
	sossego = 60 * time.Millisecond
	t.Cleanup(func() { sossego = antes })
	gio(t, m, `printf 'Segundo fator\nCódigo do aplicativo: '; read c; [ "$c" = "123456" ] && mkdir -p "$G/sftp:host=servidor"`)
	perguntas, r := conversa(t, m, "sftp://servidor/", "123456")
	if perguntas[0]["tipo"] != "outro" || !strings.Contains(perguntas[0]["texto"].(string), "Segundo fator") || r["erro"] != nil {
		t.Errorf("pergunta desconhecida: %v → %v", perguntas, r)
	}
	// O gio que não acaba nem pergunta é morto no prazo, e o que escreveu volta.
	t.Setenv("MKFILES_REMOTO_MS", "150")
	gio(t, m, `echo "tentando conectar"; exec sleep 30`)
	começo := time.Now()
	r = m.Pede(`{"op":"remoto-montar","uri":"ftp://lento/"}`)
	if r["motivo"] != "tempo" || r["saida"] != "tentando conectar" || time.Since(começo) > 5*time.Second {
		t.Errorf("demora: %v em %v", r, time.Since(começo))
	}
	// Falha comum: a última linha do gio vai na frase, a saída inteira em "saida".
	t.Setenv("MKFILES_REMOTO_MS", "3000")
	gio(t, m, `echo "linha um"; echo "gio: ftp://fora/: Host não encontrado" >&2; exit 2`)
	r = m.Pede(`{"op":"remoto-montar","uri":"ftp://fora/"}`)
	if r["motivo"] != "erro" || !strings.Contains(r["erro"].(string), "Host não encontrado") || !strings.Contains(r["saida"].(string), "linha um") {
		t.Errorf("falha: %v", r)
	}
	// O gio diz que montou mas a pasta não aparece.
	gio(t, m, `exit 0`)
	if r = m.Pede(`{"op":"remoto-montar","uri":"ftp://fantasma/"}`); r["motivo"] != "erro" {
		t.Errorf("montou sem aparecer: %v", r)
	}
	// Sem o gio instalado, e endereço torto (o gio nem é chamado).
	t.Setenv("MKFILES_GIO", m.Casa+"/nao-ha")
	if r = m.Pede(`{"op":"remoto-montar","uri":"sftp://servidor2/"}`); r["motivo"] != "falta" {
		t.Errorf("sem gio: %v", r)
	}
	os.Remove(m.Casa + "/gio.log")
	gio(t, m, `exit 0`)
	for _, ruim := range []string{"--help", "file:///etc", "sftp://ana:senha@x/", "servidor"} {
		if r = m.Pede(`{"op":"remoto-montar","uri":` + prova.J(ruim) + `}`); r["motivo"] != "endereco" {
			t.Errorf("endereço %q: %v", ruim, r)
		}
	}
	if diario(m) != "" {
		t.Errorf("endereço torto não chega ao gio: %s", diario(m))
	}
}

func TestCancelarAMontagem(t *testing.T) {
	m := motor(t)
	gio(t, m, `printf 'Password: '; read s; mkdir -p "$G/sftp:host=servidor"`)
	id := m.Manda(`{"op":"remoto-montar","uri":"sftp://servidor/"}`)
	m.Evento("pergunta-remota", nil)
	if r := m.Pede(fmt.Sprintf(`{"op":"cancelar","pedido":%d}`, id)); r["cancelado"] != true {
		t.Fatalf("cancelar: %v", r)
	}
	if r := m.Resposta(id); r["motivo"] != "recusado" {
		t.Errorf("montagem cancelada: %v", r)
	}
	// Responder depois não acha mais a pergunta.
	if r := m.Pede(fmt.Sprintf(`{"op":"remoto-responder","pedido":%d,"pergunta":1,"resposta":"x"}`, id)); r["erro"] == nil {
		t.Errorf("pergunta que já foi: %v", r)
	}
}

func TestRedeEVigia(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_DISCOS_MS", "20")
	gio(t, m, `[ "$1 $2 $3" = "list -u network:///" ] || exit 2
echo "smb://nas-da-sala/"; echo "sftp://impressora.local:22/"; echo "linha que não é endereço"; echo "dav://caixa%20forte/"`)
	r := m.Pede(`{"op":"remotos","rede":true}`)
	rede := fmt.Sprint(r["rede"])
	if !strings.Contains(rede, "nome:nas-da-sala") || !strings.Contains(rede, "uri:sftp://impressora.local:22/") || !strings.Contains(rede, "nome:caixa forte") || len(r["rede"].([]any)) != 3 {
		t.Errorf("rede: %v", r)
	}
	// Um remoto montado por fora (um celular ligado) aparece sozinho.
	m.Pasta("gvfs/mtp:host=Telefone")
	e := m.Evento("remotos", func(l prova.Linha) bool { return len(l["remotos"].([]any)) == 1 })
	um := e["remotos"].([]any)[0].(map[string]any)
	if um["protocolo"] != "mtp" || um["nome"] != "Telefone" {
		t.Errorf("celular: %v", um)
	}
	// Protocolo que o motor não conhece: aparece, sem endereço.
	m.Pasta("gvfs/google-drive:host=exemplo.org,user=ana")
	e = m.Evento("remotos", func(l prova.Linha) bool { return len(l["remotos"].([]any)) == 2 })
	if g := e["remotos"].([]any)[0].(map[string]any); g["uri"] != nil || g["protocolo"] != "google-drive" || g["nome"] != "exemplo.org" {
		t.Errorf("protocolo desconhecido: %v", g)
	}
	// Sem o gio, a rede é uma lista vazia, não um erro.
	t.Setenv("MKFILES_GIO", m.Casa+"/nao-ha")
	if r := m.Pede(`{"op":"remotos","rede":true}`); r["erro"] != nil || len(r["rede"].([]any)) != 0 {
		t.Errorf("sem gio: %v", r)
	}
	if base.GVFS() != m.Casa+"/gvfs" {
		t.Error("GVFS")
	}
}

func TestEscolha(t *testing.T) {
	numerada := "Can't verify the identity of “host”.\n[1] Log In Anyway\n[2] Cancel Login\nChoice: "
	emPortugues := "Não foi possível verificar.\n[1] Cancelar\n[2] Conectar mesmo assim\nEscolha: "
	casos := []struct {
		texto, resposta, linha string
		segue                  bool
	}{
		{numerada, "yes", "1", true},
		{numerada, "no", "2", true},
		{emPortugues, "yes", "2", true},
		{emPortugues, "no", "1", true},
		{numerada, "2", "2", true}, // o que a pessoa digitou passa direto
		{"Continue (yes/no)? ", "yes", "yes", true},
		{"Continue (yes/no)? ", "no", "no", true},
		{"[1] Aceitar\n[2] Aceitar sempre\nEscolha: ", "no", "", false}, // sem opção de não: desiste
		{"[1] Estranha\n[2] Outra\nChoice: ", "yes", "1", true},
	}
	for _, c := range casos {
		linha, segue := escolha(c.texto, c.resposta)
		if linha != c.linha || segue != c.segue {
			t.Errorf("escolha(%q, %q) = %q, %v; esperado %q, %v", c.texto, c.resposta, linha, segue, c.linha, c.segue)
		}
	}
}
