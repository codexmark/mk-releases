import QtQuick
import "Arquivos.js" as Arquivos

// A folha que pede um texto: o nome da pasta nova, do arquivo novo, do que
// vai ser colado, do favorito, ou o padrão de seleção ("*.jpg"). Com modelos,
// mostra também a lista deles para escolher um. Enter confirma.
Folha {
    id: nome

    property string frase: ""
    property string inicial: ""
    property string botao: "criar"
    property bool eNome: true
    property string erro: ""
    // O que dizer quando não é um nome e o campo está vazio.
    property string vazio: "escreva um padrão, como *.jpg"
    property var modelos: []
    property int modelo: -1
    signal confirmado(string texto, var modelo)

    function abrir() {
        campo.text = inicial;
        modelo = -1;
        erro = "";
        campo.focar();
        entradaTimer.restart();
    }
    function confirmar() {
        const texto = campo.text.trim();
        const e = eNome ? Arquivos.erroDoNome(texto) : (texto === "" ? vazio : "");
        if (e !== "") {
            erro = e;
            return;
        }
        confirmado(texto, modelo >= 0 ? modelos[modelo] : null);
    }

    rotulo: "nome"

    // O campo só existe depois de montado: a seleção do nome sem a extensão
    // vai num instante depois de abrir.
    Timer {
        id: entradaTimer
        interval: 1
        onTriggered: {
            campo.focar();
            campo.selecionar(nome.eNome ? Arquivos.semExtensao(campo.text, false) : campo.text.length);
        }
    }

    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: nome.frase !== ""
        text: nome.frase
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Flow {
        width: parent.width
        spacing: 6
        visible: nome.modelos.length > 0
        Repeater {
            model: nome.modelos
            Botao {
                required property var modelData
                required property int index
                texto: modelData.nome
                forte: nome.modelo === index
                onAcionado: {
                    nome.modelo = index;
                    campo.text = modelData.nome;
                    campo.focar();
                }
            }
        }
    }
    Campo {
        id: campo
        width: parent.width
        dica: nome.eNome ? "nome" : "*.jpg"
        onAceito: nome.confirmar()
    }
    Text {
        width: parent.width
        wrapMode: Text.WordWrap
        visible: nome.erro !== ""
        text: nome.erro
        color: Theme.copper
        font.family: Theme.sans
        font.pixelSize: 12
    }
    Row {
        spacing: 8
        Botao {
            texto: nome.botao
            forte: true
            onAcionado: nome.confirmar()
        }
        Botao {
            texto: "cancelar"
            onAcionado: nome.fechado()
        }
    }
    Rotulo { text: "enter confirma  ·  esc cancela" }
}
