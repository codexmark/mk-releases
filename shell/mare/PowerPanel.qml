import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Services.UPower

// Bateria e perfil de energia (power-profiles-daemon). Além da carga,
// mostra a saúde da bateria, os ciclos e o limite de carga (scripts/mare-battery),
// com frases que alternam como na barra do Omarchy.
Item {
    id: panel

    readonly property var dev: UPower.displayDevice
    readonly property int pct: dev ? percent(dev.percentage) : 0
    readonly property bool charging: dev && dev.state === UPowerDeviceState.Charging
    readonly property bool full: dev && dev.state === UPowerDeviceState.FullyCharged

    // {cycles, health (0..1), limit (0..100)}; qualquer um pode vir null.
    property var info: ({ cycles: null, health: null, limit: null })

    readonly property var chargingPhrases: ["Enchendo o tanque", "Bebendo da tomada", "Recarregando o fôlego",
        "Juntando elétrons", "Tomando um café"]
    readonly property var drainingPhrases: ["Bebericando joules", "Queimando a reserva", "Vivendo de bateria",
        "Gastando com moderação", "Longe da tomada"]
    readonly property var phrases: charging ? chargingPhrases : drainingPhrases
    property int phraseIndex: 0

    implicitWidth: 420
    implicitHeight: column.implicitHeight + 44

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no UPower nem no painel: são contas e
    // textos, testados fora do shell com node (tests/js/PowerPanel.test.mjs).
    // O UPower dá a carga de 0 a 100; se um dia vier de 0 a 1, não zera.
    function percent(p) {
        return Math.round(p > 1 ? p : p * 100);
    }

    function duration(seconds) {
        if (!seconds || seconds <= 0)
            return "";
        // Arredonda o total em minutos antes de separar as horas: 3599 s é
        // "1 h 0 min", não "60 min".
        const total = Math.round(seconds / 60);
        const h = Math.floor(total / 60);
        const m = total % 60;
        return h > 0 ? `${h} h ${m} min` : `${m} min`;
    }

    function wh(value) {
        return value > 0 ? value.toFixed(1).replace(".", ",") + " Wh" : "";
    }

    // A potência de agora: "+18,4 W" carregando, "−7,2 W" gastando.
    function watts(rate, charging) {
        return (charging ? "+" : "−") + rate.toFixed(1).replace(".", ",") + " W";
    }
    // --- puro: fim ---

    Component.onCompleted: batteryProc.running = true

    onChargingChanged: phraseIndex = 0

    Timer {
        interval: 2800
        repeat: true
        running: !panel.full
        onTriggered: panel.phraseIndex = (panel.phraseIndex + 1) % panel.phrases.length
    }

    Process {
        id: batteryProc
        command: ["bash", Quickshell.shellPath("scripts/mare-battery")]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    panel.info = JSON.parse(text);
                } catch (e) {}
            }
        }
    }

    component InfoPair: ColumnLayout {
        property string label: ""
        property string value: ""
        property color valueColor: Theme.text

        spacing: 3
        Layout.fillWidth: true

        Text {
            text: parent.label
            color: Theme.textFaint
            font.family: Theme.mono
            font.pixelSize: 9
            font.capitalization: Font.AllUppercase
            font.letterSpacing: 1.6
        }

        Text {
            text: parent.value !== "" ? parent.value : "—"
            color: parent.valueColor
            font.family: Theme.mono
            font.pixelSize: 12
        }
    }

    ColumnLayout {
        id: column
        x: 22
        y: 22
        width: parent.width - 44
        spacing: 12

        PanelHeader {
            Layout.fillWidth: true
            index: "01"
            title: "Bateria"
            meta: panel.full ? "cheia" : panel.charging ? "carregando" : "na bateria"
            metaColor: panel.charging ? Theme.copper : Theme.textDim
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 14

            Text {
                text: panel.pct + "%"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 40
                font.weight: Font.Light
            }

            ColumnLayout {
                spacing: 2
                Layout.alignment: Qt.AlignVCenter

                Text {
                    text: panel.full ? "Pronta para sair" : panel.phrases[panel.phraseIndex % panel.phrases.length] + "…"
                    color: Theme.textSoft
                    font.family: Theme.serif
                    font.pixelSize: 15
                    font.italic: true
                    font.weight: Font.Light
                }

                Text {
                    text: panel.charging
                        ? (panel.duration(panel.dev.timeToFull) !== "" ? "cheia em " + panel.duration(panel.dev.timeToFull) : "")
                        : (panel.duration(panel.dev ? panel.dev.timeToEmpty : 0) !== "" ? panel.duration(panel.dev.timeToEmpty) + " restantes" : "")
                    visible: text !== ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
            }

            Item { Layout.fillWidth: true }

            Text {
                visible: panel.dev && panel.dev.changeRate > 0
                text: panel.dev ? panel.watts(panel.dev.changeRate, panel.charging) : ""
                color: panel.charging ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 12
            }
        }

        // Carga, com o limite marcado quando existe.
        Item {
            Layout.fillWidth: true
            implicitHeight: 6

            Rectangle {
                anchors.fill: parent
                color: Theme.line
            }

            Rectangle {
                width: parent.width * panel.pct / 100
                height: parent.height
                color: panel.charging ? Theme.copper : Theme.text
            }

            Rectangle {
                visible: panel.info.limit !== null && panel.info.limit < 100
                x: parent.width * (panel.info.limit ?? 100) / 100 - 1
                y: -3
                width: 2
                height: parent.height + 6
                color: Theme.copper
            }
        }

        GridLayout {
            Layout.fillWidth: true
            columns: 4
            columnSpacing: 14

            InfoPair {
                label: "saúde"
                value: panel.info.health !== null ? Math.round(panel.info.health * 100) + "%" : ""
                valueColor: panel.info.health !== null && panel.info.health < 0.6 ? Theme.copper : Theme.text
            }
            InfoPair {
                label: "ciclos"
                value: panel.info.cycles !== null ? String(panel.info.cycles) : ""
            }
            InfoPair {
                label: "capacidade"
                value: panel.dev ? panel.wh(panel.dev.energyCapacity) : ""
            }
            InfoPair {
                label: "limite"
                value: panel.info.limit !== null ? panel.info.limit + "%" : "sem"
            }
        }

        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 1
            color: Theme.hairline
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 6

            Repeater {
                model: [
                    { label: "economia", profile: PowerProfile.PowerSaver },
                    { label: "balanceado", profile: PowerProfile.Balanced },
                    { label: "desempenho", profile: PowerProfile.Performance }
                ].filter(p => p.profile !== PowerProfile.Performance || PowerProfiles.hasPerformanceProfile)

                PillButton {
                    required property var modelData
                    Layout.fillWidth: true
                    text: modelData.label
                    selected: PowerProfiles.profile === modelData.profile
                    onClicked: PowerProfiles.profile = modelData.profile
                }
            }
        }
    }
}
