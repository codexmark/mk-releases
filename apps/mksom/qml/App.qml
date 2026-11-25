import QtQuick
import QtQuick.Layouts
import "Som.js" as Som

// A janela inteira: no alto, as duas áreas (Equalizador e Microfone); no
// meio, a área escolhida; por cima, as folhas (predefinições, nome, atalhos).
// Tudo o que aparece vem do motor; aqui fica quem fala com ele.
Rectangle {
    id: app
    color: Theme.ground

    // "equalizador" | "microfone"
    property string area: Motor.areaInicial
    // "" | "predefinicoes" | "nome" | "atalhos"
    property string folha: ""
    property bool pronto: false
    // O que o `estado` disse por último (para a frase da saída).
    property var estado: null
    property string recado: ""

    // ── o equalizador ──────────────────────────────────────────────────
    property bool ligado: false
    property bool ligando: false
    property var ganhos: Som.plano()
    property real pre: 0
    property var predefinicoes: []
    // O ritmo das aplicações: quando saiu a última, se há uma no ar e se os
    // controles mudaram depois dela.
    property real ultimaAplicacao: 0
    property bool emVoo: false
    property bool sujo: false

    // ── o microfone ────────────────────────────────────────────────────
    property var entradas: []
    property int entradaId: -1
    property var niveis: []
    property real nivel: Som.PISO
    property real pico: Som.PISO
    property var medidor: null
    property var gravador: null
    property string etapa: ""
    property int falta: 0
    property string recadoDoMic: ""

    readonly property bool estreita: Som.estreita(width)
    readonly property bool baixa: Som.baixa(height)
    readonly property bool livre: folha === ""
    readonly property bool responde: estado !== null && estado.som.responde === true
    readonly property string predefinicao: Som.predefinicaoDe(predefinicoes, ganhos, pre)

    // ── o motor: o estado ──────────────────────────────────────────────

    function ler() {
        Motor.pedir(["estado"], r => {
            if (r.erro) {
                app.recado = r.erro;
                return;
            }
            app.estado = r;
            app.pronto = true;
            app.ligado = r.eq.ligado && r.eq.na_saida;
            if (!app.emVoo && !app.sujo) {
                app.ganhos = r.eq.ganhos;
                app.pre = r.eq.pre;
            }
            app.predefinicoes = r.eq.predefinicoes || [];
            app.entradas = r.entradas || [];
            if (!Som.entradaDe(app.entradas, app.entradaId))
                app.entradaId = Som.entradaInicial(app.entradas);
            // (trocar de entrada já religa o medidor; aqui, só se está parado)
            if (app.medidor === null)
                app.ouvir();
        });
    }

    // ── o motor: o equalizador ─────────────────────────────────────────

    function alternar() {
        if (app.ligando || !app.responde)
            return;
        app.ligando = true;
        app.recado = "";
        Motor.pedir(["eq", app.ligado ? "desligar" : "ligar"], r => {
            app.ligando = false;
            if (r.erro)
                app.recado = r.erro;
            else
                app.recado = r.aviso || "";
            // O que ficou de verdade (ligado, por onde sai) vem do estado.
            app.ler();
        });
    }

    function mexer(banda, valor, soltou) {
        app.ganhos = Som.trocar(app.ganhos, banda, valor);
        app.pedirAplicacao(soltou);
    }

    function mexerPre(valor, soltou) {
        app.pre = Som.pre(valor);
        app.pedirAplicacao(soltou);
    }

    // Os controles mudaram: aplica já (ao soltar, ou se a última aplicação
    // já tem 100 ms), ou marca para daqui a pouco. Nunca duas ao mesmo tempo.
    function pedirAplicacao(ja) {
        app.sujo = true;
        if (app.emVoo)
            return;
        const agora = Date.now();
        if (ja || Som.devoAplicar(agora, app.ultimaAplicacao, false)) {
            app.aplicar();
        } else {
            depois.interval = Math.max(1, Som.esperaPara(agora, app.ultimaAplicacao));
            depois.restart();
        }
    }

    function aplicar() {
        if (app.emVoo || !app.sujo)
            return;
        depois.stop();
        app.sujo = false;
        app.emVoo = true;
        app.ultimaAplicacao = Date.now();
        Motor.pedir(["eq", "ajustar"], r => {
            app.emVoo = false;
            app.recado = r.erro || "";
            if (app.sujo)
                app.pedirAplicacao(false);
        }, JSON.stringify(Som.pedido(app.ganhos, app.pre)));
    }

    Timer {
        id: depois
        repeat: false
        onTriggered: app.aplicar()
    }

    function aplicarPredefinicao(nome) {
        const p = Som.acha(app.predefinicoes, nome);
        if (!p)
            return;
        // Os controles vão já para os valores dela; o motor confirma.
        depois.stop();
        app.sujo = false;
        app.ganhos = p.ganhos.slice();
        app.pre = p.pre;
        app.folha = "";
        equalizador.focar();
        Motor.pedir(["eq", "predefinicao", "aplicar", "--", p.nome], r => {
            app.recado = r.erro || "";
            if (r.erro)
                app.ler();
        });
    }

    function guardarNome(nome) {
        const de = folhaDoNome.de;
        folhaDoNome.ocupado = true;
        const args = de === "" ? ["eq", "predefinicao", "guardar", "--", nome] : ["eq", "predefinicao", "renomear", "--", de, nome];
        Motor.pedir(args, r => {
            folhaDoNome.ocupado = false;
            if (r.erro) {
                folhaDoNome.erro = r.erro;
                return;
            }
            app.predefinicoes = r.predefinicoes || app.predefinicoes;
            app.recado = de === "" ? "guardada: " + nome : "";
            app.fechar();
        });
    }

    function apagarPredefinicao(nome) {
        Motor.pedir(["eq", "predefinicao", "apagar", "--", nome], r => {
            if (r.erro) {
                app.recado = r.erro;
                return;
            }
            app.predefinicoes = r.predefinicoes || app.predefinicoes;
        });
    }

    // ── o motor: o microfone ───────────────────────────────────────────

    // Liga o medidor da entrada escolhida, só com a área Microfone à vista.
    function ouvir() {
        app.pararDeOuvir();
        if (app.area !== "microfone" || app.entradaId < 0 || !app.responde)
            return;
        const meu = Motor.fluxo(["mic", "nivel", String(app.entradaId)], l => {
            if (l.erro) {
                app.recadoDoMic = l.erro;
            } else if (typeof l.pico === "number") {
                app.nivel = l.pico;
                app.pico = Som.picoQueCai(app.pico, l.pico, 0);
                app.niveis = Som.junta(app.niveis, l);
            }
        }, () => {
            if (app.medidor === meu)
                app.medidor = null;
        });
        app.medidor = meu;
    }

    function pararDeOuvir() {
        if (app.medidor) {
            app.medidor.parar();
            app.medidor = null;
        }
        app.niveis = [];
        app.nivel = Som.PISO;
        app.pico = Som.PISO;
    }

    // O pico da barra cai devagar entre um nível e outro.
    Timer {
        interval: 66
        repeat: true
        running: app.medidor !== null
        onTriggered: app.pico = Som.picoQueCai(app.pico, Som.PISO, interval)
    }

    function escolherEntrada(id) {
        app.recadoDoMic = "";
        app.entradaId = id;
    }

    // Um pedido sobre uma entrada: se o motor recusa (ela sumiu), relê tudo.
    function pedirAoMic(args, seDeuCerto) {
        app.recadoDoMic = "";
        Motor.pedir(args, r => {
            if (r.erro) {
                app.recadoDoMic = r.erro;
                if (r.recusado)
                    app.ler();
                return;
            }
            seDeuCerto();
        });
    }

    function tornarPadrao(id) {
        pedirAoMic(["mic", "padrao", String(id)], () => app.entradas = Som.comPadrao(app.entradas, id));
    }

    // O volume anda com o pegador; o motor só é chamado ao soltar.
    function mexerVolume(id, valor, soltou) {
        const v = Som.volume(valor);
        app.entradas = Som.comEntrada(app.entradas, id, { volume: v });
        if (soltou)
            pedirAoMic(["mic", "volume", String(id), String(v)], () => {});
    }

    function trocarMudo(id) {
        const e = Som.entradaDe(app.entradas, id);
        if (!e)
            return;
        const mudo = !e.mudo;
        pedirAoMic(["mic", "mudo", String(id), mudo ? "sim" : "nao"], () => app.entradas = Som.comEntrada(app.entradas, id, { mudo: mudo }));
    }

    // Grava 5 s e toca de volta. A gravação fica na memória e o motor a
    // apaga depois de tocar.
    function gravarEOuvir() {
        if (app.etapa !== "" || app.entradaId < 0)
            return;
        app.recadoDoMic = "";
        app.etapa = "gravando";
        app.falta = 5;
        let gravou = false;
        app.gravador = Motor.fluxo(["mic", "gravar", String(app.entradaId)], l => {
            if (l.erro)
                app.recadoDoMic = l.erro;
            else if (typeof l.falta === "number")
                app.falta = l.falta;
            else if (l.ok)
                gravou = true;
        }, () => {
            app.gravador = null;
            if (!gravou) {
                app.etapa = "";
                return;
            }
            app.etapa = "ouvindo";
            app.gravador = Motor.fluxo(["mic", "ouvir"], l => {
                if (l.erro)
                    app.recadoDoMic = l.erro;
            }, () => {
                app.gravador = null;
                app.etapa = "";
            });
        });
    }

    // ── as folhas e o teclado ──────────────────────────────────────────

    function abrirFolha(qual, de) {
        // A folha aparece antes de receber o foco: um campo escondido não o pega.
        app.folha = qual;
        if (qual === "predefinicoes")
            seletor.abrir();
        if (qual === "nome")
            folhaDoNome.abrir(de || "", de || (app.predefinicao !== "" && !Som.acha(app.predefinicoes, app.predefinicao).fabrica ? app.predefinicao : ""));
    }

    function fechar() {
        if (app.folha === "")
            return;
        app.folha = "";
        if (app.area === "equalizador")
            equalizador.focar();
    }

    onAreaChanged: {
        app.recadoDoMic = "";
        if (app.area === "microfone") {
            app.ouvir();
        } else {
            app.pararDeOuvir();
            equalizador.focar();
        }
    }
    onEntradaIdChanged: ouvir()

    Component.onCompleted: {
        ler();
        if (app.area === "equalizador")
            equalizador.focar();
    }
    // A janela fecha: o medidor e a gravação param, e a gravação é apagada.
    Component.onDestruction: {
        if (app.medidor)
            app.medidor.parar();
        if (app.gravador)
            app.gravador.parar();
        Motor.limpar();
    }

    Shortcut { sequences: ["Space"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.alternar() }
    Shortcut { sequences: ["Ctrl+1"]; enabled: app.livre; onActivated: app.area = "equalizador" }
    Shortcut { sequences: ["Ctrl+2"]; enabled: app.livre; onActivated: app.area = "microfone" }
    Shortcut { sequences: ["Ctrl+S"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.abrirFolha("nome") }
    Shortcut { sequences: ["Ctrl+0"]; enabled: app.livre && app.area === "equalizador"; onActivated: app.aplicarPredefinicao("plano") }
    Shortcut { sequences: ["Up"]; enabled: app.folha === "predefinicoes"; onActivated: seletor.andar(-1) }
    Shortcut { sequences: ["Down"]; enabled: app.folha === "predefinicoes"; onActivated: seletor.andar(1) }
    Shortcut { sequences: ["Return", "Enter"]; enabled: app.folha === "predefinicoes"; onActivated: seletor.confirmar() }
    Shortcut { sequences: ["Escape"]; onActivated: app.fechar() }
    Shortcut { sequences: ["F1"]; onActivated: app.folha = app.folha === "atalhos" ? "" : app.folha === "" ? "atalhos" : app.folha }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: app.estreita ? 12 : 18
        spacing: app.baixa ? 8 : 12

        // ── o alto ─────────────────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            spacing: 10

            Row {
                visible: !app.estreita
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
                    text: "som"
                    color: Theme.copper
                    font.family: Theme.serif
                    font.pixelSize: 16
                }
            }
            Item { Layout.fillWidth: true }
            Segmento {
                opcoes: [{ v: "equalizador", texto: "Equalizador" }, { v: "microfone", texto: "Microfone" }]
                valor: app.area
                onEscolhido: v => app.area = Som.area(v)
            }
        }

        // ── o PipeWire não respondeu ───────────────────────────────────
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: app.pronto && !app.responde
            spacing: 10

            Item { Layout.fillHeight: true }
            Text {
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                text: "O som não respondeu"
                color: Theme.text
                font.family: Theme.serif
                font.pixelSize: 22
                font.weight: Font.Light
            }
            Text {
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
                text: Som.textoDaSaida(app.estado)
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
            }
            Botao {
                texto: "tentar de novo"
                forte: true
                onAcionado: app.ler()
            }
            Item { Layout.fillHeight: true }
        }

        // ── as áreas ───────────────────────────────────────────────────
        Equalizador {
            id: equalizador
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: app.area === "equalizador" && (!app.pronto || app.responde)
            ligado: app.ligado
            ocupado: app.ligando
            ativo: app.livre
            ganhos: app.ganhos
            pre: app.pre
            predefinicao: app.predefinicao
            saida: app.recado !== "" ? app.recado : app.pronto ? Som.textoDaSaida(app.estado) : ""
            estreita: app.estreita
            baixa: app.baixa
            onAlternado: app.alternar()
            onMexido: (banda, valor, soltou) => app.mexer(banda, valor, soltou)
            onPreMexido: (valor, soltou) => app.mexerPre(valor, soltou)
            onPredefinicoesPedidas: app.abrirFolha("predefinicoes")
            onGuardarPedido: app.abrirFolha("nome")
            onPlanoPedido: app.aplicarPredefinicao("plano")
            onPreCorrigido: app.mexerPre(Som.preSugerido(app.ganhos), true)
        }

        Microfone {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: app.area === "microfone" && (!app.pronto || app.responde)
            entradas: app.entradas
            entradaId: app.entradaId
            niveis: app.niveis
            nivel: app.nivel
            pico: app.pico
            ouvindo: app.medidor !== null
            etapa: app.etapa
            falta: app.falta
            recado: app.recadoDoMic
            estreita: app.estreita
            ativo: app.livre
            onEscolhida: id => app.escolherEntrada(id)
            onPadraoPedido: id => app.tornarPadrao(id)
            onVolumeMexido: (id, valor, soltou) => app.mexerVolume(id, valor, soltou)
            onMudoTrocado: id => app.trocarMudo(id)
            onGravarPedido: app.gravarEOuvir()
            onAtualizarPedido: app.ler()
        }
    }

    // ── as folhas ──────────────────────────────────────────────────────
    Predefinicoes {
        id: seletor
        anchors.fill: parent
        visible: app.folha === "predefinicoes"
        lista: app.predefinicoes
        atual: app.predefinicao
        onEscolhida: nome => app.aplicarPredefinicao(nome)
        onRenomearPedido: nome => app.abrirFolha("nome", nome)
        onApagarPedido: nome => app.apagarPredefinicao(nome)
        onFechado: app.fechar()
    }

    Nome {
        id: folhaDoNome
        anchors.fill: parent
        visible: app.folha === "nome"
        lista: app.predefinicoes
        onConfirmado: nome => app.guardarNome(nome)
        onFechado: app.fechar()
    }

    Atalhos {
        anchors.fill: parent
        visible: app.folha === "atalhos"
        onFechado: app.fechar()
    }
}
