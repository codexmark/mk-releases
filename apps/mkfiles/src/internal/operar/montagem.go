package operar

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"mkfiles/internal/base"
)

// unidadeDoDisco diz em que unidade (o st_dev) um caminho está.
func unidadeDoDisco(_ string, info fs.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev)
	}
	return 0
}

// unidadeDe é o que o motor consulta; os testes trocam para simular um disco
// montado no meio de uma pasta.
var unidadeDe = unidadeDoDisco

// outroDiscoEm é o erro de quem ia apagar (ou levar embora) por cima de um
// disco montado.
func outroDiscoEm(ponto string) error {
	return errors.New("há outro disco montado em " + ponto + ": desmonte antes")
}

// montadoEm diz se a pasta é ela mesma um ponto de montagem: está na tabela
// de montagens, ou fica em outra unidade que a pasta de cima.
func montadoEm(caminho string, info fs.FileInfo) bool {
	if !info.IsDir() || caminho == "/" {
		return false
	}
	for _, m := range base.Montagens() {
		if m.Ponto == caminho {
			return true
		}
	}
	pai, err := os.Lstat(filepath.Dir(caminho))
	return err == nil && unidadeDe(filepath.Dir(caminho), pai) != unidadeDe(caminho, info)
}

// montadoDentro procura um disco montado na pasta ou dentro dela (pela tabela
// de montagens e pela unidade de cada subpasta, sem seguir links). Devolve o
// ponto, ou "" se não há.
func montadoDentro(caminho string) string {
	info, err := os.Lstat(caminho)
	if err != nil || !info.IsDir() {
		return ""
	}
	if montadoEm(caminho, info) {
		return caminho
	}
	for _, m := range base.Montagens() {
		if m.Ponto != caminho && base.Dentro(caminho, m.Ponto) {
			return m.Ponto
		}
	}
	return outraUnidadeDentro(caminho, unidadeDe(caminho, info))
}

func outraUnidadeDentro(pasta string, unidade uint64) string {
	entradas, _ := os.ReadDir(pasta)
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		c := filepath.Join(pasta, e.Name())
		info, err := os.Lstat(c)
		if err != nil || !info.IsDir() {
			continue
		}
		if unidadeDe(c, info) != unidade {
			return c
		}
		if achou := outraUnidadeDentro(c, unidade); achou != "" {
			return achou
		}
	}
	return ""
}
