package remoto

import (
	"os"
	"path/filepath"
	"strings"

	"mkfiles/internal/base"
)

// Host é um host do ~/.ssh/config: conectar a ele é montar sftp://apelido/
// (o gvfs usa o ssh do sistema, então a chave e o agente valem).
type Host struct {
	Apelido string  `json:"apelido"`
	Host    *string `json:"host"`
	Usuario *string `json:"usuario"`
	Porta   *string `json:"porta"`
}

// PastaDoSsh é ~/.ssh (ou MKFILES_SSH_DIR).
func PastaDoSsh() string {
	if v := os.Getenv("MKFILES_SSH_DIR"); filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	if h := base.Home(); h != "" {
		return filepath.Join(h, ".ssh")
	}
	return ""
}

// Hosts lê os `Host` do config e dos arquivos que ele inclui. Só entram os
// apelidos sem curinga; HostName, User e Port são os do próprio bloco. Só o
// config e os incluídos são abertos: nenhuma chave é lida.
func Hosts() []Host {
	hosts := []Host{}
	pasta := PastaDoSsh()
	if pasta == "" {
		return hosts
	}
	vistos := map[string]bool{}
	lidos := map[string]bool{}
	var le func(arquivo string, fundo int)
	le = func(arquivo string, fundo int) {
		if fundo > 8 || lidos[arquivo] {
			return
		}
		lidos[arquivo] = true
		info, err := os.Stat(arquivo)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return
		}
		b, err := os.ReadFile(arquivo)
		if err != nil {
			return
		}
		var doBloco []int // os hosts do bloco aberto
		for _, linha := range strings.Split(string(b), "\n") {
			linha = strings.TrimSpace(linha)
			if linha == "" || linha[0] == '#' {
				continue
			}
			chave, valor := palavra(linha)
			switch strings.ToLower(chave) {
			case "include":
				for _, padrao := range campos(valor) {
					if strings.HasPrefix(padrao, "~/") {
						padrao = filepath.Join(base.Home(), padrao[2:])
					} else if !filepath.IsAbs(padrao) {
						padrao = filepath.Join(pasta, padrao)
					}
					achados, _ := filepath.Glob(padrao)
					for _, a := range achados {
						le(a, fundo+1)
					}
				}
			case "host":
				doBloco = nil
				for _, apelido := range campos(valor) {
					if strings.ContainsAny(apelido, "*?!") || strings.HasPrefix(apelido, "-") || vistos[apelido] {
						continue
					}
					vistos[apelido] = true
					hosts = append(hosts, Host{Apelido: apelido})
					doBloco = append(doBloco, len(hosts)-1)
				}
			case "match":
				doBloco = nil
			case "hostname", "user", "port":
				v := valor
				if cs := campos(valor); len(cs) > 0 {
					v = cs[0]
				}
				for _, i := range doBloco {
					alvo := &hosts[i].Host
					switch strings.ToLower(chave) {
					case "user":
						alvo = &hosts[i].Usuario
					case "port":
						alvo = &hosts[i].Porta
					}
					if *alvo == nil && v != "" { // o primeiro valor é o que vale, como no ssh
						copia := v
						*alvo = &copia
					}
				}
			}
		}
	}
	le(filepath.Join(pasta, "config"), 0)
	return hosts
}

// palavra separa a chave do valor de uma linha do config ("Chave valor" ou
// "Chave=valor").
func palavra(linha string) (chave, valor string) {
	i := strings.IndexAny(linha, " \t=")
	if i < 0 {
		return linha, ""
	}
	return linha[:i], strings.TrimLeft(linha[i:], " \t=")
}

// campos separa um valor em palavras, respeitando aspas duplas.
func campos(valor string) []string {
	var lista []string
	var atual strings.Builder
	aspas, tem := false, false
	for _, r := range valor {
		switch {
		case r == '"':
			aspas, tem = !aspas, true
		case (r == ' ' || r == '\t') && !aspas:
			if tem {
				lista = append(lista, atual.String())
				atual.Reset()
				tem = false
			}
		case r == '#' && !aspas && !tem:
			return lista
		default:
			atual.WriteRune(r)
			tem = true
		}
	}
	if tem {
		lista = append(lista, atual.String())
	}
	return lista
}
