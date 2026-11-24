import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// Histórico: o registro do pacman resumido ao que mudou nos pacotes, do mais
// novo para o mais velho, com busca pelo nome.
Item {
    id: pagina

    property string busca: ""
    property string acao: "todas"
    property var eventos: []
    property int total: 0
    property string erro: ""
    property bool leu: false

    readonly property var visiveis: acao === "todas" ? eventos : eventos.filter(e => e.acao === acao)

    function ler() {
        const q = busca.trim();
        Coletor.pedir(["historico", "--linhas", "1000"].concat(q !== "" ? ["--busca", q] : []), r => {
            if (pagina.busca.trim() !== q)
                return;
            pagina.leu = true;
            pagina.erro = r.erro || "";
            pagina.eventos = r.erro ? [] : (r.eventos || []);
            pagina.total = r.erro ? 0 : (r.total || 0);
        });
    }
    Component.onCompleted: ler()
    Timer { id: demora; interval: 350; onTriggered: pagina.ler() }
    Connections {
        target: Acao
        function onTerminou() { pagina.ler(); }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Cabecalho {
                numero: "04"
                titulo: "Histórico"
                frase: "O que entrou, mudou e saiu desta máquina, pelo registro do pacman."
            }
            Item { Layout.fillWidth: true }
            Campo { Layout.alignment: Qt.AlignBottom; Layout.preferredWidth: 200; dica: "procurar pelo nome"; onTextChanged: { pagina.busca = text; demora.restart(); } }
        }

        Segmento {
            opcoes: [{ v: "todas", texto: "tudo" }, { v: "instalado", texto: "instalado" }, { v: "atualizado", texto: "atualizado" },
                { v: "removido", texto: "removido" }]
            valor: pagina.acao
            onEscolhido: v => pagina.acao = v
        }

        Text {
            Layout.fillWidth: true
            visible: pagina.erro !== ""
            text: pagina.erro
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
            wrapMode: Text.WordWrap
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            ListView {
                id: tabela
                anchors.fill: parent
                anchors.margins: 1
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.visiveis

                delegate: Item {
                    id: linha
                    required property var modelData
                    width: tabela.width
                    height: 30
                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Text {
                        x: 14
                        width: 120
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.t_s ? Qt.formatDateTime(new Date(linha.modelData.t_s * 1000), "dd/MM/yyyy HH:mm") : "—"
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                    Text {
                        x: 144
                        width: 90
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.acao
                        color: linha.modelData.acao === "removido" || linha.modelData.acao === "rebaixado" ? Theme.copper
                            : linha.modelData.acao === "instalado" ? Theme.text : Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                    Text {
                        x: 240
                        width: Math.max(100, parent.width - 240 - 300)
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.nome
                        color: Theme.text
                        font.family: Theme.mono
                        font.pixelSize: 11
                        elide: Text.ElideRight
                    }
                    Text {
                        anchors.right: parent.right
                        anchors.rightMargin: 14
                        width: 270
                        horizontalAlignment: Text.AlignRight
                        anchors.verticalCenter: parent.verticalCenter
                        text: Pacotes.versoes(linha.modelData)
                        color: Theme.textDim
                        font.family: Theme.mono
                        font.pixelSize: 10
                        elide: Text.ElideMiddle
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: tabela.count === 0 && pagina.erro === ""
                    text: !pagina.leu ? "lendo o registro…" : "nada no registro com esse nome"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }

        Text {
            Layout.fillWidth: true
            elide: Text.ElideRight
            text: Pacotes.plural(pagina.visiveis.length, "linha", "linhas")
                + (pagina.total > pagina.eventos.length ? "  ·  as " + pagina.eventos.length + " mais novas de " + pagina.total + ": procure pelo nome para ver as antigas" : "")
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }
    }
}
