import QtQuick
import QtQuick.Layouts
import "Util.js" as Util

// MkInstalar: a pessoa solta (ou escolhe) um AppImage, um pacote do Arch ou
// um arquivo compactado; a janela mostra o que cada um é e o que instalar
// faria, e só então instala. A outra aba lista o que o MkInstalar já instalou,
// para tirar. Quem olha os arquivos e mexe no disco é o motor.
Item {
    id: pagina

    // Cada item: { arquivo, estado, ficha, erro, fase, pct, fim, atalho, executavel }.
    // estado: "lendo", "pronto", "erro", "instalando", "feito".
    property var itens: []
    property int atual: -1
    property string aba: "instalar"
    property var instalados: []
    property string erroDosInstalados: ""
    property string confirmando: ""
    readonly property var item: atual >= 0 && atual < itens.length ? itens[atual] : null
    readonly property bool ocupada: itens.some(i => i.estado === "instalando")

    function muda(arquivo, campos) {
        const i = Util.indice(itens, arquivo);
        if (i >= 0)
            itens = Util.muda(itens, i, campos);
    }

    // Põe os arquivos na lista e pede a ficha de cada um ao motor.
    function entram(caminhos) {
        const novos = Util.novos(itens, caminhos);
        if (novos.length === 0)
            return;
        itens = itens.concat(novos.map(c => ({ arquivo: c, estado: "lendo", ficha: null, erro: "", fase: null, pct: 0, fim: null,
                                               atalho: true, executavel: "" })));
        if (atual < 0)
            atual = 0;
        aba = "instalar";
        for (const c of novos)
            inspeciona(c);
    }

    function inspeciona(arquivo) {
        Motor.pedir(["inspecionar", arquivo], r => {
            if (r.erro) {
                pagina.muda(arquivo, { estado: "erro", erro: r.erro });
                return;
            }
            const sugerido = r.compactado && r.compactado.executavel_sugerido ? r.compactado.executavel_sugerido : "";
            pagina.muda(arquivo, { estado: "pronto", ficha: r, erro: "", executavel: sugerido });
        });
    }

    function instala(it) {
        const arquivo = it.arquivo;
        muda(arquivo, { estado: "instalando", fase: null, pct: 0, erro: "" });
        Motor.tarefa(Util.argsInstalar(it.ficha, { atalho: it.atalho, executavel: it.executavel }), fase => {
            if (fase.fase === "fim" || fase.erro)
                return;
            const campos = { fase: fase };
            if (fase.pct !== undefined)
                campos.pct = fase.pct;
            pagina.muda(arquivo, campos);
        }, ultima => {
            if (ultima.fase === "fim" && ultima.ok)
                pagina.muda(arquivo, { estado: "feito", fim: ultima, pct: 100 });
            else
                pagina.muda(arquivo, { estado: "pronto", erro: ultima.erro || "a instalação parou sem dizer por quê" });
            pagina.lerInstalados();
        });
    }

    // O próximo executável da lista, para escolher para quem é o atalho.
    function outroExecutavel(it) {
        const lista = it.ficha.compactado.executaveis;
        const i = lista.indexOf(it.executavel);
        muda(it.arquivo, { executavel: lista[(i + 1) % lista.length] });
    }

    function tira(arquivo) {
        const i = Util.indice(itens, arquivo);
        if (i < 0)
            return;
        const n = itens.slice();
        n.splice(i, 1);
        itens = n;
        atual = n.length === 0 ? -1 : Math.min(atual, n.length - 1);
    }

    function lerInstalados() {
        Motor.pedir(["instalados"], r => {
            pagina.erroDosInstalados = r.erro || "";
            pagina.instalados = r.itens || [];
        });
    }

    function desinstala(id) {
        confirmando = "";
        Motor.pedir(["desinstalar", id], r => {
            pagina.erroDosInstalados = r.erro || "";
            pagina.lerInstalados();
            // As fichas abertas podem ter mudado (o "já instalado").
            for (const it of pagina.itens)
                if (it.estado === "pronto" || it.estado === "feito")
                    pagina.inspeciona(it.arquivo);
        });
    }

    Component.onCompleted: {
        lerInstalados();
        // O que veio na linha de comando; sem motor (exemplo), um de cada tipo.
        if (Motor.arquivos.length > 0)
            entram(Motor.arquivos);
        else if (Motor.exemplo !== "")
            entram(["/exemplo/Krita-5.2.2-x86_64.AppImage", "/exemplo/htop-3.3.0-3-x86_64.pkg.tar.zst", "/exemplo/firefox-128.0.tar.xz"]);
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 16

        RowLayout {
            Layout.fillWidth: true
            spacing: 16
            Cabecalho {
                titulo: "Instalar"
                frase: "AppImage, pacote do Arch ou arquivo compactado: veja o que é antes de instalar."
            }
            Item { Layout.fillWidth: true }
            Segmento {
                Layout.alignment: Qt.AlignBottom
                opcoes: [{ v: "instalar", texto: "instalar" }, { v: "instalados", texto: "instalados  " + pagina.instalados.length }]
                valor: pagina.aba
                onEscolhido: v => {
                    pagina.aba = v;
                    pagina.confirmando = "";
                    if (v === "instalados")
                        pagina.lerInstalados();
                }
            }
        }

        // ── instalar ───────────────────────────────────────────────────
        Solta {
            Layout.fillWidth: true
            visible: pagina.aba === "instalar"
            titulo: "Escolher o que instalar"
            texto: "solte aqui um AppImage, um pacote (.pkg.tar.zst) ou um .tar.gz, .tar.xz, .zip"
            filtros: ["Instaláveis (*.AppImage *.appimage *.pkg.tar.zst *.pkg.tar.xz *.tar.gz *.tgz *.tar.xz *.tar.zst *.tar.bz2 *.zip)", "Todos os arquivos (*)"]
            onArquivos: caminhos => pagina.entram(caminhos)
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: pagina.aba === "instalar"
            spacing: 16

            // A lista do que foi escolhido.
            Rectangle {
                Layout.preferredWidth: 250
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
                    model: pagina.itens

                    delegate: Rectangle {
                        id: linha
                        required property var modelData
                        required property int index
                        width: ListView.view.width
                        height: 48
                        color: pagina.atual === index ? Theme.wash : Qt.rgba(0, 0, 0, 0)

                        Rectangle { visible: pagina.atual === linha.index; width: 2; height: parent.height; color: Theme.copper }
                        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                        Column {
                            x: 12
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width - 24
                            spacing: 3
                            Text {
                                width: parent.width
                                text: linha.modelData.ficha ? linha.modelData.ficha.nome : Util.nome(linha.modelData.arquivo)
                                color: linha.modelData.estado === "erro" ? Theme.textDim : Theme.text
                                font.family: Theme.sans
                                font.pixelSize: 13
                                elide: Text.ElideMiddle
                            }
                            Rotulo {
                                width: parent.width
                                elide: Text.ElideRight
                                color: linha.modelData.estado === "feito" || linha.modelData.estado === "erro" ? Theme.copper : Theme.textFaint
                                text: linha.modelData.estado === "lendo" ? "lendo…"
                                    : linha.modelData.estado === "erro" ? "não dá para instalar"
                                    : linha.modelData.estado === "instalando" ? "instalando"
                                    : linha.modelData.estado === "feito" ? "instalado"
                                    : Util.rotuloDoTipo(linha.modelData.ficha.tipo)
                            }
                        }
                        MouseArea {
                            anchors.fill: parent
                            cursorShape: Qt.PointingHandCursor
                            onClicked: pagina.atual = linha.index
                        }
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: lista.count === 0
                        text: "nada escolhido ainda"
                        color: Theme.textFaint
                        font.family: Theme.serif
                        font.pixelSize: 14
                        font.italic: true
                    }
                }
            }

            // O que o escolhido é, o plano e o botão.
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: Qt.rgba(0, 0, 0, 0)
                border.width: 1
                border.color: Theme.hairline

                Text {
                    anchors.centerIn: parent
                    visible: pagina.item === null
                    text: "o plano de instalação aparece aqui"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }

                Flickable {
                    id: rola
                    anchors.fill: parent
                    anchors.margins: 18
                    anchors.bottomMargin: 62
                    visible: pagina.item !== null
                    clip: true
                    contentHeight: detalhe.height
                    boundsBehavior: Flickable.StopAtBounds

                    Column {
                        id: detalhe
                        width: rola.width
                        spacing: 10

                        Text {
                            width: parent.width
                            text: pagina.item ? (pagina.item.ficha ? pagina.item.ficha.nome : Util.nome(pagina.item.arquivo)) : ""
                            color: Theme.text
                            font.family: Theme.serif
                            font.pixelSize: 22
                            font.weight: Font.Light
                            elide: Text.ElideRight
                        }
                        Rotulo {
                            width: parent.width
                            elide: Text.ElideRight
                            text: pagina.item && pagina.item.ficha ? Util.resumoDaFicha(pagina.item.ficha) : pagina.item && pagina.item.estado === "lendo" ? "lendo o arquivo…" : ""
                        }
                        Text {
                            width: parent.width
                            visible: text !== ""
                            text: pagina.item && pagina.item.ficha && pagina.item.ficha.descricao ? pagina.item.ficha.descricao : ""
                            color: Theme.textSoft
                            font.family: Theme.sans
                            font.pixelSize: 12
                            wrapMode: Text.WordWrap
                        }
                        Text {
                            width: parent.width
                            text: pagina.item ? pagina.item.arquivo : ""
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideMiddle
                        }

                        // O erro (do arquivo ou da instalação).
                        Text {
                            width: parent.width
                            visible: text !== ""
                            text: pagina.item ? pagina.item.erro : ""
                            color: Theme.copper
                            font.family: Theme.sans
                            font.pixelSize: 13
                            wrapMode: Text.WordWrap
                        }

                        // O plano, passo a passo.
                        Item { width: 1; height: 2; visible: plano.count > 0 }
                        Rotulo { text: "o que vai ser feito"; visible: plano.count > 0 }
                        Repeater {
                            id: plano
                            model: pagina.item && pagina.item.ficha && pagina.item.estado !== "feito" ? pagina.item.ficha.plano : []
                            Row {
                                id: passo
                                required property string modelData
                                required property int index
                                width: detalhe.width
                                spacing: 10
                                Text {
                                    width: 16
                                    text: String(passo.index + 1)
                                    color: Theme.copper
                                    font.family: Theme.mono
                                    font.pixelSize: 11
                                }
                                Text {
                                    width: parent.width - 26
                                    text: passo.modelData
                                    color: Theme.textSoft
                                    font.family: Theme.sans
                                    font.pixelSize: 13
                                    wrapMode: Text.WordWrap
                                }
                            }
                        }

                        // As dependências de um pacote.
                        Text {
                            width: parent.width
                            visible: text !== ""
                            text: pagina.item && pagina.item.ficha && pagina.item.ficha.pacote && pagina.item.ficha.pacote.dependencias.length > 0
                                ? "depende de: " + pagina.item.ficha.pacote.dependencias.join(", ") : ""
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            wrapMode: Text.WordWrap
                            maximumLineCount: 4
                            elide: Text.ElideRight
                        }

                        // Os avisos (falta o FUSE, o atalho sai mínimo, pacote roda como root…).
                        Repeater {
                            model: pagina.item && pagina.item.ficha ? (pagina.item.estado === "feito" && pagina.item.fim ? pagina.item.fim.avisos : pagina.item.ficha.avisos) : []
                            Text {
                                required property string modelData
                                width: detalhe.width
                                text: "atenção: " + modelData
                                color: Theme.copper
                                font.family: Theme.sans
                                font.pixelSize: 12
                                wrapMode: Text.WordWrap
                            }
                        }

                        // As opções: o atalho de menu e, num compactado, para quem ele é.
                        Row {
                            visible: pagina.item !== null && pagina.item.estado === "pronto" && pagina.item.ficha !== null && pagina.item.ficha.atalho !== null
                            spacing: 10
                            Chave {
                                anchors.verticalCenter: parent.verticalCenter
                                ligado: pagina.item ? pagina.item.atalho : false
                                onTrocado: pagina.muda(pagina.item.arquivo, { atalho: !pagina.item.atalho })
                            }
                            Text {
                                anchors.verticalCenter: parent.verticalCenter
                                text: "atalho no menu de aplicativos"
                                color: Theme.textSoft
                                font.family: Theme.sans
                                font.pixelSize: 12
                            }
                        }
                        Row {
                            visible: pagina.item !== null && pagina.item.estado === "pronto" && pagina.item.atalho && pagina.item.executavel !== ""
                            spacing: 10
                            Text {
                                anchors.verticalCenter: parent.verticalCenter
                                width: Math.min(implicitWidth, detalhe.width - 90)
                                text: "o atalho abre  " + (pagina.item ? pagina.item.executavel : "")
                                color: Theme.textDim
                                font.family: Theme.mono
                                font.pixelSize: 11
                                elide: Text.ElideMiddle
                            }
                            Botao {
                                anchors.verticalCenter: parent.verticalCenter
                                visible: pagina.item !== null && pagina.item.ficha !== null && pagina.item.ficha.compactado !== null
                                    && pagina.item.ficha.compactado.executaveis.length > 1
                                texto: "outro"
                                onAcionado: pagina.outroExecutavel(pagina.item)
                            }
                        }

                        // O resultado.
                        Text {
                            width: parent.width
                            visible: pagina.item !== null && pagina.item.estado === "feito"
                            text: pagina.item && pagina.item.fim
                                ? (pagina.item.fim.tipo === "pacote" ? "instalado pelo pacman. Para tirar, use o MkPacotes ou o pacman."
                                    : "instalado em " + pagina.item.fim.destino + (pagina.item.fim.atalho ? ". Já está no menu de aplicativos." : "."))
                                : ""
                            color: Theme.text
                            font.family: Theme.sans
                            font.pixelSize: 13
                            wrapMode: Text.WrapAnywhere
                        }
                    }
                }

                // O pé: o progresso e os botões.
                Item {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.bottom: parent.bottom
                    anchors.margins: 18
                    height: 30
                    visible: pagina.item !== null

                    Column {
                        anchors.left: parent.left
                        anchors.right: botoes.left
                        anchors.rightMargin: 16
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 6
                        visible: pagina.item !== null && pagina.item.estado === "instalando"
                        Barra {
                            width: parent.width
                            pct: pagina.item ? pagina.item.pct : 0
                        }
                        Text {
                            width: parent.width
                            text: pagina.item ? Util.faseDaInstalacao(pagina.item.fase) : ""
                            color: Theme.textDim
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideRight
                        }
                    }

                    Row {
                        id: botoes
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 8
                        Botao {
                            texto: "tirar da lista"
                            ligado: pagina.item !== null && pagina.item.estado !== "instalando"
                            onAcionado: pagina.tira(pagina.item.arquivo)
                        }
                        Botao {
                            visible: pagina.item !== null && pagina.item.estado !== "feito" && pagina.item.estado !== "erro"
                            forte: true
                            ligado: pagina.item !== null && pagina.item.estado === "pronto" && !pagina.ocupada
                            texto: pagina.item ? Util.botaoInstalar(pagina.item.ficha) : "instalar"
                            onAcionado: pagina.instala(pagina.item)
                        }
                    }
                }
            }
        }

        // ── instalados ─────────────────────────────────────────────────
        Text {
            Layout.fillWidth: true
            visible: pagina.aba === "instalados" && pagina.erroDosInstalados !== ""
            text: pagina.erroDosInstalados
            color: Theme.copper
            font.family: Theme.sans
            font.pixelSize: 12
            wrapMode: Text.WordWrap
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: pagina.aba === "instalados"
            color: Qt.rgba(0, 0, 0, 0)
            border.width: 1
            border.color: Theme.hairline

            ListView {
                id: feitos
                anchors.fill: parent
                anchors.margins: 1
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pagina.instalados

                delegate: Item {
                    id: posto
                    required property var modelData
                    width: ListView.view.width
                    height: 50

                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.boneA(0.06) }
                    Column {
                        x: 14
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width - 370
                        spacing: 3
                        Text {
                            width: parent.width
                            text: posto.modelData.nome + (posto.modelData.versao ? "  " + posto.modelData.versao : "")
                            color: posto.modelData.existe ? Theme.text : Theme.textDim
                            font.family: Theme.sans
                            font.pixelSize: 13
                            elide: Text.ElideRight
                        }
                        Text {
                            width: parent.width
                            text: posto.modelData.existe ? posto.modelData.destino : "sumiu de " + posto.modelData.destino
                            color: Theme.textFaint
                            font.family: Theme.mono
                            font.pixelSize: 10
                            elide: Text.ElideMiddle
                        }
                    }
                    Rotulo {
                        anchors.right: tirar.left
                        anchors.rightMargin: 16
                        anchors.verticalCenter: parent.verticalCenter
                        text: Util.rotuloDoTipo(posto.modelData.tipo) + "  ·  " + Util.tamanho(posto.modelData.tamanho_bytes)
                    }
                    Botao {
                        id: tirar
                        anchors.right: parent.right
                        anchors.rightMargin: 14
                        anchors.verticalCenter: parent.verticalCenter
                        forte: pagina.confirmando === posto.modelData.id
                        texto: pagina.confirmando === posto.modelData.id ? "apagar mesmo?" : "desinstalar"
                        onAcionado: {
                            if (pagina.confirmando === posto.modelData.id)
                                pagina.desinstala(posto.modelData.id);
                            else
                                pagina.confirmando = posto.modelData.id;
                        }
                    }
                }

                Text {
                    anchors.centerIn: parent
                    visible: feitos.count === 0
                    text: "o MkInstalar ainda não instalou nada"
                    color: Theme.textFaint
                    font.family: Theme.serif
                    font.pixelSize: 14
                    font.italic: true
                }
            }
        }
    }
}
