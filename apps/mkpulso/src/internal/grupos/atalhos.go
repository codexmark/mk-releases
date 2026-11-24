package grupos

import (
	"os"
	"path/filepath"
	"strings"

	"mkpulso/internal/base"
)

// Atalho é o que um .desktop diz de um app.
type Atalho struct {
	Nome  string // Name[pt_BR], senão Name[pt], senão Name, senão o nome do arquivo
	Icone string // Icon; "" se não tem
}

type guardado struct {
	mudou   int64
	tamanho int64
	atalho  Atalho
}

// Atalhos procura os .desktop dos apps. As pastas, na ordem em que valem:
// ~/.local/share/applications, cada pasta de XDG_DATA_DIRS (com
// /applications) e /usr/share/applications (trocável por MKPULSO_APPS).
// O que já foi lido fica guardado enquanto o arquivo não mudar.
type Atalhos struct {
	pastas []string
	lidos  map[string]guardado
}

// NovosAtalhos monta a lista de pastas com o ambiente de agora.
func NovosAtalhos() *Atalhos {
	a := &Atalhos{lidos: map[string]guardado{}}
	visto := map[string]bool{}
	poe := func(pasta string) {
		if pasta != "" && !visto[pasta] {
			visto[pasta] = true
			a.pastas = append(a.pastas, pasta)
		}
	}
	poe(base.Home(".local", "share", "applications"))
	for _, d := range filepath.SplitList(os.Getenv("XDG_DATA_DIRS")) {
		if d != "" {
			poe(filepath.Join(d, "applications"))
		}
	}
	if apps := os.Getenv("MKPULSO_APPS"); apps != "" {
		poe(apps)
	} else {
		poe("/usr/share/applications")
	}
	return a
}

// Procurar acha o atalho <nome>.desktop (ou com o nome em minúsculas: a
// classe de uma janela às vezes vem com outra caixa).
func (a *Atalhos) Procurar(nome string) (Atalho, bool) {
	if a == nil || nome == "" || strings.ContainsAny(nome, "/\x00") || strings.HasPrefix(nome, ".") {
		return Atalho{}, false
	}
	tentativas := []string{nome}
	if minusculo := strings.ToLower(nome); minusculo != nome {
		tentativas = append(tentativas, minusculo)
	}
	for _, pasta := range a.pastas {
		for _, n := range tentativas {
			caminho := filepath.Join(pasta, n+".desktop")
			info, err := os.Stat(caminho)
			if err != nil || info.IsDir() {
				continue
			}
			if g, ok := a.lidos[caminho]; ok && g.mudou == info.ModTime().UnixNano() && g.tamanho == info.Size() {
				return g.atalho, true
			}
			atalho := lerAtalho(caminho, n)
			a.lidos[caminho] = guardado{mudou: info.ModTime().UnixNano(), tamanho: info.Size(), atalho: atalho}
			return atalho, true
		}
	}
	return Atalho{}, false
}

// lerAtalho lê a seção [Desktop Entry] de um .desktop. Arquivo torto ou sem
// Name vale com o nome do arquivo.
func lerAtalho(caminho, nome string) Atalho {
	atalho := Atalho{Nome: nome}
	b, err := os.ReadFile(caminho)
	if err != nil {
		return atalho
	}
	nomes := map[string]string{}
	dentro := false
	for _, linha := range strings.Split(strings.ToValidUTF8(string(b), "�"), "\n") {
		linha = strings.TrimSpace(linha)
		if strings.HasPrefix(linha, "[") {
			dentro = linha == "[Desktop Entry]"
			continue
		}
		if !dentro || strings.HasPrefix(linha, "#") {
			continue
		}
		chave, valor, ok := strings.Cut(linha, "=")
		if !ok {
			continue
		}
		chave, valor = strings.TrimSpace(chave), strings.TrimSpace(valor)
		switch chave {
		case "Name", "Name[pt]", "Name[pt_BR]":
			if _, ja := nomes[chave]; !ja && valor != "" {
				nomes[chave] = valor
			}
		case "Icon":
			if atalho.Icone == "" {
				atalho.Icone = valor
			}
		}
	}
	for _, chave := range []string{"Name[pt_BR]", "Name[pt]", "Name"} {
		if n, ok := nomes[chave]; ok {
			atalho.Nome = n
			break
		}
	}
	return atalho
}
