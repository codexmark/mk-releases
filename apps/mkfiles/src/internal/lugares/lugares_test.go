package lugares

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

// O lsblk de mentira imprime o arquivo .lsblk.json da casa; o udisksctl de
// mentira anota a chamada em .udisks e "monta" trocando esse arquivo.
func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	lsblk := m.Grava(".bin/lsblk", "#!/bin/sh\necho \"lsblk $*\" >>\"$HOME/.chamadas\"\ncat \"$HOME/.lsblk.json\"\n")
	udisks := m.Grava(".bin/udisksctl", `#!/bin/sh
echo "udisksctl $*" >>"$HOME/.chamadas"
if [ -f "$HOME/.udisks-falha" ]; then echo "Error mounting $3: nao autorizado" >&2; exit 1; fi
if [ -f "$HOME/.lsblk-depois.json" ]; then cp "$HOME/.lsblk-depois.json" "$HOME/.lsblk.json"; fi
echo "feito"
`)
	os.Chmod(lsblk, 0o755)
	os.Chmod(udisks, 0o755)
	t.Setenv("MKFILES_LSBLK", lsblk)
	t.Setenv("MKFILES_UDISKSCTL", udisks)
	return m
}

func chamadas(m *prova.Motor) string {
	b, _ := os.ReadFile(m.Casa + "/.chamadas")
	return string(b)
}

func TestLugares(t *testing.T) {
	m := motor(t)
	ids := func() string {
		var l []string
		for _, x := range m.Pede(`{"op":"lugares"}`)["lugares"].([]any) {
			l = append(l, x.(map[string]any)["id"].(string))
		}
		return strings.Join(l, " ")
	}
	if ids() != "pessoal lixeira raiz" {
		t.Errorf("sem user-dirs.dirs: %s", ids())
	}
	m.Pasta("Documentos")
	m.Pasta("Baixados")
	m.Pasta("fora/Músicas")
	m.Grava(".config/user-dirs.dirs", `# gerado pelo xdg-user-dirs
XDG_DESKTOP_DIR="$HOME/Mesa"
XDG_DOCUMENTS_DIR="$HOME/Documentos"
XDG_DOWNLOAD_DIR="$HOME/Baixados"
XDG_PICTURES_DIR="$HOME/"
XDG_MUSIC_DIR="`+m.Casa+`/fora/Músicas"
XDG_VIDEOS_DIR="$HOME/NaoExiste"
`)
	if ids() != "pessoal documentos downloads musicas lixeira raiz" {
		t.Errorf("com user-dirs.dirs (o que não existe ou é a pasta pessoal não aparece): %s", ids())
	}
	ls := m.Pede(`{"op":"lugares"}`)["lugares"].([]any)
	if p := ls[0].(map[string]any); p["caminho"] != m.Casa || p["nome"] != "Pasta pessoal" || p["icone"] != "user-home" {
		t.Errorf("pessoal: %v", p)
	}
	if d := ls[2].(map[string]any); d["caminho"] != m.Casa+"/Baixados" || d["nome"] != "Downloads" {
		t.Errorf("downloads: %v", d)
	}
	if l := ls[4].(map[string]any); l["caminho"] != nil || l["id"] != "lixeira" {
		t.Errorf("lixeira: %v", l)
	}
	if r := ls[5].(map[string]any); r["caminho"] != "/" {
		t.Errorf("raiz: %v", r)
	}
	for _, torto := range []string{`XDG_DOCUMENTS_DIR=$HOME/Documentos`, `XDG_DOCUMENTS_DIR="relativa"`, `XDG_DOCUMENTS_DIR="$HOMEDocumentos"`} {
		m.Grava(".config/user-dirs.dirs", torto+"\n")
		if ids() != "pessoal lixeira raiz" {
			t.Errorf("linha torta %q: %s", torto, ids())
		}
	}
}

func favs(r prova.Linha) []map[string]any {
	var l []map[string]any
	lista, _ := r["favoritos"].([]any)
	for _, x := range lista {
		l = append(l, x.(map[string]any))
	}
	return l
}

