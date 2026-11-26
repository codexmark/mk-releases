import QtQuick
import "Arquivos.js" as Arquivos

// As propriedades de um item ou de vários: a aba Geral (o que é, onde está,
// quanto ocupa, contando o conteúdo das pastas enquanto o motor conta) e a
// aba Permissões (ler, gravar e executar de dono, grupo e outros; o grupo;
// aplicar ao conteúdo). Quem muda é o motor: a folha só pede.
Folha {
    id: props

    // O que o motor respondeu ao `propriedades`: a primeira linha e a contagem.
    property var dados: null
    property var contagem: ({ arquivos: 0, pastas: 0, tamanho: 0 })
    property bool contando: false
    property string aba: "geral"
    property int modo: 0
    property bool recursivo: false
    property string grupo: ""
    property string erro: ""
    signal aplicado(var mudanca)

    readonly property var itens: dados && dados.itens ? dados.itens : []
    readonly property var um: itens.length === 1 ? itens[0] : null
    readonly property bool temPasta: itens.some(it => it.tipo === "pasta")
    readonly property bool mudou: um !== null && (modo !== um.modo || grupo !== um.grupo)

    function abrir() {
        aba = "geral";
        erro = "";
        recursivo = false;
    }
    // Chega a primeira linha (ou os itens de volta de um `permissoes`).
    function carregar() {
        modo = um ? um.modo : 0;
        grupo = um ? um.grupo : "";
    }
    function aplicar() {
        if (um !== null && mudou)
            aplicado({ modo: modo !== um.modo ? modo : null, grupo: grupo !== um.grupo ? grupo : null, recursivo: recursivo });
    }

    onDadosChanged: carregar()

    rotulo: "propriedades"
    titulo: um ? um.nome : Arquivos.plural(itens.length, "item", "itens")
    largura: 480

    Segmento {
        opcoes: [{ v: "geral", texto: "geral" }, { v: "permissoes", texto: "permissões" }]
        valor: props.aba
        onEscolhido: v => props.aba = v
    }

    // --- Geral ---
    Grid {
        width: parent.width
        visible: props.aba === "geral"
        columns: 2
        columnSpacing: 14
        rowSpacing: 6

        Repeater {
            model: {
                const linhas = [];
                const it = props.um;
                if (it) {
                    linhas.push(["tipo", Arquivos.tipo(it)]);
                    linhas.push(["mime", it.mime]);
                }
                if (props.dados && props.dados.onde)
                    linhas.push(["onde", props.dados.onde]);
                if (it && it.origem)
                    linhas.push(["veio de", it.origem]);
                if (it && it.tipo === "link")
                    linhas.push(["leva a", it.alvo + (it.quebrado ? "  (quebrado)" : "")]);
                const c = props.contagem;
                const soArquivo = it && it.tipo !== "pasta";
                linhas.push(["tamanho", (soArquivo ? Arquivos.tamanho(it.tamanho) : Arquivos.tamanho(c.tamanho)) + (props.contando ? "  (contando…)" : "")]);
                if (!soArquivo)
                    linhas.push(["conteúdo", Arquivos.plural(c.arquivos, "arquivo", "arquivos") + ", " + Arquivos.plural(c.pastas, "pasta", "pastas")]);
                if (it) {
                    linhas.push(["modificado", Arquivos.data(it.mtime)]);
                    linhas.push(["dono", it.dono + " : " + it.grupo]);
                    linhas.push(["permissões", it.permissoes + "  (" + Arquivos.octal(it.modo) + ")"]);
                }
                return linhas;
            }

            Row {
                id: par
                required property var modelData
                width: props.width > 0 ? Math.min(430, props.width - 80) : 430
                spacing: 12
                Rotulo {
                    width: 84
                    anchors.baseline: valor.baseline
                    text: par.modelData[0]
                }
                Text {
                    id: valor
                    width: par.width - 96
                    wrapMode: Text.WrapAnywhere
                    maximumLineCount: 3
                    elide: Text.ElideRight
                    text: par.modelData[1]
                    color: Theme.textSoft
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
            }
        }
    }

    // --- Permissões ---
    Column {
        width: parent.width
        visible: props.aba === "permissoes"
        spacing: 8

        Text {
            width: parent.width
            wrapMode: Text.WordWrap
            visible: props.um === null
            text: "As permissões são mudadas um item por vez."
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
        }
        Text {
            width: parent.width
            wrapMode: Text.WordWrap
            visible: props.um !== null && props.dados && props.dados.sou_dono === false
            text: "Só o dono (" + (props.um ? props.um.dono : "") + ") pode mudar as permissões."
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
        }

        Grid {
            visible: props.um !== null
            columns: 4
            columnSpacing: 16
            rowSpacing: 8
            verticalItemAlignment: Grid.AlignVCenter

            Repeater {
                model: {
                    if (props.um === null)
                        return [];
                    // A primeira linha são os títulos; depois, uma linha por dono, grupo e outros.
                    const celulas = [{ t: "" }, { t: "ler" }, { t: "gravar" }, { t: props.um.tipo === "pasta" ? "entrar" : "executar" }];
                    ["dono", "grupo", "outros"].forEach((quem, q) => {
                        celulas.push({ t: quem });
                        for (const b of ["r", "w", "x"])
                            celulas.push({ q: q, b: b });
                    });
                    return celulas;
                }

                Item {
                    id: celula
                    required property var modelData
                    readonly property bool eChave: modelData.b !== undefined
                    width: 76
                    height: 20

                    Rotulo {
                        visible: !celula.eChave
                        anchors.verticalCenter: parent.verticalCenter
                        text: celula.eChave ? "" : celula.modelData.t
                    }
                    Chave {
                        visible: celula.eChave
                        anchors.verticalCenter: parent.verticalCenter
                        mudavel: !!props.dados && props.dados.sou_dono !== false
                        ligado: celula.eChave && Arquivos.temBit(props.modo, celula.modelData.q, celula.modelData.b)
                        onTrocado: props.modo = Arquivos.trocarBit(props.modo, celula.modelData.q, celula.modelData.b)
                    }
                }
            }
        }
        Text {
            visible: props.um !== null
            text: props.um ? Arquivos.octal(props.modo) + (props.modo !== props.um.modo ? "  (era " + Arquivos.octal(props.um.modo) + ")" : "") : ""
            color: Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Rotulo {
            visible: grupos.visible
            text: "grupo"
        }
        Flow {
            id: grupos
            width: parent.width
            spacing: 6
            visible: props.um !== null && !!props.dados && props.dados.sou_dono === true && (props.dados.grupos || []).length > 1
            Repeater {
                model: props.dados && props.dados.grupos ? props.dados.grupos : []
                Botao {
                    required property var modelData
                    texto: modelData
                    forte: props.grupo === modelData
                    onAcionado: props.grupo = modelData
                }
            }
        }

        Row {
            spacing: 10
            visible: props.um !== null && props.um.tipo === "pasta"
            Chave {
                ligado: props.recursivo
                onTrocado: props.recursivo = !props.recursivo
            }
            Text {
                anchors.verticalCenter: parent.verticalCenter
                text: "aplicar ao conteúdo da pasta"
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
        }
    }

    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: props.erro !== ""
        text: props.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 8
        Botao {
            visible: props.aba === "permissoes" && props.um !== null
            texto: "aplicar"
            forte: true
            ligado: props.mudou || (props.recursivo && props.um !== null)
            onAcionado: props.um !== null && (props.mudou ? props.aplicar() : props.aplicado({ modo: props.modo, grupo: null, recursivo: true }))
        }
        Botao {
            texto: "fechar"
            onAcionado: props.fechado()
        }
    }
    Rotulo { text: "ctrl+tab troca de aba  ·  esc fecha" }
}
