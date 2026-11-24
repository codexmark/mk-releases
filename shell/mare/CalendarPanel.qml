import QtQuick
import QtQuick.Layouts
import Quickshell

// Mês com a semana começando no domingo; o dia de hoje na cor do projeto.
Item {
    id: panel

    property int year: clock.date.getFullYear()
    property int month: clock.date.getMonth()

    readonly property var cells: monthCells(year, month, clock.date)

    implicitWidth: 600
    implicitHeight: 320

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no relógio nem no painel: são contas de
    // data, testadas fora do shell com node (tests/js/CalendarPanel.test.mjs).
    // `y` é o ano, `m` o mês de 0 a 11 e `today` um Date.
    readonly property var months: ["janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho",
        "agosto", "setembro", "outubro", "novembro", "dezembro"]
    readonly property var weekdays: ["domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"]

    // 42 células (6 semanas): dias do mês anterior e do próximo ficam apagados.
    function monthCells(y, m, today) {
        const first = new Date(y, m, 1).getDay();
        const days = new Date(y, m + 1, 0).getDate();
        const prevDays = new Date(y, m, 0).getDate();
        const out = [];
        for (let i = 0; i < 42; i++) {
            const d = i - first + 1;
            const inMonth = d >= 1 && d <= days;
            out.push({
                day: inMonth ? d : d < 1 ? prevDays + d : d - days,
                inMonth: inMonth,
                today: inMonth && d === today.getDate() && m === today.getMonth() && y === today.getFullYear(),
                weekend: i % 7 === 0 || i % 7 === 6
            });
        }
        return out;
    }

    // O mês que fica `delta` meses adiante (ou atrás): { year, month }.
    function shifted(y, m, delta) {
        const d = new Date(y, m + delta, 1);
        return { year: d.getFullYear(), month: d.getMonth() };
    }

    // "segunda, 23 de novembro".
    function dayLine(date) {
        return weekdays[date.getDay()] + ", " + date.getDate() + " de " + months[date.getMonth()];
    }
    // --- puro: fim ---

    function shift(delta) {
        const next = shifted(year, month, delta);
        year = next.year;
        month = next.month;
    }

    SystemClock {
        id: clock
        precision: SystemClock.Minutes
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 22
        spacing: 26

        ColumnLayout {
            Layout.preferredWidth: 252
            Layout.alignment: Qt.AlignTop
            spacing: 10

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: panel.months[panel.month] + " " + panel.year
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 20
                    font.weight: Font.Light
                    Layout.fillWidth: true
                }

                Repeater {
                    model: [{ label: "‹", delta: -1 }, { label: "›", delta: 1 }]

                    Text {
                        required property var modelData
                        text: modelData.label
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 18
                        leftPadding: 6
                        rightPadding: 6

                        HoverHandler { cursorShape: Qt.PointingHandCursor }
                        TapHandler { onTapped: panel.shift(parent.modelData.delta) }
                    }
                }
            }

            GridLayout {
                columns: 7
                rowSpacing: 2
                columnSpacing: 2

                Repeater {
                    model: ["D", "S", "T", "Q", "Q", "S", "S"]

                    Text {
                        required property string modelData
                        Layout.preferredWidth: 34
                        horizontalAlignment: Text.AlignHCenter
                        text: modelData
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                }

                Repeater {
                    model: panel.cells

                    Rectangle {
                        required property var modelData
                        Layout.preferredWidth: 34
                        Layout.preferredHeight: 30
                        radius: 2
                        color: modelData.today ? Projects.accent : "transparent"

                        Text {
                            anchors.centerIn: parent
                            text: parent.modelData.day
                            color: parent.modelData.today ? Theme.ground
                                : !parent.modelData.inMonth ? "#3A4048"
                                : parent.modelData.weekend ? Theme.textDim : Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                    }
                }
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignTop
            spacing: 6

            Text {
                text: Qt.formatDateTime(clock.date, "hh:mm")
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 44
                font.weight: Font.DemiBold
            }

            Text {
                text: panel.dayLine(clock.date)
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 14
            }

            // Clima: agora, mínima e máxima, e as próximas horas.
            ColumnLayout {
                visible: Weather.ready
                Layout.topMargin: 18
                spacing: 10

                RowLayout {
                    spacing: 12

                    Text {
                        text: Weather.ready ? Weather.icon(Weather.data.code, Weather.data.day) : ""
                        color: Projects.accent
                        font.family: Theme.mono
                        font.pixelSize: 30
                    }

                    ColumnLayout {
                        spacing: 0

                        Text {
                            text: Weather.ready ? Weather.data.temp + "°  " + Weather.describe(Weather.data.code) : ""
                            color: Theme.text
                            font.family: Theme.sans
                            font.pixelSize: 15
                            font.weight: Font.Medium
                        }

                        Text {
                            text: Weather.ready ? Weather.data.city + " · " + Weather.data.min + "° / " + Weather.data.max + "° · sensação " + Weather.data.feels + "°" : ""
                            color: Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 11
                        }
                    }
                }

                Row {
                    spacing: 12

                    Repeater {
                        model: Weather.ready ? Weather.data.hours.slice(0, 5) : []

                        Column {
                            required property var modelData
                            spacing: 3

                            Text {
                                anchors.horizontalCenter: parent.horizontalCenter
                                text: parent.modelData.time
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }

                            Text {
                                anchors.horizontalCenter: parent.horizontalCenter
                                text: Weather.icon(parent.modelData.code, true)
                                color: Theme.textSoft
                                font.family: Theme.mono
                                font.pixelSize: 14
                            }

                            Text {
                                anchors.horizontalCenter: parent.horizontalCenter
                                text: parent.modelData.temp + "°"
                                color: Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 11
                            }
                        }
                    }
                }
            }

            Text {
                visible: Weather.data !== null && !Weather.ready
                Layout.topMargin: 18
                text: Weather.data && Weather.data.error ? "Clima: " + Weather.data.error : ""
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
            }
        }
    }
}
