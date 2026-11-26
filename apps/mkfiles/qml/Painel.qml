import QtQuick
import "Arquivos.js" as Arquivos
import "Selecao.js" as Selecao
import "Abas.js" as Abas
import "Remoto.js" as Remoto

// Um painel de arquivos: as abas dele, a pasta à vista (ou a lixeira, ou os
// resultados de uma busca) numa das três visões, a seleção, o mouse inteiro
// (clique, retângulo, arrastar e soltar), o teclado de navegação e o renomear
// no lugar. As listas são virtualizadas: os itens moram num array (`itens`),
// o modelo das visões é só a contagem, e cada delegate lê o seu pelo índice.
// A janela tem dois destes (o painel duplo); o App cuida do resto.
FocusScope {
    id: painel

    // O App (pedidos, folhas, menus) e qual dos dois painéis é este.
    property var janela: null
    property int numero: 0
    property bool ativo: false
    property bool pronto: false
    property var prefs: ({})

    // As abas ({ abas, atual }, do Abas.js) e a que está à vista.
    property var abas: Abas.painel([Motor.home], 0)
    readonly property var aba: abas.abas[abas.atual]
    readonly property string caminho: aba.caminho
    readonly property bool lixeira: Abas.eLixeira(caminho)
    // Dentro de um compactado aberto como pasta: { arquivo, interno } (só leitura).
    readonly property var compactado: Arquivos.partesDoCompactado(caminho)
    // A senha de cada compactado cifrado aberto aqui: só na memória, enquanto
    // a janela vive; nunca vai para as preferências nem para o estado.
    property var segredos: ({})
    function segredoDe(arquivo) { return segredos[arquivo] || null; }

    // A busca: com texto, os resultados substituem a listagem.
    property string buscaTexto: ""
    property bool buscaConteudo: false
    property string buscaTipo: "qualquer"
    // Os filtros de tamanho e data da busca (o motor é quem filtra).
    property var filtros: ({ min: "", max: "", unidade: "MB", periodo: "", de: "", ate: "" })
    property bool filtrosAbertos: false
    readonly property bool filtrando: campoMin.comFoco || campoMax.comFoco || campoDe.comFoco || campoAte.comFoco
    function filtrar(mudanca) {
        filtros = Object.assign({}, filtros, mudanca);
        if (busca)
            buscar(buscaTexto);
    }
    readonly property bool busca: buscaTexto !== ""
    // Na lixeira e na busca cada item diz também de onde é.
    readonly property bool extra: lixeira || busca

    // A listagem: crua (como o motor mandou) e a que aparece (filtrada e ordenada).
    property var todos: []
    property var itens: []
    readonly property int total: itens.length
    property var selecao: Selecao.nova()
    property bool carregando: false
    property var erro: null
    property bool gravavel: true
    property int pedido: 0
    property string vigiado: ""
    property var espaco: null
    // O que selecionar quando aparecer (o recém-criado, o arquivo do `mkfiles arquivo`).
    property var aSelecionar: []

    // As miniaturas: caminho -> arquivo do cache.
    property var minis: ({})
    property int miniVersao: 0
    property var pedidas: ({})
    property var filaDeMinis: []
    property string grupoDeMinis: ""

    readonly property string visao: {
        const porPasta = prefs.visao_por_pasta || {};
        const v = porPasta[caminho] || prefs.visao || "icones";
        return v === "lista" || v === "compacta" ? v : "icones";
    }
    readonly property int iconePx: Math.max(32, Math.min(256, prefs.icone_px || 64))
    readonly property var vista: visao === "lista" ? linhas : visao === "compacta" ? colunas : grade
    property int alvoDeSoltar: -1
    property bool renomeando: false
    property string digitado: ""

    // As colunas da lista que cabem na largura de agora.
    readonly property var cols: {
        const c = prefs.colunas || {};
        const l = [{ id: "tamanho", titulo: "tamanho", largura: c.tamanho || 90 }];
        if (width > 430)
            l.push({ id: "mtime", titulo: lixeira ? "apagado" : "modificado", largura: c.mtime || 150 });
        if (width > 580)
            l.push({ id: "tipo", titulo: lixeira ? "veio de" : busca ? "pasta" : "tipo", largura: c.tipo || 150 });
        if (width > 740 && !extra)
            l.push({ id: "permissoes", titulo: "permissões", largura: c.permissoes || 100 });
        return l;
    }
    readonly property real larguraDoNome: {
        let soma = 0;
        for (const c of cols)
            soma += c.largura;
        return Math.max(140, width - soma - 12);
    }

    signal mudou()

    // O texto de uma coluna para um item.
    function celula(it, coluna) {
        if (coluna === "tamanho")
            return Arquivos.ePasta(it) ? "—" : Arquivos.tamanho(it.tamanho);
        if (coluna === "mtime")
            return Arquivos.data(lixeira ? it.apagado_s : it.mtime);
        if (coluna === "permissoes")
            return it.permissoes || "";
        if (lixeira)
            return it.origem ? Arquivos.til(Arquivos.pai(it.origem), Motor.home) : "";
        if (busca)
            return Arquivos.til(Arquivos.pai(it.caminho), Motor.home);
        return Arquivos.tipo(it);
    }

    function focar() { teclado.forceActiveFocus(); }
    function selecionados() { return Selecao.lista(selecao, itens); }

    // --- abas e navegação ---
    function ir(destino) {
        if (destino === caminho) {
            if (busca)
                sairDaBusca();
            return;
        }
        abas = Abas.comAba(abas, Abas.ir(aba, destino));
    }
    function voltar() { abas = Abas.comAba(abas, Abas.voltar(aba)); }
    function avancar() { abas = Abas.comAba(abas, Abas.avancar(aba)); }
    // A pasta à vista mora num servidor conectado (o ponto do gvfs)?
    readonly property bool remoto: janela !== null && !lixeira && compactado === null && Remoto.eRemoto(caminho, janela.gvfs)
    // A conexão caiu (ou o servidor foi desconectado): o painel volta para a
    // pasta pessoal e a barra de estado diz o que houve; nunca fica em branco.
    function caiuAConexao(frase) {
        const r = janela !== null ? Remoto.doRemoto(caminho, janela.remotos) : null;
        janela.avisar(frase || (r !== null ? "a conexão com " + r.nome + " caiu" : "a conexão caiu"));
        ir(Motor.home);
    }
    function subir() {
        const de = caminho;
        // Acima da raiz de um servidor só há a pasta do gvfs: sobe para casa.
        if (remoto && janela.remotos.some(r => r.raiz === de)) {
            ir(Motor.home);
            return;
        }
        if (!Abas.podeSubir(aba))
            return;
        aSelecionar = [de];
        abas = Abas.comAba(abas, Abas.subir(aba));
    }
    function novaAba(destino) { abas = Abas.abrir(abas, destino || caminho); }
    function fecharAba(i) { abas = Abas.fechar(abas, i); }
    function duplicarAba(i) { abas = Abas.duplicar(abas, i); }
    function trocarAba(delta) { abas = Abas.trocar(abas, delta); }

    onAbasChanged: mudou()
    onCaminhoChanged: abrirPasta()
    onProntoChanged: abrirPasta()
    Component.onDestruction: largar()

    // --- a listagem ---
    // Solta o que a pasta anterior tinha pendurado no motor.
    function largar() {
        if (carregando && pedido > 0)
            Motor.pedir({ op: "cancelar", pedido: pedido }, null);
        if (vigiado !== "") {
            Motor.pedir({ op: "desvigiar", caminho: vigiado }, null);
            vigiado = "";
        }
        if (grupoDeMinis !== "") {
            Motor.pedir({ op: "miniatura-cancelar", grupo: grupoDeMinis }, null);
            grupoDeMinis = "";
        }
        filaDeMinis = [];
    }
    function limpar() {
        todos = [];
        itens = [];
        selecao = Selecao.nova();
        erro = null;
        minis = ({});
        pedidas = ({});
        alvoDeSoltar = -1;
        renomeando = false;
    }
    function abrirPasta() {
        largar();
        if (!pronto)
            return;
        buscaTexto = "";
        limpar();
        carregando = true;
        espaco = null;
        if (lixeira) {
            pedido = Motor.pedir({ op: "lixeira-listar" }, chegou);
            return;
        }
        if (compactado !== null) {
            // Um compactado não é vigiado nem tem espaço livre: só a listagem.
            pedido = Motor.pedir({ op: "compactado-listar", caminho: compactado.arquivo, dentro: compactado.interno, senha: segredoDe(compactado.arquivo) }, chegou);
            return;
        }
        const onde = caminho;
        pedido = Motor.pedir({ op: "listar", caminho: onde }, chegou);
        vigiado = onde;
        Motor.pedir({ op: "vigiar", caminho: onde }, null);
        Motor.pedir({ op: "espaco", caminho: onde }, r => {
            if (onde === painel.caminho)
                painel.espaco = r.erro ? null : r;
        });
    }
    function recarregar() {
        if (busca)
            buscar(buscaTexto);
        else
            abrirPasta();
    }
    // Uma linha do fluxo (listar, lixeira-listar ou buscar).
    function chegou(r) {
        if (r.id !== pedido)
            return;
        if (r.erro) {
            carregando = false;
            // Um servidor que caiu, sumiu ou não respondeu: de volta para casa.
            if (compactado === null && !lixeira && !busca && caminho !== Motor.home && Remoto.caiu(r.motivo, caminho, janela.gvfs)) {
                caiuAConexao(r.motivo === "conexao" ? r.erro : "");
                return;
            }
            erro = { motivo: r.motivo || "erro", frase: r.motivo === "falta" && r.pacote ? r.erro + " (pacote " + r.pacote + ")" : r.erro };
            if (r.motivo === "senha" && compactado !== null)
                pedirSegredo();
            return;
        }
        const primeiro = todos.length === 0;
        const arquivo = compactado !== null ? compactado.arquivo : "";
        for (const it of r.itens || []) {
            if (arquivo !== "")
                // O "caminho" de um item de dentro é o lugar dele no compactado.
                todos.push(Object.assign({}, it, { caminho: Arquivos.noCompactado(arquivo, it.interno) }));
            else
                todos.push(lixeira && it.nome_original ? Object.assign({}, it, { nome: it.nome_original }) : it);
        }
        if (r.fim === true) {
            carregando = false;
            if (r.gravavel !== undefined)
                gravavel = r.gravavel;
            refazer();
            // A listagem chegou: o teclado é do painel (se nada mais o usa).
            refocar();
        } else if (primeiro) {
            refazer();
        } else if (!agenda.running) {
            agenda.start();
        }
    }
    // O compactado é cifrado: pede a senha e lista de novo com ela.
    function pedirSegredo() {
        const arquivo = compactado.arquivo;
        const tinha = segredoDe(arquivo) !== null;
        janela.pedirSenha({
            titulo: Arquivos.nomeDe(arquivo), erro: tinha ? "essa senha não abriu o compactado" : "",
            feito: s => {
                painel.segredos[arquivo] = s;
                if (painel.compactado !== null && painel.compactado.arquivo === arquivo)
                    painel.abrirPasta();
            }
        });
    }
    // Refaz o que aparece (filtro e ordem) sem perder a seleção nem a rolagem.
    function refazer() {
        const y = vista.contentY, x = vista.contentX;
        const novo = Arquivos.visiveis(todos, {
            ocultos: prefs.ocultos === true || extra,
            ordem: prefs.ordem || { campo: "nome", inversa: false },
            pastas_antes: prefs.pastas_antes !== false
        });
        itens = novo;
        let escolhidos = null;
        if (aSelecionar.length > 0) {
            escolhidos = Selecao.porCaminhos(novo, aSelecionar);
            if (escolhidos.n === 0)
                escolhidos = null;
            else if (!carregando)
                aSelecionar = [];
        }
        selecao = escolhidos !== null ? escolhidos : Selecao.reancorar(selecao, novo);
        if (escolhidos !== null) {
            const i = escolhidos.cursor;
            Qt.callLater(() => painel.vista.positionViewAtIndex(i, GridView.Contain));
        } else {
            Qt.callLater(() => {
                painel.vista.contentY = y;
                painel.vista.contentX = x;
                painel.vista.returnToBounds();
            });
        }
    }
    onPrefsChanged: {
        if (pronto && !carregando)
            refazer();
    }
    Timer {
        id: agenda
        interval: 160
        onTriggered: painel.refazer()
    }

    // O que o motor diz sozinho.
    function evento(e) {
        if (e.evento === "pasta") {
            if (e.caminho !== vigiado || busca)
                return;
            if (e.sumiu === true) {
                limpar();
                carregando = false;
                erro = { motivo: "sumiu", frase: "" };
            } else if (e.recarregar === true) {
                abrirPasta();
            } else {
                todos = Arquivos.aplicarEvento(todos, e);
                if (!agenda.running)
                    agenda.start();
            }
        } else if (e.evento === "miniatura") {
            if (pedidas[e.caminho] === true && e.arquivo) {
                minis[e.caminho] = e.arquivo;
                miniVersao += 1;
            }
        } else if (e.evento === "lixeira") {
            if (lixeira && !carregando)
                abrirPasta();
        }
    }

    // --- miniaturas: só do que aparece ---
    function querMiniatura(it) {
        if (visao !== "icones" || pedidas[it.caminho] === true || it.tipo === "pasta" || it.quebrado || compactado !== null)
            return;
        const m = it.mime || "";
        if (m.indexOf("image/") !== 0 && m.indexOf("video/") !== 0 && m !== "application/pdf")
            return;
        // Pela rede, só os arquivos pequenos, e só se a pessoa deixa.
        if (Remoto.eRemoto(it.caminho, janela.gvfs) && !Remoto.cabeMiniatura(it, prefs))
            return;
        pedidas[it.caminho] = true;
        filaDeMinis.push(it.caminho);
        if (!juntaMinis.running)
            juntaMinis.start();
    }
    Timer {
        id: juntaMinis
        interval: 80
        onTriggered: {
            const caminhos = painel.filaDeMinis;
            painel.filaDeMinis = [];
            if (caminhos.length === 0)
                return;
            painel.grupoDeMinis = painel.aba.id;
            Motor.pedir({ op: "miniatura", caminhos: caminhos, tamanho: painel.iconePx > 128 ? "large" : "normal", grupo: painel.grupoDeMinis, prioridade: 0 }, null);
        }
    }

    // --- busca ---
    function buscar(texto) {
        const t = texto.trim();
        if (lixeira)
            return;
        if (compactado !== null) {
            if (t !== "")
                janela.avisar("a busca não entra em compactados: extraia, ou busque na pasta");
            return;
        }
        if (t === "") {
            if (busca)
                sairDaBusca();
            return;
        }
        if (carregando && pedido > 0)
            Motor.pedir({ op: "cancelar", pedido: pedido }, null);
        limpar();
        buscaTexto = t;
        carregando = true;
        const f = Arquivos.filtrosDaBusca(filtros, Math.floor(Date.now() / 1000));
        pedido = Motor.pedir({
            op: "buscar", caminho: caminho, nome: buscaConteudo ? "" : t, tipo: buscaTipo,
            tamanho_min: f.tamanho_min, tamanho_max: f.tamanho_max, depois_s: f.depois_s, antes_s: f.antes_s,
            conteudo: buscaConteudo ? t : null, ocultos: prefs.ocultos === true, outros_discos: false
        }, chegou);
    }
    function sairDaBusca() {
        if (!busca)
            return;
        buscaTexto = "";
        abrirPasta();
    }

    // --- abrir ---
    function abrirSelecao() {
        const sel = selecionados();
        if (sel.length === 0)
            return;
        if (lixeira) {
            janela.propriedades(sel);
            return;
        }
        const pastas = sel.filter(Arquivos.ePasta);
        const arquivos = sel.filter(it => !Arquivos.ePasta(it));
        if (compactado !== null) {
            // Lá dentro: uma pasta navega; um arquivo é extraído sozinho e aberto.
            if (sel.length === 1 && pastas.length === 1)
                ir(pastas[0].caminho);
            else if (arquivos.length > 0)
                janela.abrirDeDentro(compactado.arquivo, arquivos, segredoDe(compactado.arquivo));
            return;
        }
        // Enter num compactado abre como pasta (só leitura).
        if (sel.length === 1 && arquivos.length === 1 && janela.extensoes.compactados === true && Arquivos.eCompactado(arquivos[0])) {
            ir(Arquivos.noCompactado(arquivos[0].caminho, ""));
            return;
        }
        if (sel.length === 1 && pastas.length === 1) {
            ir(pastas[0].caminho);
            return;
        }
        for (const p of pastas)
            abas = { abas: Abas.abrir(abas, p.caminho).abas, atual: abas.atual };
        if (arquivos.length > 0)
            janela.abrirArquivos(arquivos);
    }

    // --- teclado ---
    readonly property int colunasDaGrade: Math.max(1, Math.floor(grade.width / grade.cellWidth))
    readonly property int linhasDaCompacta: Math.max(1, Math.floor(colunas.height / colunas.cellHeight))

    function mostrar(i) {
        if (i >= 0)
            vista.positionViewAtIndex(i, GridView.Contain);
    }
    // O cursor mora num lugar só: `selecao.cursor` (reancorado pelo caminho do
    // item a cada vez que a lista é refeita). As visões não têm cursor próprio:
    // só mostram este.
    readonly property int cursor: selecao.cursor
    // A visão de agora, do jeito que o Selecao.destino quer saber.
    function geometria() {
        if (visao === "compacta")
            return { visao: "compacta", linhas: linhasDaCompacta, porPagina: Math.max(1, Math.floor(colunas.width / colunas.cellWidth)) };
        if (visao === "lista")
            return { visao: "lista", porPagina: Math.max(1, Math.floor(linhas.height / 26) - 1) };
        return { visao: "icones", colunas: colunasDaGrade, porPagina: Math.max(1, Math.floor(grade.height / grade.cellHeight)) };
    }
    function andar(tecla, mods) {
        const antes = selecao;
        selecao = Selecao.mover(selecao, itens, tecla, mods, geometria());
        if (selecao !== antes || selecao.cursor >= 0)
            mostrar(selecao.cursor);
    }
    function selecionarTudo() { selecao = Selecao.todos(itens); }
    function inverter() { selecao = Selecao.inverter(selecao, itens); }
    function selecionarPorPadrao(padrao) {
        selecao = Selecao.porPadrao(selecao, itens, padrao);
        mostrar(selecao.cursor);
    }
    function limparSelecao() { selecao = Selecao.nova(); }

    // Onde o menu de contexto abre quando é pedido pelo teclado: junto do
    // item do cursor.
    function pontoDoCursor() {
        const d = selecao.cursor >= 0 ? vista.itemAtIndex(selecao.cursor) : null;
        if (d)
            return d.mapToItem(null, d.width / 2, d.height / 2);
        return area.mapToItem(null, 40, 40);
    }

    // Renomear no lugar: o campo aparece em cima do nome, com o nome sem a
    // extensão selecionado.
    function renomear() {
        const i = selecao.cursor;
        if (lixeira || compactado !== null || i < 0 || i >= total || !Selecao.tem(selecao, itens[i]))
            return;
        selecao = Selecao.so(itens, i);
        mostrar(i);
        const d = vista.itemAtIndex(i);
        if (!d)
            return;
        const it = itens[i];
        const p = d.mapToItem(area, 0, 0);
        if (visao === "icones") {
            editor.x = p.x + 4;
            editor.y = p.y + iconePx + 10;
            editor.width = d.width - 8;
        } else {
            editor.x = p.x + 30;
            editor.y = p.y + (d.height - editor.height) / 2;
            editor.width = (visao === "lista" ? larguraDoNome : d.width) - 36;
        }
        editor.item = it;
        campoDoNome.text = it.nome;
        renomeando = true;
        campoDoNome.forceActiveFocus();
        campoDoNome.select(0, Arquivos.semExtensao(it.nome, Arquivos.ePasta(it)));
    }
    function confirmarNome() {
        const it = editor.item;
        const novo = campoDoNome.text.trim();
        renomeando = false;
        focar();
        if (!it || novo === it.nome)
            return;
        const e = Arquivos.erroDoNome(novo);
        if (e !== "") {
            janela.avisar(e);
            return;
        }
        aSelecionar = [Arquivos.juntar(Arquivos.pai(it.caminho), novo)];
        janela.tarefa({ op: "renomear", caminho: it.caminho, nome: novo });
    }

    // O dono do teclado do painel. É um item que nunca é destruído nem trocado
    // (as visões vêm e vão; ele fica), e o painel é um FocusScope: os dois
    // painéis não disputam o foco entre si. Todas as teclas sem modificador
    // passam por tecla(), aqui e em mais nenhum lugar.
    readonly property bool temTeclado: teclado.activeFocus
    // Devolve o teclado ao painel, se é a vez dele (painel ativo, nenhuma
    // folha, menu ou campo de texto em uso).
    function refocar() {
        if (ativo && janela !== null && janela.livre && !teclado.activeFocus)
            focar();
    }
    onVisaoChanged: {
        refocar();
        Qt.callLater(() => painel.mostrar(painel.selecao.cursor));
    }
    onAtivoChanged: refocar()
    Component.onCompleted: Qt.callLater(refocar)
    Item {
        id: teclado
        focus: true
        Keys.onPressed: event => painel.tecla(event)
    }
    readonly property var setas: {
        const s = {};
        s[Qt.Key_Up] = "cima";
        s[Qt.Key_Down] = "baixo";
        s[Qt.Key_Left] = "esquerda";
        s[Qt.Key_Right] = "direita";
        s[Qt.Key_Home] = "inicio";
        s[Qt.Key_End] = "fim";
        s[Qt.Key_PageUp] = "pagina-cima";
        s[Qt.Key_PageDown] = "pagina-baixo";
        return s;
    }
    // Uma tecla com o painel livre. Aceita o que tratou; o resto segue para
    // os atalhos com modificador da janela.
    function tecla(event) {
        // Com uma folha, um menu ou a barra lateral em uso, as teclas são deles.
        if (janela === null || !janela.livre) {
            event.accepted = false;
            return;
        }
        const ctrl = (event.modifiers & Qt.ControlModifier) !== 0;
        const shift = (event.modifiers & Qt.ShiftModifier) !== 0;
        const alt = (event.modifiers & Qt.AltModifier) !== 0;
        const k = event.key;
        event.accepted = true;
        if (alt) {
            // Alt+setas e Alt+Enter são da janela (atalhos).
            event.accepted = false;
        } else if (setas[k] !== undefined) {
            andar(setas[k], { ctrl: ctrl, shift: shift });
        } else if (k === Qt.Key_Return || k === Qt.Key_Enter) {
            abrirSelecao();
        } else if (k === Qt.Key_Backspace) {
            subir();
        } else if (k === Qt.Key_Tab || k === Qt.Key_Backtab) {
            if (ctrl)
                event.accepted = false;
            else
                janela.alternarPainel();
        } else if (k === Qt.Key_Space) {
            // Ctrl+Espaço liga e desliga o item do cursor. Espaço sozinho é
            // a prévia rápida.
            if (ctrl)
                selecao = Selecao.alternar(selecao, itens, selecao.cursor);
            else if (digitado !== "")
                pular(" ");
            else if (!event.isAutoRepeat)
                janela.acao("previa");
        } else if (k === Qt.Key_Menu) {
            janela.menuDoTeclado();
        } else if (ctrl) {
            event.accepted = false;
        } else if (k === Qt.Key_Delete) {
            if (shift)
                janela.apagarDeVez();
            else
                janela.paraALixeira();
        } else if (k === Qt.Key_Escape) {
            janela.fechar();
        } else if (k >= Qt.Key_F1 && k <= Qt.Key_F12) {
            // As teclas de função sem modificador (F10 com Shift é o menu).
            if (shift)
                event.accepted = false;
            else if (!janela.teclaDeFuncao(k))
                event.accepted = false;
        } else if (event.text.length === 1 && event.text.charCodeAt(0) > 32 && event.text.charCodeAt(0) !== 127) {
            pular(event.text);
        } else {
            event.accepted = false;
        }
    }
    // Digitar para pular ao nome.
    function pular(letra) {
        const de = digitado === "" ? selecao.cursor + 1 : selecao.cursor;
        digitado += letra;
        esquece.restart();
        const i = Arquivos.pular(itens, digitado, de);
        if (i >= 0) {
            selecao = Selecao.so(itens, i);
            mostrar(i);
        }
    }
    Timer {
        id: esquece
        interval: 900
        onTriggered: painel.digitado = ""
    }

    // --- a tela ---
    // A marca do painel ativo (só aparece com dois).
    Rectangle {
        z: 5
        width: parent.width
        height: 2
        color: Theme.copper
        visible: painel.ativo && painel.janela !== null && painel.janela.duploVisivel
    }

    // As abas.
    Rectangle {
        id: faixa
        width: parent.width
        height: 30
        color: Theme.surface
        clip: true

        Row {
            id: fila
            height: parent.height
            readonly property real larguraDaAba: Math.max(70, Math.min(180, (faixa.width - 34) / Math.max(1, painel.abas.abas.length)))

            Repeater {
                model: painel.abas.abas

                Rectangle {
                    id: abaItem
                    required property var modelData
                    required property int index
                    readonly property bool vista: index === painel.abas.atual
                    width: fila.larguraDaAba
                    height: fila.height
                    color: vista ? Theme.ground : areaDaAba.containsMouse || solta.containsDrag ? Theme.wash : Qt.rgba(0, 0, 0, 0)

                    Rectangle {
                        anchors.right: parent.right
                        width: 1
                        height: parent.height
                        color: Theme.hairline
                    }
                    Text {
                        x: 10
                        width: parent.width - 30
                        anchors.verticalCenter: parent.verticalCenter
                        elide: Text.ElideMiddle
                        text: {
                            // A raiz de um servidor mostra o nome dele, não a pasta do gvfs.
                            const r = painel.janela !== null ? Remoto.doRemoto(abaItem.modelData.caminho, painel.janela.remotos) : null;
                            return r !== null && r.raiz === abaItem.modelData.caminho ? r.nome : Abas.titulo(abaItem.modelData, Motor.home);
                        }
                        color: abaItem.vista ? Theme.text : Theme.textDim
                        font.family: Theme.sans
                        font.pixelSize: 12
                    }
                    MouseArea {
                        id: areaDaAba
                        anchors.fill: parent
                        hoverEnabled: true
                        acceptedButtons: Qt.LeftButton | Qt.MiddleButton | Qt.RightButton
                        property real inicio: 0
                        onPressed: mouse => {
                            inicio = mouse.x;
                            painel.janela.ativar(painel.numero);
                            if (mouse.button === Qt.LeftButton)
                                painel.abas = Abas.escolher(painel.abas, abaItem.index);
                        }
                        // Arrastar a aba para o lado a muda de lugar.
                        onReleased: mouse => {
                            if (mouse.button !== Qt.LeftButton || Math.abs(mouse.x - inicio) < fila.larguraDaAba / 2)
                                return;
                            const para = Math.max(0, Math.min(painel.abas.abas.length - 1, abaItem.index + Math.round((mouse.x - inicio) / fila.larguraDaAba)));
                            painel.abas = Abas.mover(painel.abas, abaItem.index, para);
                        }
                        onClicked: mouse => {
                            if (mouse.button === Qt.MiddleButton) {
                                painel.fecharAba(abaItem.index);
                            } else if (mouse.button === Qt.RightButton) {
                                const p = abaItem.mapToItem(null, mouse.x, mouse.y);
                                painel.janela.menuDaAba(painel.numero, abaItem.index, p.x, p.y);
                            }
                        }
                        onDoubleClicked: painel.duplicarAba(abaItem.index)
                    }
                    Text {
                        visible: painel.abas.abas.length > 1 && (abaItem.vista || areaDaAba.containsMouse || fechaArea.containsMouse)
                        anchors.right: parent.right
                        anchors.rightMargin: 8
                        anchors.verticalCenter: parent.verticalCenter
                        text: "×"
                        color: fechaArea.containsMouse ? Theme.copper : Theme.textDim
                        font.pixelSize: 14
                        MouseArea {
                            id: fechaArea
                            anchors.fill: parent
                            anchors.margins: -4
                            hoverEnabled: true
                            onClicked: painel.fecharAba(abaItem.index)
                        }
                    }
                    // Soltar arquivos numa aba manda para a pasta dela.
                    DropArea {
                        id: solta
                        anchors.fill: parent
                        keys: ["text/uri-list"]
                        onDropped: drop => {
                            const caminhos = Arquivos.lerListaDeUris(drop.hasUrls ? drop.urls.join("\n") : drop.getDataAsString("text/uri-list"));
                            if (caminhos.length === 0)
                                return;
                            drop.accept(drop.proposedAction);
                            painel.janela.soltar(caminhos, abaItem.modelData.caminho, drop.proposedAction);
                        }
                    }
                }
            }
            Text {
                width: 30
                height: fila.height
                horizontalAlignment: Text.AlignHCenter
                verticalAlignment: Text.AlignVCenter
                text: "+"
                color: maisArea.containsMouse ? Theme.copper : Theme.textDim
                font.pixelSize: 16
                MouseArea {
                    id: maisArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        painel.janela.ativar(painel.numero);
                        painel.novaAba("");
                    }
                }
            }
        }
        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: 1
            color: Theme.hairline
        }
    }

    // A faixa da lixeira e da busca: o que é isto, e as ações daqui.
    Rectangle {
        id: aviso
        anchors.top: faixa.bottom
        width: parent.width
        height: visible ? 34 : 0
        visible: painel.lixeira || painel.busca || painel.compactado !== null
        color: Theme.raised

        Text {
            x: 12
            width: parent.width - acoes.width - 30
            anchors.verticalCenter: parent.verticalCenter
            elide: Text.ElideRight
            text: painel.lixeira ? "Lixeira  ·  " + Arquivos.plural(painel.total, "item", "itens")
                : painel.compactado !== null ? "Dentro de " + Arquivos.nomeDe(painel.compactado.arquivo) + "  ·  só leitura"
                : (painel.carregando ? "Buscando «" : "Resultados de «") + painel.buscaTexto + "»  ·  " + painel.total
            color: Theme.textSoft
            font.family: Theme.sans
            font.pixelSize: 12
        }
        Row {
            id: acoes
            anchors.right: parent.right
            anchors.rightMargin: 8
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6
            Botao {
                visible: painel.lixeira && painel.selecao.n > 0
                texto: "restaurar"
                forte: true
                onAcionado: painel.janela.acao("restaurar")
            }
            Botao {
                visible: painel.lixeira && painel.selecao.n > 0 && painel.width > 460
                texto: "apagar de vez"
                onAcionado: painel.janela.acao("lixeira-apagar")
            }
            Botao {
                visible: painel.lixeira
                texto: "esvaziar"
                ligado: painel.total > 0
                onAcionado: painel.janela.acao("esvaziar")
            }
            Botao {
                visible: painel.compactado !== null
                texto: "extrair tudo"
                forte: true
                onAcionado: painel.janela.acao("extrair-tudo")
            }
            Botao {
                visible: painel.compactado !== null && painel.width > 460
                texto: "extrair para…"
                onAcionado: {
                    painel.limparSelecao();
                    painel.janela.acao("extrair-para");
                }
            }
            Botao {
                visible: painel.busca
                texto: "filtros"
                forte: painel.filtrosAbertos || Arquivos.temFiltro(painel.filtros)
                onAcionado: painel.filtrosAbertos = !painel.filtrosAbertos
            }
            Botao {
                visible: painel.busca && painel.width > 520
                texto: painel.buscaTipo === "qualquer" ? "tipo: tudo" : "tipo: " + painel.buscaTipo
                onAcionado: {
                    const tipos = ["qualquer", "pasta", "arquivo", "imagem", "video", "audio", "texto", "documento"];
                    painel.buscaTipo = tipos[(tipos.indexOf(painel.buscaTipo) + 1) % tipos.length];
                    painel.buscar(painel.buscaTexto);
                }
            }
            Botao {
                visible: painel.busca
                texto: "no conteúdo"
                forte: painel.buscaConteudo
                onAcionado: {
                    painel.buscaConteudo = !painel.buscaConteudo;
                    painel.buscar(painel.buscaTexto);
                }
            }
            Botao {
                visible: painel.busca
                texto: "esc volta"
                onAcionado: painel.janela.fecharBusca()
            }
        }
    }

    // Os filtros da busca: tamanho (mínimo e máximo) e data. Enter aplica.
    Rectangle {
        id: faixaDosFiltros
        anchors.top: aviso.bottom
        width: parent.width
        height: visible ? 36 : 0
        visible: painel.busca && painel.filtrosAbertos
        color: Theme.surface
        clip: true

        Row {
            x: 12
            height: parent.height
            spacing: 8

            Rotulo {
                anchors.verticalCenter: parent.verticalCenter
                text: "tamanho"
            }
            Campo {
                id: campoMin
                anchors.verticalCenter: parent.verticalCenter
                width: 54
                mono: true
                dica: "de"
                onAceito: painel.filtrar({ min: campoMin.text.trim() })
            }
            Campo {
                id: campoMax
                anchors.verticalCenter: parent.verticalCenter
                width: 54
                mono: true
                dica: "até"
                onAceito: painel.filtrar({ max: campoMax.text.trim() })
            }
            Botao {
                anchors.verticalCenter: parent.verticalCenter
                implicitWidth: 44
                texto: painel.filtros.unidade
                onAcionado: {
                    const u = ["KB", "MB", "GB"];
                    painel.filtrar({ min: campoMin.text.trim(), max: campoMax.text.trim(), unidade: u[(u.indexOf(painel.filtros.unidade) + 1) % u.length] });
                }
            }
            Rotulo {
                anchors.verticalCenter: parent.verticalCenter
                text: "data"
            }
            Botao {
                anchors.verticalCenter: parent.verticalCenter
                texto: {
                    for (const p of Arquivos.PERIODOS)
                        if (p[0] === painel.filtros.periodo)
                            return p[1];
                    return "qualquer data";
                }
                forte: painel.filtros.periodo !== ""
                onAcionado: {
                    const ids = Arquivos.PERIODOS.map(p => p[0]);
                    painel.filtrar({ periodo: ids[(ids.indexOf(painel.filtros.periodo) + 1) % ids.length], de: campoDe.text.trim(), ate: campoAte.text.trim() });
                }
            }
            Campo {
                id: campoDe
                anchors.verticalCenter: parent.verticalCenter
                visible: painel.filtros.periodo === "datas"
                width: 92
                mono: true
                dica: "AAAA-MM-DD"
                onAceito: painel.filtrar({ de: campoDe.text.trim(), ate: campoAte.text.trim() })
            }
            Campo {
                id: campoAte
                anchors.verticalCenter: parent.verticalCenter
                visible: painel.filtros.periodo === "datas"
                width: 92
                mono: true
                dica: "até AAAA-MM-DD"
                onAceito: painel.filtrar({ de: campoDe.text.trim(), ate: campoAte.text.trim() })
            }
            Botao {
                anchors.verticalCenter: parent.verticalCenter
                visible: Arquivos.temFiltro(painel.filtros)
                texto: "limpar"
                onAcionado: {
                    campoMin.text = "";
                    campoMax.text = "";
                    campoDe.text = "";
                    campoAte.text = "";
                    painel.filtrar({ min: "", max: "", periodo: "", de: "", ate: "" });
                    painel.focar();
                }
            }
        }
        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: 1
            color: Theme.hairline
        }
    }

    // O cabeçalho da lista: clicar ordena; a beira de cada coluna ajusta a largura.
    Rectangle {
        id: cabecalho
        anchors.top: faixaDosFiltros.bottom
        width: parent.width
        height: visible ? 24 : 0
        visible: painel.visao === "lista" && painel.erro === null
        color: Theme.surface

        Row {
            height: parent.height
            Repeater {
                model: [{ id: "nome", titulo: "nome", largura: painel.larguraDoNome }].concat(painel.cols)

                Item {
                    id: coluna
                    required property var modelData
                    readonly property string campo: modelData.id === "permissoes" ? "" : modelData.id
                    readonly property bool ordenada: campo !== "" && (painel.prefs.ordem || {}).campo === campo
                    width: modelData.largura
                    height: cabecalho.height

                    Rotulo {
                        x: coluna.modelData.id === "nome" ? 32 : 8
                        anchors.verticalCenter: parent.verticalCenter
                        text: coluna.modelData.titulo + (coluna.ordenada ? ((painel.prefs.ordem || {}).inversa ? "  ↓" : "  ↑") : "")
                        color: coluna.ordenada ? Theme.copper : Theme.textFaint
                    }
                    MouseArea {
                        anchors.fill: parent
                        enabled: coluna.campo !== ""
                        cursorShape: Qt.PointingHandCursor
                        onClicked: painel.janela.ordenarPor(coluna.campo)
                    }
                    // A beira esquerda da coluna: arrastar muda a largura dela.
                    MouseArea {
                        visible: coluna.modelData.id !== "nome"
                        x: -4
                        width: 8
                        height: parent.height
                        cursorShape: Qt.SplitHCursor
                        property real inicio: 0
                        property real largura: 0
                        onPressed: mouse => {
                            inicio = mapToItem(painel, mouse.x, 0).x;
                            largura = coluna.modelData.largura;
                        }
                        onReleased: mouse => {
                            const agora = mapToItem(painel, mouse.x, 0).x;
                            painel.janela.larguraDaColuna(coluna.modelData.id, Math.max(40, Math.min(600, Math.round(largura - (agora - inicio)))));
                        }
                    }
                    Rectangle {
                        visible: coluna.modelData.id !== "nome"
                        width: 1
                        height: parent.height
                        color: Theme.hairline
                    }
                }
            }
        }
        Rectangle {
            anchors.bottom: parent.bottom
            width: parent.width
            height: 1
            color: Theme.hairline
        }
    }

    // A área dos arquivos.
    Item {
        id: area
        anchors.top: cabecalho.bottom
        anchors.bottom: parent.bottom
        width: parent.width
        clip: true

        GridView {
            id: grade
            anchors.fill: parent
            visible: painel.visao === "icones" && painel.erro === null
            model: visible ? painel.total : 0
            cellWidth: painel.iconePx + 44
            cellHeight: painel.iconePx + (painel.extra ? 64 : 50)
            boundsBehavior: Flickable.StopAtBounds
            reuseItems: true
            cacheBuffer: 200
            delegate: Cartao {
                dono: painel
                width: grade.cellWidth
                height: grade.cellHeight
            }
        }
        ListView {
            id: linhas
            anchors.fill: parent
            visible: painel.visao === "lista" && painel.erro === null
            model: visible ? painel.total : 0
            boundsBehavior: Flickable.StopAtBounds
            reuseItems: true
            cacheBuffer: 200
            delegate: Linha {
                dono: painel
                width: linhas.width
                height: 26
            }
        }
        GridView {
            id: colunas
            anchors.fill: parent
            visible: painel.visao === "compacta" && painel.erro === null
            model: visible ? painel.total : 0
            flow: GridView.FlowTopToBottom
            cellWidth: 250
            cellHeight: 24
            boundsBehavior: Flickable.StopAtBounds
            reuseItems: true
            delegate: Linha {
                dono: painel
                compacta: true
                width: colunas.cellWidth
                height: colunas.cellHeight
            }
        }

        // O mouse inteiro do painel: clique, duplo clique, menu, retângulo de
        // seleção e o começo do arrastar. A roda passa para a visão (com Ctrl,
        // muda o tamanho dos ícones).
        MouseArea {
            id: mouse
            anchors.fill: parent
            acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton
            property int sobre: -1
            property int pendente: -1
            property bool retangulo: false
            property real ox: 0
            property real oy: 0
            property real px: 0
            property real py: 0
            property real ix: 0
            property real iy: 0
            property var base: null

            // O índice do item nesse ponto, ou -1 no vazio. Na lista só o nome
            // conta: o resto da linha começa um retângulo.
            function acerto(x, y) {
                const v = painel.vista;
                const i = v.indexAt(x + v.contentX, y + v.contentY);
                if (i < 0)
                    return -1;
                if (painel.visao === "lista")
                    return x <= painel.larguraDoNome ? i : -1;
                if (painel.visao === "icones") {
                    const dentro = (x + v.contentX) % grade.cellWidth;
                    return dentro < 6 || dentro > grade.cellWidth - 6 ? -1 : i;
                }
                return (x + v.contentX) % colunas.cellWidth > colunas.cellWidth - 8 ? -1 : i;
            }
            function linhaEm(x, y) {
                const v = painel.vista;
                return v.indexAt(Math.min(x, 40) + v.contentX, y + v.contentY);
            }

            onPressed: m => {
                painel.janela.ativar(painel.numero);
                painel.focar();
                if (painel.renomeando)
                    painel.confirmarNome();
                const mods = { ctrl: (m.modifiers & Qt.ControlModifier) !== 0, shift: (m.modifiers & Qt.ShiftModifier) !== 0 };
                const i = acerto(m.x, m.y);
                sobre = i;
                pendente = -1;
                retangulo = false;
                px = m.x;
                py = m.y;
                ix = m.x;
                iy = m.y;
                if (m.button === Qt.RightButton) {
                    if (i >= 0 && !Selecao.tem(painel.selecao, painel.itens[i]))
                        painel.selecao = Selecao.so(painel.itens, i);
                    else if (i < 0)
                        painel.selecao = Selecao.nova();
                    const p = mouse.mapToItem(null, m.x, m.y);
                    painel.janela.menuDoPainel(p.x, p.y);
                    return;
                }
                if (m.button === Qt.MiddleButton) {
                    if (i >= 0 && Arquivos.ePasta(painel.itens[i]))
                        painel.abas = { abas: Abas.abrir(painel.abas, painel.itens[i].caminho).abas, atual: painel.abas.atual };
                    return;
                }
                if (i >= 0) {
                    if (mods.ctrl || mods.shift || !Selecao.tem(painel.selecao, painel.itens[i]))
                        painel.selecao = Selecao.clicar(painel.selecao, painel.itens, i, mods);
                    else
                        pendente = i;
                    return;
                }
                // No vazio: começa o retângulo (com Ctrl ou Shift, soma ao que havia).
                base = mods.ctrl || mods.shift ? painel.selecao : Selecao.nova();
                ox = m.x + painel.vista.contentX;
                oy = m.y + painel.vista.contentY;
                retangulo = true;
                if (!mods.ctrl && !mods.shift)
                    painel.selecao = Selecao.nova();
            }
            // O retângulo, do ponto em que começou (no conteúdo) até o ponteiro:
            // marca o que ele cobre. Roda a cada movimento e a cada passo da
            // rolagem automática.
            function marcar() {
                const v = painel.vista;
                const r = { x: ox, y: oy, l: px + v.contentX - ox, a: py + v.contentY - oy };
                if (Math.abs(r.l) < 4 && Math.abs(r.a) < 4)
                    return;
                let indices;
                if (painel.visao === "icones")
                    indices = Selecao.noRetangulo({ n: painel.total, colunas: painel.colunasDaGrade, larguraDaCelula: grade.cellWidth, alturaDaCelula: grade.cellHeight }, r);
                else if (painel.visao === "lista")
                    indices = Selecao.noRetangulo({ n: painel.total, colunas: 1, larguraDaCelula: linhas.width, alturaDaCelula: 26 }, r);
                else
                    indices = Selecao.noRetangulo({ n: painel.total, colunas: painel.linhasDaCompacta, larguraDaCelula: colunas.cellHeight, alturaDaCelula: colunas.cellWidth },
                        { x: r.y, y: r.x, l: r.a, a: r.l });
                painel.selecao = Selecao.comIndices(base, painel.itens, indices);
            }
            // Com o retângulo aberto e o ponteiro na beira (ou fora), a lista
            // rola sozinha, mais depressa quanto mais longe.
            Timer {
                interval: 30
                repeat: true
                running: mouse.retangulo && mouse.pressed
                onTriggered: {
                    const v = painel.vista;
                    const deitada = painel.visao === "compacta";
                    const p = deitada ? mouse.px : mouse.py;
                    const lado = deitada ? area.width : area.height;
                    const beira = 28;
                    let passo = 0;
                    if (p < beira)
                        passo = -Math.min(40, (beira - p) / 2 + 2);
                    else if (p > lado - beira)
                        passo = Math.min(40, (p - (lado - beira)) / 2 + 2);
                    if (passo === 0)
                        return;
                    if (deitada) {
                        const x = Math.max(v.originX, Math.min(v.originX + Math.max(0, v.contentWidth - v.width), v.contentX + passo));
                        if (x === v.contentX)
                            return;
                        v.contentX = x;
                    } else {
                        const y = Math.max(v.originY, Math.min(v.originY + Math.max(0, v.contentHeight - v.height), v.contentY + passo));
                        if (y === v.contentY)
                            return;
                        v.contentY = y;
                    }
                    mouse.marcar();
                }
            }
            onPositionChanged: m => {
                if (!(m.buttons & Qt.LeftButton))
                    return;
                px = m.x;
                py = m.y;
                if (retangulo) {
                    marcar();
                    return;
                }
                // Arrastar o que está selecionado: para outra pasta, outro
                // painel, uma aba, a barra lateral ou outro janela.
                if (sobre >= 0 && !arrasto.Drag.active && (Math.abs(m.x - ix) > 10 || Math.abs(m.y - iy) > 10) && painel.selecao.n > 0 && !painel.lixeira
                    && painel.compactado === null) {
                    pendente = -1;
                    arrasto.caminhos = Selecao.caminhos(painel.selecao, painel.itens);
                    arrasto.Drag.active = true;
                }
            }
            onReleased: m => {
                const era = retangulo;
                retangulo = false;
                if (m.button !== Qt.LeftButton)
                    return;
                if (pendente >= 0) {
                    painel.selecao = Selecao.so(painel.itens, pendente);
                    pendente = -1;
                } else if (era && Math.abs(m.x + painel.vista.contentX - ox) < 4 && Math.abs(m.y + painel.vista.contentY - oy) < 4) {
                    // Um clique fora do nome, na lista, ainda escolhe a linha.
                    const i = painel.visao === "lista" ? linhaEm(m.x, m.y) : -1;
                    if (i >= 0)
                        painel.selecao = Selecao.so(painel.itens, i);
                }
            }
            onClicked: m => {
                if (m.button === Qt.LeftButton && painel.prefs.clique === "simples" && sobre >= 0
                    && !(m.modifiers & (Qt.ControlModifier | Qt.ShiftModifier)))
                    painel.abrirSelecao();
            }
            onDoubleClicked: m => {
                if (m.button !== Qt.LeftButton || painel.prefs.clique === "simples")
                    return;
                const i = sobre >= 0 ? sobre : (painel.visao === "lista" ? linhaEm(m.x, m.y) : -1);
                if (i >= 0) {
                    painel.selecao = Selecao.so(painel.itens, i);
                    painel.abrirSelecao();
                }
            }
            onWheel: wheel => {
                if (wheel.modifiers & Qt.ControlModifier) {
                    painel.janela.zoom(wheel.angleDelta.y > 0 ? 1 : -1);
                    wheel.accepted = true;
                } else if (painel.visao === "compacta") {
                    // A compacta rola para os lados: a roda vertical também.
                    const v = colunas;
                    v.contentX = Math.max(0, Math.min(Math.max(0, v.contentWidth - v.width), v.contentX - (wheel.angleDelta.y || wheel.angleDelta.x)));
                    wheel.accepted = true;
                } else {
                    wheel.accepted = false;
                }
            }
        }

        // O que está sendo arrastado: os URIs da seleção, para dentro da janela
        // e para outros apps.
        Item {
            id: arrasto
            property var caminhos: []
            Drag.dragType: Drag.Automatic
            Drag.supportedActions: Qt.CopyAction | Qt.MoveAction | Qt.LinkAction
            Drag.proposedAction: Qt.MoveAction
            Drag.keys: ["text/uri-list"]
            Drag.mimeData: ({ "text/uri-list": Arquivos.listaDeUris(arrasto.caminhos) })
            Drag.onDragFinished: arrasto.Drag.active = false
        }

        // Soltar aqui: na pasta sob o ponteiro, ou na pasta do painel.
        DropArea {
            anchors.fill: parent
            keys: ["text/uri-list"]
            enabled: painel.erro === null && !painel.busca && painel.compactado === null
            onPositionChanged: drag => {
                const i = mouse.acerto(drag.x, drag.y);
                painel.alvoDeSoltar = i >= 0 && Arquivos.ePasta(painel.itens[i]) && !Selecao.tem(painel.selecao, painel.itens[i]) ? i : -1;
            }
            onExited: painel.alvoDeSoltar = -1
            onDropped: drop => {
                const i = painel.alvoDeSoltar;
                painel.alvoDeSoltar = -1;
                const caminhos = Arquivos.lerListaDeUris(drop.hasUrls ? drop.urls.join("\n") : drop.getDataAsString("text/uri-list"));
                if (caminhos.length === 0)
                    return;
                drop.accept(drop.proposedAction);
                painel.janela.ativar(painel.numero);
                painel.janela.soltar(caminhos, i >= 0 ? painel.itens[i].caminho : painel.caminho, drop.proposedAction);
            }
        }

        // O retângulo de seleção.
        Rectangle {
            visible: mouse.retangulo && (Math.abs(width) > 3 || Math.abs(height) > 3)
            x: Math.min(mouse.ox - painel.vista.contentX, mouse.px)
            y: Math.min(mouse.oy - painel.vista.contentY, mouse.py)
            width: Math.abs(mouse.px - (mouse.ox - painel.vista.contentX))
            height: Math.abs(mouse.py - (mouse.oy - painel.vista.contentY))
            color: Theme.copperA(0.1)
            border.width: 1
            border.color: Theme.copper
        }

        // O campo do renomear no lugar.
        Rectangle {
            id: editor
            property var item: null
            visible: painel.renomeando
            height: 22
            color: Theme.ground
            border.width: 1
            border.color: Theme.copper
            z: 3

            TextInput {
                id: campoDoNome
                anchors.fill: parent
                anchors.leftMargin: 4
                anchors.rightMargin: 4
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.text
                selectionColor: Theme.copper
                font.family: Theme.sans
                font.pixelSize: 12
                clip: true
                selectByMouse: true
                Keys.onReturnPressed: painel.confirmarNome()
                Keys.onEnterPressed: painel.confirmarNome()
                Keys.onEscapePressed: {
                    painel.renomeando = false;
                    painel.focar();
                }
                onActiveFocusChanged: {
                    if (!activeFocus && painel.renomeando)
                        painel.renomeando = false;
                }
            }
        }

        // As barras de rolagem: mostram onde se está e dão para arrastar. A
        // compacta rola para os lados.
        Rolagem {
            visible: painel.visao !== "compacta" && painel.erro === null && conteudo > aVista + 1
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            alvo: painel.visao === "compacta" ? null : painel.vista
        }
        Rolagem {
            visible: painel.visao === "compacta" && painel.erro === null && conteudo > aVista + 1
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            horizontal: true
            alvo: colunas
        }

        // Pasta vazia, lendo, ou sem resultado: a janela nunca fica em branco.
        Column {
            anchors.centerIn: parent
            width: Math.min(parent.width - 40, 380)
            spacing: 8
            visible: painel.erro === null && painel.total === 0

            Text {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                text: !painel.pronto || painel.carregando ? (painel.busca ? "Buscando…" : "Lendo a pasta…")
                    : painel.busca ? "Nada encontrado"
                    : painel.lixeira ? "A lixeira está vazia"
                    : painel.todos.length > 0 ? "Só há itens ocultos aqui" : "Pasta vazia"
                color: Theme.textDim
                font.family: Theme.serif
                font.pixelSize: 20
                font.weight: Font.Light
            }
            Text {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                visible: painel.pronto && !painel.carregando
                text: painel.busca ? "Nenhum nome casa com «" + painel.buscaTexto + "» nesta pasta nem nas de dentro."
                    : painel.lixeira ? "O que você mandar para a lixeira aparece aqui, e dá para restaurar."
                    : painel.todos.length > 0 ? "Ctrl+H mostra os ocultos." : "Ctrl+Shift+N cria uma pasta; Ctrl+V cola."
                color: Theme.textFaint
                font.family: Theme.sans
                font.pixelSize: 12
            }
        }

        // Não deu para abrir: diz o que houve e oferece sair daqui.
        Column {
            anchors.centerIn: parent
            width: Math.min(parent.width - 40, 420)
            spacing: 10
            visible: painel.erro !== null
            z: 4

            Text {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                text: painel.erro ? Arquivos.erroDePasta(painel.erro.motivo, painel.erro.frase).titulo : ""
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
            Text {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                text: painel.erro ? Arquivos.erroDePasta(painel.erro.motivo, painel.erro.frase).frase : ""
                color: Theme.textSoft
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Text {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                elide: Text.ElideMiddle
                text: painel.lixeira ? "" : painel.caminho
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 11
            }
            Row {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 8
                Botao {
                    visible: Abas.podeSubir(painel.aba)
                    texto: "subir um nível"
                    forte: true
                    onAcionado: painel.subir()
                }
                Botao {
                    texto: "pasta pessoal"
                    onAcionado: painel.ir(Motor.home)
                }
                Botao {
                    texto: "tentar de novo"
                    onAcionado: painel.abrirPasta()
                }
            }
        }
    }
}
