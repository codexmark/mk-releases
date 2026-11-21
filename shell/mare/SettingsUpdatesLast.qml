import QtQuick
import QtQuick.Layouts

// Atualizações, o depois do clique: como foi a última rodada do mk-update,
// o registro dela, o aviso de reinício e o que chega com o Mk novo.
ColumnLayout {
    id: root

    visible: Updates.last !== null || Updates.reboot || Updates.news.length > 0
    Layout.fillWidth: true
    spacing: 6

    RowLayout {
        visible: Updates.last !== null
        Layout.fillWidth: true
        spacing: 12

        Text {
            Layout.fillWidth: true
            text: Updates.lastText()
            textFormat: Text.PlainText
            color: Updates.last && !Updates.last.ok ? Theme.copper : Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 11
            elide: Text.ElideRight
        }
        PillButton {
            visible: Updates.last !== null && Updates.last.log !== ""
            text: "ver registro"
            onClicked: Updates.openLog()
        }
    }

    Text {
        visible: Updates.reboot
        text: "reinicie para o kernel novo"
        color: Theme.copper
        font.family: Theme.mono
        font.pixelSize: 11
    }

    Text {
        visible: Updates.news.length > 0
        Layout.topMargin: 6
        text: "o que chega"
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.capitalization: Font.AllUppercase
        font.letterSpacing: Theme.labelSpacing
    }
    Repeater {
        model: Updates.news

        Text {
            required property string modelData
            Layout.fillWidth: true
            text: modelData
            textFormat: Text.PlainText
            wrapMode: Text.WordWrap
            color: Theme.textSoft
            font.family: Theme.mono
            font.pixelSize: 11
        }
    }
}
