pragma Singleton
import QtQuick
import Quickshell
import "Pacotes.js" as Pacotes

// Uma ação em andamento, do pedido ao fim: o ensaio (`plano`), a confirmação
// da pessoa, a execução (`agir`) com o registro do progresso. A Folha mostra
// este estado; as páginas só chamam planejar().
//   fase: ""            nada aberto
//         "planejando"  esperando o ensaio
//         "plano"       o plano está na tela, esperando a confirmação
//         "recusa"      o ensaio falhou (erro, ou recusa por segurança)
//         "agindo"      rodando; as linhas chegam
//         "feito" / "falhou"
Singleton {
    id: root

    property string fase: ""
    property string acao: ""
    property var nomes: []
    property bool mesmoAssim: false
    property var plano: null
    property string erro: ""
    property bool recusado: false
    property var linhas: []
    readonly property bool aberta: fase !== ""

    // Uma ação terminou (bem ou mal): quem mostra listas relê.
    signal terminou()

    function planejar(acao, nomes, mesmoAssim) {
        if (root.fase === "agindo" || root.fase === "planejando")
            return;
        root.acao = acao;
        root.nomes = nomes || [];
        root.mesmoAssim = !!mesmoAssim;
        root.plano = null;
        root.erro = "";
        root.recusado = false;
        root.linhas = [];
        root.fase = "planejando";
        Coletor.pedir(["plano"].concat(Pacotes.argsDeAcao(root.acao, root.nomes, root.mesmoAssim)), r => {
            if (root.fase !== "planejando")
                return;
            if (r.erro) {
                root.erro = r.erro;
                root.recusado = r.recusado === true;
                root.fase = "recusa";
                return;
            }
            root.plano = r;
            root.fase = "plano";
        });
    }

    // Depois de uma recusa por pacote protegido: o mesmo ensaio, com
    // --mesmo-assim. O plano volta para a tela e a pessoa confirma de novo.
    function insistir() {
        if (root.fase === "recusa" && root.recusado)
            planejar(root.acao, root.nomes, true);
    }

    function confirmar() {
        if (root.fase !== "plano" || !Pacotes.podeConfirmar(root.plano))
            return;
        root.linhas = [];
        root.fase = "agindo";
        Coletor.transmitir(["agir"].concat(Pacotes.argsDeAcao(root.acao, root.nomes, root.mesmoAssim)),
            texto => root.linhas = Pacotes.empurra(root.linhas, texto, 2000),
            fim => {
                if (fim.tipo === "erro") {
                    root.erro = fim.texto;
                    root.fase = "falhou";
                } else {
                    root.fase = "feito";
                }
                Coletor.ler();
                root.terminou();
            });
    }

    function fechar() {
        if (root.fase === "agindo")
            return;
        root.fase = "";
        root.plano = null;
        root.erro = "";
        root.linhas = [];
    }
}
