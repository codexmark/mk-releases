pragma Singleton
import QtQuick
import Quickshell

// Os tokens do Maré (shell do Mk): preto quente, texto em osso e um só destaque, o cobre.
Singleton {
    readonly property color ground: "#0b0b0b"
    readonly property color surface: "#141211"
    readonly property color raised: "#1b1917"
    readonly property color hairline: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.13)
    readonly property color line: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.26)
    readonly property color wash: Qt.rgba(205 / 255, 196 / 255, 186 / 255, 0.05)
    readonly property color text: "#cdc4ba"
    readonly property color textSoft: "#b0a9a0"
    readonly property color textDim: "#958f87"
    readonly property color textFaint: "#716b69"
    readonly property color copper: "#ce5e32"

    readonly property var families: Qt.fontFamilies()
    readonly property string serif: families.indexOf("Fraunces") >= 0 ? "Fraunces" : "Noto Serif"
    readonly property string sans: families.indexOf("Space Grotesk") >= 0 ? "Space Grotesk" : "Inter"
    readonly property string mono: families.indexOf("Space Mono") >= 0 ? "Space Mono" : "JetBrainsMono Nerd Font"
    readonly property string symbols: families.indexOf("JetBrainsMono Nerd Font") >= 0 ? "JetBrainsMono Nerd Font" : "GeistMono Nerd Font"
    readonly property real labelSpacing: 1.6

    // Ícones Material da Nerd Font (a mesma que o Maré usa).
    readonly property var icon: ({
        play: String.fromCodePoint(0xF040A), pause: String.fromCodePoint(0xF03E4),
        next: String.fromCodePoint(0xF04AD), previous: String.fromCodePoint(0xF04AE),
        heart: String.fromCodePoint(0xF02D1), heartOutline: String.fromCodePoint(0xF02D5),
        shuffle: String.fromCodePoint(0xF049F), repeat: String.fromCodePoint(0xF0456),
        repeatOne: String.fromCodePoint(0xF0458), repeatOff: String.fromCodePoint(0xF0457),
        more: String.fromCodePoint(0xF01D8), download: String.fromCodePoint(0xF01DA),
        check: String.fromCodePoint(0xF012C), close: String.fromCodePoint(0xF0156),
        home: String.fromCodePoint(0xF02DC), search: String.fromCodePoint(0xF0349),
        radio: String.fromCodePoint(0xF0439), library: String.fromCodePoint(0xF0333),
        settings: String.fromCodePoint(0xF0493), back: String.fromCodePoint(0xF004D),
        timer: String.fromCodePoint(0xF051B), playlistAdd: String.fromCodePoint(0xF0412),
        queue: String.fromCodePoint(0xF0CB8), remove: String.fromCodePoint(0xF01B4),
        plus: String.fromCodePoint(0xF0415), retry: String.fromCodePoint(0xF0450),
        speed: String.fromCodePoint(0xF04C5), lyrics: String.fromCodePoint(0xF05DA),
        volumeHigh: String.fromCodePoint(0xF057E), volumeMedium: String.fromCodePoint(0xF0580),
        volumeLow: String.fromCodePoint(0xF057F), volumeOff: String.fromCodePoint(0xF0581),
        fullscreen: String.fromCodePoint(0xF0293), fullscreenExit: String.fromCodePoint(0xF0294),
        minus: String.fromCodePoint(0xF0374)
    })

    // "1 música", "12 músicas".
    function count(n, one, many) { return n + " " + (n === 1 ? one : many); }

    function clock(seconds) {
        const s = Math.max(0, Math.floor(seconds || 0));
        return Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
    }
}
