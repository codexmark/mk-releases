// Package abrir diz com que app cada arquivo abre (mimeapps.list e os
// .desktop), monta o argv a executar e o do terminal do Mk, e lista os
// modelos. O motor não executa nada disso: quem roda o argv é a janela. Veja
// ../../../CONTRATO.md.
package abrir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mkfiles/internal/base"
)

// app é um .desktop de aplicativo.
type app struct {
	ID       string
	Arquivo  string
	Nome     string
	Exec     string
	Icone    string
	Terminal bool
	Oculto   bool // NoDisplay ou Hidden
	Mimes    []string
}

// grupos lê um arquivo no formato dos .desktop: grupo -> chave -> valor. Só
// guarda os grupos pedidos (nil: todos).
func grupos(caminho string, quais ...string) map[string]map[string]string {
	b, err := os.ReadFile(caminho)
	if err != nil || len(b) > 4<<20 {
		return nil
	}
	gs := map[string]map[string]string{}
	atual := ""
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" || linha[0] == '#' {
			continue
		}
		if linha[0] == '[' && strings.HasSuffix(linha, "]") {
			atual = linha[1 : len(linha)-1]
			continue
		}
		chave, valor, ok := strings.Cut(linha, "=")
		if !ok || atual == "" {
			continue
		}
		if len(quais) > 0 {
			serve := false
			for _, q := range quais {
				serve = serve || q == atual
			}
			if !serve {
				continue
			}
		}
		if gs[atual] == nil {
			gs[atual] = map[string]string{}
		}
		chave = strings.TrimSpace(chave)
		if _, ja := gs[atual][chave]; !ja {
			gs[atual][chave] = strings.TrimSpace(valor)
		}
	}
	return gs
}

