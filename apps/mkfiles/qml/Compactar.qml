import QtQuick
import "Arquivos.js" as Arquivos

// A folha de compactar a seleção: o nome do arquivo, o formato (só os que o
// motor diz que consegue fazer; os outros aparecem apagados, com o pacote que
// falta), o nível e, em zip e 7z, uma senha se a pessoa quiser.
Folha {
    id: compactar

    // [{ formato, disponivel, falta }], do pedido `formatos`.
    property var formatos: []
    property string formato: "zip"
    property string nivel: ""
    property string inicial: ""
    property string erro: ""
    property int quantos: 0
    readonly property bool comSenha: formato === "zip" || formato === "7z"
    readonly property var faltam: formatos.filter(f => f.disponivel !== true && f.falta)
    signal confirmado(string nome, string formato, var nivel, var segredo)

    function abrir() {
        const primeiro = formatos.filter(f => f.disponivel === true)[0];
        formato = primeiro ? primeiro.formato : "zip";
        nivel = "";
        erro = "";
        segredo.limpar();
        campo.text = Arquivos.comFormato(inicial, formato);
        foco.restart();
    }
    function escolher(f) {
        formato = f;
        campo.text = Arquivos.comFormato(campo.text.trim(), f);
        if (!comSenha)
            segredo.limpar();
    }
    function confirmar() {
        const nome = campo.text.trim();
        const e = Arquivos.erroDoNome(nome);
        if (e !== "") {
            erro = e;
            return;
        }
        const s = comSenha && segredo.text !== "" ? segredo.text : null;
        segredo.limpar();
        confirmado(nome, formato, nivel === "" ? null : Number(nivel), s);
    }
    onVisibleChanged: {
        if (!visible)
            segredo.limpar();
    }

    rotulo: "compactar"
    titulo: "Compactar " + Arquivos.plural(quantos, "item", "itens")
    largura: 500

    Timer {
        id: foco
        interval: 1
        onTriggered: {
            campo.focar();
            campo.selecionar(Arquivos.semExtensao(campo.text, false));
        }
    }
    Campo {
        id: campo
        width: parent.width
        dica: "nome do arquivo"
        onAceito: compactar.confirmar()
    }
    Rotulo { text: "formato" }
    Flow {
        width: parent.width
        spacing: 6
        Repeater {
            model: compactar.formatos
            Botao {
                required property var modelData
                texto: modelData.formato
                forte: compactar.formato === modelData.formato
                ligado: modelData.disponivel === true
                onAcionado: compactar.escolher(modelData.formato)
            }
        }
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: compactar.faltam.length > 0
        text: compactar.faltam.map(f => f.formato + " precisa do pacote " + f.falta).join("  ·  ")
        color: Theme.textFaint
        font.family: Theme.sans
        font.pixelSize: 11
    }
    Rotulo { text: "nível" }
    Segmento {
        opcoes: [{ v: "1", texto: "rápido" }, { v: "", texto: "normal" }, { v: "9", texto: "menor" }]
        valor: compactar.nivel
        onEscolhido: v => compactar.nivel = v
    }
    Rotulo {
        visible: compactar.comSenha
        text: "senha (opcional)"
    }
    Senha {
        id: segredo
        visible: compactar.comSenha
        width: parent.width
        dica: "sem senha"
        onAceito: compactar.confirmar()
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: compactar.erro !== ""
        text: compactar.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 8
        Botao {
            texto: "compactar"
            forte: true
            ligado: compactar.formatos.some(f => f.formato === compactar.formato && f.disponivel === true)
            onAcionado: compactar.confirmar()
        }
        Botao {
            texto: "cancelar"
            onAcionado: compactar.fechado()
        }
    }
    Rotulo { text: "enter confirma  ·  esc cancela" }
}
