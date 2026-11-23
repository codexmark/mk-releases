import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Downloads: o que está baixando e o que já está na pasta de música.
ListView {
    id: page
    property bool canGoBack: false
    signal back()

    readonly property var jobs: daemon.downloads.jobs
    readonly property var active: jobs.filter(j => j.status === "queued" || j.status === "downloading")
    readonly property var finished: jobs.filter(j => j.status !== "queued" && j.status !== "downloading")
    readonly property var playable: finished.filter(j => j.status === "completed").map(j => j.track)
    readonly property bool hasHistory: finished.some(j => j.status === "failed" || j.status === "cancelled")
    readonly property string folder: daemon.downloads.folder || ""

    clip: true
    spacing: 2
    cacheBuffer: 1200
    boundsBehavior: Flickable.StopAtBounds
    leftMargin: 32
    rightMargin: 32
    topMargin: 28
    bottomMargin: 32
    ScrollBar.vertical: ScrollBar {}
    model: finished
    delegate: JobRow {
        required property var modelData
        width: ListView.view.width - page.leftMargin - page.rightMargin
        job: modelData
    }

    header: ColumnLayout {
        width: page.width - page.leftMargin - page.rightMargin
        spacing: 12
        PageHeader { label: "05 · Offline"; title: "Downloads" }
        // Onde as músicas ficam, com atalho para a pasta.
        RowLayout {
            Layout.fillWidth: true
            visible: page.folder !== ""
            spacing: 10
            Text { text: Theme.icon.download; color: Theme.textFaint; font.family: Theme.symbols; font.pixelSize: 16 }
            Text {
                Layout.fillWidth: true
                text: page.folder.replace(/^\/home\/[^/]+/, "~")
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
                elide: Text.ElideMiddle
            }
            MkButton { text: "Abrir pasta"; onClicked: Qt.openUrlExternally("file://" + page.folder) }
        }
        Caption { Layout.topMargin: 12; text: "Baixando agora" }
        Hint { visible: page.active.length === 0; text: "Nada baixando agora. Baixe pelo menu ⋯ de uma faixa ou pela página de um álbum." }
        Repeater {
            model: page.active
            JobRow { required property var modelData; Layout.fillWidth: true; job: modelData }
        }
        RowLayout {
            Layout.topMargin: 16
            Layout.fillWidth: true
            Caption { text: "No computador" }
            Item { Layout.fillWidth: true }
            MkButton {
                visible: page.hasHistory
                text: "Limpar falhas"
                glyph: Theme.icon.close
                onClicked: daemon.send({ cmd: "clearDownloadHistory" })
            }
            PlayButtons { tracks: page.playable; visible: page.playable.length > 0 }
        }
        Hint { visible: page.finished.length === 0; text: "As músicas baixadas tocam sem internet e ficam na sua pasta de Música, em MkTunes." }
    }

    component JobRow: Rectangle {
        id: row
        property var job
        readonly property bool done: job.status === "completed"
        height: 64
        radius: 8
        color: jh.hovered ? Theme.wash : "transparent"
        HoverHandler { id: jh; cursorShape: row.done ? Qt.PointingHandCursor : Qt.ArrowCursor }
        TapHandler { onTapped: if (row.done) daemon.send({ cmd: "play", tracks: page.playable, index: page.playable.findIndex(t => t.id === row.job.track.id) }) }
        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: 8
            anchors.rightMargin: 4
            spacing: 12
            Cover { Layout.preferredWidth: 44; Layout.preferredHeight: 44; source: row.job.track.artwork }
            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Text { Layout.fillWidth: true; text: row.job.track.title; color: Theme.text; font.family: Theme.sans; font.pixelSize: 14; font.weight: Font.Medium; elide: Text.ElideRight }
                ProgressBar {
                    Layout.fillWidth: true
                    visible: row.job.status === "downloading"
                    value: (row.job.percent || 0) / 100
                    background: Rectangle { implicitHeight: 3; radius: 2; color: Theme.line }
                    contentItem: Item {
                        implicitHeight: 3
                        Rectangle { width: parent.width * ((row.job.percent || 0) / 100); height: 3; radius: 2; color: Theme.copper }
                    }
                }
                Text {
                    Layout.fillWidth: true
                    color: row.job.status === "failed" ? Theme.copper : Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                    elide: Text.ElideRight
                    text: {
                        switch (row.job.status) {
                        case "queued": return row.job.track.artist + " · na fila";
                        case "downloading": return row.job.track.artist + " · " + (row.job.percent || 0) + "%";
                        case "completed": return row.job.track.artist + " · no computador";
                        case "failed": return "Falhou: " + row.job.reason;
                        default: return "Cancelado";
                        }
                    }
                }
            }
            MkButton {
                text: row.job.status === "queued" || row.job.status === "downloading" ? "Cancelar"
                    : row.done ? "Remover" : "Tentar de novo"
                onClicked: {
                    const id = row.job.track.id;
                    if (row.job.status === "queued" || row.job.status === "downloading") daemon.send({ cmd: "cancelDownload", id: id });
                    else if (row.done) app.confirm("Remover \"" + row.job.track.title + "\" do computador?", "O arquivo é apagado. A música continua no catálogo e pode ser baixada de novo.", "Remover", () => daemon.send({ cmd: "removeDownload", id: id }));
                    else daemon.send({ cmd: "retryDownload", id: id });
                }
            }
        }
    }
}
