import QtQuick
import "Porto.js" as Porto

// O detalhe do host escolhido: os campos, de onde ele vem, se a porta
// responde e as ações. Editar e remover só valem para o que o MkPorto guardou;
// o que é do ~/.ssh/config da pessoa só se lê e se conecta.
Item {
    id: detalhe

    property var host: null
    property var sondas: ({})
    // Numa janela estreita o detalhe ocupa o lugar da lista: aparece o "voltar".
    property bool estreita: false
    // O teste de entrada em curso ou feito (Porto.textoDaEntrada); null se nenhum.
    property var entrada: null
    readonly property var sonda: host !== null && sondas ? (sondas[host.apelido] || null) : null
    readonly property string textoDaEntrada: Porto.textoDaEntrada(entrada, host)
    readonly property bool doApp: Porto.editavel(host)
    signal voltar()
    signal conectar()
    signal copiar()
    signal testarEntrada()
    signal enviarChave()
    signal editar()
    signal remover()

    Text {
        anchors.centerIn: parent
        visible: detalhe.host === null
        text: "escolha um host na lista"
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 15
        font.italic: true
    }

    Flickable {
        id: rolo
        anchors.fill: parent
        visible: detalhe.host !== null
        clip: true
        contentHeight: folha.implicitHeight
        boundsBehavior: Flickable.StopAtBounds

        Column {
            id: folha
            width: rolo.width
            spacing: 14

            Botao {
                visible: detalhe.estreita
                texto: "‹ voltar"
                onAcionado: detalhe.voltar()
            }

            Column {
                width: parent.width
                spacing: 6

                Rotulo { text: "host  /  " + Porto.origem(detalhe.host) }
                Text {
                    width: parent.width
                    elide: Text.ElideRight
                    text: detalhe.host ? detalhe.host.apelido : ""
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 26
                    font.weight: Font.Light
                }
                Row {
                    spacing: 8
                    Bolinha {
                        anchors.verticalCenter: parent.verticalCenter
                        estado: Porto.estado(detalhe.host, detalhe.sondas)
                    }
                    Text {
                        text: Porto.textoDoEstado(detalhe.host, detalhe.sondas)
                        color: Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 12
                    }
                }
                // O resultado do "testar entrada": some ao trocar de host.
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: detalhe.textoDaEntrada !== ""
                    text: detalhe.textoDaEntrada
                    color: detalhe.entrada !== null && detalhe.entrada.entra === true ? Theme.copper : Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
            }

            Rectangle { width: parent.width; height: 1; color: Theme.hairline }

            Column {
                width: parent.width
                spacing: 8

                Repeater {
                    model: detalhe.host === null ? [] : [
                        { k: "endereço", v: detalhe.host.host },
                        { k: "usuário", v: detalhe.host.usuario || "—" },
                        { k: "porta", v: String(detalhe.host.porta) },
                        { k: "chave", v: detalhe.host.chave || "—" },
                        { k: "salto", v: detalhe.host.salto || "—" },
                        { k: "servidor", v: (detalhe.sonda !== null && detalhe.sonda.servidor) || "—" },
                        { k: "valores", v: Porto.fonteDosValores(detalhe.host) },
                        { k: "vem de", v: Porto.origem(detalhe.host) },
                        { k: "arquivo", v: detalhe.host.arquivo }
                    ]

                    Item {
                        id: campo
                        required property var modelData
                        width: folha.width
                        height: Math.max(16, valor.implicitHeight)

                        Rotulo {
                            width: 84
                            y: 3
                            text: campo.modelData.k
                        }
                        Text {
                            id: valor
                            x: 92
                            width: parent.width - 92
                            wrapMode: Text.WrapAnywhere
                            text: campo.modelData.v
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 12
                        }
                    }
                }
            }

            Rectangle { width: parent.width; height: 1; color: Theme.hairline }

            Flow {
                width: parent.width
                spacing: 8

                Botao {
                    texto: "conectar"
                    forte: true
                    onAcionado: detalhe.conectar()
                }
                Botao {
                    texto: "copiar comando"
                    onAcionado: detalhe.copiar()
                }
                Botao {
                    texto: "testar entrada"
                    ligado: detalhe.entrada === null || detalhe.entrada.testando !== true
                    onAcionado: detalhe.testarEntrada()
                }
                Botao {
                    texto: "enviar chave"
                    onAcionado: detalhe.enviarChave()
                }
                Botao {
                    texto: "editar"
                    ligado: detalhe.doApp
                    onAcionado: detalhe.editar()
                }
                Botao {
                    texto: "remover"
                    ligado: detalhe.doApp
                    onAcionado: detalhe.remover()
                }
            }

            Text {
                width: parent.width
                wrapMode: Text.WordWrap
                text: detalhe.doApp
                    ? "Conectar abre um terminal com o ssh; quem autentica é ele, com o seu agente de chaves. O MkPorto não guarda nem pede senha."
                    : "Este host é do seu ~/.ssh/config: o MkPorto lê e conecta, mas não edita nem remove o que não é dele."
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 11
            }
        }
    }
}
