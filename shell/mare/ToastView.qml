import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Services.Notifications

// Aviso passageiro: a pílula cresce com a notificação e volta sozinha.
// Clique abre a ação do app; o × descarta.
Item {
    id: toast

    readonly property var n: Notifs.toastItem

    implicitWidth: 440
    implicitHeight: Math.max(76, row.implicitHeight + 32)

    RowLayout {
        id: row
        x: 18
        y: 16
        width: parent.width - 36
        spacing: 14

        NotifIcon {
            notification: toast.n
            Layout.alignment: Qt.AlignTop
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 3

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: toast.n ? (toast.n.appName || "Notificação") : ""
                    color: toast.n && toast.n.urgency === NotificationUrgency.Critical ? "#FF8F66" : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                }

                Text {
                    text: "×"
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 16

                    HoverHandler { cursorShape: Qt.PointingHandCursor }
                    TapHandler {
                        onTapped: {
                            if (toast.n)
                                toast.n.dismiss();
                            Mare.close();
                        }
                    }
                }
            }

            Text {
                text: toast.n ? toast.n.summary : ""
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 14
                font.weight: Font.Medium
                Layout.fillWidth: true
                elide: Text.ElideRight
            }

            Text {
                visible: text !== ""
                text: toast.n ? toast.n.body : ""
                textFormat: Text.PlainText
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 13
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                maximumLineCount: 2
                elide: Text.ElideRight
            }
        }
    }

    TapHandler {
        onTapped: {
            if (toast.n)
                Notifs.activate(toast.n);
            Mare.close();
        }
    }
}
