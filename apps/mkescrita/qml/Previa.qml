import QtQuick
import "Escrita.js" as Escrita

// A pré-visualização (Ctrl+Shift+P): o HTML que o motor fez do texto, num
// Text em RichText. O motor já escapou tudo que veio do texto; aqui só entra
// na frente a folha de estilo com as cores do tema. A rolagem acompanha a do
// texto por proporção (aproximada).
Rectangle {
    id: previa

    property string html: ""
    // Onde a rolagem do texto está, de 0 a 1.
    property real proporcao: 0
    // A pasta do documento, para as imagens de caminho relativo.
    property string base: ""
    property int zoom: 100

    readonly property string estilo: Escrita.estiloDaPrevia({
        destaque: Theme.copper.toString(), texto: Theme.text.toString(),
        suave: Theme.textSoft.toString(), apagado: Theme.textDim.toString()
    }, Theme.mono)

    color: Theme.surface
    clip: true

    Rectangle {
        width: 1
        height: parent.height
        color: Theme.hairline
    }

    Flickable {
        id: rolo
        anchors.fill: parent
        anchors.leftMargin: 1
        contentWidth: width
        contentHeight: folha.height + 96
        boundsBehavior: Flickable.StopAtBounds
        contentY: Math.max(0, contentHeight - height) * previa.proporcao

        Text {
            id: folha
            x: Math.max(28, (rolo.width - width) / 2)
            y: 48
            width: Math.min(rolo.width - 56, 640)
            textFormat: Text.RichText
            text: previa.html === "" ? "" : previa.estilo + previa.html
            baseUrl: previa.base
            color: Theme.text
            linkColor: Theme.copper
            font.family: Theme.serif
            font.pixelSize: Math.round(16 * previa.zoom / 100)
            wrapMode: Text.Wrap
            onLinkActivated: link => {
                if (Escrita.linkAbrivel(link))
                    Qt.openUrlExternally(link);
            }
        }

        Text {
            anchors.centerIn: parent
            visible: previa.html === ""
            text: "a pré-visualização aparece aqui"
            color: Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: 14
            font.italic: true
        }
    }
}
