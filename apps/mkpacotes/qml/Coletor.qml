pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Pacotes.js" as Pacotes

// A ponte com o coletor (../CONTRATO.md). Guarda os instalados e o estado (o
// que quase toda página usa); pedir() roda um subcomando e entrega a resposta
// já como objeto ({ erro } quando falha, com `recusado` se foi por segurança);
// transmitir() roda o `agir` e entrega linha a linha.
// MKPACOTES_COLETOR troca o binário; MKPACOTES_EXEMPLO aponta uma pasta de
// respostas gravadas (instalados.json, plano.json…), para ver a janela sem ele.
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKPACOTES_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKPACOTES_COLETOR") || Quickshell.shellPath("../bin/mkpacotes-coletor")

    property var pacotes: []
    property var totais: null
    property var estado: null
    property string aviso: ""
    property string erro: ""
    property bool pronto: false
    property bool lendo: false
    // Sobe a cada leitura: as páginas que guardam respostas próprias releem.
    property int leitura: 0

    function instalado(nome) {
        for (const p of pacotes)
            if (p.nome === nome)
                return p;
        return null;
    }

    // Roda um subcomando e chama feito(resposta).
    function pedir(args, feito) {
        const command = exemplo !== ""
            ? ["jq", "-c", ".", exemplo + "/" + args[0] + ".json"]
            : [bin].concat(args);
        pedido.createObject(root, { command: command, feito: feito || null, running: true });
    }

    // Roda uma ação (`agir …`): linha(texto) a cada linha do progresso e
    // fim({ tipo: "ok" | "erro", … }) uma vez, no final.
    function transmitir(args, linha, fim) {
        const command = exemplo !== ""
            ? ["jq", "-c", ".[]", exemplo + "/agir.json"]
            : [bin].concat(args);
        fluxo.createObject(root, { command: command, linha: linha || null, fim: fim || null, running: true });
    }

    // Relê os instalados e o estado (depois de uma ação, ou a pedido).
    function ler() {
        lendo = true;
        pedir(["instalados"], r => {
            root.lendo = false;
            root.erro = r.erro || "";
            if (!r.erro) {
                root.pacotes = r.pacotes || [];
                root.totais = r.totais || null;
                root.aviso = r.aviso || "";
                root.pronto = true;
            }
            root.leitura++;
        });
        pedir(["estado"], r => {
            if (!r.erro)
                root.estado = r;
        });
    }

    Component.onCompleted: ler()

    Component {
        id: pedido

        Process {
            id: p
            property var feito: null
            property string saida: ""
            stdout: StdioCollector {
                onStreamFinished: p.saida = text
            }
            onExited: (code, status) => {
                let r;
                try {
                    r = JSON.parse(p.saida.trim().split("\n").pop());
                } catch (e) {
                    r = { erro: "o coletor não respondeu (" + p.command.join(" ") + ")" };
                }
                if (r && r.erro)
                    r.recusado = code === 2;
                if (p.feito)
                    p.feito(r);
                p.destroy();
            }
        }
    }

    Component {
        id: fluxo

        Process {
            id: f
            property var linha: null
            property var fim: null
            property bool acabou: false
            property int codigo: 0

            function fecha(resposta) {
                if (f.acabou)
                    return;
                f.acabou = true;
                if (f.fim)
                    f.fim(resposta);
            }

            stdout: SplitParser {
                onRead: data => {
                    const x = Pacotes.interpreta(data);
                    if (x.tipo === "linha") {
                        if (f.linha)
                            f.linha(x.texto);
                    } else if (x.tipo !== "nada") {
                        f.fecha(x);
                    }
                }
            }
            onExited: (code, status) => {
                f.codigo = code;
                espera.start();
            }

            // A última linha pode chegar um instante depois de o processo sair.
            property Timer espera: Timer {
                interval: 400
                onTriggered: {
                    f.fecha({ tipo: "erro", texto: "o coletor parou sem responder (código " + f.codigo + ")" });
                    f.destroy();
                }
            }
        }
    }
}
