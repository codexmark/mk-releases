import QtQuick
import QtQuick.Layouts
import QtQuick.Dialogs
import "Corte.js" as Corte

// O MkCorte: a pré-visualização em cima, a linha do tempo embaixo. A edição
// (os clipes, o desfazer) mora aqui, como um valor do Corte.js; o motor só
// sonda os vídeos, tira as miniaturas e exporta. Todo o teclado passa por
// este item: com um diálogo aberto, as teclas vão para ele.
Item {
    id: editor
    // Lado a lado com outra janela: some o que é detalhe, fica o que corta.
    readonly property bool estreito: width < 720

    // O histórico (Corte.historia), ou null antes do primeiro vídeo.
    property var h: null
    // A edição provisória enquanto uma alça é arrastada (não entra no
    // desfazer até soltar).
    property var rascunho: null
    readonly property var ed: rascunho !== null ? rascunho : h !== null ? h.atual.ed : Corte.vazia()
    readonly property bool vazio: ed.clipes.length === 0
    readonly property real total: Corte.duracao(ed)
    readonly property var onde: Corte.localizar(ed, cursor)
    property real cursor: 0
    property bool zoom: false
    // arquivo → os quadros da tira.
    property var tiras: ({})
    // A linha de estado, embaixo.
    property string aviso: ""
    property bool avisoForte: false
    property int lendo: 0
    // "", "exportar", "sair", "trocar" ou "atalhos".
    property string dialogo: ""
    // Os arquivos que esperam a resposta do "trocar".
    property var pendentes: []
    // O que o seletor de arquivos vai fazer com o que a pessoa escolher.
    property string escolhendo: "abrir"

    function diz(frase, forte) {
        editor.aviso = frase;
        editor.avisoForte = forte === true;
    }

    // Leva o cursor (e a pré-visualização) para o instante t da edição.
    function vai(t) {
        editor.cursor = Corte.passo(editor.ed, t, 0);
        previa.posiciona(editor.cursor);
    }

    // Uma edição nova entra no histórico. Devolve se mudou algo.
    function muda(nova, cursorNovo) {
        if (editor.h === null || nova === editor.h.atual.ed)
            return false;
        editor.h = Corte.aplicar(editor.h, nova);
        editor.vai(cursorNovo === undefined ? editor.cursor : cursorNovo);
        return true;
    }

    // ── abrir e juntar vídeos ──────────────────────────────────────────

    // Abre os vídeos dados no lugar da edição de agora (o primeiro abre, os
    // outros entram depois dele). Pergunta antes de jogar fora uma edição.
    function abrir(caminhos) {
        if (caminhos.length === 0)
            return;
        if (Corte.haMudancas(editor.h)) {
            editor.pendentes = caminhos;
            editor.dialogo = "trocar";
            return;
        }
        previa.pausa();
        editor.h = null;
        editor.rascunho = null;
        editor.cursor = 0;
        editor.zoom = false;
        previa.posiciona(0);
        editor.carrega(caminhos, 0, 0);
    }

    // Junta os vídeos dados depois do clipe sob o cursor.
    function adicionar(caminhos) {
        if (caminhos.length > 0)
            editor.carrega(caminhos, 0, editor.cursor);
    }

    // Sonda um vídeo por vez e o põe na edição; `em` é o instante da edição
    // depois de cujo clipe o próximo entra.
    function carrega(caminhos, i, em) {
        if (i >= caminhos.length)
            return;
        const arquivo = caminhos[i];
        editor.lendo++;
        Motor.pedir(Corte.argsSondar(arquivo), r => {
            editor.lendo--;
            let depois = em;
            if (r.erro) {
                editor.diz(r.erro, true);
            } else if (editor.h === null) {
                editor.h = Corte.historia(Corte.abrirFonte(r));
                editor.vai(0);
                depois = Corte.duracao(editor.ed);
                editor.diz(Corte.nome(r.arquivo) + "  ·  " + Corte.resumoDaFonte(Corte.fonteDe(r)));
                editor.tira(r.arquivo);
            } else {
                const antes = Corte.localizar(editor.h.atual.ed, em);
                const nova = Corte.inserirFonte(editor.h.atual.ed, r, em);
                if (editor.muda(nova)) {
                    // O próximo entra depois deste.
                    depois = Corte.inicioDoClipe(nova, antes ? antes.clipe + 1 : 0);
                    editor.diz("juntou " + Corte.nome(r.arquivo) + "  ·  " + Corte.resumoDaFonte(Corte.fonteDe(r)));
                    editor.tira(r.arquivo);
                }
            }
            editor.carrega(caminhos, i + 1, depois);
        });
    }

    // Pede a tira de quadros de um vídeo (uma vez).
    function tira(arquivo) {
        if (editor.tiras[arquivo] !== undefined)
            return;
        editor.guardaTira(arquivo, []);
        Motor.pedir(Corte.argsMiniaturas(arquivo), r => {
            if (!r.erro)
                editor.guardaTira(arquivo, r.quadros);
        });
    }

    function guardaTira(arquivo, quadros) {
        const t = Object.assign({}, editor.tiras);
        t[arquivo] = quadros;
        editor.tiras = t;
    }

    function escolher(para) {
        editor.escolhendo = para;
        seletor.open();
    }

    // As URLs do seletor ou do arrasto viram caminhos; o que não é arquivo
    // local fica de fora.
    function caminhosDe(urls) {
        const lista = [];
        for (let i = 0; i < urls.length; i++) {
            const caminho = Corte.caminhoDeUrl(urls[i].toString());
            if (caminho !== "")
                lista.push(caminho);
        }
        return lista;
    }

    // ── as ações (do teclado e dos botões) ─────────────────────────────

    function faz(a) {
        const semVideo = ["abrir", "adicionar", "atalhos", "sair", "desfazer", "refazer"];
        if (editor.vazio && semVideo.indexOf(a.acao) < 0)
            return;
        const o = editor.onde;
        switch (a.acao) {
        case "tocar":
            if (!previa.tocando && editor.cursor >= editor.total - 0.05)
                editor.vai(0);
            previa.alterna();
            break;
        case "pular":
            editor.vai(Corte.pular(editor.ed, a.n));
            break;
        case "andar":
            editor.vai(Corte.passo(editor.ed, editor.cursor, a.delta));
            break;
        case "quadro":
            previa.pausa();
            editor.vai(Corte.passo(editor.ed, editor.cursor, a.delta * Corte.quadro(editor.ed, editor.cursor)));
            break;
        case "bordaAnterior":
            editor.vai(Corte.bordaAnterior(editor.ed, editor.cursor));
            break;
        case "bordaSeguinte":
            editor.vai(Corte.bordaSeguinte(editor.ed, editor.cursor));
            break;
        case "dividir":
            editor.divide(editor.cursor);
            break;
        case "remover":
            editor.remove(o.clipe);
            break;
        case "juntar":
            if (editor.muda(Corte.juntar(editor.ed, o.clipe)))
                editor.diz("clipes " + (o.clipe + 1) + " e " + (o.clipe + 2) + " juntados");
            else
                editor.diz("só dá para juntar dois pedaços vizinhos e contíguos do mesmo vídeo");
            break;
        case "moverAntes":
        case "moverDepois":
            editor.move(o.clipe, o.clipe + (a.acao === "moverAntes" ? -1 : 1));
            break;
        case "trazerInicio": {
            const nova = Corte.trazerInicio(editor.ed, editor.cursor);
            // O que havia antes do cursor saiu: ele fica no começo do clipe.
            if (editor.muda(nova, Corte.inicioDoClipe(nova, o.clipe)))
                editor.diz("início do clipe " + (o.clipe + 1) + " trazido para o cursor");
            break;
        }
        case "trazerFim":
            if (editor.muda(Corte.trazerFim(editor.ed, editor.cursor)))
                editor.diz("fim do clipe " + (o.clipe + 1) + " trazido para o cursor");
            break;
        case "desfazer":
            if (Corte.podeDesfazer(editor.h)) {
                editor.h = Corte.desfazer(editor.h);
                editor.vai(editor.cursor);
                editor.diz("desfeito");
            }
            break;
        case "refazer":
            if (Corte.podeRefazer(editor.h)) {
                editor.h = Corte.refazer(editor.h);
                editor.vai(editor.cursor);
                editor.diz("refeito");
            }
            break;
        case "zoom":
            editor.zoom = !editor.zoom;
            break;
        case "salvarQuadro":
            previa.pausa();
            Motor.pedir(Corte.argsQuadro(editor.ed, editor.cursor), r => {
                if (r.erro)
                    editor.diz(r.erro, true);
                else
                    editor.diz("quadro salvo em " + r.saida);
            });
            break;
        case "abrir":
            editor.escolher("abrir");
            break;
        case "adicionar":
            editor.escolher(editor.h === null ? "abrir" : "adicionar");
            break;
        case "exportar":
            previa.pausa();
            editor.dialogo = "exportar";
            exportar.abre();
            break;
        case "sair":
            if (Corte.haMudancas(editor.h))
                editor.dialogo = "sair";
            else
                Qt.quit();
            break;
        case "atalhos":
            previa.pausa();
            editor.dialogo = "atalhos";
            break;
        }
    }

    function divide(t) {
        if (editor.muda(Corte.dividir(editor.ed, t), t))
            editor.diz("dividido em " + Corte.tempo(t));
        else
            editor.diz("perto demais de uma borda para dividir");
    }

    function remove(i) {
        const inicio = Corte.inicioDoClipe(editor.ed, i);
        const dentro = editor.onde !== null && editor.onde.clipe === i;
        // Com o cursor no clipe removido, ele fica onde o clipe começava;
        // antes dele não muda; depois, recua junto com o resto.
        const cursorNovo = dentro ? inicio : editor.cursor > inicio ? editor.cursor - Corte.duracaoDoClipe(editor.ed.clipes[i]) : editor.cursor;
        if (editor.muda(Corte.remover(editor.ed, i), cursorNovo))
            editor.diz("clipe " + (i + 1) + " removido  ·  ctrl+z desfaz");
    }

    function move(de, para) {
        const o = editor.onde;
        const nova = Corte.mover(editor.ed, de, para);
        if (nova === editor.ed)
            return;
        // O cursor acompanha o clipe em que estava.
        let cursorNovo = editor.cursor;
        if (o !== null && o.clipe === de)
            cursorNovo = Corte.inicioDoClipe(nova, Corte.entre(para, 0, nova.clipes.length - 1)) + o.dentro;
        if (editor.muda(nova, cursorNovo))
            editor.diz("clipe " + (de + 1) + " foi para a posição " + (Corte.entre(para, 0, nova.clipes.length - 1) + 1));
    }

    // Uma alça está sendo arrastada: a edição provisória, com o encaixe, e a
    // pré-visualização mostrando onde o corte cai.
    function apara(clipe, lado, tempoNaFonte, tolerancia) {
        if (editor.h === null)
            return;
        previa.pausa();
        const base = editor.h.atual.ed;
        const alvo = Corte.encaixar(tempoNaFonte, Corte.alvosDeEncaixe(base, clipe, editor.cursor), tolerancia);
        const nova = lado === "inicio" ? Corte.apararInicio(base, clipe, alvo) : Corte.apararFim(base, clipe, alvo);
        editor.rascunho = nova;
        const c = nova.clipes[clipe];
        previa.mostra(nova.fontes[c.fonte].arquivo, lado === "inicio" ? c.inicio : Math.max(c.inicio, c.fim - 1 / nova.fontes[c.fonte].fps));
        editor.diz((lado === "inicio" ? "início" : "fim") + " do clipe " + (clipe + 1) + " em " + Corte.tempo(lado === "inicio" ? c.inicio : c.fim) +
            "  ·  duração " + Corte.tempo(Corte.duracaoDoClipe(c)));
    }

    function aparou() {
        const nova = editor.rascunho;
        editor.rascunho = null;
        if (nova === null || !editor.muda(nova))
            editor.vai(editor.cursor);
    }

    // ── o teclado ──────────────────────────────────────────────────────

    function teclaDoDialogo(event) {
        if (editor.dialogo === "atalhos") {
            editor.dialogo = "";
            return;
        }
        const acao = Corte.acaoDoDialogo(event.key, event.modifiers);
        if (acao === null)
            return;
        if (editor.dialogo === "exportar")
            exportar.tecla(acao);
        else if (editor.dialogo === "sair")
            sair.tecla(acao);
        else if (editor.dialogo === "trocar")
            trocar.tecla(acao);
    }

    focus: true
    Keys.onPressed: event => {
        event.accepted = true;
        if (editor.dialogo !== "") {
            editor.teclaDoDialogo(event);
            return;
        }
        const a = Corte.acaoDaTecla(event.key, event.modifiers);
        if (a !== null)
            editor.faz(a);
        else
            event.accepted = false;
    }

    Component.onCompleted: {
        editor.forceActiveFocus();
        if (Motor.arquivos.length > 0)
            editor.abrir(Motor.arquivos);
        else if (Motor.exemplo !== "")
            editor.abrir(["/exemplo/passeio na praia.mp4"]);
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 12

        // ── o alto: de quem é, o vídeo e os botões ─────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Column {
                Layout.fillWidth: true
                spacing: 3

                Rotulo { text: "mk  /  MkCorte" }
                Text {
                    width: parent.width
                    text: editor.vazio ? "Corte" : editor.ed.fontes.length === 1 ? Corte.nome(editor.ed.fontes[0].arquivo)
                        : Corte.nome(editor.ed.fontes[editor.ed.clipes[0].fonte].arquivo) + "  +  " + (editor.ed.fontes.length - 1)
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                    elide: Text.ElideMiddle
                }
            }
            Botao { texto: "abrir"; onAcionado: editor.faz({ acao: "abrir" }) }
            Botao { texto: "exportar"; forte: true; ligado: !editor.vazio; onAcionado: editor.faz({ acao: "exportar" }) }
            Botao { texto: "?"; onAcionado: editor.faz({ acao: "atalhos" }) }
        }

        // ── a pré-visualização ─────────────────────────────────────────
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Theme.surface
            border.width: 1
            border.color: solta.containsDrag ? Theme.copper : Theme.hairline

            Previa {
                id: previa
                anchors.fill: parent
                anchors.margins: 1
                ed: editor.ed
                semVideo: Motor.exemplo !== ""
                visible: !editor.vazio
                onAndou: t => editor.cursor = t
                onFalhou: frase => editor.diz(frase, true)
            }
            MouseArea {
                anchors.fill: parent
                enabled: !editor.vazio
                onClicked: editor.faz({ acao: "tocar" })
            }

            // Sem vídeo: o convite.
            Column {
                anchors.centerIn: parent
                visible: editor.vazio
                spacing: 12

                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: solta.containsDrag ? "pode soltar" : editor.lendo > 0 ? "lendo o vídeo…" : editor.h !== null ? "a edição ficou vazia" : "solte um vídeo aqui"
                    color: solta.containsDrag ? Theme.copper : Theme.textDim
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.italic: true
                    font.weight: Font.Light
                }
                Text {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: editor.h !== null ? "ctrl+z desfaz  ·  ctrl+shift+o junta outro vídeo" : "ou escolha um  ·  ctrl+o abre  ·  ? mostra os atalhos"
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
                Botao {
                    anchors.horizontalCenter: parent.horizontalCenter
                    texto: "escolher…"
                    onAcionado: editor.escolher("abrir")
                }
            }
        }

        // ── o transporte ───────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 8
            opacity: editor.vazio ? 0.45 : 1

            Botao {
                Layout.preferredWidth: 78
                texto: previa.tocando ? "pausar" : "tocar"
                forte: true
                ligado: !editor.vazio
                onAcionado: editor.faz({ acao: "tocar" })
            }
            Text {
                text: Corte.tempo(editor.cursor) + "  /  " + Corte.tempo(editor.total)
                color: Theme.text
                font.family: Theme.mono
                font.pixelSize: 13
            }
            Text {
                Layout.fillWidth: true
                visible: !editor.estreito
                text: editor.onde === null ? "" : "clipe " + (editor.onde.clipe + 1) + " de " + editor.ed.clipes.length + "  ·  " +
                    Corte.tempo(editor.onde.tempo) + " em " + Corte.nome(editor.ed.fontes[editor.ed.clipes[editor.onde.clipe].fonte].arquivo)
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
                elide: Text.ElideMiddle
            }
            Botao { texto: "dividir"; ligado: !editor.vazio; onAcionado: editor.faz({ acao: "dividir" }) }
            Botao { texto: "remover"; ligado: !editor.vazio; onAcionado: editor.faz({ acao: "remover" }) }
            Botao { texto: "desfazer"; ligado: Corte.podeDesfazer(editor.h); onAcionado: editor.faz({ acao: "desfazer" }) }
            Item { Layout.fillWidth: true; visible: editor.estreito }
            Botao { texto: "refazer"; visible: !editor.estreito; ligado: Corte.podeRefazer(editor.h); onAcionado: editor.faz({ acao: "refazer" }) }
            Botao { texto: editor.zoom ? "ver tudo" : "zoom"; visible: !editor.estreito; ligado: !editor.vazio; onAcionado: editor.faz({ acao: "zoom" }) }
        }

        // ── a linha do tempo ───────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 8

            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 86
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                Linha {
                    anchors.fill: parent
                    anchors.margins: 1
                    ed: editor.ed
                    cursor: editor.cursor
                    zoom: editor.zoom
                    tiras: editor.tiras
                    onIrPara: t => editor.vai(t)
                    onAparando: (clipe, lado, tempoNaFonte, tolerancia) => editor.apara(clipe, lado, tempoNaFonte, tolerancia)
                    onAparou: editor.aparou()
                    onDividirEm: t => {
                        editor.vai(t);
                        editor.divide(editor.cursor);
                    }
                    onRemoverClipe: clipe => editor.remove(clipe)
                    onMoverClipe: (de, para) => editor.move(de, para)
                }
                Text {
                    anchors.centerIn: parent
                    visible: editor.vazio
                    text: "a linha do tempo aparece aqui"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 13
                    font.italic: true
                }
            }
            // Juntar outro vídeo.
            Rectangle {
                Layout.preferredWidth: 40
                Layout.preferredHeight: 86
                color: mais.containsMouse ? Theme.wash : Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: mais.containsMouse ? Theme.copper : Theme.line

                Text {
                    anchors.centerIn: parent
                    text: "+"
                    color: mais.containsMouse ? Theme.copper : Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 20
                }
                MouseArea {
                    id: mais
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: editor.faz({ acao: "adicionar" })
                }
            }
        }

        // ── a linha de estado ──────────────────────────────────────────
        Text {
            Layout.fillWidth: true
            text: editor.lendo > 0 ? "lendo o vídeo…" : editor.aviso !== "" ? editor.aviso
                : "espaço toca  ·  s divide  ·  x remove  ·  arraste as pontas para aparar  ·  ? mostra os atalhos"
            color: editor.avisoForte && editor.lendo === 0 ? Theme.copper : Theme.textDim
            font.family: Theme.mono
            font.pixelSize: 10
            elide: Text.ElideMiddle
        }
    }

    // Soltar vídeos em qualquer lugar da janela: sem edição, abre; com uma,
    // junta depois do clipe sob o cursor.
    DropArea {
        id: solta
        anchors.fill: parent
        enabled: editor.dialogo === ""
        onDropped: drop => {
            if (!drop.hasUrls)
                return;
            const caminhos = editor.caminhosDe(drop.urls);
            drop.acceptProposedAction();
            if (editor.h === null)
                editor.abrir(caminhos);
            else
                editor.adicionar(caminhos);
        }
    }

    FileDialog {
        id: seletor
        title: editor.escolhendo === "adicionar" ? "Juntar outro vídeo" : "Abrir um vídeo"
        fileMode: FileDialog.OpenFiles
        nameFilters: ["Vídeos (*.mp4 *.mkv *.mov *.webm *.avi *.m4v *.mpg *.mpeg *.ts *.wmv *.flv *.3gp *.ogv)", "Todos os arquivos (*)"]
        onAccepted: {
            const caminhos = editor.caminhosDe(selectedFiles);
            if (editor.escolhendo === "adicionar")
                editor.adicionar(caminhos);
            else
                editor.abrir(caminhos);
        }
    }

    // ── os diálogos ────────────────────────────────────────────────────

    Exportar {
        id: exportar
        anchors.fill: parent
        visible: editor.dialogo === "exportar"
        ed: editor.h !== null ? editor.h.atual.ed : Corte.vazia()
        onFechado: editor.dialogo = ""
        onExportou: editor.h = Corte.marcarExportado(editor.h)
    }

    Confirma {
        id: sair
        anchors.fill: parent
        visible: editor.dialogo === "sair"
        titulo: "Sair sem exportar?"
        texto: "A edição não foi exportada. Se sair agora, os cortes se perdem (os vídeos originais ficam como estão)."
        opcoes: [{ v: "exportar", texto: "exportar", forte: true }, { v: "sair", texto: "sair sem exportar" }, { v: "cancelar", texto: "continuar editando" }]
        onEscolhido: v => {
            editor.dialogo = "";
            if (v === "sair")
                Qt.quit();
            else if (v === "exportar")
                editor.faz({ acao: "exportar" });
        }
    }

    Confirma {
        id: trocar
        anchors.fill: parent
        visible: editor.dialogo === "trocar"
        titulo: "Abrir outro vídeo?"
        texto: "A edição de agora não foi exportada. Dá para abrir o vídeo novo no lugar dela, ou juntá-lo ao que já está aqui."
        opcoes: [{ v: "juntar", texto: "juntar à edição", forte: true }, { v: "abrir", texto: "abrir no lugar" }, { v: "cancelar", texto: "cancelar" }]
        onEscolhido: v => {
            const caminhos = editor.pendentes;
            editor.pendentes = [];
            editor.dialogo = "";
            if (v === "juntar") {
                editor.adicionar(caminhos);
            } else if (v === "abrir") {
                // A pessoa já respondeu: a edição de agora pode ir.
                editor.h = null;
                editor.abrir(caminhos);
            }
        }
    }

    Atalhos {
        anchors.fill: parent
        visible: editor.dialogo === "atalhos"
        onFechado: editor.dialogo = ""
    }
}
