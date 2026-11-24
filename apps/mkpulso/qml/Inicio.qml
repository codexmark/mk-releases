import QtQuick
import QtQuick.Layouts
import "Pulso.js" as Pulso

// Inicialização: o que sobe quando você entra. Os atalhos de autostart e os
// serviços de usuário ligam e desligam aqui; o que é do próprio Mk aparece
// para você saber, mas não muda daqui.
Item {
    id: pagina

    property var itens: []
    property string erro: ""
    property string busca: ""
    readonly property var visiveis: itens.filter(i => {
        const q = Pulso.fold(busca);
        return q === "" || Pulso.fold(i.nome + " " + i.comando + " " + i.origem).indexOf(q) >= 0;
    })

    // Um serviço de usuário desligado some da resposta do coletor (ele lista
    // os habilitados): a linha fica guardada aqui, desligada, para dar para
    // ligar de novo sem sair da página.
    property var desligados: ({})

    function ler() {
        Coletor.pedir(["inicio"], r => {
            pagina.erro = r.erro || "";
            if (r.erro)
                return;
            const lista = r.itens || [];
            const vistos = {};
            for (const i of lista)
                vistos[i.id] = true;
            const guardados = Object.assign({}, pagina.desligados);
            for (const id in guardados) {
                if (vistos[id])
                    delete guardados[id];
                else
                    lista.push(Object.assign({}, guardados[id], { ligado: false }));
            }
            pagina.desligados = guardados;
            pagina.itens = lista;
        });
    }
    function trocar(item) {
        const desligando = item.ligado;
        Coletor.pedir(["inicio", desligando ? "desligar" : "ligar", item.id], r => {
            pagina.erro = r.erro || "";
            if (!r.erro && desligando && item.id.indexOf("unidade:") === 0) {
                const novo = Object.assign({}, pagina.desligados);
                novo[item.id] = item;
                pagina.desligados = novo;
            }
            pagina.ler();
        });
    }
    Component.onCompleted: ler()

    ColumnLayout {
        anchors.fill: parent
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            spacing: 16
            Cabecalho {
                numero: "04"
                titulo: "Inicialização"
                frase: "O que sobe sozinho quando você entra na sessão."
            }
            Item { Layout.fillWidth: true }
            Campo { Layout.alignment: Qt.AlignBottom; dica: "procurar"; onTextChanged: pagina.busca = text }
        }

        Text {
            visible: pagina.erro !== ""
            text: pagina.erro
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            ListView {
                anchors.fill: parent
                anchors.margins: 1
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.visiveis

                delegate: Item {
                    id: linha
                    required property var modelData
                    width: ListView.view.width
                    height: 46

                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Column {
                        x: 14
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width - 240
                        spacing: 2
                        Text {
                            width: parent.width
                            text: linha.modelData.nome
                            color: linha.modelData.ligado ? Theme.text : Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 13
                            elide: Text.ElideRight
                        }
                        Text {
                            width: parent.width
                            text: linha.modelData.comando
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideRight
                        }
                    }
                    Rotulo {
                        anchors.right: chave.left
                        anchors.rightMargin: 20
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.origem + (linha.modelData.do_sistema && linha.modelData.origem !== "Mk" ? "  ·  do sistema" : "")
                        color: linha.modelData.origem === "Mk" ? Theme.copper : Theme.textFaint
                    }
                    Chave {
                        id: chave
                        anchors.right: parent.right
                        anchors.rightMargin: 16
                        anchors.verticalCenter: parent.verticalCenter
                        ligado: linha.modelData.ligado
                        mudavel: linha.modelData.mudavel
                        onTrocado: pagina.trocar(linha.modelData)
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: parent.count === 0
                    text: pagina.itens.length === 0 ? "nada sobe sozinho além do sistema" : "nada casa com a busca"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }
    }
}
