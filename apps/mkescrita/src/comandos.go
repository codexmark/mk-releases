package main

import (
	"errors"
	"os"
	"strconv"
	"time"

	"mkescrita/internal/base"
	"mkescrita/internal/disco"
	"mkescrita/internal/marca"
)

func init() {
	registrar("abrir", abrir)
	registrar("salvar", salvar)
	registrar("rascunho", rascunho)
	registrar("vigiar", vigiar)
	registrar("html", html)
	registrar("contar", contar)
	registrar("estrutura", estrutura)
	registrar("exportar", exportar)
	registrar("preferencias", preferencias)
}

// responde escreve a resposta e devolve Certo (ou o erro, com o código dele).
func responde(v any, err error) int {
	if err != nil {
		return base.FalhaDe(err)
	}
	if err := base.Escreve(v); err != nil {
		return base.Erro
	}
	return base.Certo
}

// um devolve o único posicional esperado.
func um(pos []string, oQue string) (string, error) {
	if len(pos) != 1 || pos[0] == "" {
		return "", errors.New(oQue)
	}
	return pos[0], nil
}

func abrir(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return base.FalhaDe(err)
	}
	arquivo, err := um(pos, "uso: mkescrita-motor abrir <arquivo>")
	if err != nil {
		return base.FalhaDe(err)
	}
	return responde(disco.Abrir(arquivo))
}

func salvar(args []string) int {
	pos, ops, err := opcoes(args, []string{"--esperado", "--fim-de-linha", "--bytes"}, []string{"--forcar", "--quebra-final"})
	if err != nil {
		return base.FalhaDe(err)
	}
	arquivo, err := um(pos, "uso: mkescrita-motor salvar [--esperado <sha256|ausente>] [--forcar] <arquivo> (o texto vem pelo stdin)")
	if err != nil {
		return base.FalhaDe(err)
	}
	if f := ops["--fim-de-linha"]; f != "" && f != "lf" && f != "crlf" && f != "misto" {
		return base.Falha(base.Erro, "--fim-de-linha é lf, crlf ou misto, veio %q", f)
	}
	texto, err := base.LeTexto(ops["--bytes"])
	if err != nil {
		return base.FalhaDe(err)
	}
	salvo, err := disco.Salvar(disco.Pedido{Caminho: arquivo, Texto: texto, Esperado: ops["--esperado"], Forcar: ops["--forcar"] != "",
		FimDeLinha: ops["--fim-de-linha"], QuebraFinal: ops["--quebra-final"] != ""})
	var conflito *disco.Conflito
	if errors.As(err, &conflito) {
		// Não é erro: é a pergunta que a janela vai fazer à pessoa.
		_ = base.Escreve(map[string]any{"conflito": conflito.Motivo, "frase": conflito.Error(), "impressao": conflito.Impressao})
		return base.Recusado
	}
	return responde(salvo, err)
}

func rascunho(args []string) int {
	const frase = "uso: mkescrita-motor rascunho <guardar <id> | listar | ler <id> | apagar <id>>"
	if len(args) == 0 {
		return base.Falha(base.Erro, frase)
	}
	pos, ops, err := opcoes(args[1:], []string{"--origem", "--base", "--bytes"}, nil)
	if err != nil {
		return base.FalhaDe(err)
	}
	switch args[0] {
	case "listar":
		itens, err := disco.ListarRascunhos()
		return responde(map[string]any{"itens": itens}, err)
	case "guardar":
		id, err := um(pos, frase)
		if err != nil {
			return base.FalhaDe(err)
		}
		texto, err := base.LeTexto(ops["--bytes"])
		if err != nil {
			return base.FalhaDe(err)
		}
		// O dono do rascunho é quem chamou o motor: a janela.
		return responde(disco.GuardarRascunho(id, ops["--origem"], ops["--base"], texto, os.Getppid()))
	case "ler":
		id, err := um(pos, frase)
		if err != nil {
			return base.FalhaDe(err)
		}
		return responde(disco.LerRascunho(id))
	case "apagar":
		id, err := um(pos, frase)
		if err != nil {
			return base.FalhaDe(err)
		}
		return responde(map[string]any{"ok": true, "id": id}, disco.ApagarRascunho(id))
	}
	return base.Falha(base.Erro, frase)
}

