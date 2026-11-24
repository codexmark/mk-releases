package historico

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func num(v int64) *int64   { return &v }
func txt(v string) *string { return &v }

const registro = `[2019-03-01 10:00] [PACMAN] Running 'pacman -S fd'
[2019-03-01 10:01] [ALPM] installed fd (8.0-1)
[2026-11-20T23:44:58-0400] [ALPM] running 'zz-snap-pac-post.hook'...
[2026-11-20T23:45:28-0400] [PACMAN] Running '/usr/bin/pacman --noconfirm -Rnu fuse2'
[2026-11-20T23:45:29-0400] [ALPM] transaction started
[2026-11-20T23:45:30-0400] [ALPM] removed fuse2 (2.9.9-6)
[2026-11-21T08:00:00-0400] [ALPM] upgraded htop (3.5.2-1 -> 3.5.3-1)
[2026-11-21T08:00:01-0400] [ALPM] downgraded glibc (2.42-1 -> 2.41-1)
[2026-11-21T08:00:02+0000] [ALPM] reinstalled bash (5.3.20-1)
[2026-11-21T08:00:03-0400] [ALPM] installed HTOP-extra (1:2.0-1)
[2026-11-21T08:00:04-0400] [ALPM-SCRIPTLET] installed de mentira (1-1)
[data torta] [ALPM] installed relogio (1-1)
[2026-11-21T08:00:05-0400] [ALPM] warning: /etc/x installed as /etc/x.pacnew
linha sem nada
`

func escreve(t *testing.T, texto string) {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "pacman.log")
	if err := os.WriteFile(caminho, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKPACOTES_LOG", caminho)
}

func quando(t *testing.T, texto string) *int64 {
	t.Helper()
	q, err := time.Parse("2006-01-02T15:04:05-0700", texto)
	if err != nil {
		t.Fatal(err)
	}
	return num(q.Unix())
}

func TestLe_ResumeOQueFoiInstaladoAtualizadoERemovido(t *testing.T) {
	escreve(t, registro)
	eventos, total, err := Le("", 100)
	if err != nil {
		t.Fatal(err)
	}
	antigo := time.Date(2019, 3, 1, 10, 1, 0, 0, time.Local).Unix()
	quer := []Evento{
		{nil, "instalado", "relogio", nil, txt("1-1")},
		{quando(t, "2026-11-21T08:00:03-0400"), "instalado", "HTOP-extra", nil, txt("1:2.0-1")},
		{quando(t, "2026-11-21T08:00:02+0000"), "reinstalado", "bash", txt("5.3.20-1"), txt("5.3.20-1")},
		{quando(t, "2026-11-21T08:00:01-0400"), "rebaixado", "glibc", txt("2.42-1"), txt("2.41-1")},
		{quando(t, "2026-11-21T08:00:00-0400"), "atualizado", "htop", txt("3.5.2-1"), txt("3.5.3-1")},
		{quando(t, "2026-11-20T23:45:30-0400"), "removido", "fuse2", txt("2.9.9-6"), nil},
		{&antigo, "instalado", "fd", nil, txt("8.0-1")},
	}
	if total != len(quer) || !reflect.DeepEqual(eventos, quer) {
		t.Errorf("total %d", total)
		for i := range eventos {
			if i >= len(quer) || !reflect.DeepEqual(eventos[i], quer[i]) {
				t.Errorf("evento %d: %s", i, mostra(eventos[i]))
			}
		}
	}
}

func mostra(e Evento) string {
	s := func(p *string) string {
		if p == nil {
			return "nil"
		}
		return *p
	}
	var q int64 = -1
	if e.Quando != nil {
		q = *e.Quando
	}
	return strings.Join([]string{time.Unix(q, 0).UTC().String(), e.Acao, e.Nome, s(e.De), s(e.Para)}, " | ")
}

func TestLe_OMaisNovoPrimeiroECortaNoLimite(t *testing.T) {
	escreve(t, registro)
	eventos, total, err := Le("", 2)
	if err != nil || total != 7 || len(eventos) != 2 || eventos[0].Nome != "relogio" || eventos[1].Nome != "HTOP-extra" {
		t.Errorf("veio %v, total %d, %v", eventos, total, err)
	}
}

func TestLe_BuscaPeloNomeSemDiferenciarMaiusculas(t *testing.T) {
	escreve(t, registro)
	eventos, total, _ := Le("HtOp", 100)
	if total != 2 || len(eventos) != 2 || eventos[0].Nome != "HTOP-extra" || eventos[1].Nome != "htop" {
		t.Errorf("veio %v, total %d", eventos, total)
	}
	if eventos, total, _ := Le("nao-ha", 100); total != 0 || eventos == nil || len(eventos) != 0 {
		t.Errorf("nada achado é lista vazia (não nil): %v, %d", eventos, total)
	}
}

func TestLe_SemORegistroEErroEmFrase(t *testing.T) {
	t.Setenv("MKPACOTES_LOG", filepath.Join(t.TempDir(), "nada.log"))
	if _, _, err := Le("", 10); err == nil || !strings.Contains(err.Error(), "não consegui ler o registro do pacman") {
		t.Errorf("erro: %v", err)
	}
}

func TestLe_RegistroVazioELinhaGigante(t *testing.T) {
	escreve(t, "")
	if eventos, total, err := Le("", 10); err != nil || total != 0 || len(eventos) != 0 {
		t.Errorf("vazio: %v %d %v", eventos, total, err)
	}
	escreve(t, "[2026-11-21T08:00:00-0400] [ALPM-SCRIPTLET] "+strings.Repeat("x", 3<<20)+"\n[2026-11-21T08:00:01-0400] [ALPM] installed fd (1-1)\n")
	if eventos, _, err := Le("", 10); err != nil || len(eventos) != 1 {
		t.Errorf("linha gigante não pode derrubar a leitura: %v %v", eventos, err)
	}
}
