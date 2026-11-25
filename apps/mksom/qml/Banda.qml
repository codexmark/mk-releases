import QtQuick
import "Som.js" as Som

// Uma banda do equalizador: o ganho em cima, o controle vertical no meio e a
// frequência embaixo. Arrastar avisa a cada passo (soltou = false) e ao
// soltar (soltou = true); pelo teclado, com o foco nela: ↑/↓ meio dB, Page
// Up/Down 3 dB, 0 zera.
Item {
    id: banda

    property real frequencia: 1000
    property real valor: 0
    property bool ligado: true
    property bool estreita: false
    // Com uma folha aberta por cima, o teclado não mexe na banda.
    property bool ativo: true
    signal mexido(real valor, bool soltou)

    activeFocusOnTab: true

    readonly property var teclas: ({
        [Qt.Key_Up]: "cima", [Qt.Key_Down]: "baixo", [Qt.Key_PageUp]: "pagina_cima", [Qt.Key_PageDown]: "pagina_baixo", [Qt.Key_0]: "zero"
    })
    Keys.onPressed: event => {
        const tecla = banda.teclas[event.key];
        if (!banda.ativo || !tecla || (event.modifiers & (Qt.ControlModifier | Qt.AltModifier | Qt.MetaModifier)))
            return;
        banda.mexido(Som.mover(banda.valor, tecla, false), true);
        event.accepted = true;
    }

    Text {
        id: numero
        anchors.top: parent.top
        anchors.horizontalCenter: parent.horizontalCenter
        text: Som.textoDoGanho(banda.valor)
        color: banda.activeFocus ? Theme.copper : banda.valor !== 0 ? Theme.text : Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: banda.estreita ? 9 : 10
    }

    Item {
        id: trilho
        anchors.top: numero.bottom
        anchors.bottom: nome.top
        anchors.topMargin: 8
        anchors.bottomMargin: 8
        anchors.horizontalCenter: parent.horizontalCenter
        width: Math.min(parent.width, banda.estreita ? 24 : 30)

        readonly property real pos: Som.paraPosicao(banda.valor, height, Som.GANHO_MIN, Som.GANHO_MAX)

        // O foco: uma moldura fina em volta do controle.
        Rectangle {
            anchors.fill: parent
            anchors.topMargin: -6
            anchors.bottomMargin: -6
            visible: banda.activeFocus
            color: Theme.wash
            border.width: 1
            border.color: Theme.copper
        }
        // O fio.
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            width: 2
            height: parent.height
            color: Theme.hairline
        }
        // O zero, no meio.
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height / 2
            width: parent.width - 8
            height: 1
            color: Theme.line
        }
        // Do zero até o ganho.
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            width: 2
            y: Math.min(trilho.pos, parent.height / 2)
            height: Math.abs(trilho.pos - parent.height / 2)
            color: banda.ligado ? Theme.copper : Theme.textDim
        }
        // O pegador.
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            y: trilho.pos - height / 2
            width: parent.width - 6
            height: 7
            color: banda.ligado ? Theme.copper : Theme.textDim
        }
    }

    Text {
        id: nome
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        text: Som.rotulo(banda.frequencia, banda.estreita)
        color: banda.activeFocus ? Theme.copper : Theme.textDim
        font.family: Theme.mono
        font.pixelSize: banda.estreita ? 8 : 9
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        function daqui(mouse) {
            return Som.daPosicao(mouse.y - trilho.y, trilho.height, Som.GANHO_MIN, Som.GANHO_MAX);
        }
        onPressed: mouse => {
            banda.forceActiveFocus();
            const v = daqui(mouse);
            if (v !== banda.valor)
                banda.mexido(v, false);
        }
        onPositionChanged: mouse => {
            const v = daqui(mouse);
            if (pressed && v !== banda.valor)
                banda.mexido(v, false);
        }
        onReleased: banda.mexido(banda.valor, true)
        // Dois cliques zeram a banda.
        onDoubleClicked: banda.mexido(0, true)
        onWheel: wheel => banda.mexido(Som.mover(banda.valor, wheel.angleDelta.y > 0 ? "cima" : "baixo", false), true)
    }
}
