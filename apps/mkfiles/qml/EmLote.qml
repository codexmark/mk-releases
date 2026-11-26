import QtQuick
import "Arquivos.js" as Arquivos
import "Lote.js" as Lote

// A folha do renomear em lote: os modos (que se combinam) em cima e, embaixo,
// a prévia ao vivo de antes → depois, com os conflitos marcados. A conta é do
// Lote.js; o motor recebe os pares e renomeia tudo como uma tarefa só.
Folha {
    id: lote

    property var itens: []
    // { pasta, nomes } do painel, para marcar o nome que já existe (null: não se sabe).
    property var existentes: null
    property string erro: ""
    property string caixa: ""
    property string espacos: ""
    property bool curinga: false
    property bool caixaExata: false
    property bool acentos: false
    property bool aparar: false
    property bool mudarExtensao: false
    readonly property var opcoes: ({
        de: campoDe.text, para: campoPara.text, curinga: curinga, caixaExata: caixaExata,
        modelo: campoModelo.text, inicio: lote.inteiro(campoInicio.text, 1), passo: lote.inteiro(campoPasso.text, 1),
        caixa: caixa, acentos: acentos, espacos: espacos, aparar: aparar,
        extensao: mudarExtensao ? campoExtensao.text : null
    })
    readonly property var resultado: visible ? Lote.calcular(itens, opcoes, existentes) : ({ linhas: [], pares: [], mudam: 0, conflitos: 0 })
    readonly property bool pode: resultado.mudam > 0 && resultado.conflitos === 0
    signal confirmado(var pares)

    function inteiro(texto, padrao) {
        const t = String(texto).trim();
        return /^-?\d{1,9}$/.test(t) ? Number(t) : padrao;
    }
    function abrir() {
        campoDe.text = "";
        campoPara.text = "";
        campoModelo.text = "";
        campoInicio.text = "1";
        campoPasso.text = "1";
        campoExtensao.text = "";
        caixa = "";
        espacos = "";
        curinga = false;
        caixaExata = false;
        acentos = false;
        aparar = false;
        mudarExtensao = false;
        erro = "";
        foco.restart();
    }
    function confirmar() {
        if (pode)
            confirmado(resultado.pares);
    }

    rotulo: "renomear em lote"
    titulo: "Renomear " + Arquivos.plural(itens.length, "item", "itens")
    largura: 780

    Timer {
        id: foco
        interval: 1
        onTriggered: campoDe.focar()
    }

    Row {
        width: parent.width
        spacing: 12
        Campo {
            id: campoDe
            width: (parent.width - 24) * 0.28
            dica: "trocar este texto"
            onAceito: lote.confirmar()
        }
        Campo {
            id: campoPara
            width: (parent.width - 24) * 0.28
            dica: "por este"
            onAceito: lote.confirmar()
        }
        Flow {
            width: (parent.width - 24) * 0.44
            spacing: 12
            Opcao {
                texto: "* e ? valem"
                ligado: lote.curinga
                onTrocado: lote.curinga = !lote.curinga
            }
            Opcao {
                texto: "diferenciar maiúsculas"
                ligado: lote.caixaExata
                onTrocado: lote.caixaExata = !lote.caixaExata
            }
        }
    }
    Row {
        width: parent.width
        spacing: 12
        Campo {
            id: campoModelo
            width: parent.width - 164
            mono: true
            dica: "modelo: {nome} {ext} {n:3} {data:AAAA-MM-DD} {pai}"
            onAceito: lote.confirmar()
        }
        Campo {
            id: campoInicio
            width: 64
            mono: true
            dica: "início"
            onAceito: lote.confirmar()
        }
        Campo {
            id: campoPasso
            width: 64
            mono: true
            dica: "passo"
            onAceito: lote.confirmar()
        }
    }
    Flow {
        width: parent.width
        spacing: 12
        Segmento {
            opcoes: Lote.CAIXAS
            valor: lote.caixa
            onEscolhido: v => lote.caixa = v
        }
        Row {
            spacing: 6
            Rotulo {
                anchors.verticalCenter: parent.verticalCenter
                text: "espaços"
            }
            Segmento {
                opcoes: Lote.ESPACOS
                valor: lote.espacos
                onEscolhido: v => lote.espacos = v
            }
        }
    }
    Flow {
        width: parent.width
        spacing: 14
        Opcao {
            texto: "tirar acentos"
            ligado: lote.acentos
            onTrocado: lote.acentos = !lote.acentos
        }
        Opcao {
            texto: "aparar espaços"
            ligado: lote.aparar
            onTrocado: lote.aparar = !lote.aparar
        }
        Opcao {
            texto: "mudar a extensão para"
            ligado: lote.mudarExtensao
            onTrocado: {
                lote.mudarExtensao = !lote.mudarExtensao;
                if (lote.mudarExtensao)
                    campoExtensao.focar();
            }
        }
        Campo {
            id: campoExtensao
            width: 90
            mono: true
            dica: "txt"
            opacity: lote.mudarExtensao ? 1 : 0.4
            onAceito: lote.confirmar()
        }
    }

    // A prévia: antes → depois, só as linhas à vista existem.
    Rectangle {
        width: parent.width
        height: Math.max(80, Math.min(260, lote.height - 400))
        color: Theme.ground
        border.width: 1
        border.color: Theme.hairline

        ListView {
            id: lista
            anchors.fill: parent
            anchors.margins: 1
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            reuseItems: true
            model: lote.resultado.linhas.length
            delegate: Item {
                id: par
                required property int index
                readonly property var l: lote.resultado.linhas[index] || null
                readonly property bool ruim: l !== null && l.motivo !== ""
                width: lista.width
                height: 22

                Text {
                    x: 8
                    width: parent.width * 0.42 - 12
                    anchors.verticalCenter: parent.verticalCenter
                    elide: Text.ElideMiddle
                    text: par.l ? par.l.antes : ""
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Text {
                    x: parent.width * 0.42
                    anchors.verticalCenter: parent.verticalCenter
                    text: "→"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
                Text {
                    x: parent.width * 0.42 + 20
                    width: parent.width * 0.58 - 34 - (motivo.visible ? motivo.width + 8 : 0)
                    anchors.verticalCenter: parent.verticalCenter
                    elide: Text.ElideMiddle
                    text: par.l ? par.l.depois : ""
                    color: par.ruim ? Theme.copper : par.l && par.l.muda ? Theme.text : Theme.textFaint
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Rotulo {
                    id: motivo
                    visible: par.ruim
                    anchors.right: parent.right
                    anchors.rightMargin: 14
                    anchors.verticalCenter: parent.verticalCenter
                    text: par.l ? par.l.frase : ""
                    color: Theme.copper
                }
            }
        }
        Rolagem {
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            alvo: lista
        }
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        text: lote.erro !== "" ? lote.erro
            : lote.resultado.conflitos > 0 ? Arquivos.plural(lote.resultado.conflitos, "conflito", "conflitos") + ": acerte os nomes marcados antes de renomear"
            : lote.resultado.mudam === 0 ? "Nada muda ainda: escolha o que fazer com os nomes."
            : Arquivos.plural(lote.resultado.mudam, "nome muda", "nomes mudam") + ". Ctrl+Z desfaz tudo de uma vez."
        color: lote.erro !== "" || lote.resultado.conflitos > 0 ? Theme.copper : Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 8
        Botao {
            texto: "renomear"
            forte: true
            ligado: lote.pode
            onAcionado: lote.confirmar()
        }
        Botao {
            texto: "cancelar"
            onAcionado: lote.fechado()
        }
    }
    Rotulo { text: "enter renomeia  ·  esc cancela" }
}
