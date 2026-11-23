import QtQuick
import QtQuick.Effects

// Luz ambiente: a capa que toca, minúscula e desfocada, tingindo o alto do conteúdo e
// sumindo para o fundo. A imagem tem 48 px; o desfoque só é refeito quando a capa muda.
Item {
    id: ambient
    property string source
    clip: true

    Image {
        id: art
        anchors.fill: parent
        source: ambient.source
        sourceSize: Qt.size(48, 48)
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        visible: false
    }
    MultiEffect {
        anchors.fill: parent
        source: art
        blurEnabled: true
        blur: 1
        blurMax: 64
        saturation: 0.25
        opacity: art.status === Image.Ready && ambient.source !== "" ? 0.3 : 0
        Behavior on opacity { NumberAnimation { duration: 400 } }
    }
    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "transparent" }
            GradientStop { position: 1; color: Theme.ground }
        }
    }
}
