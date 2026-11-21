pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Bluetooth

// O pareamento em andamento do painel de bluetooth. Fica aqui, e não no
// painel, porque o painel é destruído ao fechar e parear leva de 5 a 20
// segundos: fechando no meio, ninguém confiaria no aparelho, conectaria nele
// ou avisaria da falha. O painel só desenha este estado e o aviso.
Singleton {
    id: root

    readonly property var adapter: Bluetooth.defaultAdapter
    readonly property bool on: adapter !== null && adapter.enabled

    // O aparelho sendo pareado ou conectado agora, e o que deu errado por último.
    property var pairingDevice: null
    property var linkingDevice: null
    // Recém-pareado e ainda com o enlace do pareamento de pé: fica em
    // observação para reconectar se o enlace cair (ver afterPaired).
    property var watchedDevice: null
    property string notice: ""

    // Esperando o "pareou" que pode chegar logo depois do "terminou".
    readonly property bool settling: settle.running
    readonly property bool active: pairingDevice !== null || linkingDevice !== null || watchedDevice !== null

    function busy(d) {
        return d.pairing || d.state === BluetoothDeviceState.Connecting
            || d.state === BluetoothDeviceState.Disconnecting;
    }

    function pair(d) {
        pairingDevice = d;
        d.pair();
    }

    function cancelPair(d) {
        if (pairingDevice === d)
            pairingDevice = null;
        settle.stop();
        d.cancelPair();
    }

    function link(d) {
        // Pareado por fora: passa a confiar, senão ele não reconecta
        // sozinho depois.
        if (!d.trusted)
            d.trusted = true;
        linkingDevice = d;
        d.connect();
    }

    function unlink(d) {
        linkingDevice = null;
        d.disconnect();
    }

    // Pareou e o pedido de pareamento já respondeu (`pairing` voltou a
    // falso, o que só acontece com os serviços resolvidos): confia, para o
    // aparelho reconectar sozinho depois, e conecta.
    //
    // Para parear, o BlueZ já sobe o enlace: o aparelho costuma aparecer como
    // conectado ANTES de pareado, e o Quickshell recusa connect() com o
    // enlace de pé. Então, se já está conectado, só observa por alguns
    // segundos: muito fone e caixa de som abre os perfis sozinho e fica; se o
    // enlace ocioso cair nessa janela, aí sim conecta de verdade.
    function afterPaired(d) {
        pairingDevice = null;
        settle.stop();
        if (!d.trusted)
            d.trusted = true;
        if (d.connected) {
            watchedDevice = d;
            watchLink.restart();
        } else {
            linkAfterPair(d);
        }
    }

    function linkAfterPair(d) {
        if (!on || !d.paired || d.connected || busy(d))
            return;
        linkingDevice = d;
        d.connect();
    }

    function unwatch() {
        watchedDevice = null;
        watchLink.stop();
    }

    // Esquecido: nada mais a acompanhar nele.
    function drop(d) {
        if (pairingDevice === d)
            pairingDevice = null;
        if (linkingDevice === d)
            linkingDevice = null;
        if (watchedDevice === d)
            unwatch();
    }

    // Adaptador trocado ou desligado: o que estava em andamento morreu junto,
    // e não é uma falha para avisar.
    function reset() {
        pairingDevice = null;
        linkingDevice = null;
        settle.stop();
        unwatch();
    }

    onAdapterChanged: reset()
    onOnChanged: {
        if (!on) {
            reset();
            notice = "";
        }
    }

    // O pareamento não avisa quando falha: `pairing` volta a falso e `paired`
    // continua falso. Espera um instante, porque o "pareou" pode chegar logo
    // depois do "terminou".
    Timer {
        id: settle
        interval: 900
        onTriggered: {
            const d = root.pairingDevice;
            root.pairingDevice = null;
            if (!d || d.pairing || !root.on)
                return;
            if (d.paired)
                root.afterPaired(d);
            else
                root.notice = "não pareou com " + d.name
                    + ". confira se ele está em modo de pareamento; se pede um código, use o bluetoothctl.";
        }
    }

    Connections {
        target: root.pairingDevice
        function onPairedChanged() {
            const d = root.pairingDevice;
            if (!d || !d.paired)
                return;
            // Confia já; conectar fica para quando o pedido responder.
            if (!d.trusted)
                d.trusted = true;
            if (!d.pairing)
                root.afterPaired(d);
        }
        function onPairingChanged() {
            const d = root.pairingDevice;
            if (!d || d.pairing)
                return;
            if (d.paired)
                root.afterPaired(d);
            else
                settle.restart();
        }
    }

    // Janela de observação do recém-pareado (ver afterPaired).
    Timer {
        id: watchLink
        interval: 10000
        onTriggered: root.watchedDevice = null
    }

    Connections {
        target: root.watchedDevice
        function onConnectedChanged() {
            const d = root.watchedDevice;
            if (!d || d.connected)
                return;
            root.unwatch();
            // Fora do sinal: o estado do aparelho ainda está assentando.
            Qt.callLater(() => root.linkAfterPair(d));
        }
    }

    // Conectar também não avisa a falha: o estado vai a "conectando" e volta
    // a "desconectado".
    Connections {
        target: root.linkingDevice
        function onStateChanged() {
            const d = root.linkingDevice;
            if (!d)
                return;
            if (d.state === BluetoothDeviceState.Connected) {
                root.linkingDevice = null;
            } else if (d.state === BluetoothDeviceState.Disconnected) {
                root.linkingDevice = null;
                // Com o adaptador desligado no meio, não é falha do aparelho.
                if (root.on)
                    root.notice = "não conectou em " + d.name + ". ele está ligado e por perto?";
            }
        }
    }
}
