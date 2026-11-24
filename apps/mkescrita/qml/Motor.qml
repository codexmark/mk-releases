pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Escrita.js" as Escrita

// A ponte com o motor (../CONTRATO.md). pedir() roda um subcomando de uma
// resposta só, mandando o texto pelo stdin quando há, e entrega a resposta já
// como objeto ({ erro } quando falha) junto com o código de saída; vigia()
// deixa o `vigiar` rodando e entrega cada evento.
// MKESCRITA_MOTOR troca o binário; MKESCRITA_EXEMPLO aponta uma pasta de
// respostas gravadas, para ver a janela sem ele.
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKESCRITA_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKESCRITA_MOTOR") || Quickshell.shellPath("../bin/mkescrita-motor")
    // O lançador, para abrir outra janela (Ctrl+N).
    readonly property string lancador: Quickshell.shellPath("../bin/mkescrita")
    // O arquivo que veio na linha de comando ("" = documento novo).
    readonly property string arquivo: exemplo !== "" ? "/exemplo/notas.md" : (Quickshell.env("MKESCRITA_ARQUIVO") || "")

    function comando(args, emFases) {
        if (exemplo !== "")
            return ["jq", "-c", emFases ? ".[]" : ".", exemplo + "/" + Escrita.arquivoDeExemplo(args)];
        return [bin].concat(args);
    }

    // Roda um subcomando e chama feito(resposta, codigoDeSaida). `entrada` é
    // o texto que vai pelo stdin (null = nenhum).
    function pedir(args, entrada, feito) {
        const p = pedido.createObject(root, {
            command: comando(args, false), feito: feito || null,
            temEntrada: entrada !== null && entrada !== undefined, entrada: entrada || ""
        });
        // Só liga depois de tudo no lugar: o processo tem de nascer já com o
        // stdin aberto (ou fechado), não descobrir isso depois.
        p.running = true;
    }

    // Deixa o `vigiar` rodando: aCada(evento) a cada linha. Devolve o
    // processo; quem chamou o para com parar().
    function vigia(args, aCada) {
        return longo.createObject(root, { command: comando(args, true), aCada: aCada || null, running: true });
    }

    function parar(processo) {
        if (!processo)
            return;
        processo.aCada = null;
        processo.running = false;
        processo.destroy();
    }

    // Abre outra janela do MkEscrita (num arquivo, ou vazia).
    function novaJanela(caminho) {
        Quickshell.execDetached(caminho ? [lancador, "--nova", "--", caminho] : [lancador, "--nova"]);
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
            // O texto vai inteiro e o stdin fecha: o motor lê até o fim e
            // confere o tamanho (--bytes) antes de gravar qualquer coisa.
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
                    r = { erro: "o motor não respondeu (" + p.command.slice(0, 2).join(" ") + ")" };
                    codigo = codigo || 1;
                }
                if (p.feito)
                    p.feito(r, codigo);
                p.destroy();
            }
        }
    }

    Component {
        id: longo

        Process {
            id: l
            property var aCada: null
            stdout: SplitParser {
                onRead: data => {
                    let r;
                    try {
                        r = JSON.parse(data);
                    } catch (e) {
                        return;
                    }
                    if (l.aCada)
                        l.aCada(r);
                }
            }
        }
    }
}
