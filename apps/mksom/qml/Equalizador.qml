import QtQuick
import QtQuick.Layouts
import "Som.js" as Som

// A área do equalizador: a chave grande, a predefinição, a curva resultante,
// as 10 bandas, o pré-ganho e o aviso de distorção. Só mostra o que recebe e
// avisa o que a pessoa mexeu: quem fala com o motor é o App.
Item {
    id: equalizador

    property bool ligado: false
    property bool ocupado: false
    property bool ativo: true
    property var ganhos: Som.plano()
    property real pre: 0
    property string predefinicao: ""
    property string saida: ""
    property bool estreita: false
    property bool baixa: false

    readonly property string aviso: Som.avisoDeDistorcao(ganhos, pre)

    signal alternado()
    signal mexido(int banda, real valor, bool soltou)
    signal preMexido(real valor, bool soltou)
    signal predefinicoesPedidas()
    signal guardarPedido()
    signal planoPedido()
    signal preCorrigido()

    // Põe o foco na primeira banda (ao abrir, e ao fechar uma folha).
    function focar() {
        if (bandas.count > 0)
            bandas.itemAt(0).forceActiveFocus();
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: equalizador.baixa ? 8 : 12

        // ── a chave e as predefinições ─────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Interruptor {
                ligado: equalizador.ligado
                mudavel: !equalizador.ocupado
                onTrocado: equalizador.alternado()
            }
            Column {
                Layout.fillWidth: true
                spacing: 1
                Text {
                    width: parent.width
                    elide: Text.ElideRight
                    text: equalizador.ocupado ? (equalizador.ligado ? "desligando…" : "ligando…") : equalizador.ligado ? "ligado" : "desligado"
                    color: equalizador.ligado ? Theme.copper : Theme.textSoft
                    font.family: Theme.serif
                    font.pixelSize: 17
                }
                Text {
                    width: parent.width
                    elide: Text.ElideRight
                    text: equalizador.saida
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                }
            }
            Botao {
                texto: (equalizador.predefinicao !== "" ? equalizador.predefinicao : "do seu jeito") + "  ▾"
                Layout.maximumWidth: equalizador.estreita ? 130 : 190
                clip: true
                onAcionado: equalizador.predefinicoesPedidas()
            }
        }

        // ── a curva ────────────────────────────────────────────────────
        Curva {
            Layout.fillWidth: true
            Layout.preferredHeight: 58
            visible: !equalizador.baixa
            ganhos: equalizador.ganhos
            pre: equalizador.pre
            ligado: equalizador.ligado
        }

        // ── as bandas ──────────────────────────────────────────────────
        Row {
            id: fileira
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumHeight: 120

            Repeater {
                id: bandas
                model: Som.BANDAS

                Banda {
                    id: uma
                    required property int index
                    required property var modelData
                    width: fileira.width / Som.BANDAS.length
                    height: fileira.height
                    frequencia: modelData
                    valor: equalizador.ganhos[index] || 0
                    ligado: equalizador.ligado
                    estreita: equalizador.estreita
                    ativo: equalizador.ativo
                    onMexido: (valor, soltou) => equalizador.mexido(uma.index, valor, soltou)
                }
            }
        }

        // ── o pré-ganho e as ações ─────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Rotulo { text: equalizador.estreita ? "pré" : "pré-ganho" }
            Deslizante {
                Layout.fillWidth: true
                min: Som.PRE_MIN
                max: Som.PRE_MAX
                passo: Som.PASSO
                passoGrande: Som.PASSO_GRANDE
                valor: equalizador.pre
                ligado: equalizador.ligado
                ativo: equalizador.ativo
                onMexido: (valor, soltou) => equalizador.preMexido(valor, soltou)
            }
            Text {
                Layout.preferredWidth: 52
                horizontalAlignment: Text.AlignRight
                text: Som.textoDb(equalizador.pre)
                color: Theme.textSoft
                font.family: Theme.mono
                font.pixelSize: 10
            }
            Botao {
                texto: equalizador.estreita ? "guardar…" : "guardar como…"
                onAcionado: equalizador.guardarPedido()
            }
            Botao {
                texto: "plano"
                ligado: equalizador.predefinicao !== "plano"
                onAcionado: equalizador.planoPedido()
            }
        }

        // ── o aviso: os ganhos passam do pré-ganho ─────────────────────
        RowLayout {
            Layout.fillWidth: true
            Layout.preferredHeight: 28
            spacing: 10

            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: equalizador.aviso !== "" ? equalizador.aviso : "↑ ↓ mexem na banda em foco  ·  f1 mostra os atalhos"
                color: equalizador.aviso !== "" ? Theme.copper : Theme.textFaint
                font.family: Theme.sans
                font.pixelSize: 11
            }
            Botao {
                visible: equalizador.aviso !== "" && Som.preSugerido(equalizador.ganhos) < equalizador.pre
                texto: "baixar"
                forte: true
                onAcionado: equalizador.preCorrigido()
            }
        }
    }
}
