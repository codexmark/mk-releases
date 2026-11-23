import QtQuick
import QtQuick.Layouts

// Página "Monitores" dos Ajustes: o desenho das telas, a lista delas e os
// controles da escolhida. Tudo aqui mexe só no rascunho (Monitors.draft); a
// tela muda em "aplicar", e volta sozinha em 15 s se ninguém confirmar.
ColumnLayout {
    id: page
    spacing: 0

    readonly property var screens: Monitors.draft
    readonly property var current: Monitors.current
    // As outras telas que servem de referência para "posição": ligadas e com
    // lugar próprio (uma que espelha outra não tem).
    readonly property var others: Monitors.references(screens, Monitors.selected)
    // Quem está espelhando a tela escolhida.
    readonly property var copies: Monitors.mirrorsOf(screens, Monitors.selected)
    // Em relação a qual delas: a escolhida aqui, ou a vizinha natural.
    property string refPick: ""
    readonly property string refName: others.some(m => m.name === refPick) ? refPick
        : Monitors.neighbour(screens, Monitors.selected)
    readonly property string relation: Monitors.relationTo(screens, Monitors.selected, refName)
    property bool allModes: false
    property bool allRates: false

    readonly property var places: [
        { v: "esquerda", label: "à esquerda" },
        { v: "direita", label: "à direita" },
        { v: "acima", label: "acima" },
        { v: "abaixo", label: "abaixo" },
        { v: "espelhar", label: "espelhar" }
    ]

    Component.onCompleted: Monitors.refresh()

    Connections {
        target: Monitors
        function onSelectedChanged() {
            page.refPick = "";
            page.allModes = false;
            page.allRates = false;
        }
    }

    // Rótulo de seção: mono pequeno em maiúsculas espaçadas.
    component Section: Text {
        Layout.topMargin: 22
        Layout.bottomMargin: 4
        color: Theme.textFaint
        font.family: Theme.mono
        font.pixelSize: 9
        font.letterSpacing: Theme.labelSpacing
    }

    // Uma linha de ajuste: nome e explicação à esquerda, botões à direita,
    // encostados na borda. Se não couberem numa linha, quebram, e o bloco
    // continua à direita.
    component Line: Item {
        id: ln
        property string label: ""
        property string desc: ""
        default property alias content: flow.data

        Layout.fillWidth: true
        implicitHeight: Math.max(62, flow.implicitHeight + 28, words.implicitHeight + 24)
        readonly property real room: width - words.width - 24
        // A largura dos botões lado a lado (o Repeater, sem largura, não conta).
        readonly property real need: {
            let total = 0;
            for (let i = 0; i < flow.children.length; i++) {
                const item = flow.children[i];
                if (item.visible && item.implicitWidth > 0)
                    total += item.implicitWidth + flow.spacing;
            }
            return Math.max(0, total - flow.spacing);
        }

        Rectangle {
            width: parent.width
            height: 1
            color: Theme.hairline
        }

        Column {
            id: words
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            width: Math.min(230, parent.width * 0.36)
            spacing: 4

            Text {
                text: ln.label
                color: Theme.text
                font.family: Theme.sans
                font.pixelSize: 14
            }
            Text {
                width: parent.width
                visible: text !== ""
                text: ln.desc
                color: Theme.textDim
                font.family: Theme.sans
                font.pixelSize: 12
                wrapMode: Text.WordWrap
            }
        }

        Flow {
            id: flow
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            width: Math.min(ln.room, ln.need)
            spacing: 6
        }
    }

    // O desenho: cada tela ligada no seu lugar, na proporção certa.
    Item {
        id: diagram
        Layout.fillWidth: true
        Layout.topMargin: 18
        implicitHeight: 190

        readonly property var shown: page.screens.filter(m => Monitors.placed(m))
        readonly property var box: Monitors.bounds(shown)
        readonly property real k: Math.min((width - 56) / box.w, (height - 44) / box.h)
        readonly property real ox: (width - box.w * k) / 2
        readonly property real oy: (height - box.h * k) / 2

        Rectangle {
            anchors.fill: parent
            radius: 2
            color: Theme.surfaceLow
            border.width: 1
            border.color: Theme.hairline
        }

        Text {
            anchors.centerIn: parent
            visible: diagram.shown.length === 0
            text: Monitors.loaded ? "nenhuma tela com lugar próprio" : "lendo as telas…"
            color: Theme.textDim
            font.family: Theme.serif
            font.pixelSize: 14
            font.italic: true
        }

        Repeater {
            model: diagram.shown

            Rectangle {
                id: tile
                required property var modelData
                readonly property bool picked: modelData.name === Monitors.selected
                readonly property var size: Monitors.logical(modelData)
                readonly property var copies: page.screens.filter(m => m.enabled && m.mirror === modelData.name)

                x: Math.round(diagram.ox + (modelData.x - diagram.box.x) * diagram.k)
                y: Math.round(diagram.oy + (modelData.y - diagram.box.y) * diagram.k)
                width: Math.round(size.w * diagram.k)
                height: Math.round(size.h * diagram.k)
                z: picked ? 1 : 0
                radius: 2
                color: boxHover.hovered || picked ? Theme.raised : Theme.surface
                border.width: 1
                border.color: picked ? Theme.copper : Theme.line

                Column {
                    anchors.centerIn: parent
                    width: parent.width - 8
                    spacing: 3

                    Text {
                        width: parent.width
                        horizontalAlignment: Text.AlignHCenter
                        elide: Text.ElideRight
                        text: tile.modelData.name
                        color: tile.picked ? Theme.copper : Theme.text
                        font.family: Theme.mono
                        font.pixelSize: 11
                    }
                    Text {
                        width: parent.width
                        visible: tile.height > 54
                        horizontalAlignment: Text.AlignHCenter
                        elide: Text.ElideRight
                        text: tile.copies.length > 0 ? "+ " + tile.copies.map(m => m.name).join(", ")
                            : tile.size.w + " × " + tile.size.h
                        color: Theme.textFaint
                        font.family: Theme.mono
                        font.pixelSize: 9
                    }
                }

                HoverHandler {
                    id: boxHover
                    cursorShape: Qt.PointingHandCursor
                }
                TapHandler {
                    onTapped: Monitors.select(tile.modelData.name)
                }
            }
        }
    }

    Text {
        visible: Monitors.overlap || Monitors.loose
        Layout.fillWidth: true
        Layout.topMargin: 8
        wrapMode: Text.WordWrap
        text: Monitors.overlap ? "duas telas estão uma por cima da outra: escolha um lado em posição"
            : "uma tela não encosta em nenhuma outra, e o mouse não chega nela: escolha um lado em posição"
        color: Theme.copper
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }

    // Todas as saídas ligadas ao computador, também as desligadas.
    Section {
        text: "TELAS"
    }

    Repeater {
        model: page.screens

        Item {
            id: row
            required property var modelData
            readonly property bool picked: modelData.name === Monitors.selected

            Layout.fillWidth: true
            implicitHeight: 46

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.hairline
            }
            Rectangle {
                anchors.fill: parent
                anchors.topMargin: 1
                color: rowHover.hovered ? Theme.wash : "transparent"
            }
            Rectangle {
                visible: row.picked
                anchors.left: parent.left
                anchors.verticalCenter: parent.verticalCenter
                width: 2
                height: 18
                color: Theme.copper
            }

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 12
                anchors.rightMargin: 4
                spacing: 12

                Text {
                    text: row.modelData.name
                    color: row.picked ? Theme.copper : row.modelData.enabled ? Theme.text : Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 12
                }
                Text {
                    Layout.fillWidth: true
                    elide: Text.ElideRight
                    text: row.modelData.description
                    color: Theme.textDim
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Text {
                    text: Monitors.summary(row.modelData)
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 10
                }
            }

            HoverHandler {
                id: rowHover
                cursorShape: Qt.PointingHandCursor
            }
            TapHandler {
                onTapped: Monitors.select(row.modelData.name)
            }
        }
    }

    // Os controles da tela escolhida.
    Section {
        visible: page.current !== null
        text: page.current ? page.current.name.toUpperCase() : ""
    }

    // O monitors.lua pessoal carrega por último: com regra lá, o que o painel
    // grava não chega a esta tela.
    Text {
        visible: page.current !== null && Monitors.ruled.indexOf(page.current.name) >= 0
        Layout.fillWidth: true
        Layout.bottomMargin: 8
        wrapMode: Text.WordWrap
        text: (page.current ? page.current.name : "") + " tem regra em hypr/monitors.lua, que vale sobre este painel:"
            + " o que mudar aqui não chega à tela enquanto a regra estiver lá"
            + " (o painel só procura o nome e a descrição da tela no arquivo, não lê as regras)"
        color: Theme.copper
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
    }

    Line {
        visible: page.current !== null && page.screens.length > 1
        label: "Ligada"
        desc: page.current && page.current.enabled && Monitors.enabledCount(page.screens) <= 1
            ? "É a única tela ligada: não dá para desligar"
            : "Desligada, as janelas dela vão para as outras. Se ficar sozinha no cabo, religa"

        MkToggle {
            checked: page.current !== null && page.current.enabled
            opacity: page.current && page.current.enabled && Monitors.enabledCount(page.screens) <= 1 ? 0.4 : 1
            onToggled: v => Monitors.toggle(v)
        }
    }

    Line {
        id: resLine
        readonly property var all: Monitors.resolutions(page.current)
        // As primeiras, e sempre a que está em uso.
        readonly property var few: all.filter((r, i) => i < 3 || (page.current && r.w === page.current.w && r.h === page.current.h))

        visible: page.current !== null && page.current.enabled && all.length > 0
        label: "Resolução"
        desc: "Quantos pontos a tela mostra"

        Repeater {
            model: page.allModes ? resLine.all : resLine.few

            PillButton {
                required property var modelData
                text: modelData.w + " × " + modelData.h
                selected: page.current !== null && page.current.w === modelData.w && page.current.h === modelData.h
                onClicked: Monitors.setMode(modelData.w, modelData.h)
            }
        }
        PillButton {
            visible: resLine.all.length > resLine.few.length
            text: page.allModes ? "menos" : "todas"
            onClicked: page.allModes = !page.allModes
        }
    }

    Line {
        id: rateLine
        readonly property var all: Monitors.rates(page.current)
        // Acima da lista do monitor, dentro dos limites que ele declara: as
        // duas mais altas, e sempre a que está em uso.
        readonly property var extra: Monitors.extraRates(page.current)
        readonly property var few: extra.filter((r, i) => i < 2 || (page.current && r === page.current.refresh))

        visible: page.current !== null && page.current.enabled && all.length > 0
        label: "Taxa"
        desc: extra.length === 0 ? "Quadros por segundo nesta resolução"
            : "Quadros por segundo nesta resolução. Com *, fora da lista do monitor:"
                + " o sinal é calculado pelos limites que ele declara, e pode não pegar"

        Repeater {
            model: page.allRates ? rateLine.extra : rateLine.few

            PillButton {
                required property var modelData
                text: Monitors.hz(modelData) + " *"
                selected: page.current !== null && page.current.refresh === modelData
                onClicked: Monitors.setMode(page.current.w, page.current.h, modelData)
            }
        }
        Repeater {
            model: rateLine.all

            PillButton {
                required property var modelData
                text: Monitors.hz(modelData)
                selected: page.current !== null && page.current.refresh === modelData
                onClicked: Monitors.setMode(page.current.w, page.current.h, modelData)
            }
        }
        PillButton {
            visible: rateLine.extra.length > rateLine.few.length
            text: page.allRates ? "menos" : "mais *"
            onClicked: page.allRates = !page.allRates
        }
    }

    Line {
        visible: page.current !== null && page.current.enabled && page.current.w > 0
        label: "Escala"
        readonly property var missing: Monitors.missingScales(page.current)
        desc: missing.length === 0 ? "O tamanho de tudo nesta tela"
            : "Sem " + missing.map(s => Monitors.percent(s)).join(" e ")
                + ": o Hyprland só aceita escalas que fecham com esta resolução"

        Repeater {
            model: page.current ? Monitors.scaleOptions(page.current) : []

            PillButton {
                required property var modelData
                text: Monitors.percent(modelData)
                selected: page.current !== null && Monitors.sameScale(page.current.scale, modelData)
                onClicked: Monitors.setScale(modelData)
            }
        }
    }

    Line {
        visible: page.current !== null && page.current.enabled
        label: "Rotação"
        desc: "Para uma tela em pé ou invertida"

        Repeater {
            model: [0, 1, 2, 3]

            PillButton {
                required property var modelData
                text: modelData * 90 + "°"
                selected: page.current !== null && page.current.transform % 4 === modelData
                onClicked: Monitors.setTransform(modelData)
            }
        }
    }

    // Uma tela que outra espelha: o espelho se desfaz daqui mesmo, sem ter
    // de adivinhar que é preciso escolher a outra.
    Line {
        visible: page.current !== null && page.current.enabled && page.copies.length > 0
        label: "Espelhada"
        desc: page.copies.join(", ") + (page.copies.length > 1 ? " mostram" : " mostra") + " a mesma imagem"

        Repeater {
            model: page.copies

            PillButton {
                required property var modelData
                text: page.copies.length > 1 ? "separar " + modelData : "desfazer espelho"
                onClicked: Monitors.splitMirror(modelData)
            }
        }
    }

    Line {
        visible: page.current !== null && page.current.enabled && page.refName !== ""
        label: "Posição"
        desc: "Em relação a " + page.refName + ". O mouse passa por esse lado"

        Repeater {
            model: page.places

            PillButton {
                required property var modelData
                text: modelData.label
                selected: page.relation === modelData.v
                onClicked: Monitors.setPlace(modelData.v, page.refName)
            }
        }
    }

    Line {
        visible: page.current !== null && page.current.enabled && page.others.length > 1
        label: "Em relação a"
        desc: "A tela que serve de referência"

        Repeater {
            model: page.others

            PillButton {
                required property var modelData
                text: modelData.name
                selected: page.refName === modelData.name
                onClicked: page.refPick = modelData.name
            }
        }
    }

    Rectangle {
        Layout.fillWidth: true
        implicitHeight: 1
        color: Theme.hairline
    }

    // Aplicar, e depois a pergunta com a contagem.
    Item {
        id: bar
        Layout.fillWidth: true
        Layout.topMargin: 18
        implicitHeight: 60

        readonly property bool asking: Monitors.pending
        readonly property bool ready: Monitors.dirty

        Rectangle {
            anchors.fill: parent
            radius: 2
            color: bar.asking ? Theme.surface : "transparent"
            border.width: 1
            border.color: bar.asking ? Theme.copper : bar.ready ? Theme.line : Theme.hairline
        }

        // O tempo que resta, escorrendo na borda de baixo.
        Rectangle {
            visible: bar.asking
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.margins: 1
            height: 2
            width: (parent.width - 2) * Math.max(0, Monitors.remaining) / Monitors.trialSeconds
            color: Theme.copper

            Behavior on width {
                NumberAnimation { duration: 1000 }
            }
        }

        Column {
            anchors.left: parent.left
            anchors.leftMargin: 16
            anchors.right: buttons.left
            anchors.rightMargin: 16
            anchors.verticalCenter: parent.verticalCenter
            spacing: 4

            Text {
                width: parent.width
                elide: Text.ElideRight
                text: bar.asking ? (Monitors.differs ? "não ficou como pedido. manter assim mesmo?" : "manter esta configuração?")
                    : Monitors.error !== "" ? Monitors.error
                    : bar.ready ? "mudanças por aplicar" : "nada por aplicar"
                color: bar.asking || bar.ready ? Theme.text : Theme.textDim
                font.family: Theme.serif
                font.pixelSize: 15
                font.italic: true
            }
            Text {
                width: parent.width
                elide: Text.ElideRight
                text: bar.asking ? "VOLTA EM " + Math.max(0, Monitors.remaining) + " S"
                        + (Monitors.differs ? " · OUTRA REGRA (HYPR/MONITORS.LUA) VALE SOBRE ESTA TELA" : "")
                    : bar.ready ? "A TELA SÓ MUDA AO APLICAR · SEM RESPOSTA EM 15 S, VOLTA"
                    : "ESCOLHA UMA TELA E MUDE O QUE QUISER"
                color: bar.asking ? Theme.copper : Theme.textFaint
                font.family: Theme.mono
                font.pixelSize: 9
                font.letterSpacing: Theme.labelSpacing
            }
        }

        Row {
            id: buttons
            anchors.right: parent.right
            anchors.rightMargin: 13
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6

            PillButton {
                visible: bar.asking
                text: "voltar"
                onClicked: Monitors.revert()
            }
            PillButton {
                visible: bar.asking
                primary: true
                text: "manter"
                onClicked: Monitors.keep()
            }
            PillButton {
                visible: !bar.asking && bar.ready
                text: "descartar"
                onClicked: Monitors.discard()
            }
            PillButton {
                visible: !bar.asking && bar.ready
                primary: true
                text: "aplicar"
                onClicked: Monitors.apply()
            }
        }
    }

    Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        text: "Se a imagem sumir depois de aplicar, não mexa em nada: em 15 segundos a tela volta ao que era."
            + (Monitors.saved.some(m => !Monitors.find(page.screens, m.name))
                ? " As telas que não estão no cabo agora continuam guardadas: "
                    + Monitors.saved.filter(m => !Monitors.find(page.screens, m.name)).map(m => m.name).join(", ") + "."
                : "")
        color: Theme.textDim
        font.family: Theme.serif
        font.pixelSize: 13
        font.italic: true
        wrapMode: Text.WordWrap
    }
}
