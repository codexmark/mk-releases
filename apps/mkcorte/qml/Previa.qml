import QtQuick
import QtMultimedia
import "Corte.js" as Corte

// A pré-visualização: toca a edição, não o arquivo. Um tocador só, que pula o
// que não está em clipe e troca de vídeo entre um clipe e outro (quem decide
// é Corte.avancar). Quem manda no cursor é o Editor: aqui só se avisa por
// onde a reprodução anda (andou).
Item {
    id: previa

    property var ed: Corte.vazia()
    property bool tocando: false
    // O clipe que está no tocador e o arquivo carregado nele.
    property int clipe: -1
    property string arquivo: ""
    // Para onde ir (segundos do vídeo) assim que o arquivo carregar; -1 se nada.
    property real alvo: -1
    // Depois de um salto no mesmo arquivo, a posição antiga ainda chega por um
    // instante: até a nova chegar (em ms), o que vem é ignorado.
    property real esperando: -1
    readonly property bool temImagem: arquivo !== ""
    // Com as respostas de exemplo (MKCORTE_EXEMPLO) não há arquivo para tocar.
    property bool semVideo: false

    signal andou(real t)
    signal falhou(string frase)

    function arquivoDoClipe(i) {
        return previa.ed.fontes[previa.ed.clipes[i].fonte].arquivo;
    }

    // Põe o tocador no instante t da edição (tocando ou parado).
    function posiciona(t) {
        const onde = Corte.localizar(previa.ed, t);
        if (!onde) {
            previa.tocando = false;
            previa.clipe = -1;
            previa.arquivo = "";
            previa.alvo = -1;
            player.stop();
            player.source = "";
            return;
        }
        previa.clipe = onde.clipe;
        previa.mostra(previa.arquivoDoClipe(onde.clipe), onde.tempo);
    }

    // Mostra um instante de um vídeo (é também o que o arrasto de uma alça
    // usa para mostrar onde o corte vai cair).
    function mostra(caminho, tempo) {
        if (previa.semVideo)
            return;
        if (caminho !== previa.arquivo) {
            previa.arquivo = caminho;
            previa.alvo = tempo;
            previa.esperando = -1;
            player.source = Corte.urlDeCaminho(caminho);
            if (previa.tocando)
                player.play();
            else
                player.pause();
            return;
        }
        if (previa.alvo >= 0) {
            // Ainda carregando: só troca o destino.
            previa.alvo = tempo;
            return;
        }
        previa.salta(tempo);
    }

    function salta(tempo) {
        previa.esperando = Math.round(tempo * 1000);
        desiste.restart();
        player.position = previa.esperando;
    }

    function alterna() {
        if (previa.ed.clipes.length === 0 || previa.semVideo)
            return;
        if (previa.tocando) {
            previa.tocando = false;
            player.pause();
            return;
        }
        previa.tocando = true;
        player.play();
    }

    function pausa() {
        if (previa.tocando) {
            previa.tocando = false;
            player.pause();
        }
    }

    // O tocador andou: segue no clipe, salta para o seguinte ou para no fim.
    function chegou() {
        if (!previa.tocando || previa.alvo >= 0 || previa.clipe < 0)
            return;
        if (previa.esperando >= 0) {
            if (Math.abs(player.position - previa.esperando) > 400)
                return;
            previa.esperando = -1;
        }
        previa.segue(Corte.avancar(previa.ed, previa.clipe, player.position / 1000));
    }

    function segue(r) {
        if (r.fim) {
            previa.tocando = false;
            player.pause();
            previa.andou(Corte.duracao(previa.ed));
            return;
        }
        previa.clipe = r.clipe;
        if (r.salta) {
            previa.mostra(previa.arquivoDoClipe(r.clipe), r.tempo);
            previa.andou(Corte.inicioDoClipe(previa.ed, r.clipe));
            return;
        }
        previa.andou(Corte.tempoNaEdicao(previa.ed, r.clipe, r.tempo));
    }

    MediaPlayer {
        id: player
        videoOutput: tela
        audioOutput: AudioOutput {}
        onPositionChanged: previa.chegou()
        onMediaStatusChanged: {
            if ((mediaStatus === MediaPlayer.LoadedMedia || mediaStatus === MediaPlayer.BufferedMedia) && previa.alvo >= 0) {
                const a = previa.alvo;
                previa.alvo = -1;
                previa.salta(a);
                if (previa.tocando)
                    player.play();
                else
                    player.pause();
            } else if (mediaStatus === MediaPlayer.EndOfMedia && previa.tocando && previa.clipe >= 0) {
                // O arquivo acabou antes do fim marcado do clipe: é o fim dele.
                previa.esperando = -1;
                previa.segue(Corte.avancar(previa.ed, previa.clipe, previa.ed.clipes[previa.clipe].fim));
            }
        }
        onErrorOccurred: (erro, texto) => {
            previa.tocando = false;
            const nome = Corte.nome(previa.arquivo);
            // Esquece o arquivo: a próxima tentativa carrega de novo.
            previa.alvo = -1;
            previa.arquivo = "";
            previa.falhou("a pré-visualização não conseguiu abrir " + nome + (texto ? " (" + texto + ")" : ""));
        }
    }

    // Se a posição nova nunca chegar perto do pedido (um vídeo com poucos
    // quadros-chave), para de esperar.
    Timer {
        id: desiste
        interval: 1200
        onTriggered: previa.esperando = -1
    }

    VideoOutput {
        id: tela
        anchors.fill: parent
        // No fim do arquivo o Qt limpa a imagem; o último quadro tem de ficar.
        endOfStreamPolicy: VideoOutput.KeepLastFrame
        visible: previa.temImagem
    }
}
