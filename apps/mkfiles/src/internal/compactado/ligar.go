package compactado

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
	"mkfiles/internal/mime"
)

// O maior arquivo que se abre de dentro de um compactado (vai para um
// temporário, que costuma ser memória).
var limiteDeAbrir int64 = 2 << 30

// indice é o que há dentro de um compactado.
type indice struct {
	formato string
	es      []entrada
	cifrado bool
}

func (i *indice) totais() (itens int, tamanho int64, sabe bool) {
	sabe = true
	for _, e := range i.es {
		itens++
		if e.tamanho < 0 {
			sabe = false
		} else {
			tamanho += e.tamanho
		}
	}
	return
}

// sessao guarda o que a área lembra numa conversa: os últimos índices lidos
// (navegar pelas pastas de dentro não relê o arquivo) e a pasta dos
// temporários abertos.
type sessao struct {
	trava   sync.Mutex
	indices map[string]*indice
	ordem   []string
	pasta   string // a pasta dos abertos desta sessão ("" enquanto não há)
	raiz    string // onde ela mora
}

const marcaDaSessao = "sessao-"

func aleatorio() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ctxDe dá um contexto que acaba quando o fluxo é cancelado.
func ctxDe(p *base.Pedido) (context.Context, context.CancelFunc) {
	ctx, cancela := context.WithCancel(context.Background())
	go func() {
		select {
		case <-p.Parou():
			cancela()
		case <-ctx.Done():
		}
	}()
	return ctx, cancela
}

// leIndice lê (ou lembra) o índice de um compactado.
func (s *sessao) leIndice(ctx context.Context, caminho string, senha *string) (*indice, error) {
	info, err := os.Stat(caminho)
	if err != nil {
		return nil, errors.New("o arquivo não existe mais")
	}
	if info.IsDir() {
		return nil, erroDeFormato
	}
	chave := caminho + "\x00" + strconv.FormatInt(info.ModTime().UnixNano(), 10) + "\x00" + strconv.FormatInt(info.Size(), 10)
	s.trava.Lock()
	lembrado := s.indices[chave]
	s.trava.Unlock()
	if lembrado != nil {
		return lembrado, nil
	}
	formato := formatoDe(caminho)
	if formato == "" {
		return nil, erroDeFormato
	}
	if pacote := falta(formato); pacote != "" {
		return nil, semFerramenta(pacote)
	}
	ind := &indice{formato: formato}
	switch {
	case formato == "7z" || (formato == "rar" && o7z() != ""):
		ind.es, err = lista7z(ctx, caminho, senha)
	case formato == "rar":
		ind.es, err = listaRar(ctx, caminho)
	default:
		l := &leitura{ctx: ctx, caminho: caminho, formato: formato}
		err = l.percorre(func(e entrada, _ func() (io.ReadCloser, error)) error {
			ind.es = append(ind.es, e)
			return nil
		})
	}
	if err != nil {
		return nil, err
	}
	for _, e := range ind.es {
		ind.cifrado = ind.cifrado || e.cifrada
	}
	s.trava.Lock()
	if s.indices == nil {
		s.indices = map[string]*indice{}
	}
	s.indices[chave] = ind
	s.ordem = append(s.ordem, chave)
	if len(s.ordem) > 4 {
		delete(s.indices, s.ordem[0])
		s.ordem = s.ordem[1:]
	}
	s.trava.Unlock()
	return ind, nil
}

