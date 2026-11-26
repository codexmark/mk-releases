import QtQuick
import QtQuick.Layouts
import Quickshell
import "Arquivos.js" as Arquivos
import "Selecao.js" as Selecao
import "Abas.js" as Abas
import "Menu.js" as Menu
import "Lote.js" as Lote
import "Remoto.js" as Remoto

// A janela do MkFiles: a barra de cima, a barra lateral, um ou dois painéis
// de arquivos, a área de tarefas e a barra de estado; por cima, o menu de
// contexto e as folhas. Aqui moram os atalhos, as ações (o que cada entrada
// de menu e cada tecla faz) e a conversa com o motor que não é de um painel
// só. A janela só mostra o que o motor diz e só executa o argv que ele devolve.
FocusScope {
    id: app

    // Fica falso até as preferências e o que abrir chegarem do motor.
    property bool pronto: false
    property var prefs: ({
        visao: "icones", visao_por_pasta: {}, ordem: { campo: "nome", inversa: false }, ocultos: false, icone_px: 64,
        pastas_antes: true, confirmar_lixeira: false, clique: "duplo", colunas: {}
    })
    property var estadoGravado: ({})
    // As extensões das fases seguintes que o motor diz ter (`sobre`).
    property var extensoes: ({})
    property var lugares: []
    property var favoritos: []
    property var discos: []
    property int naLixeira: 0

    // Os painéis: qual está ativo, e se são dois.
    property int ativo: 0
    property bool duplo: false
    readonly property bool duploVisivel: duplo && width >= 900
    readonly property bool estreita: width < 720
    property real lateralPx: 200
    readonly property var painel: ativo === 1 && duploVisivel ? painelB : painelA
    readonly property var outro: painel === painelA ? painelB : painelA

    // A folha aberta ("" = nenhuma) e o que cada uma mostra.
    property string folha: ""
    property var pergunta: null
    property var nomeDados: null
    property int pedidoDasProps: 0
    property var lugarDoMenu: null
    // A rede: a raiz do gvfs (do `sobre`), os remotos montados, o que a rede
    // local mostra, os hosts do ssh, a conexão em andamento ({ pedido, uri,
    // naFolha }) e a pergunta que o servidor fez. Senha nenhuma mora aqui.
    property string gvfs: ""
    property var remotos: []
    property var rede: []
    property var hostsDoSsh: []
    property var conexao: null
    property var perguntaRemota: null
    property var abaDoMenu: null
    property bool naLateral: false

    property var tarefas: []
    property var conflitos: []
    property var historico: ({ desfazer: null, refazer: null })
    property string aviso: ""

    // Os atalhos da janela valem quando nada está por cima e ninguém digita.
    readonly property bool livre: folha === "" && !menu.aberto && !barra.comFoco && !painelA.renomeando && !painelB.renomeando && !naLateral
        && !painelA.filtrando && !painelB.filtrando
    // A prévia rápida, o compactar e a senha de um compactado.
    property int pedidoDaPrevia: 0
    property int pedidoDaContagem: 0
    property var compactarDados: null
    property var segredoDados: null
    readonly property string soLeitura: "um compactado aberto como pasta é só de leitura: extraia o que quiser mexer"

    // --- início ---------------------------------------------------------------
    Component.onCompleted: {
        Motor.pedir({ op: "sobre" }, r => {
            if (!r.erro) {
                app.extensoes = r.extensoes || {};
                app.gvfs = r.gvfs || "";
            }
        });
        lerRemotos(false);
        Motor.pedir({ op: "prefs" }, r => {
            if (!r.erro) {
                app.prefs = Object.assign({}, app.prefs, r.prefs || {});
                app.estadoGravado = r.estado || {};
            }
            app.resolverPedidos();
        });
        lerLugares();
        Motor.pedir({ op: "discos" }, r => {
            if (!r.erro)
                app.discos = r.discos || [];
        });
        Motor.pedir({ op: "vigiar-discos", ligado: true }, null);
        Motor.pedir({ op: "lixeira-estado" }, r => {
            if (!r.erro)
                app.naLixeira = r.itens || 0;
        });
    }
    // `mkfiles pasta arquivo…`: o motor diz o que é cada caminho pedido.
    function resolverPedidos() {
        const caminhos = Abas.pedidosDe(Motor.abrir);
        if (caminhos.length === 0) {
            iniciar([]);
            return;
        }
        const vistos = [];
        let faltam = caminhos.length;
        caminhos.forEach((c, i) => {
            Motor.pedir({ op: "item", caminho: c }, r => {
                if (!r.erro && r.item)
                    vistos[i] = { caminho: c, pasta: Arquivos.ePasta(r.item) };
                faltam -= 1;
                if (faltam === 0)
                    app.iniciar(vistos.filter(v => !!v));
            });
        });
    }
    function iniciar(pedidos) {
        const i = Abas.inicial(pedidos, estadoGravado, Motor.home);
        painelA.abas = i.paineis[0];
        painelB.abas = i.paineis[1];
        painelA.aSelecionar = i.selecionar[painelA.caminho] || [];
        duplo = pedidos.length === 0 && estadoGravado.duplo === true;
        ativo = pedidos.length === 0 && estadoGravado.ativo === 1 ? 1 : 0;
        lateralPx = estadoGravado.lateral_px || 200;
        pronto = true;
        painel.focar();
    }
    function lerLugares() {
        Motor.pedir({ op: "lugares" }, r => {
            if (r.erro)
                return;
            app.lugares = r.lugares || [];
            app.favoritos = r.favoritos || [];
        });
    }

    // --- preferências e estado -------------------------------------------------
    function preferir(mudanca) {
        prefs = Object.assign({}, prefs, mudanca);
        Motor.pedir({ op: "prefs-gravar", prefs: mudanca }, null);
    }
    function guardarEstado() {
        if (pronto)
            guarda.restart();
    }
    Timer {
        id: guarda
        interval: 800
        onTriggered: Motor.pedir({ op: "estado-gravar", estado: Abas.paraEstado([painelA.abas, painelB.abas], app.ativo, app.duplo, app.lateralPx) }, null)
    }
    onDuploChanged: guardarEstado()
    onAtivoChanged: guardarEstado()

    function escolherVisao(v) {
        const porPasta = Object.assign({}, prefs.visao_por_pasta || {});
        if (!painel.lixeira && painel.compactado === null)
            porPasta[painel.caminho] = v;
        preferir({ visao: v, visao_por_pasta: porPasta });
    }
    function ordenarPor(campo) {
        const o = prefs.ordem || {};
        preferir({ ordem: { campo: campo, inversa: o.campo === campo ? !o.inversa : false } });
    }
    function larguraDaColuna(id, px) {
        const c = Object.assign({}, prefs.colunas || {});
        c[id] = px;
        preferir({ colunas: c });
    }
    function zoom(delta) {
        const px = delta === 0 ? 64 : Math.max(32, Math.min(256, (prefs.icone_px || 64) + delta * 16));
        if (px !== prefs.icone_px)
            preferir({ icone_px: px });
    }

    // --- avisos ----------------------------------------------------------------
    function avisar(texto) {
        aviso = texto || "";
        if (aviso !== "")
            apagaAviso.restart();
    }
    Timer {
        id: apagaAviso
        interval: 6000
        onTriggered: app.aviso = ""
    }

    // --- painéis ---------------------------------------------------------------
    function ativar(numero) {
        ativo = numero === 1 && duploVisivel ? 1 : 0;
        naLateral = false;
    }
    function alternarPainel() {
        if (!duploVisivel)
            return;
        ativo = ativo === 1 ? 0 : 1;
        painel.focar();
    }
    function ligarDuplo() {
        if (width < 900 && !duplo) {
            avisar("a janela está estreita demais para dois painéis");
            return;
        }
        duplo = !duplo;
        if (!duplo)
            ativo = 0;
        painel.focar();
    }
    // Abre uma pasta no outro painel (ligando o painel duplo, se preciso).
    function noOutro(caminho) {
        if (!duploVisivel) {
            if (width < 900) {
                painel.novaAba(caminho);
                return;
            }
            duplo = true;
        }
        const alvo = painel === painelA ? painelB : painelA;
        alvo.ir(caminho);
    }
    function fecharBusca() {
        barra.fecharBusca();
        painel.sairDaBusca();
        painel.focar();
    }

    // --- executar: só o argv que o motor devolveu, ou o próprio lançador ---------
    function executar(argv) {
        if (!(argv instanceof Array) || argv.length === 0)
            return;
        if (Motor.exemplo !== "") {
            avisar("exemplo: abriria " + argv.join(" "));
            return;
        }
        Quickshell.execDetached(argv);
    }
    function novaJanela(caminho) {
        executar([Quickshell.shellPath("../bin/mkfiles"), "--nova-janela", Arquivos.pastaReal(caminho || painel.caminho)]);
    }
    function lancar(r) {
        if (r.erro) {
            avisar(r.erro);
            return;
        }
        for (const l of r.lancar || [])
            executar(l.argv);
        if (r.sem_app && r.sem_app.length > 0)
            abrirCom([r.sem_app[0].caminho], r.sem_app[0].mime, Arquivos.nomeDe(r.sem_app[0].caminho));
    }
    function abrirArquivos(itens) {
        Motor.pedir({ op: "abrir", caminhos: itens.map(it => it.caminho), app: null }, lancar);
    }
    function terminal(caminho) {
        Motor.pedir({ op: "terminal", caminho: caminho }, r => {
            if (r.erro)
                app.avisar(r.erro);
            else
                app.executar(r.argv);
        });
    }

    // --- operações (tarefas do motor) --------------------------------------------
    function tarefa(pedido) {
        Motor.pedir(pedido, r => {
            if (r.erro)
                app.avisar(r.erro);
        });
    }
    function caminhosDaSelecao() {
        return Selecao.caminhos(painel.selecao, painel.itens);
    }
    function copiar(recortar) {
        const c = caminhosDaSelecao();
        if (c.length === 0 || painel.lixeira)
            return;
        if (painel.compactado !== null) {
            // De dentro de um compactado só se copia: vai uma marca, e o colar
            // extrai o que foi copiado.
            if (recortar) {
                avisar(soLeitura);
                return;
            }
            const internos = painel.selecionados().map(it => it.interno);
            Motor.pedir({ op: "area-por", tipo: "text/plain", texto: Arquivos.marcaDoCompactado(painel.compactado.arquivo, internos) }, r => {
                app.avisar(r.erro || "copiado de dentro do compactado: " + Arquivos.plural(internos.length, "item", "itens"));
            });
            return;
        }
        Motor.pedir(Arquivos.pedidoDeCopia(c, recortar), r => {
            app.avisar(r.erro || (recortar ? "recortado: " : "copiado: ") + Arquivos.plural(c.length, "item", "itens"));
        });
    }
    // O endereço (sftp://…) do que está selecionado numa pasta de rede, ou da
    // própria pasta.
    function copiarEndereco() {
        const c = caminhosDaSelecao();
        const lista = (c.length > 0 ? c : [painel.caminho]).map(x => Remoto.endereco(x, remotos)).filter(x => x !== null);
        if (lista.length === 0) {
            avisar("isto não está num servidor conectado");
            return;
        }
        Motor.pedir({ op: "area-por", tipo: "text/plain", texto: lista.join("\n") }, r => {
            app.avisar(r.erro || "endereço copiado");
        });
    }
    function copiarCaminho() {
        const c = caminhosDaSelecao();
        const texto = c.length > 0 ? c.join("\n") : painel.lixeira ? "" : painel.caminho;
        if (texto === "")
            return;
        Motor.pedir({ op: "area-por", tipo: "text/plain", texto: texto }, r => {
            app.avisar(r.erro || "caminho copiado");
        });
    }
    function colar(destino) {
        if (painel.lixeira || (painel.busca && destino === painel.caminho))
            return;
        if (Abas.eCompactado(destino)) {
            avisar(soLeitura);
            return;
        }
        Motor.pedir({ op: "area-ler" }, r => {
            const marca = r.tipo === "text/plain" ? Arquivos.lerMarca(r.texto) : null;
            if (marca !== null) {
                app.extrair(marca.arquivo, destino, "aqui", marca.internos, null);
                return;
            }
            const a = Arquivos.lerArea(r);
            if (a.tipo === "arquivos") {
                if (a.recortar)
                    app.tarefa({ op: "mover", origens: a.caminhos, destino: destino });
                else
                    app.tarefa({ op: "copiar", origens: a.caminhos, destino: destino });
            } else if (a.tipo === "imagem" || a.tipo === "texto") {
                const imagem = a.tipo === "imagem";
                app.pedirNome({
                    rotulo: "colar", titulo: imagem ? "Colar a imagem como arquivo" : "Colar o texto como arquivo",
                    frase: "O que está na área de transferência vira um arquivo novo nesta pasta.",
                    inicial: Arquivos.nomeLivre(imagem ? "Imagem colada.png" : "Texto colado.txt", app.ocupado, false), botao: "colar",
                    feito: nome => {
                        Motor.pedir({ op: "area-gravar", destino: destino, nome: nome, tipo: imagem ? "image/png" : "text/plain" }, g => {
                            if (g.erro) {
                                folhaDoNome.erro = g.erro;
                                return;
                            }
                            app.painel.aSelecionar = [Arquivos.juntar(destino, nome)];
                            app.fecharFolha();
                        });
                        return false;
                    }
                });
            } else {
                app.avisar(r.erro || "não há nada para colar");
            }
        });
    }
    function ocupado(nome) {
        return painel.todos.some(it => it.nome === nome);
    }
    // Soltar arquivos numa pasta: Ctrl copia, Shift move, Alt cria link (é o
    // que a ação proposta do arrastar traz).
    function soltar(caminhos, destino, acao) {
        if (!destino || Abas.eCompactado(destino)) {
            avisar(soLeitura);
            return;
        }
        if (Abas.eLixeira(destino)) {
            tarefa({ op: "lixeira", origens: caminhos });
            return;
        }
        if (acao === Qt.CopyAction) {
            tarefa({ op: "copiar", origens: caminhos, destino: destino });
        } else if (acao === Qt.LinkAction) {
            tarefa({ op: "ligar", origens: caminhos, destino: destino });
        } else {
            // Mover para onde já está não é nada.
            const fora = caminhos.filter(c => Arquivos.pai(c) !== destino && c !== destino);
            if (fora.length > 0)
                tarefa({ op: "mover", origens: fora, destino: destino });
        }
    }
    function paraALixeira() {
        const sel = painel.selecionados();
        if (sel.length === 0)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        if (painel.lixeira) {
            apagarDeVez();
            return;
        }
        const c = sel.map(it => it.caminho);
        // Numa pasta de rede não há lixeira: pergunta e apaga de vez.
        if (Remoto.eRemoto(painel.caminho, gvfs)) {
            apagarDeVez();
            return;
        }
        if (prefs.confirmar_lixeira === true)
            perguntar({
                titulo: "Mover " + Arquivos.plural(c.length, "item", "itens") + " para a lixeira?", frase: "Dá para restaurar depois, pela Lixeira.",
                lista: sel.map(it => it.nome), botao: "mover para a lixeira", feito: () => app.tarefa({ op: "lixeira", origens: c })
            });
        else
            tarefa({ op: "lixeira", origens: c });
    }
    function apagarDeVez() {
        const sel = painel.selecionados();
        if (sel.length === 0)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        const c = sel.map(it => it.caminho);
        const daLixeira = painel.lixeira;
        perguntar({
            titulo: "Apagar de vez " + Arquivos.plural(c.length, "item", "itens") + "?",
            frase: "Isto não tem volta: o que for apagado não vai para a lixeira" + (sel.some(Arquivos.ePasta) ? ", e as pastas vão com tudo o que têm dentro." : "."),
            lista: sel.map(it => it.nome), botao: "apagar de vez",
            feito: () => {
                if (daLixeira)
                    app.tarefa({ op: "lixeira-apagar", itens: c });
                else
                    app.tarefa({ op: "apagar", origens: c });
            }
        });
    }
    function esvaziar() {
        perguntar({
            titulo: "Esvaziar a lixeira?", frase: "Tudo o que está na lixeira é apagado de vez. Isto não tem volta.", lista: [],
            botao: "esvaziar", feito: () => app.tarefa({ op: "lixeira-esvaziar" })
        });
    }
    function paraOOutro(mover) {
        if (!duploVisivel) {
            avisar("F5 e F6 mandam a seleção para o outro painel: ligue o painel duplo (F3)");
            return;
        }
        const c = caminhosDaSelecao();
        const destino = outro.caminho;
        if (c.length === 0 || painel.lixeira || outro.lixeira || outro.busca)
            return;
        if (outro.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        if (painel.compactado !== null) {
            // De dentro de um compactado, F5 e F6 extraem a seleção para lá.
            extrair(painel.compactado.arquivo, destino, "aqui", painel.selecionados().map(it => it.interno), painel.segredoDe(painel.compactado.arquivo));
            return;
        }
        if (mover)
            tarefa({ op: "mover", origens: c, destino: destino });
        else
            tarefa({ op: "copiar", origens: c, destino: destino });
    }
    function novaPasta() {
        if (painel.lixeira || painel.busca)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        const destino = painel.caminho;
        pedirNome({
            rotulo: "novo", titulo: "Nova pasta", frase: "", inicial: Arquivos.nomeLivre("Nova pasta", ocupado, true), botao: "criar",
            feito: nome => {
                app.painel.aSelecionar = [Arquivos.juntar(destino, nome)];
                app.tarefa({ op: "criar-pasta", destino: destino, nome: nome });
                return true;
            }
        });
    }
    function novoArquivo(modelos) {
        if (painel.lixeira || painel.busca)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        const destino = painel.caminho;
        pedirNome({
            rotulo: "novo", titulo: modelos.length > 0 ? "Novo a partir de modelo" : "Novo arquivo",
            frase: modelos.length > 0 ? "Escolha o modelo e dê o nome." : "", modelos: modelos,
            inicial: Arquivos.nomeLivre("Novo arquivo.txt", ocupado, false), botao: "criar",
            feito: (nome, modelo) => {
                app.painel.aSelecionar = [Arquivos.juntar(destino, nome)];
                app.tarefa({ op: "criar-arquivo", destino: destino, nome: nome, modelo: modelo ? modelo.caminho : null });
                return true;
            }
        });
    }

    // --- folhas ------------------------------------------------------------------
    function perguntar(dados) {
        pergunta = dados;
        folha = "pergunta";
    }
    function pedirNome(dados) {
        nomeDados = dados;
        folha = "nome";
        folhaDoNome.abrir();
    }
    function fecharFolha() {
        if (folha === "propriedades" && folhaDasProps.contando)
            Motor.pedir({ op: "cancelar", pedido: pedidoDasProps }, null);
        pedidoDasProps = 0;
        if (folha === "previa")
            largarPrevia();
        segredoDados = null;
        // Fechar a pergunta do servidor, ou a folha de conectar no meio de uma
        // conexão, cancela a conexão.
        if (folha === "perguntaremota" && perguntaRemota !== null)
            Motor.pedir({ op: "remoto-responder", pedido: perguntaRemota.pedido, pergunta: perguntaRemota.pergunta, resposta: null }, null);
        perguntaRemota = null;
        if ((folha === "conectar" || folha === "perguntaremota") && conexao !== null)
            cancelarConexao();
        // Um conflito que chegou com outra folha aberta aparece agora.
        folha = conflitos.length > 0 && folha !== "conflito" ? "conflito" : "";
        if (folha === "conflito")
            folhaDoConflito.abrir();
        else
            painel.focar();
    }
    function propriedades(itens) {
        if (itens.length === 0)
            return;
        if (itens.some(it => Abas.eCompactado(it.caminho))) {
            avisar("as propriedades do que está dentro de um compactado aparecem na prévia (Espaço)");
            return;
        }
        folhaDasProps.dados = null;
        folhaDasProps.contagem = { arquivos: 0, pastas: 0, tamanho: 0 };
        folhaDasProps.contando = true;
        folhaDasProps.abrir();
        folha = "propriedades";
        pedidoDasProps = Motor.pedir({ op: "propriedades", caminhos: itens.map(it => it.caminho) }, r => {
            if (r.id !== app.pedidoDasProps)
                return;
            if (r.erro) {
                folhaDasProps.erro = r.erro;
                folhaDasProps.contando = false;
                return;
            }
            if (r.itens)
                folhaDasProps.dados = r;
            if (r.contagem)
                folhaDasProps.contagem = r.contagem;
            folhaDasProps.contando = r.fim !== true;
        });
    }
    function abrirCom(caminhos, mime, nome) {
        folhaDoAbrirCom.alvo = { caminhos: caminhos, mime: mime, nome: nome };
        folhaDoAbrirCom.apps = [];
        folhaDoAbrirCom.abrir();
        folha = "abrircom";
        pedirApps(false);
    }
    function pedirApps(todos) {
        const mime = folhaDoAbrirCom.alvo.mime;
        folhaDoAbrirCom.lendo = true;
        Motor.pedir({ op: "apps", mime: mime, todos: todos }, r => {
            folhaDoAbrirCom.lendo = false;
            if (r.erro) {
                app.avisar(r.erro);
                return;
            }
            folhaDoAbrirCom.apps = r.apps || [];
            folhaDoAbrirCom.padrao = r.padrao || "";
        });
    }
    function responderConflito(resposta, nome, todos) {
        const c = conflitos[0];
        if (!c)
            return;
        Motor.pedir({ op: "responder", tarefa: c.tarefa, conflito: c.conflito, resposta: resposta, nome: nome === "" ? null : nome, todos: todos }, r => {
            if (r.erro)
                app.avisar(r.erro);
        });
        conflitos = conflitos.slice(1);
        if (conflitos.length > 0) {
            folhaDoConflito.abrir();
        } else {
            folha = "";
            painel.focar();
        }
    }

    // --- renomear em lote ----------------------------------------------------------------
    function abrirLote() {
        const sel = painel.selecionados();
        if (sel.length < 2 || painel.lixeira)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        folhaDoLote.itens = sel;
        // Numa busca os itens vêm de pastas diferentes: quem confere o nome
        // que já existe é o motor.
        folhaDoLote.existentes = painel.busca ? null : { pasta: painel.caminho, nomes: painel.todos.map(it => it.nome) };
        folha = "lote";
        folhaDoLote.abrir();
    }

    // --- compactados -------------------------------------------------------------------
    function pedirSenha(dados) {
        segredoDados = dados;
        folha = "pedesenha";
        folhaDaSenha.abrir(dados.erro || "");
    }
    // Extrai um compactado (ou só os `internos` dele) para `destino`. Antes
    // pergunta ao motor o que ele é: falta ferramenta, pede senha ou parece
    // uma bomba, a pessoa fica sabendo antes de qualquer coisa ser escrita.
    function extrair(arquivo, destino, modo, internos, segredo) {
        if (!destino || Abas.eCompactado(destino) || Abas.eLixeira(destino)) {
            avisar("escolha uma pasta de verdade para extrair");
            return;
        }
        Motor.pedir({ op: "compactado-info", caminho: arquivo, senha: segredo || null }, r => {
            const nome = Arquivos.nomeDe(arquivo);
            const pedeSenha = (r.erro && r.motivo === "senha") || (!r.erro && r.cifrado === true && !segredo);
            if (pedeSenha) {
                app.pedirSenha({ titulo: nome, erro: segredo ? "essa senha não abriu o compactado" : "", feito: s => app.extrair(arquivo, destino, modo, internos, s) });
                return;
            }
            if (r.erro) {
                app.avisar(r.erro);
                return;
            }
            if (r.disponivel === false) {
                app.avisar("para extrair " + (r.formato || "este formato") + " falta o pacote " + (r.falta || "da ferramenta"));
                return;
            }
            const vai = semLimite => app.tarefa({ op: "extrair", caminho: arquivo, destino: destino, modo: modo, internos: internos && internos.length > 0 ? internos : null,
                senha: segredo || null, sem_limite: semLimite });
            if (r.suspeito === true)
                app.perguntar({
                    titulo: "«" + nome + "» cresce demais ao ser extraído",
                    frase: "São " + Arquivos.plural(r.itens || 0, "item", "itens") + " e " + Arquivos.tamanho(r.tamanho || 0) + " depois de extraído"
                        + (r.taxa ? " (" + Math.round(r.taxa) + " vezes o tamanho do arquivo)" : "") + ". Pode ser um arquivo feito para encher o disco. Extrair mesmo assim?",
                    lista: [], botao: "extrair mesmo assim", feito: () => vai(true)
                });
            else
                vai(false);
        });
    }
    // "Extrair para…": pergunta a pasta, e extrai numa pasta com o nome do arquivo dentro dela.
    function extrairPara(arquivo, internos, segredo) {
        const sugestao = duploVisivel && !outro.lixeira && outro.compactado === null ? outro.caminho : Arquivos.pastaReal(painel.caminho);
        pedirNome({
            rotulo: "extrair", titulo: "Extrair «" + Arquivos.nomeDe(arquivo) + "» para…", frase: "A pasta em que extrair (ela tem de existir). ~ é a pasta pessoal.",
            inicial: Arquivos.til(sugestao, Motor.home), botao: "extrair", eNome: false, vazio: "escreva o caminho de uma pasta",
            feito: texto => {
                const destino = Arquivos.limpar(texto, Motor.home);
                if (destino === null) {
                    folhaDoNome.erro = "isso não é um caminho: escreva a partir de / ou de ~";
                    return false;
                }
                app.fecharFolha();
                app.extrair(arquivo, destino, internos ? "aqui" : "pasta", internos, segredo);
                return false;
            }
        });
    }
    function compactar() {
        const sel = painel.selecionados();
        if (sel.length === 0 || painel.lixeira)
            return;
        if (painel.compactado !== null) {
            avisar(soLeitura);
            return;
        }
        const destino = painel.busca ? Arquivos.pai(sel[0].caminho) : painel.caminho;
        Motor.pedir({ op: "formatos" }, r => {
            if (r.erro) {
                app.avisar(r.erro);
                return;
            }
            app.compactarDados = { origens: sel.map(it => it.caminho), destino: destino };
            folhaDoCompactar.formatos = r.compactar || [];
            folhaDoCompactar.quantos = sel.length;
            folhaDoCompactar.inicial = Arquivos.nomeLivre(Arquivos.nomeDoCompactado(sel, destino, "zip"), app.ocupado, false);
            app.folha = "compactar";
            folhaDoCompactar.abrir();
        });
    }
    // Enter num arquivo de dentro de um compactado: o motor extrai só ele
    // para um temporário e a janela o abre.
    function abrirDeDentro(arquivo, itens, segredo) {
        for (const it of itens)
            Motor.pedir({ op: "compactado-abrir", caminho: arquivo, interno: it.interno, senha: segredo || null }, r => {
                if (r.erro)
                    app.avisar(r.erro);
                else
                    Motor.pedir({ op: "abrir", caminhos: [r.caminho], app: null }, app.lancar);
            });
    }

    // --- prévia rápida -----------------------------------------------------------------
    function abrirPrevia() {
        if (extensoes.previa !== true)
            return;
        const sel = painel.selecionados();
        const i = painel.selecao.cursor;
        const it = i >= 0 && i < painel.total && Selecao.tem(painel.selecao, painel.itens[i]) ? painel.itens[i] : sel.length > 0 ? sel[0] : null;
        if (!it)
            return;
        folha = "previa";
        mostrarPrevia(it);
    }
    function largarPrevia() {
        if (pedidoDaContagem > 0)
            Motor.pedir({ op: "cancelar", pedido: pedidoDaContagem }, null);
        pedidoDaContagem = 0;
        pedidoDaPrevia = 0;
    }
    function mostrarPrevia(it) {
        largarPrevia();
        folhaDaPrevia.reiniciar();
        folhaDaPrevia.item = it;
        folhaDaPrevia.posicao = painel.selecao.cursor >= 0 ? (painel.selecao.cursor + 1) + " de " + painel.total : "";
        if (painel.compactado !== null) {
            // O que está dentro de um compactado não está no disco: só a ficha.
            folhaDaPrevia.dados = { classe: "outro" };
            return;
        }
        folhaDaPrevia.lendo = true;
        pedidoDaPrevia = Motor.pedir({ op: "previa", caminho: it.caminho }, r => {
            if (r.id !== app.pedidoDaPrevia)
                return;
            folhaDaPrevia.lendo = false;
            if (r.erro) {
                folhaDaPrevia.erro = r.erro;
                return;
            }
            folhaDaPrevia.dados = r;
            if (r.classe === "pdf") {
                Motor.pedir({ op: "miniatura", caminhos: [it.caminho], tamanho: "x-large", grupo: "previa", prioridade: 10 }, null);
            } else if (r.classe === "pasta") {
                folhaDaPrevia.contando = true;
                app.pedidoDaContagem = Motor.pedir({ op: "propriedades", caminhos: [it.caminho] }, c => {
                    if (c.id !== app.pedidoDaContagem)
                        return;
                    if (c.contagem)
                        folhaDaPrevia.contagem = c.contagem;
                    if (c.erro || c.fim === true) {
                        folhaDaPrevia.contando = false;
                        app.pedidoDaContagem = 0;
                    }
                });
            } else if (r.classe === "compactado") {
                const dentro = [];
                app.pedidoDaContagem = Motor.pedir({ op: "compactado-listar", caminho: it.caminho, dentro: "", senha: null }, c => {
                    if (c.id !== app.pedidoDaContagem)
                        return;
                    if (c.erro) {
                        folhaDaPrevia.erro = c.motivo === "senha" ? "Compactado cifrado: Enter abre e pede a senha." : c.erro;
                        app.pedidoDaContagem = 0;
                        return;
                    }
                    for (const x of c.itens || [])
                        if (dentro.length < 5000)
                            dentro.push(x);
                    if (c.fim === true) {
                        folhaDaPrevia.listagem = Arquivos.ordenar(dentro, { campo: "nome", inversa: false }, true);
                        app.pedidoDaContagem = 0;
                    }
                });
            }
        });
    }
    function vizinhoDaPrevia(delta) {
        const i = painel.selecao.cursor + delta;
        if (i < 0 || i >= painel.total)
            return;
        painel.selecao = Selecao.so(painel.itens, i);
        painel.mostrar(i);
        mostrarPrevia(painel.itens[i]);
    }

    // --- o que o motor diz sozinho ---------------------------------------------------
    Connections {
        target: Motor
        function onEvento(e) {
            switch (e.evento) {
            case "pasta":
                painelA.evento(e);
                painelB.evento(e);
                break;
            case "miniatura":
                painelA.evento(e);
                painelB.evento(e);
                // A primeira página de um PDF, para a prévia rápida.
                if (app.folha === "previa" && e.tamanho === "x-large" && folhaDaPrevia.item && e.caminho === folhaDaPrevia.item.caminho) {
                    folhaDaPrevia.miniatura = e.arquivo || "";
                    if (!e.arquivo)
                        folhaDaPrevia.erro = "Não deu para fazer a prévia deste documento. Enter abre no app padrão.";
                }
                break;
            case "lixeira":
                app.naLixeira = e.itens || 0;
                painelA.evento(e);
                painelB.evento(e);
                break;
            case "tarefa":
                app.tarefaMudou(e);
                break;
            case "conflito":
                app.conflitos = app.conflitos.concat([e]);
                if (app.folha === "") {
                    app.folha = "conflito";
                    folhaDoConflito.abrir();
                }
                break;
            case "historico":
                app.historico = { desfazer: e.desfazer || null, refazer: e.refazer || null };
                break;
            case "discos":
                app.discos = e.discos || [];
                break;
            case "remotos":
                app.chegaramRemotos(e.remotos || []);
                break;
            case "pergunta-remota":
                app.perguntou(e);
                break;
            }
        }
    }
    // Saiu da busca navegando (uma migalha, um lugar): o campo da busca fecha junto.
    Connections {
        target: app.painel
        function onBuscaChanged() {
            if (!app.painel.busca && barra.buscando && !barra.comFoco)
                barra.fecharBusca();
        }
    }
    function tarefaMudou(e) {
        const acabou = Arquivos.tarefaAcabou(e);
        const fica = !acabou || e.estado === "erro";
        let achou = false;
        const lista = [];
        for (const t of tarefas) {
            if (t.tarefa !== e.tarefa) {
                lista.push(t);
            } else {
                achou = true;
                if (fica)
                    lista.push(e);
            }
        }
        if (!achou && fica)
            lista.push(e);
        tarefas = lista;
        if (!acabou)
            return;
        conflitos = conflitos.filter(c => c.tarefa !== e.tarefa);
        if (folha === "conflito" && conflitos.length === 0)
            fecharFolha();
        // O que a tarefa criou fica selecionado, no painel que mostra a pasta.
        if (e.criados && e.criados.length > 0)
            for (const p of [painelA, painelB])
                if (p.pronto && !p.lixeira && e.criados.some(c => Arquivos.pai(c) === p.caminho)) {
                    p.aSelecionar = e.criados;
                    p.refazer();
                }
        // Numa pasta de rede não há quem avise das mudanças na hora: ao fim de
        // cada tarefa o painel que está num servidor lê a pasta de novo.
        for (const p of [painelA, painelB])
            if (p.pronto && p.remoto && !p.busca && !p.carregando)
                p.recarregar();
        if (e.sem_lixeira && e.sem_lixeira.length > 0) {
            const c = e.sem_lixeira;
            perguntar({
                titulo: "Não há lixeira neste disco", frase: "Estes itens estão num disco em que não dá para criar uma lixeira. Apagar de vez? Isto não tem volta.",
                lista: c.map(Arquivos.nomeDe), botao: "apagar de vez", feito: () => app.tarefa({ op: "apagar", origens: c })
            });
        }
    }

    // --- menus ---------------------------------------------------------------------
    function contexto() {
        const sel = painel.selecionados();
        return { item: sel.length === 1 ? sel[0] : null, n: sel.length, lixeira: painel.lixeira, busca: painel.busca, gravavel: painel.gravavel,
            remoto: Remoto.eRemoto(painel.caminho, gvfs),
            compactado: painel.compactado !== null, duplo: duploVisivel, extensoes: extensoes };
    }
    function menuDoPainel(x, y) {
        const c = contexto();
        menu.abrir(x, y, Menu.menu(c.n === 0 ? "fundo" : c.n === 1 ? "item" : "varios", c));
    }
    function menuDoTeclado() {
        const p = painel.pontoDoCursor();
        menuDoPainel(p.x, p.y);
    }
    function menuDoLugar(linha, x, y) {
        lugarDoMenu = linha;
        menu.abrir(x, y, Menu.menu("lugar", { lugar: linha }));
    }
    function menuDaAba(numero, i, x, y) {
        ativar(numero);
        abaDoMenu = { i: i };
        menu.abrir(x, y, Menu.daAba({ n: painel.abas.abas.length }));
    }
    function menuPrincipal(x, y) {
        menu.abrir(x - 280, y, Menu.principal({ duplo: duplo, ocultos: prefs.ocultos === true, pastas_antes: prefs.pastas_antes !== false, clique: prefs.clique,
            confirmar_lixeira: prefs.confirmar_lixeira === true, desfazer: historico.desfazer, refazer: historico.refazer,
            miniaturas_remotas: prefs.miniaturas_remotas !== false, miniatura_remota_mb: prefs.miniatura_remota_mb || 5 }));
    }

    // Abrir um lugar da barra lateral (montando o disco, se preciso).
    function abrirLugar(linha, modo) {
        if (linha.tipo === "conectar") {
            abrirConectar();
            return;
        }
        const vai = caminho => {
            if (modo === "aba" || linha.novaAba)
                app.painel.novaAba(caminho);
            else if (modo === "outro")
                app.noOutro(caminho);
            else if (modo === "janela")
                app.novaJanela(caminho);
            else
                app.painel.ir(caminho);
            app.naLateral = false;
            app.painel.focar();
        };
        if (linha.tipo === "disco" && !linha.montado) {
            Motor.pedir({ op: "montar", dispositivo: linha.dispositivo }, r => {
                if (r.erro) {
                    app.avisar(r.erro);
                    return;
                }
                app.discos = r.discos || app.discos;
                const d = app.discos.filter(x => x.dispositivo === linha.dispositivo)[0];
                if (d && d.montado)
                    vai(d.montado);
            });
            return;
        }
        // Um favorito de rede (sftp://…): monta, se ainda não está, e abre.
        if (linha.remoto && linha.uri) {
            conectar(linha.uri, vai);
            return;
        }
        if (linha.caminho)
            vai(linha.caminho);
    }

    // --- rede ----------------------------------------------------------------------
    function lerRemotos(comRede) {
        Motor.pedir({ op: "remotos", rede: comRede }, r => {
            if (r.erro)
                return;
            app.chegaramRemotos(r.remotos || []);
            if (comRede)
                app.rede = r.rede || [];
        });
    }
    // A lista dos montados mudou: um painel que estava num remoto que saiu
    // volta para a pasta pessoal (não fica olhando para o nada).
    function chegaramRemotos(lista) {
        remotos = lista;
        for (const p of [painelA, painelB])
            if (p.pronto && Remoto.eRemoto(p.caminho, gvfs) && Remoto.doRemoto(p.caminho, lista) === null)
                p.caiuAConexao("o servidor foi desconectado");
    }
    function abrirConectar() {
        naLateral = false;
        folhaDeConectar.erro = "";
        folhaDeConectar.saida = "";
        folha = "conectar";
        folhaDeConectar.abrir();
        Motor.pedir({ op: "remoto-hosts" }, r => {
            if (!r.erro)
                app.hostsDoSsh = r.hosts || [];
        });
        lerRemotos(true);
    }
    // Monta um endereço (se ainda não está montado) e chama feito(caminho).
    // As perguntas do servidor chegam por evento, no meio do caminho.
    function conectar(uri, feito) {
        const ja = Remoto.montadoEm(uri, remotos);
        if (ja !== null) {
            feito(ja.caminho);
            return;
        }
        if (conexao !== null) {
            avisar("já há uma conexão em andamento");
            return;
        }
        const naFolha = folha === "conectar";
        folhaDeConectar.erro = "";
        folhaDeConectar.saida = "";
        if (!naFolha)
            avisar("conectando…");
        const pedido = Motor.pedir({ op: "remoto-montar", uri: uri, anonimo: false }, r => {
            const c = app.conexao;
            app.conexao = null;
            app.perguntaRemota = null;
            if (app.folha === "perguntaremota")
                app.folha = naFolha ? "conectar" : "";
            if (r.erro) {
                // Cancelada por quem pediu: não é erro para mostrar.
                if (c !== null && c.cancelada === true)
                    return;
                folhaDeConectar.erro = Remoto.erroDeMontar(r);
                folhaDeConectar.saida = r.saida || "";
                if (app.folha !== "conectar")
                    app.avisar(Remoto.erroDeMontar(r));
                return;
            }
            if (app.folha === "conectar")
                app.fecharFolha();
            app.lerRemotos(false);
            feito(r.caminho || r.raiz);
        });
        conexao = { pedido: pedido, uri: uri, naFolha: naFolha, cancelada: false };
    }
    function cancelarConexao() {
        if (conexao === null)
            return;
        conexao = Object.assign({}, conexao, { cancelada: true });
        Motor.pedir({ op: "cancelar", pedido: conexao.pedido }, null);
    }
    // O servidor (pelo gio) perguntou algo: usuário, senha, domínio, confirmação.
    function perguntou(e) {
        if (conexao === null || e.pedido !== conexao.pedido) {
            // Ninguém aqui espera por isto: a montagem é cancelada.
            Motor.pedir({ op: "remoto-responder", pedido: e.pedido, pergunta: e.pergunta, resposta: null }, null);
            return;
        }
        perguntaRemota = e;
        folha = "perguntaremota";
        folhaDaPerguntaRemota.abrir();
    }
    // A resposta vai ao motor e a folha fecha; a janela não fica com ela.
    function responderRemota(resposta) {
        const p = perguntaRemota;
        perguntaRemota = null;
        if (p === null)
            return;
        Motor.pedir({ op: "remoto-responder", pedido: p.pedido, pergunta: p.pergunta, resposta: resposta }, null);
        if (resposta === null && conexao !== null)
            conexao = Object.assign({}, conexao, { cancelada: true });
        folha = conexao !== null && conexao.naFolha && resposta !== null ? "conectar" : "";
        if (folha === "")
            painel.focar();
    }
    function desconectar(linha) {
        if (!linha || !linha.uri) {
            avisar("este lugar se desconecta pelo aparelho");
            return;
        }
        for (const p of [painelA, painelB])
            if (!p.lixeira && linha.caminho && Arquivos.dentro(linha.caminho, p.caminho))
                p.ir(Motor.home);
        Motor.pedir({ op: "remoto-desmontar", uri: linha.uri }, r => {
            if (r.erro)
                app.avisar(r.erro);
            else
                app.chegaramRemotos(r.remotos || []);
        });
    }
    // Desmontar ou ejetar: quem está dentro do disco sai antes.
    function soltarDisco(linha, ejetar) {
        if (linha.montado)
            for (const p of [painelA, painelB])
                if (!p.lixeira && Arquivos.dentro(linha.montado, p.caminho))
                    p.ir(Motor.home);
        const feito = r => {
            if (r.erro)
                app.avisar(r.erro);
            else
                app.discos = r.discos || app.discos;
        };
        if (ejetar)
            Motor.pedir({ op: "ejetar", dispositivo: linha.dispositivo }, feito);
        else
            Motor.pedir({ op: "desmontar", dispositivo: linha.dispositivo }, feito);
    }
    function chegaramFavoritos(r) {
        if (r.erro)
            avisar(r.erro);
        else
            favoritos = r.favoritos || [];
    }

    // O que cada entrada de menu (e cada atalho) faz. As entradas das
    // extensões (previa, renomear-lote, compactar, extrair…) só agem quando o
    // motor diz que tem a extensão.
    function acao(id) {
        const sel = painel.selecionados();
        const um = sel.length === 1 ? sel[0] : null;
        const l = lugarDoMenu;
        const dentro = painel.compactado;
        // Num compactado aberto como pasta nada se cria, renomeia nem apaga.
        if (dentro !== null && ["colar", "colar-dentro", "recortar", "renomear", "ligar", "lixeira", "apagar", "nova-pasta", "novo-arquivo", "novo-modelo",
            "executavel", "renomear-lote", "compactar", "abrir-com"].indexOf(id) >= 0) {
            avisar(soLeitura);
            return;
        }
        switch (id) {
        case "abrir": painel.abrirSelecao(); break;
        case "abrir-com": if (um) abrirCom([um.caminho], um.mime, um.nome); break;
        case "abrir-aba": if (um) painel.novaAba(um.caminho); break;
        case "abrir-outro": if (um) noOutro(um.caminho); break;
        case "abrir-janela": if (um) novaJanela(um.caminho); break;
        case "abrir-pasta": if (um) { painel.aSelecionar = [um.caminho]; barra.fecharBusca(); painel.ir(Arquivos.pai(um.caminho)); } break;
        case "recortar": copiar(true); break;
        case "copiar": copiar(false); break;
        case "colar": colar(painel.caminho); break;
        case "colar-dentro": if (um) colar(um.caminho); break;
        case "copiar-caminho": copiarCaminho(); break;
        case "copiar-endereco": copiarEndereco(); break;
        case "conectar": abrirConectar(); break;
        case "remoto-desmontar": if (l) desconectar(l); break;
        case "lugar-endereco":
            if (l && l.uri)
                Motor.pedir({ op: "area-por", tipo: "text/plain", texto: l.uri }, r => app.avisar(r.erro || "endereço copiado"));
            break;
        case "lugar-favorito": if (l && l.caminho) Motor.pedir({ op: "favorito-por", caminho: l.caminho, nome: l.nome, posicao: null }, chegaramFavoritos); break;
        case "miniaturas-remotas": preferir({ miniaturas_remotas: prefs.miniaturas_remotas === false }); break;
        case "miniatura-remota-mb": preferir({ miniatura_remota_mb: Remoto.proximoLimite(prefs.miniatura_remota_mb || 5) }); break;
        case "renomear": painel.renomear(); break;
        case "ligar": if (sel.length > 0 && !painel.lixeira) tarefa({ op: "ligar", origens: sel.map(it => it.caminho), destino: painel.busca ? Arquivos.pai(sel[0].caminho) : painel.caminho }); break;
        case "lixeira": paraALixeira(); break;
        case "apagar": apagarDeVez(); break;
        case "lixeira-apagar": apagarDeVez(); break;
        case "restaurar": if (sel.length > 0 && painel.lixeira) tarefa({ op: "restaurar", itens: sel.map(it => it.caminho) }); break;
        case "esvaziar": esvaziar(); break;
        case "nova-pasta": novaPasta(); break;
        case "novo-arquivo": novoArquivo([]); break;
        case "novo-modelo":
            Motor.pedir({ op: "modelos" }, r => {
                if (r.erro || !r.modelos || r.modelos.length === 0)
                    app.avisar(r.erro || "não há modelos em " + Arquivos.til(r.pasta || Motor.home + "/Templates", Motor.home));
                else
                    app.novoArquivo(r.modelos);
            });
            break;
        case "favorito": {
            const c = dentro !== null ? Arquivos.pastaReal(painel.caminho) : um && Arquivos.ePasta(um) ? um.caminho : painel.caminho;
            if (!painel.lixeira)
                Motor.pedir({ op: "favorito-por", caminho: c, nome: null, posicao: null }, chegaramFavoritos);
            break;
        }
        case "terminal": if (!painel.lixeira) terminal(dentro !== null ? Arquivos.pastaReal(painel.caminho) : um && Arquivos.ePasta(um) ? um.caminho : painel.caminho); break;
        case "propriedades":
            if (sel.length > 0)
                propriedades(sel);
            else if (!painel.lixeira)
                propriedades([{ caminho: dentro !== null ? dentro.arquivo : painel.caminho }]);
            break;
        case "executavel":
            if (um)
                Motor.pedir({ op: "permissoes", caminhos: [um.caminho], modo: null, executavel: (um.modo & 64) === 0, grupo: null, recursivo: false }, r => {
                    app.avisar(r.erro || (r.erros && r.erros.length > 0 ? r.erros[0].erro : ""));
                });
            break;
        case "selecionar-tudo": painel.selecionarTudo(); break;
        case "selecionar-padrao":
            pedirNome({ rotulo: "selecionar", titulo: "Selecionar por padrão", frase: "* vale por qualquer trecho, ? por uma letra. Sem eles, um pedaço do nome.",
                inicial: "*.", botao: "selecionar", eNome: false, feito: padrao => { app.painel.selecionarPorPadrao(padrao); return true; } });
            break;
        // A barra lateral.
        case "lugar-abrir": if (l) abrirLugar(l, ""); break;
        case "lugar-aba": if (l) abrirLugar(l, "aba"); break;
        case "lugar-outro": if (l) abrirLugar(l, "outro"); break;
        case "lugar-janela": if (l) abrirLugar(l, "janela"); break;
        case "lugar-terminal": if (l && l.caminho) terminal(l.caminho); break;
        case "lugar-propriedades": if (l && l.caminho) propriedades([{ caminho: l.caminho }]); break;
        case "favorito-tirar": if (l) Motor.pedir({ op: "favorito-tirar", uri: l.uri }, chegaramFavoritos); break;
        case "favorito-subir": if (l) Motor.pedir({ op: "favorito-mover", de: l.posicao, para: l.posicao - 1 }, chegaramFavoritos); break;
        case "favorito-descer": if (l) Motor.pedir({ op: "favorito-mover", de: l.posicao, para: l.posicao + 1 }, chegaramFavoritos); break;
        case "favorito-renomear":
            if (l)
                pedirNome({ rotulo: "favorito", titulo: "Renomear o favorito", frase: "Só muda o nome na barra lateral; a pasta fica como está.", inicial: l.nome,
                    botao: "renomear", feito: nome => { Motor.pedir({ op: "favorito-renomear", uri: l.uri, nome: nome }, app.chegaramFavoritos); return true; } });
            break;
        case "montar": if (l) abrirLugar(l, ""); break;
        case "desmontar": if (l) soltarDisco(l, false); break;
        case "ejetar": if (l) soltarDisco(l, true); break;
        // As abas.
        case "aba-duplicar": if (abaDoMenu) painel.duplicarAba(abaDoMenu.i); break;
        case "aba-outro": if (abaDoMenu) noOutro(painel.abas.abas[abaDoMenu.i].caminho); break;
        case "aba-fechar": if (abaDoMenu) painel.fecharAba(abaDoMenu.i); break;
        case "aba-outras": if (abaDoMenu) painel.abas = Abas.painel([painel.abas.abas[abaDoMenu.i].caminho], 0); break;
        // O menu principal.
        case "nova-janela": novaJanela(painel.lixeira ? Motor.home : painel.caminho); break;
        case "nova-aba": painel.novaAba(""); break;
        case "duplo": ligarDuplo(); break;
        case "ocultos": preferir({ ocultos: prefs.ocultos !== true }); break;
        case "pastas-antes": preferir({ pastas_antes: prefs.pastas_antes === false }); break;
        case "clique": preferir({ clique: prefs.clique === "simples" ? "duplo" : "simples" }); break;
        case "confirmar-lixeira": preferir({ confirmar_lixeira: prefs.confirmar_lixeira !== true }); break;
        case "desfazer": tarefa({ op: "desfazer" }); break;
        case "refazer": tarefa({ op: "refazer" }); break;
        case "recarregar": painel.recarregar(); break;
        case "atalhos": folha = "atalhos"; break;
        // As extensões: só agem com a extensão ligada no motor.
        case "previa": abrirPrevia(); break;
        case "renomear-lote": if (extensoes.lote === true) abrirLote(); break;
        case "compactar": if (extensoes.compactados === true) compactar(); break;
        case "extrair": if (extensoes.compactados === true && um) extrair(um.caminho, Arquivos.pai(um.caminho), "aqui", null, null); break;
        case "extrair-pasta": if (extensoes.compactados === true && um) extrair(um.caminho, Arquivos.pai(um.caminho), "pasta", null, null); break;
        case "extrair-para":
            if (extensoes.compactados !== true)
                break;
            if (dentro !== null)
                extrairPara(dentro.arquivo, sel.length > 0 ? sel.map(it => it.interno) : null, painel.segredoDe(dentro.arquivo));
            else if (um)
                extrairPara(um.caminho, null, null);
            break;
        // De dentro de um compactado, para a pasta em que ele está.
        case "extrair-tudo": if (dentro !== null) extrair(dentro.arquivo, Arquivos.pai(dentro.arquivo), "pasta", null, painel.segredoDe(dentro.arquivo)); break;
        case "extrair-selecao":
            if (dentro !== null && sel.length > 0)
                extrair(dentro.arquivo, Arquivos.pai(dentro.arquivo), "aqui", sel.map(it => it.interno), painel.segredoDe(dentro.arquivo));
            break;
        }
    }

    // Esc fecha o que está por cima, uma coisa de cada vez.
    function fechar() {
        if (menu.aberto)
            menu.fechar();
        else if (folha === "conflito")
            responderConflito("cancelar", "", false);
        else if (folha !== "")
            fecharFolha();
        else if (naLateral)
            naLateral = false;
        else if (painel.renomeando)
            painel.renomeando = false;
        else if (barra.comFoco && barra.buscando)
            fecharBusca();
        else if (barra.comFoco)
            barra.sair();
        else if (painel.busca || barra.buscando)
            fecharBusca();
        else if (painel.selecao.n > 0)
            painel.limparSelecao();
        painel.focar();
    }

    // --- teclado -----------------------------------------------------------------
    // Com a janela livre, toda tecla SEM modificador é do painel ativo, tratada
    // num lugar só (Painel.tecla): nenhum Shortcut disputa com ela. Os Shortcut
    // daqui para baixo são os com modificador, ou os de uma folha, do menu ou
    // da barra lateral (que só valem com a janela não livre).
    // As teclas de função, vindas do painel. Devolve se tratou.
    function teclaDeFuncao(k) {
        switch (k) {
        case Qt.Key_F1: folha = "atalhos"; return true;
        case Qt.Key_F2: acao(painel.selecao.n > 1 && extensoes.lote === true ? "renomear-lote" : "renomear"); return true;
        case Qt.Key_F3: ligarDuplo(); return true;
        case Qt.Key_F4: acao("terminal"); return true;
        case Qt.Key_F5:
            if (duploVisivel)
                paraOOutro(false);
            else
                painel.recarregar();
            return true;
        case Qt.Key_F6: paraOOutro(true); return true;
        }
        return false;
    }
    // A rede de segurança: uma tecla que chega aqui com a janela livre e o
    // teclado fora do painel (o foco ficou em lugar nenhum) não se perde: o
    // painel a trata e fica com o foco.
    focus: true
    Keys.onPressed: event => {
        if (app.livre && !app.painel.temTeclado) {
            app.painel.focar();
            app.painel.tecla(event);
        }
    }
    // O teclado volta ao painel sempre que a janela fica livre (fechou uma
    // folha, um menu, a edição do caminho) e quando a janela ganha o foco.
    onLivreChanged: {
        if (livre)
            painel.focar();
    }
    readonly property bool janelaAtiva: Window.active
    onJanelaAtivaChanged: {
        if (janelaAtiva && livre)
            painel.focar();
    }

    // --- atalhos -------------------------------------------------------------------
    Shortcut { sequences: ["Alt+Left"]; enabled: app.livre; onActivated: app.painel.voltar() }
    Shortcut { sequences: ["Alt+Right"]; enabled: app.livre; onActivated: app.painel.avancar() }
    Shortcut { sequences: ["Alt+Up"]; enabled: app.livre; onActivated: app.painel.subir() }
    Shortcut { sequences: ["Alt+Home"]; enabled: app.livre; onActivated: app.painel.ir(Motor.home) }
    Shortcut { sequences: ["Ctrl+L"]; enabled: app.folha === "" && !menu.aberto; onActivated: { app.naLateral = false; barra.editar(); } }
    Shortcut { sequences: ["Ctrl+F"]; enabled: app.folha === "" && !menu.aberto && !app.painel.lixeira; onActivated: { app.naLateral = false; barra.buscar(); } }
    Shortcut { sequences: ["Ctrl+T"]; enabled: app.livre; onActivated: app.painel.novaAba("") }
    Shortcut { sequences: ["Ctrl+Shift+D"]; enabled: app.livre; onActivated: app.painel.duplicarAba(app.painel.abas.atual) }
    Shortcut {
        sequences: ["Ctrl+W"]
        enabled: app.livre
        onActivated: {
            // Fechar a última aba fecha a janela.
            if (app.painel.abas.abas.length > 1)
                app.painel.fecharAba(app.painel.abas.atual);
            else
                Qt.quit();
        }
    }
    Shortcut { sequences: ["Ctrl+Tab"]; enabled: app.livre; onActivated: app.painel.trocarAba(1) }
    Shortcut { sequences: ["Ctrl+Shift+Tab"]; enabled: app.livre; onActivated: app.painel.trocarAba(-1) }
    Shortcut { sequences: ["Ctrl+Tab"]; enabled: app.folha === "propriedades"; onActivated: folhaDasProps.aba = folhaDasProps.aba === "geral" ? "permissoes" : "geral" }
    Shortcut { sequences: ["Ctrl+R"]; enabled: app.livre; onActivated: app.painel.recarregar() }
    Shortcut { sequences: ["Ctrl+1"]; enabled: app.livre; onActivated: app.escolherVisao("icones") }
    Shortcut { sequences: ["Ctrl+2"]; enabled: app.livre; onActivated: app.escolherVisao("lista") }
    Shortcut { sequences: ["Ctrl+3"]; enabled: app.livre; onActivated: app.escolherVisao("compacta") }
    Shortcut { sequences: ["Ctrl++", "Ctrl+="]; enabled: app.livre; onActivated: app.zoom(1) }
    Shortcut { sequences: ["Ctrl+-"]; enabled: app.livre; onActivated: app.zoom(-1) }
    Shortcut { sequences: ["Ctrl+0"]; enabled: app.livre; onActivated: app.zoom(0) }
    Shortcut { sequences: ["Ctrl+H"]; enabled: app.livre; onActivated: app.acao("ocultos") }
    Shortcut { sequences: ["Ctrl+A"]; enabled: app.livre; onActivated: app.painel.selecionarTudo() }
    Shortcut { sequences: ["Ctrl+Shift+I"]; enabled: app.livre; onActivated: app.painel.inverter() }
    Shortcut { sequences: ["Ctrl+S"]; enabled: app.livre; onActivated: app.acao("selecionar-padrao") }
    Shortcut { sequences: ["Ctrl+C"]; enabled: app.livre; onActivated: app.copiar(false) }
    Shortcut { sequences: ["Ctrl+X"]; enabled: app.livre; onActivated: app.copiar(true) }
    Shortcut { sequences: ["Ctrl+V"]; enabled: app.livre; onActivated: app.colar(app.painel.caminho) }
    Shortcut { sequences: ["Ctrl+Shift+C"]; enabled: app.livre; onActivated: app.copiarCaminho() }
    Shortcut { sequences: ["Ctrl+Z"]; enabled: app.livre; onActivated: app.acao("desfazer") }
    Shortcut { sequences: ["Ctrl+Shift+Z", "Ctrl+Y"]; enabled: app.livre; onActivated: app.acao("refazer") }
    Shortcut { sequences: ["Ctrl+Shift+N"]; enabled: app.livre; onActivated: app.novaPasta() }
    Shortcut { sequences: ["Alt+Return", "Alt+Enter"]; enabled: app.livre; onActivated: app.acao("propriedades") }
    Shortcut { sequences: ["Ctrl+N"]; enabled: app.livre; onActivated: app.acao("nova-janela") }
    Shortcut { sequences: ["Ctrl+K"]; enabled: app.folha === "" && !menu.aberto; onActivated: app.abrirConectar() }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "perguntaremota" && folhaDaPerguntaRemota.forma.simNao; onActivated: folhaDaPerguntaRemota.confirmar() }
    Shortcut { sequences: ["Ctrl+D"]; enabled: app.livre; onActivated: app.acao("favorito") }
    Shortcut { sequences: ["Shift+F10"]; enabled: app.livre; onActivated: app.menuDoTeclado() }
    Shortcut { sequences: ["Ctrl+B"]; enabled: app.folha === "" && !menu.aberto; onActivated: { app.naLateral = !app.naLateral; if (app.naLateral && lateral.cursor < 0) lateral.andar(1); } }
    // --- o painel pelo teclado ----------------------------------------------
    // No FloatingWindow do Quickshell a primeira tecla sem modificador não
    // chega ao Keys do painel (medido), mas chega a um Shortcut. Com a janela
    // livre, navegação e ação são Shortcut, uma vez cada, e todas entregam ao
    // mesmo tratador do painel. O Keys fica para o digitar-para-pular.
    function porAtalho(tecla, modificadores) {
        const e = { key: tecla, modifiers: modificadores, text: "", accepted: false };
        painel.focar();
        painel.tecla(e);
    }
    Shortcut { sequences: ["Down"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Down, 0) }
    Shortcut { sequences: ["Shift+Down"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Down, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Down"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Down, Qt.ControlModifier) }
    Shortcut { sequences: ["Up"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Up, 0) }
    Shortcut { sequences: ["Shift+Up"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Up, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Up"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Up, Qt.ControlModifier) }
    Shortcut { sequences: ["Left"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Left, 0) }
    Shortcut { sequences: ["Shift+Left"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Left, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Left"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Left, Qt.ControlModifier) }
    Shortcut { sequences: ["Right"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Right, 0) }
    Shortcut { sequences: ["Shift+Right"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Right, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Right"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Right, Qt.ControlModifier) }
    Shortcut { sequences: ["Home"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Home, 0) }
    Shortcut { sequences: ["Shift+Home"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Home, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Home"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Home, Qt.ControlModifier) }
    Shortcut { sequences: ["End"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_End, 0) }
    Shortcut { sequences: ["Shift+End"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_End, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+End"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_End, Qt.ControlModifier) }
    Shortcut { sequences: ["PgUp"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_PageUp, 0) }
    Shortcut { sequences: ["PgDown"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_PageDown, 0) }
    Shortcut { sequences: ["Space"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Space, 0) }
    Shortcut { sequences: ["Delete"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Delete, 0) }
    Shortcut { sequences: ["Backspace"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Backspace, 0) }
    Shortcut { sequences: ["Tab"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Tab, 0) }
    Shortcut { sequences: ["Menu"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Menu, 0) }
    Shortcut { sequences: ["F1"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F1, 0) }
    Shortcut { sequences: ["F2"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F2, 0) }
    Shortcut { sequences: ["F3"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F3, 0) }
    Shortcut { sequences: ["F4"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F4, 0) }
    Shortcut { sequences: ["F5"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F5, 0) }
    Shortcut { sequences: ["F6"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_F6, 0) }
    Shortcut { sequences: ["Escape"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Escape, 0) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Return, 0) }
    Shortcut { sequences: ["Shift+Delete"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Delete, Qt.ShiftModifier) }
    Shortcut { sequences: ["Ctrl+Space"]; enabled: app.livre; onActivated: app.porAtalho(Qt.Key_Space, Qt.ControlModifier) }
    Shortcut { sequences: ["F1"]; enabled: !app.livre; onActivated: app.folha = app.folha === "atalhos" ? "" : app.folha === "" ? "atalhos" : app.folha }
    Shortcut { sequences: ["Escape"]; enabled: !app.livre; onActivated: app.fechar() }
    // O menu de contexto, pelo teclado.
    Shortcut { sequences: ["Up"]; enabled: menu.aberto; onActivated: menu.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: menu.aberto; onActivated: menu.andar(1) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: menu.aberto; onActivated: menu.confirmar() }
    // A barra lateral, pelo teclado (Ctrl+B).
    Shortcut { sequences: ["Up"]; enabled: app.naLateral && app.folha === "" && !menu.aberto; onActivated: lateral.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.naLateral && app.folha === "" && !menu.aberto; onActivated: lateral.andar(1) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.naLateral && app.folha === "" && !menu.aberto; onActivated: lateral.confirmar() }
    // As folhas.
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "pergunta" && !menu.aberto; onActivated: folhaDaPergunta.confirmado() }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "abrircom"; onActivated: folhaDoAbrirCom.confirmar() }
    Shortcut { sequences: ["Up"]; enabled: app.folha === "abrircom"; onActivated: folhaDoAbrirCom.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.folha === "abrircom"; onActivated: folhaDoAbrirCom.andar(1) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "atalhos" || app.folha === "propriedades"; onActivated: app.fecharFolha() }
    // A prévia rápida: Espaço fecha, as setas passam para o vizinho, Enter abre.
    Shortcut { sequences: ["Space"]; autoRepeat: false; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.fecharFolha() }
    Shortcut { sequences: ["Left", "Up"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.vizinhoDaPrevia(-1) }
    Shortcut { sequences: ["Right", "Down"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: app.vizinhoDaPrevia(1) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "previa" && !menu.aberto; onActivated: folhaDaPrevia.abrirPedido() }
    Shortcut { sequences: ["Ctrl++", "Ctrl+="]; enabled: app.folha === "previa"; onActivated: folhaDaPrevia.zoom(1.2) }
    Shortcut { sequences: ["Ctrl+-"]; enabled: app.folha === "previa"; onActivated: folhaDaPrevia.zoom(1 / 1.2) }
    Shortcut { sequences: ["Ctrl+0"]; enabled: app.folha === "previa"; onActivated: folhaDaPrevia.escala = 0 }
    Shortcut { sequences: ["S"]; enabled: app.folha === "conflito" && !folhaDoConflito.renomeando; onActivated: folhaDoConflito.responder("substituir") }
    Shortcut { sequences: ["P"]; enabled: app.folha === "conflito" && !folhaDoConflito.renomeando; onActivated: folhaDoConflito.responder("pular") }
    Shortcut { sequences: ["M"]; enabled: app.folha === "conflito" && !folhaDoConflito.renomeando; onActivated: folhaDoConflito.responder("manter") }
    Shortcut { sequences: ["R"]; enabled: app.folha === "conflito" && !folhaDoConflito.renomeando; onActivated: folhaDoConflito.responder("renomear") }
    Shortcut { sequences: ["T"]; enabled: app.folha === "conflito" && !folhaDoConflito.renomeando; onActivated: folhaDoConflito.todos = !folhaDoConflito.todos }

    // --- a tela ---------------------------------------------------------------------
    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        Barra {
            id: barra
            Layout.fillWidth: true
            aba: app.painel.aba
            home: Motor.home
            remotos: app.remotos
            visao: app.painel.visao
            estreita: app.estreita
            onVoltar: app.painel.voltar()
            onAvancar: app.painel.avancar()
            onSubir: app.painel.subir()
            onIr: caminho => {
                // Uma migalha de dentro de um compactado ("arquivo//pasta") vai como está.
                const limpo = Abas.eLixeira(caminho) || (Abas.eCompactado(caminho) && caminho.charAt(0) === "/") ? caminho : Arquivos.limpar(caminho, Motor.home);
                barra.sair();
                // Um endereço (sftp://…) digitado no caminho conecta e abre.
                if (limpo === null && Remoto.erroDoEndereco(caminho.trim()) === "")
                    app.conectar(caminho.trim(), onde => app.painel.ir(onde));
                else if (limpo === null)
                    app.avisar("isso não é um caminho: escreva a partir de / ou de ~");
                else
                    app.painel.ir(limpo);
                app.painel.focar();
            }
            // Tab no campo: o motor lista a pasta de cima e o nome é completado.
            onCompletar: texto => {
                const inteiro = texto.charAt(0) === "~" ? Motor.home + texto.substring(1) : texto;
                const corte = inteiro.lastIndexOf("/");
                if (corte < 0)
                    return;
                const pasta = Arquivos.limpar(inteiro.substring(0, corte + 1), Motor.home);
                if (pasta === null)
                    return;
                const itens = [];
                Motor.pedir({ op: "listar", caminho: pasta }, r => {
                    for (const it of r.itens || [])
                        itens.push(it);
                    if (r.erro || r.fim !== true)
                        return;
                    const c = Arquivos.completar(inteiro, itens);
                    barra.completado(c.texto, c.opcoes);
                });
            }
            onBuscaMudou: texto => app.painel.buscar(texto)
            onBuscaFechada: app.fecharBusca()
            onVisaoEscolhida: v => app.escolherVisao(v)
            onMenuPedido: (x, y) => app.menuPrincipal(x, y)
            onLargado: {
                barra.sair();
                app.painel.focar();
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0

            Lateral {
                id: lateral
                Layout.fillHeight: true
                // Estreita, a barra lateral recolhe para os ícones.
                Layout.preferredWidth: app.estreita ? 44 : app.lateralPx
                recolhida: app.estreita
                lugares: app.lugares
                favoritos: app.favoritos
                discos: app.discos
                remotos: app.remotos
                naLixeira: app.naLixeira
                atual: Arquivos.pastaReal(app.painel.caminho)
                comTeclado: app.naLateral
                onAberto: linha => app.abrirLugar(linha, "")
                onMenuPedido: (linha, x, y) => app.menuDoLugar(linha, x, y)
                onFavoritoMovido: (de, para) => Motor.pedir({ op: "favorito-mover", de: de, para: para }, app.chegaramFavoritos)
                onEjetado: linha => {
                    if (linha.tipo === "remoto")
                        app.desconectar(linha);
                    else
                        app.soltarDisco(linha, linha.removivel);
                }
                onSoltado: (linha, caminhos, acao) => app.soltar(caminhos, linha.caminho, acao)
            }
            // A beira da barra lateral: arrastar muda a largura.
            Rectangle {
                Layout.fillHeight: true
                Layout.preferredWidth: 1
                color: Theme.hairline
                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -3
                    enabled: !app.estreita
                    cursorShape: Qt.SplitHCursor
                    onPositionChanged: mouse => {
                        if (pressed)
                            app.lateralPx = Math.max(120, Math.min(480, mapToItem(app, mouse.x, 0).x));
                    }
                    onReleased: app.guardarEstado()
                }
            }

            Painel {
                id: painelA
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumWidth: 200
                janela: app
                numero: 0
                ativo: app.painel === painelA && !app.naLateral
                pronto: app.pronto
                prefs: app.prefs
                onMudou: app.guardarEstado()
            }
            Rectangle {
                visible: app.duploVisivel
                Layout.fillHeight: true
                Layout.preferredWidth: 1
                color: Theme.line
            }
            // O segundo painel só existe (e só lista) com o painel duplo ligado
            // e largura para os dois.
            Painel {
                id: painelB
                visible: app.duploVisivel
                Layout.fillWidth: true
                Layout.fillHeight: true
                Layout.minimumWidth: 200
                janela: app
                numero: 1
                ativo: app.painel === painelB && !app.naLateral
                pronto: app.pronto && app.duploVisivel
                prefs: app.prefs
                onMudou: app.guardarEstado()
            }
        }

        Tarefas {
            Layout.fillWidth: true
            lista: app.tarefas
            onPausar: (t, pausar) => Motor.pedir({ op: "tarefa-pausar", tarefa: t, pausar: pausar }, null)
            onCancelar: t => Motor.pedir({ op: "tarefa-cancelar", tarefa: t }, null)
            onLida: t => app.tarefas = app.tarefas.filter(x => x.tarefa !== t)
            onDetalhes: t => app.perguntar({
                titulo: Arquivos.plural((t.erros || []).length, "erro", "erros"), frase: t.titulo || "",
                lista: (t.erros || []).map(e => Arquivos.nomeDe(e.caminho) + ": " + e.erro), botao: "ok",
                feito: () => app.tarefas = app.tarefas.filter(x => x.tarefa !== t.tarefa)
            })
        }

        // A barra de estado.
        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 24
            color: Theme.surface

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }
            Text {
                x: 12
                width: parent.width - 24 - (dica.visible ? dica.width + 12 : 0)
                anchors.verticalCenter: parent.verticalCenter
                elide: Text.ElideRight
                text: Motor.caiu ? "O motor parou de responder (bin/mkfiles-motor). Ctrl+R tenta de novo."
                    : app.aviso !== "" ? app.aviso
                    : Arquivos.estado(app.painel.total, Arquivos.soma(app.painel.selecionados()), app.painel.lixeira ? null : app.painel.espaco, app.painel.carregando)
                color: Motor.caiu || app.aviso !== "" ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 10
            }
            Rotulo {
                id: dica
                visible: !app.estreita && app.aviso === ""
                anchors.right: parent.right
                anchors.rightMargin: 12
                anchors.verticalCenter: parent.verticalCenter
                text: Motor.exemplo !== "" ? "exemplo  ·  f1 atalhos" : "f1 atalhos"
            }
        }
    }

    // --- por cima: o menu e as folhas --------------------------------------------------
    MenuContexto {
        id: menu
        anchors.fill: parent
        z: 20
        onEscolhido: id => app.acao(id)
        onAbertoChanged: {
            if (!aberto)
                app.painel.focar();
        }
    }

    Conflito {
        id: folhaDoConflito
        anchors.fill: parent
        z: 10
        visible: app.folha === "conflito"
        dados: app.conflitos.length > 0 ? app.conflitos[0] : null
        onRespondido: (resposta, nome, todos) => app.responderConflito(resposta, nome, todos)
        onFechado: app.responderConflito("cancelar", "", false)
    }
    Propriedades {
        id: folhaDasProps
        anchors.fill: parent
        z: 10
        visible: app.folha === "propriedades"
        onFechado: app.fecharFolha()
        onAplicado: mudanca => {
            const item = folhaDasProps.um;
            folhaDasProps.erro = "";
            Motor.pedir({ op: "permissoes", caminhos: [item.caminho], modo: mudanca.modo, executavel: null, grupo: mudanca.grupo, recursivo: mudanca.recursivo }, r => {
                if (r.erro) {
                    folhaDasProps.erro = r.erro;
                    return;
                }
                if (r.itens && r.itens.length > 0 && folhaDasProps.dados)
                    folhaDasProps.dados = Object.assign({}, folhaDasProps.dados, { itens: r.itens });
                if (r.erros && r.erros.length > 0)
                    folhaDasProps.erro = Arquivos.plural(r.erros.length, "item não mudou", "itens não mudaram") + ": " + r.erros[0].erro;
            });
        }
    }
    AbrirCom {
        id: folhaDoAbrirCom
        anchors.fill: parent
        z: 10
        visible: app.folha === "abrircom"
        onFechado: app.fecharFolha()
        onPedido: todos => app.pedirApps(todos)
        onEscolhido: (escolha, definir) => {
            const alvo = folhaDoAbrirCom.alvo;
            if (definir)
                Motor.pedir({ op: "definir-padrao", mime: alvo.mime, app: escolha.id }, r => {
                    if (r.erro)
                        app.avisar(r.erro);
                });
            Motor.pedir({ op: "abrir", caminhos: alvo.caminhos, app: escolha.id }, r => {
                // Aqui o app foi escolhido: o que voltar sem app não abre a folha de novo.
                if (r.erro)
                    app.avisar(r.erro);
                for (const l of r.lancar || [])
                    app.executar(l.argv);
            });
            app.fecharFolha();
        }
    }
    Pergunta {
        id: folhaDaPergunta
        anchors.fill: parent
        z: 10
        visible: app.folha === "pergunta"
        titulo: app.pergunta ? app.pergunta.titulo : ""
        frase: app.pergunta ? app.pergunta.frase : ""
        lista: app.pergunta ? app.pergunta.lista || [] : []
        botao: app.pergunta ? app.pergunta.botao : "confirmar"
        onFechado: app.fecharFolha()
        onConfirmado: {
            const feito = app.pergunta ? app.pergunta.feito : null;
            app.fecharFolha();
            if (feito)
                feito();
        }
    }
    Nome {
        id: folhaDoNome
        anchors.fill: parent
        z: 10
        visible: app.folha === "nome"
        rotulo: app.nomeDados ? app.nomeDados.rotulo || "nome" : "nome"
        titulo: app.nomeDados ? app.nomeDados.titulo : ""
        frase: app.nomeDados ? app.nomeDados.frase || "" : ""
        inicial: app.nomeDados ? app.nomeDados.inicial || "" : ""
        botao: app.nomeDados ? app.nomeDados.botao || "criar" : "criar"
        eNome: app.nomeDados ? app.nomeDados.eNome !== false : true
        vazio: app.nomeDados && app.nomeDados.vazio ? app.nomeDados.vazio : "escreva um padrão, como *.jpg"
        modelos: app.nomeDados ? app.nomeDados.modelos || [] : []
        onFechado: app.fecharFolha()
        onConfirmado: (texto, modelo) => {
            if (app.nomeDados && app.nomeDados.feito(texto, modelo) === true)
                app.fecharFolha();
        }
    }
    Atalhos {
        anchors.fill: parent
        z: 10
        visible: app.folha === "atalhos"
        onFechado: app.fecharFolha()
    }
    EmLote {
        id: folhaDoLote
        anchors.fill: parent
        z: 10
        visible: app.folha === "lote"
        onFechado: app.fecharFolha()
        onConfirmado: pares => {
            folhaDoLote.erro = "";
            Motor.pedir({ op: "renomear-lote", pares: pares }, r => {
                if (r.erro) {
                    folhaDoLote.erro = Lote.recusa(r);
                    return;
                }
                app.painel.aSelecionar = pares.map(p => Arquivos.juntar(Arquivos.pai(p[0]), p[1]));
                if (app.folha === "lote")
                    app.fecharFolha();
            });
        }
    }
    Compactar {
        id: folhaDoCompactar
        anchors.fill: parent
        z: 10
        visible: app.folha === "compactar"
        onFechado: app.fecharFolha()
        onConfirmado: (nome, formato, nivel, segredo) => {
            const d = app.compactarDados;
            if (!d)
                return;
            app.painel.aSelecionar = [Arquivos.juntar(d.destino, nome)];
            app.tarefa({ op: "compactar", origens: d.origens, destino: d.destino, nome: nome, formato: formato, nivel: nivel, senha: segredo });
            app.compactarDados = null;
            app.fecharFolha();
        }
    }
    Conectar {
        id: folhaDeConectar
        anchors.fill: parent
        z: 10
        visible: app.folha === "conectar"
        hosts: app.hostsDoSsh
        rede: app.rede
        conectando: app.conexao !== null
        onFechado: app.fecharFolha()
        onCancelado: app.cancelarConexao()
        onPedido: uri => app.conectar(uri, caminho => {
            app.painel.ir(caminho);
            app.painel.focar();
        })
    }
    PerguntaRemota {
        id: folhaDaPerguntaRemota
        anchors.fill: parent
        z: 11
        visible: app.folha === "perguntaremota"
        dados: app.perguntaRemota
        onFechado: app.fecharFolha()
        onRespondido: resposta => app.responderRemota(resposta)
    }
    PedeSenha {
        id: folhaDaSenha
        anchors.fill: parent
        z: 10
        visible: app.folha === "pedesenha"
        titulo: app.segredoDados ? app.segredoDados.titulo || "" : ""
        onFechado: app.fecharFolha()
        onConfirmado: segredo => {
            const feito = app.segredoDados ? app.segredoDados.feito : null;
            app.fecharFolha();
            if (feito)
                feito(segredo);
        }
    }
    Previa {
        id: folhaDaPrevia
        anchors.fill: parent
        z: 10
        visible: app.folha === "previa"
        onFechado: app.fecharFolha()
        onVizinho: delta => app.vizinhoDaPrevia(delta)
        onAbrirPedido: {
            app.fecharFolha();
            app.painel.abrirSelecao();
        }
    }
}
