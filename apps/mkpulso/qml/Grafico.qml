import QtQuick

// Um gráfico de linha do último minuto: a série anda da direita para a
// esquerda, com a área embaixo quase transparente. `teto` 0 = a escala
// acompanha o maior valor da série.
Canvas {
    id: grafico

    property var serie: []
    property var serie2: []
    property real teto: 100
    property int pontos: 90
    property color cor: Theme.copper
    property color cor2: Theme.textDim

    onSerieChanged: requestPaint()
    onSerie2Changed: requestPaint()
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()

    onPaint: {
        const ctx = getContext("2d");
        ctx.reset();
        const w = width, h = height;
        // Grade: três linhas de nível.
        ctx.strokeStyle = Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.08);
        ctx.lineWidth = 1;
        for (let i = 1; i <= 3; i++) {
            const y = Math.round(h * i / 4) + 0.5;
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(w, y);
            ctx.stroke();
        }
        let max = grafico.teto;
        if (!(max > 0)) {
            max = 1;
            for (const v of (grafico.serie || []).concat(grafico.serie2 || []))
                if (v > max)
                    max = v;
            max *= 1.15;
        }
        const traco = (serie, cor, area) => {
            if (!serie || serie.length < 2)
                return;
            const passo = w / (grafico.pontos - 1);
            const x0 = w - (serie.length - 1) * passo;
            ctx.beginPath();
            for (let i = 0; i < serie.length; i++) {
                const x = x0 + i * passo;
                const y = h - Math.max(0, Math.min(1, serie[i] / max)) * (h - 2) - 1;
                if (i === 0)
                    ctx.moveTo(x, y);
                else
                    ctx.lineTo(x, y);
            }
            ctx.strokeStyle = cor;
            ctx.lineWidth = 1.5;
            ctx.stroke();
            if (area) {
                ctx.lineTo(w, h);
                ctx.lineTo(x0, h);
                ctx.closePath();
                ctx.fillStyle = Qt.rgba(206 / 255, 94 / 255, 50 / 255, 0.1);
                ctx.fill();
            }
        };
        traco(grafico.serie2, grafico.cor2, false);
        traco(grafico.serie, grafico.cor, true);
    }
}
