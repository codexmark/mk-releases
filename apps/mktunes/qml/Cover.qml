import QtQuick
import QtQuick.Window

// Capa com fundo enquanto carrega; nunca decodifica maior do que aparece.
// O canto arredondado custa uma camada só: a imagem vai para a própria camada e um shader de
// distância com sinal (shaders/rounded.frag) recorta o retângulo arredondado, ou o círculo.
// `clip` não serve: o Qt recorta só no retângulo, ignorando o raio.
Rectangle {
    id: cover
    property string source
    property bool round: false
    readonly property real corner: round ? width / 2 : Math.max(4, width * 0.08)
    radius: corner
    color: Theme.raised
    // Sem capa (rádio sem logo, capa que não carregou): uma nota no lugar.
    Text {
        anchors.centerIn: parent
        visible: image.status !== Image.Ready && cover.width >= 32
        text: String.fromCodePoint(0xF075A)
        color: Theme.textFaint
        font.family: Theme.symbols
        font.pixelSize: cover.width * 0.4
    }
    Image {
        id: image
        anchors.fill: parent
        source: cover.source
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        cache: true
        smooth: true
        sourceSize: Qt.size(Math.ceil(cover.width * Screen.devicePixelRatio), Math.ceil(cover.height * Screen.devicePixelRatio))
        visible: status === Image.Ready
        opacity: status === Image.Ready ? 1 : 0
        Behavior on opacity { NumberAnimation { duration: 150 } }
        layer.enabled: visible
        layer.effect: ShaderEffect {
            // Em pixels da camada, as mesmas unidades da textura.
            property vector2d size: Qt.vector2d(image.width * Screen.devicePixelRatio, image.height * Screen.devicePixelRatio)
            property real radius: cover.corner * Screen.devicePixelRatio
            fragmentShader: Qt.resolvedUrl("shaders/rounded.frag.qsb")
        }
    }
}