// itemDe monta o item do contrato de uma entrada de dentro.
func itemDe(arquivo string, doArquivo item.Item, interno, nome string, e entrada) base.Campos {
	c := base.Campos{"nome": nome, "caminho": arquivo + "/" + interno, "interno": interno,
		"tamanho": max(e.tamanho, 0), "mtime": doArquivo.Mtime, "modo": e.modo & 0o7777,
		"permissoes": item.Permissoes(e.modo & 0o7777), "dono": doArquivo.Dono, "grupo": doArquivo.Grupo,
		"oculto": strings.HasPrefix(nome, "."), "alvo": nil, "quebrado": false, "alvo_tipo": nil}
	if !e.mtime.IsZero() {
		c["mtime"] = e.mtime.Unix()
	}
	tipo := mime.Binario
	switch e.tipo {
	case ePasta:
		c["tipo"], tipo = item.Pasta, mime.Pasta
	case eLink:
		c["tipo"], tipo, c["alvo"] = item.Link, mime.Link, e.alvo
	case eEspecial:
		c["tipo"] = item.Especial
	default:
		c["tipo"] = item.Arquivo
		if m, achou := mime.DoNome(nome); achou {
			tipo = m
		}
	}
	c["mime"] = tipo
	c["icone"], c["icone_reserva"] = mime.Icone(tipo)
	return c
}

// filhos são os itens diretos de uma pasta de dentro ("" é a raiz). Uma pasta
// que o arquivo não declara, mas que tem algo dentro, aparece assim mesmo.
func filhos(arquivo string, doArquivo item.Item, ind *indice, dentro string) []base.Campos {
	prefixo := ""
	if dentro != "" {
		prefixo = dentro + "/"
	}
	achados := map[string]base.Campos{}
	for _, e := range ind.es {
		if e.ruim != "" || !strings.HasPrefix(e.nome, prefixo) {
			continue
		}
		resto := e.nome[len(prefixo):]
		nome, abaixo, fundo := strings.Cut(resto, "/")
		if nome == "" {
			continue
		}
		if fundo && abaixo != "" {
			if _, ja := achados[nome]; !ja {
				achados[nome] = itemDe(arquivo, doArquivo, prefixo+nome, nome, entrada{tipo: ePasta, modo: 0o755})
			}
			continue
		}
		achados[nome] = itemDe(arquivo, doArquivo, prefixo+nome, nome, e)
	}
	nomes := make([]string, 0, len(achados))
	for n := range achados {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	lista := make([]base.Campos, 0, len(nomes))
	for _, n := range nomes {
		lista = append(lista, achados[n])
	}
	return lista
}

type pedido struct {
	Caminho string  `json:"caminho"`
	Dentro  string  `json:"dentro"`
	Interno string  `json:"interno"`
	Senha   *string `json:"senha"`
}

func lePedido(p *base.Pedido) (pedido, bool) {
	var q pedido
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error(), base.Campos{"motivo": "erro"})
		return q, false
	}
	if q.Senha != nil && (*q.Senha == "" || strings.ContainsAny(*q.Senha, "\n\r\x00")) {
		p.Falha("a senha não pode ser vazia nem ter quebra de linha", base.Campos{"motivo": "senha", "campo": "senha"})
		return q, false
	}
	return q, true
}

