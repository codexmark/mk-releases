package item

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

func TestListarComPrazoEConexaoCaida(t *testing.T) {
	m := motor(t)
	gvfs := m.Pasta("gvfs")
	t.Setenv("MKFILES_GVFS", gvfs)
	raiz := m.Pasta("gvfs/sftp:host=servidor")
	pasta := m.Pasta("gvfs/sftp:host=servidor/docs")
	m.Grava(pasta+"/a.txt", "x")
	// Um remoto montado lista como qualquer pasta.
	if r := m.PedeFluxo(`{"op":"listar","caminho":` + prova.J(pasta) + `}`); r[len(r)-1]["total"] != float64(1) {
		t.Fatalf("listar remoto: %v", r)
	}

	// O remoto não responde: o motor desiste no prazo, com o motivo "tempo",
	// e continua atendendo os outros pedidos enquanto espera.
	// O abre de mentira fica até o fim do teste (trocar a variável com um
	// listar largado para trás seria uma corrida): quem liga e desliga a
	// demora é a chave.
	solta := make(chan struct{})
	var preso atomic.Bool
	preso.Store(true)
	abre = func(c string) (*os.File, error) {
		if c == pasta && preso.Load() {
			<-solta
		}
		return os.Open(c)
	}
	PrazoDeAbrir = 80 * time.Millisecond
	t.Cleanup(func() {
		close(solta)
		m.S.Fecha() // ninguém mais lê as variáveis
		time.Sleep(20 * time.Millisecond)
		abre, PrazoDeAbrir = os.Open, 20*time.Second
	})
	id := m.Manda(`{"op":"listar","caminho":` + prova.J(pasta) + `}`)
	if r := m.PedeFluxo(`{"op":"listar","caminho":` + prova.J(m.Casa) + `}`); r[len(r)-1]["fim"] != true {
		t.Errorf("outro pedido enquanto o remoto não responde: %v", r)
	}
	if r := m.Resposta(id); r["motivo"] != "tempo" || r["erro"] != "servidor não respondeu a tempo" {
		t.Errorf("prazo: %v", r)
	}
	// Cancelar enquanto espera solta na hora.
	PrazoDeAbrir = 20 * time.Second
	id = m.Manda(`{"op":"listar","caminho":` + prova.J(pasta) + `}`)
	m.Pede(`{"op":"cancelar","pedido":` + prova.J(id) + `}`)
	if r := m.Resposta(id); r["cancelada"] != true {
		t.Errorf("cancelar um listar que não responde: %v", r)
	}

	// O remoto foi desmontado: "conexao", com o nome dele. Uma pasta que só
	// não existe dentro de um remoto vivo continua sendo "nao-existe".
	preso.Store(false)
	if r := m.Pede(`{"op":"listar","caminho":` + prova.J(raiz+"/nao-ha") + `}`); r["motivo"] != "nao-existe" {
		t.Errorf("pasta que não existe num remoto vivo: %v", r)
	}
	os.Remove(pasta + "/a.txt")
	os.Remove(pasta)
	os.Remove(raiz)
	if r := m.Pede(`{"op":"listar","caminho":` + prova.J(pasta) + `}`); r["motivo"] != "conexao" || r["erro"] != "a conexão com servidor caiu" {
		t.Errorf("remoto desmontado: %v", r)
	}
	if r := m.Pede(`{"op":"item","caminho":` + prova.J(pasta) + `}`); r["motivo"] != "conexao" {
		t.Errorf("item de remoto desmontado: %v", r)
	}
}
