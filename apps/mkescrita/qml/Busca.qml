import QtQuick

// A barra de buscar (Ctrl+F) e substituir (Ctrl+H). Enter ou Ctrl+G vai para
// a próxima, Shift+Enter para a anterior, Esc fecha. Quem acha as ocorrências
// é a página (Escrita.ocorrencias); aqui só os campos e os botões.
Rectangle {
    id: busca

    property bool comSubstituir: false
    property bool diferenciar: false
    property alias termo: campoDoTermo.text
    property alias por: campoDoPor.text
    property string resultado: ""
    signal proxima()
    signal anterior()
    signal trocarUma()
    signal trocarTodas()
    signal fechar()

    function foca(noSubstituir) {
        if (noSubstituir && comSubstituir && campoDoTermo.text !== "")
            campoDoPor.foca();
        else
            campoDoTermo.foca();
    }

    implicitHeight: comSubstituir ? 76 : 42
    color: Theme.surface

    Rectangle {
        anchors.top: parent.top
        width: parent.width
        height: 1
        color: Theme.hairline
    }

    Row {
        id: primeira
        anchors.left: parent.left
        anchors.leftMargin: 16
        y: 7
        spacing: 12

        Linha {
            id: campoDoTermo
            width: 260
            dica: "buscar"
            onConfirmado: comShift => comShift ? busca.anterior() : busca.proxima()
            onCancelado: busca.fechar()
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            width: 110
            text: busca.resultado
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Botao {
            texto: "Aa"
            forte: busca.diferenciar
            onAcionado: busca.diferenciar = !busca.diferenciar
        }

        Botao {
            texto: "anterior"
            onAcionado: busca.anterior()
        }

        Botao {
            texto: "próxima"
            onAcionado: busca.proxima()
        }
    }

    Row {
        anchors.left: parent.left
        anchors.leftMargin: 16
        y: 41
        spacing: 12
        visible: busca.comSubstituir

        Linha {
            id: campoDoPor
            width: 260
            dica: "substituir por"
            onConfirmado: busca.trocarUma()
            onCancelado: busca.fechar()
        }

        Item {
            width: 110
            height: 1
        }

        Botao {
            texto: "trocar"
            onAcionado: busca.trocarUma()
        }

        Botao {
            texto: "trocar todas"
            onAcionado: busca.trocarTodas()
        }
    }

    Botao {
        anchors.right: parent.right
        anchors.rightMargin: 16
        y: 7
        texto: "fechar"
        onAcionado: busca.fechar()
    }
}
