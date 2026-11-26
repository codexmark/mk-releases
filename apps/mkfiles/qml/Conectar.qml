import QtQuick
import "Remoto.js" as Remoto

// A folha de conectar a um servidor (Ctrl+K): o endereço inteiro, ou o
// formulário que o monta; embaixo, os hosts do ~/.ssh/config e o que a rede
// local mostra, para conectar com um clique. Quem monta é o motor (pelo gio);
// usuário e senha, se o servidor pedir, vêm depois, numa folha própria.
Folha {
    id: conectar

    // [{ apelido, host, usuario, porta }], do pedido `remoto-hosts`.
    property var hosts: []
    // [{ uri, nome }], do pedido `remotos` com rede.
    property var rede: []
    property bool conectando: false
    property string erro: ""
    // O que o gio escreveu, quando a conexão falha.
    property string saida: ""
    property string tipo: "sftp"
    readonly property bool comDominio: tipo === "smb"
    // O endereço que vale: o digitado inteiro, ou o do formulário.
    readonly property string doFormulario: Remoto.montarUri({ tipo: tipo, host: host.text, porta: porta.text, usuario: usuario.text,
        pasta: pasta.text, dominio: dominio.text }) || ""
    // Os conhecidos, numa lista só: os títulos e as linhas.
    readonly property var conhecidos: {
        const l = [];
        if (hosts.length > 0)
            l.push({ titulo: "do seu ~/.ssh/config" });
        for (const h of hosts)
            l.push({ nome: h.apelido, detalhe: Remoto.detalheDoHost(h), uri: Remoto.uriDoHost(h) });
        if (rede.length > 0)
            l.push({ titulo: "na rede local" });
        for (const r of rede)
            l.push({ nome: r.nome || r.uri, detalhe: r.uri, uri: r.uri });
        return l;
    }
    signal pedido(string uri)
    signal cancelado()

    function abrir() {
        erro = "";
        saida = "";
        endereco.text = "";
        foco.restart();
    }
    function confirmar() {
        if (conectando)
            return;
        const digitado = endereco.text.trim();
        if (digitado !== "") {
            const e = Remoto.erroDoEndereco(digitado);
            if (e !== "") {
                erro = e;
                return;
            }
            erro = "";
            pedido(digitado);
            return;
        }
        if (doFormulario === "") {
            erro = Remoto.erroDoHost(host.text) || Remoto.erroDaPorta(porta.text) || "escreva o endereço ou o servidor";
            return;
        }
        erro = "";
        pedido(doFormulario);
    }

    rotulo: "rede"
    titulo: "Conectar a um servidor"
    largura: 560

    Timer {
        id: foco
        interval: 1
        onTriggered: endereco.focar()
    }
    Campo {
        id: endereco
        width: parent.width
        mono: true
        dica: "sftp://usuario@servidor/pasta, smb://servidor/compartilhamento, ftp://, davs://"
        onAceito: conectar.confirmar()
    }
    Rotulo { text: "ou preencha" }
    Segmento {
        opcoes: Remoto.TIPOS
        valor: conectar.tipo
        onEscolhido: v => conectar.tipo = v
    }
    Row {
        width: parent.width
        spacing: 12
        Campo {
            id: host
            width: parent.width - 102
            dica: "servidor (nome ou número)"
            onAceito: conectar.confirmar()
        }
        Campo {
            id: porta
            width: 90
            dica: "porta " + Remoto.tipo(conectar.tipo).porta
            onAceito: conectar.confirmar()
        }
    }
    Row {
        width: parent.width
        spacing: 12
        Campo {
            id: usuario
            width: (parent.width - (conectar.comDominio ? 24 : 12)) / (conectar.comDominio ? 3 : 2)
            dica: "usuário (se precisar)"
            onAceito: conectar.confirmar()
        }
        Campo {
            id: dominio
            visible: conectar.comDominio
            width: usuario.width
            dica: "domínio"
            onAceito: conectar.confirmar()
        }
        Campo {
            id: pasta
            width: usuario.width
            dica: conectar.comDominio ? "compartilhamento/pasta" : "pasta"
            onAceito: conectar.confirmar()
        }
    }
    Text {
        width: parent.width
        elide: Text.ElideMiddle
        visible: endereco.text.trim() === "" && conectar.doFormulario !== ""
        text: conectar.doFormulario
        color: Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 11
    }

    // Os hosts do ssh e a rede local.
    Item {
        width: parent.width
        height: Math.min(132, conectar.conhecidos.length * 24)
        visible: conectar.conhecidos.length > 0

        ListView {
            id: lista
            anchors.fill: parent
            anchors.rightMargin: 10
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: conectar.conhecidos
            delegate: Item {
                id: linha
                required property var modelData
                readonly property bool eTitulo: modelData.titulo !== undefined
                width: lista.width
                height: 24

                Rotulo {
                    visible: linha.eTitulo
                    anchors.bottom: parent.bottom
                    anchors.bottomMargin: 4
                    text: linha.eTitulo ? linha.modelData.titulo : ""
                }
                Rectangle {
                    anchors.fill: parent
                    visible: !linha.eTitulo && toque.containsMouse
                    color: Theme.wash
                }
                Text {
                    visible: !linha.eTitulo
                    x: 6
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width * 0.4
                    elide: Text.ElideRight
                    text: linha.eTitulo ? "" : linha.modelData.nome
                    color: toque.containsMouse ? Theme.copper : Theme.text
                    font.family: Theme.sans
                    font.pixelSize: 12
                }
                Text {
                    visible: !linha.eTitulo
                    x: parent.width * 0.4 + 12
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width * 0.6 - 18
                    elide: Text.ElideMiddle
                    text: linha.eTitulo ? "" : linha.modelData.detalhe || ""
                    color: Theme.textDim
                    font.family: Theme.mono
                    font.pixelSize: 11
                }
                MouseArea {
                    id: toque
                    anchors.fill: parent
                    enabled: !linha.eTitulo && !conectar.conectando && !!linha.modelData.uri
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        conectar.erro = "";
                        conectar.pedido(linha.modelData.uri);
                    }
                }
            }
        }
        Rolagem {
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            alvo: lista
        }
    }

    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: conectar.erro !== ""
        text: conectar.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    // O que o gio disse, para a pessoa ler (ele pode falar outra língua).
    Text {
        width: parent.width
        wrapMode: Text.WrapAnywhere
        maximumLineCount: 5
        elide: Text.ElideRight
        visible: conectar.saida !== "" && conectar.erro !== ""
        text: conectar.saida
        color: Theme.textDim
        font.family: Theme.mono
        font.pixelSize: 11
    }
    Row {
        spacing: 8
        Botao {
            texto: conectar.conectando ? "conectando…" : "conectar"
            forte: true
            ligado: !conectar.conectando
            onAcionado: conectar.confirmar()
        }
        Botao {
            texto: conectar.conectando ? "cancelar a conexão" : "fechar"
            onAcionado: {
                if (conectar.conectando)
                    conectar.cancelado();
                else
                    conectar.fechado();
            }
        }
    }
    Rotulo { text: "enter conecta  ·  esc fecha  ·  a senha, se houver, é pedida depois e guardada só pelo chaveiro do sistema" }
}
