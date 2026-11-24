import QtQuick
import QtQuick.Layouts
import "Calc.js" as Calc

// A janela inteira: o visor (a expressão em cima, o resultado embaixo), e
// embaixo o teclado ou o histórico. Tudo o que se clica também se faz pelo
// teclado físico (a tabela está em ../CONTRATO.md). O texto da expressão é do
// Calc.js; os números são do motor.
Rectangle {
    id: app
    color: Theme.ground
    focus: true

    property var estado: Calc.novo()
    // A resposta do motor para a expressão de agora (a prévia), e o pedido dela.
    property var previa: null
    property int pedidoDaPrevia: 0
    // A frase do "=" que o motor recusou, e um recado de uma linha (copiado…).
    property string erro: ""
    property string recado: ""
    property var itens: []
    property bool historicoAberto: false
    // A tecla que acabou de ser acionada: a da tela pisca junto.
    property string piscando: ""

    readonly property var linha: Calc.visor(estado, previa, erro)
    readonly property var teclas: [
        { t: "C", a: "limpar", tipo: "operador" }, { t: "(", a: "(", tipo: "operador" },
        { t: ")", a: ")", tipo: "operador" }, { t: "⌫", a: "apagar", tipo: "operador" },
        { t: "√", a: "√", tipo: "operador" }, { t: "^", a: "^", tipo: "operador" },
        { t: "%", a: "%", tipo: "operador" }, { t: "÷", a: "÷", tipo: "operador" },
        { t: "7", a: "7", tipo: "numero" }, { t: "8", a: "8", tipo: "numero" },
        { t: "9", a: "9", tipo: "numero" }, { t: "×", a: "×", tipo: "operador" },
        { t: "4", a: "4", tipo: "numero" }, { t: "5", a: "5", tipo: "numero" },
        { t: "6", a: "6", tipo: "numero" }, { t: "−", a: "-", tipo: "operador" },
        { t: "1", a: "1", tipo: "numero" }, { t: "2", a: "2", tipo: "numero" },
        { t: "3", a: "3", tipo: "numero" }, { t: "+", a: "+", tipo: "operador" },
        { t: "±", a: "sinal", tipo: "numero" }, { t: "0", a: "0", tipo: "numero" },
        { t: ",", a: ",", tipo: "numero" }, { t: "=", a: "igual", tipo: "igual" }
    ]

    onEstadoChanged: prever()

    Component.onCompleted: {
        forceActiveFocus();
        Motor.pedir({ op: "historico" }, r => app.itens = r.itens || []);
        const inicial = Motor.inicial !== "" ? Calc.colar(Calc.novo(), Motor.inicial) : null;
        if (inicial !== null)
            estado = inicial;
    }

    // Pede a prévia da expressão de agora. Só vale a resposta do último pedido.
    function prever() {
        if (estado.fechado || !Calc.temConta(estado.expr)) {
            pedidoDaPrevia = 0;
            previa = null;
            return;
        }
        pedidoDaPrevia = Motor.pedir({ op: "calcular", expr: Calc.paraMotor(estado.expr) }, r => {
            if (r.id === app.pedidoDaPrevia)
                app.previa = r;
        });
    }

    // O "=": calcula, guarda no histórico e deixa a expressão à vista.
    function confirmar() {
        if (estado.fechado || estado.expr === "")
            return;
        const pedida = estado.expr;
        Motor.pedir({ op: "calcular", expr: Calc.paraMotor(pedida), guardar: true }, r => {
            if (r.itens)
                app.itens = r.itens;
            // Se a pessoa já digitou outra coisa, esta resposta ficou velha.
            if (app.estado.fechado || app.estado.expr !== pedida)
                return;
            if (r.erro) {
                app.erro = r.erro;
                return;
            }
            app.estado = Calc.confirmado(app.estado, r);
            app.recado = r.aviso || "";
        });
    }

    function copiar() {
        const texto = Calc.paraCopiar(estado, previa);
        if (texto === "") {
            recado = "nada para copiar";
            return;
        }
        prancheta.text = texto;
        prancheta.selectAll();
        prancheta.copy();
        recado = "copiado: " + texto;
    }

    function colar() {
        prancheta.text = "";
        prancheta.paste();
        const novo = Calc.colar(estado, prancheta.text);
        prancheta.text = "";
        if (novo === null) {
            recado = "o que está copiado não é uma conta";
            return;
        }
        erro = "";
        estado = novo;
    }

    // Uma ação, venha de onde vier (tecla da tela ou teclado físico).
    function agir(a) {
        piscando = a;
        pisca.restart();
        recado = "";
        switch (a) {
        case "igual":
            confirmar();
            break;
        case "esc":
            if (historicoAberto) {
                historicoAberto = false;
            } else if (Calc.vazio(estado) && erro === "") {
                Qt.quit();
            } else {
                erro = "";
                estado = Calc.novo();
            }
            break;
        case "copiar":
            copiar();
            break;
        case "colar":
            colar();
            break;
        case "historico":
            historicoAberto = !historicoAberto;
            break;
        default:
            erro = "";
            estado = Calc.tecla(estado, a);
        }
    }

    Keys.onPressed: event => {
        const lista = Calc.acoes(event.key, event.text, event.modifiers);
        if (lista.length === 0)
            return;
        event.accepted = true;
        for (const a of lista)
            app.agir(a);
    }

    Timer {
        id: pisca
        interval: 110
        onTriggered: app.piscando = ""
    }

    // A área de transferência, pelo próprio Qt: um campo de texto que nunca
    // aparece, só para copiar e colar.
    TextEdit {
        id: prancheta
        visible: false
        textFormat: TextEdit.PlainText
        activeFocusOnPress: false
    }

    // Um clique em qualquer lugar devolve o teclado à calculadora.
    MouseArea {
        anchors.fill: parent
        onPressed: mouse => {
            app.forceActiveFocus();
            mouse.accepted = false;
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 14
        spacing: 10

        RowLayout {
            Layout.fillWidth: true
            spacing: 8

            Rotulo {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: app.recado !== "" ? app.recado : "mk  /  MkCalc"
                color: app.recado !== "" ? Theme.textDim : Theme.textFaint
            }
            Botao {
                texto: app.historicoAberto ? "teclado" : "histórico"
                forte: app.historicoAberto
                onAcionado: app.agir("historico")
            }
        }

        // ── o visor ────────────────────────────────────────────────────
        Rectangle {
            Layout.fillWidth: true
            implicitHeight: 112
            color: Theme.surface
            border.width: 1
            border.color: app.linha.tipo === "erro" ? Theme.copper : Theme.hairline

            // A expressão: uma linha só, e o que não cabe some pela esquerda
            // (o fim, onde se digita, fica sempre à vista).
            Item {
                id: faixa
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.margins: 12
                height: 22
                clip: true

                Text {
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    text: Calc.mostrar(app.estado.expr)
                    color: app.estado.fechado ? Theme.textDim : Theme.textSoft
                    font.family: Theme.mono
                    font.pixelSize: 15
                }
            }

            Text {
                id: sinal
                anchors.left: parent.left
                anchors.leftMargin: 12
                anchors.baseline: resultado.baseline
                visible: app.linha.tipo === "resultado" || app.linha.tipo === "previa"
                text: app.linha.sinal
                color: app.linha.tipo === "resultado" ? Theme.copper : Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 15
            }

            // O resultado: grande depois do "=", apagado enquanto é prévia.
            // Encolhe para caber; o que ainda assim não cabe some pela esquerda.
            Text {
                id: resultado
                readonly property bool frase: app.linha.tipo === "erro" || app.linha.tipo === "aviso"
                anchors.left: sinal.visible ? sinal.right : parent.left
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                anchors.leftMargin: 10
                anchors.rightMargin: 12
                anchors.bottomMargin: 10
                height: 50
                text: app.linha.tipo === "vazio" ? (app.estado.expr === "" ? "0" : "") : app.linha.texto
                color: app.linha.tipo === "resultado" ? Theme.text
                    : app.linha.tipo === "erro" ? Theme.copper
                    : app.linha.tipo === "previa" ? Theme.textDim : Theme.textFaint
                font.family: frase ? Theme.sans : Theme.mono
                font.pixelSize: frase ? 14 : 32
                fontSizeMode: frase ? Text.FixedSize : Text.HorizontalFit
                minimumPixelSize: 13
                wrapMode: frase ? Text.WordWrap : Text.NoWrap
                maximumLineCount: frase ? 2 : 1
                elide: frase ? Text.ElideRight : Text.ElideLeft
                horizontalAlignment: Text.AlignRight
                verticalAlignment: Text.AlignBottom
            }
        }

        // ── o teclado ou o histórico ───────────────────────────────────
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            GridLayout {
                anchors.fill: parent
                visible: !app.historicoAberto
                columns: 4
                rowSpacing: 6
                columnSpacing: 6

                Repeater {
                    model: app.teclas

                    Tecla {
                        required property var modelData
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        Layout.preferredWidth: 1
                        Layout.preferredHeight: 1
                        texto: modelData.t
                        tipo: modelData.tipo
                        ativa: app.piscando === modelData.a
                        onAcionado: app.agir(modelData.a)
                    }
                }
            }

            Historico {
                anchors.fill: parent
                visible: app.historicoAberto
                itens: app.itens
                onEscolhido: item => {
                    app.erro = "";
                    app.recado = "";
                    app.estado = Calc.doHistorico(app.estado, item);
                    app.historicoAberto = false;
                }
                onApagado: item => Motor.pedir({ op: "apagar", item: item.id }, r => {
                    if (r.itens)
                        app.itens = r.itens;
                    else
                        Motor.pedir({ op: "historico" }, h => app.itens = h.itens || []);
                })
                onLimpo: Motor.pedir({ op: "limpar" }, r => {
                    if (r.itens)
                        app.itens = r.itens;
                    else
                        app.recado = r.erro || "";
                })
            }
        }
    }
}
