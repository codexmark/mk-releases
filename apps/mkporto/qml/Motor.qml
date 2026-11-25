pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Porto.js" as Porto

// A ponte com o motor (../CONTRATO.md). Um processo por pedido: pedir() roda
// um subcomando de uma resposta só e a entrega já como objeto ({ erro } quando
// falha); tarefa() roda um que solta várias linhas (o `sondar`) e entrega cada
// uma; abrir() abre o terminal que o motor mandou abrir.
// MKPORTO_MOTOR troca o binário; MKPORTO_EXEMPLO aponta uma pasta de respostas
// gravadas, para ver a janela sem ele (e sem abrir terminal nenhum).
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKPORTO_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKPORTO_MOTOR") || Quickshell.shellPath("../bin/mkporto-motor")

    function comando(args, emLinhas) {
        if (exemplo !== "")
            return ["jq", "-c", emLinhas ? ".[]" : ".", exemplo + "/" + Porto.arquivoDeExemplo(args)];
        return [bin].concat(args);
    }

    // Roda um subcomando e chama feito(resposta). `entrada`, se vier, é o
    // texto que vai para a entrada padrão dele (o JSON do `guardar`).
    function pedir(args, feito, entrada) {
        const tem = typeof entrada === "string" && exemplo === "";
        pedido.createObject(root, {
            command: comando(args, false), feito: feito || null, temEntrada: tem, entrada: tem ? entrada : "", running: true
        });
    }

    // Roda um subcomando de várias linhas: aCadaLinha(objeto) a cada uma e
    // noFim() quando o processo acaba.
    function tarefa(args, aCadaLinha, noFim) {
        return longa.createObject(root, { command: comando(args, true), aCadaLinha: aCadaLinha || null, noFim: noFim || null, running: true });
    }

    // Abre o terminal que o `comando` do motor devolveu: o argv dele, inteiro
    // e sem shell. A janela não monta comando nenhum. Devolve "" se abriu, ou
    // a frase de por que não.
    function abrir(resposta) {
        if (exemplo !== "")
            return Porto.abriria(resposta);
        if (!resposta || !Array.isArray(resposta.terminal) || resposta.terminal.length === 0)
            return (resposta && resposta.aviso) || "não achei um terminal para abrir";
        Quickshell.execDetached(resposta.terminal);
        return "";
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
                    r = { erro: "o motor não respondeu (" + p.command.slice(0, 2).join(" ") + ")" };
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
        id: longa

        Process {
            id: l
            property var aCadaLinha: null
            property var noFim: null
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
