pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.UPower
import "Modes.js" as Logic

// Os modos: apresentação (projetar a tela, compartilhar numa chamada) e jogo
// (tirar o desktop do caminho). Os dois ligam o não perturbe e seguram a
// ociosidade pela cafeína; o jogo ainda põe o perfil de energia em
// "desempenho", se a máquina oferecer, e desliga as animações, o desfoque e a
// sombra do Hyprland. São exclusivos: ligar um desliga o outro. Ao desligar,
// cada coisa volta ao que era antes, menos o que a pessoa mexeu à mão no
// meio. A conta de "o que era antes" está no Modes.js. Nada é guardado: ao
// entrar de novo na sessão, os dois começam desligados.
Singleton {
    id: root

    property var ledger: Logic.empty()
    readonly property string mode: ledger.mode
    readonly property bool on: mode !== ""
    readonly property bool present: mode === "present"
    readonly property bool game: mode === "game"
    readonly property string label: Logic.label(mode)

    // Enquanto o próprio modo mexe nos interruptores, a mudança não é "à mão".
    property bool applying: false
    // O que o Hyprland tinha ligado quando o jogo começou: { blur, shadow }.
    property var was: ({})

    function now() {
        return {
            dnd: Notifs.dnd,
            idle: Caffeine.on,
            // Sem o power-profiles-daemon não há perfil.
            profile: PowerProfiles.profile === undefined ? null : PowerProfiles.profile
        };
    }

    function caps() {
        return { performance: PowerProfiles.hasPerformanceProfile ? PowerProfile.Performance : null };
    }

    // Liga o modo `name` ("present" ou "game"), ou desliga se já é ele.
    // Devolve se ele ficou ligado.
    function toggle(name) {
        commit(Logic.toggle(ledger, name, now(), caps()));
        return mode === name;
    }

    function off() {
        commit(Logic.leave(ledger, now()));
    }

    function commit(step) {
        const played = game;
        applying = true;
        ledger = step.state;
        if (step.apply.dnd !== undefined)
            Notifs.dnd = step.apply.dnd;
        if (step.apply.idle !== undefined)
            Caffeine.on = step.apply.idle;
        if (step.apply.profile !== undefined)
            PowerProfiles.profile = step.apply.profile;
        applying = false;
        if (played !== game)
            quiet(game);
    }

    function reconcile() {
        if (!applying)
            ledger = Logic.reconcile(ledger, now());
    }

    // O Hyprland no modo jogo. Vai por `hyprctl eval`, como o Lua dos
    // Ajustes, e fica em Settings.overlay enquanto durar, para um ajuste
    // mexido no meio do jogo não religar as animações. As animações desligam
    // já; o desfoque e a sombra, só depois de lidos como ligados (found).
    function quiet(enable) {
        if (enable) {
            was = ({});
            Settings.overlay = Logic.hyprOff(was);
            send(Settings.overlay);
            blurProbe.running = true;
            shadowProbe.running = true;
        } else {
            Settings.overlay = "";
            send(Logic.hyprBack(was, Settings.get("motion.speed") > 0));
            was = ({});
        }
    }

    function send(lua) {
        Quickshell.execDetached(["hyprctl", "eval", lua]);
    }

    function found(key, text) {
        if (!game || Logic.optionOn(text) !== true)
            return;
        const next = Object.assign({}, was);
        next[key] = true;
        was = next;
        Settings.overlay = Logic.hyprOff(was);
        send(Settings.overlay);
    }

    Connections {
        target: Notifs
        function onDndChanged() {
            root.reconcile();
        }
    }

    Connections {
        target: Caffeine
        function onOnChanged() {
            root.reconcile();
        }
    }

    Connections {
        target: PowerProfiles
        function onProfileChanged() {
            root.reconcile();
        }
    }

    Process {
        id: blurProbe
        command: ["hyprctl", "-j", "getoption", "decoration:blur:enabled"]
        stdout: StdioCollector {
            onStreamFinished: root.found("blur", text)
        }
    }

    Process {
        id: shadowProbe
        command: ["hyprctl", "-j", "getoption", "decoration:shadow:enabled"]
        stdout: StdioCollector {
            onStreamFinished: root.found("shadow", text)
        }
    }
}
