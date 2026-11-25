pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.Pipewire

// Estado da linha: qual painel ela mostra agora.
// rest é a pílula; os outros são painéis que crescem a partir dela.
// toast e osd são passageiros: voltam sozinhos para rest.
Singleton {
    id: root

    readonly property var panels: ["project", "volume", "power", "network", "calendar", "picker",
        "launcher", "switcher", "notifications", "session", "media", "bluetooth", "display", "system",
        "clipboard", "wallpaper", "settings", "capture", "emoji", "controls"]
    readonly property var transients: ["toast", "osd"]

    // O descanso de tela está na tela (ScreenSaver.qml, um por monitor).
    property bool saver: false
    property string mode: "rest"

    // Quando um painel deu lugar a outro (e não abriu do repouso). O Monitor
    // usa para não confundir essa troca com um clique fora.
    property double switchedAt: 0
    property string lastMode: "rest"
    onModeChanged: {
        if (mode !== "rest" && lastMode !== "rest")
            switchedAt = Date.now();
        lastMode = mode;
    }

    // Onde o painel desce: o centro (em x, na tela) do item da ilha que foi
    // clicado. -1 = no meio (painéis abertos por atalho de teclado).
    property real originX: -1

    // Nada de geometria de tela mora aqui além de originX: este singleton é
    // um só para todos os monitores, e cada monitor tem a sua ilha e o seu
    // painel. O encaixe entre os dois é ligado no Monitor.qml.

    // Posição no Alt+Tab; o painel faz o módulo pelo número de janelas.
    property int switchIndex: 0

    // OSD: o que mudou (volume, brilho) e o novo valor, de 0 a 1.
    property string osdKind: ""
    property real osdValue: 0
    property bool osdMuted: false

    readonly property bool isTransient: transients.indexOf(mode) >= 0

    // Painéis que recebem o teclado enquanto abertos.
    // Todo painel aberto recebe o teclado: é o que faz o Esc fechá-lo (antes
    // só a busca, o Alt+Tab e a área de transferência recebiam, e nos outros
    // o Esc ia para a janela de baixo). Os passageiros (aviso, OSD), não.
    readonly property bool wantsKeyboard: mode !== "rest" && mode !== "picker" && !isTransient && !sheetMode

    // Para uma janela ignorar o espaço reservado da ilha (ficar por cima ou
    // por trás dela), o protocolo pede a zona exclusiva -1. Com o tamanho da
    // interface diferente de 100% (QT_SCALE_FACTOR) o valor é multiplicado
    // pela escala, e -1 virava 0 ("fique abaixo da ilha"): os painéis desciam
    // e o fundo parava antes do alto. Este é o valor que, depois da escala,
    // dá -1. Use com exclusionMode: ExclusionMode.Normal.
    readonly property real uiScale: Number(Quickshell.env("QT_SCALE_FACTOR")) || 1
    readonly property int ignoreZone: -Math.ceil(1 / uiScale)

    // Os Ajustes são uma folha no meio da tela (SettingsWindow), não descem da ilha.
    readonly property bool sheetMode: mode === "settings"
    property string settingsPage: "ilha"

    function toggle(panel) {
        if (panels.indexOf(panel) < 0)
            return;
        if (mode !== panel)
            originX = -1;
        mode = mode === panel ? "rest" : panel;
    }

    // Abre (ou fecha) um painel embaixo do item da ilha em `x`.
    function openAt(panel, x) {
        if (panels.indexOf(panel) < 0)
            return;
        if (mode === panel) {
            mode = "rest";
            return;
        }
        originX = x;
        mode = panel;
    }

    function open(panel) {
        if (panels.indexOf(panel) >= 0)
            mode = panel;
    }

    function close() {
        mode = "rest";
    }

    // Mostra um estado passageiro por `ms`, sem atropelar um painel aberto.
    function flash(transient, ms) {
        if (mode !== "rest" && !isTransient)
            return;
        mode = transient;
        flashTimer.interval = ms;
        flashTimer.restart();
    }

    function osd(kind, value, muted) {
        osdKind = kind;
        osdValue = Math.max(0, Math.min(1, value));
        osdMuted = muted === true;
        flash("osd", 1300);
    }

    // Primeiro Alt+Tab abre já na janela anterior; os seguintes avançam.
    function switchStep(step) {
        if (mode !== "switcher") {
            switchIndex = step > 0 ? 1 : -1;
            mode = "switcher";
        } else {
            switchIndex += step;
        }
    }

    function changeVolume(delta) {
        const sink = Pipewire.defaultAudioSink;
        if (!sink || !sink.audio)
            return;
        sink.audio.muted = false;
        sink.audio.volume = Math.max(0, Math.min(1, sink.audio.volume + delta));
    }

    function changeBrightness(arg) {
        brightnessProc.command = ["brightnessctl", "-m", "set", arg];
        brightnessProc.running = true;
    }

    Timer {
        id: flashTimer
        onTriggered: {
            if (root.isTransient)
                root.mode = "rest";
        }
    }

    // brightnessctl -m imprime: device,class,atual,percentual,máximo
    Process {
        id: brightnessProc
        stdout: StdioCollector {
            onStreamFinished: {
                const pct = parseInt((text.split(",")[3] || "").replace("%", ""));
                if (!isNaN(pct)) {
                    Display.brightness = pct / 100;
                    root.osd("brightness", pct / 100);
                }
            }
        }
    }

    // qs -c mare ipc call mare toggle calendar
    IpcHandler {
        target: "mare"

        function toggle(panel: string): void {
            root.toggle(panel);
        }

        function close(): void {
            root.close();
        }


        // Captura direta, sem painel: shot copiar | area | janela | tela | ler
        function shot(mode: string): void {
            root.close();
            Capture.shot(mode);
        }

        // Começa ou para a gravação da tela.
        function record(): void {
            Capture.toggleRecording();
        }

        // Bloqueia a tela (Super + L, hypridle, antes de suspender).
        function lock(): void {
            root.close();
            root.saver = false;
            Lock.lock();
        }

        // O descanso de tela: saver on | off | toggle. Quem liga é o
        // hypridle (Ajustes › Energia); qualquer tecla ou o mouse desliga.
        function saver(state: string): string {
            if (state === "on") {
                root.close();
                root.saver = !Lock.locked;
            } else if (state === "off")
                root.saver = false;
            else if (state === "toggle") {
                root.close();
                root.saver = !root.saver && !Lock.locked;
            } else
                return "uso: saver on | off | toggle";
            return root.saver ? "ligado" : "desligado";
        }

        // Abre os Ajustes numa página: settings janelas
        function settings(page: string): void {
            if (page !== "")
                root.settingsPage = page;
            root.mode = "settings";
        }

        // Muda um ajuste, como o painel: set hypr.gapsIn 10 / set island.showWeather false
        function set(key: string, value: string): string {
            if (Settings.defaults[key] === undefined)
                return "ajuste desconhecido: " + key;
            try {
                Settings.set(key, JSON.parse(value));
            } catch (e) {
                Settings.set(key, value);
            }
            return key + " = " + JSON.stringify(Settings.get(key));
        }

        // Boas-vindas: welcome abrir | continuar | fechar | fim | seguir | voltar | estado
        function welcome(action: string): string {
            if (action === "abrir") {
                root.close();
                Welcome.open();
            } else if (action === "continuar")
                Welcome.resume();
            else if (action === "fechar")
                Welcome.close();
            else if (action === "fim")
                Welcome.finish();
            else if (action === "seguir")
                Welcome.next();
            else if (action === "voltar")
                Welcome.back();
            else if (action !== "estado")
                return "uso: welcome abrir | continuar | fechar | fim | seguir | voltar | estado";
            return (!Welcome.shown ? "fechada" : (Welcome.paused ? "recolhida" : "aberta") + ", passo " + (Welcome.step + 1) + " de " + Welcome.count)
                + (Welcome.seen ? ", já vista" : ", ainda não vista");
        }

        function state(): string {
            return root.saver ? "saver" : root.mode;
        }

        // Abre um painel embaixo da posição x da tela (como um clique na ilha).
        function toggleAt(panel: string, x: real): void {
            root.openAt(panel, x);
        }

        // Alt+Tab: next / prev
        function switcher(direction: string): void {
            root.switchStep(direction === "prev" ? -1 : 1);
        }

        // Roda um programa no diretório do projeto em foco: here lazygit
        function here(cmd: string): void {
            Projects.openHere(cmd);
        }

        // Teclas de mídia: volume +5 / -5 / mute; brilho +5 / -5 (em %)
        function volume(change: string): void {
            if (change === "mute") {
                const sink = Pipewire.defaultAudioSink;
                if (sink && sink.audio)
                    sink.audio.muted = !sink.audio.muted;
            } else {
                root.changeVolume(parseInt(change) / 100);
            }
        }

        function brightness(change: string): void {
            const n = parseInt(change);
            root.changeBrightness(Math.abs(n) + "%" + (n < 0 ? "-" : "+"));
        }

        function dnd(): bool {
            Notifs.dnd = !Notifs.dnd;
            return Notifs.dnd;
        }

        // Modo apresentação e modo jogo (Modes.qml): alternam e devolvem se
        // ficaram ligados. Ligar um desliga o outro.
        function present(): bool {
            return Modes.toggle("present");
        }

        function game(): bool {
            return Modes.toggle("game");
        }
    }
}
