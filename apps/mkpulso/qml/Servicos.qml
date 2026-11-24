import QtQuick
import QtQuick.Layouts
import "Pulso.js" as Pulso

// Serviços: as unidades do systemd, as suas e as do sistema. Escolher uma
// mostra o registro recente e libera as ações; as do sistema pedem a senha
// pelo polkit (quem pergunta é o Maré).
Item {
    id: pagina

    property string escopo: "usuario"
    property string busca: ""
    property string filtro: "todos"
    property var servicos: []
    property string erro: ""
    property string escolhida: ""
    property var registro: []
    property string recado: ""
    property bool recadoErro: false

    readonly property var visiveis: servicos.filter(s => {
        if (filtro === "ativos" && s.estado !== "ativo")
            return false;
        if (filtro === "falhas" && s.estado !== "falhou")
            return false;
        const q = Pulso.fold(busca);
        return q === "" || Pulso.fold(s.unidade + " " + s.descricao).indexOf(q) >= 0;
    })
    readonly property var alvo: servicos.find(s => s.unidade === escolhida) || null

    function args(extra) {
        return escopo === "usuario" ? extra.concat(["--usuario"]) : extra;
    }
    function ler() {
        Coletor.pedir(["servicos", escopo === "usuario" ? "--usuario" : "--sistema"], r => {
            pagina.erro = r.erro || "";
            if (!r.erro)
                pagina.servicos = r.servicos || [];
        });
    }
    function lerRegistro() {
        if (escolhida === "") {
            registro = [];
            return;
        }
        Coletor.pedir(args(["registro", escolhida, "--linhas", "60"]), r => pagina.registro = r.erro ? [{ t_s: null, nivel: "erro", texto: r.erro }] : (r.linhas || []));
    }
    function agir(acao, feito) {
        if (!alvo)
            return;
        Coletor.pedir(args(["servico", acao, alvo.unidade]), r => {
            pagina.recadoErro = !!r.erro;
            pagina.recado = r.erro ? r.erro : alvo.unidade + ": " + feito;
            some.restart();
            pagina.ler();
            pagina.lerRegistro();
        });
    }
    onEscopoChanged: { escolhida = ""; registro = []; ler(); }
    onEscolhidaChanged: lerRegistro()
    Component.onCompleted: ler()

    Timer { interval: 5000; repeat: true; running: true; onTriggered: pagina.ler() }
    Timer { id: some; interval: 6000; onTriggered: pagina.recado = "" }

    ColumnLayout {
        anchors.fill: parent
        spacing: 14

        RowLayout {
            Layout.fillWidth: true
            spacing: 14
            Cabecalho {
                numero: "05"
                titulo: "Serviços"
                frase: "As unidades do systemd: o que está de pé, o que falhou e o que sobe sozinho."
            }
            Item { Layout.fillWidth: true }
            Segmento {
                Layout.alignment: Qt.AlignBottom
                opcoes: [{ v: "usuario", texto: "seus" }, { v: "sistema", texto: "do sistema" }]
                valor: pagina.escopo
                onEscolhido: v => pagina.escopo = v
            }
            Segmento {
                Layout.alignment: Qt.AlignBottom
                opcoes: [{ v: "todos", texto: "todos" }, { v: "ativos", texto: "ativos" }, { v: "falhas", texto: "falhas" }]
                valor: pagina.filtro
                onEscolhido: v => pagina.filtro = v
            }
            Campo { Layout.alignment: Qt.AlignBottom; Layout.preferredWidth: 200; dica: "procurar"; onTextChanged: pagina.busca = text }
        }

        Text {
            visible: pagina.erro !== ""
            text: pagina.erro
            color: Theme.copper
            font.family: Theme.mono
            font.pixelSize: 11
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            ListView {
                id: lista
                anchors.fill: parent
                anchors.margins: 1
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.visiveis

                delegate: Item {
                    id: linha
                    required property var modelData
                    readonly property bool atual: pagina.escolhida === modelData.unidade
                    width: lista.width
                    height: 38

                    Rectangle {
                        anchors.fill: parent
                        color: linha.atual ? Theme.copperA(0.1) : passa.containsMouse ? Theme.boneA(0.03) : Qt.rgba(0, 0, 0, 0)
                    }
                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Rectangle { visible: linha.atual; width: 2; height: parent.height; color: Theme.copper }
                    MouseArea {
                        id: passa
                        anchors.fill: parent
                        hoverEnabled: true
                        onClicked: pagina.escolhida = linha.atual ? "" : linha.modelData.unidade
                    }

                    // O estado: cheio de cobre se falhou, de osso se está de pé, vazado se parado.
                    Rectangle {
                        x: 14
                        anchors.verticalCenter: parent.verticalCenter
                        width: 8
                        height: 8
                        radius: 4
                        color: linha.modelData.estado === "falhou" ? Theme.copper : linha.modelData.estado === "ativo" ? Theme.text : Qt.rgba(0, 0, 0, 0)
                        border.width: 1
                        border.color: linha.modelData.estado === "falhou" ? Theme.copper : linha.modelData.estado === "ativo" ? Theme.text : Theme.textFaint
                    }
                    Text {
                        x: 34
                        width: 280
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.unidade
                        color: linha.modelData.estado === "inativo" ? Theme.textDim : Theme.text
                        font.family: Theme.mono
                        font.pixelSize: 11
                        elide: Text.ElideMiddle
                    }
                    Text {
                        x: 324
                        width: parent.width - 324 - 330
                        anchors.verticalCenter: parent.verticalCenter
                        text: linha.modelData.descricao
                        color: Theme.textDim
                        font.family: Theme.sans
                        font.pixelSize: 12
                        elide: Text.ElideRight
                    }
                    Row {
                        anchors.right: parent.right
                        anchors.rightMargin: 14
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 0
                        Text {
                            width: 86
                            horizontalAlignment: Text.AlignRight
                            text: linha.modelData.estado + (linha.modelData.sub && linha.modelData.estado === "ativo" && linha.modelData.sub !== "running" ? " (" + linha.modelData.sub + ")" : "")
                            color: linha.modelData.estado === "falhou" ? Theme.copper : Theme.textSoft
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideRight
                        }
                        Text {
                            width: 84
                            horizontalAlignment: Text.AlignRight
                            text: Pulso.tamanho(linha.modelData.mem_kb)
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                        Text {
                            width: 130
                            horizontalAlignment: Text.AlignRight
                            text: linha.modelData.habilitado === "sim" ? "sobe sozinho" : linha.modelData.habilitado === "não" ? "não sobe sozinho" : linha.modelData.habilitado
                            color: linha.modelData.habilitado === "sim" ? Theme.textSoft : Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                        }
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: lista.count === 0 && pagina.erro === ""
                    text: pagina.servicos.length === 0 ? "lendo os serviços…" : "nenhum serviço assim"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }

        // O registro da unidade escolhida.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: pagina.alvo !== null ? 170 : 0
            visible: pagina.alvo !== null
            color: Theme.surface
            border.width: 1
            border.color: Theme.hairline

            ListView {
                anchors.fill: parent
                anchors.margins: 10
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.registro
                onCountChanged: positionViewAtEnd()
                delegate: Text {
                    required property var modelData
                    width: ListView.view.width
                    text: (modelData.t_s ? Qt.formatDateTime(new Date(modelData.t_s * 1000), "dd/MM HH:mm:ss") + "  " : "") + modelData.texto
                    color: modelData.nivel === "erro" ? Theme.copper : modelData.nivel === "aviso" ? Theme.text : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                    wrapMode: Text.WrapAnywhere
                }
                Text {
                    anchors.centerIn: parent
                    visible: parent.count === 0
                    text: "sem linhas no registro"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 13
                    font.italic: true
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 10
            Text {
                Layout.fillWidth: true
                elide: Text.ElideRight
                text: pagina.recado !== "" ? pagina.recado
                    : pagina.alvo === null ? "escolha um serviço para ver o registro e agir sobre ele"
                    : pagina.alvo.unidade + (pagina.alvo.pid ? "  ·  PID " + pagina.alvo.pid : "")
                        + (pagina.alvo.desde_s ? "  ·  há " + Pulso.desde(pagina.alvo.desde_s, Date.now() / 1000) : "")
                color: pagina.recado !== "" && pagina.recadoErro ? Theme.copper : Theme.textDim
                font.family: Theme.mono
                font.pixelSize: 11
            }
            Botao {
                texto: pagina.alvo !== null && pagina.alvo.habilitado === "sim" ? "não subir sozinho" : "subir sozinho"
                ligado: pagina.alvo !== null && (pagina.alvo.habilitado === "sim" || pagina.alvo.habilitado === "não")
                onAcionado: pagina.agir(pagina.alvo.habilitado === "sim" ? "desabilitar" : "habilitar", pagina.alvo.habilitado === "sim" ? "não sobe mais sozinho" : "passa a subir sozinho")
            }
            Botao { texto: "reiniciar"; ligado: pagina.alvo !== null; onAcionado: pagina.agir("reiniciar", "reiniciado") }
            Botao {
                texto: pagina.alvo !== null && pagina.alvo.estado === "ativo" ? "parar" : "iniciar"
                forte: true
                ligado: pagina.alvo !== null
                onAcionado: pagina.agir(pagina.alvo.estado === "ativo" ? "parar" : "iniciar", pagina.alvo.estado === "ativo" ? "parado" : "iniciado")
            }
        }
    }
}
