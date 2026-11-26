import QtQuick

// A folha que pede a senha de um compactado cifrado. Enter confirma.
Folha {
    id: pede

    property string frase: ""
    property string erro: ""
    signal confirmado(string segredo)

    function abrir(aviso) {
        campo.limpar();
        erro = aviso || "";
        foco.restart();
    }
    function confirmar() {
        if (campo.text === "") {
            erro = "escreva a senha";
            return;
        }
        const segredo = campo.text;
        campo.limpar();
        confirmado(segredo);
    }
    // Ao fechar, o que foi digitado some.
    onVisibleChanged: {
        if (!visible)
            campo.limpar();
    }

    rotulo: "senha"

    Timer {
        id: foco
        interval: 1
        onTriggered: campo.focar()
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        text: pede.frase !== "" ? pede.frase : "Este compactado é cifrado. A senha vai só para o motor, que a passa à ferramenta sem escrever em lugar nenhum."
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Senha {
        id: campo
        width: parent.width
        onAceito: pede.confirmar()
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: pede.erro !== ""
        text: pede.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 8
        Botao {
            texto: "continuar"
            forte: true
            onAcionado: pede.confirmar()
        }
        Botao {
            texto: "cancelar"
            onAcionado: pede.fechado()
        }
    }
    Rotulo { text: "enter confirma  ·  esc cancela" }
}
