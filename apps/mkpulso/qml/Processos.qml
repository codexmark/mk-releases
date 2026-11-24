import QtQuick
import QtQuick.Layouts
import "Pulso.js" as Pulso

// Processos: os apps e os serviços agrupados por identidade, com o que cada
// grupo gasta agora ou na média do último minuto e dos últimos três. Clicar
// num grupo o escolhe (e abre os processos dele); as ações valem para a linha
// escolhida.
Item {
    id: pagina

    property string busca: ""
    property string janela: "agora"
    property string coluna: "cpu_pct"
    property bool decrescente: true
    property var abertos: ({})
    property var recolhidas: ({})
    // A linha escolhida: um grupo ("g:<chave>", veja Pulso.chave) ou um processo ("p:<pid>").
    property string escolha: ""
    property string recado: ""
    property bool recadoErro: false

    readonly property var grupos: Coletor.retrato ? Coletor.retrato.grupos : []
    readonly property var visiveis: Pulso.ordenar(Pulso.filtrar(grupos, busca), coluna, decrescente, janela)
    readonly property var linhas: Pulso.linhas(Pulso.secoes(visiveis), abertos, recolhidas)
    readonly property var colunas: [
        { id: "cpu_pct", nome: "CPU", largura: 78 }, { id: "mem_kb", nome: "Memória", largura: 92 },
        { id: "disco_bps", nome: "Disco", largura: 92 }, { id: "gpu_pct", nome: "GPU", largura: 66 },
        { id: "energia", nome: "Energia", largura: 100 }, { id: "pid", nome: "PID", largura: 76 }
    ]
    readonly property var tetos: ({ cpu_pct: Math.max(100, Pulso.teto(grupos, "cpu_pct", janela)),
        mem_kb: Coletor.retrato && Coletor.retrato.mem ? Coletor.retrato.mem.total_kb / 4 : 1,
        disco_bps: Math.max(1048576, Pulso.teto(grupos, "disco_bps", janela)), gpu_pct: 100 })

    // O alvo das ações: a linha escolhida, se ainda existe neste retrato.
    readonly property var alvo: {
        for (const l of linhas) {
            if (l.tipo === "grupo" && escolha === "g:" + Pulso.chave(l.grupo))
                return { grupo: l.grupo, processo: null };
            if (l.tipo === "processo" && escolha === "p:" + l.processo.pid)
                return { grupo: l.grupo, processo: l.processo };
        }
        return null;
    }

    function texto(g, coluna) {
        const v = Pulso.valor(g, coluna, janela);
        if (coluna === "cpu_pct" || coluna === "gpu_pct")
            return Pulso.pct(v);
        if (coluna === "mem_kb")
            return Pulso.tamanho(v);
        if (coluna === "disco_bps")
            return Pulso.taxa(v);
        if (coluna === "energia")
            return g.energia || "—";
        return v === null || v === undefined ? "—" : String(v);
    }

    function ordenarPor(id) {
        if (coluna === id)
            decrescente = !decrescente;
        else {
            coluna = id;
            decrescente = id !== "nome";
        }
    }

    function alternar(mapa, chave) {
        const novo = Object.assign({}, mapa);
        if (novo[chave])
            delete novo[chave];
        else
            novo[chave] = true;
        return novo;
    }

    function respondeu(r, feito) {
        recadoErro = !!r.erro;
        recado = r.erro ? r.erro : feito;
        some.restart();
    }

    function sinal(nome, feito) {
        if (!alvo)
            return;
        if (alvo.processo)
            Coletor.pedir(["sinal", String(alvo.processo.pid), nome], r => respondeu(r, feito));
        else
            // O grupo vai pela unidade dele (a do usuário com --usuario); os
            // pids do retrato cobrem o que estiver fora dela.
            Coletor.pedir(["grupo", alvo.grupo.id, nome, "--pids", alvo.grupo.processos.map(p => p.pid).join(",")]
                .concat(alvo.grupo.escopo === "usuario" ? ["--usuario"] : []), r => respondeu(r, feito));
    }

    function prioridade(delta) {
        const p = alvo ? (alvo.processo || alvo.grupo.processos.find(x => x.pid === alvo.grupo.pid) || alvo.grupo.processos[0]) : null;
        if (!p)
            return;
        const nice = Math.max(-20, Math.min(19, (p.nice || 0) + delta));
        Coletor.pedir(["prioridade", String(p.pid), String(nice)], r => respondeu(r, "prioridade " + nice));
    }

    Timer { id: some; interval: 5000; onTriggered: pagina.recado = "" }

    ColumnLayout {
        anchors.fill: parent
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            spacing: 16

            Cabecalho {
                numero: "01"
                titulo: "Processos"
                frase: "Apps e serviços agrupados por identidade; cada leitura é de um processo vivo."
            }
            Item { Layout.fillWidth: true }
            Segmento {
                Layout.alignment: Qt.AlignBottom
                opcoes: [{ v: "agora", texto: "agora" }, { v: "1m", texto: "1 min" }, { v: "3m", texto: "3 min" }]
                valor: pagina.janela
                onEscolhido: v => pagina.janela = v
            }
            Campo {
                Layout.alignment: Qt.AlignBottom
                Layout.preferredWidth: 250
                dica: "nome, comando, PID ou janela"
                onTextChanged: pagina.busca = text
            }
        }

        // A tabela.
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            // Títulos das colunas; clicar ordena.
            Item {
                id: titulos
                x: 1
                y: 1
                width: parent.width - 2
                height: 34

                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.hairline }

                Rotulo {
                    x: 14
                    anchors.verticalCenter: parent.verticalCenter
                    text: "nome" + (pagina.coluna === "nome" ? (pagina.decrescente ? "  ↓" : "  ↑") : "")
                    color: pagina.coluna === "nome" ? Theme.copper : Theme.textFaint
                    MouseArea { anchors.fill: parent; anchors.margins: -8; cursorShape: Qt.PointingHandCursor; onClicked: pagina.ordenarPor("nome") }
                }
                Rotulo {
                    anchors.right: celulas.left
                    anchors.rightMargin: 18
                    anchors.verticalCenter: parent.verticalCenter
                    text: (Coletor.retrato ? Coletor.retrato.totais.grupos + " grupos  /  " + Coletor.retrato.totais.processos + " processos" : "")
                }
                Row {
                    id: celulas
                    anchors.right: parent.right
                    height: parent.height
                    Repeater {
                        model: pagina.colunas
                        Item {
                            required property var modelData
                            width: modelData.largura
                            height: titulos.height
                            Rotulo {
                                anchors.right: parent.right
                                anchors.rightMargin: 12
                                anchors.verticalCenter: parent.verticalCenter
                                text: parent.modelData.nome + (pagina.coluna === parent.modelData.id ? (pagina.decrescente ? " ↓" : " ↑") : "")
                                color: pagina.coluna === parent.modelData.id ? Theme.copper : Theme.textFaint
                            }
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: pagina.ordenarPor(parent.modelData.id) }
                        }
                    }
                }
            }

            ListView {
                id: lista
                x: 1
                anchors.top: titulos.bottom
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 1
                width: parent.width - 2
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.linhas

                delegate: Item {
                    id: linha
                    required property var modelData
                    readonly property bool secao: modelData.tipo === "secao"
                    readonly property bool filho: modelData.tipo === "processo"
                    readonly property var g: modelData.grupo || null
                    readonly property var p: modelData.processo || null
                    readonly property string chave: secao ? "" : filho ? "p:" + p.pid : "g:" + Pulso.chave(g)
                    readonly property bool escolhida: !secao && pagina.escolha === chave
                    width: lista.width
                    height: secao ? 30 : filho ? 30 : 40

                    Rectangle {
                        anchors.fill: parent
                        color: linha.secao ? Theme.boneA(0.035) : linha.escolhida ? Theme.copperA(0.1) : passa.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)
                    }
                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Rectangle { visible: linha.escolhida; width: 2; height: parent.height; color: Theme.copper }

                    MouseArea {
                        id: passa
                        anchors.fill: parent
                        hoverEnabled: true
                        onClicked: {
                            if (linha.secao)
                                pagina.recolhidas = pagina.alternar(pagina.recolhidas, linha.modelData.secao.tipo);
                            else if (linha.filho)
                                pagina.escolha = linha.chave;
                            else {
                                if (pagina.escolha === linha.chave || !pagina.abertos[Pulso.chave(linha.g)])
                                    pagina.abertos = pagina.alternar(pagina.abertos, Pulso.chave(linha.g));
                                pagina.escolha = linha.chave;
                            }
                        }
                    }

                    // Título de seção.
                    Row {
                        visible: linha.secao
                        x: 14
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 10
                        Text {
                            text: linha.secao && pagina.recolhidas[linha.modelData.secao.tipo] ? "›" : "⌄"
                            color: Theme.copper
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                        Rotulo { text: linha.secao ? linha.modelData.secao.titulo : ""; color: Theme.textSoft; anchors.verticalCenter: parent.verticalCenter }
                        Rotulo { text: linha.secao ? String(linha.modelData.secao.grupos.length) : ""; anchors.verticalCenter: parent.verticalCenter }
                    }

                    // Grupo ou processo.
                    Item {
                        visible: !linha.secao
                        anchors.fill: parent

                        Text {
                            x: linha.filho ? 44 : 14
                            anchors.verticalCenter: parent.verticalCenter
                            visible: !linha.filho && linha.g !== null && linha.g.processos.length > 1
                            text: linha.g !== null && pagina.abertos[Pulso.chave(linha.g)] ? "⌄" : "›"
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 11
                        }
                        Column {
                            x: linha.filho ? 58 : 36
                            anchors.verticalCenter: parent.verticalCenter
                            width: celulasLinha.x - x - 12
                            spacing: 1
                            Row {
                                spacing: 8
                                Text {
                                    text: linha.secao ? "" : linha.filho ? linha.p.nome : linha.g.nome
                                    color: linha.filho ? Theme.textSoft : Theme.text
                                    font.family: Theme.sans
                                    font.pixelSize: linha.filho ? 12 : 13
                                    elide: Text.ElideRight
                                }
                                Rotulo {
                                    visible: !linha.filho && linha.g !== null && linha.g.n > 1
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: linha.g !== null ? String(linha.g.n) : ""
                                }
                                Rotulo {
                                    visible: !linha.filho && linha.g !== null && linha.g.janelas.length > 0
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: Theme.copper
                                    text: linha.g !== null && linha.g.janelas.length > 0
                                        ? (linha.g.janelas.length === 1 ? "janela" : linha.g.janelas.length + " janelas") : ""
                                }
                                Rotulo {
                                    visible: linha.filho && linha.p !== null && linha.p.estado !== "dormindo" && linha.p.estado !== "ocioso"
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: Theme.copper
                                    text: linha.p !== null ? linha.p.estado : ""
                                }
                            }
                            Text {
                                visible: !linha.secao
                                width: parent.width
                                text: linha.secao ? "" : linha.filho ? linha.p.cmd
                                    : (linha.g.janelas.length > 0 ? linha.g.janelas[0].titulo : linha.g.unidade !== "" ? linha.g.unidade : linha.g.id)
                                color: Theme.textFaint
                                font.family: Theme.mono
                                font.pixelSize: 10
                                elide: Text.ElideRight
                            }
                        }

                        Row {
                            id: celulasLinha
                            anchors.right: parent.right
                            height: parent.height
                            Repeater {
                                model: pagina.colunas
                                Item {
                                    id: cel
                                    required property var modelData
                                    readonly property var fonte: linha.secao ? null : linha.filho ? linha.p : linha.g
                                    readonly property var bruto: fonte === null ? null
                                        : linha.filho ? fonte[modelData.id] : Pulso.valor(fonte, modelData.id, pagina.janela)
                                    width: modelData.largura
                                    height: linha.height

                                    Rectangle {
                                        anchors.fill: parent
                                        anchors.bottomMargin: 1
                                        visible: cel.modelData.id !== "pid"
                                        color: cel.modelData.id === "energia"
                                            ? Theme.copperA(linha.filho || cel.fonte === null ? 0 : Pulso.nivelEnergia(cel.fonte.energia) * 0.045)
                                            : Theme.copperA(Pulso.calor(cel.bruto, pagina.tetos[cel.modelData.id] || 0) * 0.3)
                                    }
                                    Text {
                                        anchors.right: parent.right
                                        anchors.rightMargin: 12
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: cel.fonte === null ? ""
                                            : linha.filho ? (cel.modelData.id === "energia" ? "" : cel.modelData.id === "pid" ? String(cel.fonte.pid)
                                                : cel.modelData.id === "mem_kb" ? Pulso.tamanho(cel.bruto) : cel.modelData.id === "disco_bps" ? Pulso.taxa(cel.bruto) : Pulso.pct(cel.bruto))
                                            : pagina.texto(cel.fonte, cel.modelData.id)
                                        color: linha.filho ? Theme.textDim : Theme.textSoft
                                        font.family: Theme.mono
                                        font.pixelSize: 11
                                    }
                                }
                            }
                        }
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: lista.count === 0
                    text: "nada casa com \"" + pagina.busca + "\""
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }

        // As ações da linha escolhida.
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: pagina.recado !== "" ? pagina.recado
                    : pagina.alvo === null ? "escolha uma linha para agir sobre ela"
                    : pagina.alvo.processo ? pagina.alvo.processo.nome + "  ·  PID " + pagina.alvo.processo.pid + "  ·  " + pagina.alvo.processo.usuario
                        + "  ·  há " + Pulso.desde(pagina.alvo.processo.inicio_s, Coletor.retrato.t_ms / 1000)
                    : pagina.alvo.grupo.nome + "  ·  " + pagina.alvo.grupo.n + (pagina.alvo.grupo.n === 1 ? " processo" : " processos")
                        + (pagina.alvo.grupo.unidade !== "" ? "  ·  " + pagina.alvo.grupo.unidade : "")
                color: pagina.recado !== "" && pagina.recadoErro ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
            Botao { texto: "prioridade −"; ligado: pagina.alvo !== null; onAcionado: pagina.prioridade(1) }
            Botao { texto: "prioridade +"; ligado: pagina.alvo !== null; onAcionado: pagina.prioridade(-1) }
            Botao { texto: "pausar"; ligado: pagina.alvo !== null; onAcionado: pagina.sinal("pausar", "pausado") }
            Botao { texto: "continuar"; ligado: pagina.alvo !== null; onAcionado: pagina.sinal("continuar", "continuando") }
            Botao { texto: "encerrar"; forte: true; ligado: pagina.alvo !== null; onAcionado: pagina.sinal("encerrar", "pedido de encerrar enviado") }
            Botao { texto: "forçar"; ligado: pagina.alvo !== null; onAcionado: pagina.sinal("matar", "encerrado à força") }
        }
    }
}
