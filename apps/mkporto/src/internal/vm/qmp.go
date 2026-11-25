package vm

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

// PrazoDoQMP é quanto se espera por cada passo da conversa com o QEMU.
var PrazoDoQMP = 3 * time.Second

// Uma linha do QMP não passa disto (a saudação e as respostas são curtas).
const linhaDoQMP = 256 * 1024

// QMP é uma conversa aberta com o QEMU de uma máquina.
type QMP struct {
	conexao net.Conn
	leitor  *bufio.Reader
}

// mensagem é o que o QEMU manda: a saudação, uma resposta ou um evento.
type mensagem struct {
	QMP    json.RawMessage `json:"QMP"`
	Return json.RawMessage `json:"return"`
	Error  *struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
	Event string `json:"event"`
}

// AbreQMP conecta no socket da máquina, lê a saudação e negocia
// (qmp_capabilities): daí em diante o QEMU aceita comandos.
func AbreQMP(socket string) (*QMP, error) {
	conexao, err := net.DialTimeout("unix", socket, PrazoDoQMP)
	if err != nil {
		return nil, fmt.Errorf("o QEMU não atende no socket: %v", err)
	}
	q := &QMP{conexao: conexao, leitor: bufio.NewReaderSize(conexao, 64*1024)}
	_ = conexao.SetDeadline(time.Now().Add(PrazoDoQMP))
	saudacao, err := q.le()
	if err != nil || saudacao.QMP == nil {
		q.Fecha()
		return nil, errors.New("o QEMU não se apresentou no socket")
	}
	if _, err := q.Manda("qmp_capabilities"); err != nil {
		q.Fecha()
		return nil, err
	}
	return q, nil
}

// le lê uma mensagem (uma linha de JSON).
func (q *QMP) le() (*mensagem, error) {
	var linha []byte
	for {
		pedaco, prefixo, err := q.leitor.ReadLine()
		if err != nil {
			return nil, err
		}
		linha = append(linha, pedaco...)
		if len(linha) > linhaDoQMP {
			return nil, errors.New("linha comprida demais")
		}
		if !prefixo {
			break
		}
	}
	var m mensagem
	if err := json.Unmarshal(linha, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Manda executa um comando e devolve o `return` dele. Os eventos que o QEMU
// soltar no meio (uma máquina viva fala sozinha) são ignorados.
func (q *QMP) Manda(comando string) (json.RawMessage, error) {
	_ = q.conexao.SetDeadline(time.Now().Add(PrazoDoQMP))
	pedido, _ := json.Marshal(map[string]string{"execute": comando})
	if _, err := q.conexao.Write(append(pedido, '\n')); err != nil {
		return nil, fmt.Errorf("o QEMU não recebeu o %s: %v", comando, err)
	}
	for {
		m, err := q.le()
		if err != nil {
			return nil, fmt.Errorf("o QEMU não respondeu ao %s", comando)
		}
		switch {
		case m.Error != nil:
			return nil, fmt.Errorf("o QEMU recusou o %s: %s", comando, m.Error.Desc)
		case m.Return != nil:
			return m.Return, nil
		}
		// Um evento (ou outra coisa): não é a resposta.
	}
}

// Fecha encerra a conversa.
func (q *QMP) Fecha() { _ = q.conexao.Close() }

// Pergunta abre a conversa, manda um comando e fecha.
func Pergunta(socket, comando string) (json.RawMessage, error) {
	q, err := AbreQMP(socket)
	if err != nil {
		return nil, err
	}
	defer q.Fecha()
	return q.Manda(comando)
}

// Estado pergunta ao QEMU como a máquina está (query-status): "ligada" ou
// "pausada".
func Estado(socket string) (string, error) {
	resposta, err := Pergunta(socket, "query-status")
	if err != nil {
		return "", err
	}
	var s struct {
		Running bool   `json:"running"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(resposta, &s); err != nil {
		return "", errors.New("não entendi o que o QEMU disse do estado")
	}
	if s.Status == "paused" && !s.Running {
		return "pausada", nil
	}
	return "ligada", nil
}
