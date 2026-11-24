import QtQuick
import QtQuick.Layouts
import "Calc.js" as Calc

// O histórico: as contas confirmadas com "=", a mais nova em cima. Clicar
// numa traz o resultado de volta para a expressão.
Item {
    id: painel

    property var itens: []
    // "limpar" pede uma segunda vez antes de apagar tudo.
    property bool armado: false
    signal escolhido(var item)
    signal apagado(var item)
    signal limpo()

    onVisibleChanged: armado = false

    ColumnLayout {
        anchors.fill: parent
        spacing: 8

        RowLayout {
            Layout.fillWidth: true
            spacing: 8

            Rotulo {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: painel.itens.length === 1 ? "histórico  ·  1 conta" : "histórico  ·  " + painel.itens.length + " contas"
            }
            Botao {
                texto: painel.armado ? "apagar tudo?" : "limpar"
                forte: painel.armado
                ligado: painel.itens.length > 0
                onAcionado: {
                    if (!painel.armado) {
                        painel.armado = true;
                        return;
                    }
                    painel.armado = false;
                    painel.limpo();
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 1
            color: Theme.hairline
        }

        Text {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: painel.itens.length === 0
            text: "Nenhuma conta ainda.\nO que você confirma com = fica aqui."
            color: Theme.textFaint
            font.family: Theme.sans
            font.pixelSize: 12
            wrapMode: Text.WordWrap
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
        }

        ListView {
            id: lista
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: painel.itens.length > 0
            clip: true
            spacing: 4
            boundsBehavior: Flickable.StopAtBounds
            model: painel.itens

            delegate: Rectangle {
                id: linha
                required property var modelData
                width: lista.width
                height: 52
                color: sobre.containsMouse ? Theme.wash : Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: sobre.containsMouse ? Theme.line : Theme.hairline

                MouseArea {
                    id: sobre
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: painel.escolhido(linha.modelData)
                }

                // Tirar só esta conta.
                Text {
                    id: tirar
                    anchors.left: parent.left
                    anchors.leftMargin: 10
                    anchors.verticalCenter: parent.verticalCenter
                    text: "×"
                    color: tirarArea.containsMouse ? Theme.copper : Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 14
                    opacity: sobre.containsMouse || tirarArea.containsMouse ? 1 : 0.35

                    MouseArea {
                        id: tirarArea
                        anchors.fill: parent
                        anchors.margins: -8
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: painel.apagado(linha.modelData)
                    }
                }

                Column {
                    anchors.left: tirar.right
                    anchors.leftMargin: 12
                    anchors.right: parent.right
                    anchors.rightMargin: 10
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 3

                    Text {
                        width: parent.width
                        text: Calc.mostrar(linha.modelData.expressao.replace(/\s+/g, ""))
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 11
                        horizontalAlignment: Text.AlignRight
                        elide: Text.ElideLeft
                    }
                    Text {
                        width: parent.width
                        text: (linha.modelData.aproximado ? "≈ " : "= ") + linha.modelData.resultado
                        color: Theme.text
                        font.family: Theme.mono
                        font.pixelSize: 16
                        horizontalAlignment: Text.AlignRight
                        elide: Text.ElideLeft
                    }
                }
            }
        }
    }
}
