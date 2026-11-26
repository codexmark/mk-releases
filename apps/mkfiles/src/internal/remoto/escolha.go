package remoto

import (
	"regexp"
	"strings"
)

var opcaoNumerada = regexp.MustCompile(`(?m)^\s*\[(\d+)\]\s*(.+?)\s*$`)

// escolha traduz o "yes"/"no" que a janela manda numa confirmação para o que
// o gio espera. Quando a pergunta veio com opções numeradas ("[1] Sim",
// "[2] Não"), a resposta é o número da opção que quer dizer sim ou não; sem
// opções numeradas, a resposta vai como veio. Um "não" sem opção que o diga
// é desistir da montagem (segue = false). Qualquer outro texto (a pessoa
// digitou o número) passa direto. O formato das opções é dedução.
func escolha(texto, resposta string) (linha string, segue bool) {
	r := strings.ToLower(strings.TrimSpace(resposta))
	sim, nao := r == "yes" || r == "sim", r == "no" || r == "não" || r == "nao"
	if !sim && !nao {
		return resposta, true
	}
	opcoes := opcaoNumerada.FindAllStringSubmatch(texto, -1)
	if len(opcoes) == 0 {
		return resposta, true
	}
	palavras := []string{"yes", "sim", "continu", "connect", "conect", "accept", "aceit", "anyway", "mesmo assim", "log in", "entrar", "ok"}
	if nao {
		palavras = []string{"no", "não", "nao", "cancel", "abort", "desist", "reject", "recus"}
	}
	for _, o := range opcoes {
		rotulo := strings.ToLower(o[2])
		for _, p := range palavras {
			if rotulo == p || strings.HasPrefix(rotulo, p) || strings.Contains(rotulo, " "+p) {
				return o[1], true
			}
		}
	}
	if sim {
		return opcoes[0][1], true
	}
	return "", false
}
