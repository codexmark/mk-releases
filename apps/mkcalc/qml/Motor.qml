pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Calc.js" as Calc

// A ponte com o motor (../CONTRATO.md). O motor fica de pé no modo `servir`:
// pedir() manda uma linha JSON com um id novo e entrega a resposta desse id
// (já como objeto; { erro } quando falha) a quem pediu. É assim que a prévia
// sai a cada tecla sem abrir um processo por conta.
// MKCALC_MOTOR troca o binário; MKCALC_EXEMPLO aponta uma pasta de respostas
// gravadas, para ver a janela sem ele; MKCALC_EXPR é a expressão com que a
// janela abre (o lançador passa: `mkcalc "2+2"`).
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKCALC_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKCALC_MOTOR") || Quickshell.shellPath("../bin/mkcalc-motor")
    readonly property string inicial: Quickshell.env("MKCALC_EXPR") || ""

    property int ultimoId: 0
    // Quem espera cada resposta (id -> função) e as linhas que esperam o
    // motor subir.
    property var esperando: ({})
    property var fila: []
    property bool dePe: false

    // Manda um pedido ({ op, expr… }) e chama feito(resposta). Devolve o id.
    function pedir(pedido, feito) {
        ultimoId += 1;
        const p = Object.assign({ id: ultimoId }, pedido);
        if (exemplo !== "") {
            gravado.createObject(root, { command: Calc.comandoDeExemplo(exemplo, p), pedidoId: p.id, feito: feito || null, running: true });
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
        const feito = esperando[r.id];
        delete esperando[r.id];
        if (feito)
            feito(r);
    }

    // O motor caiu (ou nem subiu): ninguém fica esperando para sempre.
    function desistir() {
        const ids = Object.keys(esperando);
        fila = [];
        for (const id of ids)
            entregar({ id: Number(id), erro: "o motor não respondeu (bin/mkcalc-motor)", incompleta: false });
    }

    Process {
        id: servidor
        command: [root.bin, "servir"]
        stdinEnabled: true
        onStarted: {
            root.dePe = true;
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

    // Uma resposta gravada (MKCALC_EXEMPLO): um jq por pedido.
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
                let r;
                try {
                    r = JSON.parse(g.saida.trim().split("\n").pop());
                } catch (e) {
                    r = { id: g.pedidoId, erro: "o exemplo não respondeu", incompleta: false };
                }
                if (g.feito)
                    g.feito(r);
                g.destroy();
            }
        }
    }
}
