package acoes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mkpacotes/internal/alpm"
	"mkpacotes/internal/aur"
	"mkpacotes/internal/base"
	"mkpacotes/internal/pacman"
)

// Entra é um pacote que a ação instala.
type Entra struct {
	Nome   string  `json:"nome"`
	Versao *string `json:"versao"`
	Origem *string `json:"origem"`
	Baixar *int64  `json:"baixar_bytes"`
	Pedido bool    `json:"pedido"`
}

// Sai é um pacote que a ação remove.
type Sai struct {
	Nome    string `json:"nome"`
	Versao  string `json:"versao"`
	Tamanho int64  `json:"tamanho_bytes"`
	Pedido  bool   `json:"pedido"`
}

// Cache é o retrato da pasta de pacotes baixados.
type Cache struct {
	Arquivos int   `json:"arquivos"`
	Bytes    int64 `json:"bytes"`
	Sobram   int   `json:"sobram"` // quantos ficam depois de limpar
}

// Plano é o ensaio de uma ação: o contrato do subcomando `plano`.
type Plano struct {
	Acao       string     `json:"acao"`
	Pedidos    []string   `json:"pedidos"`
	Entra      []Entra    `json:"entra"`
	Sai        []Sai      `json:"sai"`
	Baixar     *int64     `json:"baixar_bytes"`
	Liberar    *int64     `json:"liberar_bytes"`
	Comandos   [][]string `json:"comandos"`
	Avisos     []string   `json:"avisos"`
	Travado    bool       `json:"travado"`
	NadaAFazer bool       `json:"nada_a_fazer"`
	Cache      *Cache     `json:"cache"`
}

// Acoes são as ações do contrato, na ordem do uso.
func Acoes() []string { return []string{"instalar", "remover", "orfaos", "cache", "atualizar"} }

// Planeja faz o ensaio de uma ação, sem mudar nada e sem root.
func Planeja(acao string, nomes []string, mesmoAssim bool) (*Plano, error) {
	nomes, err := limpa(nomes)
	if err != nil {
		return nil, err
	}
	p := &Plano{Acao: acao, Pedidos: nomes, Entra: []Entra{}, Sai: []Sai{}, Comandos: [][]string{}, Avisos: []string{}}
	if p.Pedidos == nil {
		p.Pedidos = []string{}
	}
	switch acao {
	case "instalar", "remover":
		if len(nomes) == 0 {
			return nil, errors.New("faltou dizer o pacote")
		}
	case "orfaos", "cache", "atualizar":
		if len(nomes) > 0 {
			return nil, fmt.Errorf("a ação %s não leva pacote", acao)
		}
	default:
		return nil, fmt.Errorf("não conheço a ação %q (as que há: %s)", acao, strings.Join(Acoes(), ", "))
	}
	switch acao {
	case "instalar":
		err = p.instalar()
	case "remover":
		err = p.remover(mesmoAssim)
	case "orfaos":
		err = p.orfaos(mesmoAssim)
	case "cache":
		err = p.cache()
	case "atualizar":
		err = p.atualizar()
	}
	if err != nil {
		return nil, err
	}
	p.Travado = Travado()
	return p, nil
}

