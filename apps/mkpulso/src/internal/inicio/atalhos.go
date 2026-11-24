package inicio

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"mkpulso/internal/base"
)

// As duas pastas de autostart do padrão XDG. A do usuário sobrepõe a do
// sistema, arquivo por arquivo.
func pastaDoSistema() string { return base.Etc("xdg", "autostart") }
func pastaDoUsuario() string { return base.Home(".config", "autostart") }

const grupoPrincipal = "[Desktop Entry]"

// separa parte a linha "Chave = valor" de um .desktop.
func separa(linha string) (chave, valor string, ok bool) {
	linha = strings.TrimSpace(linha)
	if linha == "" || linha[0] == '#' || linha[0] == '[' {
		return "", "", false
	}
	chave, valor, ok = strings.Cut(linha, "=")
	return strings.TrimSpace(chave), strings.TrimSpace(valor), ok
}

// chaves lê as chaves do grupo [Desktop Entry] (e só dele).
func chaves(conteudo []byte) map[string]string {
	m := map[string]string{}
	dentro := false
	for _, linha := range strings.Split(string(conteudo), "\n") {
		if aparada := strings.TrimSpace(linha); strings.HasPrefix(aparada, "[") {
			dentro = aparada == grupoPrincipal
			continue
		}
		if !dentro {
			continue
		}
		if chave, valor, ok := separa(linha); ok {
			if _, ja := m[chave]; !ja {
				m[chave] = valor
			}
		}
	}
	return m
}

// poe escreve chave=valor no grupo [Desktop Entry]: troca a linha que já
// existe ou acrescenta uma no fim do grupo. O resto do arquivo fica igual.
func poe(conteudo []byte, chave, valor string) []byte {
	linhas := strings.Split(string(conteudo), "\n")
	nova := chave + "=" + valor
	var saida []string
	dentro, posta := false, false
	fimDoGrupo := -1 // onde, em saida, acaba a última linha não vazia do grupo
	for _, linha := range linhas {
		aparada := strings.TrimSpace(linha)
		if strings.HasPrefix(aparada, "[") {
			dentro = aparada == grupoPrincipal && fimDoGrupo < 0
			saida = append(saida, linha)
			if dentro {
				fimDoGrupo = len(saida)
			}
			continue
		}
		if dentro {
			if c, _, ok := separa(linha); ok && c == chave {
				if !posta {
					saida = append(saida, nova)
					posta = true
					fimDoGrupo = len(saida)
				}
				continue // uma chave repetida some
			}
			if aparada != "" {
				saida = append(saida, linha)
				fimDoGrupo = len(saida)
				continue
			}
		}
		saida = append(saida, linha)
	}
	switch {
	case posta:
	case fimDoGrupo < 0:
		saida = append([]string{grupoPrincipal, nova}, saida...)
	default:
		saida = append(saida[:fimDoGrupo], append([]string{nova}, saida[fimDoGrupo:]...)...)
		if fimDoGrupo == len(saida)-1 {
			saida = append(saida, "") // o arquivo não acabava em quebra de linha
		}
	}
	return []byte(strings.Join(saida, "\n"))
}

// tira apaga a chave do grupo [Desktop Entry].
func tira(conteudo []byte, chave string) []byte {
	var saida []string
	dentro := false
	for _, linha := range strings.Split(string(conteudo), "\n") {
		if aparada := strings.TrimSpace(linha); strings.HasPrefix(aparada, "[") {
			dentro = aparada == grupoPrincipal
		} else if dentro {
			if c, _, ok := separa(linha); ok && c == chave {
				continue
			}
		}
		saida = append(saida, linha)
	}
	return []byte(strings.Join(saida, "\n"))
}

// ligado diz se o atalho sobe: não está escondido nem desligado do jeito do
// GNOME.
func ligado(conteudo []byte) bool {
	c := chaves(conteudo)
	return c["Hidden"] != "true" && c["X-GNOME-Autostart-enabled"] != "false"
}

