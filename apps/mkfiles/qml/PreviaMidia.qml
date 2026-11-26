import QtQuick
import QtMultimedia

// O tocador da prévia rápida (áudio e vídeo), pelo QtMultimedia. Mora num
// arquivo à parte, criado só por um Loader: se o módulo faltar na máquina, é
// só este pedaço que não carrega, e a prévia diz o que falta.
Item {
    id: midia

    property string fonte: ""
    property bool comVideo: false
    readonly property bool tocando: tocador.playbackState === MediaPlayer.PlayingState

    function alternar() {
        if (tocando)
            tocador.pause();
        else
            tocador.play();
    }
    function parar() { tocador.stop(); }
    function tempo(ms) {
        const s = Math.max(0, Math.floor(ms / 1000));
        const m = Math.floor(s / 60);
        return m + ":" + (s % 60 < 10 ? "0" : "") + (s % 60);
    }
    onFonteChanged: {
        if (fonte !== "")
            tocador.play();
    }

    MediaPlayer {
        id: tocador
        source: midia.fonte
        audioOutput: AudioOutput {
            id: som
            volume: 0.8
        }
        videoOutput: tela
    }
    VideoOutput {
        id: tela
        visible: midia.comVideo
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: controles.top
        anchors.bottomMargin: 8
        fillMode: VideoOutput.PreserveAspectFit
    }
    Text {
        visible: !midia.comVideo
        anchors.centerIn: tela
        text: "♪"
        color: midia.tocando ? Theme.copper : Theme.textFaint
        font.pixelSize: 72
    }
    Text {
        visible: tocador.errorString !== ""
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: controles.top
        anchors.bottomMargin: 12
        width: parent.width - 40
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.WordWrap
        text: "Não deu para tocar: " + tocador.errorString
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        id: controles
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        height: 28
        spacing: 10

        Botao {
            implicitWidth: 76
            texto: midia.tocando ? "pausar" : "tocar"
            forte: true
            onAcionado: midia.alternar()
        }
        Deslize {
            anchors.verticalCenter: parent.verticalCenter
            width: Math.max(80, midia.width - 400)
            valor: tocador.duration > 0 ? tocador.position / tocador.duration : 0
            onMovido: v => {
                if (tocador.seekable)
                    tocador.position = v * tocador.duration;
            }
        }
        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: midia.tempo(tocador.position) + " / " + midia.tempo(tocador.duration)
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }
        Rotulo {
            anchors.verticalCenter: parent.verticalCenter
            text: "volume"
        }
        Deslize {
            anchors.verticalCenter: parent.verticalCenter
            width: 80
            valor: som.volume
            onMovido: v => som.volume = v
        }
    }
}
