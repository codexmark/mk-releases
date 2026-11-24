package aur

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func num(v int64) *int64 { return &v }

// aurFalso sobe um AUR de mentira e faz dele o MKPACOTES_AUR do teste.
// `pedidos` guarda o caminho e a consulta de cada chamada.
func aurFalso(t *testing.T, responde http.HandlerFunc) *[]string {
	t.Helper()
	var pedidos []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pedidos = append(pedidos, r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		responde(w, r)
	}))
	t.Cleanup(s.Close)
	t.Setenv("MKPACOTES_AUR", s.URL)
	return &pedidos
}

const respostaYay = `{"resultcount":2,"type":"search","version":5,"results":[
 {"Name":"yay","Version":"12.5.7-1","Description":"Yet another yogurt","URL":"https://github.com/Jguer/yay","NumVotes":2500,
  "Popularity":25.5,"OutOfDate":null,"Maintainer":"jguer","LastModified":1795000000,
  "Depends":["pacman>6.1","git"],"MakeDepends":["go>=1.24"],"OptDepends":["sudo: privilege elevation"],"License":["GPL-3.0-or-later"],
  "Provides":["yay"],"Conflicts":["yay-bin"]},
 {"Name":"yay-bin","Version":"12.5.7-1","Description":null,"URL":null,"NumVotes":300,"Popularity":3,"OutOfDate":1794000000,
  "Maintainer":null,"LastModified":1794500000}
]}`

func TestBusca_PedeAoAUREDevolveOsPacotes(t *testing.T) {
	pedidos := aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, respostaYay) })
	lista, err := Busca("yay gurte")
	if err != nil {
		t.Fatal(err)
	}
	quer := []Pacote{
		{Nome: "yay", Versao: "12.5.7-1", Descricao: "Yet another yogurt", URL: "https://github.com/Jguer/yay", Votos: 2500, Popularidade: 25.5,
			Mantenedor: "jguer", ModificadoEm: num(1795000000), Depende: []string{"pacman>6.1", "git"}, Construcao: []string{"go>=1.24"},
			Opcionais: []string{"sudo: privilege elevation"}, Licencas: []string{"GPL-3.0-or-later"}, Fornece: []string{"yay"}, Conflita: []string{"yay-bin"}},
		{Nome: "yay-bin", Versao: "12.5.7-1", Votos: 300, Popularidade: 3, DesatualizadoDesde: num(1794000000), ModificadoEm: num(1794500000)},
	}
	if !reflect.DeepEqual(lista, quer) {
		t.Errorf("veio %+v\nquer %+v", lista, quer)
	}
	if quer := []string{"/rpc/v5/search/yay%20gurte?by=name-desc"}; !reflect.DeepEqual(*pedidos, quer) {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestBusca_TextoEstranhoVaiEscapadoNoCaminho(t *testing.T) {
	pedidos := aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"type":"search","results":[]}`) })
	if _, err := Busca("a/b?c=d&e #f"); err != nil {
		t.Fatal(err)
	}
	// A barra, a interrogação e o jogo da velha não viram caminho, consulta nem âncora.
	if quer := []string{"/rpc/v5/search/a%2Fb%3Fc=d&e%20%23f?by=name-desc"}; !reflect.DeepEqual(*pedidos, quer) {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestInfo_VariosNomesNumPedidoSo(t *testing.T) {
	pedidos := aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, respostaYay) })
	lista, err := Info([]string{"yay", "yay-bin"})
	if err != nil || len(lista) != 2 || lista[0].Nome != "yay" {
		t.Fatalf("veio %v, %v", lista, err)
	}
	if quer := []string{"/rpc/v5/info?arg%5B%5D=yay&arg%5B%5D=yay-bin"}; !reflect.DeepEqual(*pedidos, quer) {
		t.Errorf("pediu %v", *pedidos)
	}
	if lista, err := Info(nil); err != nil || len(lista) != 0 || len(*pedidos) != 1 {
		t.Errorf("sem nome nenhum não há pedido: %v %v", lista, err)
	}
}

func TestInfo_MuitosNomesVaoEmLotes(t *testing.T) {
	pedidos := aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"type":"multiinfo","results":[]}`) })
	nomes := make([]string, 250)
	for i := range nomes {
		nomes[i] = fmt.Sprintf("pacote-%d", i)
	}
	if _, err := Info(nomes); err != nil {
		t.Fatal(err)
	}
	if len(*pedidos) != 3 {
		t.Errorf("250 nomes em lotes de 100 são 3 pedidos, foram %d", len(*pedidos))
	}
}

func TestInfo_NomeQueNaoENomeDePacoteNaoSaiDaMaquina(t *testing.T) {
	pedidos := aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, respostaYay) })
	if _, err := Info([]string{"yay", "a&b=c"}); err == nil || !strings.Contains(err.Error(), "não é nome de pacote") {
		t.Errorf("erro: %v", err)
	}
	if len(*pedidos) != 0 {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestErros_ViramFrase(t *testing.T) {
	aurFalso(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"type":"error","error":"Too many package results.","results":[]}`)
	})
	if _, err := Busca("a"); err == nil || err.Error() != "o AUR recusou a busca: Too many package results." {
		t.Errorf("erro do AUR: %v", err)
	}
	aurFalso(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fora do ar", http.StatusServiceUnavailable)
	})
	if _, err := Busca("yay"); err == nil || err.Error() != "o AUR respondeu com o código 503" {
		t.Errorf("código: %v", err)
	}
	aurFalso(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<html>portal do hotel</html>`) })
	if _, err := Busca("yay"); err == nil || err.Error() != "o AUR respondeu algo que não entendi" {
		t.Errorf("resposta torta: %v", err)
	}
}

func TestSemRede_ErroEmFraseENaoDemora(t *testing.T) {
	// Um servidor que já fechou: a porta recusa a conexão.
	s := httptest.NewServer(http.NotFoundHandler())
	s.Close()
	t.Setenv("MKPACOTES_AUR", s.URL)
	antes := time.Now()
	_, err := Busca("yay")
	if err == nil || err.Error() != "não consegui falar com o AUR (sem rede?)" {
		t.Errorf("erro: %v", err)
	}
	if time.Since(antes) > 3*time.Second {
		t.Errorf("demorou %v", time.Since(antes))
	}
}

func TestServidorQueNaoRespondeEstouraOPrazo(t *testing.T) {
	antes := Prazo
	Prazo = 150 * time.Millisecond
	t.Cleanup(func() { Prazo = antes })
	solta := make(chan struct{})
	aurFalso(t, func(w http.ResponseWriter, r *http.Request) { <-solta })
	t.Cleanup(func() { close(solta) })
	if _, err := Busca("yay"); err == nil || err.Error() != "não consegui falar com o AUR (sem rede?)" {
		t.Errorf("erro: %v", err)
	}
}
