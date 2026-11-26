import QtQuick
import QtQuick.Layouts
import "Arquivos.js" as Arquivos
import "Abas.js" as Abas
import "Remoto.js" as Remoto

// A barra de cima: voltar, avançar, subir; o caminho em migalhas clicáveis,
// que vira um campo editável com Ctrl+L (Tab completa nomes de pasta); a
// busca; o seletor de visão; e o menu.
Rectangle {
    id: barra

    property var aba: null
    property string home: "/"
    property string visao: "icones"
    property bool estreita: false
    property bool editando: false
    property bool buscando: false
    property var opcoes: []
    // Os servidores montados: num deles as migalhas são "servidor › pasta".
    property var remotos: []
    readonly property bool comFoco: caminho.activeFocus || busca.activeFocus
    readonly property string onde: aba ? aba.caminho : "/"
    readonly property bool lixeira: Abas.eLixeira(onde)

    signal voltar()
    signal avancar()
    signal subir()
    signal ir(string caminho)
    signal completar(string texto)
    signal buscaMudou(string texto)
    signal buscaFechada()
    signal visaoEscolhida(string v)
    signal menuPedido(real x, real y)
    signal largado()

    function editar() {
        editando = true;
        // Dentro de um compactado, o que se edita é a pasta em que ele está.
        const real = Arquivos.pastaReal(onde);
        caminho.text = lixeira ? "" : Arquivos.til(real, home) + (real === "/" || real === home ? "" : "/");
        if (real === home)
            caminho.text = "~/";
        caminho.forceActiveFocus();
        caminho.selectAll();
    }
    function completado(texto, lista) {
        caminho.text = texto;
        caminho.cursorPosition = texto.length;
        opcoes = lista;
    }
    function buscar() {
        buscando = true;
        busca.forceActiveFocus();
        busca.selectAll();
    }
    function sair() {
        editando = false;
        opcoes = [];
    }
    function fecharBusca() {
        buscando = false;
        busca.text = "";
    }

    implicitHeight: 42 + (opcoes.length > 0 && editando ? 22 : 0)
    color: Theme.surface

    RowLayout {
        id: fila
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: 7
        height: 28
        spacing: 6

        Botao {
            implicitWidth: 30
            texto: "←"
            ligado: !!barra.aba && Abas.podeVoltar(barra.aba)
            onAcionado: barra.voltar()
        }
        Botao {
            implicitWidth: 30
            texto: "→"
            ligado: !!barra.aba && Abas.podeAvancar(barra.aba)
            onAcionado: barra.avancar()
        }
        Botao {
            implicitWidth: 30
            texto: "↑"
            ligado: !!barra.aba && Abas.podeSubir(barra.aba)
            onAcionado: barra.subir()
        }

        // O caminho: migalhas, ou o campo.
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumWidth: 80
            color: Theme.ground
            border.width: 1
            border.color: caminho.activeFocus ? Theme.copper : Theme.hairline
            clip: true

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.IBeamCursor
                onClicked: barra.editar()
            }

            // As migalhas ficam coladas à direita quando não cabem: o fim do
            // caminho é o que importa.
            Row {
                id: migalhas
                visible: !barra.editando
                x: Math.min(8, parent.width - width - 8)
                anchors.verticalCenter: parent.verticalCenter
                spacing: 2

                Repeater {
                    model: barra.lixeira ? [{ nome: "Lixeira", caminho: Abas.LIXEIRA }]
                        : Remoto.migalhas(barra.onde, barra.remotos) || Arquivos.migalhas(barra.onde, barra.home)

                    Row {
                        id: migalha
                        required property var modelData
                        required property int index
                        spacing: 2
                        Text {
                            visible: migalha.index > 0
                            text: "›"
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                        Text {
                            text: migalha.modelData.nome
                            color: migalhaArea.containsMouse ? Theme.copper : migalha.modelData.caminho === barra.onde ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 13
                            MouseArea {
                                id: migalhaArea
                                anchors.fill: parent
                                anchors.margins: -3
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: barra.ir(migalha.modelData.caminho)
                            }
                        }
                    }
                }
            }

            TextInput {
                id: caminho
                visible: barra.editando
                anchors.fill: parent
                anchors.leftMargin: 8
                anchors.rightMargin: 8
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.text
                selectionColor: Theme.copper
                font.family: Theme.mono
                font.pixelSize: 12
                clip: true
                selectByMouse: true
                Keys.onReturnPressed: barra.ir(text)
                Keys.onEnterPressed: barra.ir(text)
                Keys.onTabPressed: barra.completar(text)
                Keys.onEscapePressed: barra.largado()
                onTextEdited: barra.opcoes = []
                onActiveFocusChanged: {
                    if (!activeFocus)
                        barra.sair();
                }
            }
        }

        // A busca.
        Rectangle {
            Layout.preferredWidth: barra.buscando ? (barra.estreita ? 130 : 200) : 30
            Layout.fillHeight: true
            color: barra.buscando ? Theme.ground : Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: busca.activeFocus ? Theme.copper : lupa.containsMouse ? Theme.copper : Theme.line
            clip: true

            Text {
                visible: !barra.buscando
                anchors.centerIn: parent
                text: "⌕"
                color: Theme.textSoft
                font.pixelSize: 15
            }
            MouseArea {
                id: lupa
                anchors.fill: parent
                enabled: !barra.buscando
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: barra.buscar()
            }
            TextInput {
                id: busca
                visible: barra.buscando
                anchors.fill: parent
                anchors.leftMargin: 8
                anchors.rightMargin: 8
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.text
                selectionColor: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 13
                clip: true
                selectByMouse: true
                onTextEdited: atraso.restart()
                Keys.onReturnPressed: {
                    atraso.stop();
                    barra.buscaMudou(text);
                    barra.largado();
                }
                Keys.onEnterPressed: {
                    atraso.stop();
                    barra.buscaMudou(text);
                    barra.largado();
                }
                Keys.onDownPressed: barra.largado()
                Keys.onEscapePressed: barra.buscaFechada()

                Text {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: busca.text === ""
                    text: "buscar aqui"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 13
                    font.italic: true
                }
            }
            // A busca sai enquanto se digita, sem um pedido por tecla.
            Timer {
                id: atraso
                interval: 300
                onTriggered: barra.buscaMudou(busca.text)
            }
        }

        Segmento {
            opcoes: barra.estreita
                ? [{ v: "icones", texto: "▦" }, { v: "lista", texto: "☰" }, { v: "compacta", texto: "▤" }]
                : [{ v: "icones", texto: "ícones" }, { v: "lista", texto: "lista" }, { v: "compacta", texto: "compacta" }]
            valor: barra.visao
            onEscolhido: v => barra.visaoEscolhida(v)
        }
        Botao {
            id: botaoDoMenu
            implicitWidth: 30
            texto: "⋮"
            onAcionado: {
                const p = botaoDoMenu.mapToItem(null, 0, botaoDoMenu.height + 4);
                barra.menuPedido(p.x, p.y);
            }
        }
    }

    // As pastas que ainda cabem no que foi digitado (depois de um Tab).
    Text {
        visible: barra.editando && barra.opcoes.length > 0
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.leftMargin: 12
        anchors.rightMargin: 12
        anchors.bottomMargin: 5
        elide: Text.ElideRight
        text: barra.opcoes.join("   ")
        color: Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 11
    }

    Rectangle {
        anchors.bottom: parent.bottom
        width: parent.width
        height: 1
        color: Theme.hairline
    }
}
