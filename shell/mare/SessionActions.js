.pragma library

// As ações de sessão, numa lista só para o painel Sessão e para os Controles.
// `confirm` pede um segundo clique (o botão vira "confirmar" por 3 segundos).
// Dormir é o que Ajustes › Energia escolheu (Settings.sleepMode): só com
// "suspender" o botão existe.
function actions(sleepMode) {
    var all = [
        { id: "lock", icon: "󰌾", label: "bloquear", confirm: false, cmd: ["loginctl", "lock-session"] },
        { id: "suspend", icon: "󰤄", label: "suspender", confirm: false, cmd: ["systemctl", "suspend"], sleep: "suspender" },
        { id: "logout", icon: "󰍃", label: "sair", confirm: true, cmd: ["sh", "-c", "uwsm stop || hyprctl dispatch 'hl.dsp.exit()'"] },
        { id: "reboot", icon: "󰜉", label: "reiniciar", confirm: true, cmd: ["systemctl", "reboot"] },
        { id: "poweroff", icon: "󰐥", label: "desligar", confirm: true, cmd: ["systemctl", "poweroff"] }
    ];
    return all.filter(function (a) { return a.sleep === undefined || a.sleep === sleepMode; });
}
