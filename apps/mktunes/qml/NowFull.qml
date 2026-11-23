import QtQuick
import QtQuick.Effects
import QtQuick.Layouts

// "Tocando agora" em tela cheia: capa grande à esquerda e a letra grande à direita, com o
// acerto de sincronia. A barra do player continua embaixo. Esc ou F11 fecha.
Rectangle {
    id: full
    readonly property var now: daemon.now
    signal close()
    color: Theme.ground

    // Fundo: a própria capa, desfocada e escura. Uma passada só, refeita quando a capa muda.
    Image {
        id: backdrop
        anchors.fill: parent
        source: full.now.artwork
        sourceSize: Qt.size(96, 96)
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        visible: false
    }
    MultiEffect {
        anchors.fill: parent
        source: backdrop
        visible: backdrop.status === Image.Ready
        blurEnabled: true
        blur: 1
        blurMax: 48
        saturation: 0.2
        brightness: -0.55
        opacity: 0.9
    }

    // Engole os cliques: nada atrás da tela cheia reage.
    MouseArea { anchors.fill: parent; acceptedButtons: Qt.AllButtons; onWheel: wheel => wheel.accepted = true }

    Glyph {
        anchors.top: parent.top
        anchors.right: parent.right
        anchors.margins: 20
        glyph: Theme.icon.fullscreenExit
        label: "Sair da tela cheia (Esc)"
        onClicked: full.close()
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 56
        spacing: 56

        ColumnLayout {
            Layout.preferredWidth: Math.min(440, full.width * 0.38)
            Layout.maximumWidth: 440
            Layout.alignment: Qt.AlignVCenter
            spacing: 14
            Cover {
                Layout.fillWidth: true
                Layout.preferredHeight: width
                source: full.now.artwork
            }
            Text {
                Layout.fillWidth: true
                Layout.topMargin: 8
                text: full.now.title
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 34
                wrapMode: Text.WordWrap
                maximumLineCount: 2
                elide: Text.ElideRight
            }
            Text {
                Layout.fillWidth: true
                text: full.now.artist
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 16
                elide: Text.ElideRight
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 16
            RowLayout {
                spacing: 6
                Caption { text: "Letra"; accent: true }
                Item { Layout.fillWidth: true }
                // Acerto de sincronia: só faz sentido com letra sincronizada.
                RowLayout {
                    visible: daemon.lyrics.status === "synced"
                    spacing: 4
                    Glyph { glyph: Theme.icon.minus; size: 30; label: "Letra mais cedo"; onClicked: app.lyricsOffset -= 500 }
                    Text {
                        Layout.preferredWidth: 120
                        horizontalAlignment: Text.AlignHCenter
                        text: app.lyricsOffset === 0 ? "Sincronia" : "Sincronia " + (app.lyricsOffset > 0 ? "+" : "−") + (Math.abs(app.lyricsOffset) / 1000).toFixed(1).replace(".", ",") + " s"
                        color: app.lyricsOffset === 0 ? Theme.textDim : Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 12
                        TapHandler { onTapped: app.lyricsOffset = 0 }
                    }
                    Glyph { glyph: Theme.icon.plus; size: 30; label: "Letra mais tarde"; onClicked: app.lyricsOffset += 500 }
                }
                Item { Layout.preferredWidth: 48 }
            }
            Lyrics {
                Layout.fillWidth: true
                Layout.fillHeight: true
                fontSize: 28
                offsetMs: app.lyricsOffset
            }
        }
    }
}