// liga devolve o conteúdo sem o que desliga o atalho.
func liga(conteudo []byte) []byte {
	conteudo = tira(conteudo, "Hidden")
	if chaves(conteudo)["X-GNOME-Autostart-enabled"] == "false" {
		conteudo = poe(conteudo, "X-GNOME-Autostart-enabled", "true")
	}
	return conteudo
}

// igual compara dois atalhos sem ligar para o Hidden nem para espaço no fim.
func igual(a, b []byte) bool {
	limpa := func(c []byte) string {
		linhas := strings.Split(string(tira(c, "Hidden")), "\n")
		for i := range linhas {
			linhas[i] = strings.TrimRight(linhas[i], " \t\r")
		}
		return strings.TrimRight(strings.Join(linhas, "\n"), "\n")
	}
	return limpa(a) == limpa(b)
}

// lista parte um valor de lista do .desktop ("GNOME;KDE;").
func lista(valor string) []string {
	var itens []string
	for _, v := range strings.Split(valor, ";") {
		if v = strings.TrimSpace(v); v != "" {
			itens = append(itens, v)
		}
	}
	return itens
}

// valeAqui diz se o atalho é para este ambiente: OnlyShowIn e NotShowIn
// contra o XDG_CURRENT_DESKTOP (o do Mk é Hyprland), e o TryExec instalado.
func valeAqui(c map[string]string) bool {
	ambiente := lista(strings.ReplaceAll(os.Getenv("XDG_CURRENT_DESKTOP"), ":", ";"))
	if len(ambiente) == 0 {
		ambiente = []string{"Hyprland"}
	}
	nele := func(nomes []string) bool {
		for _, n := range nomes {
			for _, a := range ambiente {
				if n == a {
					return true
				}
			}
		}
		return false
	}
	if so, tem := c["OnlyShowIn"]; tem && !nele(lista(so)) {
		return false
	}
	if nele(lista(c["NotShowIn"])) {
		return false
	}
	if programa := c["TryExec"]; programa != "" {
		if _, err := exec.LookPath(programa); err != nil {
			return false
		}
	}
	return true
}

// comandoDoExec tira do Exec os códigos de campo (%U, %f...) do padrão.
func comandoDoExec(valor string) string {
	var partes []string
	for _, p := range strings.Fields(valor) {
		if len(p) == 2 && p[0] == '%' && p[1] != '%' {
			continue
		}
		partes = append(partes, strings.ReplaceAll(p, "%%", "%"))
	}
	return strings.Join(partes, " ")
}

// leAtalho lê um .desktop que seja arquivo de verdade (ou link para um).
func leAtalho(caminho string) ([]byte, bool) {
	if info, err := os.Stat(caminho); err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	b, err := os.ReadFile(caminho)
	return b, err == nil
}

func listaAtalhos() ([]Item, error) {
	nomes := map[string]bool{}
	for _, pasta := range []string{pastaDoSistema(), pastaDoUsuario()} {
		entradas, _ := os.ReadDir(pasta) // sem a pasta, sem atalhos
		for _, e := range entradas {
			if strings.HasSuffix(e.Name(), ".desktop") {
				nomes[e.Name()] = true
			}
		}
	}
	ordem := make([]string, 0, len(nomes))
	for n := range nomes {
		ordem = append(ordem, n)
	}
	sort.Strings(ordem)

	itens := []Item{}
	for _, nome := range ordem {
		doSistema, temSistema := leAtalho(filepath.Join(pastaDoSistema(), nome))
		doUsuario, temUsuario := leAtalho(filepath.Join(pastaDoUsuario(), nome))
		if !temSistema && !temUsuario {
			continue
		}
		// O id é o caminho do sistema quando há um: assim ele não muda
		// quando a cópia de desligar aparece ou some.
		id, efetivo := filepath.Join(pastaDoSistema(), nome), doSistema
		if !temSistema {
			id = filepath.Join(pastaDoUsuario(), nome)
		}
		if temUsuario {
			efetivo = doUsuario
		}
		c := chaves(efetivo)
		if !valeAqui(c) {
			continue
		}
		base := map[string]string{}
		if temSistema && temUsuario {
			// A cópia que só desliga não repete o nome nem o comando.
			base = chaves(doSistema)
			if !ligado(efetivo) && !valeAqui(base) {
				continue
			}
		}
		chave := func(nomes ...string) string {
			for _, m := range []map[string]string{c, base} {
				for _, n := range nomes {
					if v := m[n]; v != "" {
						return v
					}
				}
			}
			return ""
		}
		item := Item{ID: prefixoAtalho + id, Origem: origemAtalho, Nome: chave("Name[pt_BR]", "Name[pt]", "Name"),
			Comando: comandoDoExec(chave("Exec")), Ligado: ligado(efetivo), DoSistema: temSistema, Mudavel: true}
		if item.Nome == "" {
			item.Nome = strings.TrimSuffix(nome, ".desktop")
		}
		itens = append(itens, item)
	}
	return itens, nil
}

