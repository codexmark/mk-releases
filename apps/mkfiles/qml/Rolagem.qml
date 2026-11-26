import QtQuick

// A barra de rolagem de uma lista (ou de qualquer Flickable): mostra onde se
// está e dá para arrastar; um clique fora do pegador leva a lista para lá.
// Fica por cima da beira da lista: quem usa dá as âncoras.
Item {
    id: rolagem

    // O Flickable (ListView, GridView…) que ela rola.
    property var alvo: null
    property bool horizontal: false
    readonly property real conteudo: !alvo ? 0 : horizontal ? alvo.contentWidth : alvo.contentHeight
    readonly property real aVista: !alvo ? 0 : horizontal ? alvo.width : alvo.height
    readonly property real origem: !alvo ? 0 : horizontal ? alvo.originX : alvo.originY
    readonly property real posicao: !alvo ? 0 : (horizontal ? alvo.contentX : alvo.contentY) - origem
    readonly property real trilho: horizontal ? width : height
    readonly property real pegador: conteudo > 0 ? Math.max(28, Math.min(trilho, aVista / conteudo * trilho)) : 0
    readonly property real folga: Math.max(1, trilho - pegador)
    readonly property real onde: conteudo > aVista ? Math.max(0, Math.min(folga, posicao / (conteudo - aVista) * folga)) : 0
    readonly property bool arrastando: area.pressed

    // Leva a lista para o ponto em que o pegador começa em `p` (no trilho).
    function levar(p) {
        if (!alvo || conteudo <= aVista)
            return;
        const destino = origem + Math.max(0, Math.min(1, p / folga)) * (conteudo - aVista);
        if (horizontal)
            alvo.contentX = destino;
        else
            alvo.contentY = destino;
    }

    visible: alvo !== null && conteudo > aVista + 1
    implicitWidth: 10
    implicitHeight: 10

    Rectangle {
        x: rolagem.horizontal ? rolagem.onde : rolagem.width - width - 2
        y: rolagem.horizontal ? rolagem.height - height - 2 : rolagem.onde
        width: rolagem.horizontal ? rolagem.pegador : (area.containsMouse || area.pressed ? 6 : 3)
        height: rolagem.horizontal ? (area.containsMouse || area.pressed ? 6 : 3) : rolagem.pegador
        color: area.pressed ? Theme.copper : Theme.boneA(area.containsMouse ? 0.5 : 0.3)
    }
    MouseArea {
        id: area
        anchors.fill: parent
        hoverEnabled: true
        preventStealing: true
        // Onde, dentro do pegador, ele foi agarrado.
        property real pega: 0
        onPressed: mouse => {
            const p = rolagem.horizontal ? mouse.x : mouse.y;
            if (p < rolagem.onde || p > rolagem.onde + rolagem.pegador) {
                // Fora do pegador: ele vem para o ponto do clique.
                pega = rolagem.pegador / 2;
                rolagem.levar(p - pega);
            } else {
                pega = p - rolagem.onde;
            }
        }
        onPositionChanged: mouse => {
            if (pressed)
                rolagem.levar((rolagem.horizontal ? mouse.x : mouse.y) - pega);
        }
        onWheel: wheel => wheel.accepted = false
    }
}
