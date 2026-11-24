import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// A folha de confirmação: antes de agir, o plano (o que entra, o que sai,
// quanto pesa e os comandos que vão rodar); durante, o registro do progresso;
// no fim, se deu certo. Mostra o estado de Acao; nada aqui decide.
Item {
    id: folha

    readonly property var plano: Acao.plano
    readonly property bool instalando: plano !== null && plano.acao === "instalar"
    readonly property var itens: plano === null ? [] : instalando ? (plano.entra || []) : (plano.sai || [])

    // O fundo apaga a janela e segura os cliques.
    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.86
    }
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.AllButtons
        onWheel: wheel => wheel.accepted = true
    }
    focus: visible
    Keys.onEscapePressed: Acao.fechar()

    Rectangle {
        id: cartao
        anchors.centerIn: parent
        width: Math.min(680, parent.width - 80)
        height: Math.min(560, parent.height - 60)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line

        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 24
            spacing: 12

            Rotulo {
                text: Acao.fase === "agindo" ? "em andamento" : Acao.fase === "feito" ? "feito" : Acao.fase === "falhou" ? "não deu certo"
                    : Acao.fase === "recusa" ? (Acao.recusado ? "recusado" : "não dá") : "confirmação"
                color: Acao.fase === "falhou" || Acao.fase === "recusa" ? Theme.copper : Theme.textFaint
            }
            Text {
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                text: Acao.fase === "planejando" ? "Conferindo o que isso muda…"
                    : folha.plano !== null ? (Acao.fase === "plano" ? Pacotes.tituloDoPlano(folha.plano) : Pacotes.tituloDoPlano(folha.plano).replace("?", ""))
                    : Acao.fase === "recusa" ? "O MkPacotes não fez nada" : ""
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 24
                font.weight: Font.Light
            }

            // ── o ensaio falhou ou foi recusado ────────────────────────
            Text {
                Layout.fillWidth: true
                visible: Acao.fase === "recusa" || Acao.fase === "falhou"
                wrapMode: Text.WordWrap
                text: Acao.erro
                color: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 13
            }

            // ── o plano ────────────────────────────────────────────────
            Text {
                Layout.fillWidth: true
                visible: Acao.fase === "plano"
                wrapMode: Text.WordWrap
                text: Pacotes.resumoDoPlano(folha.plano)
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 13
            }
            Repeater {
                model: Acao.fase === "plano" && folha.plano !== null ? (folha.plano.avisos || []) : []
                Text {
                    required property var modelData
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                    text: "atenção: " + modelData
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
            }
            Text {
                Layout.fillWidth: true
                visible: Acao.fase === "plano" && folha.plano !== null && folha.plano.travado === true && folha.plano.acao !== "atualizar"
                wrapMode: Text.WordWrap
                text: "O pacman está travado agora (outro pacman está rodando): espere ele terminar e peça de novo."
                color: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 12
            }

            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: Acao.fase === "plano" && folha.itens.length > 0
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                ListView {
                    id: listaDoPlano
                    anchors.fill: parent
                    anchors.margins: 1
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: folha.itens
                    delegate: Item {
                        id: linhaDoPlano
                        required property var modelData
                        width: listaDoPlano.width
                        height: 26
                        Text {
                            x: 12
                            anchors.verticalCenter: parent.verticalCenter
                            text: folha.instalando ? "+" : "−"
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                        Text {
                            x: 30
                            width: parent.width - 30 - 300
                            anchors.verticalCenter: parent.verticalCenter
                            text: linhaDoPlano.modelData.nome
                            color: linhaDoPlano.modelData.pedido ? Theme.text : Theme.textSoft
                            font.family: Theme.mono
                            font.pixelSize: 11
                            elide: Text.ElideRight
                        }
                        Text {
                            anchors.right: parent.right
                            anchors.rightMargin: 196
                            width: 100
                            horizontalAlignment: Text.AlignRight
                            anchors.verticalCenter: parent.verticalCenter
                            text: linhaDoPlano.modelData.versao || ""
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideRight
                        }
                        Text {
                            anchors.right: parent.right
                            anchors.rightMargin: 96
                            width: 90
                            horizontalAlignment: Text.AlignRight
                            anchors.verticalCenter: parent.verticalCenter
                            text: linhaDoPlano.modelData.pedido ? (folha.instalando ? Pacotes.origem(linhaDoPlano.modelData.origem) : "pedido")
                                : (folha.instalando ? "dependência" : "sem uso")
                            color: linhaDoPlano.modelData.pedido ? Theme.textDim : Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                        Text {
                            anchors.right: parent.right
                            anchors.rightMargin: 12
                            width: 80
                            horizontalAlignment: Text.AlignRight
                            anchors.verticalCenter: parent.verticalCenter
                            text: Pacotes.tamanho(folha.instalando ? linhaDoPlano.modelData.baixar_bytes : linhaDoPlano.modelData.tamanho_bytes)
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                    }
                }
            }
            Repeater {
                model: Acao.fase === "plano" && folha.plano !== null ? (folha.plano.comandos || []) : []
                Text {
                    required property var modelData
                    Layout.fillWidth: true
                    wrapMode: Text.WrapAnywhere
                    maximumLineCount: 3
                    elide: Text.ElideRight
                    text: "$ " + Pacotes.linhaDeComando(modelData)
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }

            // ── o registro ─────────────────────────────────────────────
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: Acao.fase === "agindo" || Acao.fase === "feito" || Acao.fase === "falhou"
                color: Theme.ground
                border.width: 1
                border.color: Theme.hairline

                ListView {
                    id: registro
                    anchors.fill: parent
                    anchors.margins: 10
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: Acao.linhas
                    onCountChanged: positionViewAtEnd()
                    delegate: Text {
                        required property var modelData
                        width: registro.width
                        text: modelData
                        color: modelData.indexOf("$ ") === 0 ? Theme.text : modelData.indexOf("error") === 0 || modelData.indexOf("erro") === 0 ? Theme.copper : Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                        wrapMode: Text.WrapAnywhere
                    }
                    Text {
                        anchors.centerIn: parent
                        visible: registro.count === 0
                        text: Acao.fase === "agindo" ? "esperando a senha…" : "sem linhas"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: 13
                        font.italic: true
                    }
                }
            }

            // O que não tem lista nem registro ocupa o meio.
            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true
                visible: Acao.fase === "planejando" || Acao.fase === "recusa" || (Acao.fase === "plano" && folha.itens.length === 0)
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: 10
                Text {
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                    text: Acao.fase === "agindo" ? "a senha é pedida pelo Mk; não feche a janela"
                        : Acao.fase === "feito" ? "pronto"
                        : Acao.fase === "plano" ? "nada muda antes de você confirmar" : ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
                Botao {
                    visible: Acao.fase === "recusa" && Acao.recusado && Acao.acao === "remover" && !Acao.mesmoAssim
                    texto: "remover mesmo assim"
                    onAcionado: Acao.insistir()
                }
                Botao {
                    texto: Acao.fase === "plano" || Acao.fase === "planejando" ? "cancelar" : "fechar"
                    ligado: Acao.fase !== "agindo"
                    onAcionado: Acao.fechar()
                }
                Botao {
                    visible: Acao.fase === "plano"
                    texto: folha.plano === null ? "confirmar"
                        : folha.plano.acao === "instalar" ? "instalar"
                        : folha.plano.acao === "atualizar" ? "abrir a atualização"
                        : folha.plano.acao === "cache" ? "limpar" : "remover"
                    forte: true
                    ligado: Pacotes.podeConfirmar(folha.plano)
                    onAcionado: Acao.confirmar()
                }
            }
        }
    }
}