func TestFavoritos(t *testing.T) {
	m := motor(t)
	arq := m.Casa + "/.config/gtk-3.0/bookmarks"
	le := func() string { b, _ := os.ReadFile(arq); return string(b) }
	work := m.Pasta("Work")
	espaco := m.Pasta("Meus Projetos")
	fotos := m.Pasta("Fotos")
	if f := favs(m.Pede(`{"op":"lugares"}`)); len(f) != 0 {
		t.Errorf("sem arquivo: %v", f)
	}
	uWork, uEspaco := "file://"+work, "file://"+m.Casa+"/Meus%20Projetos"
	m.Grava(".config/gtk-3.0/bookmarks", uWork+"\nsftp://ana@servidor/home/ana Servidor\n\nfile://"+m.Casa+"/sumiu Antiga\n")

	f := favs(m.Pede(`{"op":"lugares"}`))
	if len(f) != 3 || f[0]["caminho"] != work || f[0]["nome"] != "Work" || f[0]["existe"] != true || f[0]["remoto"] != false || f[0]["uri"] != uWork {
		t.Errorf("ler: %v", f)
	}
	if f[1]["remoto"] != true || f[1]["caminho"] != nil || f[1]["nome"] != "Servidor" || f[1]["uri"] != "sftp://ana@servidor/home/ana" {
		t.Errorf("remoto: %v", f[1])
	}
	if f[2]["existe"] != false || f[2]["nome"] != "Antiga" {
		t.Errorf("pasta que sumiu: %v", f[2])
	}

	// Acrescentar no fim, com nome, e numa posição; a linha em branco e a
	// remota ficam como estavam.
	f = favs(m.Pede(`{"op":"favorito-por","caminho":` + prova.J(espaco) + `,"nome":null,"posicao":null}`))
	if len(f) != 4 || f[3]["caminho"] != espaco || f[3]["nome"] != "Meus Projetos" || f[3]["uri"] != uEspaco {
		t.Errorf("acrescentar: %v", f)
	}
	m.Pede(`{"op":"favorito-por","caminho":` + prova.J(fotos) + `,"nome":"Minhas fotos","posicao":1}`)
	quer := uWork + "\nfile://" + fotos + " Minhas fotos\nsftp://ana@servidor/home/ana Servidor\n\nfile://" + m.Casa + "/sumiu Antiga\n" + uEspaco + "\n"
	if le() != quer {
		t.Errorf("arquivo:\n%s\nesperado:\n%s", le(), quer)
	}
	// De novo a mesma pasta: não duplica.
	if f = favs(m.Pede(`{"op":"favorito-por","caminho":` + prova.J(fotos) + `}`)); len(f) != 5 || le() != quer {
		t.Errorf("duplicou: %v", f)
	}
	// Não sobra temporário.
	if e, _ := os.ReadDir(filepath.Dir(arq)); len(e) != 1 {
		t.Errorf("sobrou temporário: %v", e)
	}

	// Reordenar: o último vai para o começo; a linha em branco fica onde está.
	f = favs(m.Pede(`{"op":"favorito-mover","de":4,"para":0}`))
	if f[0]["caminho"] != espaco || f[1]["caminho"] != work || f[4]["nome"] != "Antiga" {
		t.Errorf("mover: %v", f)
	}
	if linhas := strings.Split(le(), "\n"); linhas[3] != "" || len(linhas) != 7 {
		t.Errorf("a linha em branco saiu do lugar: %q", linhas)
	}
	f = favs(m.Pede(`{"op":"favorito-mover","de":0,"para":2}`))
	if f[0]["caminho"] != work || f[2]["caminho"] != espaco {
		t.Errorf("mover para o meio: %v", f)
	}
	if r := m.Pede(`{"op":"favorito-mover","de":9,"para":0}`); r["erro"] == nil {
		t.Error("posição que não existe")
	}

	// Renomear (o remoto também), tirar o nome, e tirar.
	f = favs(m.Pede(`{"op":"favorito-renomear","uri":` + prova.J(uWork) + `,"nome":"Trabalho  "}`))
	if f[0]["nome"] != "Trabalho" || !strings.Contains(le(), uWork+" Trabalho\n") {
		t.Errorf("renomear: %v", f[0])
	}
	f = favs(m.Pede(`{"op":"favorito-renomear","uri":` + prova.J(uWork) + `,"nome":""}`))
	if f[0]["nome"] != "Work" || !strings.Contains(le(), uWork+"\n") {
		t.Errorf("tirar o nome: %v", f[0])
	}
	m.Pede(`{"op":"favorito-renomear","uri":"sftp://ana@servidor/home/ana","nome":"Casa"}`)
	if !strings.Contains(le(), "sftp://ana@servidor/home/ana Casa\n") {
		t.Errorf("renomear o remoto: %s", le())
	}
	if r := m.Pede(`{"op":"favorito-renomear","uri":` + prova.J(uWork) + `,"nome":"a\nfile:///x"}`); r["campo"] != "nome" {
		t.Errorf("nome com quebra de linha: %v", r)
	}
	f = favs(m.Pede(`{"op":"favorito-tirar","uri":` + prova.J(uWork) + `}`))
	if len(f) != 4 || strings.Contains(le(), uWork+"\n") || !strings.Contains(le(), "sftp://") {
		t.Errorf("tirar: %v", f)
	}
	if r := m.Pede(`{"op":"favorito-tirar","uri":` + prova.J(uWork) + `}`); r["erro"] == nil {
		t.Error("tirar o que não está")
	}

	arquivo := m.Grava("a.txt", "x")
	for _, ruim := range []string{"relativa", m.Casa + "/nao-ha", arquivo, m.Casa + "/../x"} {
		if r := m.Pede(`{"op":"favorito-por","caminho":` + prova.J(ruim) + `}`); r["erro"] == nil {
			t.Errorf("favorito-por %q devia falhar", ruim)
		}
	}
}

