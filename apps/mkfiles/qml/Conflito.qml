import QtQuick
import "Arquivos.js" as Arquivos

// O conflito de uma cópia ou de um mover: já existe algo com esse nome no
// destino. Os dois lado a lado (tamanho e data), as respostas do contrato e
// "aplicar a todos". S substitui, P pula, M mantém os dois, R renomeia.
Folha {
    id: conflito

    // O evento "conflito" do motor: { tarefa, conflito, acao, origem, destino, sugestao, restam }.
    property var dados: null
    property bool todos: false
    property bool renomeando: false
    signal respondido(string resposta, string nome, bool todos)

    readonly property var origem: dados ? dados.origem : null
    readonly property var destino: dados ? dados.destino : null

    function abrir() {
        todos = false;
        renomeando = false;
        campo.text = dados ? dados.sugestao || "" : "";
    }
    function responder(resposta) {
        if (resposta === "renomear") {
            if (!renomeando) {
                renomeando = true;
                campo.focar();
                return;
            }
            if (Arquivos.erroDoNome(campo.text.trim()) !== "")
                return;
            respondido("renomear", campo.text.trim(), false);
            return;
        }
        respondido(resposta, "", todos);
    }

    rotulo: "conflito"
    titulo: destino ? "Já existe «" + destino.nome + "»" : ""
    largura: 560

    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        text: conflito.destino ? "em " + Arquivos.pai(conflito.destino.caminho)
            + (conflito.dados && conflito.dados.restam > 0 ? "  ·  mais " + Arquivos.plural(conflito.dados.restam, "item", "itens") + " na fila" : "") : ""
        color: Theme.textDim
        font.family: Theme.sans
        font.pixelSize: 12
    }

    Row {
        width: parent.width
        spacing: 12

        Repeater {
            model: [{ rotulo: "o que chega", lado: "origem" }, { rotulo: "o que está lá", lado: "destino" }]

            Rectangle {
                id: parte
                required property var modelData
                readonly property var item: modelData.lado === "origem" ? conflito.origem : conflito.destino
                width: (parent.width - 12) / 2
                height: ficha.implicitHeight + 20
                color: Theme.raised
                border.width: 1
                border.color: Theme.hairline

                Column {
                    id: ficha
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.margins: 10
                    spacing: 4

                    Rotulo { text: parte.modelData.rotulo }
                    Row {
                        spacing: 8
                        Icone {
                            lado: 32
                            nome: parte.item ? parte.item.icone : ""
                            reserva: parte.item ? parte.item.icone_reserva : "text-x-generic"
                        }
                        Column {
                            width: ficha.width - 40
                            Text {
                                width: parent.width
                                elide: Text.ElideMiddle
                                text: parte.item ? parte.item.nome : ""
                                color: Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                            }
                            Text {
                                text: parte.item ? Arquivos.tipo(parte.item) : ""
                                color: Theme.textDim
                                font.family: Theme.sans
                                font.pixelSize: 11
                            }
                        }
                    }
                    Text {
                        text: parte.item ? (Arquivos.ePasta(parte.item) ? "pasta" : Arquivos.tamanho(parte.item.tamanho)) : ""
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                    Text {
                        text: parte.item ? Arquivos.data(parte.item.mtime) : ""
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                    Text {
                        visible: !!conflito.origem && !!conflito.destino && parte.modelData.lado === "origem"
                        text: !conflito.origem || !conflito.destino ? ""
                            : conflito.origem.mtime > conflito.destino.mtime ? "mais novo"
                            : conflito.origem.mtime < conflito.destino.mtime ? "mais antigo" : "da mesma data"
                        color: Theme.copper
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                }
            }
        }
    }

    Campo {
        id: campo
        width: parent.width
        visible: conflito.renomeando
        dica: "o nome novo"
        onAceito: conflito.responder("renomear")
    }

    Flow {
        width: parent.width
        spacing: 8
        Botao {
            texto: Arquivos.ePasta(conflito.origem) && Arquivos.ePasta(conflito.destino) ? "mesclar" : "substituir"
            forte: true
            onAcionado: conflito.responder("substituir")
        }
        Botao {
            texto: "pular"
            onAcionado: conflito.responder("pular")
        }
        Botao {
            texto: "manter os dois"
            onAcionado: conflito.responder("manter")
        }
        Botao {
            texto: conflito.renomeando ? "gravar com este nome" : "renomear…"
            forte: conflito.renomeando
            onAcionado: conflito.responder("renomear")
        }
        Botao {
            texto: "cancelar tudo"
            onAcionado: conflito.responder("cancelar")
        }
    }
    Row {
        spacing: 10
        Chave {
            ligado: conflito.todos
            onTrocado: conflito.todos = !conflito.todos
        }
        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: "aplicar a todos os conflitos desta tarefa"
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
        }
    }
    Text {
        visible: !!conflito.dados && conflito.dados.sugestao !== undefined
        text: conflito.dados ? "manter os dois grava como «" + conflito.dados.sugestao + "»" : ""
        color: Theme.textFaint
        font.family: Theme.sans
        font.pixelSize: 11
    }
    Rotulo { text: "s substitui  ·  p pula  ·  m mantém os dois  ·  r renomeia  ·  t todos  ·  esc cancela a tarefa" }
}
