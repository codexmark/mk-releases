import QtQuick
import QtQuick.Layouts
import "Pacotes.js" as Pacotes

// Procurar: nos repositórios do Arch e no AUR, numa lista só. Cada resultado
// diz de onde é e se já está instalado; sem rede, os do Arch continuam.
Item {
    id: pagina

    property string texto: ""
    property string pedido: ""
    property var resultados: []
    property bool cortado: false
    property bool procurando: false
    property bool procurou: false
    property string erro: ""
    property string erroAur: ""
    property string erroOficial: ""
    property bool comAur: true
    property string escolhido: ""
    property var marcas: ({})

    readonly property var marcados: Pacotes.marcados(marcas)

    function procurar() {
        const q = texto.trim();
        if (q.length < 2)
            return;
        pedido = q;
        procurando = true;
        Coletor.pedir(["procurar"].concat(comAur ? [] : ["--sem-aur"], ["--"], q.split(/\s+/)), r => {
            if (pagina.pedido !== q)
                return;
            pagina.procurando = false;
            pagina.procurou = true;
            pagina.erro = r.erro || "";
            pagina.resultados = r.erro ? [] : (r.resultados || []);
            pagina.cortado = !r.erro && r.cortado === true;
            pagina.erroAur = r.erro_aur || "";
            pagina.erroOficial = r.erro_oficial || "";
        });
    }
    onComAurChanged: procurar()

    // Quem digita espera um instante; Enter procura na hora.
    Timer { id: demora; interval: 450; onTriggered: pagina.procurar() }
    // Depois de instalar ou remover, a lista diz a verdade de novo.
    Connections {
        target: Acao
        function onTerminou() { pagina.marcas = ({}); pagina.procurar(); }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Cabecalho {
                numero: "02"
                titulo: "Procurar"
                frase: "Nos repositórios do Arch e no AUR, numa lista só."
            }
            Item { Layout.fillWidth: true }
            Segmento {
                Layout.alignment: Qt.AlignBottom
                opcoes: [{ v: "sim", texto: "arch + aur" }, { v: "nao", texto: "só arch" }]
                valor: pagina.comAur ? "sim" : "nao"
                onEscolhido: v => pagina.comAur = v === "sim"
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Campo {
                id: campo
                Layout.fillWidth: true
                dica: "nome ou descrição: duas letras ou mais"
                onTextChanged: { pagina.texto = text; demora.restart(); }
                onAceito: { demora.stop(); pagina.procurar(); }
                Component.onCompleted: campo.focar()
            }
            Botao { texto: "procurar"; forte: true; ligado: pagina.texto.trim().length >= 2; onAcionado: { demora.stop(); pagina.procurar(); } }
        }

        Text {
            Layout.fillWidth: true
            visible: text !== ""
            wrapMode: Text.WordWrap
            text: pagina.erro !== "" ? pagina.erro
                : pagina.erroAur !== "" ? "AUR: " + pagina.erroAur + ". Os resultados abaixo são só dos repositórios do Arch."
                : pagina.erroOficial !== "" ? "Arch: " + pagina.erroOficial + ". Os resultados abaixo são só do AUR."
                : pagina.cortado ? "há mais resultados do que cabem aqui: escreva mais para apertar a busca" : ""
            color: pagina.cortado && pagina.erro === "" && pagina.erroAur === "" && pagina.erroOficial === "" ? Theme.textDim : Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 14

            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                ListView {
                    id: lista
                    anchors.fill: parent
                    anchors.margins: 1
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: pagina.resultados
                    readonly property bool comDescricao: width >= 700

                    delegate: Item {
                        id: linha
                        required property var modelData
                        readonly property bool atual: pagina.escolhido === modelData.nome
                        width: lista.width
                        height: 34

                        Rectangle {
                            anchors.fill: parent
                            color: linha.atual ? Theme.copperA(0.1) : passa.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)
                        }
                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                        Rectangle { visible: linha.atual; width: 2; height: parent.height; color: Theme.copper }
                        MouseArea {
                            id: passa
                            anchors.fill: parent
                            hoverEnabled: true
                            onClicked: pagina.escolhido = linha.atual ? "" : linha.modelData.nome
                        }
                        Marca {
                            x: 13
                            anchors.verticalCenter: parent.verticalCenter
                            marcado: pagina.marcas[linha.modelData.nome] === true
                            ligado: !linha.modelData.instalado && Pacotes.nomeValido(linha.modelData.nome)
                            onTrocado: pagina.marcas = Pacotes.alterna(pagina.marcas, linha.modelData.nome)
                        }
                        Text {
                            id: oNome
                            x: 40
                            width: Math.min(240, parent.width - 40 - direita.width - 20)
                            anchors.verticalCenter: parent.verticalCenter
                            text: linha.modelData.nome
                            color: Theme.text
                            font.family: Theme.mono
                            font.pixelSize: 11
                            elide: Text.ElideRight
                        }
                        Text {
                            visible: lista.comDescricao
                            anchors.left: oNome.right
                            anchors.leftMargin: 14
                            anchors.right: direita.left
                            anchors.rightMargin: 14
                            anchors.verticalCenter: parent.verticalCenter
                            text: linha.modelData.descricao
                            color: Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 12
                            elide: Text.ElideRight
                        }
                        Row {
                            id: direita
                            anchors.right: parent.right
                            anchors.rightMargin: 14
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 0
                            Text {
                                width: 130
                                horizontalAlignment: Text.AlignRight
                                text: linha.modelData.versao
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 10
                                elide: Text.ElideMiddle
                            }
                            Text {
                                width: 110
                                horizontalAlignment: Text.AlignRight
                                text: linha.modelData.origem === "aur"
                                    ? "AUR  " + (linha.modelData.desatualizado ? "velho" : Pacotes.plural(linha.modelData.votos || 0, "voto", "votos"))
                                    : linha.modelData.origem
                                color: linha.modelData.origem === "aur" ? Theme.copper : Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                                elide: Text.ElideRight
                            }
                            Text {
                                width: 84
                                horizontalAlignment: Text.AlignRight
                                text: !linha.modelData.instalado ? ""
                                    : linha.modelData.versao_instalada !== linha.modelData.versao ? "tem outra" : "instalado"
                                color: Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 10
                            }
                        }
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: lista.count === 0
                        text: pagina.procurando ? "procurando…"
                            : !pagina.procurou ? "escreva o que procura"
                            : pagina.erro !== "" ? "a busca falhou" : "nada com esse nome nem com essa descrição"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }
                }
            }

            Detalhe {
                Layout.preferredWidth: 340
                Layout.fillHeight: true
                visible: pagina.escolhido !== ""
                nome: pagina.escolhido
                onAbrir: nome => pagina.escolhido = nome
                onFechar: pagina.escolhido = ""
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 10
            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: pagina.marcados.length > 0 ? Pacotes.plural(pagina.marcados.length, "marcado", "marcados") + ": " + pagina.marcados.join(", ")
                    : pagina.procurou ? Pacotes.plural(pagina.resultados.length, "resultado", "resultados") + (pagina.procurando ? "  ·  procurando…" : "")
                    : "clique num resultado para ver a ficha e instalar"
                color: Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
            Botao {
                visible: pagina.marcados.length > 0
                texto: "desmarcar"
                onAcionado: pagina.marcas = ({})
            }
            Botao {
                visible: pagina.marcados.length > 0
                texto: pagina.marcados.length > 1 ? "instalar os " + pagina.marcados.length + " marcados" : "instalar o marcado"
                forte: true
                onAcionado: Acao.planejar("instalar", pagina.marcados, false)
            }
        }
    }
}