func vigiar(args []string) int {
	pos, ops, err := opcoes(args, []string{"--esperado", "--intervalo-ms"}, nil)
	if err != nil {
		return base.FalhaDe(err)
	}
	arquivo, err := um(pos, "uso: mkescrita-motor vigiar [--esperado <sha256|ausente>] [--intervalo-ms N] <arquivo>")
	if err != nil {
		return base.FalhaDe(err)
	}
	intervalo := 1000
	if v := ops["--intervalo-ms"]; v != "" {
		if intervalo, err = strconv.Atoi(v); err != nil || intervalo < 10 || intervalo > 600000 {
			return base.Falha(base.Erro, "--intervalo-ms vai de 10 a 600000, veio %q", v)
		}
	}
	// Vigia enquanto a janela que o abriu viver e houver para quem escrever.
	pai := os.Getppid()
	vivo := true
	disco.Vigiar(arquivo, ops["--esperado"], time.Duration(intervalo)*time.Millisecond, func(e disco.Evento) {
		if base.Escreve(e) != nil {
			vivo = false
		}
	}, func() bool { return vivo && os.Getppid() == pai })
	return base.Certo
}

func html(args []string) int {
	pos, ops, err := opcoes(args, []string{"--titulo", "--bytes"}, []string{"--pagina"})
	if err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkescrita-motor html [--pagina] [--titulo <título>] (o Markdown vem pelo stdin)")
	}
	texto, err := base.LeTexto(ops["--bytes"])
	if err != nil {
		return base.FalhaDe(err)
	}
	if ops["--pagina"] != "" {
		return responde(map[string]string{"html": marca.Pagina(string(texto), ops["--titulo"])}, nil)
	}
	return responde(map[string]string{"html": marca.HTML(string(texto))}, nil)
}

func contar(args []string) int {
	pos, ops, err := opcoes(args, []string{"--bytes"}, nil)
	if err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkescrita-motor contar (o texto vem pelo stdin)")
	}
	texto, err := base.LeTexto(ops["--bytes"])
	if err != nil {
		return base.FalhaDe(err)
	}
	return responde(marca.Contar(string(texto)), nil)
}

func estrutura(args []string) int {
	pos, ops, err := opcoes(args, []string{"--bytes"}, nil)
	if err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkescrita-motor estrutura (o Markdown vem pelo stdin)")
	}
	texto, err := base.LeTexto(ops["--bytes"])
	if err != nil {
		return base.FalhaDe(err)
	}
	return responde(map[string]any{"titulos": marca.Estrutura(string(texto))}, nil)
}

func exportar(args []string) int {
	const frase = "uso: mkescrita-motor exportar html [--forcar] [--titulo <título>] <saída> (o Markdown vem pelo stdin)"
	pos, ops, err := opcoes(args, []string{"--titulo", "--bytes"}, []string{"--forcar"})
	if err != nil {
		return base.FalhaDe(err)
	}
	if len(pos) != 2 || pos[0] != "html" || pos[1] == "" {
		return base.Falha(base.Erro, frase)
	}
	texto, err := base.LeTexto(ops["--bytes"])
	if err != nil {
		return base.FalhaDe(err)
	}
	return responde(disco.Exportar(pos[1], []byte(marca.Pagina(string(texto), ops["--titulo"])), ops["--forcar"] != ""))
}

func preferencias(args []string) int {
	const frase = "uso: mkescrita-motor preferencias <ler | gravar> (gravar recebe um objeto JSON pelo stdin)"
	if len(args) != 1 {
		return base.Falha(base.Erro, frase)
	}
	switch args[0] {
	case "ler":
		return responde(disco.LerPrefs(), nil)
	case "gravar":
		mudancas, err := base.LeTexto("")
		if err != nil {
			return base.FalhaDe(err)
		}
		return responde(disco.GravarPrefs(mudancas))
	}
	return base.Falha(base.Erro, frase)
}
