import QtQuick

// "Abrir com": os apps que abrem este tipo de arquivo (o padrão primeiro),
// "outro app…" para ver todos, e a chave de definir como padrão. Setas e
// Enter escolhem.
Folha {
    id: abrirCom

    // { caminhos, mime, nome }
    property var alvo: null
    property var apps: []
    property string padrao: ""
    property bool todos: false
    property bool definir: false
    property bool lendo: false
    property int atual: 0
    signal escolhido(var app, bool definir)
    signal pedido(bool todos)

    function abrir() {
        todos = false;
        definir = false;
        atual = 0;
    }
    function andar(delta) {
        if (apps.length > 0) {
            atual = Math.max(0, Math.min(apps.length - 1, atual + delta));
            lista.positionViewAtIndex(atual, ListView.Contain);
        }
    }
    function confirmar() {
        if (atual >= 0 && atual < apps.length)
            escolhido(apps[atual], definir);
    }

    rotulo: "abrir com"
    titulo: alvo ? alvo.nome : ""
    largura: 460

    Text {
        text: abrirCom.alvo ? abrirCom.alvo.mime : ""
        color: Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 11
    }

    ListView {
        id: lista
        width: parent.width
        height: Math.min(contentHeight, Math.max(120, abrirCom.height - 300))
        clip: true
        model: abrirCom.apps
        currentIndex: abrirCom.atual
        boundsBehavior: Flickable.StopAtBounds

        // Por cima da lista (não rola com ela).
        Rolagem {
            parent: lista
            z: 2
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            alvo: lista
        }

        delegate: Rectangle {
            id: linha
            required property var modelData
            required property int index
            width: lista.width
            height: 32
            color: linha.index === abrirCom.atual ? Theme.wash : Qt.rgba(0, 0, 0, 0)
            border.width: linha.index === abrirCom.atual ? 1 : 0
            border.color: Theme.copper

            Icone {
                id: figura
                x: 8
                anchors.verticalCenter: parent.verticalCenter
                lado: 20
                nome: linha.modelData.icone || "application-x-executable"
                reserva: "application-x-executable"
            }
            Text {
                anchors.left: figura.right
                anchors.leftMargin: 10
                anchors.right: marca.left
                anchors.verticalCenter: parent.verticalCenter
                elide: Text.ElideRight
                text: linha.modelData.nome
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 13
            }
            Rotulo {
                id: marca
                anchors.right: parent.right
                anchors.rightMargin: 8
                anchors.verticalCenter: parent.verticalCenter
                text: linha.modelData.padrao ? "padrão" : linha.modelData.terminal ? "terminal" : ""
            }
            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: abrirCom.atual = linha.index
                onDoubleClicked: {
                    abrirCom.atual = linha.index;
                    abrirCom.confirmar();
                }
            }
        }
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: abrirCom.apps.length === 0
        text: abrirCom.lendo ? "procurando os apps…"
            : abrirCom.todos ? "Não achei nenhum app instalado." : "Nenhum app diz abrir este tipo. Veja em «outro app…»."
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 10
        Chave {
            ligado: abrirCom.definir
            onTrocado: abrirCom.definir = !abrirCom.definir
        }
        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: "definir como padrão para este tipo"
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
        }
    }
    Row {
        spacing: 8
        Botao {
            texto: "abrir"
            forte: true
            ligado: abrirCom.apps.length > 0
            onAcionado: abrirCom.confirmar()
        }
        Botao {
            texto: abrirCom.todos ? "só os deste tipo" : "outro app…"
            onAcionado: {
                abrirCom.todos = !abrirCom.todos;
                abrirCom.atual = 0;
                abrirCom.pedido(abrirCom.todos);
            }
        }
        Botao {
            texto: "cancelar"
            onAcionado: abrirCom.fechado()
        }
    }
    Rotulo { text: "↑ ↓ escolhem  ·  enter abre  ·  esc cancela" }
}
