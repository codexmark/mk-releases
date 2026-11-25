import QtQuick
import QtQuick.Layouts
import "Som.js" as Som

// A área do microfone: as entradas, o medidor de nível ao vivo, o volume e o
// mudo da escolhida, "gravar e ouvir" e o diagnóstico numa frase. Só mostra o
// que recebe e avisa o que a pessoa mexeu: quem fala com o motor é o App.
Item {
    id: microfone

    property var entradas: []
    property int entradaId: -1
    // Os últimos níveis do medidor (uns 2 s), o de agora e o pico que cai.
    property var niveis: []
    property real nivel: Som.PISO
    property real pico: Som.PISO
    property bool ouvindo: false
    // "" | "gravando" | "ouvindo", e os segundos que faltam da gravação.
    property string etapa: ""
    property int falta: 0
    property string recado: ""
    property bool estreita: false
    property bool ativo: true

    readonly property var entrada: Som.entradaDe(entradas, entradaId)
    readonly property var diagnostico: Som.diagnostico(entradas, entrada, niveis, ouvindo)

    signal escolhida(int id)
    signal padraoPedido(int id)
    signal volumeMexido(int id, real valor, bool soltou)
    signal mudoTrocado(int id)
    signal gravarPedido()
    signal atualizarPedido()

    ColumnLayout {
        anchors.fill: parent
        spacing: 10

        // ── as entradas ────────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            Rotulo {
                Layout.fillWidth: true
                text: microfone.entradas.length === 1 ? "1 entrada" : microfone.entradas.length + " entradas"
            }
            Botao {
                implicitHeight: 22
                texto: "atualizar"
                onAcionado: microfone.atualizarPedido()
            }
        }

        Flickable {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumHeight: 64
            contentHeight: lista.implicitHeight
            clip: true
            boundsBehavior: Flickable.StopAtBounds

            Column {
                id: lista
                width: parent.width
                spacing: 2

                Text {
                    visible: microfone.entradas.length === 0
                    width: parent.width
                    wrapMode: Text.WordWrap
                    text: "Nenhuma entrada de som. Ligue um microfone e clique em atualizar."
                    color: Theme.textDim
                    font.family: Theme.serif
                    font.italic: true
                    font.pixelSize: 14
                }

                Repeater {
                    model: microfone.entradas

                    Rectangle {
                        id: linha
                        required property var modelData
                        readonly property bool atual: modelData.id === microfone.entradaId
                        width: lista.width
                        height: 34
                        color: atual ? Theme.wash : Qt.rgba(0, 0, 0, 0)
                        border.width: atual ? 1 : 0
                        border.color: Theme.copper

                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: microfone.escolhida(linha.modelData.id)
                        }
                        Text {
                            anchors.left: parent.left
                            anchors.leftMargin: 10
                            anchors.right: marcas.left
                            anchors.rightMargin: 8
                            anchors.verticalCenter: parent.verticalCenter
                            elide: Text.ElideRight
                            text: linha.modelData.descricao
                            color: linha.atual ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 13
                        }
                        Row {
                            id: marcas
                            anchors.right: parent.right
                            anchors.rightMargin: 10
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 8
                            Rotulo {
                                visible: linha.modelData.mudo === true
                                text: "muda"
                                color: Theme.copper
                            }
                            Rotulo {
                                visible: linha.modelData.padrao === true
                                text: "padrão"
                            }
                            Rotulo { text: Som.textoDoVolume(linha.modelData.volume) }
                        }
                    }
                }
            }
        }

        // ── o medidor ──────────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10
            visible: microfone.entrada !== null

            Medidor {
                Layout.fillWidth: true
                nivel: microfone.nivel
                pico: microfone.pico
                ligado: microfone.ouvindo && !(microfone.entrada && microfone.entrada.mudo)
            }
            Text {
                Layout.preferredWidth: 58
                horizontalAlignment: Text.AlignRight
                text: microfone.ouvindo && microfone.niveis.length > 0 ? Som.textoDb(Math.round(microfone.pico)) : "—"
                color: Theme.textSoft
                font.family: Theme.mono
                font.pixelSize: 10
            }
        }

        // ── o volume e o mudo ──────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10
            visible: microfone.entrada !== null

            Rotulo { text: "volume" }
            Deslizante {
                Layout.fillWidth: true
                min: 0
                max: Som.VOLUME_MAX
                passo: 1
                passoGrande: 10
                valor: microfone.entrada && microfone.entrada.volume !== null ? microfone.entrada.volume : 0
                ligado: microfone.entrada !== null && !microfone.entrada.mudo
                ativo: microfone.ativo
                onMexido: (valor, soltou) => microfone.volumeMexido(microfone.entradaId, valor, soltou)
            }
            Text {
                Layout.preferredWidth: 40
                horizontalAlignment: Text.AlignRight
                text: Som.textoDoVolume(microfone.entrada ? microfone.entrada.volume : null)
                color: Theme.textSoft
                font.family: Theme.mono
                font.pixelSize: 10
            }
            Rotulo { text: "mudo" }
            Chave {
                ligado: microfone.entrada !== null && microfone.entrada.mudo === true
                onTrocado: microfone.mudoTrocado(microfone.entradaId)
            }
        }

        // ── gravar e ouvir, tornar padrão ──────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 8
            visible: microfone.entrada !== null

            Botao {
                texto: Som.textoDaGravacao(microfone.etapa, microfone.falta)
                forte: true
                ligado: microfone.etapa === ""
                onAcionado: microfone.gravarPedido()
            }
            Botao {
                visible: microfone.entrada !== null && !microfone.entrada.padrao
                texto: microfone.estreita ? "padrão" : "tornar a padrão"
                onAcionado: microfone.padraoPedido(microfone.entradaId)
            }
            Item { Layout.fillWidth: true }
        }

        // ── o diagnóstico ──────────────────────────────────────────────
        Text {
            Layout.fillWidth: true
            Layout.preferredHeight: 34
            wrapMode: Text.WordWrap
            maximumLineCount: 2
            elide: Text.ElideRight
            verticalAlignment: Text.AlignVCenter
            text: microfone.recado !== "" ? microfone.recado : microfone.diagnostico.texto
            color: microfone.recado !== "" || microfone.diagnostico.problema ? Theme.copper : Theme.textSoft
            font.family: Theme.serif
            font.pixelSize: 15
        }
    }
}
