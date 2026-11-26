import QtQuick
import "Arquivos.js" as Arquivos
import "Abas.js" as Abas
import "Remoto.js" as Remoto

// A barra lateral: os lugares fixos, os favoritos (arrastáveis para
// reordenar), os discos (com a barra de uso e o botão de ejetar), a rede (os
// servidores montados, com o desconectar, e a entrada de conectar) e a lixeira.
// Recolhida, mostra só os ícones. Soltar arquivos num lugar copia ou move
// para lá.
Rectangle {
    id: lateral

    property var lugares: []
    property var favoritos: []
    property var discos: []
    // Os remotos montados (do pedido `remotos`): { uri, raiz, nome, protocolo, host }.
    property var remotos: []
    property int naLixeira: 0
    property string atual: ""
    property bool recolhida: false
    // A linha em que o teclado está (Ctrl+B leva o foco para cá).
    property int cursor: -1
    property bool comTeclado: false

    signal aberto(var linha)
    signal menuPedido(var linha, real x, real y)
    signal favoritoMovido(int de, int para)
    signal ejetado(var linha)
    signal soltado(var linha, var caminhos, int acao)

    // Tudo numa lista só: { tipo, nome, caminho, icone, … } e os títulos.
    readonly property var linhas: {
        const l = [];
        for (const g of lugares)
            if (g.id !== "lixeira")
                l.push({ tipo: "lugar", nome: g.nome, caminho: g.caminho, icone: g.icone || "folder" });
        if (favoritos.length > 0)
            l.push({ tipo: "titulo", nome: "favoritos" });
        favoritos.forEach((f, i) => l.push({ tipo: "favorito", nome: f.nome, caminho: f.caminho, uri: f.uri, remoto: f.remoto === true,
            existe: f.existe !== false, icone: f.remoto ? "folder-remote" : "folder", posicao: i, primeiro: i === 0, ultimo: i === favoritos.length - 1 }));
        if (discos.length > 0)
            l.push({ tipo: "titulo", nome: "discos" });
        for (const d of discos)
            l.push({ tipo: "disco", nome: d.nome, caminho: d.montado || null, montado: d.montado || null, dispositivo: d.dispositivo,
                removivel: d.removivel === true, livre: d.livre, total: d.total, icone: d.removivel ? "drive-removable-media" : "drive-harddisk" });
        l.push({ tipo: "titulo", nome: "rede" });
        for (const r of Remoto.linhas(remotos))
            l.push(r);
        l.push({ tipo: "titulo", nome: "" });
        l.push({ tipo: "lixeira", nome: "Lixeira", caminho: Abas.LIXEIRA, icone: naLixeira > 0 ? "user-trash-full" : "user-trash", itens: naLixeira });
        return l;
    }

    function andar(delta) {
        let i = cursor;
        for (let k = 0; k < linhas.length; k++) {
            i = ((i + delta) % linhas.length + linhas.length) % linhas.length;
            if (linhas[i].tipo !== "titulo")
                break;
        }
        cursor = i;
        lista.positionViewAtIndex(i, ListView.Contain);
    }
    function confirmar() {
        if (cursor >= 0 && cursor < linhas.length)
            aberto(linhas[cursor]);
    }

    color: Theme.surface

    ListView {
        id: lista
        anchors.fill: parent
        anchors.topMargin: 6
        anchors.bottomMargin: 6
        clip: true
        model: lateral.linhas
        boundsBehavior: Flickable.StopAtBounds

        delegate: Item {
            id: linha
            required property var modelData
            required property int index
            readonly property bool titulo: modelData.tipo === "titulo"
            readonly property bool aqui: !titulo && modelData.caminho !== null
                && (modelData.caminho === lateral.atual || (modelData.tipo === "remoto" && Remoto.dentroDe(modelData.caminho, lateral.atual)))
            width: lista.width
            height: titulo ? (lateral.recolhida ? 9 : 26) : (modelData.tipo === "disco" && modelData.total && !lateral.recolhida ? 38 : 28)

            Rotulo {
                visible: linha.titulo && !lateral.recolhida
                x: 12
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 4
                text: linha.modelData.nome
            }
            Rectangle {
                visible: linha.titulo && lateral.recolhida
                anchors.centerIn: parent
                width: parent.width - 16
                height: 1
                color: Theme.hairline
            }
            Rectangle {
                anchors.fill: parent
                anchors.leftMargin: 4
                anchors.rightMargin: 4
                visible: !linha.titulo && (linha.aqui || area.containsMouse || alvo.containsDrag || (lateral.comTeclado && linha.index === lateral.cursor))
                color: Theme.wash
                border.width: alvo.containsDrag || (lateral.comTeclado && linha.index === lateral.cursor) ? 1 : 0
                border.color: Theme.copper
            }
            Rectangle {
                visible: linha.aqui
                x: 4
                width: 2
                height: parent.height
                color: Theme.copper
            }
            Icone {
                id: figura
                visible: !linha.titulo
                x: lateral.recolhida ? (parent.width - width) / 2 : 12
                y: 6
                lado: 16
                nome: linha.titulo ? "" : linha.modelData.icone
                reserva: "folder"
                opacity: linha.modelData.existe === false || (linha.modelData.tipo === "disco" && !linha.modelData.montado) ? 0.45 : 1
            }
            Text {
                visible: !linha.titulo && !lateral.recolhida
                x: 36
                y: 6
                width: parent.width - 36 - (ejetar.visible ? 30 : 8)
                elide: Text.ElideRight
                text: linha.modelData.nome + (linha.modelData.tipo === "lixeira" && linha.modelData.itens > 0 ? "  (" + linha.modelData.itens + ")" : "")
                color: linha.aqui ? Theme.copper : linha.modelData.existe === false ? Theme.textFaint : Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
            // A barra de uso de um disco montado.
            Rectangle {
                visible: linha.modelData.tipo === "disco" && !!linha.modelData.total && !lateral.recolhida
                x: 36
                y: 27
                width: parent.width - 48
                height: 3
                color: Theme.hairline
                Rectangle {
                    width: linha.modelData.total ? parent.width * Math.max(0, Math.min(1, 1 - linha.modelData.livre / linha.modelData.total)) : 0
                    height: parent.height
                    color: linha.modelData.total && linha.modelData.livre / linha.modelData.total < 0.1 ? Theme.copper : Theme.textDim
                }
            }

            MouseArea {
                id: area
                anchors.fill: parent
                enabled: !linha.titulo
                hoverEnabled: true
                acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton
                cursorShape: Qt.PointingHandCursor
                // Arrastar um favorito para cima ou para baixo o muda de lugar.
                property real inicio: 0
                property bool arrastou: false
                onPressed: mouse => {
                    inicio = mouse.y;
                    arrastou = false;
                }
                onPositionChanged: mouse => {
                    if (pressed && linha.modelData.tipo === "favorito" && Math.abs(mouse.y - inicio) > 10)
                        arrastou = true;
                }
                onReleased: mouse => {
                    if (!arrastou)
                        return;
                    const passos = Math.round((mouse.y - inicio) / 28);
                    const para = Math.max(0, Math.min(lateral.favoritos.length - 1, linha.modelData.posicao + passos));
                    if (para !== linha.modelData.posicao)
                        lateral.favoritoMovido(linha.modelData.posicao, para);
                }
                onClicked: mouse => {
                    if (arrastou)
                        return;
                    if (mouse.button === Qt.RightButton) {
                        const p = linha.mapToItem(null, mouse.x, mouse.y);
                        lateral.menuPedido(linha.modelData, p.x, p.y);
                    } else {
                        lateral.aberto(Object.assign({ novaAba: mouse.button === Qt.MiddleButton || (mouse.modifiers & Qt.ControlModifier) !== 0 }, linha.modelData));
                    }
                }
            }
            // Ejetar (ou desmontar) um disco montado; desconectar um remoto.
            Text {
                id: ejetar
                visible: (linha.modelData.tipo === "disco" || linha.modelData.tipo === "remoto") && !!linha.modelData.montado && !lateral.recolhida
                anchors.right: parent.right
                anchors.rightMargin: 10
                y: 5
                text: "⏏"
                color: ejetarArea.containsMouse ? Theme.copper : Theme.textDim
                font.pixelSize: 13
                MouseArea {
                    id: ejetarArea
                    anchors.fill: parent
                    anchors.margins: -5
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: lateral.ejetado(linha.modelData)
                }
            }
            DropArea {
                id: alvo
                anchors.fill: parent
                enabled: !linha.titulo && linha.modelData.caminho !== null && linha.modelData.remoto !== true
                keys: ["text/uri-list"]
                onDropped: drop => {
                    const caminhos = Arquivos.lerListaDeUris(drop.hasUrls ? drop.urls.join("\n") : drop.getDataAsString("text/uri-list"));
                    if (caminhos.length === 0)
                        return;
                    drop.accept(drop.proposedAction);
                    lateral.soltado(linha.modelData, caminhos, drop.proposedAction);
                }
            }
        }
    }
}
