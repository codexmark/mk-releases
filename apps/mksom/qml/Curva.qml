import QtQuick
import "Som.js" as Som

// A curva resultante: a soma das bandas mais o pré-ganho, de 20 Hz a 20 kHz
// (as contas são do Som.js). O zero é a linha do meio; os risquinhos, as
// bandas.
Canvas {
    id: curva

    property var ganhos: []
    property real pre: 0
    property bool ligado: true

    onGanhosChanged: requestPaint()
    onPreChanged: requestPaint()
    onLigadoChanged: requestPaint()
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()

    onPaint: {
        const ctx = getContext("2d");
        ctx.clearRect(0, 0, width, height);
        if (width <= 0 || height <= 0)
            return;
        ctx.lineWidth = 1;
        ctx.strokeStyle = Theme.hairline;
        ctx.beginPath();
        ctx.moveTo(0, Math.round(height / 2) + 0.5);
        ctx.lineTo(width, Math.round(height / 2) + 0.5);
        for (let i = 0; i < Som.BANDAS.length; i++) {
            const x = Math.round(Som.xDaFrequencia(Som.BANDAS[i], width)) + 0.5;
            ctx.moveTo(x, height / 2 - 3);
            ctx.lineTo(x, height / 2 + 3);
        }
        ctx.stroke();

        const pontos = Som.pontos(ganhos, pre, width, height, Math.max(24, Math.round(width / 4)));
        ctx.lineWidth = 1.5;
        ctx.strokeStyle = ligado ? Theme.copper : Theme.textDim;
        ctx.beginPath();
        for (let i = 0; i < pontos.length; i++) {
            if (i === 0)
                ctx.moveTo(pontos[i].x, pontos[i].y);
            else
                ctx.lineTo(pontos[i].x, pontos[i].y);
        }
        ctx.stroke();
    }
}
