package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"mkpacotes/internal/acoes"
	"mkpacotes/internal/base"
)

func init() { registrar("estado", cmdEstado) }

func existe(comando string) bool {
	_, err := exec.LookPath(comando)
	return err == nil
}

// sincronizadoEm: a hora da base sincronizada mais recente (sync/*.db).
func sincronizadoEm() *int64 {
	bases, _ := filepath.Glob(base.PacmanDB("sync", "*.db"))
	var maior *int64
	for _, b := range bases {
		info, err := os.Stat(b)
		if err != nil {
			continue
		}
		if s := info.ModTime().Unix(); maior == nil || s > *maior {
			maior = &s
		}
	}
	return maior
}

// mkpacotes-coletor estado
func cmdEstado(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor estado")
	}
	var cache any
	if c, _, err := acoes.MedeCache(); err == nil {
		cache = map[string]any{"arquivos": c.Arquivos, "bytes": c.Bytes}
	}
	mkUpdate := existe("mk-update")
	if !mkUpdate {
		info, err := os.Stat(base.Mk("bin", "mk-update"))
		mkUpdate = err == nil && !info.IsDir() && info.Mode()&0o111 != 0
	}
	_ = base.Escreve(map[string]any{
		"travado":           acoes.Travado(),
		"sincronizado_em_s": sincronizadoEm(),
		"cache":             cache,
		"comandos":          map[string]bool{"pacman": existe("pacman"), "pkexec": existe("pkexec"), "yay": existe("yay"), "mk_update": mkUpdate},
	})
	return base.Certo
}
