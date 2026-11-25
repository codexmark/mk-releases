import QtQuick

// Remover uma máquina leva o disco junto e não tem volta: a folha só deixa
// depois que a pessoa digita o nome dela.
Item {
    id: folha

    property string nome: ""
    property string erro: ""
    property bool ocupado: false
    readonly property bool confere: digitado.text === nome && nome !== ""
    signal confirmado(string digitado)
    signal fechado()

    function abrir() {
        digitado.text = "";
        digitado.focar();
    }

    function tenta() {
        if (confere && !ocupado)
            confirmado(digitado.text);
    }

    Fundo {
        anchors.fill: parent
        onFechado: folha.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(460, parent.width - 32)
        height: corpo.implicitHeight + 44
        color: Theme.surface
        border.width: 1
        border.color: Theme.line

        MouseArea { anchors.fill: parent }

        Column {
            id: corpo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 10

            Rotulo { text: "remover a máquina" }
            Text {
                width: parent.width
                wrapMode: Text.WrapAnywhere
                text: "Remover " + folha.nome + "?"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                text: "A pasta da máquina é apagada inteira: o disco, os snapshots e a configuração. Não há lixeira. Para confirmar, digite o nome dela."
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Entrada {
                id: digitado
                width: parent.width
                rotulo: "o nome da máquina"
                dica: folha.nome
                mono: true
                erro: folha.erro
                onAceito: folha.tenta()
            }
            Row {
                spacing: 8
                Botao {
                    texto: folha.ocupado ? "removendo…" : "remover de vez"
                    forte: true
                    ligado: folha.confere && !folha.ocupado
                    onAcionado: folha.tenta()
                }
                Botao {
                    texto: "cancelar"
                    onAcionado: folha.fechado()
                }
            }
            Rotulo { text: "esc cancela" }
        }
    }
}