func (p *Plano) instalar() error {
	repos, err := pacman.Repos()
	if err != nil {
		return err
	}
	local, err := alpm.Le()
	if err != nil {
		return err
	}
	var oficiais, resto []string
	for _, n := range p.Pedidos {
		if _, ok := repos[n]; ok {
			oficiais = append(oficiais, n)
		} else {
			resto = append(resto, n)
		}
	}
	pedido := map[string]bool{}
	for _, n := range p.Pedidos {
		pedido[n] = true
	}
	var baixar int64
	if len(oficiais) > 0 {
		itens, err := pacman.EnsaioInstalar(oficiais)
		if err != nil {
			return fmt.Errorf("o pacman não consegue instalar: %v", err)
		}
		vem := map[string]bool{}
		for _, it := range itens {
			versao, repo, b := it.Versao, it.Repo, it.Bytes
			p.Entra = append(p.Entra, Entra{Nome: it.Nome, Versao: &versao, Origem: &repo, Baixar: &b, Pedido: pedido[it.Nome]})
			vem[it.Nome] = true
			baixar += b
		}
		var falta []string
		for _, n := range oficiais {
			if !vem[n] {
				p.Avisos = append(p.Avisos, n+" já está instalado e em dia")
			} else {
				falta = append(falta, n)
			}
		}
		if len(falta) > 0 {
			p.Comandos = append(p.Comandos, append([]string{"pkexec", "pacman", "-S", "--needed", "--noconfirm", "--"}, falta...))
		}
	}
	doAUR, err := p.doAUR(resto, local)
	if err != nil {
		return err
	}
	if len(doAUR) > 0 {
		p.Avisos = append(p.Avisos, "do AUR: o yay baixa e compila "+lista(doAUR)+" nesta máquina, com a receita de quem mantém o pacote (não é do Arch nem do Mk)")
		if _, err := exec.LookPath("yay"); err != nil {
			p.Avisos = append(p.Avisos, "para instalar do AUR é preciso o yay, e não o achei nesta máquina")
		}
		p.Comandos = append(p.Comandos, append([]string{"yay", "-S", "--needed", "--noconfirm", "--sudo", "pkexec", "--"}, doAUR...))
	} else {
		p.Baixar = &baixar
	}
	p.NadaAFazer = len(p.Comandos) == 0
	return nil
}

// doAUR procura no AUR o que os repositórios não têm e põe no plano o que
// entra: o pacote e as dependências que faltam nesta máquina.
func (p *Plano) doAUR(nomes []string, local *alpm.Base) ([]string, error) {
	if len(nomes) == 0 {
		return nil, nil
	}
	fichas, err := aur.Info(nomes)
	if err != nil {
		verbo := "não está"
		if len(nomes) > 1 {
			verbo = "não estão"
		}
		return nil, fmt.Errorf("%s %s nos repositórios, e não deu para procurar no AUR: %v", lista(nomes), verbo, err)
	}
	porNome := map[string]aur.Pacote{}
	for _, f := range fichas {
		porNome[f.Nome] = f
	}
	var instalar []string
	ja := map[string]bool{}
	for _, e := range p.Entra {
		ja[e.Nome] = true
	}
	for _, n := range nomes {
		if _, ok := porNome[n]; !ok {
			return nil, fmt.Errorf("não achei %s nos repositórios nem no AUR", n)
		}
	}
	origem := "aur"
	var faltam []Entra
	for _, n := range nomes {
		f := porNome[n]
		if inst, ok := local.Pacotes[n]; ok && alpm.Vercmp(f.Versao, inst.Versao) <= 0 {
			p.Avisos = append(p.Avisos, n+" já está instalado e em dia")
			continue
		}
		versao := f.Versao
		p.Entra = append(p.Entra, Entra{Nome: n, Versao: &versao, Origem: &origem, Pedido: true})
		ja[n] = true
		instalar = append(instalar, n)
		for _, dep := range append(append([]string{}, f.Depende...), f.Construcao...) {
			nome := alpm.NomeDe(dep)
			if nome == "" || ja[nome] || local.Satisfeito(dep) {
				continue
			}
			ja[nome] = true
			faltam = append(faltam, Entra{Nome: nome})
		}
	}
	// Uma dependência que também foi pedida já está na lista.
	for _, f := range faltam {
		repetido := false
		for _, e := range p.Entra {
			repetido = repetido || e.Nome == f.Nome
		}
		if !repetido {
			p.Entra = append(p.Entra, f)
		}
	}
	return instalar, nil
}

func (p *Plano) remover(mesmoAssim bool) error {
	local, err := alpm.Le()
	if err != nil {
		return err
	}
	return p.removerDe(local, mesmoAssim)
}

