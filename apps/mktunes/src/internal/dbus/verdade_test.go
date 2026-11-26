// SPDX-License-Identifier: GPL-3.0-or-later

package dbus

import (
	"os"
	"testing"
)

// Contra o barramento de sessão de verdade, só quando pedido
// (MKTUNES_BARRAMENTO_DE_VERDADE=1): o aperto de mão, duas perguntas ao
// próprio barramento e uma chamada a si mesmo, que vai e volta por ele. Não
// pede nome nenhum nem toca em outro programa.
func TestBarramentoDeVerdade(t *testing.T) {
	if os.Getenv("MKTUNES_BARRAMENTO_DE_VERDADE") != "1" {
		t.Skip("só com MKTUNES_BARRAMENTO_DE_VERDADE=1")
	}
	var c *Conexao
	c, erro := Liga(func(m *Mensagem) {
		c.Responde(m, "a{sv}", map[string]Variante{"eco": {"as", []string{m.Membro, m.Corpo[0].(string)}}, "n": {"x", int64(-7)}})
	})
	if erro != nil {
		t.Fatal(erro)
	}
	defer c.Fecha()
	if len(c.Nome) < 2 || c.Nome[0] != ':' {
		t.Fatalf("nome único: %q", c.Nome)
	}
	id, erro := c.Chama(nomeDoBarramento, caminhoDoBarramento, nomeDoBarramento, "GetId", "")
	if erro != nil || len(id.Corpo) != 1 {
		t.Fatalf("GetId: %v %v", id, erro)
	}
	nomes, erro := c.Chama(nomeDoBarramento, caminhoDoBarramento, nomeDoBarramento, "ListNames", "")
	if erro != nil {
		t.Fatal(erro)
	}
	lista, _ := nomes.Corpo[0].([]string)
	if len(lista) == 0 {
		t.Fatalf("ListNames: %v", nomes.Corpo)
	}
	dono, erro := c.Chama(nomeDoBarramento, caminhoDoBarramento, nomeDoBarramento, "NameHasOwner", "s", "org.freedesktop.DBus")
	if erro != nil || dono.Corpo[0] != true {
		t.Fatalf("NameHasOwner: %v %v", dono, erro)
	}
	eco, erro := c.Chama(c.Nome, "/dev/mk/mktunes/prova", "dev.mk.mktunes.Prova", "Eco", "sxd", "olá", int64(1), 0.5)
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, []any{map[string]Variante{"eco": {"as", []string{"Eco", "olá"}}, "n": {"x", int64(-7)}}}, eco.Corpo)
	igual(t, c.Nome, eco.Remetente)
	t.Logf("barramento %v, %d nomes, eu sou %s", id.Corpo[0], len(lista), c.Nome)
}
