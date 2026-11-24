import QtQuick
import QtQuick.Dialogs
import "Util.js" as Util

// Onde os arquivos entram: soltando em cima (arrastados do gerenciador de
// arquivos) ou pelo botão, que abre o seletor de arquivos do sistema.
Rectangle {
    id: solta

    property string texto: ""
    property string titulo: "Escolher arquivos"
    property var filtros: []
    signal arquivos(var caminhos)

    // As URLs do seletor ou do arrasto viram caminhos; o que não é arquivo
    // local fica de fora.
    function entrega(urls) {
        const lista = [];
        for (let i = 0; i < urls.length; i++) {
            const caminho = Util.caminhoDeUrl(urls[i].toString());
            if (caminho !== "")
                lista.push(caminho);
        }
        if (lista.length > 0)
            solta.arquivos(lista);
    }

    implicitHeight: 58
    color: area.containsDrag ? Theme.wash : Qt.rgba(0, 0, 0, 0)
    border.width: 1
    border.color: area.containsDrag ? Theme.copper : Theme.line

    Text {
        anchors.left: parent.left
        anchors.leftMargin: 16
        anchors.right: escolher.left
        anchors.rightMargin: 12
        anchors.verticalCenter: parent.verticalCenter
        text: area.containsDrag ? "pode soltar" : solta.texto
        color: area.containsDrag ? Theme.copper : Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 14
        font.italic: true
        elide: Text.ElideRight
    }

    DropArea {
        id: area
        anchors.fill: parent
        onDropped: drop => {
            if (drop.hasUrls) {
                solta.entrega(drop.urls);
                drop.acceptProposedAction();
            }
        }
    }

    Botao {
        id: escolher
        anchors.right: parent.right
        anchors.rightMargin: 14
        anchors.verticalCenter: parent.verticalCenter
        texto: "escolher…"
        onAcionado: dialogo.open()
    }

    FileDialog {
        id: dialogo
        title: solta.titulo
        fileMode: FileDialog.OpenFiles
        nameFilters: solta.filtros
        onAccepted: solta.entrega(selectedFiles)
    }
}
