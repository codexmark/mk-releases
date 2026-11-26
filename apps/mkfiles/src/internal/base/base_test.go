package base

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func casa(t *testing.T) string {
	t.Helper()
	c := t.TempDir()
	t.Setenv("HOME", c)
	for _, v := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_DATA_DIRS", "XDG_CONFIG_DIRS"} {
		t.Setenv(v, "")
	}
	return c
}

func TestPastas(t *testing.T) {
	c := casa(t)
	if Config("mk", "x") != c+"/.config/mk/x" || Dados("Trash") != c+"/.local/share/Trash" ||
		Cache() != c+"/.cache" || Estado("mk") != c+"/.local/state/mk" {
		t.Error("os padrões do XDG")
	}
	t.Setenv("XDG_CONFIG_HOME", "/tmp/cfg/")
	t.Setenv("XDG_DATA_HOME", "relativo") // relativo não vale: fica o padrão
	t.Setenv("XDG_DATA_DIRS", "/a:/b/:c")
	if Config("x") != "/tmp/cfg/x" || Dados() != c+"/.local/share" {
		t.Error("XDG_* trocam a pasta, se absolutos")
	}
	if d := PastasDeDados(); strings.Join(d, " ") != c+"/.local/share /a /b" {
		t.Errorf("PastasDeDados: %v", d)
	}
	if d := PastasDeConfig(); strings.Join(d, " ") != "/tmp/cfg /etc/xdg" {
		t.Errorf("PastasDeConfig: %v", d)
	}
	t.Setenv("MKFILES_X", "meu")
	if Comando("MKFILES_X", "lsblk") != "meu" || Comando("MKFILES_NAO_HA", "lsblk") != "lsblk" {
		t.Error("Comando")
	}
	t.Setenv("MKFILES_R", "torto")
	if Raiz("MKFILES_R", "/etc") != "/etc" {
		t.Error("Raiz relativa não vale")
	}
}

func TestCaminhoENome(t *testing.T) {
	for _, bom := range []string{"/", "/a", "/a/b c/ç.txt"} {
		if err := Caminho(bom); err != nil {
			t.Errorf("Caminho(%q): %v", bom, err)
		}
	}
	for _, ruim := range []string{"", "a", "./a", "/a/", "/a//b", "/a/../b", "/a/./b", "/a\x00b", "~/a"} {
		if Caminho(ruim) == nil {
			t.Errorf("Caminho(%q) devia ser recusado", ruim)
		}
	}
	if Caminhos(nil) == nil || Caminhos([]string{"/a", "b"}) == nil || Caminhos([]string{"/a", "/b"}) != nil {
		t.Error("Caminhos")
	}
	for _, bom := range []string{"a", "a b.txt", ".oculto", "ç", "a..b"} {
		if err := Nome(bom); err != nil {
			t.Errorf("Nome(%q): %v", bom, err)
		}
	}
	for _, ruim := range []string{"", ".", "..", "a/b", "/", "a\x00", strings.Repeat("x", 256)} {
		if Nome(ruim) == nil {
			t.Errorf("Nome(%q) devia ser recusado", ruim)
		}
	}
}

func TestProtegido(t *testing.T) {
	t.Setenv("HOME", "/home/ana")
	for _, ruim := range []string{"", "/", "/home/ana", "/home", "relativo", "/home/ana/../ana"} {
		if Protegido(ruim) == nil {
			t.Errorf("Protegido(%q) devia recusar", ruim)
		}
	}
	for _, bom := range []string{"/home/ana/Documentos", "/home/anabela", "/tmp/x", "/home/bia"} {
		if err := Protegido(bom); err != nil {
			t.Errorf("Protegido(%q): %v", bom, err)
		}
	}
	// Sem pasta pessoal conhecida, nada é mexido.
	t.Setenv("HOME", "")
	if Protegido("/tmp/x") == nil {
		t.Error("sem HOME tem de recusar tudo")
	}
}

func TestDentro(t *testing.T) {
	if !Dentro("/a", "/a") || !Dentro("/a", "/a/b") || Dentro("/a", "/ab") || !Dentro("/", "/x") || Dentro("/a/b", "/a") {
		t.Error("Dentro")
	}
}

