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
    readonly property int pct: dev ? percent(dev.percentage) : 0
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

    // --- puro: início ---
    // Daqui até "puro: fim" nada toca no UPower nem no Maré: é a conta dos
    // degraus, testada fora do shell com node (tests/js/BatteryWatch.test.mjs).
    // O UPower dá a carga de 0 a 100; se um dia vier de 0 a 1, não zera.
    function percent(p) {
        return Math.round(p > 1 ? p : p * 100);
    }

    // Um passo do aviso. `charging`: na tomada (carregando, cheia ou esperando
    // carga); `discharging`: gastando a bateria; `enabled`: o aviso está ligado
    // nos Ajustes; `wasLow`/`wasCritical`: os degraus já avisados nesta
    // descarga. Devolve os degraus avisados depois do passo e o aviso a dar
    // agora: { low, critical, warn: "" | "low" | "critical" }.
    function step(p, charging, discharging, enabled, wasLow, wasCritical) {
        const next = { warn: "" };
        next.low = wasLow;
        next.critical = wasCritical;
        if (charging) {
            next.low = false;
            next.critical = false;
            return next;
        }
        if (p > low + slack)
            next.low = false;
        if (p > critical + slack)
            next.critical = false;
        if (!discharging || !enabled)
            return next;
        if (p <= critical && !next.critical) {
            next.critical = true;
            next.low = true;
            next.warn = "critical";
        } else if (p <= low && !next.low) {
            next.low = true;
            next.warn = "low";
        }
        return next;
    }
    // --- puro: fim ---

    function check() {
        if (!present)
            return;
        const charging = state === UPowerDeviceState.Charging || state === UPowerDeviceState.FullyCharged
            || state === UPowerDeviceState.PendingCharge;
        const discharging = state === UPowerDeviceState.Discharging;
        // O ajuste só é lido quando decide algo (fora da tomada e descarregando).
        const enabled = discharging && Settings.get("power.lowWarn") !== false;
        const next = step(pct, charging, discharging, enabled, warned.low, warned.critical);
        warned.low = next.low;
        warned.critical = next.critical;
        if (next.warn === "critical")
            Quickshell.execDetached(["notify-send", "-a", "Mk", "-u", "critical", "-i", "battery-caution",
                "bateria em " + pct + "%: ligue na tomada"]);
        else if (next.warn === "low")
            Quickshell.execDetached(["notify-send", "-a", "Mk", "-i", "battery-low", "bateria em " + pct + "%"]);
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