// semEscape desfaz os escapes de um valor de texto do .desktop (\s \n \t \r \\).
func semEscape(v string) string {
	if !strings.ContainsRune(v, '\\') {
		return v
	}
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' && i+1 < len(v) {
			i++
			switch v[i] {
			case 's':
				b.WriteByte(' ')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(v[i])
			}
			continue
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

func lerApp(id, caminho string) (app, bool) {
	g := grupos(caminho, "Desktop Entry")["Desktop Entry"]
	if g == nil || (g["Type"] != "" && g["Type"] != "Application") {
		return app{}, false
	}
	a := app{ID: id, Arquivo: caminho, Exec: semEscape(g["Exec"]), Icone: g["Icon"], Terminal: g["Terminal"] == "true",
		Oculto: g["NoDisplay"] == "true" || g["Hidden"] == "true"}
	for _, chave := range []string{"Name[pt_BR]", "Name[pt]", "Name"} {
		if g[chave] != "" {
			a.Nome = semEscape(g[chave])
			break
		}
	}
	if a.Nome == "" {
		a.Nome = strings.TrimSuffix(id, ".desktop")
	}
	for _, m := range strings.Split(g["MimeType"], ";") {
		if m = strings.TrimSpace(m); m != "" {
			a.Mimes = append(a.Mimes, m)
		}
	}
	return a, true
}

// osApps lê todos os .desktop das pastas applications/, na ordem do XDG: o
// primeiro id achado vale (o da pessoa esconde o do sistema). A subpasta entra
// no id com "-".
func osApps() (map[string]app, []string) {
	apps := map[string]app{}
	var ordem []string
	for _, raiz := range base.PastasDeDados() {
		pasta := filepath.Join(raiz, "applications")
		var achados []string
		_ = filepath.WalkDir(pasta, func(c string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(c, ".desktop") {
				achados = append(achados, c)
			}
			return nil
		})
		sort.Strings(achados)
		for _, c := range achados {
			rel, err := filepath.Rel(pasta, c)
			if err != nil {
				continue
			}
			id := strings.ReplaceAll(rel, "/", "-")
			if _, ja := apps[id]; ja {
				continue
			}
			a, ok := lerApp(id, c)
			if !ok {
				// Um arquivo que não é aplicativo ainda esconde o do sistema.
				a = app{ID: id, Oculto: true}
			}
			apps[id] = a
			ordem = append(ordem, id)
		}
	}
	return apps, ordem
}

func (a app) serve() bool { return !a.Oculto && a.Exec != "" }

// Os arquivos mimeapps.list, na ordem de leitura do padrão XDG.
func arquivosDeAssociacao() []string {
	var arqs []string
	for _, c := range base.PastasDeConfig() {
		arqs = append(arqs, filepath.Join(c, "mimeapps.list"))
	}
	for _, d := range base.PastasDeDados() {
		arqs = append(arqs, filepath.Join(d, "applications", "mimeapps.list"))
	}
	return arqs
}

func idsDe(valor string) []string {
	var ids []string
	for _, id := range strings.Split(valor, ";") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// associados diz, para um MIME, o app padrão ("" se nenhum) e os apps que o
// abrem, em ordem: os do mimeapps.list (Default e Added, arquivo a arquivo) e
// depois os que o declaram em MimeType. Um app removido num arquivo não volta
// pelos arquivos seguintes nem pelo MimeType.
func associados(mime string, apps map[string]app, ordem []string) (padrao string, lista []string) {
	removidos := map[string]bool{}
	visto := map[string]bool{}
	poe := func(id string) bool {
		a, existe := apps[id]
		if !existe || !a.serve() || removidos[id] {
			return false
		}
		if !visto[id] {
			visto[id] = true
			lista = append(lista, id)
		}
		return true
	}
	for _, arq := range arquivosDeAssociacao() {
		g := grupos(arq, "Default Applications", "Added Associations", "Removed Associations")
		if g == nil {
			continue
		}
		for _, id := range idsDe(g["Removed Associations"][mime]) {
			if !visto[id] {
				removidos[id] = true
			}
		}
		// O padrão é uma escolha explícita: vale mesmo que um arquivo tenha
		// removido a associação.
		for _, id := range idsDe(g["Default Applications"][mime]) {
			if a, existe := apps[id]; existe && a.serve() && padrao == "" {
				padrao = id
			}
		}
		for _, id := range idsDe(g["Added Associations"][mime]) {
			poe(id)
		}
	}
	var declaram []string
	for _, id := range ordem {
		for _, m := range apps[id].Mimes {
			if m == mime {
				declaram = append(declaram, id)
				break
			}
		}
	}
	for _, id := range declaram {
		poe(id)
	}
	if padrao == "" && len(lista) > 0 {
		padrao = lista[0]
	}
	if padrao != "" {
		resto := []string{padrao}
		for _, id := range lista {
			if id != padrao {
				resto = append(resto, id)
			}
		}
		lista = resto
	}
	return padrao, lista
}

// definirPadrao troca (ou acrescenta) a linha do MIME no grupo [Default
// Applications] do mimeapps.list da pessoa. O resto do arquivo fica como está.
func definirPadrao(mime, id string) error {
	caminho := base.Config("mimeapps.list")
	b, err := os.ReadFile(caminho)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	nova := mime + "=" + id + ";"
	var linhas []string
	if len(b) > 0 {
		linhas = strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	}
	dentro, feito, fimDoGrupo := false, false, -1
	for i, linha := range linhas {
		limpa := strings.TrimSpace(linha)
		if strings.HasPrefix(limpa, "[") {
			dentro = limpa == "[Default Applications]"
			if dentro && fimDoGrupo < 0 {
				fimDoGrupo = i + 1
			}
			continue
		}
		if !dentro {
			continue
		}
		if limpa != "" {
			fimDoGrupo = i + 1
		}
		if chave, _, ok := strings.Cut(limpa, "="); ok && strings.TrimSpace(chave) == mime {
			if feito {
				linhas[i] = "\x00" // uma segunda linha do mesmo MIME sai
			} else {
				linhas[i] = nova
				feito = true
			}
		}
	}
	switch {
	case feito:
	case fimDoGrupo >= 0:
		linhas = append(linhas[:fimDoGrupo], append([]string{nova}, linhas[fimDoGrupo:]...)...)
	default:
		if len(linhas) > 0 && strings.TrimSpace(linhas[len(linhas)-1]) != "" {
			linhas = append(linhas, "")
		}
		linhas = append(linhas, "[Default Applications]", nova)
	}
	var saida strings.Builder
	for _, l := range linhas {
		if l != "\x00" {
			saida.WriteString(l + "\n")
		}
	}
	return base.GravaAtomico(caminho, []byte(saida.String()), 0o644)
}