func TestGravaAtomico(t *testing.T) {
	pasta := t.TempDir()
	alvo := filepath.Join(pasta, "sub", "a.json")
	if err := GravaAtomico(alvo, []byte("um"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := GravaAtomico(alvo, []byte("dois"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(alvo)
	info, _ := os.Stat(alvo)
	if string(b) != "dois" || info.Mode().Perm() != 0o600 {
		t.Errorf("conteúdo %q, modo %v", b, info.Mode())
	}
	// Não sobra temporário.
	if e, _ := os.ReadDir(filepath.Dir(alvo)); len(e) != 1 {
		t.Errorf("sobrou temporário: %v", e)
	}
	if GravaAtomico("relativo", nil, 0o600) == nil {
		t.Error("caminho relativo")
	}
}

func TestURI(t *testing.T) {
	c := "/home/ana/Meus Documentos/ação #1%.txt"
	u := URI(c)
	if u != "file:///home/ana/Meus%20Documentos/a%C3%A7%C3%A3o%20%231%25.txt" {
		t.Errorf("URI: %s", u)
	}
	if volta, ok := DeURI(u); !ok || volta != c {
		t.Errorf("DeURI: %q %v", volta, ok)
	}
	if volta, ok := DeURI("file://localhost/a/b"); !ok || volta != "/a/b" {
		t.Errorf("localhost: %q", volta)
	}
	for _, ruim := range []string{"sftp://x/a", "file://outra/a", "/a", "file:///a%zz", "file:///a%00b"} {
		if _, ok := DeURI(ruim); ok {
			t.Errorf("DeURI(%q) devia falhar", ruim)
		}
	}
}

func TestMontagens(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "mountinfo")
	os.WriteFile(arq, []byte(`22 1 8:2 / / rw,relatime shared:1 - ext4 /dev/sda2 rw
30 22 8:17 / /run/media/ana/PEN\040DRIVE rw - vfat /dev/sdb1 rw
31 22 0:5 / /proc rw - proc proc rw
linha torta
`), 0o644)
	t.Setenv("MKFILES_MOUNTINFO", arq)
	ms := Montagens()
	if len(ms) != 3 || ms[1].Ponto != "/run/media/ana/PEN DRIVE" || ms[1].Tipo != "vfat" || ms[1].Origem != "/dev/sdb1" {
		t.Fatalf("Montagens: %+v", ms)
	}
	if PontoDe("/run/media/ana/PEN DRIVE/a/b") != "/run/media/ana/PEN DRIVE" || PontoDe("/home/ana") != "/" || PontoDe("/proc") != "/proc" {
		t.Error("PontoDe")
	}
}

func linhas(b *bytes.Buffer) []map[string]any {
	var ls []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) == nil {
			ls = append(ls, m)
		}
	}
	return ls
}

func TestSessao(t *testing.T) {
	var saida bytes.Buffer
	s := NovaSessao(&saida)
	s.Ligar("eco", func(p *Pedido) {
		var q struct {
			Texto string `json:"texto"`
		}
		_ = p.Ler(&q)
		p.Responde(Campos{"texto": q.Texto + " <ç>"})
	})
	s.Ligar("ruim", func(p *Pedido) { p.FalhaDeCampo("nome", "nome ocupado") })
	s.Ligar("nao", func(p *Pedido) { p.Recusa("não mexo na raiz") })
	comecou := make(chan bool)
	s.Ligar("fluxo", func(p *Pedido) {
		p.Fluxo(func(cancelado func() bool) {
			p.Responde(Campos{"fim": false})
			comecou <- true
			<-p.Parou()
			p.Responde(Campos{"fim": true, "cancelada": cancelado()})
		})
	})
	s.Atender([]byte(`{"id":1,"op":"eco","texto":"oi"}`))
	s.Atender([]byte(`lixo`))
	s.Atender([]byte(`{"id":"b","op":"dançar"}`))
	s.Atender([]byte(`{"id":3,"op":"ruim"}`))
	s.Atender([]byte(`{"id":{"x":1},"op":"eco"}`))
	if s.Codigo() != Erro {
		t.Errorf("código depois de um erro: %d", s.Codigo())
	}
	s.Atender([]byte(`{"id":5,"op":"nao"}`))
	s.Atender([]byte(`{"id":6,"op":"fluxo"}`))
	<-comecou
	s.Atender([]byte(`{"id":7,"op":"cancelar","pedido":99}`))
	s.Atender([]byte(`{"id":8,"op":"cancelar","pedido":6}`))
	s.Espera()
	s.Evento("pasta", Campos{"caminho": "/x"})
	ls := linhas(&saida)
	if len(ls) != 11 {
		t.Fatalf("esperava 11 linhas, vieram %d: %s", len(ls), saida.String())
	}
	if !strings.Contains(saida.String(), "oi <ç>") {
		t.Error("a saída não escapa acentos nem <>")
	}
	if ls[1]["erro"] != "pedido que não é JSON" || ls[1]["id"] != nil {
		t.Errorf("lixo: %v", ls[1])
	}
	if ls[2]["erro"] != "não conheço o pedido «dançar»" || ls[2]["id"] != "b" {
		t.Errorf("op desconhecido: %v", ls[2])
	}
	if ls[3]["campo"] != "nome" || ls[4]["id"] != nil {
		t.Errorf("campo e id torto: %v %v", ls[3], ls[4])
	}
	if ls[5]["recusado"] != true || s.Codigo() != Recusado {
		t.Errorf("recusa: %v, código %d", ls[5], s.Codigo())
	}
	// A resposta do cancelar e a última linha do fluxo cancelado saem de duas
	// goroutines: a ordem entre elas não é garantida (conferir pela posição
	// fazia este teste falhar de vez em quando). Vale o conteúdo de cada uma.
	doID := func(id float64, serve func(map[string]any) bool) map[string]any {
		for _, l := range ls[6:10] {
			if l["id"] == id && serve(l) {
				return l
			}
		}
		t.Errorf("faltou a linha do pedido %v em %v", id, ls[6:10])
		return map[string]any{}
	}
	if ls[6]["fim"] != false || ls[6]["id"] != float64(6) || ls[7]["cancelado"] != false || ls[7]["id"] != float64(7) {
		t.Errorf("o começo do fluxo e o cancelar de quem não existe: %v %v", ls[6], ls[7])
	}
	doID(8, func(l map[string]any) bool { return l["cancelado"] == true })
	doID(6, func(l map[string]any) bool { return l["fim"] == true && l["cancelada"] == true })
	if ls[10]["evento"] != "pasta" || ls[10]["caminho"] != "/x" {
		t.Errorf("evento: %v", ls[10])
	}
	if ops := strings.Join(s.Ops(), " "); ops != "cancelar eco fluxo nao ruim" {
		t.Errorf("Ops: %s", ops)
	}

	// Fechar cancela os fluxos abertos, roda os AoFechar e espera.
	fechou := false
	s.AoFechar(func() { fechou = true })
	s.Atender([]byte(`{"id":9,"op":"fluxo"}`))
	<-comecou
	s.Fecha()
	s.Fecha()
	if !fechou || !strings.Contains(saida.String(), `"id":9`) {
		t.Error("Fecha")
	}

	// A chamada de um subcomando só: as linhas saem sem id.
	saida.Reset()
	um := NovaSessao(&saida)
	um.SemID = true
	um.Ligar("eco", func(p *Pedido) { p.Responde(Campos{"ok": true}) })
	um.Atender([]byte(`{"op":"eco"}`))
	um.Atender([]byte(`x`))
	if strings.Contains(saida.String(), `"id"`) {
		t.Errorf("SemID: %s", saida.String())
	}
}

func TestRemoto(t *testing.T) {
	g := t.TempDir()
	t.Setenv("MKFILES_GVFS", g)
	raiz := filepath.Join(g, "sftp:host=servidor.exemplo,user=ana")
	os.MkdirAll(filepath.Join(raiz, "home"), 0o755)
	r, nome, ok := Remoto(raiz + "/home/a.txt")
	if !ok || r != raiz || nome != "servidor.exemplo" {
		t.Errorf("Remoto: %q %q %v", r, nome, ok)
	}
	if r, _, ok = Remoto(raiz); !ok || r != raiz {
		t.Error("a raiz do remoto é remota")
	}
	for _, local := range []string{g, "/tmp/x", g + "x/y", ""} {
		if _, _, ok := Remoto(local); ok {
			t.Errorf("Remoto(%q) não é remoto", local)
		}
	}
	if p, pares := PartesDoRemoto("smb-share:server=nas,share=Meus%20Arquivos"); p != "smb-share" || pares["share"] != "Meus Arquivos" {
		t.Errorf("PartesDoRemoto: %s %v", p, pares)
	}
	if NomeDoRemoto("smb-share:server=nas,share=x") != "nas" || NomeDoRemoto("estranho") != "estranho" {
		t.Error("NomeDoRemoto")
	}
	// Só erro de conexão, e só em caminho remoto, vira "a conexão caiu".
	if f, caiu := ConexaoCaiu(raiz+"/home", syscall.ENOTCONN); !caiu || f != "a conexão com servidor.exemplo caiu" {
		t.Errorf("ENOTCONN: %q %v", f, caiu)
	}
	if _, caiu := ConexaoCaiu(raiz+"/home/nao-ha", fs.ErrNotExist); caiu {
		t.Error("arquivo que não existe num remoto de pé não é queda")
	}
	if _, caiu := ConexaoCaiu("/tmp/x", syscall.ENOTCONN); caiu {
		t.Error("caminho local nunca é queda de conexão")
	}
	os.RemoveAll(raiz)
	if _, caiu := ConexaoCaiu(raiz+"/home", fs.ErrNotExist); !caiu {
		t.Error("a raiz sumiu: é queda")
	}
	// Sem MKFILES_GVFS vale $XDG_RUNTIME_DIR/gvfs; sem os dois, nada é remoto.
	t.Setenv("MKFILES_GVFS", "")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if GVFS() != "/run/user/1000/gvfs" {
		t.Errorf("GVFS: %s", GVFS())
	}
	t.Setenv("XDG_RUNTIME_DIR", "")
	if GVFS() != "" {
		t.Error("sem onde, GVFS é vazio")
	}
}

func TestProtegidoNosRemotos(t *testing.T) {
	t.Setenv("HOME", "/home/ana")
	t.Setenv("MKFILES_GVFS", "/run/user/1000/gvfs")
	for _, ruim := range []string{"/run/user/1000/gvfs", "/run/user/1000", "/run", "/run/user/1000/gvfs/sftp:host=servidor"} {
		if Protegido(ruim) == nil {
			t.Errorf("Protegido(%q) devia recusar", ruim)
		}
	}
	for _, bom := range []string{"/run/user/1000/gvfs/sftp:host=servidor/docs", "/run/user/1000/outra", "/run/media/ana/PEN"} {
		if err := Protegido(bom); err != nil {
			t.Errorf("Protegido(%q): %v", bom, err)
		}
	}
}