const arvore = `{"blockdevices": [
 {"name":"nvme0n1","path":"/dev/nvme0n1","label":null,"fstype":null,"size":512110190592,"mountpoints":[null],"rm":false,"hotplug":false,"type":"disk","uuid":null,
  "children":[
   {"name":"nvme0n1p1","path":"/dev/nvme0n1p1","label":null,"fstype":"vfat","size":1073741824,"mountpoints":["/boot"],"rm":false,"hotplug":false,"type":"part","uuid":"AAAA-BBBB"},
   {"name":"nvme0n1p2","path":"/dev/nvme0n1p2","label":"raiz","fstype":"btrfs","size":400000000000,"mountpoints":["/","/home","/var/log"],"rm":false,"hotplug":false,"type":"part","uuid":"u-raiz"},
   {"name":"nvme0n1p3","path":"/dev/nvme0n1p3","label":null,"fstype":"swap","size":8000000000,"mountpoints":["[SWAP]"],"rm":false,"hotplug":false,"type":"part","uuid":"u-swap"},
   {"name":"nvme0n1p4","path":"/dev/nvme0n1p4","label":"Dados","fstype":"ext4","size":100000000000,"mountpoints":["@MONTADO@"],"rm":false,"hotplug":false,"type":"part","uuid":"u-dados"},
   {"name":"nvme0n1p5","path":"/dev/nvme0n1p5","label":null,"fstype":"crypto_LUKS","size":1000,"mountpoints":[null],"rm":false,"hotplug":false,"type":"part","uuid":"u-luks"}
  ]},
 {"name":"sdb","path":"/dev/sdb","label":null,"fstype":null,"size":"16000000000","mountpoints":[null],"rm":"1","hotplug":"1","type":"disk","uuid":null,
  "children":[
   {"name":"sdb1","path":"/dev/sdb1","label":"PENDRIVE","fstype":"vfat","size":"15000000000","mountpoints":[@PEN@],"rm":"1","hotplug":"1","type":"part","uuid":"1234-ABCD"},
   {"name":"sdb2","path":"/dev/sdb2","label":null,"fstype":"ext4","size":"1500000000","mountpoints":[null],"rm":false,"hotplug":false,"type":"part","uuid":null}
  ]},
 {"name":"loop0","path":"/dev/loop0","label":null,"fstype":"ext4","size":1000,"mountpoints":["/mnt/x"],"rm":false,"hotplug":false,"type":"loop","uuid":null},
 {"name":"mal","path":"/dev/sd c; rm","label":null,"fstype":"ext4","size":1000,"mountpoints":[null],"rm":false,"hotplug":false,"type":"part","uuid":null}
]}`

