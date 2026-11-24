import QtQuick
import "Corte.js" as Corte

// A linha do tempo: os clipes na ordem, cada um com a tira de quadros e uma
// alça em cada ponta. Arrastar a alça apara; arrastar o clipe reordena; um
// clique leva o cursor; duplo clique divide; o × (ao passar o ponteiro)
// remove. Os clipes são só desenho: um MouseArea único cobre a linha e
// pergunta ao Corte.toque o que há sob o ponteiro, então nada se perde quando
// a edição muda no meio de um arrasto.
Item {
    id: linha

    property var ed: Corte.vazia()
    property real cursor: 0
    property bool zoom: false
    // arquivo → os quadros da tira (a resposta do `miniaturas`).
    property var tiras: ({})

    // Durante um arrasto de alça a escala fica parada (a duração muda a cada
    // movimento, e a linha não pode escorregar debaixo do ponteiro).
    property var congelada: null
    readonly property var v: congelada || Corte.vista(ed, cursor, width, zoom)
    readonly property real escala: v.escala
    readonly property real desloca: v.desloca
    readonly property var atual: Corte.localizar(ed, cursor)

    // O que está sob o ponteiro, e o arrasto em curso:
    // { tipo, clipe, x0, t0, moveu }.
    property int sobre: -1
    property string sobreTipo: ""
    property var arrasto: null
    property real marca: -1

    signal irPara(real t)
    signal aparando(int clipe, string lado, real tempoNaFonte, real tolerancia)
    signal aparou()
    signal dividirEm(real t)
    signal removerClipe(int clipe)
    signal moverClipe(int de, int para)

    function apertou(x, y) {
        const q = Corte.toque(linha.ed, linha.v, x, y);
        if (q.tipo === "nada") {
            linha.irPara(Corte.tempoDoX(linha.v, x));
            return;
        }
        const c = linha.ed.clipes[q.clipe];
        linha.arrasto = { tipo: q.tipo, clipe: q.clipe, x0: x, t0: q.tipo === "inicio" ? c.inicio : c.fim, moveu: false };
        if (q.tipo === "inicio" || q.tipo === "fim")
            linha.congelada = linha.v;
    }

    function moveu(x, y) {
        const a = linha.arrasto;
        if (!a) {
            const q = Corte.toque(linha.ed, linha.v, x, y);
            linha.sobre = q.clipe;
            linha.sobreTipo = q.tipo;
            return;
        }
        if (a.tipo === "remover" || (!a.moveu && Math.abs(x - a.x0) < 4))
            return;
        if (!a.moveu)
            linha.arrasto = Object.assign({}, a, { moveu: true });
        if (a.tipo === "clipe") {
            linha.marca = Corte.marcaDoArrasto(linha.ed, linha.v, a.clipe, Corte.tempoDoX(linha.v, x));
            return;
        }
        linha.aparando(a.clipe, a.tipo, a.t0 + (x - a.x0) / linha.escala, 8 / linha.escala);
    }

    function soltou(x, y) {
        const a = linha.arrasto, vista = linha.v;
        linha.arrasto = null;
        linha.marca = -1;
        if (!a) {
            linha.congelada = null;
            return;
        }
        if (a.tipo === "remover") {
            const q = Corte.toque(linha.ed, vista, x, y);
            if (q.tipo === "remover" && q.clipe === a.clipe)
                linha.removerClipe(a.clipe);
        } else if (!a.moveu) {
            linha.irPara(Corte.tempoDoX(vista, x));
        } else if (a.tipo === "clipe") {
            linha.moverClipe(a.clipe, Corte.destinoDoArrasto(linha.ed, a.clipe, Corte.tempoDoX(vista, x)));
        } else {
            linha.aparou();
        }
        linha.congelada = null;
    }

    implicitHeight: 84
    clip: true

    Repeater {
        model: linha.ed.clipes

        Rectangle {
            id: clipe
            required property var modelData
            required property int index
            readonly property bool doCursor: linha.atual !== null && linha.atual.clipe === index
            readonly property bool apontado: linha.sobre === index
            readonly property bool levado: linha.arrasto !== null && linha.arrasto.tipo === "clipe" && linha.arrasto.moveu && linha.arrasto.clipe === index

            x: Corte.inicioDoClipe(linha.ed, index) * linha.escala - linha.desloca
            width: Math.max(2, Corte.duracaoDoClipe(modelData) * linha.escala)
            height: linha.height
            color: Theme.surface
            opacity: levado ? 0.45 : 1
            clip: true

            // A tira de quadros.
            Repeater {
                model: Corte.fatias(clipe.modelData, clipe.width, clipe.height * 16 / 9, linha.tiras[linha.ed.fontes[clipe.modelData.fonte].arquivo] || [])

                Image {
                    required property var modelData
                    x: modelData.x
                    width: modelData.largura
                    height: clipe.height
                    source: Corte.urlDeCaminho(modelData.arquivo)
                    fillMode: Image.PreserveAspectCrop
                    clip: true
                    opacity: clipe.doCursor ? 0.9 : 0.55
                }
            }

            // O contorno, por cima da tira: cobre no clipe sob o cursor.
            Rectangle {
                anchors.fill: parent
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: clipe.doCursor ? Theme.copper : Theme.line
            }

            // As alças.
            Rectangle {
                width: 4
                height: parent.height
                color: clipe.apontado && linha.sobreTipo === "inicio" ? Theme.copper : Theme.boneA(0.45)
            }
            Rectangle {
                anchors.right: parent.right
                width: 4
                height: parent.height
                color: clipe.apontado && linha.sobreTipo === "fim" ? Theme.copper : Theme.boneA(0.45)
            }

            // O número e a duração.
            Rectangle {
                visible: clipe.width >= 64
                x: 8
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 5
                width: etiqueta.implicitWidth + 10
                height: 16
                color: Theme.ground
                opacity: 0.85
            }
            Text {
                id: etiqueta
                visible: clipe.width >= 64
                x: 13
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 7
                text: (clipe.index + 1) + "  ·  " + Corte.tempo(Corte.duracaoDoClipe(clipe.modelData))
                color: clipe.doCursor ? Theme.text : Theme.textSoft
                font.family: Theme.mono
                font.pixelSize: 9
            }

            // O × de remover, só com o ponteiro em cima (as medidas são as do Corte.toque).
            Rectangle {
                visible: clipe.apontado && clipe.width >= 44 && (linha.arrasto === null || linha.arrasto.tipo === "remover")
                x: clipe.width - 28
                y: 2
                width: 18
                height: 18
                color: Theme.ground
                border.width: 1
                border.color: linha.sobreTipo === "remover" ? Theme.copper : Theme.line

                Text {
                    anchors.centerIn: parent
                    text: "×"
                    color: linha.sobreTipo === "remover" ? Theme.copper : Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 13
                }
            }
        }
    }

    // Onde o clipe arrastado vai cair.
    Rectangle {
        visible: linha.marca >= 0
        x: Math.max(0, Math.min(linha.width - width, linha.marca - 1))
        width: 3
        height: parent.height
        color: Theme.copper
    }

    // O cursor.
    Rectangle {
        visible: linha.ed.clipes.length > 0
        x: Math.max(0, Math.min(linha.width - width, linha.cursor * linha.escala - linha.desloca - 1))
        width: 2
        height: parent.height
        color: Theme.text
    }

    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        preventStealing: true
        cursorShape: linha.sobreTipo === "inicio" || linha.sobreTipo === "fim" ? Qt.SizeHorCursor
            : linha.sobreTipo === "remover" ? Qt.PointingHandCursor : Qt.ArrowCursor
        onPressed: mouse => linha.apertou(mouse.x, mouse.y)
        onPositionChanged: mouse => linha.moveu(mouse.x, mouse.y)
        onReleased: mouse => linha.soltou(mouse.x, mouse.y)
        onCanceled: linha.soltou(-1, -1)
        onDoubleClicked: mouse => linha.dividirEm(Corte.tempoDoX(linha.v, mouse.x))
        onExited: {
            if (linha.arrasto === null) {
                linha.sobre = -1;
                linha.sobreTipo = "";
            }
        }
    }
}
