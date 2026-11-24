import QtQuick
import "Escrita.js" as Escrita

// O seletor de fonte do rodapé: as famílias instaladas (Qt.fontFamilies(),
// pelo Theme), com um filtro. A primeira entrada volta à monoespaçada do tema.
Rectangle {
    id: fontes

    property var familias: []
    // A família em uso ("" = a do tema).
    property string emUso: ""
    readonly property var filtradas: [""].concat(Escrita.filtrarFontes(familias, filtro.text))
    signal escolhida(string familia)
    signal fechar()

    function foca() {
        filtro.text = "";
        lista.currentIndex = Math.max(0, filtradas.indexOf(emUso));
        filtro.foca();
    }

    function confirma() {
        if (lista.currentIndex >= 0 && lista.currentIndex < filtradas.length)
            fontes.escolhida(filtradas[lista.currentIndex]);
    }

    width: 300
    height: 340
    color: Theme.raised
    border.width: 1
    border.color: Theme.line

    Linha {
        id: filtro
        x: 14
        y: 10
        width: parent.width - 28
        dica: "fonte"
        onTextChanged: lista.currentIndex = 0
        onConfirmado: fontes.confirma()
        onCancelado: fontes.fechar()
        onDesce: lista.currentIndex = Math.min(fontes.filtradas.length - 1, lista.currentIndex + 1)
        onSobe: lista.currentIndex = Math.max(0, lista.currentIndex - 1)
    }

    ListView {
        id: lista
        anchors.top: filtro.bottom
        anchors.topMargin: 8
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 6
        x: 1
        width: parent.width - 2
        clip: true
        model: fontes.filtradas
        boundsBehavior: Flickable.StopAtBounds
        highlightMoveDuration: 0

        delegate: Rectangle {
            id: item
            required property string modelData
            required property int index
            width: lista.width
            height: 28
            color: area.containsMouse || lista.currentIndex === index ? Theme.wash : Qt.rgba(0, 0, 0, 0)

            Text {
                anchors.verticalCenter: parent.verticalCenter
                x: 14
                width: parent.width - 28
                text: item.modelData === "" ? "a do tema (" + Theme.mono + ")" : item.modelData
                color: item.modelData === fontes.emUso ? Theme.copper : Theme.textSoft
                font.family: item.modelData === "" ? Theme.mono : item.modelData
                font.pixelSize: 13
                elide: Text.ElideRight
            }

            MouseArea {
                id: area
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: fontes.escolhida(item.modelData)
            }
        }
    }
}
