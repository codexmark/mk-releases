pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Services.Mpris

// O player em destaque: o escolhido no painel; senão o que está tocando;
// senão o último que tocou.
Singleton {
    id: root

    readonly property var players: Mpris.players.values
    property var pinned: null
    property var lastPlaying: null
    readonly property var player: (pinned && players.indexOf(pinned) >= 0 ? pinned : null)
        ?? players.find(p => p.isPlaying)
        ?? (lastPlaying && players.indexOf(lastPlaying) >= 0 ? lastPlaying : players[0] ?? null)
    readonly property bool active: player !== null && player.trackTitle !== ""

    onPlayerChanged: {
        if (player && player.isPlaying)
            lastPlaying = player;
    }

    function format(seconds) {
        if (!seconds || seconds < 0)
            return "0:00";
        const s = Math.floor(seconds);
        return Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
    }

    // O MPRIS não avisa quando a posição anda: pede a atualização a cada segundo
    // enquanto toca.
    Timer {
        interval: 1000
        repeat: true
        running: root.player !== null && root.player.isPlaying
        onTriggered: root.player.positionChanged()
    }
}
