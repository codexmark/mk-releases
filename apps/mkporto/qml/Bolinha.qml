import QtQuick

// O estado de um host, numa bolinha: cheia de cobre se a porta responde,
// apagada se não responde, um anel de cobre se só se chega por salto, um anel
// que pulsa enquanto a sonda não volta e um anel fino se ainda não foi sondado.
Rectangle {
    id: bolinha

    // "responde" | "nao" | "salto" | "sondando" | "desconhecido" (Porto.estado)
    property string estado: "desconhecido"

    implicitWidth: 9
    implicitHeight: 9
    radius: width / 2
    color: estado === "responde" ? Theme.copper : estado === "nao" ? Theme.textFaint : Qt.rgba(0, 0, 0, 0)
    border.width: estado === "responde" || estado === "nao" ? 0 : 1
    border.color: estado === "salto" ? Theme.copper : estado === "sondando" ? Theme.textSoft : Theme.line

    SequentialAnimation on opacity {
        running: bolinha.estado === "sondando" && bolinha.visible
        loops: Animation.Infinite
        NumberAnimation { to: 0.25; duration: 450 }
        NumberAnimation { to: 1; duration: 450 }
    }
    onEstadoChanged: if (estado !== "sondando") opacity = 1
}
