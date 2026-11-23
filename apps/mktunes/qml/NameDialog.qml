import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

// Pede um nome (nova playlist, renomear). Enter confirma, Esc cancela.
Popup {
    id: dialog
    property string title
    property string confirmText: "Criar"
    property string initial: ""
    signal confirmed(string name)

    function ask(heading, button, start) {
        dialog.title = heading;
        dialog.confirmText = button;
        dialog.initial = start || "";
        field.text = dialog.initial;
        dialog.open();
        field.forceActiveFocus();
        field.selectAll();
    }

    modal: true
    anchors.centerIn: Overlay.overlay
    width: 380
    padding: 20
    background: Rectangle { radius: 12; color: Theme.raised; border.color: Theme.line }
    Overlay.modal: Rectangle { color: Qt.rgba(0, 0, 0, 0.5) }

    ColumnLayout {
        anchors.fill: parent
        spacing: 16
        Text { text: dialog.title; color: Theme.text; font.family: Theme.serif; font.pixelSize: 20 }
        MkField {
            id: field
            Layout.fillWidth: true
            placeholderText: "Nome"
            onAccepted: confirm.clicked()
        }
        RowLayout {
            Layout.alignment: Qt.AlignRight
            spacing: 8
            MkButton { text: "Cancelar"; onClicked: dialog.close() }
            MkButton {
                id: confirm
                text: dialog.confirmText
                primary: true
                enabled: field.text.trim() !== ""
                onClicked: {
                    if (field.text.trim() === "") return;
                    dialog.confirmed(field.text.trim());
                    dialog.close();
                }
            }
        }
    }
}
