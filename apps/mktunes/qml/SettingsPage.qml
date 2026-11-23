import QtQuick
import QtQuick.Controls
import QtQuick.Dialogs
import QtQuick.Layouts

// Ajustes: reprodução, downloads e o backup da biblioteca.
Flickable {
    id: page
    property bool canGoBack: false
    signal back()
    readonly property var s: daemon.settings

    clip: true
    contentHeight: column.implicitHeight + 64
    boundsBehavior: Flickable.StopAtBounds
    ScrollBar.vertical: ScrollBar {}

    function set(key, value) { daemon.send({ cmd: "setting", key: key, value: value }); }

    ColumnLayout {
        id: column
        x: 32
        y: 28
        width: Math.min(page.width - 64, 720)
        spacing: 8

        PageHeader { label: "06 · Ajustes"; title: "Ajustes" }

        Caption { Layout.topMargin: 20; text: "Reprodução" }
        Toggle {
            title: "Continuar tocando músicas parecidas"
            detail: "Quando a fila acaba, o MkTunes emenda músicas no clima da última."
            checked: page.s.autoplay
            onToggled: value => page.set("autoplay", value)
        }
        Toggle {
            title: "Igualar o volume entre músicas"
            detail: "Abaixa as faixas gravadas mais altas, quando o catálogo informa."
            checked: page.s.normalizeVolume
            onToggled: value => page.set("normalizeVolume", value)
        }
        SettingRow {
            title: "Qualidade do áudio"
            detail: page.s.audioQuality === "high" ? "A melhor que o catálogo oferece." : "Até 128 kbps, para gastar menos internet."
            RowLayout {
                spacing: 8
                Chip { text: "Alta"; selected: page.s.audioQuality === "high"; onClicked: page.set("audioQuality", "high") }
                Chip { text: "Econômica"; selected: page.s.audioQuality === "economy"; onClicked: page.set("audioQuality", "economy") }
            }
        }

        Caption { Layout.topMargin: 20; text: "Downloads" }
        SettingRow {
            title: "Downloads ao mesmo tempo"
            detail: "Mais downloads juntos terminam antes, mas dividem a internet."
            RowLayout {
                spacing: 4
                Glyph { glyph: "−"; label: "Menos"; enabled: page.s.downloadWorkers > 1; onClicked: page.set("downloadWorkers", page.s.downloadWorkers - 1) }
                Text { text: page.s.downloadWorkers; color: Theme.text; font.family: Theme.mono; font.pixelSize: 15; Layout.preferredWidth: 20; horizontalAlignment: Text.AlignHCenter }
                Glyph { glyph: "+"; label: "Mais"; enabled: page.s.downloadWorkers < page.s.maxWorkers; onClicked: page.set("downloadWorkers", page.s.downloadWorkers + 1) }
            }
        }

        Caption { Layout.topMargin: 20; text: "Backup" }
        SettingRow {
            title: "Salvar a biblioteca num arquivo"
            detail: "Curtidas, playlists, álbuns salvos, histórico e ajustes, em " + page.s.backupsDir + "."
            MkButton { text: "Salvar backup"; onClicked: daemon.send({ cmd: "exportBackup" }) }
        }
        SettingRow {
            title: "Restaurar de um arquivo"
            detail: "Junta o backup ao que já está aqui; nada é apagado."
            MkButton { text: "Escolher arquivo…"; onClicked: picker.open() }
        }

        Caption { Layout.topMargin: 20; text: "Sobre" }
        Hint { text: "MkTunes " + page.s.version + " · GPL-3.0 · música sem conta e sem anúncio." }
    }

    FileDialog {
        id: picker
        title: "Restaurar backup do MkTunes"
        currentFolder: "file://" + page.s.backupsDir
        nameFilters: ["Backup do MkTunes (*.json)", "Todos os arquivos (*)"]
        onAccepted: daemon.send({ cmd: "importBackup", path: decodeURIComponent(selectedFile.toString().replace(/^file:\/\//, "")) })
    }

    component SettingRow: RowLayout {
        id: settingRow
        property string title
        property string detail
        default property alias control: holder.data
        Layout.fillWidth: true
        Layout.topMargin: 6
        spacing: 20
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2
            Text { Layout.fillWidth: true; text: settingRow.title; color: Theme.text; font.family: Theme.sans; font.pixelSize: 14; font.weight: Font.Medium; wrapMode: Text.WordWrap }
            Text { Layout.fillWidth: true; text: settingRow.detail; color: Theme.textDim; font.family: Theme.sans; font.pixelSize: 12; wrapMode: Text.WordWrap }
        }
        Item { id: holder; implicitWidth: childrenRect.width; implicitHeight: childrenRect.height }
    }

    component Toggle: SettingRow {
        id: toggle
        property bool checked
        signal toggled(bool value)
        Rectangle {
            width: 44
            height: 24
            radius: 12
            color: toggle.checked ? Theme.copper : Theme.raised
            border.color: toggle.checked ? Theme.copper : Theme.line
            Accessible.role: Accessible.CheckBox
            Accessible.name: toggle.title
            Accessible.checked: toggle.checked
            Rectangle {
                width: 18; height: 18; radius: 9
                y: 3
                x: toggle.checked ? 23 : 3
                color: toggle.checked ? Theme.ground : Theme.textDim
                Behavior on x { NumberAnimation { duration: 120 } }
            }
            HoverHandler { cursorShape: Qt.PointingHandCursor }
            TapHandler { onTapped: toggle.toggled(!toggle.checked) }
        }
    }
}