func (p *Plano) removerDe(local *alpm.Base, mesmoAssim bool) error {
	saindo := map[string]bool{}
	for _, n := range p.Pedidos {
		if _, ok := local.Pacotes[n]; !ok {
			return fmt.Errorf("%s não está instalado", n)
		}
		saindo[n] = true
	}
	// Antes de qualquer comando: os protegidos pedidos pelo nome.
	if quais := protegidosEm(p.Pedidos); len(quais) > 0 && !mesmoAssim {
		return recusaProtegidos(quais)
	}
	for _, n := range p.Pedidos {
		var dependem []string
		for _, q := range local.RequeridoPor(n) {
			if !saindo[q] {
				dependem = append(dependem, q)
			}
		}
		if len(dependem) == 1 {
			return fmt.Errorf("não dá para remover %s: %s depende dele", n, dependem[0])
		}
		if len(dependem) > 1 {
			if len(dependem) > 8 {
				dependem = append(dependem[:8], fmt.Sprintf("outros %d", len(dependem)-8))
			}
			return fmt.Errorf("não dá para remover %s: %s dependem dele", n, lista(dependem))
		}
	}
	itens, err := pacman.EnsaioRemover(p.Pedidos)
	if err != nil {
		return fmt.Errorf("o pacman não consegue remover: %v", err)
	}
	var nomes []string
	var liberar int64
	for _, it := range itens {
		s := Sai{Nome: it.Nome, Versao: it.Versao, Pedido: saindo[it.Nome]}
		if inst, ok := local.Pacotes[it.Nome]; ok {
			s.Tamanho = inst.Tamanho
		}
		liberar += s.Tamanho
		p.Sai = append(p.Sai, s)
		nomes = append(nomes, it.Nome)
	}
	// E os que sairiam de carona, como dependência que sobrou.
	if quais := protegidosEm(nomes); len(quais) > 0 {
		if !mesmoAssim {
			return recusaProtegidos(quais)
		}
		for _, q := range quais {
			p.Avisos = append(p.Avisos, q+" faz parte da base do sistema: sem ele a máquina pode não ligar ou não abrir a sessão")
		}
	}
	p.Liberar = &liberar
	p.NadaAFazer = len(p.Sai) == 0
	if !p.NadaAFazer {
		p.Comandos = append(p.Comandos, append([]string{"pkexec", "pacman", "-Rns", "--noconfirm", "--"}, p.Pedidos...))
	}
	return nil
}

func recusaProtegidos(quais []string) error {
	if len(quais) == 1 {
		return recusa("não removo %s: faz parte da base do sistema, e sem ele a máquina pode não ligar ou não abrir a sessão (só com --mesmo-assim)", quais[0])
	}
	return recusa("não removo %s: fazem parte da base do sistema, e sem eles a máquina pode não ligar ou não abrir a sessão (só com --mesmo-assim)", lista(quais))
}

func (p *Plano) orfaos(mesmoAssim bool) error {
	local, err := alpm.Le()
	if err != nil {
		return err
	}
	protegidos := Protegidos()
	p.Pedidos = nil
	for _, n := range local.Orfaos() {
		switch {
		case alpm.ValidaNome(n) != nil:
			p.Avisos = append(p.Avisos, fmt.Sprintf("%q é órfão, mas o nome dele não é aceito: fica", n))
		case protegidos[n]:
			p.Avisos = append(p.Avisos, n+" é órfão, mas faz parte da base do sistema: fica")
		default:
			p.Pedidos = append(p.Pedidos, n)
		}
	}
	if len(p.Pedidos) == 0 {
		return errors.New("não há pacote órfão para remover")
	}
	return p.removerDe(local, mesmoAssim)
}

// MedeCache conta os pacotes baixados (e as assinaturas deles) e quanto
// deles não é de nenhum instalado: é o que o `pacman -Sc` tira.
func MedeCache() (c *Cache, liberar int64, err error) {
	entradas, err := os.ReadDir(base.Cache())
	if err != nil {
		return nil, 0, fmt.Errorf("não consegui ler o cache de pacotes (%s)", base.Cache())
	}
	instalados := map[string]bool{}
	if local, err := alpm.Le(); err == nil {
		for _, inst := range local.Pacotes {
			instalados[inst.Nome+"-"+inst.Versao] = true
		}
	}
	c = &Cache{}
	for _, e := range entradas {
		id, ok := idDoArquivo(e.Name())
		if !ok || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		c.Arquivos++
		c.Bytes += info.Size()
		if instalados[id] {
			c.Sobram++
		} else {
			liberar += info.Size()
		}
	}
	return c, liberar, nil
}

