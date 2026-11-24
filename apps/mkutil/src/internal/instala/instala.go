package instala

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mkutil/internal/base"
)

// Ficha é a resposta do `inspecionar`: o que o arquivo é e o que instalar faria.
type Ficha struct {
	Arquivo      string           `json:"arquivo"`
	Tipo         string           `json:"tipo"`
	ID           string           `json:"id"`
	Nome         string           `json:"nome"`
	Versao       *string          `json:"versao"`
	Descricao    *string          `json:"descricao"`
	Tamanho      int64            `json:"tamanho_bytes"`
	Destino      *string          `json:"destino"`
	Atalho       *string          `json:"atalho"`
	JaInstalado  bool             `json:"ja_instalado"`
	PrecisaSenha bool             `json:"precisa_senha"`
	Plano        []string         `json:"plano"`
	Avisos       []string         `json:"avisos"`
	AppImage     *FichaAppImage   `json:"appimage"`
	Pacote       *FichaPacote     `json:"pacote"`
	Compactado   *FichaCompactado `json:"compactado"`

	formato string
	dentro  deDentro
}

func naoVazio(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Onde o pacman mora. Vai por extenso para o pkexec: com o nome solto, ele
// procuraria no PATH de quem chamou, e o que rodaria como root seria o
// primeiro "pacman" que aparecesse.
var caminhoDoPacman = "/usr/bin/pacman"

// Inspeciona olha o arquivo e monta a ficha. Não muda nada no disco.
func Inspeciona(arquivo string) (*Ficha, error) {
	abs, err := filepath.Abs(arquivo)
	if err != nil {
		return nil, fmt.Errorf("não entendi o caminho %s", arquivo)
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("não achei o arquivo %s", base.Til(abs))
	}
	defer f.Close()
	return inspeciona(f, abs)
}

// inspeciona monta a ficha de um arquivo já aberto. `abs` é o caminho de
// onde ele veio (o nome dá o id); `f` é de onde se lê.
func inspeciona(f *os.File, abs string) (*Ficha, error) {
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s não é um arquivo", base.Til(abs))
	}
	nomeDoArquivo := filepath.Base(abs)
	_, tipo, formato := extensaoDe(nomeDoArquivo)
	id, nome, versao := Identifica(nomeDoArquivo)
	ficha := &Ficha{Arquivo: abs, ID: id, Nome: nome, Versao: versao, Tamanho: fi.Size(), Plano: []string{}, Avisos: []string{}, formato: formato}
	tipoAI := tipoDeAppImage(f)
	switch {
	case tipoAI != 0:
		tipo = "appimage"
	case tipo == "appimage":
		return nil, errors.New("o arquivo tem nome de AppImage mas não é um (o download pode ter ficado pela metade)")
	case tipo == "deb":
		return nil, errors.New("um .deb é pacote do Debian e do Ubuntu, e o Mk é Arch: procure o app no MkPacotes, ou baixe o AppImage ou o .tar.gz dele")
	case tipo == "rpm":
		return nil, errors.New("um .rpm é pacote do Fedora e do openSUSE, e o Mk é Arch: procure o app no MkPacotes, ou baixe o AppImage ou o .tar.gz dele")
	case tipo == "flatpak":
		return nil, errors.New("isto é do Flatpak, que tem o instalador dele: use o MkPacotes ou `flatpak install`")
	case tipo == "":
		return nil, errors.New("não sei instalar este tipo de arquivo: o MkInstalar entende AppImage, pacote do Arch (.pkg.tar.zst) e .tar.gz, .tar.xz, .tar.zst, .tar.bz2 e .zip")
	}
	ficha.Tipo = tipo

	itens, errRegistro := LeRegistro()
	if errRegistro != nil {
		ficha.Avisos = append(ficha.Avisos, errRegistro.Error())
	}

	switch tipo {
	case "appimage":
		ai := &FichaAppImage{Tipo: tipoAI, Fuse: true}
		ficha.AppImage = ai
		if tipoAI == 2 {
			ficha.dentro = leDeDentro(f)
			ai.TemDesktop, ai.TemIcone = ficha.dentro.temAtalho, ficha.dentro.icone != nil
			if !ai.TemDesktop {
				motivo := ficha.dentro.motivo
				if motivo == "" {
					motivo = "não há um .desktop na raiz"
				}
				ficha.Avisos = append(ficha.Avisos, "não deu para ler o atalho de dentro do AppImage ("+motivo+"): o atalho sai só com o nome do arquivo")
			}
		} else {
			ficha.Avisos = append(ficha.Avisos, "AppImage do tipo 1 (o formato antigo): o atalho sai só com o nome do arquivo e um ícone genérico")
		}
		if n := ficha.dentro.desktop["Name"]; n != "" {
			ficha.Nome = n
		}
		ficha.Descricao = naoVazio(ficha.dentro.desktop["Comment"])
		if ficha.Versao == nil {
			ficha.Versao = naoVazio(ficha.dentro.desktop["X-AppImage-Version"])
		}
		if precisaDeFuse2(f) && !temFuse2() {
			ai.Fuse = false
			ficha.Avisos = append(ficha.Avisos, "este AppImage precisa do FUSE 2 (libfuse.so.2), que não está instalado: sem ele o app não abre. Instale o pacote fuse2")
		}
		destino := base.Aplicativos(id + ".AppImage")
		atalho := base.Atalhos("mkinstalar-" + id + ".desktop")
		ficha.Destino, ficha.Atalho = &destino, &atalho
		ficha.Plano = []string{"copiar para " + base.Til(destino), "marcar como executável", "criar o atalho no menu"}
	case "pacote":
		bruto, err := lePKGINFO(abs, formato)
		if err != nil {
			return nil, err
		}
		p, err := interpretaPKGINFO(bruto)
		if err != nil {
			return nil, err
		}
		p.Instalado = versaoInstalada(p.Nome)
		ficha.Pacote = &p
		ficha.ID, ficha.Nome, ficha.Versao, ficha.Descricao = FazId(p.Nome), p.Nome, &p.Versao, p.Descricao
		ficha.PrecisaSenha = true
		passo := "instalar " + p.Nome + " " + p.Versao + " com o pacman (pede a senha de administrador)"
		if p.Instalado != nil {
			passo = "trocar " + p.Nome + " " + *p.Instalado + " por " + p.Versao + " com o pacman (pede a senha de administrador)"
			if *p.Instalado == p.Versao {
				passo = p.Nome + " " + p.Versao + " já está instalado: o pacman não faz nada"
			}
		}
		ficha.Plano = []string{passo}
		if n := len(p.Dependencias); n > 0 {
			ficha.Plano = append(ficha.Plano, fmt.Sprintf("o pacman baixa o que faltar das %d dependências", n))
		}
		ficha.Avisos = append(ficha.Avisos, "um pacote roda scripts como administrador ao instalar: só instale de onde você confia")
		return ficha, nil
	case "compactado":
		c, err := inspecionaCompactado(abs, formato, id)
		if err != nil {
			return nil, err
		}
		ficha.Compactado = c
		destino := base.Aplicativos(id)
		ficha.Destino = &destino
		ficha.Plano = []string{fmt.Sprintf("extrair %d itens para %s", c.Entradas, base.Til(destino))}
		if c.ExecutavelSugerido != nil {
			atalho := base.Atalhos("mkinstalar-" + id + ".desktop")
			ficha.Atalho = &atalho
			ficha.Plano = append(ficha.Plano, "criar o atalho no menu para "+*c.ExecutavelSugerido)
		} else {
			ficha.Avisos = append(ficha.Avisos, "não achei um executável dentro do arquivo: a pasta é extraída, mas sem atalho no menu")
		}
	}

	ficha.JaInstalado = acha(itens, ficha.ID) != nil
	if ficha.JaInstalado {
		ficha.Plano = append([]string{"tirar a versão que o MkInstalar instalou antes"}, ficha.Plano...)
	} else if _, err := os.Lstat(*ficha.Destino); err == nil {
		ficha.Avisos = append(ficha.Avisos, "já existe "+base.Til(*ficha.Destino)+", que não foi o MkInstalar que pôs: tire de lá antes de instalar")
	}
	return ficha, nil
}