func gravaArvore(m *prova.Motor, arquivo, pen string) {
	if pen != "null" {
		pen = prova.J(pen)
	}
	m.Grava(arquivo, strings.NewReplacer("@MONTADO@", m.Casa, "@PEN@", pen).Replace(arvore))
}

func discos(r prova.Linha) map[string]map[string]any {
	ds := map[string]map[string]any{}
	lista, _ := r["discos"].([]any)
	for _, x := range lista {
		d := x.(map[string]any)
		ds[d["dispositivo"].(string)] = d
	}
	return ds
}

func TestDiscos(t *testing.T) {
	m := motor(t)
	gravaArvore(m, ".lsblk.json", "null")
	r := m.Pede(`{"op":"discos"}`)
	ds := discos(r)
	if len(ds) != 3 {
		t.Fatalf("entram só os volumes da pessoa (sem swap, sistema, LUKS, loop, caminho torto): %v", r)
	}
	dados := ds["/dev/nvme0n1p4"]
	if dados["nome"] != "Dados" || dados["rotulo"] != "Dados" || dados["sistema"] != "ext4" || dados["tamanho"] != float64(100000000000) ||
		dados["montado"] != m.Casa || dados["removivel"] != false || dados["uuid"] != "u-dados" {
		t.Errorf("Dados: %v", dados)
	}
	if livre, total := dados["livre"].(float64), dados["total"].(float64); livre <= 0 || total < livre {
		t.Errorf("espaço do montado: %v de %v", livre, total)
	}
	pen := ds["/dev/sdb1"]
	if pen["nome"] != "PENDRIVE" || pen["montado"] != nil || pen["livre"] != nil || pen["total"] != nil || pen["removivel"] != true ||
		pen["tamanho"] != float64(15000000000) {
		t.Errorf("PENDRIVE: %v", pen)
	}
	// Sem rótulo, o nome diz o tamanho; a partição herda o "removível" do disco.
	if s := ds["/dev/sdb2"]; s["nome"] != "Volume de 1,5 GB" || s["rotulo"] != nil || s["removivel"] != true || s["uuid"] != nil {
		t.Errorf("sem rótulo: %v", s)
	}
	if !strings.Contains(chamadas(m), "lsblk --json --bytes -o NAME,PATH,LABEL,FSTYPE,SIZE,MOUNTPOINTS,RM,HOTPLUG,TYPE,UUID") {
		t.Errorf("a linha do lsblk: %s", chamadas(m))
	}
	for b, quer := range map[int64]string{999: "999 B", 16000000000: "16 GB", 1500000000: "1,5 GB", 2000000000000: "2 TB", 512000000: "512 MB"} {
		if tamanhoLegivel(b) != quer {
			t.Errorf("tamanhoLegivel(%d) = %s", b, tamanhoLegivel(b))
		}
	}

	m.Grava(".lsblk.json", "isto não é json")
	if r := m.Pede(`{"op":"discos"}`); r["erro"] == nil {
		t.Error("lsblk que responde lixo")
	}
	t.Setenv("MKFILES_LSBLK", m.Casa+"/.bin/nao-ha")
	if r := m.Pede(`{"op":"discos"}`); r["erro"] == nil {
		t.Error("sem lsblk")
	}
}

