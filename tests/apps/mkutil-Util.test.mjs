// apps/mkutil/qml/Util.js: a lógica das janelas do MkUtil (MkInstalar e
// MkComprimir) que não depende do QML: escrever tamanhos e tempos, montar os
// argumentos do motor, estimar o tamanho antes de comprimir, a conta do
// "caber em N MB" e a fila.
import { readFileSync } from "node:fs";
import { t, eq, igual, assert } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const U = load(REPO + "/apps/mkutil/qml/Util.js");
// Os mesmos casos que o teste em Go do motor usa (a conta é do contrato).
const vetores = JSON.parse(readFileSync(REPO + "/apps/mkutil/src/internal/midia/testdata/caber.json", "utf8"));

const video = (extra) => Object.assign({
    arquivo: "/v/férias.mp4", tipo: "video", tamanho_bytes: 48000000, duracao_s: 62.4, largura: 1920, altura: 1080,
    fps: 30, codec: "h264", taxa_bps: 6000000, audio: { codec: "aac", taxa_bps: 128000, canais: 2 }
}, extra || {});
const foto = (extra) => Object.assign({
    arquivo: "/f/foto.png", tipo: "imagem", tamanho_bytes: 9000000, duracao_s: null, largura: 4000, altura: 3000,
    fps: null, codec: "png", taxa_bps: null, audio: null
}, extra || {});
const op = (extra) => Object.assign(U.opcoesPadrao(), extra || {});

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["valido", "num", "tamanho", "taxa", "tempo", "resta", "resolucao", "nome", "pasta", "caminhoDeUrl",
        "lerArquivos", "novos", "muda", "indice", "rotuloDoTipo", "arquivoDeExemplo", "argsInstalar", "botaoInstalar",
        "resumoDaFicha", "faseDaInstalacao", "opcoesPadrao", "par", "dimensoes", "porLargura", "caber", "estimar",
        "textoDaEstimativa", "argsComprimir", "resumoDaMidia", "textoDoProgresso", "textoDoResultado", "proximo",
        "prontos", "total", "tem"];
    const funcoes = Object.keys(U).filter((k) => typeof U[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("valido: só número de verdade, finito e não negativo", () => {
    for (const v of [0, 1, 0.5, 1e12])
        eq(U.valido(v), true, String(v));
    for (const v of [null, undefined, NaN, Infinity, -1, "3", {}, []])
        eq(U.valido(v), false, String(v));
});

t("num: uma casa só quando o número é pequeno, com vírgula", () => {
    eq(U.num(5.94), "5,9");
    eq(U.num(0.04), "0,0");
    eq(U.num(12.5), "13");
    eq(U.num(9.96), "10");
});

t("tamanho: de bytes para a unidade que cabe, de 1000 em 1000", () => {
    eq(U.tamanho(0), "0 B");
    eq(U.tamanho(999), "999 B");
    eq(U.tamanho(1000), "1,0 KB");
    eq(U.tamanho(839000), "839 KB");
    eq(U.tamanho(7600000), "7,6 MB");
    eq(U.tamanho(48000000), "48 MB");
    eq(U.tamanho(5400000000), "5,4 GB");
    eq(U.tamanho(3e12), "3,0 TB");
    for (const v of [null, undefined, NaN, -1, "9"])
        eq(U.tamanho(v), "—");
});

t("taxa: bits por segundo", () => {
    eq(U.taxa(0), "0 kb/s");
    eq(U.taxa(898871), "899 kb/s");
    eq(U.taxa(6153846), "6,2 Mb/s");
    eq(U.taxa(25000000), "25 Mb/s");
    eq(U.taxa(null), "—");
});

t("tempo: segundos como um relógio", () => {
    eq(U.tempo(0), "0:00");
    eq(U.tempo(7), "0:07");
    eq(U.tempo(62.4), "1:02");
    eq(U.tempo(59.6), "1:00");
    eq(U.tempo(3723), "1:02:03");
    eq(U.tempo(null), "—");
});

t("resta: quanto falta, em palavras", () => {
    eq(U.resta(0), "falta 1 s");
    eq(U.resta(1), "falta 1 s");
    eq(U.resta(40.2), "faltam 41 s");
    eq(U.resta(89), "faltam 89 s");
    eq(U.resta(200), "faltam 3 min");
    eq(U.resta(7200), "faltam 2,0 h");
    eq(U.resta(null), "");
    eq(U.resta(-3), "");
});

t("resolucao: largura × altura, ou um traço", () => {
    eq(U.resolucao(1920, 1080), "1920 × 1080");
    eq(U.resolucao(0, 1080), "—");
    eq(U.resolucao(null, null), "—");
});

t("nome e pasta: as duas metades de um caminho", () => {
    eq(U.nome("/home/ana/Vídeos/minhas férias.mp4"), "minhas férias.mp4");
    eq(U.pasta("/home/ana/Vídeos/minhas férias.mp4"), "/home/ana/Vídeos");
    eq(U.nome("solto.mp4"), "solto.mp4");
    eq(U.pasta("solto.mp4"), "");
    eq(U.pasta("/raiz.mp4"), "/");
    eq(U.nome(null), "");
    eq(U.pasta(undefined), "");
});

t("caminhoDeUrl: a URL que o seletor e o arrastar entregam vira caminho", () => {
    eq(U.caminhoDeUrl("file:///home/ana/a%20b%20%C3%A9%20%2523.mp4"), "/home/ana/a b é %23.mp4");
    eq(U.caminhoDeUrl("file://localhost/tmp/x.zip"), "/tmp/x.zip");
    eq(U.caminhoDeUrl("/já/é/caminho.mp4"), "/já/é/caminho.mp4");
    eq(U.caminhoDeUrl({ toString: () => "file:///tmp/y.AppImage" }), "/tmp/y.AppImage");
    // O que não é arquivo local não entra.
    for (const u of ["https://exemplo.org/x.AppImage", "file://outra-maquina/x", "", null, undefined, "file:///tmp/%E0%A4%A"])
        eq(U.caminhoDeUrl(u), "", String(u));
});

t("lerArquivos: a lista que o lançador passa; lixo vira lista vazia", () => {
    igual(U.lerArquivos('["/a/b.mp4","/c d/é.png"]'), ["/a/b.mp4", "/c d/é.png"]);
    igual(U.lerArquivos(""), []);
    igual(U.lerArquivos(undefined), []);
    igual(U.lerArquivos("isto não é json"), []);
    igual(U.lerArquivos('{"a":1}'), []);
    igual(U.lerArquivos('["relativo.mp4", 3, null, "/abs.mp4"]'), ["/abs.mp4"]);
});

t("novos: só o que ainda não está na lista, sem repetir", () => {
    const itens = [{ arquivo: "/a" }, { arquivo: "/b" }];
    igual(U.novos(itens, ["/b", "/c", "/c", "", null, "/a", "/d"]), ["/c", "/d"]);
    igual(U.novos(null, ["/x"]), ["/x"]);
    igual(U.novos(itens, null), []);
});

t("muda e indice: a lista nova tem o item trocado e a velha fica como era", () => {
    const itens = [{ arquivo: "/a", estado: "lendo" }, { arquivo: "/b", estado: "lendo" }];
    const n = U.muda(itens, 1, { estado: "pronto", pct: 3 });
    igual(n[1], { arquivo: "/b", estado: "pronto", pct: 3 });
    eq(itens[1].estado, "lendo");
    eq(n === itens, false);
    eq(n[0], itens[0]);
    igual(U.muda(itens, 5, { estado: "x" }), itens);
    igual(U.muda(itens, -1, { estado: "x" }), itens);
    igual(U.muda(null, 0, {}), []);
    eq(U.indice(itens, "/b"), 1);
    eq(U.indice(itens, "/z"), -1);
    eq(U.indice(null, "/z"), -1);
});

t("rotuloDoTipo: o nome de cada tipo, como a pessoa lê", () => {
    eq(U.rotuloDoTipo("appimage"), "AppImage");
    eq(U.rotuloDoTipo("pacote"), "pacote do Arch");
    eq(U.rotuloDoTipo("compactado"), "arquivo compactado");
    eq(U.rotuloDoTipo("video"), "vídeo");
    eq(U.rotuloDoTipo("imagem"), "imagem");
    eq(U.rotuloDoTipo("banana"), "—");
});

t("arquivoDeExemplo: de que arquivo gravado sai a resposta de cada pedido", () => {
    eq(U.arquivoDeExemplo(["inspecionar", "/x/Krita-5.AppImage"]), "inspecionar-appimage.json");
    eq(U.arquivoDeExemplo(["inspecionar", "/x/htop-3.pkg.tar.zst"]), "inspecionar-pacote.json");
    eq(U.arquivoDeExemplo(["inspecionar", "/x/firefox.tar.xz"]), "inspecionar-compactado.json");
    eq(U.arquivoDeExemplo(["inspecionar", "/x/jogo.ZIP"]), "inspecionar-compactado.json");
    eq(U.arquivoDeExemplo(["midia", "/x/a.MP4"]), "midia-video.json");
    eq(U.arquivoDeExemplo(["midia", "/x/a.jpeg"]), "midia-imagem.json");
    eq(U.arquivoDeExemplo(["midia", "/x/a.PNG"]), "midia-imagem.json");
    eq(U.arquivoDeExemplo(["instalar", "/x/a.AppImage", "--substituir"]), "instalar.json");
    eq(U.arquivoDeExemplo(["comprimir", "/x/a.mp4", "--predefinicao", "menor"]), "comprimir.json");
    for (const sub of ["instalados", "desinstalar", "codificadores", "descartar"])
        eq(U.arquivoDeExemplo([sub]), sub + ".json");
});

t("argsInstalar: o que a janela manda ao motor para cada ficha", () => {
    const ai = { arquivo: "/x/-a b.AppImage", tipo: "appimage", ja_instalado: false };
    igual(U.argsInstalar(ai, { atalho: true }), ["instalar", "/x/-a b.AppImage"]);
    igual(U.argsInstalar(ai), ["instalar", "/x/-a b.AppImage"]);
    igual(U.argsInstalar(ai, { atalho: false }), ["instalar", "/x/-a b.AppImage", "--sem-atalho"]);
    igual(U.argsInstalar(Object.assign({}, ai, { ja_instalado: true }), {}), ["instalar", "/x/-a b.AppImage", "--substituir"]);
    const tgz = { arquivo: "/x/f.tar.gz", tipo: "compactado", ja_instalado: false };
    igual(U.argsInstalar(tgz, { atalho: true, executavel: "bin/f" }), ["instalar", "/x/f.tar.gz", "--executavel", "bin/f"]);
    igual(U.argsInstalar(tgz, { atalho: false, executavel: "bin/f" }), ["instalar", "/x/f.tar.gz", "--sem-atalho"]);
    // Pacote: nada de atalho nem de substituir (quem cuida é o pacman).
    igual(U.argsInstalar({ arquivo: "/x/h.pkg.tar.zst", tipo: "pacote", ja_instalado: false }, { atalho: false, executavel: "x" }),
        ["instalar", "/x/h.pkg.tar.zst"]);
    // O --executavel não vai para um AppImage.
    igual(U.argsInstalar(ai, { executavel: "x" }), ["instalar", "/x/-a b.AppImage"]);
});

t("botaoInstalar: o botão diz o que vai acontecer", () => {
    eq(U.botaoInstalar(null), "instalar");
    eq(U.botaoInstalar({ tipo: "appimage", ja_instalado: false }), "instalar");
    eq(U.botaoInstalar({ tipo: "compactado", ja_instalado: true }), "substituir");
    eq(U.botaoInstalar({ tipo: "pacote", pacote: { instalado: null } }), "instalar (pede a senha)");
    eq(U.botaoInstalar({ tipo: "pacote", pacote: { instalado: "1.0-1" } }), "atualizar (pede a senha)");
});

t("resumoDaFicha: tipo, versão e o tamanho que importa", () => {
    eq(U.resumoDaFicha(null), "");
    eq(U.resumoDaFicha({ tipo: "appimage", versao: "5.2.2", tamanho_bytes: 412000000 }), "AppImage  ·  5.2.2  ·  412 MB");
    eq(U.resumoDaFicha({ tipo: "appimage", versao: null, tamanho_bytes: 412000000 }), "AppImage  ·  412 MB");
    // Compactado: o tamanho depois de extraído; pacote: o instalado.
    eq(U.resumoDaFicha({ tipo: "compactado", versao: "128.0", tamanho_bytes: 80000000, compactado: { tamanho_total_bytes: 250000000 } }),
        "arquivo compactado  ·  128.0  ·  250 MB");
    eq(U.resumoDaFicha({ tipo: "pacote", versao: "3.3.0-3", tamanho_bytes: 150000, pacote: { tamanho_instalado_bytes: 412000 } }),
        "pacote do Arch  ·  3.3.0-3  ·  412 KB");
    eq(U.resumoDaFicha({ tipo: "pacote", versao: "1", tamanho_bytes: 150000, pacote: { tamanho_instalado_bytes: null } }),
        "pacote do Arch  ·  1  ·  150 KB");
});

t("faseDaInstalacao: o que mostrar enquanto instala", () => {
    eq(U.faseDaInstalacao({ fase: "copiando", pct: 40.4 }), "copiando 40%");
    eq(U.faseDaInstalacao({ fase: "extraindo", pct: 62.5, entrada: "a/b" }), "extraindo 63%");
    eq(U.faseDaInstalacao({ fase: "extraindo" }), "extraindo");
    eq(U.faseDaInstalacao({ fase: "pacman", linha: "(1/1) instalando htop" }), "(1/1) instalando htop");
    eq(U.faseDaInstalacao({ fase: "atalho" }), "criando o atalho");
    eq(U.faseDaInstalacao({ fase: "fim" }), "");
    eq(U.faseDaInstalacao(null), "");
});

t("opcoesPadrao: o H.264 por padrão, sem aceleração, e cada chamada dá um objeto novo", () => {
    const a = U.opcoesPadrao(), b = U.opcoesPadrao();
    eq(a.codec, "h264");
    eq(a.acelerar, "");
    eq(a.predefinicao, "equilibrio");
    eq(a.semAudio, false);
    a.codec = "av1";
    eq(b.codec, "h264");
    igual(U.predefinicoesDeVideo.map((p) => p.v), ["menor", "equilibrio", "alta", "caber", "resolucao", "sem-audio", "gif", "webp"]);
});

t("par: o número par mais perto", () => {
    eq(U.par(853.33), 854);
    eq(U.par(852.98), 852);
    eq(U.par(3), 4);
    eq(U.par(2.9), 2);
});

t("dimensoes: a mesma conta do motor, nos mesmos casos", () => {
    assert.ok(vetores.dimensoes.length >= 8);
    for (const c of vetores.dimensoes)
        igual(U.dimensoes(c.largura, c.altura, c.lado), { largura: c.saida_largura, altura: c.saida_altura }, JSON.stringify(c));
    igual(U.dimensoes(1920, 1080, undefined), { largura: 1920, altura: 1080 });
});

t("porLargura: limita a largura sem ampliar, com dimensões pares", () => {
    igual(U.porLargura(1920, 1080, 480), { largura: 480, altura: 270 });
    igual(U.porLargura(1920, 1080, 4000), { largura: 1920, altura: 1080 });
    igual(U.porLargura(1081, 1921, 480), { largura: 480, altura: 852 });
    igual(U.porLargura(1921, 1081, 0), { largura: 1920, altura: 1080 });
    igual(U.porLargura(1000, 3, 500), { largura: 500, altura: 2 });
});

t("caber: a taxa para caber em N MB sai da duração (os casos do contrato, os mesmos do motor)", () => {
    assert.ok(vetores.caber.length >= 10);
    for (const c of vetores.caber) {
        const r = U.caber(c.alvo_mb, c.duracao_s, c.audio, c.largura, c.altura, c.fps);
        if (!c.cabe)
            igual(r, { cabe: false }, JSON.stringify(c));
        else
            igual(r, { cabe: true, video_bps: c.video_bps, audio_bps: c.audio_bps, largura: c.saida_largura, altura: c.saida_altura },
                JSON.stringify(c));
    }
    // O caso de mandar um vídeo de um minuto por mensageiro, à mão:
    // 8 MB × 8.000.000 × 0,97 ÷ 62,4 s = 994.871 b/s; menos 96 kb/s de áudio.
    igual(U.caber(8, 62.4, true, 1920, 1080, 30), { cabe: true, video_bps: 898871, audio_bps: 96000, largura: 854, altura: 480 });
    for (const ruim of [[0, 10], [-1, 10], [8, 0], [NaN, 10], [8, null]])
        igual(U.caber(ruim[0], ruim[1], true, 1920, 1080, 30), { cabe: false });
});

t("caber: o resultado nunca passa do tamanho pedido", () => {
    for (const mb of [1, 8, 10, 25, 50, 100, 500])
        for (const dur of [5, 30, 62.4, 600, 3600]) {
            const r = U.caber(mb, dur, true, 1920, 1080, 30);
            if (r.cabe)
                assert.ok((r.video_bps + r.audio_bps) * dur / 8 <= mb * 1000000, mb + " MB em " + dur + " s");
        }
});

t("estimar: cada predefinição de vídeo, antes de rodar", () => {
    const v = video();
    // menor: 1280×720 a 30 q/s × 0,035 bit por pixel + 64 kb/s de áudio, por 62,4 s.
    igual(U.estimar(v, op({ predefinicao: "menor" })), { bytes: Math.round((1280 * 720 * 30 * 0.035 + 64000) * 62.4 / 8), encolhe: true, nota: "1280 × 720" });
    // equilíbrio: fica em 1080p, sem nota de resolução.
    igual(U.estimar(v, op({ predefinicao: "equilibrio" })), { bytes: Math.round((1920 * 1080 * 30 * 0.07 + 128000) * 62.4 / 8), encolhe: true, nota: "" });
    // alta num arquivo que já é enxuto: a taxa não passa da do original, e não deve encolher.
    const alta = U.estimar(v, op({ predefinicao: "alta" }));
    eq(alta.bytes, Math.round((6000000 + 160000) * 62.4 / 8));
    eq(alta.encolhe, false);
    eq(alta.nota, "não deve encolher");
    // H.265 e AV1 gastam menos para a mesma imagem.
    const h264 = U.estimar(v, op({ predefinicao: "equilibrio" })).bytes;
    const h265 = U.estimar(v, op({ predefinicao: "equilibrio", codec: "h265" })).bytes;
    const av1 = U.estimar(v, op({ predefinicao: "equilibrio", codec: "av1" })).bytes;
    assert.ok(av1 < h265 && h265 < h264);
    // Sem áudio, menos; resolução pedida, menos ainda.
    assert.ok(U.estimar(v, op({ predefinicao: "equilibrio", semAudio: true })).bytes < h264);
    const p480 = U.estimar(v, op({ predefinicao: "resolucao", lado: 480 }));
    eq(p480.nota, "854 × 480");
    assert.ok(p480.bytes < U.estimar(v, op({ predefinicao: "resolucao", lado: 1080 })).bytes);
    // A ordem das três: menor < equilíbrio < alta.
    assert.ok(U.estimar(v, op({ predefinicao: "menor" })).bytes < h264 && h264 < alta.bytes);
});

t("estimar: caber, tirar o áudio, GIF e WebP", () => {
    const v = video();
    const cabe = U.estimar(v, op({ predefinicao: "caber", alvoMb: 8 }));
    eq(cabe.bytes, Math.round(994871 * 62.4 / 8));
    assert.ok(cabe.bytes <= 8000000);
    eq(cabe.encolhe, true);
    eq(cabe.nota, "854 × 480, 899 kb/s");
    igual(U.estimar(v, op({ predefinicao: "caber", alvoMb: 500 })), { bytes: 48000000, encolhe: false, nota: "o original já cabe" });
    igual(U.estimar(video({ duracao_s: 7200 }), op({ predefinicao: "caber", alvoMb: 1 })),
        { bytes: null, encolhe: false, nota: "não cabe: longo demais para esse tamanho" });
    igual(U.estimar(v, op({ predefinicao: "caber", alvoMb: 0 })), { bytes: null, encolhe: false, nota: "diga o tamanho" });
    // Tirar o áudio: o tamanho menos a trilha (128 kb/s × 62,4 s).
    igual(U.estimar(v, op({ predefinicao: "sem-audio" })), { bytes: 48000000 - 128000 * 62.4 / 8, encolhe: false, nota: "não deve encolher" });
    igual(U.estimar(video({ audio: null }), op({ predefinicao: "sem-audio" })), { bytes: null, encolhe: false, nota: "já não tem áudio" });
    const gif = U.estimar(v, op({ predefinicao: "gif" }));
    eq(gif.bytes, Math.round(480 * 270 * 12 * 62.4 * 0.25));
    eq(gif.nota, "sem som");
    assert.ok(U.estimar(v, op({ predefinicao: "webp" })).bytes < gif.bytes);
});

t("estimar: imagem, pelo formato, a qualidade e a largura", () => {
    const f = foto();
    const jpeg80 = U.estimar(f, op({ formato: "jpeg", qualidade: 80 }));
    eq(jpeg80.bytes, Math.round(4000 * 3000 * (0.4 + 3.6 * 0.512) / 8));
    eq(jpeg80.encolhe, true);
    assert.ok(U.estimar(f, op({ formato: "jpeg", qualidade: 50 })).bytes < jpeg80.bytes);
    assert.ok(U.estimar(f, op({ formato: "webp", qualidade: 80 })).bytes < jpeg80.bytes);
    assert.ok(U.estimar(f, op({ formato: "avif", qualidade: 80 })).bytes < U.estimar(f, op({ formato: "webp", qualidade: 80 })).bytes);
    eq(U.estimar(f, op({ formato: "jpeg", qualidade: 80, largura: 2000 })).bytes, Math.round(2000 * 1500 * (0.4 + 3.6 * 0.512) / 8));
    // Uma largura maior que a da foto não amplia.
    eq(U.estimar(f, op({ formato: "jpeg", qualidade: 80, largura: 9000 })).bytes, jpeg80.bytes);
    // Uma foto que já é pequena: avisa que não deve encolher.
    const pequena = U.estimar(foto({ tamanho_bytes: 200000 }), op({ formato: "jpeg", qualidade: 95 }));
    eq(pequena.encolhe, false);
    eq(pequena.nota, "não deve encolher");
});

t("estimar: sem dados não inventa número", () => {
    igual(U.estimar(null, op()), { bytes: null, encolhe: false, nota: "" });
    igual(U.estimar(video({ tamanho_bytes: null }), op()), { bytes: null, encolhe: false, nota: "" });
    igual(U.estimar(video({ duracao_s: null }), op()), { bytes: null, encolhe: false, nota: "sem a duração não dá para estimar" });
    igual(U.estimar(video(), op({ predefinicao: "banana" })), { bytes: null, encolhe: false, nota: "" });
    // Sem fps e sem taxa conhecidos, estima com 30 q/s e sem teto.
    eq(U.estimar(video({ fps: null, taxa_bps: null }), op({ predefinicao: "menor" })).bytes,
        Math.round((1280 * 720 * 30 * 0.035 + 64000) * 62.4 / 8));
});

t("textoDaEstimativa: o que aparece ao lado de cada arquivo", () => {
    const v = video();
    eq(U.textoDaEstimativa(v, { bytes: 7600000, encolhe: true, nota: "" }), "≈ 7,6 MB (−84%)");
    eq(U.textoDaEstimativa(v, { bytes: 51000000, encolhe: false, nota: "não deve encolher" }), "≈ 51 MB: não deve encolher");
    eq(U.textoDaEstimativa(v, { bytes: null, encolhe: false, nota: "não cabe: longo demais para esse tamanho" }), "não cabe: longo demais para esse tamanho");
    eq(U.textoDaEstimativa(v, { bytes: null, encolhe: false, nota: "" }), "—");
    eq(U.textoDaEstimativa(v, null), "—");
});

t("argsComprimir: o que a janela manda ao motor", () => {
    const v = video({ arquivo: "/v/-i %d.mp4" });
    igual(U.argsComprimir(v, op()), ["comprimir", "/v/-i %d.mp4", "--predefinicao", "equilibrio"]);
    igual(U.argsComprimir(v, op({ predefinicao: "caber", alvoMb: 25 })), ["comprimir", "/v/-i %d.mp4", "--predefinicao", "caber", "--alvo-mb", "25"]);
    igual(U.argsComprimir(v, op({ predefinicao: "resolucao", lado: 480, codec: "h265", acelerar: "vaapi", semAudio: true })),
        ["comprimir", "/v/-i %d.mp4", "--predefinicao", "resolucao", "--lado", "480", "--codec", "h265", "--acelerar", "vaapi", "--sem-audio"]);
    // Sem pedir, nada de aceleração nem de codec: o motor usa o H.264 por software.
    for (const p of ["menor", "equilibrio", "alta"])
        igual(U.argsComprimir(v, op({ predefinicao: p })), ["comprimir", "/v/-i %d.mp4", "--predefinicao", p]);
    // GIF, WebP e tirar o áudio não levam codec nem aceleração.
    for (const p of ["gif", "webp", "sem-audio"])
        igual(U.argsComprimir(v, op({ predefinicao: p, codec: "av1", acelerar: "nvenc", semAudio: true })), ["comprimir", "/v/-i %d.mp4", "--predefinicao", p]);
    // Imagem: a predefinição de vídeo escolhida não importa.
    igual(U.argsComprimir(foto(), op({ predefinicao: "caber", formato: "webp", qualidade: 70.4, largura: 1600 })),
        ["comprimir", "/f/foto.png", "--predefinicao", "imagem", "--formato", "webp", "--qualidade", "70", "--largura", "1600"]);
    igual(U.argsComprimir(foto(), op({ qualidade: 0, largura: 0 })),
        ["comprimir", "/f/foto.png", "--predefinicao", "imagem", "--formato", "jpeg", "--qualidade", "80"]);
    igual(U.argsComprimir(foto(), op({ qualidade: 400 })).slice(-2), ["--qualidade", "100"]);
});

t("resumoDaMidia: resolução, duração, codec e tamanho", () => {
    eq(U.resumoDaMidia(video()), "1920 × 1080  ·  1:02  ·  h264  ·  48 MB");
    eq(U.resumoDaMidia(foto()), "4000 × 3000  ·  png  ·  9,0 MB");
    eq(U.resumoDaMidia(null), "");
});

t("textoDoProgresso: porcentagem, passada, velocidade e quanto falta", () => {
    eq(U.textoDoProgresso({ fase: "codificando", passada: 1, passadas: 2, pct: 18.4, velocidade: 3.1, resta_s: 41, tamanho_bytes: 0 }),
        "18%  ·  passada 1 de 2  ·  3,1x  ·  faltam 41 s");
    eq(U.textoDoProgresso({ fase: "codificando", passada: 1, passadas: 1, pct: 99.9, velocidade: 12.5, resta_s: 0, tamanho_bytes: 0 }),
        "99%  ·  13x  ·  falta 1 s");
    eq(U.textoDoProgresso({ fase: "codificando", passada: 1, passadas: 1, pct: 0, velocidade: null, resta_s: null, tamanho_bytes: 0 }), "0%");
    eq(U.textoDoProgresso({ fase: "plano" }), "");
    eq(U.textoDoProgresso(null), "");
});

t("textoDoResultado: quanto encolheu, ou que ficou maior", () => {
    eq(U.textoDoResultado({ antes_bytes: 48000000, depois_bytes: 7600000, economia_pct: 84.2, maior: false }), "48 MB → 7,6 MB (−84%)");
    eq(U.textoDoResultado({ antes_bytes: 48000000, depois_bytes: 51000000, economia_pct: -6.3, maior: true }), "48 MB → 51 MB: ficou maior que o original");
    eq(U.textoDoResultado({ antes_bytes: 1000, depois_bytes: 1000, economia_pct: 0, maior: false }), "1,0 KB → 1,0 KB (−0%)");
    eq(U.textoDoResultado(null), "");
});

t("a fila: um por vez, na ordem, e os totais", () => {
    const itens = [
        { arquivo: "/a", estado: "feito", midia: video(), fim: { antes_bytes: 100, depois_bytes: 40 } },
        { arquivo: "/b", estado: "codificando", midia: video() },
        { arquivo: "/c", estado: "fila", midia: video() },
        { arquivo: "/d", estado: "erro" },
        { arquivo: "/e", estado: "fila", midia: foto() },
        { arquivo: "/f", estado: "pronto", midia: video() },
        { arquivo: "/g", estado: "pronto", midia: foto() },
        { arquivo: "/h", estado: "lendo" }
    ];
    eq(U.proximo(itens), 2);
    eq(U.proximo(itens.slice(5)), -1);
    eq(U.proximo(null), -1);
    eq(U.prontos(itens), 2);
    eq(U.prontos([]), 0);
    eq(U.prontos(null), 0);
    eq(U.tem(itens, "video"), true);
    eq(U.tem(itens, "imagem"), true);
    eq(U.tem([itens[0], itens[3]], "imagem"), false);
    eq(U.tem(null, "video"), false);
    // O total estimado: só os prontos com estimativa.
    const o = op({ predefinicao: "menor" });
    igual(U.total(itens, o, false), {
        antes: 48000000 + 9000000,
        depois: U.estimar(video(), o).bytes + U.estimar(foto(), o).bytes,
        n: 2
    });
    // O total feito: só os terminados.
    igual(U.total(itens, o, true), { antes: 100, depois: 40, n: 1 });
    igual(U.total(null, o, false), { antes: 0, depois: 0, n: 0 });
    // O que não tem estimativa (não cabe) fica fora da soma.
    igual(U.total([{ estado: "pronto", midia: video({ duracao_s: 7200 }) }], op({ predefinicao: "caber", alvoMb: 1 }), false), { antes: 0, depois: 0, n: 0 });
});