// idDoArquivo tira de "bash-5.3.20-1-x86_64.pkg.tar.zst(.sig)" o
// "bash-5.3.20-1"; false se não é arquivo de pacote.
func idDoArquivo(nome string) (string, bool) {
	nome = strings.TrimSuffix(nome, ".sig")
	i := strings.Index(nome, ".pkg.tar")
	if i < 0 {
		return "", false
	}
	nome = nome[:i]
	j := strings.LastIndexByte(nome, '-')
	if j <= 0 {
		return "", false
	}
	return nome[:j], true
}

func (p *Plano) cache() error {
	c, liberar, err := MedeCache()
	if err != nil {
		return err
	}
	p.Cache, p.Liberar = c, &liberar
	p.NadaAFazer = c.Arquivos == c.Sobram
	if !p.NadaAFazer {
		p.Comandos = append(p.Comandos, []string{"pkexec", "pacman", "-Sc", "--noconfirm"})
	}
	return nil
}

// Os terminais que o Mk sabe abrir rodando um comando (os mesmos do Maré), na
// ordem de preferência, e como cada um recebe o título e o comando.
var terminais = []string{"ghostty", "kitty", "foot", "alacritty", "wezterm"}

const tituloDaAtualizacao = "Mk · atualizações"

func linhaDoTerminal(nome, comando string) []string {
	switch nome {
	case "kitty", "foot":
		return []string{nome, "--title=" + tituloDaAtualizacao, comando}
	case "alacritty":
		return []string{nome, "--title", tituloDaAtualizacao, "-e", comando}
	case "wezterm":
		// Sem --always-new-process o comando volta na hora, com a janela
		// nascendo num wezterm já aberto.
		return []string{nome, "start", "--always-new-process", "--", comando}
	}
	// O ghostty abre com a config do Mk, como no Maré.
	return []string{"ghostty", "--config-default-files=false", "--config-file=?" + base.Config("mk", "ghostty", "config"),
		"--title=" + tituloDaAtualizacao, "-e", comando}
}

// terminal escolhe o terminal: o de MKPACOTES_TERMINAL, o dos Ajustes do Mk
// ("apps.terminal") ou o primeiro que houver. Só entram os que o Mk conhece.
func terminal() (string, error) {
	var ajustes map[string]any
	if b, err := os.ReadFile(base.Config("mk", "ajustes.json")); err == nil {
		_ = json.Unmarshal(b, &ajustes)
	}
	doAjuste, _ := ajustes["apps.terminal"].(string)
	for _, pedido := range append([]string{os.Getenv("MKPACOTES_TERMINAL"), doAjuste}, terminais...) {
		for _, t := range terminais {
			if t != pedido {
				continue
			}
			if _, err := exec.LookPath(t); err == nil {
				return t, nil
			}
		}
	}
	return "", errors.New("não achei um terminal para abrir a atualização (ghostty, kitty, foot, alacritty ou wezterm)")
}

func (p *Plano) atualizar() error {
	mk, err := exec.LookPath("mk-update")
	if err != nil {
		mk = base.Mk("bin", "mk-update")
		if info, err := os.Stat(mk); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return errors.New("não achei o mk-update: este sistema não parece ser um Mk")
		}
	}
	if abs, err := filepath.Abs(mk); err == nil {
		mk = abs
	}
	t, err := terminal()
	if err != nil {
		return err
	}
	p.Comandos = append(p.Comandos, linhaDoTerminal(t, mk))
	p.Avisos = append(p.Avisos, "a atualização abre num terminal: é o mk-update, que atualiza o Mk, o Arch e o AUR e pede a senha lá")
	return nil
}
