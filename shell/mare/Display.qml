pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// Tela: brilho (brightnessctl) e luz noturna (hyprsunset).
Singleton {
    id: root

    property bool hasBacklight: false
    property real brightness: 0
    readonly property int nightTemperature: Settings.get("night.temperature")
    readonly property bool nightLight: sunset.running

    // Com a luz ligada, trocar o calor reinicia o hyprsunset.
    onNightTemperatureChanged: {
        if (sunset.running) {
            sunset.running = false;
            sunset.running = true;
        }
    }

    // Luz noturna automática: liga e desliga nas horas dos Ajustes. Só age na
    // virada, então ligar ou desligar à mão vale até a próxima virada.
    property bool scheduleWanted: false
    function nightNow() {
        const h = new Date().getHours();
        const from = Settings.get("night.from"), to = Settings.get("night.to");
        return from > to ? (h >= from || h < to) : (h >= from && h < to);
    }
    function followSchedule(force) {
        if (!Settings.get("night.auto"))
            return;
        const want = nightNow();
        if (force || want !== scheduleWanted)
            sunset.running = want;
        scheduleWanted = want;
    }

    // Só quando o horário muda nos Ajustes (ou chega do arquivo no login).
    readonly property string nightPlan: [Settings.get("night.auto"), Settings.get("night.from"), Settings.get("night.to")].join()
    onNightPlanChanged: followSchedule(true)

    Timer {
        interval: 60000
        repeat: true
        running: true
        onTriggered: root.followSchedule(false)
    }

    function refresh() {
        info.running = true;
    }

    function set(value) {
        const pct = Math.round(Math.max(0.01, Math.min(1, value)) * 100);
        brightness = pct / 100;
        setter.command = ["brightnessctl", "-q", "set", pct + "%"];
        setter.running = true;
    }

    function toggleNight() {
        sunset.running = !sunset.running;
    }

    Component.onCompleted: refresh()

    // brightnessctl -m: device,class,atual,percentual,máximo
    Process {
        id: info
        command: ["brightnessctl", "-m", "-c", "backlight"]
        stdout: StdioCollector {
            onStreamFinished: {
                const pct = parseInt((text.split(",")[3] || "").replace("%", ""));
                root.hasBacklight = !isNaN(pct);
                if (root.hasBacklight)
                    root.brightness = pct / 100;
            }
        }
    }

    Process {
        id: setter
    }

    // Enquanto este processo vive, a tela fica quente; parar o processo
    // devolve a cor normal.
    Process {
        id: sunset
        command: ["hyprsunset", "-t", String(root.nightTemperature)]
    }
}
