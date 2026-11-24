import QtQuick

// Uma pergunta por cima da janela: uma frase e poucas respostas, cada uma com
// a sua tecla. `opcoes` é [{ valor, texto, tecla, forte, enter, esc }]:
// `tecla` é a letra que responde, `enter` e `esc` dizem qual responde a Enter
// e a Esc. `interprete(tecla)`, se dado, decide a resposta no lugar disso
// (recebe a letra, "enter" ou "esc" e devolve um valor ou "").
Item {
    id: pergunta

    property string titulo: ""
    property string texto: ""
    property string detalhe: ""
    property var opcoes: []
    property var interprete: null
    signal respondida(string valor)

    function foca() {
        teclas.forceActiveFocus();
    }

    function responde(tecla) {
        if (pergunta.interprete) {
            const valor = pergunta.interprete(tecla);
            for (let i = 0; i < opcoes.length; i++)
                if (valor !== "" && opcoes[i].valor === valor)
                    return valor;
        }
        for (let i = 0; i < opcoes.length; i++) {
            const o = opcoes[i];
            if ((tecla === "enter" && o.enter) || (tecla === "esc" && o.esc)
                || (tecla.length === 1 && o.tecla && o.tecla.toLowerCase() === tecla.toLowerCase()))
                return o.valor;
        }
        return "";
    }

    anchors.fill: parent
    onVisibleChanged: if (visible) foca()

    // O texto fica esmaecido atrás da pergunta.
    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.72
    }

    // Um clique fora da folha não vaza para o texto.
    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.AllButtons
        hoverEnabled: true
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        id: folha
        anchors.centerIn: parent
        width: Math.min(parent.width - 48, 520)
        height: conteudo.implicitHeight + 48
        color: Theme.raised
        border.width: 1
        border.color: Theme.line

        Item {
            id: teclas
            Keys.onPressed: event => {
                let tecla = event.text;
                if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter)
                    tecla = "enter";
                else if (event.key === Qt.Key_Escape)
                    tecla = "esc";
                const valor = pergunta.responde(tecla);
                if (valor !== "")
                    pergunta.respondida(valor);
                // Nenhuma tecla passa para o texto enquanto a pergunta está na tela.
                event.accepted = true;
            }
        }

        Column {
            id: conteudo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 24
            spacing: 12

            Rotulo {
                text: pergunta.titulo
                visible: text !== ""
            }

            Text {
                width: parent.width
                text: pergunta.texto
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 17
                wrapMode: Text.Wrap
            }

            Text {
                width: parent.width
                visible: text !== ""
                text: pergunta.detalhe
                color: Theme.textDim
                font.family: Theme.serif
                font.pixelSize: 13
                font.italic: true
                wrapMode: Text.Wrap
                maximumLineCount: 3
                elide: Text.ElideRight
            }

            Item {
                width: 1
                height: 4
            }

            Flow {
                width: parent.width
                spacing: 10

                Repeater {
                    model: pergunta.opcoes

                    Botao {
                        required property var modelData
                        texto: modelData.texto + (modelData.tecla ? "  " + modelData.tecla : "")
                        forte: !!modelData.forte
                        onAcionado: pergunta.respondida(modelData.valor)
                    }
                }
            }
        }
    }
}
