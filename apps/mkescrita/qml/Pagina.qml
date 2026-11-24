import QtQuick
import QtQuick.Dialogs
import "Escrita.js" as Escrita

// A página do MkEscrita: uma coluna de texto centrada e, em volta dela, o que
// aparece só quando se pede (busca, sumário, pré-visualização, perguntas).
// Aqui mora o estado do documento; as decisões (quando salvar, o que fazer ao
// fechar, o que é conflito) são do Escrita.js, e o disco é do motor.
Item {
    id: pagina

    // --- o documento ---------------------------------------------------------
    // O arquivo em edição ("" = documento sem nome).
    property string caminho: ""
    // { fase, pendente }: a máquina de estados do salvamento (Escrita.transicao).
    property var estado: Escrita.estadoInicial()
    // O sha256 do arquivo como a janela o conhece ("ausente" = não existia).
    property string conhecida: ""
    property string fimDeLinha: "lf"
    property bool quebraFinal: true
    property bool soLeitura: false
    // O texto veio de um rascunho recuperado e a pessoa ainda não salvou:
    // enquanto isso o salvamento automático não grava.
    property bool recuperado: false
    // O motivo do conflito em aberto ("" = nenhum): mudou, sumiu, apareceu.
    property string conflito: ""
    // edit.text muda por código (abrir, recarregar): não é a pessoa digitando.
    property bool carregando: false
    // A pessoa pediu para fechar e falta só o salvamento acabar.
    property bool fecharDepois: false
    // Um evento do vigia que chegou no meio de um salvamento.
    property var eventoGuardado: null
    property var vigia: null

    // --- rascunho ------------------------------------------------------------
    property string rascunhoId: Escrita.idDeRascunho(Date.now(), Math.random())
    property bool temRascunho: false
    // Quantos "guardar" ainda estão a caminho do motor, e o que fazer quando
    // acabarem (apagar o rascunho só depois: senão o guardar atrasado o recria).
    property int guardando: 0
    property var depoisDeGuardar: null
    // O rascunho de uma sessão que morreu, oferecido ao abrir.
    property var oferecido: null

    // --- o que o motor conta -------------------------------------------------
    property var contagem: null
    property var titulos: []
    property string htmlDaPrevia: ""

    // --- preferências (lidas e gravadas pelo motor) --------------------------
    property string fonte: ""
    property int zoomPct: 100
    property bool previa: false
    property bool prefsLidas: false

    // --- o que está à vista --------------------------------------------------
    property bool foco: false
    property bool sumarioAberto: false
    property bool buscaAberta: false
    property bool fontesAbertas: false
    property bool ajudaAberta: false
    // Qual pergunta está na tela: "", "fechar", "conflito", "rascunho",
    // "sobrescrever" ou "erro".
    property string pergunta: ""
    property string erroNaTela: ""
    property string exportando: ""
    property string aviso: ""

    // As ocorrências da busca, refeitas quando o termo ou o texto mudam.
    property var achadas: []

    readonly property var doc: ({
        fase: estado.fase, pendente: estado.pendente, caminho: caminho, recuperado: recuperado,
        soLeitura: soLeitura, vazio: edit.length === 0
    })
    readonly property string titulo: Escrita.tituloDaJanela(doc)
    readonly property string fonteEmUso: fonte !== "" && Theme.families.indexOf(fonte) >= 0 ? fonte : Theme.mono
    readonly property bool livre: pergunta === "" && !ajudaAberta

    signal sair()
    signal reaparecer()
    signal telaCheia()

    // --- abrir ---------------------------------------------------------------

    function comeca() {
        Motor.pedir(["preferencias", "ler"], null, r => {
            if (!r.erro) {
                pagina.fonte = r.fonte || "";
                pagina.zoomPct = Escrita.zoom(r.zoom, 0);
                pagina.previa = !!r.previa;
            }
            pagina.prefsLidas = true;
            if (pagina.previa)
                pagina.pedePrevia();
        });
        if (Motor.arquivo !== "")
            abre(Motor.arquivo, () => pagina.procuraRascunho());
        else
            procuraRascunho();
        edit.forceActiveFocus();
    }

    // Lê um arquivo do disco para a janela. Só chame quando não há texto a
    // perder (Escrita.ondeAbrir decide isso antes).
    function abre(arquivo, depois) {
        Motor.pedir(["abrir", "--", arquivo], null, r => {
            if (r.erro)
                pagina.mostraErro(r.erro);
            else
                pagina.carrega(r);
            if (depois)
                depois(r);
        });
    }

    function carrega(r) {
        const cursor = r.arquivo === pagina.caminho ? edit.cursorPosition : 0;
        pagina.carregando = true;
        edit.text = r.texto;
        pagina.carregando = false;
        edit.cursorPosition = Math.min(cursor, edit.length);
        pagina.caminho = r.arquivo;
        pagina.conhecida = r.novo || !r.impressao ? "ausente" : r.impressao.sha256;
        pagina.fimDeLinha = r.fim_de_linha || "lf";
        pagina.quebraFinal = r.quebra_final !== false;
        pagina.soLeitura = !!r.so_leitura;
        pagina.recuperado = false;
        pagina.conflito = "";
        pagina.eventoGuardado = null;
        pagina.estado = Escrita.transicao(pagina.estado, "abriu");
        salvarLogo.stop();
        rascunhoLogo.stop();
        pagina.vigiar();
        pagina.reconta();
    }

    function mostraErro(frase) {
        pagina.erroNaTela = frase;
        pagina.pergunta = "erro";
    }

    function avisa(frase) {
        pagina.aviso = frase;
        avisoSome.restart();
    }

    // --- vigiar o arquivo ----------------------------------------------------

    function vigiar() {
        Motor.parar(pagina.vigia);
        pagina.vigia = null;
        if (pagina.caminho === "")
            return;
        pagina.vigia = Motor.vigia(["vigiar", "--esperado", pagina.conhecida || "ausente", "--", pagina.caminho],
            e => pagina.mudouPorFora(e));
    }

    function mudouPorFora(e) {
        // No meio de um salvamento, o aviso pode ser do que eu mesmo estou
        // gravando: espera a resposta do salvar para saber.
        if (pagina.estado.fase === "salvando") {
            pagina.eventoGuardado = e;
            return;
        }
        const decisao = Escrita.aoMudarPorFora(pagina.doc, e, pagina.conhecida);
        if (decisao === "recarregar") {
            pagina.abre(pagina.caminho, null);
        } else if (decisao === "conflito") {
            pagina.entraEmConflito(e.evento === "sumiu" ? "sumiu" : "mudou");
        }
    }

    function entraEmConflito(motivo) {
        pagina.conflito = motivo;
        pagina.estado = Escrita.transicao(pagina.estado, "conflito");
        salvarLogo.stop();
        // O texto que está na janela vai para o rascunho já.
        pagina.guardaRascunho();
        if (pagina.pergunta === "" || pagina.pergunta === "fechar")
            pagina.pergunta = "conflito";
        if (pagina.fecharDepois) {
            pagina.fecharDepois = false;
            pagina.reaparecer();
        }
    }

    // --- salvar --------------------------------------------------------------

    // Ctrl+S: grava no arquivo do documento; sem nome, pergunta onde.
    function salva() {
        if (pagina.caminho === "") {
            pagina.salvaComo();
            return;
        }
        if (pagina.estado.fase === "conflito") {
            pagina.pergunta = "conflito";
            return;
        }
        pagina.grava(pagina.caminho, false);
    }

    function salvaComo() {
        if (pagina.caminho !== "")
            dialogoDeSalvar.currentFolder = Escrita.urlDaPasta(pagina.caminho);
        dialogoDeSalvar.open();
    }

    // Manda o texto ao motor. `forcar`: gravar mesmo que o disco tenha mudado
    // (a pessoa escolheu "manter a minha" ou um arquivo novo no seletor).
    function grava(alvo, forcar) {
        if (pagina.estado.fase === "salvando")
            return;
        const outro = alvo !== pagina.caminho;
        // Salvar o que já está no disco não faz nada (a não ser noutro arquivo).
        if (!Escrita.temMudancas(pagina.estado) && !outro && !forcar) {
            if (pagina.fecharDepois)
                pagina.fecha();
            return;
        }
        if (!Escrita.temMudancas(pagina.estado))
            pagina.estado = Escrita.transicao(pagina.estado, "editou");
        pagina.estado = Escrita.transicao(pagina.estado, "salvar");
        salvarLogo.stop();
        const texto = edit.text;
        const args = Escrita.argsSalvar(alvo, texto, outro ? "" : pagina.conhecida, {
            forcar: forcar || outro, fimDeLinha: outro ? "lf" : pagina.fimDeLinha, quebraFinal: outro ? true : pagina.quebraFinal
        });
        Motor.pedir(args, texto, r => pagina.gravou(r, outro));
    }

    function gravou(r, outro) {
        if (r.conflito) {
            pagina.entraEmConflito(r.conflito);
            return;
        }
        if (r.erro) {
            pagina.estado = Escrita.transicao(pagina.estado, "falhou");
            pagina.guardaRascunho();
            if (pagina.fecharDepois) {
                pagina.fecharDepois = false;
                pagina.reaparecer();
            }
            pagina.mostraErro(r.erro);
            return;
        }
        pagina.conhecida = r.impressao.sha256;
        pagina.caminho = r.arquivo;
        pagina.soLeitura = false;
        pagina.recuperado = false;
        pagina.conflito = "";
        if (outro) {
            pagina.fimDeLinha = "lf";
            pagina.quebraFinal = true;
        }
        pagina.estado = Escrita.transicao(pagina.estado, "salvou");
        if (outro)
            pagina.vigiar();
        // O aviso do vigia que esperava: agora dá para saber se era o meu.
        const guardado = pagina.eventoGuardado;
        pagina.eventoGuardado = null;
        if (guardado)
            pagina.mudouPorFora(guardado);
        if (pagina.estado.fase === "salvo") {
            pagina.apagaRascunho(null);
            if (pagina.fecharDepois)
                pagina.fecha();
        } else if (pagina.estado.fase === "sujo") {
            // Digitaram no meio: o que falta vai no próximo.
            if (pagina.fecharDepois)
                pagina.grava(pagina.caminho, false);
            else
                salvarLogo.restart();
        }
    }

    // --- rascunhos -----------------------------------------------------------

    function guardaRascunho() {
        if (!Escrita.deveGuardarRascunho(pagina.doc))
            return;
        const texto = edit.text;
        pagina.temRascunho = true;
        pagina.guardando++;
        Motor.pedir(Escrita.argsRascunho(pagina.rascunhoId, texto, pagina.caminho, pagina.conhecida), texto, () => {
            pagina.guardando--;
            if (pagina.guardando === 0 && pagina.depoisDeGuardar) {
                const f = pagina.depoisDeGuardar;
                pagina.depoisDeGuardar = null;
                f();
            }
        });
    }

    function apagaRascunho(depois) {
        rascunhoLogo.stop();
        if (pagina.guardando > 0) {
            pagina.depoisDeGuardar = () => pagina.apagaRascunho(depois);
            return;
        }
        if (!pagina.temRascunho) {
            if (depois)
                depois();
            return;
        }
        pagina.temRascunho = false;
        Motor.pedir(["rascunho", "apagar", pagina.rascunhoId], null, () => {
            if (depois)
                depois();
        });
    }

    function procuraRascunho() {
        Motor.pedir(["rascunho", "listar"], null, r => {
            if (r.erro || pagina.pergunta !== "")
                return;
            const achado = Escrita.rascunhoParaOferecer(r.itens, pagina.caminho);
            if (achado) {
                pagina.oferecido = achado;
                pagina.pergunta = "rascunho";
            }
        });
    }

    // Traz o rascunho para a janela. Nunca grava nada: o texto entra como
    // mudança não salva e o salvamento automático fica parado até a pessoa
    // salvar por conta própria.
    function recupera(resumo) {
        Motor.pedir(["rascunho", "ler", resumo.id], null, r => {
            if (r.erro) {
                pagina.mostraErro(r.erro);
                return;
            }
            const poe = () => {
                pagina.carregando = true;
                edit.text = r.texto;
                pagina.carregando = false;
                edit.cursorPosition = edit.length;
                // Contra o que conferir no salvar: o arquivo como era quando o
                // rascunho foi feito. Se mudou desde então, vira conflito.
                if (pagina.caminho !== "" && r.base_sha256)
                    pagina.conhecida = r.base_sha256;
                pagina.recuperado = true;
                pagina.rascunhoId = r.id;
                pagina.temRascunho = true;
                pagina.estado = Escrita.transicao(pagina.estado, "editou");
                pagina.reconta();
            };
            if (r.origem && r.origem !== pagina.caminho)
                pagina.abre(r.origem, a => poe());
            else
                poe();
        });
    }

    // --- fechar --------------------------------------------------------------

    // A pessoa pediu para fechar (Ctrl+W, ou a janela foi fechada pelo
    // sistema: aí `janelaSumiu` e, se for preciso perguntar, ela reaparece).
    function pedeFechar(janelaSumiu) {
        const decisao = Escrita.aoFechar(pagina.doc);
        if (decisao === "fechar") {
            pagina.fecha();
        } else if (decisao === "salvar") {
            pagina.fecharDepois = true;
            if (pagina.estado.fase !== "salvando")
                pagina.grava(pagina.caminho, false);
        } else {
            // O texto vai para o rascunho antes de qualquer pergunta.
            pagina.guardaRascunho();
            if (janelaSumiu)
                pagina.reaparecer();
            pagina.pergunta = pagina.estado.fase === "conflito" ? "conflito" : "fechar";
        }
    }

    // Fecha de vez: o rascunho já não serve (o texto está no disco, ou a
    // pessoa mandou descartar).
    function fecha() {
        Motor.parar(pagina.vigia);
        pagina.vigia = null;
        pagina.apagaRascunho(() => pagina.sair());
    }

    function respondeu(valor) {
        const qual = pagina.pergunta;
        pagina.pergunta = "";
        edit.forceActiveFocus();
        if (qual === "fechar") {
            if (valor === "salvar") {
                pagina.fecharDepois = true;
                if (pagina.caminho === "")
                    pagina.salvaComo();
                else
                    pagina.grava(pagina.caminho, false);
            } else if (valor === "descartar") {
                pagina.temRascunho = true;
                pagina.fecha();
            }
        } else if (qual === "conflito") {
            if (valor === "recarregar") {
                pagina.estado = Escrita.transicao(pagina.estado, "abriu");
                pagina.apagaRascunho(null);
                pagina.abre(pagina.caminho, null);
            } else if (valor === "manter") {
                pagina.estado = Escrita.transicao(pagina.estado, "manter");
                pagina.conflito = "";
                pagina.grava(pagina.caminho, true);
            } else if (valor === "como") {
                pagina.salvaComo();
            }
        } else if (qual === "rascunho") {
            const r = pagina.oferecido;
            pagina.oferecido = null;
            if (valor === "recuperar")
                pagina.recupera(r);
            else if (valor === "descartar")
                Motor.pedir(["rascunho", "apagar", r.id], null, null);
        } else if (qual === "sobrescrever") {
            if (valor === "sim")
                pagina.exporta(pagina.exportando, true);
        }
    }

    // --- o que o motor conta -------------------------------------------------

    function reconta() {
        const texto = edit.text;
        Motor.pedir(Escrita.argsTexto("contar", texto), texto, r => {
            if (!r.erro)
                pagina.contagem = r;
        });
        if (pagina.sumarioAberto)
            pagina.pedeTitulos();
        if (pagina.previa)
            pagina.pedePrevia();
    }

    function pedeTitulos() {
        const texto = edit.text;
        Motor.pedir(Escrita.argsTexto("estrutura", texto), texto, r => {
            if (!r.erro)
                pagina.titulos = r.titulos || [];
        });
    }

    function pedePrevia() {
        const texto = edit.text;
        Motor.pedir(Escrita.argsTexto("html", texto), texto, r => {
            if (!r.erro)
                pagina.htmlDaPrevia = r.html || "";
        });
    }

    function gravaPrefs() {
        if (!pagina.prefsLidas)
            return;
        Motor.pedir(["preferencias", "gravar"], JSON.stringify({ fonte: pagina.fonte, zoom: pagina.zoomPct, previa: pagina.previa }), null);
    }

    function mudaZoom(passos) {
        pagina.zoomPct = passos === 0 ? 100 : Escrita.zoom(pagina.zoomPct, passos);
        prefsLogo.restart();
    }

    // --- exportar e copiar ---------------------------------------------------

    function exporta(saida, forcar) {
        const texto = edit.text;
        pagina.exportando = saida;
        Motor.pedir(Escrita.argsExportar(saida, texto, forcar), texto, (r, codigo) => {
            if (r.erro && codigo === 2 && !forcar)
                pagina.pergunta = "sobrescrever";
            else if (r.erro)
                pagina.mostraErro(r.erro);
            else
                pagina.avisa("exportado para " + r.nome);
        });
    }

    function copiaComoHtml() {
        const texto = edit.text;
        Motor.pedir(Escrita.argsTexto("html", texto), texto, r => {
            if (r.erro) {
                pagina.mostraErro(r.erro);
                return;
            }
            copiador.text = r.html;
            copiador.selectAll();
            copiador.copy();
            copiador.text = "";
            pagina.avisa("HTML copiado");
        });
    }

    // --- editar --------------------------------------------------------------

    // Aplica uma edição do Escrita.js no texto, por remove/insert, para ela
    // entrar no desfazer do TextEdit.
    function aplica(e) {
        if (!e)
            return false;
        if (e.fim > e.inicio)
            edit.remove(e.inicio, e.fim);
        if (e.texto !== "")
            edit.insert(e.inicio, e.texto);
        if (e.selInicio === e.selFim)
            edit.cursorPosition = e.selInicio;
        else
            edit.select(e.selInicio, e.selFim);
        return true;
    }

    function marca(qual) {
        pagina.aplica(Escrita.alternarMarca(edit.text, edit.selectionStart, edit.selectionEnd, qual));
    }

    function vaiParaLinha(linha) {
        edit.cursorPosition = Escrita.posicaoDaLinha(edit.text, linha);
        // O título fica perto do topo, não na última linha visível.
        rolo.contentY = Math.max(0, Math.min(rolo.contentHeight - rolo.height, edit.y + edit.cursorRectangle.y - 60));
        edit.forceActiveFocus();
    }

    // --- buscar --------------------------------------------------------------

    function abreBusca(comSubstituir) {
        if (edit.selectedText !== "" && edit.selectedText.indexOf("\n") < 0)
            busca.termo = edit.selectedText;
        busca.comSubstituir = comSubstituir;
        pagina.buscaAberta = true;
        pagina.refazBusca();
        busca.foca(comSubstituir);
    }

    function fechaBusca() {
        pagina.buscaAberta = false;
        edit.forceActiveFocus();
    }

    function refazBusca() {
        pagina.achadas = pagina.buscaAberta ? Escrita.ocorrencias(edit.text, busca.termo, busca.diferenciar) : [];
    }

    function vaiNaBusca(paraTras) {
        if (!pagina.buscaAberta) {
            pagina.abreBusca(false);
            return;
        }
        pagina.refazBusca();
        const de = paraTras ? edit.selectionStart : edit.selectionEnd;
        const i = Escrita.proxima(pagina.achadas, de, paraTras);
        if (i >= 0)
            edit.select(pagina.achadas[i].inicio, pagina.achadas[i].fim);
    }

    function trocaUma() {
        const e = Escrita.substituirUma(edit.text, busca.termo, busca.por, busca.diferenciar, edit.selectionStart, edit.selectionEnd);
        if (e)
            pagina.aplica(e);
        pagina.vaiNaBusca(false);
    }

    function trocaTodas() {
        const r = Escrita.substituirTodas(edit.text, busca.termo, busca.por, busca.diferenciar);
        if (r.edicao)
            pagina.aplica(r.edicao);
        pagina.refazBusca();
        pagina.avisa(r.quantas === 0 ? "nada para trocar" : r.quantas === 1 ? "1 troca" : Escrita.milhar(r.quantas) + " trocas");
    }

    // --- o que abre e fecha --------------------------------------------------

    function alternaSumario() {
        pagina.sumarioAberto = !pagina.sumarioAberto;
        if (pagina.sumarioAberto) {
            pagina.pedeTitulos();
            sumario.foca();
        } else {
            edit.forceActiveFocus();
        }
    }

    function alternaPrevia() {
        pagina.previa = !pagina.previa;
        if (pagina.previa)
            pagina.pedePrevia();
        prefsLogo.restart();
    }

    function alternaFontes() {
        pagina.fontesAbertas = !pagina.fontesAbertas;
        if (pagina.fontesAbertas)
            fontes.foca();
        else
            edit.forceActiveFocus();
    }

    function abreArquivo() {
        if (pagina.caminho !== "")
            dialogoDeAbrir.currentFolder = Escrita.urlDaPasta(pagina.caminho);
        dialogoDeAbrir.open();
    }

    Component.onCompleted: comeca()
    onPerguntaChanged: if (pergunta === "") edit.forceActiveFocus()

    // --- os relógios ---------------------------------------------------------

    // O rascunho: a cada pausa na digitação.
    Timer {
        id: rascunhoLogo
        interval: 900
        onTriggered: pagina.guardaRascunho()
    }

    // O salvamento automático: pouco depois de parar de digitar, e só quando
    // o Escrita.js deixa (documento com nome, sem conflito).
    Timer {
        id: salvarLogo
        interval: 1800
        onTriggered: {
            if (Escrita.deveSalvarSozinho(pagina.doc))
                pagina.grava(pagina.caminho, false);
        }
    }

    Timer {
        id: contaLogo
        interval: 350
        onTriggered: pagina.reconta()
    }

    Timer {
        id: prefsLogo
        interval: 600
        onTriggered: pagina.gravaPrefs()
    }

    Timer {
        id: avisoSome
        interval: 5000
        onTriggered: pagina.aviso = ""
    }

    // --- o teclado -----------------------------------------------------------

    Shortcut { sequences: ["Ctrl+S"]; enabled: pagina.livre; onActivated: pagina.salva() }
    Shortcut { sequences: ["Ctrl+Shift+S"]; enabled: pagina.livre; onActivated: pagina.salvaComo() }
    Shortcut { sequences: ["Ctrl+O"]; enabled: pagina.livre; onActivated: pagina.abreArquivo() }
    Shortcut { sequences: ["Ctrl+N"]; enabled: pagina.livre; onActivated: Motor.novaJanela("") }
    Shortcut { sequences: ["Ctrl+W", "Ctrl+Q"]; enabled: pagina.livre; onActivated: pagina.pedeFechar(false) }
    Shortcut { sequences: ["Ctrl+Y"]; enabled: pagina.livre; onActivated: edit.redo() }
    Shortcut { sequences: ["F11"]; onActivated: pagina.telaCheia() }
    Shortcut { sequences: ["Ctrl+F"]; enabled: pagina.livre; onActivated: pagina.abreBusca(false) }
    Shortcut { sequences: ["Ctrl+H"]; enabled: pagina.livre; onActivated: pagina.abreBusca(true) }
    Shortcut { sequences: ["Ctrl+G"]; enabled: pagina.livre; onActivated: pagina.vaiNaBusca(false) }
    Shortcut { sequences: ["Ctrl+Shift+G"]; enabled: pagina.livre; onActivated: pagina.vaiNaBusca(true) }
    Shortcut { sequences: ["Ctrl+B"]; enabled: pagina.livre; onActivated: pagina.marca("**") }
    Shortcut { sequences: ["Ctrl+I"]; enabled: pagina.livre; onActivated: pagina.marca("*") }
    Shortcut { sequences: ["Ctrl+Shift+X"]; enabled: pagina.livre; onActivated: pagina.marca("~~") }
    Shortcut {
        sequences: ["Ctrl+K"]
        enabled: pagina.livre
        onActivated: pagina.aplica(Escrita.inserirLink(edit.text, edit.selectionStart, edit.selectionEnd))
    }
    Shortcut { sequences: ["Ctrl+=", "Ctrl++"]; onActivated: pagina.mudaZoom(1) }
    Shortcut { sequences: ["Ctrl+-"]; onActivated: pagina.mudaZoom(-1) }
    Shortcut { sequences: ["Ctrl+0"]; onActivated: pagina.mudaZoom(0) }
    Shortcut { sequences: ["Ctrl+Shift+C"]; enabled: pagina.livre; onActivated: pagina.copiaComoHtml() }
    Shortcut { sequences: ["Ctrl+Shift+E"]; enabled: pagina.livre; onActivated: dialogoDeExportar.abre() }
    Shortcut { sequences: ["Ctrl+Shift+P"]; enabled: pagina.livre; onActivated: pagina.alternaPrevia() }
    Shortcut { sequences: ["Ctrl+Shift+O"]; enabled: pagina.livre; onActivated: pagina.alternaSumario() }
    Shortcut { sequences: ["Ctrl+Shift+F"]; enabled: pagina.livre; onActivated: pagina.foco = !pagina.foco }
    Shortcut { sequences: ["F1", "Ctrl+/"]; enabled: pagina.pergunta === ""; onActivated: pagina.ajudaAberta = !pagina.ajudaAberta }

    // --- a tela --------------------------------------------------------------

    // O sumário, à esquerda.
    Sumario {
        id: sumario
        visible: pagina.sumarioAberto
        anchors.left: parent.left
        anchors.top: parent.top
        anchors.bottom: rodape.top
        width: visible ? Math.min(280, pagina.width * 0.32) : 0
        titulos: pagina.titulos
        atual: Escrita.tituloAtual(pagina.titulos, pagina.sumarioAberto ? Escrita.linhaDe(edit.text, edit.cursorPosition) : 0)
        onEscolhido: linha => pagina.vaiParaLinha(linha)
        onFechar: pagina.alternaSumario()
    }

    // O texto.
    Item {
        id: area
        anchors.left: sumario.right
        anchors.top: parent.top
        anchors.bottom: busca.visible ? busca.top : rodape.top
        width: (pagina.width - sumario.width) * (pagina.previa ? 0.5 : 1)

        // A largura confortável da coluna: 72 caracteres da fonte em uso.
        TextMetrics {
            id: regua
            font: edit.font
            text: "000000000000000000000000000000000000000000000000000000000000000000000000"
        }

        Flickable {
            id: rolo
            anchors.fill: parent
            contentWidth: width
            contentHeight: edit.y + edit.height + Math.max(120, height * 0.45)
            boundsBehavior: Flickable.StopAtBounds
            clip: true

            // Onde a rolagem está, de 0 a 1 (a pré-visualização acompanha).
            readonly property real proporcao: contentHeight > height ? Math.max(0, Math.min(1, contentY / (contentHeight - height))) : 0

            // Mantém o cursor à vista, com uma folga em cima e embaixo.
            function mostra(r) {
                const folga = Math.min(80, height / 4);
                const topo = edit.y + r.y;
                if (topo - folga < contentY)
                    contentY = Math.max(0, topo - folga);
                else if (topo + r.height + folga > contentY + height)
                    contentY = Math.min(Math.max(0, contentHeight - height), topo + r.height + folga - height);
            }

            // Clicar no respiro em volta da coluna põe o foco no texto.
            MouseArea {
                width: rolo.contentWidth
                height: rolo.contentHeight
                cursorShape: Qt.IBeamCursor
                onClicked: {
                    edit.forceActiveFocus();
                    edit.cursorPosition = edit.length;
                }
            }

            TextEdit {
                id: edit
                x: Math.max(24, (rolo.width - width) / 2)
                y: 56
                width: Math.max(120, Math.min(rolo.width - 48, Math.ceil(regua.width) + 2))
                textFormat: TextEdit.PlainText
                wrapMode: TextEdit.Wrap
                color: Theme.text
                selectionColor: Theme.copper
                selectedTextColor: Theme.ground
                font.family: pagina.fonteEmUso
                font.pixelSize: Math.round(16 * pagina.zoomPct / 100)
                tabStopDistance: regua.width / 18
                selectByMouse: true
                persistentSelection: true
                focus: true
                activeFocusOnPress: true

                cursorDelegate: Rectangle {
                    id: cursor
                    width: 2
                    color: Theme.copper
                    visible: edit.activeFocus

                    SequentialAnimation on opacity {
                        running: edit.activeFocus
                        loops: Animation.Infinite
                        NumberAnimation { to: 1; duration: 0 }
                        PauseAnimation { duration: 600 }
                        NumberAnimation { to: 0; duration: 0 }
                        PauseAnimation { duration: 500 }
                    }
                }

                onCursorRectangleChanged: rolo.mostra(cursorRectangle)

                onTextChanged: {
                    if (pagina.carregando)
                        return;
                    pagina.estado = Escrita.transicao(pagina.estado, "editou");
                    rascunhoLogo.restart();
                    salvarLogo.restart();
                    contaLogo.restart();
                    if (pagina.buscaAberta)
                        pagina.refazBusca();
                }

                Keys.onPressed: event => {
                    // Ctrl+K é "apagar até o fim da linha" para o campo de texto do Qt:
                    // se o atalho da página não pegar a tecla, ela é pega aqui.
                    if (event.key === Qt.Key_K && (event.modifiers & Qt.ControlModifier)) {
                        pagina.aplica(Escrita.inserirLink(edit.text, edit.selectionStart, edit.selectionEnd));
                        event.accepted = true;
                        return;
                    }
                    if (event.modifiers & (Qt.ControlModifier | Qt.AltModifier | Qt.MetaModifier))
                        return;
                    if (event.key === Qt.Key_Escape) {
                        if (pagina.fontesAbertas)
                            pagina.alternaFontes();
                        else if (pagina.buscaAberta)
                            pagina.fechaBusca();
                        else if (pagina.sumarioAberto)
                            pagina.alternaSumario();
                        else if (pagina.foco)
                            pagina.foco = false;
                        return;
                    }
                    let e = null;
                    if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                        // Shift+Enter é sempre uma quebra simples.
                        if (!(event.modifiers & Qt.ShiftModifier))
                            e = Escrita.aoEnter(edit.text, edit.selectionStart, edit.selectionEnd);
                    } else if (event.key === Qt.Key_Tab) {
                        e = Escrita.recuar(edit.text, edit.selectionStart, edit.selectionEnd, false);
                    } else if (event.key === Qt.Key_Backtab) {
                        e = Escrita.recuar(edit.text, edit.selectionStart, edit.selectionEnd, true);
                        // Shift+Tab nunca tira o foco do texto.
                        event.accepted = true;
                    } else if (event.text !== "") {
                        e = Escrita.envolver(edit.text, edit.selectionStart, edit.selectionEnd, event.text);
                    }
                    if (pagina.aplica(e))
                        event.accepted = true;
                }

                // Ctrl+rolagem muda o tamanho do texto.
                WheelHandler {
                    acceptedModifiers: Qt.ControlModifier
                    onWheel: event => pagina.mudaZoom(event.angleDelta.y > 0 ? 1 : -1)
                }
            }

            // O modo foco: tudo menos o parágrafo do cursor fica esmaecido.
            Item {
                id: veu
                visible: pagina.foco && edit.length > 0
                readonly property var par: visible ? Escrita.paragrafoEm(edit.text, edit.cursorPosition) : null
                // edit.width e edit.contentHeight entram para refazer a conta quando o texto reflui.
                readonly property rect comeco: par && edit.width > 0 && edit.contentHeight >= 0 ? edit.positionToRectangle(par.inicio) : Qt.rect(0, 0, 0, 0)
                readonly property rect fim: par && edit.width > 0 && edit.contentHeight >= 0 ? edit.positionToRectangle(par.fim) : Qt.rect(0, 0, 0, 0)

                Rectangle {
                    x: 0
                    y: 0
                    width: rolo.contentWidth
                    height: Math.max(0, edit.y + veu.comeco.y)
                    color: Theme.ground
                    opacity: 0.68
                }

                Rectangle {
                    x: 0
                    y: edit.y + veu.fim.y + veu.fim.height
                    width: rolo.contentWidth
                    height: Math.max(0, rolo.contentHeight - y)
                    color: Theme.ground
                    opacity: 0.68
                }
            }
        }

        WheelHandler {
            acceptedModifiers: Qt.ControlModifier
            onWheel: event => pagina.mudaZoom(event.angleDelta.y > 0 ? 1 : -1)
        }

        // Onde se está no texto: um traço fino à direita, só quando há o que rolar.
        Rectangle {
            visible: rolo.contentHeight > rolo.height && !pagina.foco
            anchors.right: parent.right
            anchors.rightMargin: 3
            width: 2
            y: rolo.height * rolo.contentY / rolo.contentHeight
            height: Math.max(24, rolo.height * rolo.height / rolo.contentHeight)
            color: Theme.line
        }

        // O documento vazio diz o que fazer.
        Text {
            visible: edit.length === 0 && pagina.pergunta === ""
            x: edit.x
            y: edit.y - rolo.contentY
            width: edit.width
            text: "Escreva. F1 mostra os atalhos."
            color: Theme.textFaint
            font.family: Theme.serif
            font.pixelSize: edit.font.pixelSize
            font.italic: true
        }
    }

    // A pré-visualização, ao lado.
    Previa {
        visible: pagina.previa
        anchors.left: area.right
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: busca.visible ? busca.top : rodape.top
        html: pagina.htmlDaPrevia
        proporcao: rolo.proporcao
        base: Escrita.urlDaPasta(pagina.caminho)
        zoom: pagina.zoomPct
    }

    Busca {
        id: busca
        visible: pagina.buscaAberta
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: rodape.top
        resultado: Escrita.textoDaBusca(termo, pagina.achadas.length, Escrita.atual(pagina.achadas, edit.selectionStart, edit.selectionEnd))
        onTermoChanged: pagina.refazBusca()
        onDiferenciarChanged: pagina.refazBusca()
        onProxima: pagina.vaiNaBusca(false)
        onAnterior: pagina.vaiNaBusca(true)
        onTrocarUma: pagina.trocaUma()
        onTrocarTodas: pagina.trocaTodas()
        onFechar: pagina.fechaBusca()
    }

    Rodape {
        id: rodape
        // No modo foco o rodapé some.
        visible: !pagina.foco
        height: visible ? implicitHeight : 0
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        nome: Escrita.nome(pagina.caminho) || "sem título"
        estado: Escrita.textoDoEstado(pagina.doc)
        alerta: pagina.estado.fase === "conflito" || pagina.estado.fase === "erro"
        aviso: pagina.aviso
        palavras: pagina.contagem ? Escrita.textoDasPalavras(pagina.contagem.palavras) : ""
        leitura: pagina.contagem ? Escrita.tempoDeLeitura(pagina.contagem.leitura_s) : ""
        fonte: pagina.fonteEmUso
        zoom: pagina.zoomPct
        onEstadoClicado: pagina.estado.fase === "conflito" ? pagina.pergunta = "conflito" : pagina.salva()
        onFonteClicada: pagina.alternaFontes()
        onZoomClicado: pagina.mudaZoom(0)
    }

    Fontes {
        id: fontes
        visible: pagina.fontesAbertas && !pagina.foco
        anchors.right: parent.right
        anchors.rightMargin: 12
        anchors.bottom: rodape.top
        anchors.bottomMargin: 6
        height: Math.min(340, pagina.height - 80)
        familias: Theme.families
        emUso: pagina.fonte
        onEscolhida: familia => {
            pagina.fonte = familia;
            pagina.gravaPrefs();
            pagina.alternaFontes();
        }
        onFechar: pagina.alternaFontes()
    }

    Ajuda {
        visible: pagina.ajudaAberta
        onFechar: {
            pagina.ajudaAberta = false;
            edit.forceActiveFocus();
        }
    }

    // --- as perguntas --------------------------------------------------------

    // Fechar com mudanças não salvas: Salvar / Descartar / Cancelar, por S, D,
    // C, Enter e Esc (quem lê a tecla é o Escrita.respostaDaTecla).
    Pergunta {
        visible: pagina.pergunta === "fechar"
        titulo: "Fechar"
        texto: (Escrita.nome(pagina.caminho) || "Este texto") + " tem mudanças que não foram salvas."
        detalhe: pagina.caminho === "" ? "Sem um nome, ele só existe nesta janela (e no rascunho)." : ""
        interprete: tecla => Escrita.respostaDaTecla(tecla)
        opcoes: [
            { valor: "salvar", texto: "Salvar", tecla: "S", forte: true, enter: true },
            { valor: "descartar", texto: "Descartar", tecla: "D" },
            { valor: "cancelar", texto: "Cancelar", tecla: "C", esc: true }
        ]
        onRespondida: valor => pagina.respondeu(valor)
    }

    // O arquivo mudou por fora: nada grava até a pessoa decidir.
    Pergunta {
        visible: pagina.pergunta === "conflito"
        titulo: "Conflito"
        texto: Escrita.fraseDoConflito(pagina.conflito, Escrita.nome(pagina.caminho))
        detalhe: "O salvamento automático fica parado até você decidir."
        opcoes: pagina.conflito === "sumiu" ? [
            { valor: "manter", texto: "Gravar de novo", tecla: "M", forte: true },
            { valor: "como", texto: "Salvar como…", tecla: "S" },
            { valor: "depois", texto: "Decidir depois", tecla: "", esc: true }
        ] : [
            { valor: "recarregar", texto: "Recarregar do disco", tecla: "R" },
            { valor: "manter", texto: "Manter a minha", tecla: "M" },
            { valor: "como", texto: "Salvar como…", tecla: "S", forte: true },
            { valor: "depois", texto: "Decidir depois", tecla: "", esc: true }
        ]
        onRespondida: valor => pagina.respondeu(valor)
    }

    // Um rascunho de uma sessão que morreu.
    Pergunta {
        visible: pagina.pergunta === "rascunho"
        titulo: "Rascunho não salvo"
        texto: pagina.oferecido
            ? "Há um texto não salvo de " + (pagina.oferecido.nome || "um documento sem nome") + ", guardado "
                + Escrita.haQuanto(Date.now() / 1000, pagina.oferecido.guardado_em_s) + "."
            : ""
        detalhe: pagina.oferecido ? pagina.oferecido.trecho : ""
        opcoes: [
            { valor: "recuperar", texto: "Recuperar", tecla: "R", forte: true, enter: true },
            { valor: "descartar", texto: "Descartar", tecla: "D" },
            { valor: "depois", texto: "Depois", tecla: "", esc: true }
        ]
        onRespondida: valor => pagina.respondeu(valor)
    }

    Pergunta {
        visible: pagina.pergunta === "sobrescrever"
        titulo: "Exportar"
        texto: Escrita.nome(pagina.exportando) + " já existe. Passar por cima?"
        opcoes: [
            { valor: "sim", texto: "Substituir", tecla: "S" },
            { valor: "nao", texto: "Cancelar", tecla: "C", forte: true, enter: true, esc: true }
        ]
        onRespondida: valor => pagina.respondeu(valor)
    }

    Pergunta {
        visible: pagina.pergunta === "erro"
        titulo: "Não deu"
        texto: pagina.erroNaTela
        opcoes: [{ valor: "ok", texto: "Entendi", tecla: "", forte: true, enter: true, esc: true }]
        onRespondida: valor => pagina.respondeu(valor)
    }

    // --- os seletores de arquivo (os do sistema, como no MkUtil) -------------

    FileDialog {
        id: dialogoDeAbrir
        title: "Abrir um texto"
        fileMode: FileDialog.OpenFile
        nameFilters: ["Markdown e texto (*.md *.markdown *.mkd *.txt)", "Todos os arquivos (*)"]
        onAccepted: {
            const escolhido = Escrita.caminhoDeUrl(selectedFile.toString());
            if (escolhido === "")
                return;
            // Nunca troca um texto não salvo por outro: abre noutra janela.
            if (Escrita.ondeAbrir(pagina.doc) === "aqui")
                pagina.abre(escolhido, null);
            else
                Motor.novaJanela(escolhido);
        }
    }

    FileDialog {
        id: dialogoDeSalvar
        title: "Salvar como"
        fileMode: FileDialog.SaveFile
        defaultSuffix: "md"
        nameFilters: ["Markdown (*.md)", "Texto (*.txt)", "Todos os arquivos (*)"]
        onAccepted: {
            const escolhido = Escrita.caminhoDeUrl(selectedFile.toString());
            if (escolhido !== "")
                pagina.grava(escolhido, true);
            else
                pagina.fecharDepois = false;
        }
        onRejected: pagina.fecharDepois = false
    }

    FileDialog {
        id: dialogoDeExportar
        title: "Exportar HTML"
        fileMode: FileDialog.SaveFile
        defaultSuffix: "html"
        nameFilters: ["HTML (*.html)"]

        function abre() {
            if (pagina.caminho !== "") {
                currentFolder = Escrita.urlDaPasta(pagina.caminho);
                selectedFile = Escrita.urlDaPasta(pagina.caminho) + encodeURIComponent(Escrita.nomeDoExportado(pagina.caminho));
            }
            open();
        }

        onAccepted: {
            const escolhido = Escrita.caminhoDeUrl(selectedFile.toString());
            if (escolhido !== "")
                pagina.exporta(escolhido, false);
        }
    }

    // De onde o "copiar como HTML" sai: um TextEdit fora da tela.
    TextEdit {
        id: copiador
        visible: false
        textFormat: TextEdit.PlainText
    }
}
