package vigia

import (
	"os"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

// Num remoto (a pasta do gvfs) o inotify não vê nada: a pasta é relida.
func TestRemotoERelido(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_GVFS", m.Pasta("gvfs"))
	t.Setenv("MKFILES_RELER_MS", "20")
	pasta := m.Pasta("gvfs/sftp:host=servidor/docs")
	m.Grava("gvfs/sftp:host=servidor/docs/a.txt", "um")
	if r := m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != true {
		t.Fatalf("vigiar: %v", r)
	}
	// Não é inotify: nenhum descritor foi aberto para esta pasta.
	// Sem mudança, nenhum evento.
	m.SemEvento("pasta", 120*time.Millisecond)
	m.Grava("gvfs/sftp:host=servidor/docs/b.txt", "dois")
	e := m.Evento("pasta", daPasta(pasta))
	if e["recarregar"] != true || e["sumiu"] != false {
		t.Errorf("mudou: a janela lista de novo: %v", e)
	}
	// Mudança de conteúdo (tamanho) também conta.
	m.Grava("gvfs/sftp:host=servidor/docs/a.txt", "um bem maior")
	if e = m.Evento("pasta", daPasta(pasta)); e["recarregar"] != true {
		t.Errorf("tamanho mudou: %v", e)
	}
	// Duas referências: um desvigiar não para a releitura.
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != true {
		t.Errorf("ainda há uma referência: %v", r)
	}
	os.Remove(pasta + "/b.txt")
	m.Evento("pasta", daPasta(pasta))
	// O remoto foi desmontado (a pasta some): sumiu, e a releitura para.
	os.Remove(pasta + "/a.txt")
	os.Remove(pasta)
	var sumiu prova.Linha
	for sumiu == nil {
		if e := m.Evento("pasta", daPasta(pasta)); e["sumiu"] == true {
			sumiu = e
		}
	}
	m.SemEvento("pasta", 120*time.Millisecond)
}

func TestDesvigiarParaAReleitura(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_GVFS", m.Pasta("gvfs"))
	t.Setenv("MKFILES_RELER_MS", "20")
	pasta := m.Pasta("gvfs/smb-share:server=nas,share=x")
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != false {
		t.Errorf("desvigiar: %v", r)
	}
	m.Grava("gvfs/smb-share:server=nas,share=x/novo.txt", "x")
	m.SemEvento("pasta", 150*time.Millisecond)
	if Releitura() != 20*time.Millisecond {
		t.Error("MKFILES_RELER_MS")
	}
	t.Setenv("MKFILES_RELER_MS", "torto")
	if Releitura() != 5*time.Second {
		t.Error("o padrão é 5 s")
	}
}
