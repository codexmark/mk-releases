import QtQuick
import "Arquivos.js" as Arquivos

// A área de tarefas: as operações em andamento (com o progresso, pausar e
// cancelar) e as que acabaram com erro, que ficam até serem lidas. Some
// quando não há nenhuma.
Rectangle {
    id: tarefas

    // Os últimos eventos "tarefa" do motor, um por tarefa.
    property var lista: []
    signal pausar(int tarefa, bool pausar)
    signal cancelar(int tarefa)
    signal lida(int tarefa)
    signal detalhes(var tarefa)

    visible: lista.length > 0
    implicitHeight: visible ? Math.min(3, lista.length) * 44 + 1 : 0
    color: Theme.surface

    Rectangle {
        width: parent.width
        height: 1
        color: Theme.hairline
    }

    ListView {
        anchors.fill: parent
        anchors.topMargin: 1
        clip: true
        model: tarefas.lista
        boundsBehavior: Flickable.StopAtBounds

        delegate: Item {
            id: linha
            required property var modelData
            readonly property bool acabou: Arquivos.tarefaAcabou(modelData)
            readonly property bool comErro: modelData.estado === "erro"
            width: ListView.view.width
            height: 44

            Text {
                id: nome
                x: 12
                y: 6
                width: parent.width - botoes.width - 30
                elide: Text.ElideMiddle
                text: linha.modelData.titulo || linha.modelData.acao
                color: linha.comErro ? Theme.copper : Theme.text
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Text {
                x: 12
                y: 23
                width: nome.width
                elide: Text.ElideMiddle
                text: Arquivos.detalheDaTarefa(linha.modelData)
                    + (linha.modelData.estado === "rodando" && linha.modelData.atual ? "  ·  " + Arquivos.nomeDe(linha.modelData.atual) : "")
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 10
            }
            // A barra de progresso, ao pé da linha.
            Rectangle {
                anchors.bottom: parent.bottom
                x: 12
                width: parent.width - 24
                height: 2
                color: Theme.hairline
                visible: !linha.acabou
                Rectangle {
                    width: parent.width * Arquivos.progresso(linha.modelData)
                    height: parent.height
                    color: linha.modelData.estado === "pausada" ? Theme.textDim : Theme.copper
                }
            }
            Row {
                id: botoes
                anchors.right: parent.right
                anchors.rightMargin: 12
                anchors.verticalCenter: parent.verticalCenter
                spacing: 6
                Botao {
                    visible: linha.comErro && (linha.modelData.erros || []).length > 1
                    texto: "ver erros"
                    onAcionado: tarefas.detalhes(linha.modelData)
                }
                Botao {
                    visible: !linha.acabou && (linha.modelData.estado === "rodando" || linha.modelData.estado === "pausada")
                    texto: linha.modelData.estado === "pausada" ? "continuar" : "pausar"
                    onAcionado: tarefas.pausar(linha.modelData.tarefa, linha.modelData.estado !== "pausada")
                }
                Botao {
                    visible: !linha.acabou
                    texto: "cancelar"
                    onAcionado: tarefas.cancelar(linha.modelData.tarefa)
                }
                Botao {
                    visible: linha.acabou
                    texto: "ok"
                    forte: true
                    onAcionado: tarefas.lida(linha.modelData.tarefa)
                }
            }
        }
    }
}
