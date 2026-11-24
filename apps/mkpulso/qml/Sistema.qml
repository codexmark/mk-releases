import QtQuick
import QtQuick.Layouts
import "Pulso.js" as Pulso

// Sistema: a ficha da máquina. Lida uma vez ao abrir a página (e de novo no
// botão): o que ela é não muda a cada segundo.
Item {
    id: pagina

    property var f: null
    property string erro: ""

    function ler() {
        Coletor.pedir(["sistema"], r => {
            if (r.erro)
                pagina.erro = r.erro;
            else {
                pagina.erro = "";
                pagina.f = r;
            }
        });
    }
    Component.onCompleted: ler()

    function ou(v, sufixo) {
        return v === null || v === undefined || v === "" ? "—" : String(v) + (sufixo || "");
    }

    readonly property var blocos: f === null ? [] : [
        { titulo: "máquina", linhas: [
            ["nome", ou(f.maquina)], ["modelo", ou(f.modelo)], ["placa", ou(f.placa)], ["firmware", ou(f.firmware)] ] },
        { titulo: "processador e memória", linhas: [
            ["processador", ou(f.cpu && f.cpu.nome)],
            ["núcleos", f.cpu ? ou(f.cpu.nucleos) + " (" + ou(f.cpu.threads) + " linhas)" : "—"],
            ["até", f.cpu ? ou(f.cpu.max_mhz, " MHz") : "—"], ["memória", Pulso.tamanho(f.mem_total_kb)] ] },
        { titulo: "vídeo", linhas: (f.gpus || []).map(g => [ou(g.driver), ou(g.nome)])
            .concat((f.telas || []).map(t => [ou(t.nome), ou(t.largura) + " × " + ou(t.altura) + " a " + (t.hz ? Pulso.num(t.hz) + " Hz" : "—")
                + (t.escala && t.escala !== 1 ? "  ·  escala " + t.escala : "") + (t.descricao ? "  ·  " + t.descricao : "")])) },
        { titulo: "discos", linhas: (f.discos || []).map(d => [ou(d.nome), ou(d.modelo) + "  ·  " + Pulso.tamanho(d.tamanho_kb)
            + (d.rotativo === true ? "  ·  disco rígido" : d.rotativo === false ? "  ·  SSD" : "") + (d.saude ? "  ·  saúde " + d.saude : "")]) },
        { titulo: "bateria", linhas: f.bateria ? [ ["modelo", ou(f.bateria.modelo)], ["ciclos", ou(f.bateria.ciclos)],
            ["saúde", ou(f.bateria.saude_pct, "%")], ["capacidade", f.bateria.capacidade_wh !== null ? Pulso.num(f.bateria.capacidade_wh) + " Wh" : "—"] ] : [] },
        { titulo: "sistema", linhas: f.sistema ? [ ["sistema", ou(f.sistema.nome)],
            ["versão do Mk", ou(f.sistema.versao_mk) + (f.sistema.data_mk ? "  ·  " + f.sistema.data_mk : "")],
            ["kernel", ou(f.sistema.kernel)], ["compositor", ou(f.sistema.compositor)],
            ["ligado há", Pulso.duracao(f.sistema.ligado_ha_s)], ["pacotes", ou(f.sistema.pacotes)] ] : [] }
    ].filter(b => b.linhas.length > 0)

    ColumnLayout {
        anchors.fill: parent
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            Cabecalho {
                numero: "03"
                titulo: "Sistema"
                frase: "O que esta máquina é: peças, discos, telas e a versão do Mk."
            }
            Item { Layout.fillWidth: true }
            Botao { Layout.alignment: Qt.AlignBottom; texto: "ler de novo"; onAcionado: pagina.ler() }
        }

        Text {
            visible: pagina.erro !== "" || pagina.f === null
            text: pagina.erro !== "" ? pagina.erro : "lendo a ficha…"
            color: pagina.erro !== "" ? Theme.copper : Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 14
            font.italic: true
        }

        Flickable {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            contentHeight: ficha.implicitHeight
            boundsBehavior: Flickable.StopAtBounds

            Column {
                id: ficha
                width: parent.width
                spacing: 22

                Repeater {
                    model: pagina.blocos
                    Column {
                        id: bloco
                        required property var modelData
                        width: ficha.width
                        spacing: 0

                        Rotulo { text: bloco.modelData.titulo; bottomPadding: 8 }
                        Rectangle { width: parent.width; height: 1; color: Theme.hairline }
                        Repeater {
                            model: bloco.modelData.linhas
                            Item {
                                required property var modelData
                                width: bloco.width
                                height: 30
                                Text {
                                    width: 190
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: parent.modelData[0]
                                    color: Theme.textDim
                                    font.family: Theme.sans
                                    font.pixelSize: 12
                                    elide: Text.ElideRight
                                }
                                Text {
                                    x: 200
                                    width: parent.width - 200
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: parent.modelData[1]
                                    color: Theme.text
                                    font.family: Theme.mono
                                    font.pixelSize: 12
                                    elide: Text.ElideRight
                                }
                                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                            }
                        }
                    }
                }
            }
        }
    }
}
