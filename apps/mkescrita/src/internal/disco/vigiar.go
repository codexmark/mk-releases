package disco

import (
	"os"
	"time"
)

// Evento é uma linha do `vigiar`: o arquivo mudou por fora ou sumiu.
type Evento struct {
	// Evento: "mudou" ou "sumiu".
	Evento    string     `json:"evento"`
	Impressao *Impressao `json:"impressao"`
}

// Vigiar olha o arquivo a cada `intervalo` e chama `emite` quando o conteúdo
// dele não é mais o conhecido. `conhecido` é o sha256 que a janela tem (ou
// Ausente); vazio quer dizer "o que estiver lá agora". Só relê o arquivo
// quando o tamanho ou a data mudam. Para quando `segue` devolve false.
func Vigiar(caminho, conhecido string, intervalo time.Duration, emite func(Evento), segue func() bool) {
	var tamanho, mtime int64 = -1, -1
	primeira := conhecido == ""
	for segue() {
		info, err := os.Stat(caminho)
		switch {
		case err != nil || !info.Mode().IsRegular():
			if primeira {
				conhecido = Ausente
			} else if conhecido != Ausente {
				conhecido = Ausente
				emite(Evento{Evento: "sumiu"})
			}
			tamanho, mtime = -1, -1
		case info.Size() != tamanho || info.ModTime().UnixNano() != mtime:
			imp, err := Imprime(caminho)
			if err != nil || imp == nil {
				break // some no meio da leitura ou cresceu demais: a próxima volta decide
			}
			tamanho, mtime = info.Size(), info.ModTime().UnixNano()
			if primeira {
				conhecido = imp.Sha256
			} else if imp.Sha256 != conhecido {
				conhecido = imp.Sha256
				emite(Evento{Evento: "mudou", Impressao: imp})
			}
		}
		primeira = false
		time.Sleep(intervalo)
	}
}
