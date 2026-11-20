pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io

// CPU, memória, temperatura, rede e disco, a cada 2 s, com o último minuto
// guardado para os gráficos. `alert` acende quando algo passa do limite.
Singleton {
    id: root

    readonly property int interval: 2000
    readonly property int keep: 30

    property real cpu: 0
    property real mem: 0
    property real memUsedGb: 0
    property real memTotalGb: 0
    property real temp: 0
    property real rxRate: 0
    property real txRate: 0
    property real disk: 0
    property real diskUsedGb: 0
    property real diskTotalGb: 0

    property var cpuHistory: []
    property var memHistory: []
    property var tempHistory: []
    property var netHistory: []

    readonly property bool alert: cpu > 0.9 || mem > 0.9 || temp > 90 || disk > 0.95
    readonly property string alertText: cpu > 0.9 ? "CPU " + Math.round(cpu * 100) + "%"
        : mem > 0.9 ? "memória " + Math.round(mem * 100) + "%"
        : temp > 90 ? Math.round(temp) + "°C"
        : disk > 0.95 ? "disco cheio" : ""

    property var last: null

    function push(list, value) {
        const next = list.concat([value]);
        return next.length > keep ? next.slice(next.length - keep) : next;
    }

    // Bytes por segundo em texto curto: 512 B/s, 2,7 K/s, 1,2 M/s
    function rate(bps) {
        if (bps < 1024)
            return Math.round(bps) + " B/s";
        if (bps < 1024 * 1024)
            return (bps / 1024).toFixed(1).replace(".", ",") + " K/s";
        return (bps / 1024 / 1024).toFixed(1).replace(".", ",") + " M/s";
    }

    function parse(text) {
        const v = {};
        text.split("\n").forEach(line => {
            const parts = line.trim().split(/\s+/);
            if (parts.length === 2)
                v[parts[0]] = Number(parts[1]);
        });
        const now = Date.now();

        if (last) {
            const dt = Math.max(0.5, (now - last.time) / 1000);
            const total = v.cpu_total - last.cpu_total;
            cpu = total > 0 ? Math.max(0, Math.min(1, 1 - (v.cpu_idle - last.cpu_idle) / total)) : 0;
            rxRate = Math.max(0, (v.net_rx - last.net_rx) / dt);
            txRate = Math.max(0, (v.net_tx - last.net_tx) / dt);
            cpuHistory = push(cpuHistory, cpu);
            netHistory = push(netHistory, rxRate + txRate);
        }

        if (v.mem_total > 0) {
            mem = 1 - v.mem_avail / v.mem_total;
            memTotalGb = v.mem_total / 1024 / 1024;
            memUsedGb = (v.mem_total - v.mem_avail) / 1024 / 1024;
            memHistory = push(memHistory, mem);
        }
        if (v.temp > 0) {
            temp = v.temp / 1000;
            tempHistory = push(tempHistory, Math.min(1, temp / 100));
        }
        if (v.disk_total > 0) {
            disk = v.disk_used / v.disk_total;
            diskTotalGb = v.disk_total / 1e9;
            diskUsedGb = v.disk_used / 1e9;
        }

        v.time = now;
        last = v;
    }

    Timer {
        interval: root.interval
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: reader.running = true
    }

    Process {
        id: reader
        command: ["bash", Quickshell.shellPath("scripts/mare-stats")]
        stdout: StdioCollector {
            onStreamFinished: root.parse(text)
        }
    }
}
