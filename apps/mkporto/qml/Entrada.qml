import QtQuick

// Um campo do formulário: o rótulo, o campo e, embaixo, o que há de errado
// com o que foi digitado (a linha fica reservada, para nada pular).
Column {
    id: entrada

    property string rotulo: ""
    property alias text: campo.text
    property alias dica: campo.dica
    property alias mono: campo.mono
    property string erro: ""
    signal aceito()

    function focar() { campo.focar(); }

    spacing: 2

    Rotulo {
        text: entrada.rotulo
        color: entrada.erro !== "" ? Theme.copper : Theme.textFaint
    }
    Campo {
        id: campo
        width: parent.width
        onAceito: entrada.aceito()
    }
    Text {
        width: parent.width
        height: 14
        elide: Text.ElideRight
        text: entrada.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 10
    }
}
