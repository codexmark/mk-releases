import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// Atualizações e cuidado: o que há para atualizar (quem atualiza é o
// mk-update, num terminal), os órfãos e o cache de pacotes.
Item {
    id: pagina

    property var lista: []
    property var totais: null
    property string fonte: ""
    property string erro: ""
    property string erroAur: ""
    property string erroOficial: ""
    property bool lendo: false
    property bool leu: false

    readonly property var e: Coletor.estado
    readonly property var t: Coletor.totais

    function ler() {
        lendo = true;
        Coletor.pedir(["atualizacoes"], r => {
            pagina.lendo = false;
            pagina.leu = true;
            pagina.erro = r.erro || "";
            pagina.lista = r.erro ? [] : (r.atualizacoes || []);
            pagina.totais = r.erro ? null : r.totais;
            pagina.fonte = r.fonte || "";
            pagina.erroAur = r.erro_aur || "";
            pagina.erroOficial = r.erro_oficial || "";
        });
    }
    Component.onCompleted: ler()
    Connections {
        target: Acao
        function onTerminou() { pagina.ler(); }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Cabecalho {
                numero: "03"
                titulo: "Atualizações"
                frase: "O que há de novo. Quem atualiza é o mk atualizar, num terminal."
            }
            Item { Layout.fillWidth: true }
            Botao { Layout.alignment: Qt.AlignBottom; texto: pagina.lendo ? "procurando…" : "procurar de novo"; ligado: !pagina.lendo; onAcionado: pagina.ler() }
            Botao {
                Layout.alignment: Qt.AlignBottom
                texto: "atualizar tudo"
                forte: true
                ligado: pagina.e !== null && pagina.e.comandos.mk_update === true
                onAcionado: Acao.planejar("atualizar", [], false)
            }
        }

        Text {
            Layout.fillWidth: true
            visible: text !== ""
            wrapMode: Text.WordWrap
            text: pagina.erro !== "" ? pagina.erro
                : pagina.erroOficial !== "" ? "Arch: " + pagina.erroOficial
                : pagina.erroAur !== "" ? "AUR: " + pagina.erroAur + ". A lista abaixo é só dos repositórios do Arch."
                : pagina.e !== null && pagina.e.comandos.mk_update !== true ? "não achei o mk-update: este sistema não parece ser um Mk, e o botão de atualizar fica desligado"
                : ""
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            ListView {
                id: tabela
                anchors.fill: parent
                anchors.margins: 1
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.lista

                delegate: Item {
                    id: linha
                    required property var modelData
                    width: tabela.width
                    height: 34
                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Text {
                        x: 14
                        width: Math.max(120, parent.width - 14 - 480)
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.nome
                        color: Theme.text
                        font.family: Theme.mono
                        font.pixelSize: 11
                        elide: Text.ElideRight
                    }
                    Row {
                        anchors.right: parent.right
                        anchors.rightMargin: 14
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 0
                        Text {
                            width: 170
                            horizontalAlignment: Text.AlignRight
                            text: linha.modelData.de
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideMiddle
                        }
                        Text {
                            width: 30
                            horizontalAlignment: Text.AlignHCenter
                            text: "→"
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                        Text {
                            width: 170
                            text: linha.modelData.para
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideMiddle
                        }
                        Text {
                            width: 80
                            horizontalAlignment: Text.AlignRight
                            text: Pacotes.origem(linha.modelData.origem)
                            color: linha.modelData.origem === "aur" ? Theme.copper : Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: tabela.count === 0
                    text: pagina.lendo || !pagina.leu ? "procurando atualizações…" : pagina.erro !== "" ? "não deu para procurar" : "tudo em dia"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }

        Text {
            Layout.fillWidth: true
            elide: Text.ElideRight
            text: (pagina.totais ? pagina.totais.oficiais + " do Arch" + "  ·  " + pagina.totais.aur + " do AUR" : "")
                + (pagina.fonte === "pacman" ? "  ·  pelo que a última sincronização sabe" : pagina.fonte === "checkupdates" ? "  ·  conferido agora nos espelhos" : "")
                + (pagina.e && pagina.e.sincronizado_em_s ? "  ·  última sincronização " + Pacotes.quando(pagina.e.sincronizado_em_s, Date.now() / 1000) : "")
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }

        // ── cuidado ────────────────────────────────────────────────────
        Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 1; color: Theme.hairline }
        RowLayout {
            Layout.fillWidth: true
            // O pé tem a altura do conteúdo: a linha vertical (fillHeight) faria
            // ele disputar a altura com a lista e espremê-la.
            Layout.fillHeight: false
            spacing: 24

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Rotulo { text: "órfãos" }
                Text {
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                    text: !pagina.t ? "—" : pagina.t.orfaos === 0 ? "Nenhum pacote sobrando."
                        : Pacotes.plural(pagina.t.orfaos, "pacote veio", "pacotes vieram") + " como dependência e nada mais usa."
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
            }
            Botao {
                texto: "remover os órfãos"
                ligado: pagina.t !== null && pagina.t.orfaos > 0
                onAcionado: Acao.planejar("orfaos", [], false)
            }

            Rectangle { Layout.preferredWidth: 1; Layout.fillHeight: true; color: Theme.hairline }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Rotulo { text: "cache de pacotes" }
                Text {
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                    text: !pagina.e || !pagina.e.cache ? "—"
                        : pagina.e.cache.arquivos === 0 ? "O cache está vazio."
                        : Pacotes.tamanho(pagina.e.cache.bytes) + " em " + Pacotes.plural(pagina.e.cache.arquivos, "arquivo baixado", "arquivos baixados") + "."
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
            }
            Botao {
                texto: "limpar o cache"
                ligado: pagina.e !== null && pagina.e.cache !== null && pagina.e.cache.arquivos > 0
                onAcionado: Acao.planejar("cache", [], false)
            }
        }
    }
}
