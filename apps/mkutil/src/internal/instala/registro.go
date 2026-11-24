package instala

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mkutil/internal/base"
)

// Item é uma linha do registro do que o MkInstalar instalou
// (~/.local/state/mk/instalados.json).
type Item struct {
	ID          string  `json:"id"`
	Tipo        string  `json:"tipo"`
	Nome        string  `json:"nome"`
	Versao      *string `json:"versao"`
	Origem      string  `json:"origem"`
	Destino     string  `json:"destino"`
	Atalho      *string `json:"atalho"`
	Icone       *string `json:"icone"`
	InstaladoEm int64   `json:"instalado_em_s"`
	Tamanho     int64   `json:"tamanho_bytes"`
}

type arquivoDeRegistro struct {
	Versao int    `json:"versao"`
	Itens  []Item `json:"itens"`
}

// LeRegistro devolve o que está anotado, em ordem de nome. Sem o arquivo, nada.
func LeRegistro() ([]Item, error) {
	b, err := os.ReadFile(base.Registro())
	if errors.Is(err, os.ErrNotExist) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("não consegui ler o registro (%s): %v", base.Til(base.Registro()), err)
	}
	var a arquivoDeRegistro
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("o registro (%s) está ilegível: conserte ou apague o arquivo", base.Til(base.Registro()))
	}
	itens := a.Itens
	if itens == nil {
		itens = []Item{}
	}
	sort.SliceStable(itens, func(i, j int) bool { return itens[i].ID < itens[j].ID })
	return itens, nil
}

func gravaRegistro(itens []Item) error {
	b, err := json.MarshalIndent(arquivoDeRegistro{Versao: 1, Itens: itens}, "", "  ")
	if err != nil {
		return err
	}
	return gravaInteiro(base.Registro(), append(b, '\n'), 0o644)
}

// gravaInteiro escreve um arquivo de uma vez: num temporário ao lado, trocado
// no fim, para nunca ficar pela metade.
func gravaInteiro(caminho string, conteudo []byte, modo os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(caminho), ".mkinstalar-*.parcial")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(conteudo)
	if err == nil {
		err = f.Chmod(modo)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(tmp, caminho)
	}
	if err != nil {
		_ = os.Remove(tmp)
	}
	return err
}

// anota põe (ou troca) um item no registro.
func anota(novo Item) error {
	itens, err := LeRegistro()
	if err != nil {
		return err
	}
	out := itens[:0]
	for _, i := range itens {
		if i.ID != novo.ID {
			out = append(out, i)
		}
	}
	return gravaRegistro(append(out, novo))
}

func acha(itens []Item, id string) *Item {
	for i := range itens {
		if itens[i].ID == id {
			return &itens[i]
		}
	}
	return nil
}

// filhoDireto diz se caminho é um nome direto dentro de pasta, escrito sem
// voltas ("..", barras a mais).
func filhoDireto(caminho, pasta string) bool {
	return caminho != "" && filepath.IsAbs(caminho) && filepath.Clean(caminho) == caminho && filepath.Dir(caminho) == pasta
}

// confere recusa um item cujo caminho não é dos que o MkInstalar escreve. O
// registro é um arquivo que qualquer programa da pessoa pode mudar: o que
// está nele não manda apagar fora de ~/Aplicativos e dos atalhos.
func confere(i Item) error {
	if !IdValido(i.ID) {
		return base.Recusa("o registro tem um item de nome inválido: não mexo")
	}
	if !filhoDireto(i.Destino, base.Aplicativos()) || strings.HasPrefix(filepath.Base(i.Destino), ".") {
		return base.Recusa("o registro diz que %s está em %s, fora de %s: não apago", i.ID, i.Destino, base.Til(base.Aplicativos()))
	}
	if i.Atalho != nil {
		nome := filepath.Base(*i.Atalho)
		if !filhoDireto(*i.Atalho, base.Atalhos()) || !strings.HasPrefix(nome, "mkinstalar-") || !strings.HasSuffix(nome, ".desktop") {
			return base.Recusa("o registro diz que o atalho de %s é %s, que não é um atalho do MkInstalar: não apago", i.ID, *i.Atalho)
		}
	}
	if i.Icone != nil && !filhoDireto(*i.Icone, base.Icones()) {
		return base.Recusa("o registro diz que o ícone de %s é %s, fora de %s: não apago", i.ID, *i.Icone, base.Til(base.Icones()))
	}
	return nil
}

// apaga tira do disco o que um item (já conferido) tem. Um link simbólico é
// apagado como link, nunca seguido. Devolve o que existia e foi apagado.
func apaga(i Item) ([]string, error) {
	removidos := []string{}
	caminhos := []string{i.Destino}
	if i.Atalho != nil {
		caminhos = append(caminhos, *i.Atalho)
	}
	if i.Icone != nil {
		caminhos = append(caminhos, *i.Icone)
	}
	for _, c := range caminhos {
		if _, err := os.Lstat(c); err != nil {
			continue
		}
		// Uma pasta extraída pode ter subpastas sem permissão de escrita.
		_ = filepath.WalkDir(c, func(p string, d os.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				_ = os.Chmod(p, 0o700)
			}
			return nil
		})
		if err := os.RemoveAll(c); err != nil {
			return removidos, fmt.Errorf("não consegui apagar %s: %v", base.Til(c), err)
		}
		removidos = append(removidos, c)
	}
	return removidos, nil
}

// Desinstala apaga o que o item tem e o tira do registro.
func Desinstala(id string) ([]string, error) {
	if !IdValido(id) {
		return nil, fmt.Errorf("%q não é nome de app instalado", id)
	}
	itens, err := LeRegistro()
	if err != nil {
		return nil, err
	}
	item := acha(itens, id)
	if item == nil {
		return nil, fmt.Errorf("%s não foi instalado pelo MkInstalar", id)
	}
	if err := confere(*item); err != nil {
		return nil, err
	}
	removidos, err := apaga(*item)
	if err != nil {
		return nil, err
	}
	resto := []Item{}
	for _, i := range itens {
		if i.ID != id {
			resto = append(resto, i)
		}
	}
	if err := gravaRegistro(resto); err != nil {
		return removidos, err
	}
	atualizaMenu()
	return removidos, nil
}

// atualizaMenu roda o update-desktop-database na pasta dos atalhos, se o
// comando existir. Se falhar, tanto faz: o menu lê os atalhos de novo sozinho.
func atualizaMenu() {
	if base.Tem("update-desktop-database") {
		_, _ = base.Roda("update-desktop-database", 15*time.Second, nil, "-q", "--", base.Atalhos())
	}
}

// Existe diz se um caminho ainda está no disco (um link conta, mesmo quebrado).
func Existe(caminho string) bool {
	_, err := os.Lstat(caminho)
	return err == nil
}
