import QtQuick
import QtQuick.Dialogs
import "Porto.js" as Porto

// A folha da máquina nova. Confere o que se digita com as regras do
// `vm criar` (Porto.validarMaquina); quem cria (a pasta, o disco) é o motor,
// que confere tudo de novo e ainda olha se a ISO existe.
Item {
    id: nova

    property var maquinas: []
    // A resposta do `vm requisitos`: o UEFI precisa do OVMF.
    property var requisitos: null
    property string erro: ""
    property bool ocupado: false
    property bool tentou: false
    property bool uefi: false
    property bool tresD: false
    signal criar(var pedido)
    signal fechado()

    readonly property var form: ({
        nome: nome.text, iso: iso.text, memoria_mb: memoria.text, cpus: cpus.text, disco_gb: disco.text,
        uefi: nova.uefi, tres_d: nova.tresD
    })
    readonly property var erros: Porto.validarMaquina(form, maquinas, requisitos)
    readonly property bool bom: Porto.maquinaValida(erros)

    function abrir() {
        const vazia = Porto.maquinaVazia(requisitos);
        tentou = false;
        nome.text = vazia.nome;
        iso.text = vazia.iso;
        memoria.text = vazia.memoria_mb;
        cpus.text = vazia.cpus;
        disco.text = vazia.disco_gb;
        uefi = vazia.uefi;
        tresD = vazia.tres_d;
        nome.focar();
    }

    function tenta() {
        tentou = true;
        if (bom && !ocupado)
            criar(Porto.pedidoDeMaquina(form));
    }

    Fundo {
        anchors.fill: parent
        onFechado: nova.fechado()
    }

    // O seletor de arquivos do sistema, para a ISO.
    FileDialog {
        id: seletor
        title: "A ISO de instalação"
        fileMode: FileDialog.OpenFile
        nameFilters: ["Imagens de disco (*.iso *.img)", "Todos os arquivos (*)"]
        onAccepted: {
            const caminho = Porto.caminhoDeUrl(selectedFile.toString());
            if (caminho !== "")
                iso.text = caminho;
        }
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(560, parent.width - 32)
        height: Math.min(parent.height - 24, corpo.implicitHeight + 40)
        color: Theme.surface
        border.width: 1
        border.color: Theme.line
        clip: true

        // Um clique no cartão não fecha a folha.
        MouseArea { anchors.fill: parent }

        Flickable {
            id: rolo
            anchors.fill: parent
            anchors.margins: 20
            contentHeight: corpo.implicitHeight
            boundsBehavior: Flickable.StopAtBounds
            clip: true

            Column {
                id: corpo
                width: rolo.width
                spacing: 8

                Rotulo { text: "nova máquina  /  qemu" }
                Text {
                    text: "Uma máquina nova"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                    bottomPadding: 4
                }
                Entrada {
                    id: nome
                    width: parent.width
                    rotulo: "nome"
                    dica: "oficina (minúsculas, algarismos, ponto, hífen)"
                    mono: true
                    erro: text === "" && !nova.tentou ? "" : nova.erros.nome
                    onAceito: nova.tenta()
                }
                Row {
                    width: parent.width
                    spacing: 8
                    Entrada {
                        id: iso
                        width: parent.width - escolher.width - 8
                        rotulo: "iso de instalação"
                        dica: "o caminho inteiro da ISO; vazio é sem CD"
                        mono: true
                        erro: nova.erros.iso
                        onAceito: nova.tenta()
                    }
                    Botao {
                        id: escolher
                        y: 14
                        texto: "escolher…"
                        onAcionado: seletor.open()
                    }
                }
                Row {
                    width: parent.width
                    spacing: 16
                    Entrada {
                        id: memoria
                        width: (parent.width - 32) / 3
                        rotulo: "memória (mb)"
                        dica: "4096"
                        mono: true
                        erro: nova.erros.memoria_mb
                        onAceito: nova.tenta()
                    }
                    Entrada {
                        id: cpus
                        width: (parent.width - 32) / 3
                        rotulo: "cpus"
                        dica: "2"
                        mono: true
                        erro: nova.erros.cpus
                        onAceito: nova.tenta()
                    }
                    Entrada {
                        id: disco
                        width: (parent.width - 32) / 3
                        rotulo: "disco (gb)"
                        dica: "40"
                        mono: true
                        erro: nova.erros.disco_gb
                        onAceito: nova.tenta()
                    }
                }
                Row {
                    spacing: 10
                    Chave {
                        anchors.verticalCenter: parent.verticalCenter
                        ligado: nova.uefi
                        onTrocado: nova.uefi = !nova.uefi
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        text: "UEFI (os sistemas de hoje esperam por ele)"
                        color: Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 12
                    }
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: nova.erros.uefi !== ""
                    text: nova.erros.uefi
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 10
                }
                Row {
                    spacing: 10
                    Chave {
                        anchors.verticalCenter: parent.verticalCenter
                        ligado: nova.tresD
                        onTrocado: nova.tresD = !nova.tresD
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        text: "aceleração 3D (o sistema de dentro precisa do driver virtio)"
                        color: Theme.textSoft
                        font.family: Theme.sans
                        font.pixelSize: 12
                    }
                }
                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: nova.erro !== ""
                    text: nova.erro
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Row {
                    spacing: 8
                    Botao {
                        texto: nova.ocupado ? "criando…" : "criar"
                        forte: true
                        ligado: !nova.ocupado && (nova.bom || !nova.tentou)
                        onAcionado: nova.tenta()
                    }
                    Botao {
                        texto: "cancelar"
                        onAcionado: nova.fechado()
                    }
                }
                Rotulo { text: "o disco cresce conforme o uso  ·  esc fecha" }
            }
        }
    }
}
