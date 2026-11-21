pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.Pam

// O bloqueio do Mk: o estado que todas as telas dividem (LockSurface.qml
// desenha). Super + L, o hypridle e a suspensão chamam `ipc call mare lock`.
//
// O desbloqueio segue a skin Orbital do Ryoku: ao apertar Enter os anéis do
// relógio dão corda (aceleram, a tela treme, saem faíscas) enquanto o PAM
// confere a senha; se der certo, um clarão e a tela some; se não, a corda
// volta e o painel balança.
Singleton {
    id: root

    property bool locked: false
    property string password: ""
    property bool busy: false
    property bool failed: false
    property string message: ""
    property string realName: ""
    readonly property string user: Quickshell.env("USER") || ""

    // Animação: corda (0..1), clarão (0..1) e a escala do "estouro".
    property real windup: 0
    property real boom: 0
    property real boomScale: 1
    property bool authOk: false

    // Caps Lock, lido do LED do teclado só enquanto a tela está bloqueada.
    property bool capsLock: false

    // O que o faillock disse nesta tentativa: conta travada e por quantos minutos.
    property bool lockedOut: false
    property string lockedFor: ""

    // Layout ativo quando há mais de um. Ligado aqui para o Keyboard já ter
    // perguntado ao Hyprland antes do primeiro bloqueio.
    readonly property string layout: Keyboard.multiple ? Keyboard.current : ""

    // Quando a tentativa começou: o Esc só larga depois de 3 s, para não
    // servir de atalho por cima da espera do PAM numa senha errada.
    property double startedAt: 0

    signal shake

    function lock() {
        if (locked)
            return;
        password = "";
        failed = false;
        message = "";
        windup = 0;
        boom = 0;
        boomScale = 1;
        authOk = false;
        busy = false;
        locked = true;
    }

    function submit() {
        if (busy || password === "")
            return;
        busy = true;
        failed = false;
        authOk = false;
        message = "";
        lockedOut = false;
        lockedFor = "";
        startedAt = Date.now();
        windupAnim.restart();
        hangTimer.restart();
        pam.start();
    }

    // A tentativa não passou: a corda volta e o painel balança.
    function fail(text, clear) {
        hangTimer.stop();
        busy = false;
        authOk = false;
        failed = true;
        message = text;
        if (clear)
            password = "";
        windupAnim.stop();
        unwind.restart();
        shake();
    }

    // O PAM travou: larga a tentativa. Nunca desbloqueia; depois que o PAM
    // disse sim, não há o que cancelar.
    function drop() {
        if (!busy || authOk)
            return;
        pam.abort();
        fail("não respondeu, tente de novo", true);
    }

    // O Esc da tela.
    function cancel() {
        if (Date.now() - startedAt >= 3000)
            drop();
    }

    Timer {
        id: hangTimer
        interval: 10000
        onTriggered: root.drop()
    }

    function lockedText(min) {
        return "conta travada, tentativas demais" + (min ? " · " + min + " min" : "");
    }

    // Minutos de trava que restam segundo a contagem do faillock (0 = conta
    // livre, -1 = travada sem prazo). `text` é a saída do tallyProc: a
    // contagem, as linhas do pam_faillock no system-auth e o faillock.conf.
    function lockoutLeft(text, now) {
        const parts = text.split(/^---$/m);
        const opt = (name, def) => {
            const m = (parts[1] || "").match(new RegExp("\\b" + name + "=(\\d+)"))
                || (parts[2] || "").match(new RegExp("^\\s*" + name + "\\s*=\\s*(\\d+)", "m"));
            return m ? parseInt(m[1]) : def;
        };
        const deny = opt("deny", 3);
        const unlock = opt("unlock_time", 600);
        const interval = opt("fail_interval", 900);
        const times = [];
        for (const l of (parts[0] || "").split("\n")) {
            const m = l.match(/^(\d{4}-\d\d-\d\d) (\d\d:\d\d:\d\d)\s.*\sV\s*$/);
            if (m)
                times.push(new Date(m[1] + "T" + m[2]).getTime() / 1000);
        }
        if (deny === 0 || times.length === 0)
            return 0;
        const last = Math.max(...times);
        if (times.filter(t => last - t < interval).length < deny)
            return 0;
        if (unlock === 0)
            return -1;
        const left = last + unlock - now / 1000;
        return left > 0 ? Math.ceil(left / 60) : 0;
    }

    // Com `silent` no preauth (o padrão do Arch) o faillock não avisa nada
    // pela conversa; depois de uma senha recusada vale a contagem dele, que o
    // próprio usuário pode ler.
    Process {
        id: tallyProc
        command: ["sh", "-c", "faillock --user \"$USER\" 2>/dev/null; echo ---; grep -h '^auth.*pam_faillock' /etc/pam.d/system-auth 2>/dev/null; echo ---; cat /etc/security/faillock.conf 2>/dev/null"]
        stdout: StdioCollector {
            onStreamFinished: {
                // Só troca o texto de uma falha que ainda está na tela.
                if (!root.failed || root.busy || root.message !== "senha errada")
                    return;
                const left = root.lockoutLeft(text, Date.now());
                if (left !== 0)
                    root.message = root.lockedText(left > 0 ? left : "");
            }
        }
    }

    // Quando a corda chega ao fim e o PAM já disse sim: o clarão.
    function maybeBoom() {
        if (authOk && !windupAnim.running && !boomAnim.running)
            boomAnim.restart();
    }

    PamContext {
        id: pam
        // A mesma pilha do hyprlock (auth include login), que vem com o Mk.
        config: "hyprlock"
        user: root.user

        onResponseRequiredChanged: {
            if (responseRequired) {
                respond(root.password);
            }
        }
        // O pam_faillock avisa pela conversa que a conta está travada, em
        // inglês ou em português conforme o locale.
        onPamMessage: {
            const m = pam.message || "";
            if (/locked|bloquead/i.test(m))
                root.lockedOut = true;
            const left = m.match(/(\d+)\s*min|min\D*(\d+)/i);
            if (left)
                root.lockedFor = left[1] || left[2];
        }
        onCompleted: result => {
            // Uma tentativa já largada (tempo esgotado, Esc) não vale mais nada.
            if (!root.busy || root.authOk)
                return;
            if (result === PamResult.Success) {
                hangTimer.stop();
                root.authOk = true;
                root.maybeBoom();
            } else if (root.lockedOut) {
                root.fail(root.lockedText(root.lockedFor), true);
            } else if (result === PamResult.MaxTries) {
                root.fail("tentativas demais", true);
            } else {
                root.fail("senha errada", true);
                tallyProc.running = true;
            }
        }
        onError: error => {
            if (!root.busy || root.authOk)
                return;
            root.fail("não deu para conferir a senha (PAM)", false);
        }
    }

    NumberAnimation {
        id: windupAnim
        target: root
        property: "windup"
        from: 0
        to: 1
        duration: 1400
        easing.type: Easing.InQuint
        onFinished: root.maybeBoom()
    }

    NumberAnimation {
        id: unwind
        target: root
        property: "windup"
        to: 0
        duration: 500
        easing.type: Easing.OutCubic
    }

    SequentialAnimation {
        id: boomAnim
        ParallelAnimation {
            NumberAnimation { target: root; property: "boomScale"; to: 30; duration: 170; easing.type: Easing.InQuad }
            NumberAnimation { target: root; property: "boom"; to: 1; duration: 130; easing.type: Easing.InQuad }
        }
        ScriptAction {
            script: {
                root.locked = false;
                root.busy = false;
                root.password = "";
                root.windup = 0;
                root.boom = 0;
                root.boomScale = 1;
            }
        }
    }

    // Caps Lock: o Quickshell não expõe os modificadores, então vale o LED do
    // teclado no sysfs, relido a cada meio segundo e só com a tela bloqueada.
    Timer {
        interval: 500
        repeat: true
        triggeredOnStart: true
        running: root.locked
        onTriggered: capsProc.running = true
        onRunningChanged: if (!running) root.capsLock = false
    }
    Process {
        id: capsProc
        command: ["sh", "-c", "cat /sys/class/leds/*capslock*/brightness 2>/dev/null"]
        stdout: StdioCollector {
            onStreamFinished: root.capsLock = root.locked && /[1-9]/.test(text)
        }
    }

    // Nome de verdade do usuário (o campo GECOS), para a tela.
    Process {
        running: true
        command: ["sh", "-c", "getent passwd \"$USER\" | cut -d: -f5 | cut -d, -f1"]
        stdout: StdioCollector {
            onStreamFinished: root.realName = text.trim()
        }
    }
}
