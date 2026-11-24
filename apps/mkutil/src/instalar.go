package main

import (
	"mkutil/internal/base"
	"mkutil/internal/instala"
)

func init() {
	registrar("inspecionar", cmdInspecionar)
	registrar("instalar", cmdInstalar)
	registrar("instalados", cmdInstalados)
	registrar("desinstalar", cmdDesinstalar)
}

// mkutil-motor inspecionar <arquivo>
func cmdInspecionar(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkutil-motor inspecionar <arquivo>")
	}
	ficha, err := instala.Inspeciona(pos[0])
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(ficha)
	return base.Certo
}

// mkutil-motor instalar <arquivo> [--substituir] [--sem-atalho] [--executavel <caminho>]
func cmdInstalar(args []string) int {
	pos, ops, err := opcoes(args, []string{"--executavel"}, []string{"--substituir", "--sem-atalho"})
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkutil-motor instalar <arquivo> [--substituir] [--sem-atalho] [--executavel <caminho>]")
	}
	fim, err := instala.Instala(pos[0], instala.Opcoes{Substituir: ops["--substituir"] != "", SemAtalho: ops["--sem-atalho"] != "",
		Executavel: ops["--executavel"]}, func(v any) { _ = base.Escreve(v) })
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(fim)
	return base.Certo
}

// instalado é um item do registro mais o que o disco diz agora.
type instalado struct {
	instala.Item
	Existe bool `json:"existe"`
}

// mkutil-motor instalados
func cmdInstalados(args []string) int {
	if len(args) != 0 {
		return base.Falha(base.Erro, "uso: mkutil-motor instalados")
	}
	itens, err := instala.LeRegistro()
	if err != nil {
		return base.FalhaDe(err)
	}
	lista := make([]instalado, 0, len(itens))
	for _, i := range itens {
		lista = append(lista, instalado{i, instala.Existe(i.Destino)})
	}
	_ = base.Escreve(struct {
		Itens []instalado `json:"itens"`
	}{lista})
	return base.Certo
}

// mkutil-motor desinstalar <id>
func cmdDesinstalar(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkutil-motor desinstalar <id>")
	}
	removidos, err := instala.Desinstala(pos[0])
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(struct {
		Ok        bool     `json:"ok"`
		ID        string   `json:"id"`
		Removidos []string `json:"removidos"`
	}{true, pos[0], removidos})
	return base.Certo
}
