import QtQuick
import "Arquivos.js" as Arquivos

// A prévia rápida (Espaço): uma folha grande com o item em foco. Imagem (com
// zoom), texto, áudio e vídeo, a primeira página de um PDF (pela miniatura
// grande), o que há numa pasta ou num compactado; do resto, o ícone e a
// ficha. As setas passam para o vizinho sem fechar; Enter abre no app padrão.
// O App pede ao motor (`previa`, `miniatura`, `propriedades`,
// `compactado-listar`) e põe aqui o que chegar.
Item {
    id: previa

    property var item: null
    // A resposta do `previa`: { classe, texto, cortado, binario, itens, total }.
    property var dados: null
    property bool lendo: false
    property string erro: ""
    // A miniatura x-large (PDF), a contagem (pasta) e a listagem (compactado).
    property string miniatura: ""
    property var contagem: null
    property bool contando: false
    property var listagem: []
    property string posicao: ""
    // 0 = ajustada à folha; senão, o fator (1 = tamanho real).
    property real escala: 0

    readonly property string classe: dados && dados.classe ? dados.classe : ""
    readonly property bool animada: item !== null && (item.mime === "image/gif" || item.mime === "image/webp" || item.mime === "image/apng")
    readonly property var lista: classe === "compactado" ? listagem : dados && dados.itens ? dados.itens : []
    readonly property bool daMidia: classe === "audio" || classe === "video"

    signal fechado()
    signal abrirPedido()
    signal vizinho(int delta)

    function reiniciar() {
        dados = null;
        erro = "";
        miniatura = "";
        contagem = null;
        contando = false;
        listagem = [];
        escala = 0;
    }
    function zoom(fator) {
        const atual = escala > 0 ? escala : figura.ajuste;
        escala = Math.max(0.05, Math.min(16, atual * fator));
    }

    Fundo {
        anchors.fill: parent
        onFechado: previa.fechado()
    }

    Rectangle {
        id: cartao
        anchors.fill: parent
        anchors.margins: Math.min(28, parent.width / 30)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        MouseArea {
            anchors.fill: parent
            acceptedButtons: Qt.AllButtons
            onWheel: wheel => wheel.accepted = true
        }

        // O alto: o que é, e os botões.
        Item {
            id: alto
            x: 18
            y: 12
            width: parent.width - 36
            height: 44

            Column {
                width: parent.width - botoes.width - 12
                spacing: 3
                Text {
                    width: parent.width
                    elide: Text.ElideMiddle
                    text: previa.item ? previa.item.nome : ""
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 19
                    font.weight: Font.Light
                }
                Text {
                    width: parent.width
                    elide: Text.ElideRight
                    text: !previa.item ? ""
                        : Arquivos.tipo(previa.item) + (Arquivos.ePasta(previa.item) ? "" : "  ·  " + Arquivos.tamanho(previa.item.tamanho))
                        + (previa.item.mtime ? "  ·  " + Arquivos.data(previa.item.mtime) : "") + (previa.posicao !== "" ? "  ·  " + previa.posicao : "")
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }
            Row {
                id: botoes
                anchors.right: parent.right
                spacing: 6
                Botao {
                    visible: previa.classe === "imagem" && cartao.width > 560
                    texto: "ajustar"
                    forte: previa.escala === 0
                    onAcionado: previa.escala = 0
                }
                Botao {
                    visible: previa.classe === "imagem" && cartao.width > 560
                    texto: "100%"
                    forte: previa.escala === 1
                    onAcionado: previa.escala = 1
                }
                Botao {
                    texto: "abrir"
                    forte: true
                    onAcionado: previa.abrirPedido()
                }
                Botao {
                    texto: "fechar"
                    onAcionado: previa.fechado()
                }
            }
        }
        Rectangle {
            y: 60
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        Item {
            id: corpo
            x: 1
            y: 61
            width: parent.width - 2
            height: parent.height - 61 - 26
            clip: true

            // --- imagem: ajustada, ou no fator que a roda escolher ---
            Flickable {
                id: quadro
                anchors.fill: parent
                visible: previa.classe === "imagem"
                contentWidth: Math.max(width, figura.width)
                contentHeight: Math.max(height, figura.height)
                boundsBehavior: Flickable.StopAtBounds
                clip: true

                Item {
                    id: figura
                    readonly property var fonte: previa.animada ? movel : parada
                    readonly property real larguraReal: Math.max(1, fonte.implicitWidth)
                    readonly property real alturaReal: Math.max(1, fonte.implicitHeight)
                    readonly property real ajuste: Math.min(1, (quadro.width - 24) / larguraReal, (quadro.height - 24) / alturaReal)
                    readonly property real fator: previa.escala > 0 ? previa.escala : ajuste
                    width: larguraReal * fator
                    height: alturaReal * fator
                    x: Math.max(0, (quadro.width - width) / 2)
                    y: Math.max(0, (quadro.height - height) / 2)

                    Image {
                        id: parada
                        anchors.fill: parent
                        visible: !previa.animada
                        asynchronous: true
                        cache: false
                        autoTransform: true
                        fillMode: Image.PreserveAspectFit
                        source: quadro.visible && !previa.animada && previa.item ? Arquivos.uri(previa.item.caminho) : ""
                    }
                    AnimatedImage {
                        id: movel
                        anchors.fill: parent
                        visible: previa.animada
                        cache: false
                        fillMode: Image.PreserveAspectFit
                        playing: visible
                        source: quadro.visible && previa.animada && previa.item ? Arquivos.uri(previa.item.caminho) : ""
                    }
                }
                WheelHandler {
                    onWheel: event => previa.zoom(event.angleDelta.y > 0 ? 1.2 : 1 / 1.2)
                }
            }
            Text {
                visible: quadro.visible && figura.fonte.status === Image.Error
                anchors.centerIn: parent
                text: "Esta imagem não abre aqui (falta o plugin do formato)."
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
            }
            Rolagem {
                visible: quadro.visible && quadro.contentHeight > quadro.height + 1
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                alvo: quadro
            }
            Rolagem {
                visible: quadro.visible && quadro.contentWidth > quadro.width + 1
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                horizontal: true
                alvo: quadro
            }

            // --- texto e código ---
            Flickable {
                id: folhaDeTexto
                anchors.fill: parent
                anchors.margins: 14
                visible: previa.classe === "texto" && previa.dados !== null && previa.dados.binario !== true
                contentWidth: width
                contentHeight: escrito.implicitHeight
                boundsBehavior: Flickable.StopAtBounds
                clip: true

                Text {
                    id: escrito
                    width: folhaDeTexto.width - 12
                    textFormat: Text.PlainText
                    wrapMode: Text.Wrap
                    text: folhaDeTexto.visible ? (previa.dados.texto || "") + (previa.dados.cortado === true ? "\n\n… (só o começo do arquivo)" : "") : ""
                    color: Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 12
                }
            }
            Rolagem {
                visible: folhaDeTexto.visible && folhaDeTexto.contentHeight > folhaDeTexto.height + 1
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                alvo: folhaDeTexto
            }

            // --- áudio e vídeo: o tocador vem de um arquivo à parte ---
            Loader {
                id: midia
                anchors.fill: parent
                anchors.margins: 14
                active: previa.visible && previa.daMidia && previa.item !== null
                source: "PreviaMidia.qml"
                onLoaded: {
                    item.comVideo = Qt.binding(() => previa.classe === "video");
                    item.fonte = Qt.binding(() => previa.daMidia && previa.item ? Arquivos.uri(previa.item.caminho) : "");
                }
            }
            Text {
                visible: previa.daMidia && midia.status === Loader.Error
                anchors.centerIn: parent
                width: parent.width - 60
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                text: "Para tocar aqui falta o qt6-multimedia. Enter abre no app padrão."
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 13
            }

            // --- PDF: a primeira página, pela miniatura grande ---
            Image {
                id: pagina
                anchors.fill: parent
                anchors.margins: 14
                anchors.bottomMargin: 34
                visible: previa.classe === "pdf" && previa.miniatura !== ""
                asynchronous: true
                cache: false
                fillMode: Image.PreserveAspectFit
                source: visible ? Arquivos.uri(previa.miniatura) : ""
            }
            Text {
                visible: previa.classe === "pdf"
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 10
                text: previa.miniatura !== "" ? "É só uma prévia da primeira página. Enter abre o documento."
                    : "Fazendo a prévia da primeira página…"
                color: Theme.textFaint
                font.family: Theme.sans
                font.pixelSize: 12
            }

            // --- pasta e compactado: o que há dentro ---
            Item {
                anchors.fill: parent
                anchors.margins: 14
                visible: previa.classe === "pasta" || previa.classe === "compactado"

                Text {
                    id: resumo
                    width: parent.width
                    elide: Text.ElideRight
                    text: previa.classe === "compactado"
                        ? Arquivos.plural(previa.lista.length, "item", "itens") + " na raiz do compactado"
                        : previa.contagem === null ? "Contando…"
                        : Arquivos.plural(previa.contagem.arquivos, "arquivo", "arquivos") + ", " + Arquivos.plural(previa.contagem.pastas, "pasta", "pastas")
                            + "  ·  " + Arquivos.tamanho(previa.contagem.tamanho) + (previa.contando ? "  (contando…)" : "")
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 13
                }
                ListView {
                    id: dentro
                    anchors.top: resumo.bottom
                    anchors.topMargin: 10
                    anchors.bottom: parent.bottom
                    width: parent.width
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    reuseItems: true
                    model: parent.visible ? previa.lista.length : 0
                    delegate: Item {
                        id: entrada
                        required property int index
                        readonly property var it: previa.lista[index] || null
                        width: dentro.width
                        height: 24

                        Icone {
                            x: 4
                            anchors.verticalCenter: parent.verticalCenter
                            lado: 16
                            nome: entrada.it ? entrada.it.icone : ""
                            reserva: entrada.it ? entrada.it.icone_reserva : "text-x-generic"
                        }
                        Text {
                            x: 28
                            width: parent.width - 140
                            anchors.verticalCenter: parent.verticalCenter
                            elide: Text.ElideMiddle
                            text: entrada.it ? entrada.it.nome : ""
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                        }
                        Text {
                            anchors.right: parent.right
                            anchors.rightMargin: 16
                            anchors.verticalCenter: parent.verticalCenter
                            text: !entrada.it || Arquivos.ePasta(entrada.it) ? "" : Arquivos.tamanho(entrada.it.tamanho)
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                    }
                }
                Rolagem {
                    anchors.right: parent.right
                    anchors.top: dentro.top
                    anchors.bottom: dentro.bottom
                    alvo: dentro
                }
                Text {
                    visible: previa.dados !== null && previa.dados.total > previa.lista.length && previa.classe === "pasta"
                    anchors.right: parent.right
                    anchors.rightMargin: 16
                    anchors.top: parent.top
                    text: previa.dados ? "os primeiros " + previa.lista.length + " de " + previa.dados.total : ""
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }

            // --- o resto: o ícone grande e a ficha (e o que não deu para ler) ---
            Column {
                anchors.centerIn: parent
                width: Math.min(parent.width - 60, 460)
                spacing: 10
                visible: previa.lendo || previa.erro !== "" || previa.classe === "outro" || previa.classe === ""
                    || (previa.classe === "texto" && previa.dados !== null && previa.dados.binario === true)

                Icone {
                    anchors.horizontalCenter: parent.horizontalCenter
                    lado: 128
                    nome: previa.item ? previa.item.icone || "" : ""
                    reserva: previa.item ? previa.item.icone_reserva || "application-x-generic" : "application-x-generic"
                }
                Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.WordWrap
                    text: previa.erro !== "" ? previa.erro
                        : previa.lendo ? "Lendo…"
                        : previa.classe === "texto" ? "Arquivo binário: não há texto para mostrar."
                        : "Não há prévia para este tipo. Enter abre no app padrão."
                    color: previa.erro !== "" ? Theme.copper : Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 13
                }
                Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.WordWrap
                    visible: previa.item !== null
                    text: !previa.item ? "" : Arquivos.tipo(previa.item) + "  ·  " + Arquivos.tamanho(previa.item.tamanho)
                        + (previa.item.mime ? "\n" + previa.item.mime : "")
                        + (previa.item.mtime ? "\nmodificado em " + Arquivos.data(previa.item.mtime) : "")
                        + (previa.item.permissoes ? "\n" + previa.item.permissoes + "  " + (previa.item.dono || "") + ":" + (previa.item.grupo || "") : "")
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                    lineHeight: 1.4
                }
            }
        }

        Rotulo {
            anchors.bottom: parent.bottom
            anchors.bottomMargin: 8
            x: 18
            width: parent.width - 36
            elide: Text.ElideRight
            text: "espaço ou esc fecha  ·  ← → passam  ·  enter abre" + (previa.classe === "imagem" ? "  ·  roda dá zoom" : "")
        }
    }
}
