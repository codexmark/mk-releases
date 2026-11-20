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
        windupAnim.restart();
        pam.start();
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
        onCompleted: result => {
            if (result === PamResult.Success) {
                root.authOk = true;
                root.maybeBoom();
            } else {
                root.busy = false;
                root.failed = true;
                root.message = "senha errada";
                root.password = "";
                windupAnim.stop();
                unwind.restart();
                root.shake();
            }
        }
        onError: error => {
            root.busy = false;
            root.failed = true;
            root.message = "não deu para conferir a senha (PAM)";
            windupAnim.stop();
            unwind.restart();
            root.shake();
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

    // Nome de verdade do usuário (o campo GECOS), para a tela.
    Process {
        running: true
        command: ["sh", "-c", "getent passwd \"$USER\" | cut -d: -f5 | cut -d, -f1"]
        stdout: StdioCollector {
            onStreamFinished: root.realName = text.trim()
        }
    }
}
