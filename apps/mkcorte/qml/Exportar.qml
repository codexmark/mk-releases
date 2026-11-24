import QtQuick
import "Corte.js" as Corte

// O diálogo de exportar: a qualidade, o modo (exato ou rápido, quando dá), o
// que o `plano` do motor respondeu, o progresso, cancelar e, no fim, mostrar
// na pasta. Responde pelo teclado: ↑ ↓ escolhem a linha, ← → (ou Tab) a
// opção, Enter confirma, Esc cancela a exportação ou fecha.
Item {
    id: exportar

    property var ed: Corte.vazia()
    property string qualidade: "original"
    property string modo: "exato"
    // A resposta do `plano` (ou { erro }), e por que o rápido não serve.
    property var plano: null
    property string nota: ""
    property int pedidos: 0
    // "pronto", "exportando", "feito" ou "erro".
    property string estado: "pronto"
    property var fase: null
    property var fim: null
    property string erro: ""
    property var tarefa: null
    // A linha marcada (0 qualidade, 1 modo, 2 os botões) e o botão marcado.
    property int linha: 2
    property int botao: 0
    readonly property bool planoBom: plano !== null && !plano.erro
    readonly property bool podeRapido: planoBom && plano.sem_recodificar.pode === true
    readonly property var botoes: estado === "pronto" ? [{ v: "exportar", texto: "exportar", forte: true }, { v: "fechar", texto: "fechar" }]
        : estado === "exportando" ? [{ v: "cancelar", texto: "cancelar", forte: true }]
        : estado === "feito" ? [{ v: "pasta", texto: "mostrar na pasta", forte: true }, { v: "fechar", texto: "fechar" }]
        : [{ v: "exportar", texto: "tentar de novo", forte: true }, { v: "fechar", texto: "fechar" }]

    signal fechado()
    signal exportou()

    // Abre (ou reabre) para a edição de agora.
    function abre() {
        if (exportar.tarefa !== null)
            return;
        exportar.estado = "pronto";
        exportar.fase = null;
        exportar.fim = null;
        exportar.erro = "";
        exportar.linha = 2;
        exportar.botao = 0;
        exportar.pede();
    }

    // Pergunta ao motor como seria. Uma resposta atrasada (de um pedido
    // anterior) não vale.
    function pede() {
        const n = ++exportar.pedidos;
        exportar.plano = null;
        Motor.pedir(Corte.argsPlano(exportar.ed, { qualidade: exportar.qualidade, modo: exportar.modo }), r => {
            if (n !== exportar.pedidos)
                return;
            // O rápido não serve para esta edição: o motor recusa e diz por
            // quê. Volta para o exato e pergunta de novo.
            if (r.erro && exportar.modo === "rapido") {
                exportar.nota = r.erro;
                exportar.modo = "exato";
                exportar.pede();
                return;
            }
            exportar.plano = r;
            if (!r.erro && !r.sem_recodificar.pode)
                exportar.nota = "o modo rápido não serve aqui: " + r.sem_recodificar.motivo;
            else if (!r.erro && exportar.modo === "exato")
                exportar.nota = "";
        });
    }

    function escolheQualidade(v) {
        if (exportar.estado !== "pronto" || v === exportar.qualidade)
            return;
        exportar.qualidade = v;
        exportar.pede();
    }

    function escolheModo(v) {
        if (exportar.estado !== "pronto" || v === exportar.modo)
            return;
        if (v === "rapido" && exportar.planoBom && !exportar.podeRapido)
            return;
        exportar.modo = v;
        exportar.pede();
    }

    function comeca() {
        if (exportar.tarefa !== null || !exportar.planoBom)
            return;
        exportar.estado = "exportando";
        exportar.fase = null;
        exportar.erro = "";
        exportar.botao = 0;
        exportar.linha = 2;
        exportar.tarefa = Motor.tarefa(Corte.argsExportar(exportar.ed, { qualidade: exportar.qualidade, modo: exportar.modo }), fase => {
            if (fase.fase === "exportando")
                exportar.fase = fase;
        }, ultima => {
            exportar.tarefa = null;
            exportar.botao = 0;
            if (ultima.fase === "fim" && ultima.ok) {
                exportar.fim = ultima;
                exportar.estado = "feito";
                exportar.exportou();
            } else if (ultima.fase === "cancelado") {
                exportar.estado = "pronto";
            } else {
                exportar.erro = ultima.erro || "o ffmpeg parou sem dizer por quê";
                exportar.estado = "erro";
            }
        });
    }

    // Cancela: o motor mata o ffmpeg e apaga a saída pela metade.
    function cancela() {
        if (exportar.tarefa !== null)
            exportar.tarefa.running = false;
    }

    function aciona(v) {
        if (v === "exportar")
            exportar.comeca();
        else if (v === "cancelar")
            exportar.cancela();
        else if (v === "pasta")
            Qt.openUrlExternally(Corte.urlDeCaminho(Corte.pasta(exportar.fim.saida)));
        else if (v === "fechar" && exportar.tarefa === null)
            exportar.fechado();
    }

    function gira(lista, valor, delta) {
        for (let i = 0; i < lista.length; i++)
            if (lista[i].v === valor)
                return lista[Corte.girar(i, lista.length, delta)].v;
        return valor;
    }

    // acao: o que o Corte.acaoDoDialogo devolveu.
    function tecla(acao) {
        const ajustavel = exportar.estado === "pronto";
        if (acao === "cancelar") {
            if (exportar.tarefa !== null)
                exportar.cancela();
            else
                exportar.fechado();
        } else if (acao === "cima" || acao === "baixo") {
            exportar.linha = ajustavel ? Corte.girar(exportar.linha, 3, acao === "cima" ? -1 : 1) : 2;
        } else if (acao === "proximo" || acao === "anterior") {
            const d = acao === "proximo" ? 1 : -1;
            if (exportar.linha === 0 && ajustavel)
                exportar.escolheQualidade(exportar.gira(qualidades.opcoes, exportar.qualidade, d));
            else if (exportar.linha === 1 && ajustavel)
                exportar.escolheModo(exportar.modo === "exato" ? "rapido" : "exato");
            else
                exportar.botao = Corte.girar(exportar.botao, exportar.botoes.length, d);
        } else if (acao === "confirmar") {
            // Nas linhas de opção, Enter é o botão principal.
            exportar.aciona(exportar.botoes[exportar.linha === 2 ? Math.min(exportar.botao, exportar.botoes.length - 1) : 0].v);
        }
    }

    // A janela fechou no meio: para o motor, que mata o ffmpeg e apaga a saída
    // pela metade.
    Component.onDestruction: cancela()

    Rectangle {
        anchors.fill: parent
        color: Theme.ground
        opacity: 0.86
    }
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        onWheel: wheel => wheel.accepted = true
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(parent.width - 40, 560)
        height: Math.min(parent.height - 20, corpo.implicitHeight + 44)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        Column {
            id: corpo
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 22
            spacing: 12

            Text {
                text: "Exportar"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }

            // A qualidade.
            Row {
                spacing: 10
                opacity: exportar.estado === "pronto" ? 1 : 0.5

                Rotulo {
                    anchors.verticalCenter: parent.verticalCenter
                    width: 86
                    text: (exportar.linha === 0 ? "› " : "  ") + "qualidade"
                    color: exportar.linha === 0 ? Theme.copper : Theme.textFaint
                }
                Segmento {
                    id: qualidades
                    opcoes: [{ v: "original", texto: "original" }, { v: "1080", texto: "1080p" }, { v: "720", texto: "720p" }]
                    valor: exportar.qualidade
                    onEscolhido: v => exportar.escolheQualidade(v)
                }
            }

            // O modo.
            Row {
                spacing: 10
                opacity: exportar.estado === "pronto" ? 1 : 0.5

                Rotulo {
                    anchors.verticalCenter: parent.verticalCenter
                    width: 86
                    text: (exportar.linha === 1 ? "› " : "  ") + "modo"
                    color: exportar.linha === 1 ? Theme.copper : Theme.textFaint
                }
                Segmento {
                    opcoes: [{ v: "exato", texto: "exato" }, { v: "rapido", texto: "rápido, sem recodificar" }]
                    valor: exportar.modo
                    onEscolhido: v => exportar.escolheModo(v)
                }
            }
            // O que cada modo é, com a ressalva do rápido dita antes de a pessoa escolher.
            Text {
                width: parent.width
                text: exportar.modo === "rapido" && exportar.podeRapido ? exportar.plano.sem_recodificar.aviso
                    : exportar.nota !== "" ? exportar.nota
                    : exportar.podeRapido ? "Exato: recodifica, e o corte cai no quadro marcado. Rápido: copia o vídeo como está, em segundos, mas o corte cai no quadro-chave anterior."
                    : "Exato: recodifica, e o corte cai no quadro marcado."
                color: exportar.modo === "rapido" ? Theme.copper : Theme.textDim
                font.family: Theme.serif
                font.pixelSize: 12
                font.italic: true
                wrapMode: Text.WordWrap
            }

            Rectangle { width: parent.width; height: 1; color: Theme.hairline }

            // O que o plano respondeu.
            Rotulo { text: "o que vai sair" }
            Text {
                width: parent.width
                text: exportar.plano === null ? "calculando…" : exportar.plano.erro ? exportar.plano.erro : Corte.textoDoPlano(exportar.plano)
                color: exportar.plano !== null && exportar.plano.erro ? Theme.copper : Theme.text
                font.family: Theme.mono
                font.pixelSize: 12
                wrapMode: Text.WordWrap
            }
            Text {
                width: parent.width
                visible: exportar.planoBom || exportar.fim !== null
                text: exportar.fim !== null ? exportar.fim.saida : exportar.planoBom ? exportar.plano.saida : ""
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
                elide: Text.ElideMiddle
            }
            Repeater {
                model: exportar.planoBom && exportar.modo === "exato" ? exportar.plano.avisos : []

                Text {
                    required property string modelData
                    width: corpo.width
                    text: modelData
                    color: Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 12
                    font.italic: true
                    wrapMode: Text.WordWrap
                }
            }

            // O progresso, o resultado ou o erro.
            Barra {
                width: parent.width
                visible: exportar.estado === "exportando" || exportar.estado === "feito"
                pct: exportar.estado === "feito" ? 100 : exportar.fase ? exportar.fase.pct : 0
            }
            Text {
                width: parent.width
                visible: exportar.estado !== "pronto"
                text: exportar.estado === "exportando" ? Corte.textoDoProgresso(exportar.fase)
                    : exportar.estado === "feito" ? "pronto  ·  " + Corte.tamanho(exportar.fim.tamanho_bytes)
                    : exportar.erro
                color: exportar.estado === "exportando" ? Theme.textSoft : Theme.copper
                font.family: Theme.mono
                font.pixelSize: 11
                wrapMode: Text.WordWrap
            }

            Row {
                spacing: 8
                topPadding: 4

                Repeater {
                    model: exportar.botoes

                    Column {
                        id: resposta
                        required property var modelData
                        required property int index
                        spacing: 3

                        Botao {
                            texto: resposta.modelData.texto
                            forte: resposta.modelData.forte === true
                            ligado: resposta.modelData.v !== "exportar" || exportar.planoBom
                            onAcionado: exportar.aciona(resposta.modelData.v)
                        }
                        Rectangle {
                            width: parent.width
                            height: 2
                            color: exportar.linha === 2 && exportar.botao === resposta.index ? Theme.copper : Qt.rgba(0, 0, 0, 0)
                        }
                    }
                }
            }
            Rotulo { text: "↑ ↓ a linha  ·  ← → a opção  ·  enter confirma  ·  esc " + (exportar.estado === "exportando" ? "cancela" : "fecha") }
        }
    }
}
