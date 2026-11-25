import QtQuick
import "Porto.js" as Porto

// Os snapshots de uma máquina (o qemu-img, no disco dela): a lista, criar um
// com rótulo, e restaurar ou apagar (cada um com a sua pergunta). Só com a
// máquina desligada: quem abre a folha já conferiu, e o motor recusa de novo.
Item {
    id: folha

    property string nome: ""
    property var snapshots: []
    property bool lendo: false
    property string erro: ""
    property bool ocupado: false
    signal criar(string rotulo)
    signal restaurar(string rotulo)
    signal apagar(string rotulo)
    signal fechado()

    readonly property string erroDoRotulo: rotulo.text === "" ? "" : Porto.erroDoRotulo(rotulo.text)

    function abrir() {
        rotulo.text = "";
        rotulo.focar();
    }

    function tenta() {
        if (rotulo.text !== "" && erroDoRotulo === "" && !ocupado)
            criar(rotulo.text);
    }

    Fundo {
        anchors.fill: parent
        onFechado: folha.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(560, parent.width - 32)
        height: Math.min(parent.height - 24, corpo.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea { anchors.fill: parent }

        Flickable {
            id: rolo
            anchors.fill: parent
            anchors.margins: 22
            contentHeight: corpo.implicitHeight
            boundsBehavior: Flickable.StopAtBounds
            clip: true

            Column {
                id: corpo
                width: rolo.width
                spacing: 10

                Rotulo { text: "snapshots  /  " + folha.nome }
                Text {
                    text: "O disco, como estava"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                }
                Row {
                    width: parent.width
                    spacing: 8
                    Entrada {
                        id: rotulo
                        width: parent.width - guardar.width - 8
                        rotulo: "novo snapshot"
                        dica: "antes-de-atualizar"
                        mono: true
                        erro: folha.erroDoRotulo
                        onAceito: folha.tenta()
                    }
                    Botao {
                        id: guardar
                        y: 14
                        texto: "criar"
                        forte: true
                        ligado: rotulo.text !== "" && folha.erroDoRotulo === "" && !folha.ocupado
                        onAcionado: folha.tenta()
                    }
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: folha.erro !== ""
                    text: folha.erro
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Text {
                    width: parent.width
                    visible: folha.snapshots.length === 0
                    text: folha.lendo ? "lendo os snapshots…" : "nenhum snapshot ainda"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }

                Repeater {
                    model: folha.snapshots

                    Rectangle {
                        id: item
                        required property var modelData
                        width: corpo.width
                        height: 46
                        color: Qt.rgba(0, 0, 0, 0)
                        border.width: 1
                        border.color: Theme.hairline

                        Column {
                            anchors.left: parent.left
                            anchors.right: acoes.left
                            anchors.margins: 12
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 3
                            Text {
                                width: parent.width
                                elide: Text.ElideMiddle
                                text: item.modelData.rotulo
                                color: Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                            }
                            Text {
                                width: parent.width
                                elide: Text.ElideRight
                                text: item.modelData.data + "  ·  " + item.modelData.tamanho
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                        }
                        Row {
                            id: acoes
                            anchors.right: parent.right
                            anchors.rightMargin: 10
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 6
                            Botao {
                                texto: "restaurar"
                                ligado: !folha.ocupado
                                onAcionado: folha.restaurar(item.modelData.rotulo)
                            }
                            Botao {
                                texto: "apagar"
                                ligado: !folha.ocupado
                                onAcionado: folha.apagar(item.modelData.rotulo)
                            }
                        }
                    }
                }

                Botao {
                    texto: "fechar"
                    onAcionado: folha.fechado()
                }
            }
        }
    }
}
