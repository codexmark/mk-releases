import QtQuick
import QtQuick.Layouts
import "Util.js" as Util

// MkComprimir: a pessoa solta (ou escolhe) vídeos e imagens, diz como quer
// encolher e vê a estimativa de cada um antes de rodar. A fila roda um por
// vez, com o progresso que o ffmpeg dá; a saída fica ao lado do original e
// nunca em cima de nada. Quem chama o ffmpeg é o motor.
Item {
    id: pagina

    // Cada item: { arquivo, estado, midia, erro, fase, pct, fim }.
    // estado: "lendo", "pronto", "erro", "fila", "codificando", "feito", "cancelado".
    property var itens: []
    property var opcoes: Util.opcoesPadrao()
    property var caps: ({ ffmpeg: true, codecs: ["h264"], aceleracao: [], imagens: ["jpeg"], animados: [] })
    property var tarefa: null
    property string rodando: ""
    readonly property bool ocupada: rodando !== ""
    readonly property bool temVideo: Util.tem(itens, "video")
    readonly property bool temImagem: Util.tem(itens, "imagem")
    readonly property bool porQualidade: ["menor", "equilibrio", "alta", "caber", "resolucao"].indexOf(opcoes.predefinicao) >= 0
    readonly property var previsto: Util.total(itens, opcoes, false)
    readonly property var feito: Util.total(itens, opcoes, true)

    function muda(arquivo, campos) {
        const i = Util.indice(itens, arquivo);
        if (i >= 0)
            itens = Util.muda(itens, i, campos);
    }

    function opcao(campos) {
        opcoes = Object.assign({}, opcoes, campos);
    }

    function entram(caminhos) {
        const novos = Util.novos(itens, caminhos);
        if (novos.length === 0)
            return;
        itens = itens.concat(novos.map(c => ({ arquivo: c, estado: "lendo", midia: null, erro: "", fase: null, pct: 0, fim: null })));
        for (const c of novos)
            sonda(c);
    }

    function sonda(arquivo) {
        Motor.pedir(["midia", arquivo], r => {
            if (r.erro)
                pagina.muda(arquivo, { estado: "erro", erro: r.erro });
            else
                pagina.muda(arquivo, { estado: "pronto", midia: r, erro: "" });
        });
    }

    function tira(arquivo) {
        const i = Util.indice(itens, arquivo);
        if (i < 0)
            return;
        const n = itens.slice();
        n.splice(i, 1);
        itens = n;
    }

    // Põe na fila tudo o que está pronto e começa pelo primeiro.
    function comeca() {
        itens = itens.map(i => i.estado === "pronto" && i.midia ? Object.assign({}, i, { estado: "fila", erro: "", fase: null, pct: 0 }) : i);
        segue();
    }

    function segue() {
        const i = Util.proximo(itens);
        if (i < 0) {
            rodando = "";
            tarefa = null;
            return;
        }
        const it = itens[i], arquivo = it.arquivo;
        rodando = arquivo;
        muda(arquivo, { estado: "codificando" });
        tarefa = Motor.tarefa(Util.argsComprimir(it.midia, opcoes), fase => {
            if (fase.fase === "codificando")
                pagina.muda(arquivo, { fase: fase, pct: fase.pct });
        }, ultima => {
            if (ultima.fase === "fim" && ultima.ok)
                pagina.muda(arquivo, { estado: "feito", fim: ultima, pct: 100 });
            else if (ultima.fase === "cancelado")
                pagina.muda(arquivo, { estado: "pronto", fase: null, pct: 0 });
            else
                pagina.muda(arquivo, { estado: "pronto", fase: null, pct: 0, erro: ultima.erro || "o ffmpeg parou sem dizer por quê" });
            pagina.segue();
        });
    }

    // Cancela tudo: esvazia a fila e para o que está rodando (o motor mata o
    // ffmpeg e apaga a saída pela metade).
    function cancela() {
        itens = itens.map(i => i.estado === "fila" ? Object.assign({}, i, { estado: "pronto" }) : i);
        if (tarefa)
            tarefa.running = false;
    }

    function descarta(it) {
        Motor.pedir(["descartar", it.fim.saida, "--original", it.arquivo], r => {
            if (r.erro)
                pagina.muda(it.arquivo, { erro: r.erro });
            else
                pagina.muda(it.arquivo, { estado: "pronto", fim: null, pct: 0, erro: "" });
        });
    }

    // A janela fechou no meio: para o motor, que mata o ffmpeg e apaga a saída
    // pela metade.
    Component.onDestruction: {
        if (tarefa)
            tarefa.running = false;
    }

    Component.onCompleted: {
        Motor.pedir(["codificadores"], r => {
            if (r.erro)
                return;
            pagina.caps = r;
            // Só o que o ffmpeg desta máquina tem.
            if (r.imagens.length > 0 && r.imagens.indexOf(pagina.opcoes.formato) < 0)
                pagina.opcao({ formato: r.imagens[0] });
        });
        if (Motor.arquivos.length > 0)
            entram(Motor.arquivos);
        else if (Motor.exemplo !== "")
            entram(["/exemplo/férias na praia.mp4", "/exemplo/foto do grupo.png"]);
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        Cabecalho {
            titulo: "Comprimir"
            frase: "Vídeos e imagens menores, com o tamanho estimado antes de começar. O original fica como está."
        }

        Text {
            Layout.fillWidth: true
            visible: !pagina.caps.ffmpeg
            text: "Falta o ffmpeg nesta máquina: sem ele não dá para comprimir. Rode mk atualizar."
            color: Theme.copper
            font.family: Theme.sans
            font.pixelSize: 13
            wrapMode: Text.WordWrap
        }

        Solta {
            Layout.fillWidth: true
            titulo: "Escolher vídeos e imagens"
            texto: "solte aqui vídeos ou imagens"
            filtros: ["Vídeos e imagens (*.mp4 *.mkv *.mov *.webm *.avi *.m4v *.gif *.jpg *.jpeg *.png *.webp *.avif *.bmp *.tif *.tiff *.heic)", "Todos os arquivos (*)"]
            onArquivos: caminhos => pagina.entram(caminhos)
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 16

            // ── a fila ─────────────────────────────────────────────────
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                ListView {
                    id: fila
                    anchors.fill: parent
                    anchors.margins: 1
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    model: pagina.itens

                    delegate: Item {
                        id: linha
                        required property var modelData
                        readonly property var it: modelData
                        readonly property var est: it.midia && it.estado === "pronto" ? Util.estimar(it.midia, pagina.opcoes) : null
                        width: ListView.view.width
                        height: 64

                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }

                        Column {
                            anchors.left: parent.left
                            anchors.leftMargin: 14
                            anchors.right: acoes.left
                            anchors.rightMargin: 12
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 4

                            Text {
                                width: parent.width
                                text: Util.nome(linha.it.arquivo)
                                color: linha.it.estado === "erro" ? Theme.textDim : Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                                elide: Text.ElideMiddle
                            }
                            Text {
                                width: parent.width
                                text: linha.it.estado === "lendo" ? "lendo…" : linha.it.midia ? Util.resumoDaMidia(linha.it.midia) : ""
                                visible: text !== ""
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                                elide: Text.ElideRight
                            }
                            // Uma linha de estado: o erro, a estimativa, o progresso ou o resultado.
                            Barra {
                                width: parent.width
                                visible: linha.it.estado === "codificando"
                                pct: linha.it.pct
                            }
                            Text {
                                width: parent.width
                                elide: Text.ElideRight
                                font.family: Theme.mono
                                font.pixelSize: 10
                                color: linha.it.erro !== "" || linha.it.estado === "feito" || (linha.est !== null && !linha.est.encolhe) ? Theme.copper : Theme.textSoft
                                text: linha.it.erro !== "" ? linha.it.erro
                                    : linha.it.estado === "fila" ? "na fila"
                                    : linha.it.estado === "codificando" ? (linha.it.fase ? Util.textoDoProgresso(linha.it.fase) : "começando…")
                                    : linha.it.estado === "feito" ? Util.textoDoResultado(linha.it.fim)
                                    : linha.est !== null ? Util.textoDaEstimativa(linha.it.midia, linha.est) + (linha.est.encolhe && linha.est.nota !== "" ? "  ·  " + linha.est.nota : "")
                                    : ""
                            }
                        }

                        Row {
                            id: acoes
                            anchors.right: parent.right
                            anchors.rightMargin: 12
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 6
                            // A saída ficou maior que o original: oferece apagar.
                            Botao {
                                visible: linha.it.estado === "feito" && linha.it.fim !== null && linha.it.fim.maior
                                forte: true
                                texto: "apagar a saída"
                                onAcionado: pagina.descarta(linha.it)
                            }
                            Botao {
                                visible: linha.it.estado !== "codificando" && linha.it.estado !== "fila"
                                texto: "tirar"
                                onAcionado: pagina.tira(linha.it.arquivo)
                            }
                        }
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: fila.count === 0
                        text: "nenhum arquivo ainda"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }
                }
            }

            // ── as opções ──────────────────────────────────────────────
            Rectangle {
                Layout.preferredWidth: 310
                Layout.fillHeight: true
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                Flickable {
                    id: rola
                    anchors.fill: parent
                    anchors.margins: 14
                    anchors.bottomMargin: 76
                    clip: true
                    contentHeight: painel.height
                    boundsBehavior: Flickable.StopAtBounds

                    Column {
                        id: painel
                        width: rola.width
                        spacing: 8
                        opacity: pagina.ocupada ? 0.5 : 1
                        enabled: !pagina.ocupada

                        // Vídeo: as predefinições, cada uma com o que faz.
                        Rotulo { text: "vídeo"; visible: pagina.temVideo || !pagina.temImagem }
                        Column {
                            width: parent.width
                            visible: pagina.temVideo || !pagina.temImagem
                            Repeater {
                                model: Util.predefinicoesDeVideo.filter(p => (p.v !== "gif" && p.v !== "webp") || pagina.caps.animados.indexOf(p.v) >= 0)
                                Rectangle {
                                    id: pre
                                    required property var modelData
                                    readonly property bool atual: pagina.opcoes.predefinicao === modelData.v
                                    width: painel.width
                                    height: 24
                                    color: atual ? Theme.wash : toque.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)

                                    Rectangle { visible: pre.atual; width: 2; height: parent.height; color: Theme.copper }
                                    Text {
                                        id: nomeDaPre
                                        x: 10
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: pre.modelData.texto
                                        color: pre.atual ? Theme.text : Theme.textSoft
                                        font.family: Theme.sans
                                        font.pixelSize: 12
                                    }
                                    Text {
                                        anchors.left: nomeDaPre.right
                                        anchors.leftMargin: 8
                                        anchors.right: parent.right
                                        anchors.rightMargin: 6
                                        anchors.verticalCenter: parent.verticalCenter
                                        horizontalAlignment: Text.AlignRight
                                        text: pre.modelData.frase
                                        color: Theme.textFaint
                                        font.family: Theme.serif
                                        font.pixelSize: 11
                                        font.italic: true
                                        elide: Text.ElideRight
                                    }
                                    MouseArea {
                                        id: toque
                                        anchors.fill: parent
                                        hoverEnabled: true
                                        cursorShape: Qt.PointingHandCursor
                                        onClicked: pagina.opcao({ predefinicao: pre.modelData.v })
                                    }
                                }
                            }
                        }

                        // Caber: o tamanho, com os limites comuns à mão.
                        Row {
                            visible: pagina.temVideo && pagina.opcoes.predefinicao === "caber"
                            spacing: 8
                            Campo {
                                id: alvo
                                anchors.verticalCenter: parent.verticalCenter
                                implicitWidth: 56
                                so: "numero"
                                dica: "MB"
                                // Sem ligação: o que a pessoa digita ("2,") não pode ser reescrito no meio.
                                Component.onCompleted: text = String(pagina.opcoes.alvoMb).replace(".", ",")
                                onTextChanged: {
                                    const v = parseFloat(text.replace(",", "."));
                                    if (v > 0 && v !== pagina.opcoes.alvoMb)
                                        pagina.opcao({ alvoMb: v });
                                }
                            }
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; text: "MB" }
                            Segmento {
                                anchors.verticalCenter: parent.verticalCenter
                                opcoes: [{ v: "8", texto: "8" }, { v: "10", texto: "10" }, { v: "25", texto: "25" }, { v: "50", texto: "50" }]
                                valor: String(pagina.opcoes.alvoMb)
                                onEscolhido: v => {
                                    pagina.opcao({ alvoMb: Number(v) });
                                    alvo.text = v;
                                }
                            }
                        }

                        // Só reduzir a resolução: qual.
                        Segmento {
                            visible: pagina.temVideo && pagina.opcoes.predefinicao === "resolucao"
                            opcoes: [{ v: "1080", texto: "1080p" }, { v: "720", texto: "720p" }, { v: "480", texto: "480p" }]
                            valor: String(pagina.opcoes.lado)
                            onEscolhido: v => pagina.opcao({ lado: Number(v) })
                        }

                        // O codec, a aceleração (só se a pessoa pedir) e o áudio.
                        Item { width: 1; height: 2; visible: pagina.temVideo && pagina.porQualidade }
                        Row {
                            visible: pagina.temVideo && pagina.porQualidade && pagina.caps.codecs.length > 1
                            spacing: 10
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; width: 64; text: "codec" }
                            Segmento {
                                opcoes: [{ v: "h264", texto: "h.264" }, { v: "h265", texto: "h.265" }, { v: "av1", texto: "av1" }].filter(c => pagina.caps.codecs.indexOf(c.v) >= 0)
                                valor: pagina.opcoes.codec
                                onEscolhido: v => pagina.opcao({ codec: v })
                            }
                        }
                        Text {
                            width: parent.width
                            visible: pagina.temVideo && pagina.porQualidade && pagina.opcoes.codec !== "h264"
                            text: "menor que o H.264, mas nem todo aparelho e mensageiro abre"
                            color: Theme.textFaint
                            font.family: Theme.serif
                            font.pixelSize: 11
                            font.italic: true
                            wrapMode: Text.WordWrap
                        }
                        Row {
                            visible: pagina.temVideo && pagina.porQualidade && pagina.caps.aceleracao.length > 0
                            spacing: 10
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; width: 64; text: "placa" }
                            Segmento {
                                opcoes: [{ v: "", texto: "não usar" }].concat(pagina.caps.aceleracao.map(a => ({ v: a, texto: a })))
                                valor: pagina.opcoes.acelerar
                                onEscolhido: v => pagina.opcao({ acelerar: v })
                            }
                        }
                        Text {
                            width: parent.width
                            visible: pagina.temVideo && pagina.porQualidade && pagina.opcoes.acelerar !== ""
                            text: "mais rápido, arquivo maior; só funciona se a placa desta máquina tiver esse codificador"
                            color: Theme.textFaint
                            font.family: Theme.serif
                            font.pixelSize: 11
                            font.italic: true
                            wrapMode: Text.WordWrap
                        }
                        Row {
                            visible: pagina.temVideo && pagina.porQualidade
                            spacing: 10
                            Chave {
                                anchors.verticalCenter: parent.verticalCenter
                                ligado: pagina.opcoes.semAudio
                                onTrocado: pagina.opcao({ semAudio: !pagina.opcoes.semAudio })
                            }
                            Text {
                                anchors.verticalCenter: parent.verticalCenter
                                text: "tirar o áudio"
                                color: Theme.textSoft
                                font.family: Theme.sans
                                font.pixelSize: 12
                            }
                        }

                        // Imagem: formato, qualidade e largura.
                        Item { width: 1; height: 6; visible: pagina.temImagem && pagina.temVideo }
                        Rotulo { text: "imagem"; visible: pagina.temImagem }
                        Row {
                            visible: pagina.temImagem
                            spacing: 10
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; width: 64; text: "formato" }
                            Segmento {
                                opcoes: [{ v: "jpeg", texto: "jpeg" }, { v: "webp", texto: "webp" }, { v: "avif", texto: "avif" }].filter(f => pagina.caps.imagens.indexOf(f.v) >= 0)
                                valor: pagina.opcoes.formato
                                onEscolhido: v => pagina.opcao({ formato: v })
                            }
                        }
                        Row {
                            visible: pagina.temImagem
                            spacing: 10
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; width: 64; text: "qualidade" }
                            Segmento {
                                opcoes: [{ v: "60", texto: "menor" }, { v: "80", texto: "boa" }, { v: "92", texto: "alta" }]
                                valor: String(pagina.opcoes.qualidade)
                                onEscolhido: v => pagina.opcao({ qualidade: Number(v) })
                            }
                        }
                        Row {
                            visible: pagina.temImagem
                            spacing: 10
                            Rotulo { anchors.verticalCenter: parent.verticalCenter; width: 64; text: "largura" }
                            Campo {
                                anchors.verticalCenter: parent.verticalCenter
                                implicitWidth: 150
                                so: "numero"
                                dica: "a original (em pixels)"
                                onTextChanged: {
                                    const v = parseInt(text, 10);
                                    pagina.opcao({ largura: v >= 16 ? v : 0 });
                                }
                            }
                        }
                    }
                }

                // O pé: o total e o botão.
                Column {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.bottom: parent.bottom
                    anchors.margins: 14
                    spacing: 8

                    Rectangle { width: parent.width; height: 1; color: Theme.hairline }
                    Text {
                        width: parent.width
                        elide: Text.ElideRight
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 10
                        text: pagina.ocupada ? "comprimindo  " + Util.nome(pagina.rodando)
                            : pagina.feito.n > 0 && pagina.previsto.n === 0 ? "feito:  " + Util.tamanho(pagina.feito.antes) + " → " + Util.tamanho(pagina.feito.depois)
                            : pagina.previsto.n > 0 ? "estimado:  " + Util.tamanho(pagina.previsto.antes) + " → ≈ " + Util.tamanho(pagina.previsto.depois)
                            : "a estimativa aparece aqui"
                    }
                    Botao {
                        width: parent.width
                        forte: true
                        ligado: pagina.ocupada || (Util.prontos(pagina.itens) > 0 && pagina.caps.ffmpeg)
                        texto: pagina.ocupada ? "cancelar"
                            : Util.prontos(pagina.itens) === 1 ? "comprimir 1 arquivo"
                            : "comprimir " + Util.prontos(pagina.itens) + " arquivos"
                        onAcionado: pagina.ocupada ? pagina.cancela() : pagina.comeca()
                    }
                }
            }
        }
    }
}
