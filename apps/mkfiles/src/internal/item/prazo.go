package item

import (
	"os"
	"syscall"
	"time"

	"mkfiles/internal/base"
)

// PrazoDeAbrir é quanto o motor espera uma pasta abrir (ou um item responder)
// antes de desistir: um remoto sem rede pode não responder nunca.
var PrazoDeAbrir = 20 * time.Second

// abre é o os.Open; os testes trocam para simular um remoto que não responde.
var abre = os.Open

// abrePasta abre um caminho e confere que é uma pasta.
func abrePasta(caminho string) (*os.File, error) {
	pasta, err := abre(caminho)
	if err != nil {
		return nil, err
	}
	info, err := pasta.Stat()
	if err == nil && !info.IsDir() {
		err = syscall.ENOTDIR
	}
	if err != nil {
		pasta.Close()
		return nil, err
	}
	return pasta, nil
}

// comPrazo roda fn por fora e espera o resultado até PrazoDeAbrir (ou até
// parou fechar). noPrazo é false se desistiu: fn fica para trás e o que ela
// devolver é jogado fora (um arquivo aberto é fechado).
func comPrazo[T any](parou <-chan struct{}, fn func() (T, error)) (valor T, err error, noPrazo bool) {
	type resultado struct {
		valor T
		err   error
	}
	veio := make(chan resultado, 1)
	largou := make(chan struct{})
	go func() {
		v, e := fn()
		select {
		case veio <- resultado{v, e}:
		default:
		}
		select {
		case <-largou:
			// Ninguém mais espera: não deixa um descritor aberto.
			if f, ok := any(v).(*os.File); ok && f != nil {
				f.Close()
			}
		default:
		}
	}()
	relogio := time.NewTimer(PrazoDeAbrir)
	defer relogio.Stop()
	select {
	case r := <-veio:
		return r.valor, r.err, true
	case <-relogio.C:
	case <-parou:
	}
	close(largou)
	// Entre o fim de fn e o fechar de largou: se o valor chegou, é nosso.
	select {
	case r := <-veio:
		if f, ok := any(r.valor).(*os.File); ok && f != nil {
			f.Close()
		}
	default:
	}
	return valor, nil, false
}

func fraseDoPrazo(caminho string) string {
	if _, nome, ok := base.Remoto(caminho); ok {
		return nome + " não respondeu a tempo"
	}
	return "a pasta não respondeu a tempo"
}

// MotivoEm é o Motivo que sabe do caminho: num remoto, um erro de conexão
// perdida vira o motivo "conexao" e a frase "a conexão com X caiu".
func MotivoEm(caminho string, err error) (motivo, frase string) {
	if frase, caiu := base.ConexaoCaiu(caminho, err); caiu {
		return "conexao", frase
	}
	return Motivo(err)
}
