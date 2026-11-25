import QtQuick
import QtQuick.Layouts
import "Porto.js" as Porto

// A área "Máquinas": máquinas virtuais com o QEMU direto. A lista com o
// estado de cada uma, o detalhe com as ações e as folhas (nova máquina,
// snapshots, remover, as perguntas). Quem cria, liga e fala com o QEMU é o
// motor (`vm …`); aqui só se mostra o que ele diz. A tela de uma máquina é a
// janela do próprio QEMU, não esta.
Item {
    id: pagina

    // Onde as folhas se abrem (a janela inteira, não só esta área).
    property Item palco: null
    // A área está na frente e a janela visível: só assim a lista se atualiza.
    property bool naFrente: false
    property bool estreita: false
    signal recado(string texto)
    // Um host novo foi para a área Hosts: ela precisa reler.
    signal hostCriado()

    // A resposta do `vm requisitos` e a do `vm listar`.
    property var requisitos: null
    property var maquinas: []
    property bool lida: false
    property bool listando: false
    // O relógio da última leitura (s), para o "ligada há…".
    property real agora: 0
    property string nome: ""
    property bool detalheAberto: false
    // "" | "nova" | "snapshots" | "remover" | "forcar" | "snapshot"
    property string folha: ""
    property string erroDaFolha: ""
    property bool ocupado: false
    // As duas últimas respostas do `vm uso` da máquina escolhida.
    property var usoAntes: null
    property var usoAgora: null
    property var snapshots: []
    property bool lendoSnapshots: false
    // A pergunta de restaurar ou apagar: { acao, rotulo }.
    property var pedidoDeSnapshot: null

    readonly property var maquina: Porto.maquinaEscolhida(maquinas, nome)
    readonly property bool pronto: requisitos !== null && requisitos.pronto === true
    readonly property bool viva: maquina !== null && maquina.estado !== "desligada"
    onNomeChanged: {
        usoAntes = null;
        usoAgora = null;
    }

    // ── o motor ────────────────────────────────────────────────────────

    // O que há e o que falta para ter máquinas (ao abrir e no "conferir de novo").
    function conferir() {
        Motor.pedir(["vm", "requisitos"], r => {
            if (r.erro) {
                pagina.recado(r.erro);
                return;
            }
            pagina.requisitos = r;
        });
        listar();
    }

    function listar() {
        if (pagina.listando)
            return;
        pagina.listando = true;
        Motor.pedir(["vm", "listar"], r => {
            pagina.listando = false;
            if (r.erro) {
                pagina.recado(r.erro);
                return;
            }
            pagina.maquinas = r.maquinas || [];
            pagina.agora = Date.now() / 1000;
            pagina.lida = true;
            if (!Porto.maquinaEscolhida(pagina.maquinas, pagina.nome))
                pagina.nome = Porto.maquinaVizinha(pagina.maquinas, "", 1);
        });
    }

    function lerUso() {
        if (!pagina.viva)
            return;
        const alvo = pagina.nome;
        Motor.pedir(["vm", "uso", "--", alvo], r => {
            if (r.erro || alvo !== pagina.nome)
                return;
            pagina.usoAntes = pagina.usoAgora;
            pagina.usoAgora = r;
        });
    }

    // Uma batida do relógio: só com a área na frente e a janela visível.
    function pulso() {
        if (pagina.requisitos === null) {
            conferir();
            return;
        }
        listar();
        lerUso();
    }

    // O fim comum de uma ação: o erro (na folha, se houver uma aberta) ou o
    // recado, e a lista de novo.
    function depois(r, frase) {
        pagina.ocupado = false;
        if (r.erro) {
            if (pagina.folha !== "")
                pagina.erroDaFolha = r.erro;
            else
                pagina.recado(r.erro);
            return false;
        }
        pagina.folha = "";
        pagina.recado(r.avisos && r.avisos.length > 0 ? frase + " (" + r.avisos.join("; ") + ")" : frase);
        listar();
        return true;
    }

    // O Enter: o que a tela está pedindo. Sem o QEMU, instalar; sem máquina
    // nenhuma, criar a primeira; senão, ligar a escolhida.
    function principal() {
        if (pagina.requisitos !== null && !pagina.pronto)
            pagina.instalar();
        else if (pagina.pronto && pagina.lida && pagina.maquinas.length === 0)
            pagina.abrir("nova");
        else
            pagina.ligar(false);
    }

    function ligar(semIso) {
        if (!Porto.podeNaMaquina(pagina.maquina, "ligar") || pagina.ocupado)
            return;
        const alvo = pagina.nome;
        pagina.ocupado = true;
        pagina.recado("ligando a " + alvo + "…");
        if (semIso)
            Motor.pedir(["vm", "ligar", "--sem-iso", "--", alvo], r => pagina.depois(r, "a " + alvo + " está ligada, sem a ISO"));
        else
            Motor.pedir(["vm", "ligar", "--", alvo], r => pagina.depois(r, "a " + alvo + " está ligada: a tela dela é a janela do QEMU"));
    }

    function desligar() {
        if (!Porto.podeNaMaquina(pagina.maquina, "desligar"))
            return;
        const alvo = pagina.nome;
        Motor.pedir(["vm", "desligar", "--", alvo], r => pagina.depois(r, "pedido de desligar enviado à " + alvo));
    }

    // Forçar é tirar da tomada: só depois da pergunta.
    function forcar() {
        if (!Porto.podeNaMaquina(pagina.maquina, "forcar") || pagina.ocupado)
            return;
        const alvo = pagina.nome;
        pagina.ocupado = true;
        Motor.pedir(["vm", "desligar", "--forcar", "--", alvo], r => pagina.depois(r, "a " + alvo + " foi desligada na marra"));
    }

    function pausarOuContinuar() {
        const alvo = pagina.nome;
        if (Porto.podeNaMaquina(pagina.maquina, "pausar"))
            Motor.pedir(["vm", "pausar", "--", alvo], r => pagina.depois(r, "a " + alvo + " está pausada"));
        else if (Porto.podeNaMaquina(pagina.maquina, "continuar"))
            Motor.pedir(["vm", "continuar", "--", alvo], r => pagina.depois(r, "a " + alvo + " voltou a andar"));
    }

    function criar(pedido) {
        pagina.ocupado = true;
        pagina.erroDaFolha = "";
        Motor.pedir(["vm", "criar"], r => {
            if (pagina.depois(r, "criada a máquina " + pedido.nome))
                pagina.nome = pedido.nome;
        }, JSON.stringify(pedido));
    }

    function lerSnapshots() {
        const alvo = pagina.nome;
        pagina.lendoSnapshots = true;
        Motor.pedir(["vm", "snapshot", "listar", "--", alvo], r => {
            pagina.lendoSnapshots = false;
            if (r.erro)
                pagina.erroDaFolha = r.erro;
            else if (alvo === pagina.nome)
                pagina.snapshots = r.snapshots || [];
        });
    }

    // acao: "criar" | "restaurar" | "apagar" (as do contrato).
    function snapshot(acao, rotulo) {
        if (["criar", "restaurar", "apagar"].indexOf(acao) < 0 || pagina.ocupado)
            return;
        const alvo = pagina.nome;
        pagina.ocupado = true;
        pagina.erroDaFolha = "";
        Motor.pedir(["vm", "snapshot", acao, "--", alvo, rotulo], r => {
            pagina.ocupado = false;
            pagina.folha = "snapshots";
            if (r.erro) {
                pagina.erroDaFolha = r.erro;
                return;
            }
            pagina.recado("snapshot " + rotulo + ": " + ({ criar: "criado", restaurar: "restaurado", apagar: "apagado" })[acao]);
            pagina.lerSnapshots();
        });
    }

    function remover(digitado) {
        if (!Porto.podeNaMaquina(pagina.maquina, "remover") || pagina.ocupado)
            return;
        const alvo = pagina.nome;
        pagina.ocupado = true;
        pagina.erroDaFolha = "";
        Motor.pedir(["vm", "remover", "--confirmo", digitado, "--", alvo], r => {
            if (pagina.depois(r, "removida a máquina " + alvo))
                pagina.detalheAberto = false;
        });
    }

    function criarHost() {
        if (!pagina.maquina)
            return;
        const alvo = pagina.nome;
        Motor.pedir(["vm", "host", "--", alvo], r => {
            if (pagina.depois(r, "o host " + alvo + " está na área Hosts (127.0.0.1, porta " + pagina.maquina.porta_ssh + ")"))
                pagina.hostCriado();
        });
    }

    // Abre o terminal que o `vm requisitos` mandou abrir, para instalar o QEMU.
    function instalar() {
        if (pagina.requisitos === null)
            return;
        pagina.recado(Motor.abrir(pagina.requisitos.instalar) || "instalando num terminal; quando terminar, use \"conferir de novo\"");
    }

    // ── a janela ───────────────────────────────────────────────────────

    function abrir(qual) {
        if (qual === "nova" && !pagina.pronto)
            return;
        if ((qual === "snapshots" || qual === "remover") && !Porto.podeNaMaquina(pagina.maquina, qual)) {
            pagina.recado(Porto.porQueNao(pagina.maquina, qual));
            return;
        }
        if (qual === "forcar" && !Porto.podeNaMaquina(pagina.maquina, "forcar"))
            return;
        pagina.erroDaFolha = "";
        pagina.folha = qual;
        if (qual === "nova")
            novaMaquina.abrir();
        else if (qual === "remover")
            removeMaquina.abrir();
        else if (qual === "snapshots") {
            pagina.snapshots = [];
            folhaDeSnapshots.abrir();
            lerSnapshots();
        }
    }

    function perguntarSnapshot(acao, rotulo) {
        pagina.pedidoDeSnapshot = { acao: acao, rotulo: rotulo };
        pagina.erroDaFolha = "";
        pagina.folha = "snapshot";
    }

    // Esc: fecha a folha (a pergunta de um snapshot volta para a lista deles),
    // ou volta do detalhe para a lista numa janela estreita.
    function fechar() {
        if (pagina.folha === "snapshot")
            pagina.folha = "snapshots";
        else if (pagina.folha !== "")
            pagina.folha = "";
        else if (pagina.estreita)
            pagina.detalheAberto = false;
    }

    function escolher(qual) {
        pagina.nome = qual;
        if (pagina.estreita)
            pagina.detalheAberto = true;
    }

    function andar(passo) {
        pagina.nome = Porto.maquinaVizinha(pagina.maquinas, pagina.nome, passo);
    }

    // A lista se atualiza sozinha só enquanto a área está na frente e a janela
    // visível: um `vm listar` a cada 2 s, e o `vm uso` da máquina escolhida.
    // Com a área escondida este relógio está parado e nada roda.
    Timer {
        interval: 2000
        repeat: true
        triggeredOnStart: true
        running: pagina.naFrente
        onTriggered: pagina.pulso()
    }

    // ── sem resposta ainda, sem o QEMU, sem máquina nenhuma ────────────
    Text {
        anchors.centerIn: parent
        visible: pagina.requisitos === null || (pagina.pronto && !pagina.lida)
        text: "olhando as máquinas…"
        color: Theme.textFaint
        font.family: Theme.serif
        font.pixelSize: 16
        font.italic: true
    }

    Column {
        anchors.centerIn: parent
        width: Math.min(parent.width - 40, 460)
        spacing: 14
        visible: pagina.requisitos !== null && !pagina.pronto

        Text {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            text: "Falta o QEMU"
            color: Theme.text
            font.family: Theme.serif
            font.pixelSize: 24
            font.weight: Font.Light
        }
        Text {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.WordWrap
            text: "As máquinas virtuais do MkPorto rodam no QEMU, que o Mk não traz. Falta: " + Porto.faltas(pagina.requisitos).join(" e ")
                + ". \"instalar\" abre um terminal com o pacman (qemu-desktop e edk2-ovmf); a senha é digitada lá."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 12
        }
        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: 8
            Botao {
                texto: "instalar"
                forte: true
                onAcionado: pagina.instalar()
            }
            Botao {
                texto: "conferir de novo"
                onAcionado: pagina.conferir()
            }
        }
    }

    Column {
        anchors.centerIn: parent
        width: Math.min(parent.width - 40, 460)
        spacing: 14
        visible: pagina.pronto && pagina.lida && pagina.maquinas.length === 0

        Text {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            text: "Nenhuma máquina ainda"
            color: Theme.text
            font.family: Theme.serif
            font.pixelSize: 24
            font.weight: Font.Light
        }
        Text {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.WordWrap
            text: "Uma máquina virtual é um computador dentro de uma janela: boa para experimentar um sistema sem mexer no seu. Crie a primeira com uma ISO de instalação."
            color: Theme.textDim
            font.family: Theme.sans
            font.pixelSize: 12
        }
        Botao {
            anchors.horizontalCenter: parent.horizontalCenter
            texto: "nova máquina"
            forte: true
            onAcionado: pagina.abrir("nova")
        }
    }

    // ── a lista e o detalhe ────────────────────────────────────────────
    Item {
        anchors.fill: parent
        visible: pagina.pronto && pagina.lida && pagina.maquinas.length > 0

        ColumnLayout {
            id: coluna
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            width: pagina.estreita ? parent.width : Math.max(250, Math.round(parent.width * 0.38))
            visible: !pagina.estreita || !pagina.detalheAberto
            spacing: 10

            ListView {
                id: vista
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.maquinas

                delegate: Rectangle {
                    id: item
                    required property var modelData
                    readonly property bool atual: modelData.nome === pagina.nome
                    width: vista.width
                    height: 46
                    color: atual ? Theme.wash : toque.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)

                    Rectangle {
                        visible: item.atual
                        width: 2
                        height: parent.height
                        color: Theme.copper
                    }
                    Bolinha {
                        x: 14
                        anchors.verticalCenter: parent.verticalCenter
                        estado: Porto.bolinhaDaMaquina(item.modelData.estado)
                    }
                    Column {
                        anchors.left: parent.left
                        anchors.leftMargin: 34
                        anchors.right: situacao.left
                        anchors.rightMargin: 8
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 3

                        Text {
                            width: parent.width
                            elide: Text.ElideRight
                            text: item.modelData.nome
                            color: item.atual ? Theme.text : Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 13
                        }
                        Text {
                            width: parent.width
                            elide: Text.ElideRight
                            text: Porto.resumoDaMaquina(item.modelData)
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                    }
                    Text {
                        id: situacao
                        anchors.right: parent.right
                        anchors.rightMargin: 10
                        anchors.verticalCenter: parent.verticalCenter
                        text: item.modelData.estado === "desligada" ? "" : item.modelData.estado
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 10
                    }
                    MouseArea {
                        id: toque
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: pagina.escolher(item.modelData.nome)
                    }
                }
            }
            // Sem o KVM as máquinas ligam, só que emuladas: um aviso discreto.
            Text {
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                visible: pagina.requisitos !== null && pagina.requisitos.kvm.tem === false
                text: "sem acesso ao KVM: as máquinas ligam, mas ficam lentas"
                color: Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 10
            }
        }
        Rectangle {
            visible: !pagina.estreita
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            x: coluna.width + 14
            width: 1
            color: Theme.hairline
        }

        Flickable {
            id: rolo
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            anchors.left: parent.left
            anchors.leftMargin: pagina.estreita ? 0 : coluna.width + 30
            visible: (!pagina.estreita || pagina.detalheAberto) && pagina.maquina !== null
            clip: true
            contentHeight: detalhe.implicitHeight
            boundsBehavior: Flickable.StopAtBounds

            Column {
                id: detalhe
                width: rolo.width
                spacing: 14

                Botao {
                    visible: pagina.estreita
                    texto: "‹ voltar"
                    onAcionado: pagina.detalheAberto = false
                }
                Column {
                    width: parent.width
                    spacing: 6

                    Rotulo { text: "máquina  /  qemu" }
                    Text {
                        width: parent.width
                        elide: Text.ElideRight
                        text: pagina.maquina ? pagina.maquina.nome : ""
                        color: Theme.text
                        font.family: Theme.serif
                        font.pixelSize: 26
                        font.weight: Font.Light
                    }
                    Row {
                        spacing: 8
                        Bolinha {
                            anchors.verticalCenter: parent.verticalCenter
                            estado: Porto.bolinhaDaMaquina(pagina.maquina ? pagina.maquina.estado : "")
                        }
                        Text {
                            text: Porto.textoDoEstadoDaMaquina(pagina.maquina, pagina.agora)
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                        }
                    }
                    // CPU e memória do QEMU, só enquanto a máquina está de pé.
                    Text {
                        visible: pagina.viva && pagina.usoAgora !== null
                        text: pagina.usoAgora === null ? ""
                            : Porto.textoDoUso(Porto.cpuPct(pagina.usoAntes, pagina.usoAgora), pagina.usoAgora.rss_kb)
                        color: Theme.textSoft
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                }

                Rectangle { width: parent.width; height: 1; color: Theme.hairline }

                Column {
                    width: parent.width
                    spacing: 8

                    Repeater {
                        model: pagina.maquina === null ? [] : [
                            { k: "memória", v: Porto.memoria(pagina.maquina.memoria_mb) },
                            { k: "cpus", v: String(pagina.maquina.cpus) },
                            { k: "disco", v: pagina.maquina.disco_gb + " GB  ·  " + Porto.tamanho(pagina.maquina.disco_bytes) + " em uso" },
                            { k: "iso", v: pagina.maquina.iso || "—" },
                            { k: "firmware", v: pagina.maquina.uefi ? "UEFI" : "BIOS" },
                            { k: "3d", v: pagina.maquina.tres_d ? "ligado" : "desligado" },
                            { k: "porta ssh", v: "127.0.0.1, porta " + pagina.maquina.porta_ssh + " (a 22 de dentro)" },
                            { k: "pid", v: pagina.maquina.pid === null ? "—" : String(pagina.maquina.pid) }
                        ]

                        Item {
                            id: campo
                            required property var modelData
                            width: detalhe.width
                            height: Math.max(16, valor.implicitHeight)

                            Rotulo {
                                width: 84
                                y: 3
                                text: campo.modelData.k
                            }
                            Text {
                                id: valor
                                x: 92
                                width: parent.width - 92
                                wrapMode: Text.WrapAnywhere
                                text: campo.modelData.v
                                color: Theme.text
                                font.family: Theme.mono
                                font.pixelSize: 12
                            }
                        }
                    }
                }

                Rectangle { width: parent.width; height: 1; color: Theme.hairline }

                Flow {
                    width: parent.width
                    spacing: 8

                    Botao {
                        texto: pagina.ocupado ? "um instante…" : "ligar"
                        forte: true
                        ligado: Porto.podeNaMaquina(pagina.maquina, "ligar") && !pagina.ocupado
                        onAcionado: pagina.ligar(false)
                    }
                    Botao {
                        visible: pagina.maquina !== null && pagina.maquina.iso !== null
                        texto: "ligar sem a iso"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "ligar") && !pagina.ocupado
                        onAcionado: pagina.ligar(true)
                    }
                    Botao {
                        texto: "desligar"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "desligar")
                        onAcionado: pagina.desligar()
                    }
                    Botao {
                        texto: "forçar"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "forcar")
                        onAcionado: pagina.abrir("forcar")
                    }
                    Botao {
                        texto: Porto.podeNaMaquina(pagina.maquina, "continuar") ? "continuar" : "pausar"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "pausar") || Porto.podeNaMaquina(pagina.maquina, "continuar")
                        onAcionado: pagina.pausarOuContinuar()
                    }
                    Botao {
                        texto: "snapshots"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "snapshots")
                        onAcionado: pagina.abrir("snapshots")
                    }
                    Botao {
                        texto: "criar host ssh"
                        onAcionado: pagina.criarHost()
                    }
                    Botao {
                        texto: "remover"
                        ligado: Porto.podeNaMaquina(pagina.maquina, "remover")
                        onAcionado: pagina.abrir("remover")
                    }
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: text !== ""
                    text: Porto.porQueNao(pagina.maquina, "snapshots")
                    color: Theme.textFaint
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    text: "A tela da máquina é uma janela do QEMU; fechar essa janela desliga na marra."
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 11
                }
            }
        }
    }

    // ── as folhas, por cima da janela inteira ──────────────────────────
    Item {
        parent: pagina.palco !== null ? pagina.palco : pagina
        anchors.fill: parent
        visible: pagina.folha !== ""

        NovaMaquina {
            id: novaMaquina
            anchors.fill: parent
            visible: pagina.folha === "nova"
            maquinas: pagina.maquinas
            requisitos: pagina.requisitos
            erro: pagina.erroDaFolha
            ocupado: pagina.ocupado
            onCriar: pedido => pagina.criar(pedido)
            onFechado: pagina.fechar()
        }
        Snapshots {
            id: folhaDeSnapshots
            anchors.fill: parent
            visible: pagina.folha === "snapshots"
            nome: pagina.nome
            snapshots: pagina.snapshots
            lendo: pagina.lendoSnapshots
            erro: pagina.erroDaFolha
            ocupado: pagina.ocupado
            onCriar: rotulo => pagina.snapshot("criar", rotulo)
            onRestaurar: rotulo => pagina.perguntarSnapshot("restaurar", rotulo)
            onApagar: rotulo => pagina.perguntarSnapshot("apagar", rotulo)
            onFechado: pagina.fechar()
        }
        RemoveMaquina {
            id: removeMaquina
            anchors.fill: parent
            visible: pagina.folha === "remover"
            nome: pagina.nome
            erro: pagina.erroDaFolha
            ocupado: pagina.ocupado
            onConfirmado: digitado => pagina.remover(digitado)
            onFechado: pagina.fechar()
        }
        Pergunta {
            anchors.fill: parent
            visible: pagina.folha === "forcar"
            titulo: "Desligar " + pagina.nome + " na marra?"
            texto: "É tirar da tomada: o que o sistema de dentro não gravou se perde. Use quando o desligar comum não resolveu."
            acao: "forçar"
            erro: pagina.erroDaFolha
            ocupado: pagina.ocupado
            onConfirmado: pagina.forcar()
            onFechado: pagina.fechar()
        }
        Pergunta {
            anchors.fill: parent
            visible: pagina.folha === "snapshot"
            readonly property bool restaura: pagina.pedidoDeSnapshot !== null && pagina.pedidoDeSnapshot.acao === "restaurar"
            rotulo: "snapshots  /  " + pagina.nome
            titulo: (restaura ? "Restaurar " : "Apagar ") + (pagina.pedidoDeSnapshot ? pagina.pedidoDeSnapshot.rotulo : "") + "?"
            texto: restaura ? "O disco volta a ser o que era nesse snapshot: tudo o que mudou depois dele se perde."
                : "O snapshot some; o disco de agora fica como está."
            acao: restaura ? "restaurar" : "apagar"
            erro: pagina.erroDaFolha
            ocupado: pagina.ocupado
            onConfirmado: pagina.snapshot(pagina.pedidoDeSnapshot.acao, pagina.pedidoDeSnapshot.rotulo)
            onFechado: pagina.fechar()
        }
    }
}
