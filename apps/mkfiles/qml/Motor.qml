pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Arquivos.js" as Arquivos

// A ponte com o motor (../CONTRATO.md). O motor fica de pé no modo `servir`:
// pedir() manda uma linha JSON com um id novo e entrega a quem pediu cada
// linha de resposta desse id (uma, ou várias num fluxo, até a de "fim":
// true). O que o motor diz sozinho ({ evento }) sai pelo sinal evento().
// MKFILES_MOTOR troca o binário; MKFILES_EXEMPLO aponta uma pasta de respostas
// gravadas, para ver a janela sem ele; MKFILES_ABRIR traz os caminhos pedidos
// na linha de comando (o lançador passa), um por linha.
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKFILES_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKFILES_MOTOR") || Quickshell.shellPath("../bin/mkfiles-motor")
    readonly property string abrir: Quickshell.env("MKFILES_ABRIR") || ""
    readonly property string home: Quickshell.env("HOME") || "/"

    property int ultimoId: 0
    // Quem espera cada resposta (id -> função) e as linhas que esperam o
    // motor subir.
    property var esperando: ({})
    property var fila: []
    property bool dePe: false
    // O motor caiu com gente esperando: a janela avisa.
    property bool caiu: false

    signal evento(var e)

    // Manda um pedido ({ op, … }) e chama feito(linha) a cada linha da
    // resposta. Devolve o id.
    function pedir(pedido, feito) {
        ultimoId += 1;
        const p = Object.assign({ id: ultimoId }, pedido);
        if (exemplo !== "") {
            gravado.createObject(root, { command: Arquivos.comandoDeExemplo(exemplo, p), pedidoId: p.id, feito: feito || null, running: true });
            return p.id;
        }
        esperando[p.id] = feito || null;
        const linha = JSON.stringify(p) + "\n";
        if (dePe) {
            servidor.write(linha);
        } else {
            fila.push(linha);
            servidor.running = true;
        }
        return p.id;
    }

    function entregar(r) {
        if (r.evento !== undefined) {
            root.evento(r);
            return;
        }
        const feito = esperando[r.id];
        // Num fluxo, só a linha de "fim": true (ou um erro) encerra.
        if (r.fim !== false)
            delete esperando[r.id];
        if (feito)
            feito(r);
    }

    // O motor caiu (ou nem subiu): ninguém fica esperando para sempre.
    function desistir() {
        const ids = Object.keys(esperando);
        fila = [];
        if (ids.length > 0)
            caiu = true;
        for (const id of ids)
            entregar({ id: Number(id), erro: "o motor não respondeu (bin/mkfiles-motor)", motivo: "motor", fim: true });
    }

    Process {
        id: servidor
        command: [root.bin, "servir"]
        stdinEnabled: true
        onStarted: {
            root.dePe = true;
            root.caiu = false;
            const linhas = root.fila;
            root.fila = [];
            for (const linha of linhas)
                servidor.write(linha);
        }
        onRunningChanged: {
            if (!running) {
                root.dePe = false;
                root.desistir();
            }
        }
        stdout: SplitParser {
            onRead: data => {
                let r;
                try {
                    r = JSON.parse(data);
                } catch (e) {
                    return;
                }
                root.entregar(r);
            }
        }
    }

    // As linhas gravadas de um pedido (MKFILES_EXEMPLO): um jq por pedido. As
    // respostas ganham o id; os eventos saem depois delas.
    Component {
        id: gravado

        Process {
            id: g
            property int pedidoId: 0
            property var feito: null
            property string saida: ""
            stdout: StdioCollector {
                onStreamFinished: g.saida = text
            }
            onExited: {
                const eventos = [];
                let respondeu = false;
                for (const linha of g.saida.split("\n")) {
                    if (linha.trim() === "")
                        continue;
                    let r;
                    try {
                        r = JSON.parse(linha);
                    } catch (e) {
                        continue;
                    }
                    if (r.evento !== undefined) {
                        eventos.push(r);
                        continue;
                    }
                    respondeu = true;
                    r.id = g.pedidoId;
                    if (g.feito)
                        g.feito(r);
                }
                if (!respondeu && g.feito)
                    g.feito({ id: g.pedidoId, erro: "o exemplo não tem resposta para este pedido", fim: true });
                for (const e of eventos)
                    root.evento(e);
                g.destroy();
            }
        }
    }
}
