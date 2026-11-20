import QtQuick

// Gráfico de linha mínimo: valores de 0 a 1 (ou normalizados pelo maior).
Canvas {
    id: spark

    property var values: []
    property color color: Projects.accent
    property bool normalize: false

    onValuesChanged: requestPaint()
    onColorChanged: requestPaint()

    onPaint: {
        const ctx = getContext("2d");
        ctx.reset();
        if (values.length < 2)
            return;
        const top = normalize ? Math.max(1, ...values) : 1;
        const step = width / (values.length - 1);
        const y = v => height - 1 - (Math.min(v, top) / top) * (height - 2);

        // Área suave embaixo da linha
        ctx.beginPath();
        ctx.moveTo(0, height);
        values.forEach((v, i) => ctx.lineTo(i * step, y(v)));
        ctx.lineTo(width, height);
        ctx.closePath();
        ctx.fillStyle = Qt.rgba(color.r, color.g, color.b, 0.12);
        ctx.fill();

        ctx.beginPath();
        values.forEach((v, i) => i === 0 ? ctx.moveTo(0, y(v)) : ctx.lineTo(i * step, y(v)));
        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;
        ctx.stroke();
    }
}
