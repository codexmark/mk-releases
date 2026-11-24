pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import "Pulso.js" as Pulso

// A ponte com o coletor (../CONTRATO.md). `retrato` é o último retrato do
// `watch`; as séries guardam o último minuto para os gráficos; pedir() roda um
// subcomando e entrega a resposta já como objeto ({ erro } quando falha).
// MKPULSO_COLETOR troca o binário; MKPULSO_EXEMPLO aponta uma pasta de
// respostas gravadas (watch.json, sistema.json…), para ver a janela sem ele.
Singleton {
    id: root

    readonly property string exemplo: Quickshell.env("MKPULSO_EXEMPLO") || ""
    readonly property string bin: Quickshell.env("MKPULSO_COLETOR") || Quickshell.shellPath("../bin/mkpulso-coletor")
    readonly property int pontos: 90

    property var retrato: null
    property string erro: ""
    readonly property bool pronto: retrato !== null

    // O último minuto e meio, um ponto por retrato.
    property var cpuSerie: []
    property var memSerie: []
    property var rxSerie: []
    property var txSerie: []
    property var lerSerie: []
    property var gravarSerie: []
    property var gpuSerie: []
    property var tempSerie: []

    function soma(lista, campo) {
        let s = 0;
        for (const x of lista || [])
            s += x[campo] || 0;
        return s;
    }

    function chegou(linha) {
        let r;
        try {
            r = JSON.parse(linha);
        } catch (e) {
            return;
        }
        if (r.erro) {
            erro = r.erro;
            return;
        }
        erro = "";
        retrato = r;
        cpuSerie = Pulso.empurra(cpuSerie, r.cpu ? r.cpu.pct : 0, pontos);
        memSerie = Pulso.empurra(memSerie, r.mem && r.mem.total_kb > 0 ? r.mem.usado_kb / r.mem.total_kb * 100 : 0, pontos);
        rxSerie = Pulso.empurra(rxSerie, soma(r.rede, "rx_bps"), pontos);
        txSerie = Pulso.empurra(txSerie, soma(r.rede, "tx_bps"), pontos);
        lerSerie = Pulso.empurra(lerSerie, soma(r.discos, "leitura_bps"), pontos);
        gravarSerie = Pulso.empurra(gravarSerie, soma(r.discos, "escrita_bps"), pontos);
        gpuSerie = Pulso.empurra(gpuSerie, r.gpus && r.gpus.length > 0 ? r.gpus[0].uso_pct : 0, pontos);
        tempSerie = Pulso.empurra(tempSerie, r.cpu ? r.cpu.temp_c : 0, pontos);
    }

    // Roda um subcomando e chama feito(resposta).
    function pedir(args, feito) {
        const command = exemplo !== ""
            ? ["jq", "-c", ".", exemplo + "/" + args[0] + ".json"]
            : [bin].concat(args);
        pedido.createObject(root, { command: command, feito: feito || null, running: true });
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
                    r = { erro: "o coletor não respondeu (" + p.command.join(" ") + ")" };
                }
                if (p.feito)
                    p.feito(r);
                p.destroy();
            }
        }
    }

    Process {
        id: watch
        running: true
        command: root.exemplo !== ""
            ? ["jq", "-c", ".", root.exemplo + "/watch.json"]
            : [root.bin, "watch", "--intervalo", "1000"]
        stdout: SplitParser {
            onRead: data => root.chegou(data)
        }
        onExited: (code, status) => {
            if (root.exemplo === "" && code !== 0)
                root.erro = "o coletor parou (código " + code + ")";
        }
    }

    // O coletor caiu: tenta de novo, sem pressa.
    Timer {
        interval: 3000
        repeat: true
        running: root.exemplo === "" && !watch.running
        onTriggered: watch.running = true
    }
}
