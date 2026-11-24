import QtQuick
import QtQuick.Layouts
import "Pulso.js" as Pulso

// Desempenho: a máquina inteira no último minuto e meio. CPU com os núcleos,
// memória, disco, rede, GPU e temperatura, e os discos com o que está ocupado.
Item {
    id: pagina

    readonly property var r: Coletor.retrato
    readonly property var gpu: r && r.gpus && r.gpus.length > 0 ? r.gpus[0] : null

    ColumnLayout {
        anchors.fill: parent
        spacing: 16

        Cabecalho {
            numero: "02"
            titulo: "Desempenho"
            frase: "A máquina inteira, do último minuto e meio até agora."
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            columns: width > 900 ? 3 : 2
            rowSpacing: 12
            columnSpacing: 12

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "cpu"
                valor: pagina.r && pagina.r.cpu ? Pulso.pct(pagina.r.cpu.pct) : "—"
                detalhe: pagina.r && pagina.r.cpu
                    ? (pagina.r.cpu.freq_mhz ? Math.round(pagina.r.cpu.freq_mhz) + " MHz  ·  " : "") + "carga " + (pagina.r.cpu.carga || []).map(c => c.toFixed(2)).join(" ")
                    : ""
                serie: Coletor.cpuSerie
                teto: 100

                // Os núcleos, um traço cada.
                Row {
                    spacing: 3
                    Repeater {
                        model: pagina.r && pagina.r.cpu ? pagina.r.cpu.por_nucleo : []
                        Rectangle {
                            required property var modelData
                            width: 14
                            height: 18
                            color: Theme.boneA(0.06)
                            Rectangle {
                                anchors.bottom: parent.bottom
                                width: parent.width
                                height: Math.max(1, parent.height * Math.min(1, parent.modelData / 100))
                                color: Theme.copper
                            }
                        }
                    }
                }
            }

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "memória"
                valor: pagina.r && pagina.r.mem ? Pulso.tamanho(pagina.r.mem.usado_kb) : "—"
                detalhe: pagina.r && pagina.r.mem ? "de " + Pulso.tamanho(pagina.r.mem.total_kb) + "  ·  cache " + Pulso.tamanho(pagina.r.mem.cache_kb) : ""
                serie: Coletor.memSerie
                teto: 100

                Text {
                    text: pagina.r && pagina.r.mem && pagina.r.mem.swap_total_kb > 0
                        ? "swap " + Pulso.tamanho(pagina.r.mem.swap_usado_kb) + " de " + Pulso.tamanho(pagina.r.mem.swap_total_kb) : "sem swap"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "temperatura"
                valor: pagina.r && pagina.r.cpu && pagina.r.cpu.temp_c !== null ? Math.round(pagina.r.cpu.temp_c) + "°" : "—"
                detalhe: pagina.r && pagina.r.bateria ? "bateria " + pagina.r.bateria.pct + "%" + (pagina.r.bateria.carregando ? ", carregando" : "")
                    + (pagina.r.bateria.watts !== null ? "  ·  " + Pulso.num(pagina.r.bateria.watts) + " W" : "") : ""
                serie: Coletor.tempSerie
                teto: 100
            }

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "disco  ·  escrita (cobre) e leitura"
                valor: Pulso.taxa(Coletor.gravarSerie.length > 0 ? Coletor.gravarSerie[Coletor.gravarSerie.length - 1] : null)
                detalhe: "leitura " + Pulso.taxa(Coletor.lerSerie.length > 0 ? Coletor.lerSerie[Coletor.lerSerie.length - 1] : null)
                serie: Coletor.gravarSerie
                serie2: Coletor.lerSerie
                teto: 0
            }

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "rede  ·  recebendo (cobre) e enviando"
                valor: Pulso.taxa(Coletor.rxSerie.length > 0 ? Coletor.rxSerie[Coletor.rxSerie.length - 1] : null)
                detalhe: "enviando " + Pulso.taxa(Coletor.txSerie.length > 0 ? Coletor.txSerie[Coletor.txSerie.length - 1] : null)
                    + (pagina.r && pagina.r.rede && pagina.r.rede.length > 0 ? "  ·  " + pagina.r.rede.map(i => i.nome).join(", ") : "")
                serie: Coletor.rxSerie
                serie2: Coletor.txSerie
                teto: 0
            }

            Cartao {
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumHeight: 170
                rotulo: "gpu"
                valor: pagina.gpu && pagina.gpu.uso_pct !== null ? Pulso.pct(pagina.gpu.uso_pct) : "—"
                detalhe: pagina.gpu ? pagina.gpu.nome + (pagina.gpu.mem_total_kb ? "  ·  " + Pulso.tamanho(pagina.gpu.mem_usada_kb) + " de " + Pulso.tamanho(pagina.gpu.mem_total_kb) : "")
                    : "nenhuma placa com leitura"
                serie: Coletor.gpuSerie
                teto: 100
            }
        }

        // Os discos montados.
        Column {
            Layout.fillWidth: true
            spacing: 8

            Rotulo { text: "discos" }
            Repeater {
                model: pagina.r ? pagina.r.montagens : []
                Item {
                    id: m
                    required property var modelData
                    readonly property real cheio: modelData.total_kb > 0 ? modelData.usado_kb / modelData.total_kb : 0
                    width: parent.width
                    height: 22

                    Text {
                        width: 180
                        anchors.verticalCenter: parent.verticalCenter
                        text: m.modelData.onde
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                        elide: Text.ElideMiddle
                    }
                    Rectangle {
                        x: 190
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width - 190 - 230
                        height: 4
                        color: Theme.boneA(0.08)
                        Rectangle { width: parent.width * Math.min(1, m.cheio); height: parent.height; color: m.cheio > 0.9 ? Theme.copper : Theme.textDim }
                    }
                    Text {
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                        text: Pulso.tamanho(m.modelData.usado_kb) + " de " + Pulso.tamanho(m.modelData.total_kb) + "  ·  " + m.modelData.fs
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                }
            }
        }
    }
}
