import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Confirma algo que não se desfaz (apagar playlist, remover download).
Popup {
    id: dialog
    property string title
    property string detail
    property string button
    property var action: null
    function ask(t, d, b, a) { title = t; detail = d; button = b; action = a; open(); }
    modal: true
    anchors.centerIn: Overlay.overlay
    width: 420
    padding: 20
    background: Rectangle { radius: 12; color: Theme.raised; border.color: Theme.line }
    Overlay.modal: Rectangle { color: Qt.rgba(0, 0, 0, 0.5) }
    ColumnLayout {
        anchors.fill: parent
        spacing: 12
        Text { Layout.fillWidth: true; text: dialog.title; color: Theme.text; font.family: Theme.serif; font.pixelSize: 20; wrapMode: Text.WordWrap }
        Hint { Layout.fillWidth: true; text: dialog.detail }
        RowLayout {
            Layout.alignment: Qt.AlignRight
            Layout.topMargin: 8
            spacing: 8
            MkButton { text: "Cancelar"; onClicked: dialog.close() }
            MkButton { text: dialog.button; primary: true; onClicked: { if (dialog.action) dialog.action(); dialog.close(); } }
        }
    }
}