// Opcoes do `instalar`.
type Opcoes struct {
	Substituir bool
	SemAtalho  bool
	Executavel string
}

// Fim é a última linha do `instalar`.
type Fim struct {
	Fase    string   `json:"fase"`
	Ok      bool     `json:"ok"`
	ID      string   `json:"id"`
	Tipo    string   `json:"tipo"`
	Nome    string   `json:"nome"`
	Destino *string  `json:"destino"`
	Atalho  *string  `json:"atalho"`
	Avisos  []string `json:"avisos"`
}

type passo struct {
	Fase    string   `json:"fase"`
	Pct     *float64 `json:"pct,omitempty"`
	Entrada string   `json:"entrada,omitempty"`
	Linha   string   `json:"linha,omitempty"`
}

// andamento solta as fases de porcentagem sem inundar: uma a cada ponto.
type andamento struct {
	solta  func(any)
	fase   string
	ultimo float64
}

func (a *andamento) vai(pct float64, entrada string) {
	if pct-a.ultimo < 1 {
		return
	}
	a.ultimo = pct
	p := float64(int(pct*10)) / 10
	a.solta(passo{Fase: a.fase, Pct: &p, Entrada: entrada})
}

// Instala faz o que a ficha planejou. `solta` recebe cada fase (menos a
// última, que é o Fim devolvido).
func Instala(arquivo string, op Opcoes, solta func(any)) (*Fim, error) {
	ficha, err := Inspeciona(arquivo)
	if err != nil {
		return nil, err
	}
	if ficha.Tipo == "pacote" {
		return instalaPacote(ficha, solta)
	}
	itens, err := LeRegistro()
	if err != nil {
		return nil, err
	}
	antigo := acha(itens, ficha.ID)
	if antigo != nil {
		if !op.Substituir {
			return nil, fmt.Errorf("%s já está instalado: desinstale antes, ou peça para substituir", ficha.Nome)
		}
		if err := confere(*antigo); err != nil {
			return nil, err
		}
	} else if _, err := os.Lstat(*ficha.Destino); err == nil {
		return nil, fmt.Errorf("já existe %s, que não foi o MkInstalar que pôs: tire de lá antes de instalar", base.Til(*ficha.Destino))
	}
	if err := os.MkdirAll(base.Aplicativos(), 0o755); err != nil {
		return nil, fmt.Errorf("não consegui criar %s: %v", base.Til(base.Aplicativos()), err)
	}
	if ficha.Tipo == "appimage" {
		return instalaAppImage(ficha, antigo, op, solta)
	}
	return instalaCompactado(ficha, antigo, op, solta)
}