// mudaAtalho liga ou desliga um atalho pelo caminho que veio no id. O
// caminho tem de ser, ao pé da letra, um .desktop direto numa das duas
// pastas de autostart: nada de "..", de subpasta ou de outro lugar.
func mudaAtalho(caminho string, ligar bool) error {
	nome := filepath.Base(caminho)
	pasta := filepath.Dir(caminho)
	if !filepath.IsAbs(caminho) || filepath.Clean(caminho) != caminho || !strings.HasSuffix(nome, ".desktop") ||
		(pasta != pastaDoSistema() && pasta != pastaDoUsuario()) {
		return fmt.Errorf("%q não é um atalho de início das pastas de autostart", caminho)
	}
	doSistema, temSistema := leAtalho(filepath.Join(pastaDoSistema(), nome))
	destino := filepath.Join(pastaDoUsuario(), nome)
	doUsuario, temUsuario := leAtalho(destino)
	if !temSistema && !temUsuario {
		return fmt.Errorf("o atalho %s %w mais", nome, errNaoExiste)
	}
	efetivo := doSistema
	if temUsuario {
		efetivo = doUsuario
	}
	if !ligar {
		// O jeito do padrão XDG: uma cópia na home com Hidden=true.
		return grava(destino, poe(efetivo, "Hidden", "true"), doUsuario, temUsuario)
	}
	novo := liga(efetivo)
	if temSistema {
		if len(chaves(novo)) == 0 {
			novo = liga(doSistema) // a cópia só servia para desligar
		}
		if ligado(doSistema) && igual(novo, doSistema) {
			// A cópia só diferia por desligar: sem ela, vale o do sistema.
			if err := os.Remove(destino); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("não deu para apagar %s: %v", destino, err)
			}
			return nil
		}
	}
	return grava(destino, novo, doUsuario, temUsuario)
}

// grava escreve o atalho de forma atômica (um arquivo ao lado e um rename),
// a não ser que já esteja como se quer.
func grava(destino string, novo, atual []byte, tem bool) error {
	if tem && bytes.Equal(novo, atual) {
		return nil
	}
	pasta := filepath.Dir(destino)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return fmt.Errorf("não deu para criar %s: %v", pasta, err)
	}
	tmp, err := os.CreateTemp(pasta, ".mkpulso-*.tmp")
	if err != nil {
		return fmt.Errorf("não deu para escrever em %s: %v", pasta, err)
	}
	defer os.Remove(tmp.Name()) // depois do rename não há mais o que apagar
	_, err = tmp.Write(novo)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if fecha := tmp.Close(); err == nil {
		err = fecha
	}
	if err == nil {
		err = os.Rename(tmp.Name(), destino)
	}
	if err != nil {
		return fmt.Errorf("não deu para gravar %s: %v", destino, err)
	}
	return nil
}
