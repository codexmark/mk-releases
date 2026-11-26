import QtQuick
import "Remoto.js" as Remoto

// O que o servidor (pelo gio) pergunta ao conectar: usuário, senha, domínio
// ou uma confirmação. A resposta vai ao motor, que a escreve na entrada do
// gio; a senha mora só aqui enquanto a folha está aberta, e some ao fechar.
// "Lembrar" não é do MkFiles: é do chaveiro do sistema, que o gvfs usa sozinho.
Folha {
    id: remota

    // { pedido, pergunta, tipo, texto, padrao, de_novo }, do evento `pergunta-remota`.
    property var dados: null
    readonly property var forma: Remoto.pergunta(dados)
    signal respondido(var resposta)

    function abrir() {
        segredo.limpar();
        claro.text = dados && dados.padrao && !forma.oculto ? dados.padrao : "";
        foco.restart();
    }
    function confirmar() {
        if (forma.simNao) {
            respondido("yes");
            return;
        }
        const resposta = forma.oculto ? segredo.text : claro.text;
        segredo.limpar();
        claro.text = "";
        respondido(resposta);
    }
    // Ao fechar, o que foi digitado some.
    onVisibleChanged: {
        if (!visible) {
            segredo.limpar();
            claro.text = "";
        }
    }

    rotulo: "rede"
    titulo: forma.titulo
    largura: 480

    Timer {
        id: foco
        interval: 1
        onTriggered: {
            if (remota.forma.oculto)
                segredo.focar();
            else if (!remota.forma.simNao)
                claro.focar();
        }
    }
    // O texto do gio, como veio.
    Text {
        width: parent.width
        wrapMode: Text.WrapAnywhere
        maximumLineCount: 8
        elide: Text.ElideRight
        text: remota.dados ? remota.dados.texto || "" : ""
        color: Theme.textSoft
        font.family: Theme.mono
        font.pixelSize: 11
    }
    Senha {
        id: segredo
        width: parent.width
        visible: remota.forma.oculto
        dica: remota.forma.dica
        onAceito: remota.confirmar()
    }
    Campo {
        id: claro
        width: parent.width
        visible: !remota.forma.oculto && !remota.forma.simNao
        dica: remota.forma.dica
        onAceito: remota.confirmar()
    }
    Row {
        spacing: 8
        Botao {
            texto: remota.forma.simNao ? "sim, continuar" : "continuar"
            forte: true
            onAcionado: remota.confirmar()
        }
        Botao {
            visible: remota.forma.simNao
            texto: "não"
            onAcionado: remota.respondido("no")
        }
        Botao {
            texto: "cancelar a conexão"
            onAcionado: remota.respondido(null)
        }
    }
    Rotulo { text: remota.forma.simNao ? "enter confirma  ·  esc cancela a conexão" : "enter continua  ·  esc cancela a conexão" }
}
