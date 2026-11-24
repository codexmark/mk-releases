import QtQuick
import QtQuick.Layouts
import Quickshell
import "SettingsSchema.js" as Schema

// O fim da página Energia dos Ajustes, abaixo de "Ao dormir": os avisos de
// quando suspender não serve nesta máquina. O que ela tem vem do
// scripts/mare-sleep, por Settings.caps.sono. Daqui nada muda a escolha da
// pessoa: o aviso só diz a saída.
ColumnLayout {
    id: block
    spacing: 0

    readonly property var sono: Settings.caps && Settings.caps.sono ? Settings.caps.sono : null
    readonly property string choice: Settings.get("power.sleep")
    readonly property var risky: sono !== null && sono.risky ? sono.risky : []

    component Line: Text {
        Layout.fillWidth: true
        Layout.topMargin: 14
        wrapMode: Text.WordWrap
        color: Theme.textSoft
        font.family: Theme.sans
        font.pixelSize: 13
        lineHeight: 1.3
    }

    // A placa que não acorda: só com suspender escolhido.
    Line {
        visible: block.risky.length > 0 && block.choice === "suspender"
        color: Theme.copper
        text: "Esta placa de vídeo (" + block.risky.join(", ") + ") é conhecida por não voltar do suspender: a tela fica preta e só reiniciando.  Escolha não dormir."
    }

    // Uma escolha antiga que o Mk não tem mais (hibernar): nada fica marcado
    // acima, e o computador não dorme até a pessoa escolher.
    Line {
        visible: Schema.orphan(Settings.values, "power.sleep")
        color: Theme.copper
        text: "A escolha que estava salva aqui (" + block.choice + ") não existe mais no Mk. Enquanto você não escolher, o computador não dorme."
    }

    // Suspender proibido no sistema (sleep.conf): o Mk não suspende.
    Line {
        visible: block.sono !== null && block.sono.canSuspend === false && block.choice === "suspender"
        color: Theme.copper
        text: "O suspender está desligado neste sistema (sleep.conf): com esta escolha o computador não dorme."
    }
}
