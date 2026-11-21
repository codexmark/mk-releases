pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Hyprland

// As boas-vindas: a folha que aparece uma vez, no primeiro login de cada
// usuário (WelcomeWindow.qml desenha), e que dá para abrir de novo depois
// (busca: "boas-vindas"; Ajustes › Sobre; `ipc call mare welcome abrir`).
//
// "Já vi" é um arquivo, <raiz>/.bem-vindo, ao lado do ajustes.json: fora das
// pastas que a atualização troca, então reinstalar o shell não mostra a folha
// de novo. Apagar o arquivo faz ela voltar no próximo login.
Singleton {
    id: root

    readonly property string seenPath: Settings.dir + "/.bem-vindo"
    readonly property int count: 4

    // Quem já tinha o Mk antes das boas-vindas existirem não é "primeiro
    // login": a migração de usuário abaixo grava o .bem-vindo para essa gente.
    // Só que o mk-update reinicia o Maré ANTES de rodar as migrações, e logo
    // em seguida pede a senha do sudo no terminal: se a folha subisse nesse
    // meio-tempo, levava o teclado embora. Por isso a abertura automática só
    // vale quando a migração já consta como feita (uma instalação nova nasce
    // com todas marcadas; veja migrations/README.md). Lido uma vez só, ao
    // subir: quem está no meio da atualização não vê a folha nesta rodada, e
    // na próxima o .bem-vindo já existe.
    //
    // Sobra o novato que ainda não terminou o passeio e reinicia o Maré no
    // meio da sessão (o mesmo mk-update, `mk reiniciar-mare`): veja `late`.
    readonly property string migration: "20261123-bem-vindo.sh"
    readonly property string gatePath: (Quickshell.env("XDG_STATE_HOME") || (Quickshell.env("HOME") + "/.local/state"))
        + "/mk/migrations/" + migration

    // A folha está pedida. A janela ainda a esconde enquanto um painel da ilha
    // ou a tela de bloqueio estão na frente (veja WelcomeWindow.qml).
    property bool shown: false
    property int step: 0

    // Recolhida: quem abre um painel no meio do passeio (a folha convida a
    // experimentar o toque no Super e a busca) não a recebe de volta por cima
    // do que acabou de abrir. Fica só uma etiqueta no pé da tela, sem teclado,
    // e um clique nela traz a folha de volta no mesmo passo.
    property bool paused: false

    // checked: já se sabe se o arquivo existe. seen: ele existe.
    property bool checked: false
    property bool seen: false
    // gateChecked / fresh: o mesmo para a marca da migração.
    property bool gateChecked: false
    property bool fresh: false
    // A folha subiu sozinha (e não a pedido).
    property bool auto: false

    // O Maré subiu com a sessão já andando (foi reiniciado), e não no login.
    // A assinatura do Hyprland traz a hora em que ele subiu:
    // <commit>_<segundos>_<número>. Nesse caso a folha volta recolhida, sem
    // teclado: quem reiniciou costuma estar digitando num terminal.
    readonly property bool late: {
        const born = Number((Quickshell.env("HYPRLAND_INSTANCE_SIGNATURE") || "").split("_")[1]);
        return born > 0 && Date.now() / 1000 - born > 120;
    }

    // Um painel de verdade na frente (os passageiros, aviso e OSD, não contam).
    // O painel do projeto só existe num workspace ligado a um projeto (a mesma
    // conta do Tide.qml): fora dele o modo muda e nada aparece.
    readonly property bool panelUp: Mare.mode !== "rest" && !Mare.isTransient
        && !(Mare.mode === "project" && (!Projects.current || Projects.free))
    onPanelUpChanged: {
        if (panelUp)
            collapse();
    }

    // A folha também ensina atalhos que não são painéis (terminal, lazygit,
    // workspaces): quem os experimenta recolhe a folha do mesmo jeito, senão
    // ela ficava com o teclado por cima da janela que acabou de abrir.
    Connections {
        target: Hyprland
        function onRawEvent(event) {
            if (event.name === "openwindow" || event.name === "workspacev2")
                root.collapse();
        }
    }

    // Primeira vez: só depois de ler os ajustes (a velocidade do movimento vem
    // de lá) e nunca por cima do bloqueio. O Timer dá um instante para a ilha
    // assentar antes de a folha subir.
    readonly property bool firstRun: checked && gateChecked && fresh && !seen && Settings.loaded && !Lock.locked
    onFirstRunChanged: {
        if (firstRun)
            greet.restart();
        else
            greet.stop();
    }

    function open() {
        step = 0;
        paused = false;
        auto = false;
        shown = true;
    }

    // Só esconde; não conta como visto.
    function close() {
        shown = false;
        paused = false;
    }

    // Recolhe a folha para a etiqueta; não fecha nem conta como visto.
    function collapse() {
        if (shown)
            paused = true;
    }

    // Traz de volta a folha recolhida, no passo em que estava.
    function resume() {
        if (shown)
            paused = false;
    }

    // Esconde e marca como visto (o fim do passeio, o "pular", o Esc).
    function finish() {
        shown = false;
        paused = false;
        if (seen)
            return;
        seen = true;
        mark.setText("visto em " + Qt.formatDateTime(new Date(), "yyyy-MM-dd HH:mm") + "\n");
    }

    function next() {
        if (step >= count - 1)
            finish();
        else
            step++;
    }

    function back() {
        if (step > 0)
            step--;
    }

    Timer {
        id: greet
        interval: 900
        onTriggered: {
            if (!root.firstRun || root.shown)
                return;
            // Com um painel aberto, espera: a folha não sobe por baixo dele
            // para tomar o teclado quando ele fechar.
            if (root.panelUp) {
                greet.restart();
                return;
            }
            // Sem passar por "aberta" quando volta recolhida: a folha não
            // chega a pedir o teclado.
            root.step = 0;
            root.paused = root.late;
            root.auto = true;
            root.shown = true;
        }
    }

    FileView {
        id: mark
        path: root.seenPath
        printErrors: false
        // O arquivo pode chegar com o Maré já no ar (a migração, outra sessão
        // do mesmo usuário): vale como visto na hora.
        watchChanges: true
        onFileChanged: reload()
        onLoaded: {
            root.seen = true;
            root.checked = true;
            // Subiu sozinha e o "já vi" apareceu por fora: recolhe.
            if (root.shown && root.auto)
                root.close();
        }
        onLoadFailed: root.checked = true
    }

    FileView {
        path: root.gatePath
        printErrors: false
        onLoaded: {
            root.fresh = true;
            root.gateChecked = true;
        }
        onLoadFailed: root.gateChecked = true
    }
}
