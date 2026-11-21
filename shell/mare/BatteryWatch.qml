pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Services.UPower

// Aviso de bateria fraca em dois degraus: 20% (normal) e 8% (crítico, que
// fura o não perturbe). Cada degrau avisa uma vez por descarga e se rearma
// na tomada ou quando a carga passa do degrau + 3. Sem bateria, não faz nada.
Singleton {
    id: root

    readonly property var dev: UPower.displayDevice
    readonly property bool present: dev !== null && dev.ready && dev.isLaptopBattery
    readonly property int pct: dev ? Math.round(dev.percentage > 1 ? dev.percentage : dev.percentage * 100) : 0
    readonly property int state: dev ? dev.state : UPowerDeviceState.Unknown

    readonly property int low: 20
    readonly property int critical: 8
    readonly property int slack: 3

    // Sobrevive ao recarregar do shell, para não repetir o aviso já dado.
    PersistentProperties {
        id: warned
        reloadableId: "batteryWatch"
        property bool low: false
        property bool critical: false
    }

    function check() {
        if (!present)
            return;
        if (state === UPowerDeviceState.Charging || state === UPowerDeviceState.FullyCharged
                || state === UPowerDeviceState.PendingCharge) {
            warned.low = false;
            warned.critical = false;
            return;
        }
        if (pct > low + slack)
            warned.low = false;
        if (pct > critical + slack)
            warned.critical = false;
        if (state !== UPowerDeviceState.Discharging || Settings.get("power.lowWarn") === false)
            return;
        if (pct <= critical && !warned.critical) {
            warned.critical = true;
            warned.low = true;
            Quickshell.execDetached(["notify-send", "-a", "Mk", "-u", "critical", "-i", "battery-caution",
                "bateria em " + pct + "%: ligue na tomada"]);
        } else if (pct <= low && !warned.low) {
            warned.low = true;
            Quickshell.execDetached(["notify-send", "-a", "Mk", "-i", "battery-low", "bateria em " + pct + "%"]);
        }
    }

    onPresentChanged: settle.restart()
    onPctChanged: settle.restart()
    onStateChanged: settle.restart()

    // Espera a leitura assentar: no início o UPower ainda não respondeu, e ao
    // tirar da tomada o estado oscila por um instante.
    Timer {
        id: settle
        interval: 3000
        running: true
        onTriggered: root.check()
    }
}
