import QtQuick
import Quickshell

// Um ícone do tema, pelo nome (com um genérico de reserva), ou uma imagem
// pelo caminho (a miniatura, o ícone de um app que dá o arquivo).
Image {
    id: icone

    property string nome: ""
    property string reserva: "application-x-generic"
    property string arquivo: ""
    property int lado: 16

    width: lado
    height: lado
    sourceSize.width: lado
    sourceSize.height: lado
    fillMode: Image.PreserveAspectFit
    asynchronous: true
    cache: true
    smooth: true
    source: arquivo !== "" ? "file://" + arquivo
        : nome === "" ? ""
        : nome.charAt(0) === "/" ? "file://" + nome
        : Quickshell.iconPath(nome, reserva)
}
