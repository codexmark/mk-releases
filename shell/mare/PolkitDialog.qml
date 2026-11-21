import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Hyprland
import Quickshell.Wayland
import Quickshell.Services.Polkit

// O pedido de senha de administrador (polkit), feito pelo Maré: uma folha no
// meio da tela em foco, em português, com a mensagem inteira (quebra em
// linhas, nunca corta). Substitui o hyprpolkitagent, que só tem uma linha
// para a mensagem e os rótulos em inglês. Só um agente de polkit registra por
// sessão: com este no ar, o hyprpolkitagent não pode subir (ele continua
// instalado, de reserva: systemctl --user start hyprpolkitagent).
// Com a tela bloqueada a folha ficaria escondida atrás do bloqueio e o
// pedido, esperando para sempre: é cancelado na hora.
Scope {
    id: root

    PolkitAgent {
        id: agent
    }

    readonly property var flow: agent.flow
    readonly property bool hidden: agent.isActive && Lock.locked
    onHiddenChanged: {
        if (hidden && flow)
            flow.cancelAuthenticationRequest();
    }
    readonly property string who: {
        const i = flow ? flow.selectedIdentity : null;
        return i ? (i.displayName || i.string || "") : "";
    }

    PanelWindow {
        id: win

        visible: agent.isActive && !Lock.locked
        screen: {
            const m = Hyprland.focusedMonitor;
            const found = m ? Quickshell.screens.find(s => s.name === m.name) : null;
            return found || Quickshell.screens[0];
        }
        color: "transparent"
        // A tela inteira, também atrás da ilha (veja Mare.ignoreZone).
        exclusionMode: ExclusionMode.Normal
        exclusiveZone: Mare.ignoreZone
        WlrLayershell.layer: WlrLayer.Overlay
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.Exclusive
        WlrLayershell.namespace: "mare-polkit"

        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }

        property bool details: false

        function submit() {
            if (!root.flow || !root.flow.isResponseRequired)
                return;
            root.flow.submit(input.text);
            input.text = "";
        }
        function cancel() {
            if (root.flow)
                root.flow.cancelAuthenticationRequest();
        }

        onVisibleChanged: {
            input.text = "";
            details = false;
            if (visible)
                input.forceActiveFocus();
        }

        // O véu: clique fora cancela.
        Rectangle {
            anchors.fill: parent
            color: Qt.rgba(11 / 255, 11 / 255, 11 / 255, 0.72)

            TapHandler {
                onTapped: win.cancel()
            }
        }

        Rectangle {
            id: sheet
            anchors.centerIn: parent
            width: Math.min(460, parent.width - 48)
            height: body.implicitHeight + 56
            radius: 2
            color: Theme.surface
            border.width: 1
            border.color: Theme.line

            // Clique na folha não atravessa para o véu.
            TapHandler {}

            ColumnLayout {
                id: body
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.margins: 28
                spacing: 0

                Text {
                    text: "ADMINISTRADOR"
                    color: Theme.copper
                    font.family: Theme.mono
                    font.pixelSize: 11
                    font.letterSpacing: Theme.labelSpacing
                }

                Text {
                    Layout.topMargin: 10
                    Layout.fillWidth: true
                    text: "confirme com a sua senha"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 24
                    wrapMode: Text.WordWrap
                }

                // A mensagem do polkit, inteira.
                Text {
                    Layout.topMargin: 14
                    Layout.fillWidth: true
                    text: root.flow ? root.flow.message : ""
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 14
                    lineHeight: 1.3
                    wrapMode: Text.Wrap
                    textFormat: Text.PlainText
                }

                Rectangle {
                    Layout.topMargin: 22
                    Layout.fillWidth: true
                    height: 40
                    radius: 2
                    color: Theme.surfaceLow
                    border.width: 1
                    border.color: input.activeFocus ? Theme.copper : Theme.line

                    Text {
                        anchors.left: parent.left
                        anchors.leftMargin: 12
                        anchors.verticalCenter: parent.verticalCenter
                        visible: input.text === ""
                        text: root.who !== "" ? "senha de " + root.who : "senha"
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 12
                    }

                    TextInput {
                        id: input
                        anchors.fill: parent
                        anchors.leftMargin: 12
                        anchors.rightMargin: 12
                        verticalAlignment: TextInput.AlignVCenter
                        echoMode: root.flow && root.flow.responseVisible ? TextInput.Normal : TextInput.Password
                        passwordCharacter: "•"
                        color: Theme.text
                        selectionColor: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 14
                        clip: true
                        focus: true
                        Keys.onReturnPressed: win.submit()
                        Keys.onEnterPressed: win.submit()
                        Keys.onEscapePressed: win.cancel()
                    }
                }

                // Senha errada ou recado do sistema (leitor de digital etc.).
                Text {
                    Layout.topMargin: 10
                    Layout.fillWidth: true
                    readonly property string extra: root.flow ? root.flow.supplementaryMessage : ""
                    readonly property bool bad: root.flow ? (root.flow.failed || root.flow.supplementaryIsError) : false
                    visible: text !== ""
                    text: extra !== "" ? extra : (bad ? "senha incorreta, tente de novo" : "")
                    color: bad ? "#ff5f40" : Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                    wrapMode: Text.Wrap
                    textFormat: Text.PlainText
                }

                RowLayout {
                    Layout.topMargin: 22
                    Layout.fillWidth: true
                    spacing: 10

                    Text {
                        text: win.details ? "menos" : "detalhes"
                        color: detailsHover.hovered ? Theme.text : Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 11

                        HoverHandler {
                            id: detailsHover
                            cursorShape: Qt.PointingHandCursor
                        }
                        TapHandler {
                            onTapped: win.details = !win.details
                        }
                    }

                    Item {
                        Layout.fillWidth: true
                    }

                    PillButton {
                        text: "cancelar"
                        onClicked: win.cancel()
                    }
                    PillButton {
                        text: "autenticar"
                        primary: true
                        onClicked: win.submit()
                    }
                }

                Text {
                    Layout.topMargin: 16
                    Layout.fillWidth: true
                    visible: win.details
                    text: root.flow ? root.flow.actionId : ""
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 11
                    wrapMode: Text.WrapAnywhere
                    textFormat: Text.PlainText
                }
            }
        }
    }
}