func instalaAppImage(ficha *Ficha, antigo *Item, op Opcoes, solta func(any)) (fim *Fim, err error) {
	origem, err := os.Open(ficha.Arquivo)
	if err != nil {
		return nil, fmt.Errorf("não consegui abrir o arquivo: %v", err)
	}
	defer origem.Close()
	tmp, err := os.CreateTemp(base.Aplicativos(), ".mkinstalar-*.parcial")
	if err != nil {
		return nil, fmt.Errorf("não consegui escrever em %s: %v", base.Til(base.Aplicativos()), err)
	}
	criados := []string{tmp.Name()}
	defer func() {
		tmp.Close()
		if err != nil {
			for _, c := range criados {
				_ = os.Remove(c)
			}
		}
	}()
	and := &andamento{solta: solta, fase: "copiando"}
	buf := make([]byte, 1<<20)
	var copiados int64
	for {
		n, e := origem.Read(buf)
		if n > 0 {
			if _, err = tmp.Write(buf[:n]); err != nil {
				return nil, fmt.Errorf("não consegui copiar (disco cheio?): %v", err)
			}
			copiados += int64(n)
			if ficha.Tamanho > 0 {
				and.vai(min(100, float64(copiados)*100/float64(ficha.Tamanho)), "")
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("não consegui ler o arquivo: %v", e)
		}
	}
	if err = tmp.Chmod(0o755); err != nil {
		return nil, err
	}
	// O que vai para o atalho é lido da cópia: é ela que fica instalada, e o
	// original pode ter sido trocado depois da inspeção.
	nova, err := inspeciona(tmp, ficha.Arquivo)
	if err != nil {
		return nil, err
	}
	if nova.Tipo != "appimage" {
		return nil, errors.New("o arquivo mudou enquanto era copiado: tente de novo")
	}
	if err = tmp.Sync(); err != nil {
		return nil, fmt.Errorf("não consegui copiar (disco cheio?): %v", err)
	}
	if antigo != nil {
		if _, err = apaga(*antigo); err != nil {
			return nil, err
		}
	}
	destino := *ficha.Destino
	if err = os.Rename(tmp.Name(), destino); err != nil {
		return nil, fmt.Errorf("não consegui pôr o app em %s: %v", base.Til(destino), err)
	}
	criados = []string{destino}

	item := Item{ID: ficha.ID, Tipo: "appimage", Nome: nova.Nome, Versao: nova.Versao, Origem: filepath.Base(ficha.Arquivo),
		Destino: destino, InstaladoEm: time.Now().Unix(), Tamanho: copiados}
	icone := ""
	if nova.dentro.icone != nil {
		icone = base.Icones(ficha.ID + nova.dentro.extensao)
		if err = gravaInteiro(icone, nova.dentro.icone, 0o644); err != nil {
			return nil, fmt.Errorf("não consegui guardar o ícone: %v", err)
		}
		criados = append(criados, icone)
		item.Icone = &icone
	}
	if !op.SemAtalho {
		solta(passo{Fase: "atalho"})
		var conteudo []byte
		if conteudo, err = fazAtalho(atalhoNovo{ID: ficha.ID, Nome: nova.Nome, Executavel: destino, Icone: icone, Original: nova.dentro.desktop}); err != nil {
			return nil, err
		}
		atalho := *ficha.Atalho
		if err = gravaInteiro(atalho, conteudo, 0o644); err != nil {
			return nil, fmt.Errorf("não consegui gravar o atalho: %v", err)
		}
		criados = append(criados, atalho)
		item.Atalho = &atalho
	}
	if err = anota(item); err != nil {
		return nil, err
	}
	atualizaMenu()
	return &Fim{Fase: "fim", Ok: true, ID: item.ID, Tipo: item.Tipo, Nome: item.Nome, Destino: &item.Destino, Atalho: item.Atalho, Avisos: nova.Avisos}, nil
}

// apagaPasta tira uma pasta mesmo que tenha subpastas sem permissão de escrita.
func apagaPasta(pasta string) {
	_ = filepath.WalkDir(pasta, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(p, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(pasta)
}

func instalaCompactado(ficha *Ficha, antigo *Item, op Opcoes, solta func(any)) (fim *Fim, err error) {
	c := ficha.Compactado
	tmp, err := os.MkdirTemp(base.Aplicativos(), ".mkinstalar-*.parcial")
	if err != nil {
		return nil, fmt.Errorf("não consegui escrever em %s: %v", base.Til(base.Aplicativos()), err)
	}
	criados := []string{}
	defer func() {
		if err != nil {
			apagaPasta(tmp)
			for _, c := range criados {
				apagaPasta(c)
			}
		}
	}()
	tira := ""
	if c.PastaUnica != nil {
		tira = *c.PastaUnica
	}
	and := &andamento{solta: solta, fase: "extraindo"}
	escritos, err := extrai(ficha.Arquivo, ficha.formato, tmp, tira, and.vai)
	if err != nil {
		return nil, err
	}

	// Para quem é o atalho: o pedido, ou o sugerido. Tem de ser um arquivo
	// comum de dentro da pasta.
	executavel := op.Executavel
	if executavel == "" && c.ExecutavelSugerido != nil {
		executavel = *c.ExecutavelSugerido
	}
	avisos := ficha.Avisos
	if executavel != "" && !op.SemAtalho {
		if executavel, err = nomeSeguro(executavel); err != nil || executavel == "" {
			return nil, base.Recusa("o executável pedido não é um caminho de dentro do arquivo")
		}
		raiz, e := os.OpenRoot(tmp)
		if e != nil {
			return nil, e
		}
		fi, e := raiz.Stat(executavel)
		if e == nil && fi.Mode().IsRegular() {
			e = raiz.Chmod(executavel, fi.Mode().Perm()|0o111)
		} else if e == nil {
			e = errors.New("não é um arquivo")
		}
		raiz.Close()
		if e != nil {
			return nil, fmt.Errorf("o executável pedido (%s) não existe dentro do arquivo", executavel)
		}
	}

	if antigo != nil {
		if _, err = apaga(*antigo); err != nil {
			return nil, err
		}
	}
	destino := *ficha.Destino
	if err = os.Chmod(tmp, 0o755); err != nil {
		return nil, err
	}
	if err = os.Rename(tmp, destino); err != nil {
		return nil, fmt.Errorf("não consegui pôr o app em %s: %v", base.Til(destino), err)
	}
	criados = append(criados, destino)

	item := Item{ID: ficha.ID, Tipo: "compactado", Nome: ficha.Nome, Versao: ficha.Versao, Origem: filepath.Base(ficha.Arquivo),
		Destino: destino, InstaladoEm: time.Now().Unix(), Tamanho: escritos}
	if executavel != "" && !op.SemAtalho {
		solta(passo{Fase: "atalho"})
		var original map[string]string
		if len(c.Atalhos) > 0 {
			original = leArquivoDeDentro(destino, c.Atalhos[0])
		}
		if n := original["Name"]; n != "" {
			item.Nome = n
		}
		var conteudo []byte
		conteudo, err = fazAtalho(atalhoNovo{ID: ficha.ID, Nome: item.Nome, Executavel: filepath.Join(destino, filepath.FromSlash(executavel)),
			Icone: achaIcone(destino, ficha.ID, original["Icon"]), Original: original})
		if err != nil {
			return nil, err
		}
		atalho := base.Atalhos("mkinstalar-" + ficha.ID + ".desktop")
		if err = gravaInteiro(atalho, conteudo, 0o644); err != nil {
			return nil, fmt.Errorf("não consegui gravar o atalho: %v", err)
		}
		criados = append(criados, atalho)
		item.Atalho = &atalho
	}
	if err = anota(item); err != nil {
		return nil, err
	}
	atualizaMenu()
	return &Fim{Fase: "fim", Ok: true, ID: item.ID, Tipo: item.Tipo, Nome: item.Nome, Destino: &item.Destino, Atalho: item.Atalho, Avisos: avisos}, nil
}

// leArquivoDeDentro lê um .desktop da pasta extraída (sem sair dela).
func leArquivoDeDentro(pasta, rel string) map[string]string {
	raiz, err := os.OpenRoot(pasta)
	if err != nil {
		return nil
	}
	defer raiz.Close()
	f, err := raiz.Open(rel)
	if err != nil {
		return nil
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(f, maximoDoAtalho))
	if err != nil {
		return nil
	}
	return leDesktop(b)
}

// achaIcone procura um ícone na pasta extraída: o que o atalho de dentro
// nomeia, o que tem o nome do app, ou um "icon.png". O mais raso ganha.
func achaIcone(pasta, id, doAtalho string) string {
	querem := map[string]int{}
	n := 0
	poe := func(nome string) {
		if nome == "" || strings.ContainsAny(nome, "/\\") {
			return
		}
		for _, ext := range []string{".png", ".svg"} {
			if _, ok := querem[strings.ToLower(nome)+ext]; !ok {
				querem[strings.ToLower(nome)+ext] = n
				n++
			}
		}
	}
	poe(strings.TrimSuffix(strings.TrimSuffix(doAtalho, ".png"), ".svg"))
	poe(id)
	poe("icon")
	poe("logo")
	melhor, nota, vistos := "", 1<<30, 0
	_ = filepath.WalkDir(pasta, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(pasta, p)
		fundo := strings.Count(rel, string(filepath.Separator))
		if vistos++; vistos > 20000 {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if fundo >= 6 {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if ordem, ok := querem[strings.ToLower(d.Name())]; ok {
			if n := fundo*100 + ordem; n < nota {
				melhor, nota = p, n
			}
		}
		return nil
	})
	return melhor
}

// instalaPacote chama o pacman pelo pkexec: o polkit (o Maré é o agente) pede
// a senha. O motor nunca vê a senha e nunca chama o sudo.
func instalaPacote(ficha *Ficha, solta func(any)) (*Fim, error) {
	pkexec, err := exec.LookPath("pkexec")
	if err != nil {
		return nil, errors.New("não achei o pkexec (o polkit) nesta máquina: sem ele não dá para pedir a senha de administrador")
	}
	cmd := exec.Command(pkexec, caminhoDoPacman, "-U", "--needed", "--noconfirm", "--", ficha.Arquivo)
	saida, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("não consegui rodar o pkexec: %v", err)
	}
	ultima := ""
	linhas := bufio.NewScanner(saida)
	linhas.Buffer(make([]byte, 64<<10), 1<<20)
	for linhas.Scan() {
		if linha := limpaValor(linhas.Text()); linha != "" {
			ultima = linha
			solta(passo{Fase: "pacman", Linha: linha})
		}
	}
	err = cmd.Wait()
	var saiu *exec.ExitError
	if errors.As(err, &saiu) {
		if c := saiu.ExitCode(); c == 126 || c == 127 {
			return nil, errors.New("a senha não foi dada: nada foi instalado")
		}
		if ultima != "" {
			return nil, fmt.Errorf("o pacman não instalou: %s", ultima)
		}
		return nil, fmt.Errorf("o pacman não instalou (código %d)", saiu.ExitCode())
	}
	if err != nil {
		return nil, fmt.Errorf("o pacman não instalou: %v", err)
	}
	return &Fim{Fase: "fim", Ok: true, ID: ficha.ID, Tipo: "pacote", Nome: ficha.Nome, Avisos: []string{}}, nil
}
