import QtQuick
import "Som.js" as Som

// Um controle deitado (o pré-ganho, o volume da entrada): um fio com um
// pegador. Arrastar avisa a cada passo e ao soltar; com o foco: ←/→ (ou ↓/↑)
// um passo, Page Up/Down um passo grande.
Item {
    id: deslizante

    property real valor: 0
    property real min: 0
    property real max: 100
    property real passo: 1
    property real passoGrande: 10
    property bool ligado: true
    property bool ativo: true
    signal mexido(real valor, bool soltou)

    implicitWidth: 160
    implicitHeight: 22
    activeFocusOnTab: true

    readonly property var teclas: ({
        [Qt.Key_Right]: passo, [Qt.Key_Up]: passo, [Qt.Key_Left]: -passo, [Qt.Key_Down]: -passo,
        [Qt.Key_PageUp]: passoGrande, [Qt.Key_PageDown]: -passoGrande
    })
    Keys.onPressed: event => {
        const delta = deslizante.teclas[event.key];
        if (!deslizante.ativo || delta === undefined || (event.modifiers & (Qt.ControlModifier | Qt.AltModifier | Qt.MetaModifier)))
            return;
        deslizante.mexido(Som.andar(deslizante.valor, delta, deslizante.min, deslizante.max, deslizante.passo), true);
        event.accepted = true;
    }

    readonly property real pos: (Som.prender(valor, min, max) - min) / (max - min) * fio.width

    Rectangle {
        id: fio
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: 5
        anchors.rightMargin: 5
        anchors.verticalCenter: parent.verticalCenter
        height: 2
        color: deslizante.activeFocus ? Theme.line : Theme.hairline
    }
    Rectangle {
        anchors.left: fio.left
        anchors.verticalCenter: parent.verticalCenter
        width: deslizante.pos
        height: 2
        color: deslizante.ligado ? Theme.copper : Theme.textDim
    }
    Rectangle {
        x: fio.x + deslizante.pos - width / 2
        anchors.verticalCenter: parent.verticalCenter
        width: 8
        height: 16
        color: deslizante.ligado ? Theme.copper : Theme.textDim
        border.width: deslizante.activeFocus ? 1 : 0
        border.color: Theme.text
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        function daqui(mouse) {
            return Som.noTrilho(mouse.x - fio.x, fio.width, deslizante.min, deslizante.max, deslizante.passo);
        }
        onPressed: mouse => {
            deslizante.forceActiveFocus();
            const v = daqui(mouse);
            if (v !== deslizante.valor)
                deslizante.mexido(v, false);
        }
        onPositionChanged: mouse => {
            const v = daqui(mouse);
            if (pressed && v !== deslizante.valor)
                deslizante.mexido(v, false);
        }
        onReleased: deslizante.mexido(deslizante.valor, true)
        onWheel: wheel => deslizante.mexido(
            Som.andar(deslizante.valor, wheel.angleDelta.y > 0 ? deslizante.passo : -deslizante.passo, deslizante.min, deslizante.max, deslizante.passo), true)
    }
}