// Ligar põe na sessão os pedidos formatos, compactado-info, compactado-listar
// e compactado-abrir. As tarefas extrair e compactar entram pelo trilho
// (operar.Registrar, no init deste pacote): basta o operar.Ligar.
func Ligar(s *base.Sessao) {
	se := &sessao{}
	s.AoFechar(se.fecha)

	s.Ligar("formatos", func(p *base.Pedido) {
		p.Responde(base.Campos{"extrair": disponibilidade(deExtrair), "compactar": disponibilidade(deCompactar)})
	})

	s.Ligar("compactado-info", func(p *base.Pedido) {
		q, ok := lePedido(p)
		if !ok {
			return
		}
		info, err := os.Stat(q.Caminho)
		if err != nil || info.IsDir() {
			p.Falha("o arquivo não existe mais", base.Campos{"motivo": "erro"})
			return
		}
		formato := formatoDe(q.Caminho)
		if formato == "" {
			p.Falha(erroDeFormato.frase, camposDoErro(erroDeFormato))
			return
		}
		r := base.Campos{"formato": formato, "disponivel": true, "falta": nil, "cifrado": false,
			"itens": nil, "tamanho": nil, "taxa": nil, "suspeito": false}
		if pacote := falta(formato); pacote != "" {
			r["disponivel"], r["falta"] = false, pacote
			p.Responde(r)
			return
		}
		// Só os formatos com índice são lidos aqui: um tar comprimido teria
		// de ser descompactado inteiro para se saber o tamanho.
		if formato == "zip" || formato == "7z" || formato == "rar" || formato == "tar" {
			ctx, cancela := context.WithTimeout(context.Background(), prazoDeListar)
			ind, err := se.leIndice(ctx, q.Caminho, q.Senha)
			cancela()
			var e *erroDe
			switch {
			case errors.As(err, &e) && e.motivo == "senha":
				r["cifrado"] = true
			case err != nil:
				p.Falha(err.Error(), camposDoErro(err))
				return
			default:
				r["cifrado"] = ind.cifrado
				itens, tamanho, sabe := ind.totais()
				r["itens"] = itens
				if sabe {
					r["tamanho"] = tamanho
					r["taxa"] = float64(tamanho) / float64(max(info.Size(), 1))
					r["suspeito"] = passaDoLimite(tamanho, info.Size())
				}
			}
			if formato == "zip" && r["cifrado"] == true && o7z() == "" {
				r["disponivel"], r["falta"] = false, "7zip"
			}
		}
		p.Responde(r)
	})

	s.Ligar("compactado-listar", func(p *base.Pedido) {
		q, ok := lePedido(p)
		if !ok {
			return
		}
		dentro := ""
		if q.Dentro != "" {
			var ruim string
			if dentro, ruim = nomeLimpo(q.Dentro); ruim != "" {
				p.Falha("pasta de dentro inválida: "+ruim, base.Campos{"motivo": "erro"})
				return
			}
		}
		doArquivo, err := item.De(q.Caminho, false)
		if err != nil {
			p.Falha("o arquivo não existe mais", base.Campos{"motivo": "nao-existe"})
			return
		}
		p.Fluxo(func(cancelado func() bool) {
			ctx, cancela := ctxDe(p)
			defer cancela()
			ind, err := se.leIndice(ctx, q.Caminho, q.Senha)
			if cancelado() {
				p.Responde(base.Campos{"itens": []base.Campos{}, "fim": true, "cancelada": true})
				return
			}
			if err != nil {
				p.Falha(err.Error(), camposDoErro(err))
				return
			}
			lista := filhos(q.Caminho, doArquivo, ind, dentro)
			lote := item.Lote()
			for len(lista) > lote {
				p.Responde(base.Campos{"itens": lista[:lote], "fim": false})
				lista = lista[lote:]
			}
			p.Responde(base.Campos{"itens": lista, "fim": true, "formato": ind.formato, "cifrado": ind.cifrado})
		})
	})

	s.Ligar("compactado-abrir", func(p *base.Pedido) {
		q, ok := lePedido(p)
		if !ok {
			return
		}
		interno, ruim := nomeLimpo(q.Interno)
		if ruim != "" {
			p.Falha("arquivo de dentro inválido: "+ruim, base.Campos{"motivo": "erro"})
			return
		}
		s.Vai(func() {
			caminho, err := se.abre(q.Caminho, interno, q.Senha)
			if err != nil {
				p.Falha(err.Error(), camposDoErro(err))
				return
			}
			p.Responde(base.Campos{"caminho": caminho})
		})
	})
}

// aPasta devolve (criando) a pasta dos temporários desta sessão:
// $XDG_RUNTIME_DIR/mkfiles/sessao-…, só da pessoa.
func (s *sessao) aPasta() (string, error) {
	s.trava.Lock()
	defer s.trava.Unlock()
	if s.pasta != "" {
		return s.pasta, nil
	}
	raiz := base.Cache("mkfiles", "abertos")
	if v := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(v) {
		raiz = filepath.Join(filepath.Clean(v), "mkfiles")
	}
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		return "", err
	}
	pasta := filepath.Join(raiz, marcaDaSessao+strconv.Itoa(os.Getpid())+"-"+aleatorio())
	if err := os.Mkdir(pasta, 0o700); err != nil {
		return "", err
	}
	s.pasta, s.raiz = pasta, raiz
	return pasta, nil
}

