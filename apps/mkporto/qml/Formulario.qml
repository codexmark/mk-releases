import QtQuick
import "Porto.js" as Porto

// O formulário de host (novo ou edição). Confere o que se digita na hora, com
// as mesmas regras do motor (Porto.validar); quem grava é o motor, que confere
// tudo de novo. Não há campo de senha: a senha, se houver, é com o ssh.
Item {
    id: formulario

    // Os hosts que já existem e as chaves públicas (para sugerir a privada).
    property var hosts: []
    property var chaves: []
    // O apelido de quem está sendo editado; "" num host novo.
    property string anterior: ""
    // O que o motor respondeu de errado ao guardar, e se há um pedido andando.
    property string erro: ""
    property bool ocupado: false
    // Só reclama de um campo vazio depois que a pessoa tentou guardar.
    property bool tentou: false
    signal guardar(var pedido)
    signal fechado()

    readonly property var form: ({
        apelido: apelido.text, host: host.text, usuario: usuario.text,
        porta: porta.text, chave: chave.text, salto: salto.text
    })
    readonly property var erros: Porto.validar(form, hosts, anterior)
    readonly property bool bom: Porto.valido(erros)

    // Abre com os campos de `inicial` (Porto.formularioDe ou formularioVazio).
    function abrir(inicial, apelidoAnterior) {
        anterior = apelidoAnterior || "";
        tentou = false;
        apelido.text = inicial.apelido;
        host.text = inicial.host;
        usuario.text = inicial.usuario;
        porta.text = inicial.porta;
        chave.text = inicial.chave;
        salto.text = inicial.salto;
        apelido.focar();
    }

    function tenta() {
        tentou = true;
        if (bom && !ocupado)
            guardar(Porto.pedido(form, anterior));
    }

    // O erro de um campo: o vazio só aparece depois da primeira tentativa.
    function erroDe(campo, texto) {
        return texto === "" && !tentou ? "" : erros[campo];
    }

    Fundo {
        anchors.fill: parent
        onFechado: formulario.fechado()
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

                Rotulo { text: formulario.anterior === "" ? "novo host" : "editar  /  " + formulario.anterior }
                Text {
                    text: formulario.anterior === "" ? "Um host novo" : "Editar o host"
                    color: Theme.text
                    font.family: Theme.serif
                    font.pixelSize: 22
                    font.weight: Font.Light
                    bottomPadding: 4
                }

                Row {
                    width: parent.width
                    spacing: 16
                    Entrada {
                        id: apelido
                        width: (parent.width - 16) * 0.4
                        rotulo: "apelido"
                        dica: "atlas"
                        erro: formulario.erroDe("apelido", text)
                        onAceito: formulario.tenta()
                    }
                    Entrada {
                        id: host
                        width: (parent.width - 16) * 0.6
                        rotulo: "endereço"
                        dica: "atlas.oficina.example ou 192.0.2.10"
                        mono: true
                        erro: formulario.erroDe("host", text)
                        onAceito: formulario.tenta()
                    }
                }
                Row {
                    width: parent.width
                    spacing: 16
                    Entrada {
                        id: usuario
                        width: (parent.width - 16) * 0.7
                        rotulo: "usuário"
                        dica: "o seu, se ficar vazio"
                        erro: formulario.erroDe("usuario", text)
                        onAceito: formulario.tenta()
                    }
                    Entrada {
                        id: porta
                        width: (parent.width - 16) * 0.3
                        rotulo: "porta"
                        dica: "22"
                        mono: true
                        erro: formulario.erroDe("porta", text)
                        onAceito: formulario.tenta()
                    }
                }
                Entrada {
                    id: chave
                    width: parent.width
                    rotulo: "chave"
                    dica: "o caminho da chave; vazio deixa o ssh escolher"
                    mono: true
                    erro: formulario.erroDe("chave", text)
                    onAceito: formulario.tenta()
                }
                // As chaves de ~/.ssh, pelo nome: um toque preenche o campo. O
                // arquivo da chave privada nunca é aberto pelo MkPorto.
                Flow {
                    width: parent.width
                    spacing: 6
                    visible: formulario.chaves.length > 0

                    Repeater {
                        model: formulario.chaves

                        Botao {
                            id: sugestao
                            required property var modelData
                            texto: Porto.sugestaoDeChave(modelData)
                            visible: texto !== ""
                            onAcionado: chave.text = sugestao.texto
                        }
                    }
                }
                Entrada {
                    id: salto
                    width: parent.width
                    rotulo: "salto"
                    dica: "outro host por onde passar (ProxyJump); vazio é direto"
                    mono: true
                    erro: formulario.erroDe("salto", text)
                    onAceito: formulario.tenta()
                }

                Text {
                    width: parent.width
                    wrapMode: Text.WordWrap
                    visible: formulario.erro !== ""
                    text: formulario.erro
                    color: Theme.copper
                    font.family: Theme.sans
                    font.pixelSize: 12
                }

                Row {
                    spacing: 8
                    Botao {
                        texto: formulario.ocupado ? "guardando…" : "guardar"
                        forte: true
                        ligado: !formulario.ocupado && (formulario.bom || !formulario.tentou)
                        onAcionado: formulario.tenta()
                    }
                    Botao {
                        texto: "cancelar"
                        onAcionado: formulario.fechado()
                    }
                }
                Rotulo { text: "vai para ~/.ssh/config.d/mkporto.conf  ·  esc fecha" }
            }
        }
    }
}
