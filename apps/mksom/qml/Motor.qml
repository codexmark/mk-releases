pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Som.js" as Som

// A ponte com o motor (../CONTRATO.md). Um processo por pedido: pedir() roda
// um subcomando de uma resposta só e a entrega já como objeto ({ erro } quando
// falha); fluxo() roda um que solta várias linhas (o medidor, a gravação) e
// segura a entrada padrão dele aberta: o motor para quando ela fecha.
// MKSOM_MOTOR troca o binário; MKSOM_EXEMPLO aponta uma pasta de respostas
// gravadas, para ver a janela sem ele (e sem tocar nem gravar som).
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKSOM_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKSOM_MOTOR") || Quickshell.shellPath("../bin/mksom-motor")
    // A área em que a janela abre (o lançador diz: `mksom microfone`).
    readonly property string areaInicial: Som.area(Quickshell.env("MKSOM_AREA") || "")

    function comando(args, emLinhas) {
        if (exemplo !== "")
            return ["jq", "-c", emLinhas ? ".[]" : ".", exemplo + "/" + Som.arquivoDeExemplo(args)];
        return [bin].concat(args);
    }

    // Roda um subcomando e chama feito(resposta). `entrada`, se vier, é o
    // texto que vai para a entrada padrão dele (o JSON do `eq ajustar`).
    function pedir(args, feito, entrada) {
        const tem = typeof entrada === "string" && exemplo === "";
        pedido.createObject(root, {
            command: comando(args, false), feito: feito || null, temEntrada: tem, entrada: tem ? entrada : "", running: true
        });
    }

    // Roda um subcomando de várias linhas: aCadaLinha(objeto) a cada uma e
    // noFim() quando o processo acaba. Devolve o processo: parar() o encerra.
    function fluxo(args, aCadaLinha, noFim) {
        return longo.createObject(root, { command: comando(args, true), aCadaLinha: aCadaLinha || null, noFim: noFim || null, running: true });
    }

    // Apaga a gravação de teste, se houver: a janela chama ao fechar. Vai
    // solto, porque a janela não fica para esperar a resposta.
    function limpar() {
        if (exemplo === "")
            Quickshell.execDetached([bin, "mic", "limpar"]);
    }

    Component {
        id: pedido

        Process {
            id: p
            property var feito: null
            property bool temEntrada: false
            property string entrada: ""
            property string saida: ""
            stdinEnabled: temEntrada
            // O pedido vai inteiro e a entrada fecha: o motor lê até o fim.
            onStarted: {
                if (p.temEntrada) {
                    p.write(p.entrada);
                    p.stdinEnabled = false;
                }
            }
            stdout: StdioCollector {
                onStreamFinished: p.saida = text
            }
            onExited: (codigo, estado) => {
                let r;
                try {
                    r = JSON.parse(p.saida.trim().split("\n").pop());
                } catch (e) {
                    r = { erro: "o motor não respondeu (" + p.command.slice(0, 3).join(" ") + ")" };
                }
                if (r && r.erro)
                    r.recusado = codigo === 2;
                if (p.feito)
                    p.feito(r);
                p.destroy();
            }
        }
    }

    Component {
        id: longo

        Process {
            id: l
            property var aCadaLinha: null
            property var noFim: null
            // A entrada padrão fica aberta enquanto o fluxo interessa: fechá-la
            // é o jeito de mandar o motor parar (e ele mata o pw-record).
            stdinEnabled: true

            function parar() {
                l.aCadaLinha = null;
                l.stdinEnabled = false;
                l.running = false;
            }

            stdout: SplitParser {
                onRead: data => {
                    let r;
                    try {
                        r = JSON.parse(data);
                    } catch (e) {
                        return;
                    }
                    if (l.aCadaLinha)
                        l.aCadaLinha(r);
                }
            }
            // A última linha pode chegar junto com o fim do processo: espera
            // a fila de eventos esvaziar antes de fechar a conta.
            onExited: Qt.callLater(() => {
                if (l.noFim)
                    l.noFim();
                l.destroy();
            })
        }
    }
}
