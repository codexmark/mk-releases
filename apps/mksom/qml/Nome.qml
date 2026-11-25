import QtQuick
import "Som.js" as Som

// A folha do nome de uma predefinição: "guardar como…" e "renomear". O nome é
// conferido ao digitar com as mesmas regras do motor (Som.erroDoNome).
Item {
    id: nome

    // "" (guardar os valores de agora) ou o nome da que está sendo renomeada.
    property string de: ""
    property var lista: []
    property string erro: ""
    property bool ocupado: false
    readonly property string digitado: campo.text
    readonly property string problema: Som.erroDoNome(digitado, lista)
    readonly property bool vaiTrocar: de === "" && problema === "" && Som.acha(lista, Som.aparado(digitado)) !== null
    signal confirmado(string nome)
    signal fechado()

    function abrir(deQual, sugestao) {
        de = deQual;
        erro = "";
        campo.text = sugestao;
        campo.focar();
    }

    function confirmar() {
        if (!ocupado && problema === "")
            nome.confirmado(Som.aparado(digitado));
    }

    Fundo {
        anchors.fill: parent
        onFechado: nome.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 32, 380)
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

            Text {
                width: parent.width
                elide: Text.ElideRight
                text: nome.de === "" ? "Guardar como…" : "Renomear " + nome.de
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }
            Campo {
                id: campo
                width: parent.width
                dica: "o nome da predefinição"
                onAceito: nome.confirmar()
            }
            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                text: nome.erro !== "" ? nome.erro
                    : nome.digitado !== "" && nome.problema !== "" ? nome.problema
                    : nome.vaiTrocar ? "já existe: os valores de agora ficam no lugar dos dela"
                    : nome.de === "" ? "guarda os ganhos e o pré-ganho de agora" : "só o nome muda"
                color: nome.erro !== "" || (nome.digitado !== "" && nome.problema !== "") ? Theme.copper : Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Row {
                spacing: 8
                Botao {
                    texto: nome.ocupado ? "guardando…" : nome.de === "" ? "guardar" : "renomear"
                    forte: true
                    ligado: !nome.ocupado && nome.problema === ""
                    onAcionado: nome.confirmar()
                }
                Botao {
                    texto: "cancelar"
                    onAcionado: nome.fechado()
                }
            }
            Rotulo { text: "enter confirma  ·  esc cancela" }
        }
    }
}
