pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Hyprland

// Único ponto que manda comandos ao Hyprland. Com a config em Lua, o
// `dispatch` do Hyprland espera código Lua (hl.dsp...), não o formato antigo
// ("exec ...", "workspace 2"); aqui cada ação sai no formato que a sessão entende.
Singleton {
    readonly property bool lua: Hyprland.usingLua

    // Texto JS -> string Lua entre aspas (o escape do JSON é aceito pelo Lua).
    function quote(s) {
        return JSON.stringify(String(s));
    }

    function exec(cmd) {
        Hyprland.dispatch(lua ? `hl.dsp.exec_cmd(${quote(cmd)})` : `exec ${cmd}`);
    }

    function workspace(id) {
        Hyprland.dispatch(lua ? `hl.dsp.focus({ workspace = ${Number(id)} })` : `workspace ${Number(id)}`);
    }

    function focusWindow(address) {
        const addr = String(address).startsWith("0x") ? address : "0x" + address;
        Hyprland.dispatch(lua ? `hl.dsp.focus({ window = ${quote("address:" + addr)} })` : `focuswindow address:${addr}`);
    }
}