// fecha apaga os temporários que a sessão abriu.
func (s *sessao) fecha() {
	s.trava.Lock()
	pasta, raiz := s.pasta, s.raiz
	s.pasta = ""
	s.trava.Unlock()
	limpa(pasta, raiz, marcaDaSessao)
}

// abre extrai um arquivo de dentro, só ele, para um temporário da sessão.
func (s *sessao) abre(arquivo, interno string, senha *string) (string, error) {
	ctx, cancela := context.WithTimeout(context.Background(), prazoDeExtrair)
	defer cancela()
	ind, err := s.leIndice(ctx, arquivo, senha)
	if err != nil {
		return "", err
	}
	var alvo *entrada
	for i := range ind.es {
		if ind.es[i].ruim == "" && ind.es[i].nome == interno {
			alvo = &ind.es[i]
		}
	}
	if alvo == nil {
		return "", errors.New("isso não está mais dentro do arquivo")
	}
	if alvo.tipo != eArquivo {
		return "", errors.New("só dá para abrir um arquivo")
	}
	if alvo.tamanho > limiteDeAbrir {
		return "", errGrande
	}
	pasta, err := s.aPasta()
	if err != nil {
		return "", err
	}
	lugar := filepath.Join(pasta, aleatorio())
	if err := os.Mkdir(lugar, 0o700); err != nil {
		return "", err
	}
	nome := interno[strings.LastIndexByte(interno, '/')+1:]
	info, _ := os.Stat(arquivo)
	var doArquivo int64
	if info != nil {
		doArquivo = info.Size()
	}
	quieto := semAndamento(func() bool { return ctx.Err() != nil })
	deFora := ind.formato == "7z" || ind.formato == "rar" || (ind.formato == "zip" && alvo.cifrada)
	if deFora {
		if ind.formato == "zip" && o7z() == "" {
			return "", erroCifradoZip
		}
		if senha == nil && alvo.cifrada {
			return "", erroPedeSenha
		}
		if _, tamanho, sabe := ind.totais(); sabe && (tamanho > limiteDeAbrir || passaDoLimite(tamanho, doArquivo)) {
			return "", errGrande
		}
		tudo := filepath.Join(pasta, "t-"+aleatorio())
		if err := os.Mkdir(tudo, 0o700); err != nil {
			return "", err
		}
		defer limpa(tudo, pasta, "t-")
		if err := extraiDeFora(ctx, arquivo, ind.formato, senha, tudo); err != nil {
			return "", err
		}
		if err := varre(tudo, quieto, arquivo, doArquivo, false); err != nil {
			return "", err
		}
		origem := filepath.Join(tudo, filepath.FromSlash(interno))
		if la, err := os.Lstat(origem); err != nil || !la.Mode().IsRegular() {
			return "", errors.New("a ferramenta não extraiu esse arquivo")
		}
		if err := os.Rename(origem, filepath.Join(lugar, nome)); err != nil {
			return "", err
		}
		return filepath.Join(lugar, nome), nil
	}
	achou := errors.New("achou")
	x := &extrator{raiz: lugar, arquivo: arquivo, doArquivo: doArquivo, a: quieto, teto: limiteDeAbrir,
		quer: func(n string) (string, bool) { return nome, n == interno }}
	l := &leitura{ctx: ctx, caminho: arquivo, formato: ind.formato}
	err = l.percorre(func(e entrada, abre func() (io.ReadCloser, error)) error {
		if e.ruim != "" || e.nome != interno || e.tipo != eArquivo {
			return nil
		}
		if err := x.recebe(e, abre); err != nil {
			return err
		}
		return achou
	})
	if err != nil && err != achou {
		limpa(lugar, pasta, filepath.Base(lugar))
		return "", err
	}
	if _, e := os.Lstat(filepath.Join(lugar, nome)); e != nil {
		if quieto.temAviso() {
			return "", errors.New(quieto.primeiro())
		}
		return "", errors.New("não deu para extrair esse arquivo")
	}
	return filepath.Join(lugar, nome), nil
}
