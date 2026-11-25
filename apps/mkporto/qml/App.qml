import QtQuick
import QtQuick.Layouts
import "Porto.js" as Porto

// A janela inteira: no alto, as três áreas (Hosts, Chaves e Máquinas) e as ações; no
// meio, a lista de hosts com o detalhe ao lado (ou, numa janela estreita, um
// de cada vez, com um "voltar"); por cima, as folhas (formulário, confirmação,
// escolha de chave, atalhos). Tudo o que aparece vem do motor.
Rectangle {
    id: app
    color: Theme.ground

    property var hosts: []
    property var chaves: []
    // { apelido: { sondando, responde, ms, motivo } } (Porto.aplicaSonda)
    property var sondas: ({})
    property int sondando: 0
    property bool pronto: false
    property string erro: ""
    property string apelido: ""
    // "hosts" | "chaves" | "maquinas"
    property string area: "hosts"
    // Numa janela estreita: se o detalhe está na frente da lista.
    property bool detalheAberto: false
    // "" | "novo" | "editar" | "remover" | "chave" | "gerar" | "atalhos"
    property string folha: ""
    property string erroDaFolha: ""
    property bool ocupado: false
    property string recado: ""
    // Se a janela está à vista (quem diz é o shell.qml): a área Máquinas só se
    // atualiza sozinha com ela visível.
    property bool janelaVisivel: true
    // O teste de entrada: { apelido, testando } e depois a resposta do `entrar`.
    // É de um host só: some ao trocar de host.
    property var entrada: null
    onApelidoChanged: entrada = null

    readonly property bool estreita: Porto.estreita(width)
    readonly property bool apertada: Porto.apertada(width)
    readonly property var visiveis: Porto.visiveis(hosts, lista.busca)
    readonly property var host: Porto.escolhido(hosts, apelido)
    // Nenhuma folha aberta, nem as da área Máquinas (que cuida das dela).
    readonly property bool livre: folha === "" && maquinas.folha === ""
    readonly property bool nasMaquinas: livre && area === "maquinas"

    // ── o motor ────────────────────────────────────────────────────────

    // Relê os hosts e as chaves. depois() roda quando a lista chegou.
    function ler(depois) {
        Motor.pedir(["listar"], r => {
            if (r.erro) {
                app.erro = r.erro;
                return;
            }
            app.erro = "";
            app.hosts = r.hosts || [];
            app.pronto = true;
            if (!Porto.escolhido(app.hosts, app.apelido))
                app.apelido = Porto.vizinho(Porto.visiveis(app.hosts, ""), "", 1);
            if (depois)
                depois();
        });
        Motor.pedir(["chaves"], r => {
            if (!r.erro)
                app.chaves = r.chaves || [];
        });
    }

    // Sonda estes apelidos (todos, se não vier lista). Uma vez por pedido:
    // ao abrir, no F5 e depois de guardar. Nunca em laço.
    function sondar(apelidos) {
        const quais = apelidos || app.hosts.map(h => h.apelido);
        if (quais.length === 0 || (!apelidos && app.sondando > 0))
            return;
        app.sondas = Porto.comecaSonda(app.sondas, quais);
        app.sondando++;
        Motor.tarefa(["sondar", "--"].concat(quais), r => {
            app.sondas = Porto.aplicaSonda(app.sondas, r);
        }, () => {
            app.sondando--;
            // O motor parou sem dizer de alguém: ninguém fica "sondando" para sempre.
            if (app.sondando === 0)
                app.sondas = Porto.encerraSonda(app.sondas);
        });
    }

    function conectar() {
        if (!app.host)
            return;
        Motor.pedir(["comando", "conectar", "--", app.host.apelido], r => {
            app.recado = r.erro ? r.erro : (Motor.abrir(r) || "abrindo " + r.texto + " num terminal");
        });
    }

    function copiarComando() {
        if (!app.host)
            return;
        Motor.pedir(["comando", "conectar", "--", app.host.apelido], r => {
            if (r.erro)
                app.recado = r.erro;
            else
                app.copiar(r.texto, "copiado: " + r.texto);
        });
    }

    function enviarChave(nome) {
        if (!app.host)
            return;
        app.folha = "";
        Motor.pedir(["comando", "chave", "--", app.host.apelido, nome], r => {
            app.recado = r.erro ? r.erro : (Motor.abrir(r) || "enviando " + nome + " num terminal");
        });
    }

    // Relê só as chaves (depois de gerar uma num terminal, ninguém avisa).
    function lerChaves() {
        Motor.pedir(["chaves"], r => {
            if (r.erro) {
                app.recado = r.erro;
                return;
            }
            app.chaves = r.chaves || [];
            app.recado = Porto.plural(app.chaves.length, "chave pública", "chaves públicas") + " em ~/.ssh";
        });
    }

    // Testa se a pessoa entra no host com o que já tem, sem pedir senha. Só
    // quando ela pede; o resultado fica no detalhe.
    function testarEntrada() {
        if (!app.host || (app.entrada !== null && app.entrada.testando === true))
            return;
        const alvo = app.host.apelido;
        app.entrada = { apelido: alvo, testando: true };
        Motor.pedir(["entrar", "--", alvo], r => {
            // Trocaram de host no meio: a resposta não é mais de ninguém.
            if (app.entrada === null || app.entrada.apelido !== alvo)
                return;
            app.entrada = r.erro ? { apelido: alvo, erro: r.erro } : r;
        });
    }

    // A chave nova: o motor confere o nome e diz o que abrir; quem gera é o
    // ssh-keygen, no terminal.
    function gerarChave(form) {
        app.ocupado = true;
        app.erroDaFolha = "";
        Motor.pedir(["comando", "gerar", "--"].concat(Porto.argsDaChave(form)), r => {
            app.ocupado = false;
            if (r.erro) {
                app.erroDaFolha = r.erro;
                return;
            }
            app.folha = "";
            app.recado = Motor.abrir(r) || "gerando a chave num terminal; quando terminar, atualize (F5)";
        });
    }

    function guardar(pedido) {
        app.ocupado = true;
        app.erroDaFolha = "";
        Motor.pedir(["guardar"], r => {
            app.ocupado = false;
            if (r.erro) {
                app.erroDaFolha = r.erro;
                return;
            }
            app.folha = "";
            app.recado = "guardado: " + pedido.apelido;
            app.apelido = pedido.apelido;
            app.ler(() => app.sondar([pedido.apelido]));
        }, JSON.stringify(pedido));
    }

    function remover() {
        if (!app.host || !Porto.editavel(app.host) || app.ocupado)
            return;
        const alvo = app.host.apelido;
        app.ocupado = true;
        app.erroDaFolha = "";
        Motor.pedir(["remover", "--", alvo], r => {
            app.ocupado = false;
            if (r.erro) {
                app.erroDaFolha = r.erro;
                return;
            }
            app.folha = "";
            app.detalheAberto = false;
            app.recado = "removido: " + alvo;
            app.ler();
        });
    }

    // ── a janela ───────────────────────────────────────────────────────

    function copiar(texto, recado) {
        prancheta.text = texto;
        prancheta.selectAll();
        prancheta.copy();
        prancheta.text = "";
        app.recado = recado;
    }

    function abrirFolha(qual) {
        // Na área Máquinas, "novo" é a máquina nova.
        if (qual === "novo" && app.area === "maquinas") {
            maquinas.abrir("nova");
            return;
        }
        if (qual === "editar" || qual === "remover") {
            if (!Porto.editavel(app.host)) {
                app.recado = app.host ? "este host é do seu ~/.ssh/config: o MkPorto não edita nem remove" : "";
                return;
            }
        }
        if (qual === "chave" && !app.host)
            return;
        app.erroDaFolha = "";
        app.folha = qual;
        if (qual === "novo")
            formulario.abrir(Porto.formularioVazio(), "");
        else if (qual === "editar")
            formulario.abrir(Porto.formularioDe(app.host), app.host.apelido);
        else if (qual === "gerar")
            novaChave.abrir();
    }

    function fechar() {
        if (app.folha !== "") {
            app.folha = "";
            return;
        }
        if (maquinas.folha !== "" || app.area === "maquinas") {
            maquinas.fechar();
            return;
        }
        if (app.estreita && app.detalheAberto) {
            app.detalheAberto = false;
            return;
        }
        if (lista.busca !== "")
            lista.busca = "";
    }

    function escolher(qual) {
        app.apelido = qual;
        if (app.estreita)
            app.detalheAberto = true;
    }

    function andar(passo) {
        app.apelido = Porto.vizinho(app.visiveis, app.apelido, passo);
    }

    Component.onCompleted: ler(() => sondar())

    Shortcut { sequences: ["Ctrl+F"]; enabled: app.livre; onActivated: { app.area = "hosts"; app.detalheAberto = false; lista.focarBusca(); } }
    Shortcut { sequences: ["Ctrl+N"]; enabled: app.livre; onActivated: app.abrirFolha("novo") }
    Shortcut { sequences: ["Ctrl+E"]; enabled: app.livre && app.area === "hosts"; onActivated: app.abrirFolha("editar") }
    Shortcut { sequences: ["Ctrl+K"]; enabled: app.livre && app.area === "hosts"; onActivated: app.abrirFolha("chave") }
    Shortcut { sequences: ["Delete"]; enabled: app.livre && app.area === "hosts"; onActivated: app.abrirFolha("remover") }
    Shortcut { sequences: ["Ctrl+Shift+C"]; enabled: app.livre && app.area === "hosts"; onActivated: app.copiarComando() }
    Shortcut { sequences: ["Ctrl+T"]; enabled: app.livre && app.area === "hosts"; onActivated: app.testarEntrada() }
    Shortcut { sequences: ["Ctrl+G"]; enabled: app.livre; onActivated: { app.area = "chaves"; app.abrirFolha("gerar"); } }
    // F5 refaz o que a área mostra: sonda os hosts, ou relê as chaves.
    Shortcut { sequences: ["F5"]; enabled: app.livre && app.area === "hosts"; onActivated: app.sondar() }
    Shortcut { sequences: ["F5"]; enabled: app.livre && app.area === "chaves"; onActivated: app.lerChaves() }
    Shortcut { sequences: ["Ctrl+1"]; enabled: app.livre; onActivated: app.area = "hosts" }
    Shortcut { sequences: ["Ctrl+2"]; enabled: app.livre; onActivated: app.area = "chaves" }
    Shortcut { sequences: ["Ctrl+3"]; enabled: app.livre; onActivated: app.area = "maquinas" }
    // Na área Máquinas: Enter liga, as setas andam, F5 confere de novo, Delete
    // abre a folha de remover; com a pergunta do forçar na tela, Enter confirma.
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.nasMaquinas; onActivated: maquinas.principal() }
    Shortcut { sequences: ["Up"]; enabled: app.nasMaquinas; onActivated: maquinas.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.nasMaquinas; onActivated: maquinas.andar(1) }
    Shortcut { sequences: ["F5"]; enabled: app.nasMaquinas; onActivated: maquinas.conferir() }
    Shortcut { sequences: ["Delete"]; enabled: app.nasMaquinas; onActivated: maquinas.abrir("remover") }
    Shortcut { sequences: ["Return", "Enter"]; enabled: maquinas.folha === "forcar"; onActivated: maquinas.forcar() }
    Shortcut { sequences: ["Up"]; enabled: app.livre && app.area === "hosts"; onActivated: app.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.livre && app.area === "hosts"; onActivated: app.andar(1) }
    // Enter conecta; com a pergunta do remover na tela, confirma.
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.livre && app.area === "hosts"; onActivated: app.conectar() }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "remover"; onActivated: app.remover() }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.confirmar() }
    Shortcut { sequences: ["Up"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.folha === "chave"; onActivated: escolhaDeChave.andar(1) }
    Shortcut { sequences: ["Escape"]; onActivated: app.fechar() }
    Shortcut { sequences: ["F1"]; onActivated: app.folha = app.folha === "atalhos" ? "" : app.folha === "" ? "atalhos" : app.folha }

    // A área de transferência, pelo próprio Qt: um campo de texto que nunca
    // aparece, só para copiar.
    TextEdit {
        id: prancheta
        visible: false
        textFormat: TextEdit.PlainText
        activeFocusOnPress: false
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 14

        // ── o alto ─────────────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Row {
                visible: !app.apertada
                spacing: 7
                Text {
                    id: marca
                    text: "Mk"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                }
                Text {
                    anchors.baseline: marca.baseline
                    text: "porto"
                    color: Theme.copper
                    font.family: Theme.serif
                    font.pixelSize: 16
                    font.italic: true
                }
            }
            Segmento {
                Layout.leftMargin: app.apertada ? 0 : 8
                opcoes: [{ v: "hosts", texto: "Hosts" }, { v: "chaves", texto: "Chaves" }, { v: "maquinas", texto: "Máquinas" }]
                valor: app.area
                onEscolhido: v => app.area = v
            }
            Item { Layout.fillWidth: true }
            Botao {
                visible: app.area === "hosts" && !app.apertada
                texto: app.sondando > 0 ? "sondando…" : app.estreita ? "sondar" : "sondar de novo"
                ligado: app.sondando === 0 && app.hosts.length > 0
                onAcionado: app.sondar()
            }
            Botao {
                texto: app.area === "maquinas" ? (app.estreita ? "nova" : "nova máquina") : app.estreita ? "novo" : "novo host"
                forte: true
                ligado: app.area !== "maquinas" || maquinas.pronto
                onAcionado: app.abrirFolha("novo")
            }
        }

        Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 1; color: Theme.hairline }

        // ── o meio ─────────────────────────────────────────────────────
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            // O motor ainda não respondeu, ou não respondeu: a janela diz o
            // que houve em vez de ficar vazia.
            Text {
                anchors.centerIn: parent
                width: parent.width - 60
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.WordWrap
                visible: !app.pronto && app.area !== "maquinas"
                text: app.erro !== "" ? app.erro : "lendo o seu ~/.ssh/config…"
                color: app.erro !== "" ? Theme.copper : Theme.textFaint
                font.family: Theme.serif
                font.pixelSize: 16
                font.italic: true
            }

            // Sem host nenhum: explica e oferece o primeiro.
            Column {
                anchors.centerIn: parent
                width: Math.min(parent.width - 40, 440)
                spacing: 14
                visible: app.pronto && app.area === "hosts" && app.hosts.length === 0

                Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    text: "Nenhum host ainda"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 24
                    font.weight: Font.Light
                }
                Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.WordWrap
                    text: "O MkPorto mostra as máquinas do seu ~/.ssh/config e guarda as que você criar aqui. Conectar abre um terminal com o ssh; senha nenhuma passa por esta janela."
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Botao {
                    anchors.horizontalCenter: parent.horizontalCenter
                    texto: "novo host"
                    forte: true
                    onAcionado: app.abrirFolha("novo")
                }
            }

            Item {
                anchors.fill: parent
                visible: app.pronto && app.area === "hosts" && app.hosts.length > 0

                Lista {
                    id: lista
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    width: app.estreita ? parent.width : Math.max(250, Math.round(parent.width * 0.38))
                    visible: !app.estreita || !app.detalheAberto
                    hosts: app.visiveis
                    total: app.hosts.length
                    sondas: app.sondas
                    escolhido: app.apelido
                    onEscolher: apelido => app.escolher(apelido)
                    onRemover: app.abrirFolha("remover")
                    onConectar: apelido => {
                        app.apelido = apelido;
                        app.conectar();
                    }
                }
                Rectangle {
                    id: divisa
                    visible: !app.estreita
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    x: lista.width + 14
                    width: 1
                    color: Theme.hairline
                }
                Detalhe {
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.right: parent.right
                    anchors.left: parent.left
                    anchors.leftMargin: app.estreita ? 0 : lista.width + 30
                    visible: !app.estreita || app.detalheAberto
                    host: app.host
                    sondas: app.sondas
                    estreita: app.estreita
                    entrada: app.entrada
                    onTestarEntrada: app.testarEntrada()
                    onVoltar: app.detalheAberto = false
                    onConectar: app.conectar()
                    onCopiar: app.copiarComando()
                    onEnviarChave: app.abrirFolha("chave")
                    onEditar: app.abrirFolha("editar")
                    onRemover: app.abrirFolha("remover")
                }
            }

            Chaves {
                anchors.fill: parent
                visible: app.pronto && app.area === "chaves"
                chaves: app.chaves
                onCopiar: chave => app.copiar(chave.publica, "copiada a chave pública " + chave.nome)
                onNovaChave: app.abrirFolha("gerar")
                onAtualizar: app.lerChaves()
            }

            // As máquinas virtuais: a área cuida do seu estado e das suas folhas
            // (que se abrem por cima da janela inteira), e só se atualiza
            // sozinha enquanto está na frente e a janela visível.
            Maquinas {
                id: maquinas
                anchors.fill: parent
                visible: app.area === "maquinas"
                palco: app
                naFrente: app.area === "maquinas" && app.janelaVisivel
                estreita: app.estreita
                onRecado: texto => app.recado = texto
                onHostCriado: app.ler()
            }
        }

        // ── o pé ───────────────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 12

            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: app.recado !== "" ? app.recado : app.area === "maquinas" ? Porto.resumoDasMaquinas(maquinas.maquinas)
                    : app.pronto ? Porto.resumo(app.hosts, app.sondas) : ""
                color: app.recado !== "" ? Theme.textSoft : Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
            }
            Rotulo { text: "f1 atalhos" }
        }
    }

    // ── as folhas, por cima de tudo ────────────────────────────────────
    Formulario {
        id: formulario
        anchors.fill: parent
        visible: app.folha === "novo" || app.folha === "editar"
        hosts: app.hosts
        chaves: app.chaves
        erro: app.erroDaFolha
        ocupado: app.ocupado
        onGuardar: pedido => app.guardar(pedido)
        onFechado: app.fechar()
    }
    Confirma {
        anchors.fill: parent
        visible: app.folha === "remover"
        apelido: app.host ? app.host.apelido : ""
        erro: app.erroDaFolha
        ocupado: app.ocupado
        onConfirmado: app.remover()
        onFechado: app.fechar()
    }
    Escolha {
        id: escolhaDeChave
        anchors.fill: parent
        visible: app.folha === "chave"
        apelido: app.host ? app.host.apelido : ""
        chaves: app.chaves
        onEscolhida: nome => app.enviarChave(nome)
        onFechado: app.fechar()
    }
    NovaChave {
        id: novaChave
        anchors.fill: parent
        visible: app.folha === "gerar"
        chaves: app.chaves
        erro: app.erroDaFolha
        ocupado: app.ocupado
        onGerar: form => app.gerarChave(form)
        onFechado: app.fechar()
    }
    Atalhos {
        anchors.fill: parent
        visible: app.folha === "atalhos"
        onFechado: app.fechar()
    }
}
