package disco

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mkescrita/internal/base"
)

// Rascunho é um texto ainda não salvo, guardado para o caso de o app morrer.
type Rascunho struct {
	ID string `json:"id"`
	// Origem: o arquivo que estava aberto ("" para um documento sem nome).
	Origem string `json:"origem"`
	Nome   string `json:"nome"`
	// Base: o sha256 do arquivo de origem quando o rascunho foi guardado
	// (ou "ausente"/""): é contra ela que um salvar depois da recuperação
	// confere se o arquivo mudou.
	Base       string `json:"base_sha256"`
	GuardadoEm int64  `json:"guardado_em_s"`
	// Dono: o processo da janela que guardou. Enquanto ele vive, o rascunho
	// é de uma janela aberta e não é oferecido a outra.
	Dono     int    `json:"dono_pid"`
	DonoNome string `json:"dono_nome"`
	Texto    string `json:"texto"`
}

// Guardado é um item de `rascunho listar` (sem o texto inteiro).
type Guardado struct {
	ID         string `json:"id"`
	Origem     string `json:"origem"`
	Nome       string `json:"nome"`
	GuardadoEm int64  `json:"guardado_em_s"`
	Bytes      int    `json:"tamanho_bytes"`
	Trecho     string `json:"trecho"`
	Vivo       bool   `json:"vivo"`
}

// Agora é o relógio (os testes trocam).
var Agora = func() int64 { return time.Now().Unix() }

var idValido = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func arquivoDoRascunho(id string) (string, error) {
	if !idValido.MatchString(id) {
		return "", fmt.Errorf("identificador de rascunho inválido: %q", id)
	}
	return filepath.Join(base.Rascunhos(), id+".json"), nil
}

// nomeDoProcesso lê o nome de um processo vivo ("" se não existe).
func nomeDoProcesso(pid int) string {
	if pid <= 0 {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(base.Proc(), strconv.Itoa(pid), "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// GuardarRascunho grava (ou regrava) o rascunho `id`. `dono` é o processo da janela.
func GuardarRascunho(id, origem, baseSha string, texto []byte, dono int) (*Guardado, error) {
	caminho, err := arquivoDoRascunho(id)
	if err != nil {
		return nil, err
	}
	if origem != "" && !filepath.IsAbs(origem) {
		return nil, fmt.Errorf("a origem do rascunho tem de ser um caminho absoluto: %q", origem)
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		return nil, fmt.Errorf("não consegui criar a pasta dos rascunhos: %v", frase(err))
	}
	r := Rascunho{ID: id, Origem: origem, Nome: base.Nome(origem), Base: baseSha, GuardadoEm: Agora(),
		Dono: dono, DonoNome: nomeDoProcesso(dono), Texto: string(texto)}
	dados, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if err := gravaAtomico(caminho, append(dados, '\n'), 0o600, nil); err != nil {
		return nil, err
	}
	g := resumoDe(&r)
	return &g, nil
}

func trecho(texto string) string {
	for _, linha := range strings.Split(texto, "\n") {
		linha = strings.Join(strings.Fields(linha), " ")
		if linha == "" {
			continue
		}
		if letras := []rune(linha); len(letras) > 80 {
			return string(letras[:80]) + "…"
		}
		return linha
	}
	return ""
}

func resumoDe(r *Rascunho) Guardado {
	return Guardado{ID: r.ID, Origem: r.Origem, Nome: r.Nome, GuardadoEm: r.GuardadoEm, Bytes: len(r.Texto),
		Trecho: trecho(r.Texto), Vivo: r.DonoNome != "" && nomeDoProcesso(r.Dono) == r.DonoNome}
}

// LerRascunho devolve o rascunho inteiro.
func LerRascunho(id string) (*Rascunho, error) {
	caminho, err := arquivoDoRascunho(id)
	if err != nil {
		return nil, err
	}
	dados, err := os.ReadFile(caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("não há rascunho %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("não consegui ler o rascunho: %v", frase(err))
	}
	var r Rascunho
	if err := json.Unmarshal(dados, &r); err != nil || r.ID != id {
		return nil, fmt.Errorf("o rascunho %s está estragado", id)
	}
	return &r, nil
}

// ListarRascunhos devolve os rascunhos guardados, o mais recente primeiro. Um
// arquivo estragado fica de fora (e fica no disco: ninguém apaga texto aqui).
func ListarRascunhos() ([]Guardado, error) {
	itens := []Guardado{}
	entradas, err := os.ReadDir(base.Rascunhos())
	if errors.Is(err, fs.ErrNotExist) {
		return itens, nil
	}
	if err != nil {
		return nil, fmt.Errorf("não consegui ler a pasta dos rascunhos: %v", frase(err))
	}
	for _, e := range entradas {
		id, tem := strings.CutSuffix(e.Name(), ".json")
		if !tem || !idValido.MatchString(id) {
			continue
		}
		if r, err := LerRascunho(id); err == nil {
			itens = append(itens, resumoDe(r))
		}
	}
	sort.SliceStable(itens, func(i, j int) bool {
		if itens[i].GuardadoEm != itens[j].GuardadoEm {
			return itens[i].GuardadoEm > itens[j].GuardadoEm
		}
		return itens[i].ID < itens[j].ID
	})
	return itens, nil
}

// ApagarRascunho apaga um rascunho. Apagar o que já não existe não é erro.
func ApagarRascunho(id string) error {
	caminho, err := arquivoDoRascunho(id)
	if err != nil {
		return err
	}
	if err := os.Remove(caminho); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("não consegui apagar o rascunho: %v", frase(err))
	}
	return nil
}
