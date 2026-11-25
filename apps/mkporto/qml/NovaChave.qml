import QtQuick
import "Porto.js" as Porto

// A folha da chave nova: o nome do arquivo e o comentário. Quem gera é o
// ssh-keygen, num terminal; é lá que ele pergunta o que quiser proteger a
// chave, e o MkPorto não vê a resposta. Aqui não há campo para isso.
Item {
    id: nova

    // As chaves públicas que já existem (para não repetir o nome).
    property var chaves: []
    property string erro: ""
    property bool ocupado: false
    // Só reclama do nome vazio depois que a pessoa tentou.
    property bool tentou: false
    signal gerar(var form)
    signal fechado()

    readonly property var form: ({ nome: nome.text, comentario: comentario.text })
    readonly property var erros: Porto.validarChave(form, chaves)
    readonly property bool bom: Porto.chaveValida(erros)

    function abrir() {
        tentou = false;
        nome.text = "";
        comentario.text = "";
        nome.focar();
    }

    function tenta() {
        tentou = true;
        if (bom && !ocupado)
            gerar(form);
    }

    Fundo {
        anchors.fill: parent
        onFechado: nova.fechado()
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(480, parent.width - 32)
        height: Math.min(parent.height - 24, corpo.implicitHeight + 40)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        // Um clique no cartão não fecha a folha.
        MouseArea { anchors.fill: parent }

        Flickable {
            id: rolo
            anchors.fill: parent
            anchors.margins: 20
            contentHeight: corpo.implicitHeight
            boundsBehavior: Flickable.StopAtBounds
            clip: true

            Column {
                id: corpo
                width: rolo.width
                spacing: 8

                Rotulo { text: "nova chave  /  ed25519" }
                Text {
                    text: "Uma chave nova"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                    bottomPadding: 4
                }
                Entrada {
                    id: nome
                    width: parent.width
                    rotulo: "nome"
                    dica: "id_ed25519_trabalho (vira um arquivo em ~/.ssh)"
                    mono: true
                    erro: text === "" && !nova.tentou ? "" : nova.erros.nome
                    onAceito: nova.tenta()
                }
                Entrada {
                    id: comentario
                    width: parent.width
                    rotulo: "comentário"
                    dica: "quem é a chave; vazio vira usuario@maquina"
                    erro: nova.erros.comentario
                    onAceito: nova.tenta()
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    text: "Abre um terminal com o ssh-keygen. O que ele perguntar é respondido lá, só para ele. Quando terminar, volte e use \"atualizar\"."
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: nova.erro !== ""
                    text: nova.erro
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Row {
                    spacing: 8
                    Botao {
                        texto: "gerar num terminal"
                        forte: true
                        ligado: !nova.ocupado && (nova.bom || !nova.tentou)
                        onAcionado: nova.tenta()
                    }
                    Botao {
                        texto: "cancelar"
                        onAcionado: nova.fechado()
                    }
                }
                Rotulo { text: "esc fecha" }
            }
        }
    }
}
