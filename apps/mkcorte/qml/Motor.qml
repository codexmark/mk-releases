pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Corte.js" as Corte

// A ponte com o motor (../CONTRATO.md). pedir() roda um subcomando de uma
// resposta só e a entrega já como objeto ({ erro } quando falha); tarefa()
// roda o demorado (exportar), entrega cada fase e devolve o processo, para
// dar para cancelar (parar o processo manda SIGTERM).
// MKCORTE_MOTOR troca o binário; MKCORTE_EXEMPLO aponta uma pasta de
// respostas gravadas, para ver a janela sem ele.
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKCORTE_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKCORTE_MOTOR") || Quickshell.shellPath("../bin/mkcorte-motor")
    // Os arquivos que vieram na linha de comando.
    readonly property var arquivos: Corte.lerArquivos(Quickshell.env("MKCORTE_ARQUIVOS") || "")

    function comando(args, emFases) {
        if (exemplo !== "")
            return ["jq", "-c", emFases ? ".[]" : ".", exemplo + "/" + Corte.arquivoDeExemplo(args)];
        return [bin].concat(args);
    }

    // Roda um subcomando e chama feito(resposta).
    function pedir(args, feito) {
        pedido.createObject(root, { command: comando(args, false), feito: feito || null, running: true });
    }

    // Roda um subcomando demorado: aCadaFase(fase) a cada linha e, quando o
    // processo acaba, noFim(ultimaLinha). Devolve o processo.
    function tarefa(args, aCadaFase, noFim) {
        return longa.createObject(root, { command: comando(args, true), aCadaFase: aCadaFase || null, noFim: noFim || null, running: true });
    }

    Component {
        id: pedido

        Process {
            id: p
            property var feito: null
            property string saida: ""
            stdout: StdioCollector {
                onStreamFinished: p.saida = text
            }
            onExited: {
                let r;
                try {
                    r = JSON.parse(p.saida.trim().split("\n").pop());
                } catch (e) {
                    r = { erro: "o motor não respondeu (" + p.command.slice(0, 2).join(" ") + ")" };
                }
                if (p.feito)
                    p.feito(r);
                p.destroy();
            }
        }
    }

    Component {
        id: longa

        Process {
            id: l
            property var aCadaFase: null
            property var noFim: null
            property var ultima: null
            stdout: SplitParser {
                onRead: data => {
                    let r;
                    try {
                        r = JSON.parse(data);
                    } catch (e) {
                        return;
                    }
                    l.ultima = r;
                    if (l.aCadaFase)
                        l.aCadaFase(r);
                }
            }
            // A última linha pode chegar junto com o fim do processo: espera
            // a fila de eventos esvaziar antes de fechar a conta.
            onExited: Qt.callLater(() => {
                if (l.noFim)
                    l.noFim(l.ultima || { erro: "o motor parou sem responder" });
                l.destroy();
            })
        }
    }
}
