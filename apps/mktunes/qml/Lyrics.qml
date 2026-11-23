import QtQuick
import QtQuick.Controls

// Letra: a linha que está tocando em cobre e centralizada; clicar numa linha pula para ela.
Item {
    id: root
    readonly property var lyrics: daemon.lyrics
    readonly property var lines: lyrics.status === "synced" ? lyrics.lines : []
    // Letra adiantada ou atrasada em relação ao áudio: o ouvinte acerta em passos de meio segundo.
    property int offsetMs: 0
    property int fontSize: 16
    readonly property real positionMs: daemon.progress.position * 1000 + offsetMs

    // Busca binária: a última linha que já começou.
    readonly property int currentLine: {
        let lo = 0, hi = lines.length - 1, found = -1;
        while (lo <= hi) {
            const mid = (lo + hi) >> 1;
            if (lines[mid].ms <= positionMs + 250) { found = mid; lo = mid + 1; } else hi = mid - 1;
        }
        return found;
    }
    onCurrentLineChanged: if (currentLine >= 0 && !synced.moving) synced.positionViewAtIndex(currentLine, ListView.Center)

    ListView {
        id: synced
        anchors.fill: parent
        visible: root.lyrics.status === "synced"
        clip: true
        spacing: root.fontSize * 0.6
        model: root.lines
        highlightRangeMode: ListView.NoHighlightRange
        delegate: Text {
            required property var modelData
            required property int index
            width: ListView.view.width
            text: modelData.text || "♪"
            wrapMode: Text.WordWrap
            color: index === root.currentLine ? Theme.copper : (index < root.currentLine ? Theme.textFaint : Theme.textSoft)
            font.family: Theme.sans
            font.pixelSize: root.fontSize
            font.weight: index === root.currentLine ? Font.DemiBold : Font.Normal
            Behavior on color { ColorAnimation { duration: 180 } }
            TapHandler { onTapped: daemon.send({ cmd: "seek", seconds: Math.max(0, modelData.ms - root.offsetMs) / 1000 }) }
            HoverHandler { cursorShape: Qt.PointingHandCursor }
        }
    }

    Flickable {
        anchors.fill: parent
        visible: root.lyrics.status === "plain"
        clip: true
        contentHeight: plain.implicitHeight
        ScrollBar.vertical: ScrollBar {}
        Text {
            id: plain
            width: parent.width
            text: root.lyrics.text || ""
            wrapMode: Text.WordWrap
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: root.fontSize - 1
            lineHeight: 1.3
        }
    }

    Hint {
        anchors.fill: parent
        visible: root.lyrics.status !== "synced" && root.lyrics.status !== "plain"
        text: root.lyrics.status === "loading" ? "Procurando a letra…"
            : root.lyrics.status === "unavailable" ? "O serviço de letras não respondeu."
            : daemon.now.status === "idle" ? "" : "Sem letra para esta música."
    }
}