func TestMontarDesmontarEjetar(t *testing.T) {
	m := motor(t)
	ponto := m.Pasta("midia/PENDRIVE")
	gravaArvore(m, ".lsblk.json", "null")
	gravaArvore(m, ".lsblk-depois.json", ponto)

	r := m.Pede(`{"op":"montar","dispositivo":"/dev/sdb1"}`)
	if r["montado"] != ponto || discos(r)["/dev/sdb1"]["montado"] != ponto {
		t.Fatalf("montar: %v", r)
	}
	if !strings.Contains(chamadas(m), "udisksctl mount -b /dev/sdb1\n") {
		t.Errorf("chamadas: %s", chamadas(m))
	}
	// Já montado: não chama de novo.
	os.Remove(m.Casa + "/.chamadas")
	m.Pede(`{"op":"montar","dispositivo":"/dev/sdb1"}`)
	if strings.Contains(chamadas(m), "udisksctl") {
		t.Errorf("montar o que já está montado: %s", chamadas(m))
	}

	gravaArvore(m, ".lsblk-depois.json", "null")
	r = m.Pede(`{"op":"desmontar","dispositivo":"/dev/sdb1"}`)
	if discos(r)["/dev/sdb1"]["montado"] != nil || !strings.Contains(chamadas(m), "udisksctl unmount -b /dev/sdb1\n") {
		t.Errorf("desmontar: %v / %s", r, chamadas(m))
	}

	// Ejetar: desmonta o que está montado e desliga o disco pai.
	gravaArvore(m, ".lsblk.json", ponto)
	os.Remove(m.Casa + "/.chamadas")
	if r = m.Pede(`{"op":"ejetar","dispositivo":"/dev/sdb2"}`); r["erro"] != nil {
		t.Fatalf("ejetar: %v", r)
	}
	if c := chamadas(m); !strings.Contains(c, "udisksctl unmount -b /dev/sdb1\nudisksctl power-off -b /dev/sdb\n") {
		t.Errorf("ejetar: %s", c)
	}

	// O udisksctl que recusa: a frase dele vai para a janela.
	gravaArvore(m, ".lsblk.json", "null")
	m.Grava(".udisks-falha", "")
	if r = m.Pede(`{"op":"montar","dispositivo":"/dev/sdb1"}`); r["erro"] != "Error mounting /dev/sdb1: nao autorizado" {
		t.Errorf("falha do udisksctl: %v", r)
	}
	os.Remove(m.Casa + "/.udisks-falha")

	// Só um dispositivo da lista, e com cara de dispositivo, chega ao comando.
	os.Remove(m.Casa + "/.chamadas")
	for _, ruim := range []string{"", "sdb1", "/dev/sdz9", "/dev/../etc/passwd", "/dev/sd c; rm", "/dev/nvme0n1p2", "-b", "/dev/sdb1 --force"} {
		if r := m.Pede(`{"op":"montar","dispositivo":` + prova.J(ruim) + `}`); r["campo"] != "dispositivo" {
			t.Errorf("dispositivo %q devia ser recusado: %v", ruim, r)
		}
	}
	if strings.Contains(chamadas(m), "udisksctl") {
		t.Errorf("um dispositivo recusado chegou ao udisksctl: %s", chamadas(m))
	}
}

func TestVigiarDiscos(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_DISCOS_MS", "20")
	ponto := m.Pasta("midia/PENDRIVE")
	gravaArvore(m, ".lsblk.json", "null")
	if r := m.Pede(`{"op":"vigiar-discos","ligado":true}`); r["ligado"] != true {
		t.Fatalf("ligar: %v", r)
	}
	// Nada mudou: nenhum evento.
	m.SemEvento("discos", 120*time.Millisecond)
	gravaArvore(m, ".lsblk.json", ponto)
	e := m.Evento("discos", nil)
	if discos(e)["/dev/sdb1"]["montado"] != ponto {
		t.Errorf("o evento traz a lista nova: %v", e)
	}
	// Uma montagem que o lsblk não mostra de outro jeito também avisa.
	m.Montagem(m.Casa+"/outra", "/dev/x")
	m.Evento("discos", nil)
	if r := m.Pede(`{"op":"vigiar-discos","ligado":false}`); r["ligado"] != false {
		t.Fatalf("desligar: %v", r)
	}
	time.Sleep(60 * time.Millisecond)
	gravaArvore(m, ".lsblk.json", "null")
	m.SemEvento("discos", 150*time.Millisecond)
}

func TestEspaco(t *testing.T) {
	m := motor(t)
	r := m.Pede(`{"op":"espaco","caminho":` + prova.J(m.Casa) + `}`)
	if livre, ok := r["livre"].(float64); !ok || livre <= 0 || r["total"].(float64) < livre {
		t.Errorf("espaco: %v", r)
	}
	for _, ruim := range []string{"relativo", m.Casa + "/nao-ha"} {
		if r := m.Pede(`{"op":"espaco","caminho":` + prova.J(ruim) + `}`); r["erro"] == nil {
			t.Errorf("espaco %q devia falhar", ruim)
		}
	}
}
