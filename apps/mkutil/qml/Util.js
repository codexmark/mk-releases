.pragma library

// A lógica das janelas do MkUtil (MkInstalar e MkComprimir) que não depende do
// QML: escrever tamanhos, taxas e tempos do jeito que a pessoa lê, montar os
// argumentos do motor, estimar o tamanho de uma compressão antes de rodar e
// cuidar da fila. Os dados vêm do motor (../CONTRATO.md); o que ele não soube
// chega como null e aqui vira um traço, nunca zero.

function valido(v) {
    return typeof v === "number" && isFinite(v) && v >= 0;
}

// Uma casa decimal abaixo de 10, nenhuma dali em diante, com vírgula.
function num(v) {
    var s = v.toFixed(1);
    return Number(s) < 10 ? s.replace(".", ",") : String(Math.round(v));
}

// bytes -> "839 KB", "48 MB", "5,4 GB" (de 1000 em 1000, como os mensageiros contam).
function tamanho(bytes) {
    if (!valido(bytes))
        return "—";
    var unidades = ["B", "KB", "MB", "GB", "TB"], i = 0, v = bytes;
    while (v >= 1000 && i < unidades.length - 1) {
        v /= 1000;
        i++;
    }
    return (i === 0 ? String(Math.round(v)) : num(v)) + " " + unidades[i];
}

// bits por segundo -> "898 kb/s", "6,2 Mb/s".
function taxa(bps) {
    if (!valido(bps))
        return "—";
    if (bps < 1000000)
        return Math.round(bps / 1000) + " kb/s";
    return num(bps / 1000000) + " Mb/s";
}

// Segundos como um relógio: "0:07", "1:02", "1:02:03".
function tempo(s) {
    if (!valido(s))
        return "—";
    s = Math.round(s);
    var h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), seg = s % 60;
    var dois = function (n) { return n < 10 ? "0" + n : String(n); };
    return h > 0 ? h + ":" + dois(m) + ":" + dois(seg) : m + ":" + dois(seg);
}

// Quanto falta, em palavras: "falta 1 s", "faltam 41 s", "faltam 3 min".
function resta(s) {
    if (!valido(s))
        return "";
    s = Math.ceil(s);
    if (s <= 1)
        return "falta 1 s";
    if (s < 90)
        return "faltam " + s + " s";
    if (s < 5400)
        return "faltam " + Math.round(s / 60) + " min";
    return "faltam " + num(s / 3600) + " h";
}

function resolucao(largura, altura) {
    return valido(largura) && valido(altura) && largura > 0 && altura > 0 ? largura + " × " + altura : "—";
}

// O nome do arquivo e a pasta dele.
function nome(caminho) {
    var s = String(caminho || "");
    return s.slice(s.lastIndexOf("/") + 1);
}

function pasta(caminho) {
    var s = String(caminho || ""), i = s.lastIndexOf("/");
    return i <= 0 ? (i === 0 ? "/" : "") : s.slice(0, i);
}

// "file:///home/ana/a%20b.mp4" -> "/home/ana/a b.mp4". O que não é arquivo
// local (http, vazio) vira "".
function caminhoDeUrl(url) {
    var s = String(url === null || url === undefined ? "" : url);
    if (s.charAt(0) === "/")
        return s;
    var m = /^file:\/\/(localhost)?(\/.*)$/.exec(s);
    if (!m)
        return "";
    try {
        return decodeURIComponent(m[2]);
    } catch (e) {
        return "";
    }
}

// A lista de arquivos que o lançador passou (MKUTIL_ARQUIVOS, uma lista JSON).
function lerArquivos(texto) {
    var lista;
    try {
        lista = JSON.parse(texto || "[]");
    } catch (e) {
        return [];
    }
    if (!Array.isArray(lista))
        return [];
    return lista.filter(function (c) { return typeof c === "string" && c.charAt(0) === "/"; });
}

// Os caminhos novos que ainda não estão na lista, na ordem, sem repetir.
function novos(itens, caminhos) {
    var tem = {}, out = [], i;
    for (i = 0; i < (itens || []).length; i++)
        tem[itens[i].arquivo] = true;
    for (i = 0; i < (caminhos || []).length; i++) {
        var c = caminhos[i];
        if (typeof c === "string" && c !== "" && !tem[c]) {
            tem[c] = true;
            out.push(c);
        }
    }
    return out;
}

// Uma cópia da lista com os campos do item i trocados (a lista do QML só
// percebe a mudança se for outra lista).
function muda(itens, i, campos) {
    var n = (itens || []).slice();
    if (i < 0 || i >= n.length)
        return n;
    var novo = {}, k;
    for (k in n[i])
        novo[k] = n[i][k];
    for (k in campos)
        novo[k] = campos[k];
    n[i] = novo;
    return n;
}

// Onde está o arquivo na lista (-1 se não está).
function indice(itens, arquivo) {
    for (var i = 0; i < (itens || []).length; i++)
        if (itens[i].arquivo === arquivo)
            return i;
    return -1;
}

var rotulos = {
    appimage: "AppImage", pacote: "pacote do Arch", compactado: "arquivo compactado", video: "vídeo", imagem: "imagem"
};

function rotuloDoTipo(tipo) {
    return rotulos[tipo] || "—";
}

// De que arquivo de MKUTIL_EXEMPLO sai a resposta de um pedido ao motor.
function arquivoDeExemplo(args) {
    var sub = (args || [])[0] || "", alvo = "", i;
    for (i = 1; i < args.length; i++)
        if (args[i].charAt(0) !== "-") {
            alvo = args[i].toLowerCase();
            break;
        }
    if (sub === "inspecionar") {
        if (/\.pkg\.tar\.(zst|xz)$/.test(alvo))
            return "inspecionar-pacote.json";
        return /\.appimage$/.test(alvo) ? "inspecionar-appimage.json" : "inspecionar-compactado.json";
    }
    if (sub === "midia")
        return /\.(png|jpe?g|webp|avif|bmp|tiff?|heic)$/.test(alvo) ? "midia-imagem.json" : "midia-video.json";
    return sub + ".json";
}

// ── MkInstalar ─────────────────────────────────────────────────────────

// Os argumentos do `instalar` para uma ficha. opcoes: { atalho, executavel }.
function argsInstalar(ficha, opcoes) {
    var o = opcoes || {}, args = ["instalar", ficha.arquivo];
    if (ficha.ja_instalado)
        args.push("--substituir");
    if (ficha.tipo !== "pacote" && o.atalho === false)
        args.push("--sem-atalho");
    if (ficha.tipo === "compactado" && o.executavel && o.atalho !== false)
        args.push("--executavel", o.executavel);
    return args;
}

// O texto do botão que instala.
function botaoInstalar(ficha) {
    if (!ficha)
        return "instalar";
    if (ficha.tipo === "pacote")
        return ficha.pacote && ficha.pacote.instalado ? "atualizar (pede a senha)" : "instalar (pede a senha)";
    return ficha.ja_instalado ? "substituir" : "instalar";
}

// Uma linha sobre a ficha: "AppImage · 5.2.2 · 412 MB".
function resumoDaFicha(ficha) {
    if (!ficha)
        return "";
    var partes = [rotuloDoTipo(ficha.tipo)];
    if (ficha.versao)
        partes.push(ficha.versao);
    var bytes = ficha.tipo === "compactado" && ficha.compactado ? ficha.compactado.tamanho_total_bytes
        : ficha.tipo === "pacote" && ficha.pacote && valido(ficha.pacote.tamanho_instalado_bytes) ? ficha.pacote.tamanho_instalado_bytes
        : ficha.tamanho_bytes;
    partes.push(tamanho(bytes));
    return partes.join("  ·  ");
}

// O que mostrar enquanto instala: "copiando 40%", a linha do pacman…
function faseDaInstalacao(fase) {
    if (!fase)
        return "";
    if (fase.fase === "copiando" || fase.fase === "extraindo")
        return fase.fase + (valido(fase.pct) ? " " + Math.round(fase.pct) + "%" : "");
    if (fase.fase === "pacman")
        return fase.linha || "";
    if (fase.fase === "atalho")
        return "criando o atalho";
    return "";
}

// ── MkComprimir ────────────────────────────────────────────────────────

var predefinicoesDeVideo = [
    { v: "menor", texto: "menor possível", frase: "720p, para mandar e não guardar" },
    { v: "equilibrio", texto: "equilíbrio", frase: "até 1080p, boa para quase tudo" },
    { v: "alta", texto: "alta qualidade", frase: "resolução original, quase sem perda" },
    { v: "caber", texto: "caber em um tamanho", frase: "para o limite do mensageiro" },
    { v: "resolucao", texto: "só reduzir a resolução", frase: "1080p, 720p ou 480p" },
    { v: "sem-audio", texto: "tirar o áudio", frase: "sem recodificar a imagem" },
    { v: "gif", texto: "virar GIF", frase: "animação curta, sem som" },
    { v: "webp", texto: "virar WebP animado", frase: "como o GIF, bem menor" }
];

// As opções com que a janela começa.
function opcoesPadrao() {
    return { predefinicao: "equilibrio", alvoMb: 8, lado: 720, codec: "h264", acelerar: "", semAudio: false,
        formato: "jpeg", qualidade: 80, largura: 0 };
}

function par(v) {
    return Math.floor(v / 2 + 0.5) * 2;
}

// O lado menor limitado a `lado`, sem ampliar, com dimensões pares. É a mesma
// conta do motor (os dois testes usam os mesmos casos).
function dimensoes(largura, altura, lado) {
    var menor = Math.min(largura, altura);
    if (!(lado > 0) || menor <= lado)
        return { largura: Math.max(2, Math.floor(largura / 2) * 2), altura: Math.max(2, Math.floor(altura / 2) * 2) };
    lado = Math.floor(lado / 2) * 2;
    var maior = Math.max(2, par(Math.max(largura, altura) * lado / menor));
    return largura >= altura ? { largura: maior, altura: lado } : { largura: lado, altura: maior };
}

function porLargura(largura, altura, alvo) {
    if (!(alvo > 0) || largura <= alvo)
        return { largura: Math.max(2, Math.floor(largura / 2) * 2), altura: Math.max(2, Math.floor(altura / 2) * 2) };
    alvo = Math.max(2, Math.floor(alvo / 2) * 2);
    return { largura: alvo, altura: Math.max(2, par(altura * alvo / largura)) };
}

// A conta do "caber em N MB", como está no contrato: a taxa sai da duração.
// Devolve { cabe, video_bps, audio_bps, largura, altura }.
function caber(alvoMb, duracao, temAudio, largura, altura, fps) {
    if (!(alvoMb > 0) || !(duracao > 0))
        return { cabe: false };
    var total = Math.floor(alvoMb * 8000000 * 0.97 / duracao);
    var audio = !temAudio ? 0 : total >= 500000 ? 96000 : total >= 200000 ? 64000 : 32000;
    var video = total - audio;
    if (video < 50000)
        return { cabe: false };
    var f = fps > 0 ? fps : 30, menor = Math.min(largura, altura);
    var lados = [menor].concat([1080, 720, 480, 360, 240].filter(function (x) { return x < menor; }));
    for (var i = 0; i < lados.length; i++) {
        var d = dimensoes(largura, altura, lados[i]);
        if (video / (d.largura * d.altura * f) >= 0.045)
            return { cabe: true, video_bps: video, audio_bps: audio, largura: d.largura, altura: d.altura };
    }
    var m = dimensoes(largura, altura, Math.min(240, menor));
    return { cabe: true, video_bps: video, audio_bps: audio, largura: m.largura, altura: m.altura };
}

// Bits por pixel de cada predefinição por qualidade (H.264), o lado máximo e
// o áudio. Os outros codecs gastam uma fração disso para a mesma imagem.
var porQualidade = {
    menor: { bpp: 0.035, lado: 720, audio: 64000 },
    equilibrio: { bpp: 0.07, lado: 1080, audio: 128000 },
    alta: { bpp: 0.13, lado: 0, audio: 160000 },
    resolucao: { bpp: 0.08, lado: 0, audio: 128000 }
};
var fatorDoCodec = { h264: 1, h265: 0.65, av1: 0.55 };
var fatorDoFormato = { jpeg: 1, webp: 0.7, avif: 0.5 };

// A estimativa do tamanho da saída, antes de rodar. É uma conta de padeiro
// (a compressão depende do que há na imagem), por isso a janela mostra com
// "≈". Devolve { bytes, encolhe, nota }: bytes null quando não dá para saber
// ou não cabe; `nota` explica.
function estimar(midia, opcoes) {
    var o = opcoes || {};
    if (!midia || !valido(midia.tamanho_bytes))
        return { bytes: null, encolhe: false, nota: "" };
    var antes = midia.tamanho_bytes, bytes, nota = "";
    var fecha = function (b, n) {
        b = Math.max(1, Math.round(b));
        return { bytes: b, encolhe: b < antes * 0.97, nota: n || (b < antes * 0.97 ? "" : "não deve encolher") };
    };
    if (midia.tipo === "imagem") {
        var q = o.qualidade > 0 ? Math.min(100, o.qualidade) : 80;
        var di = o.largura > 0 ? porLargura(midia.largura, midia.altura, o.largura) : { largura: midia.largura, altura: midia.altura };
        var bpp = (0.4 + 3.6 * Math.pow(q / 100, 3)) * (fatorDoFormato[o.formato] || 1);
        return fecha(di.largura * di.altura * bpp / 8);
    }
    var dur = midia.duracao_s;
    if (!(dur > 0))
        return { bytes: null, encolhe: false, nota: "sem a duração não dá para estimar" };
    var fps = midia.fps > 0 ? midia.fps : 30;
    var temAudio = !!midia.audio && !o.semAudio;
    var p = o.predefinicao;
    if (p === "sem-audio") {
        if (!midia.audio)
            return { bytes: null, encolhe: false, nota: "já não tem áudio" };
        var ab = valido(midia.audio.taxa_bps) && midia.audio.taxa_bps > 0 ? midia.audio.taxa_bps : 128000;
        return fecha(Math.max(antes * 0.05, antes - ab * dur / 8));
    }
    if (p === "gif" || p === "webp") {
        var quadros = (p === "gif" ? 12 : 15) * dur;
        var da = porLargura(midia.largura, midia.altura, p === "gif" ? 480 : 640);
        return fecha(da.largura * da.altura * quadros * (p === "gif" ? 0.25 : 0.04), "sem som");
    }
    if (p === "caber") {
        var c = caber(o.alvoMb, dur, temAudio, midia.largura, midia.altura, fps);
        if (!c.cabe)
            return { bytes: null, encolhe: false, nota: o.alvoMb > 0 ? "não cabe: longo demais para esse tamanho" : "diga o tamanho" };
        bytes = (c.video_bps + c.audio_bps) * dur / 8;
        if (bytes >= antes)
            return { bytes: antes, encolhe: false, nota: "o original já cabe" };
        return fecha(bytes, resolucao(c.largura, c.altura) + ", " + taxa(c.video_bps));
    }
    var pq = porQualidade[p];
    if (!pq)
        return { bytes: null, encolhe: false, nota: "" };
    var d = dimensoes(midia.largura, midia.altura, p === "resolucao" ? o.lado : pq.lado);
    var video = d.largura * d.altura * fps * pq.bpp * (fatorDoCodec[o.codec] || 1);
    // Recodificar não inventa detalhe: a saída não passa muito da taxa do original.
    if (valido(midia.taxa_bps) && midia.taxa_bps > 0)
        video = Math.min(video, midia.taxa_bps);
    if (d.largura !== midia.largura || d.altura !== midia.altura)
        nota = resolucao(d.largura, d.altura);
    return fecha((video + (temAudio ? pq.audio : 0)) * dur / 8, nota);
}

// "≈ 7,6 MB (−84%)", "≈ 51 MB: não deve encolher", "não cabe: …".
function textoDaEstimativa(midia, est) {
    if (!est || est.bytes === null)
        return est && est.nota ? est.nota : "—";
    var antes = midia.tamanho_bytes;
    if (!est.encolhe)
        return "≈ " + tamanho(est.bytes) + ": " + est.nota;
    var menos = antes > 0 ? Math.round((1 - est.bytes / antes) * 100) : 0;
    return "≈ " + tamanho(est.bytes) + " (−" + menos + "%)";
}

// Os argumentos do `comprimir` para um arquivo com as opções da janela.
function argsComprimir(midia, opcoes) {
    var o = opcoes || {};
    if (midia.tipo === "imagem") {
        var a = ["comprimir", midia.arquivo, "--predefinicao", "imagem", "--formato", o.formato || "jpeg",
            "--qualidade", String(Math.round(o.qualidade > 0 ? Math.min(100, o.qualidade) : 80))];
        if (o.largura > 0)
            a.push("--largura", String(Math.round(o.largura)));
        return a;
    }
    var p = o.predefinicao || "equilibrio";
    var args = ["comprimir", midia.arquivo, "--predefinicao", p];
    if (p === "caber")
        args.push("--alvo-mb", String(o.alvoMb));
    if (p === "resolucao")
        args.push("--lado", String(o.lado));
    if (["menor", "equilibrio", "alta", "caber", "resolucao"].indexOf(p) >= 0) {
        if (o.codec && o.codec !== "h264")
            args.push("--codec", o.codec);
        if (o.acelerar)
            args.push("--acelerar", o.acelerar);
        if (o.semAudio)
            args.push("--sem-audio");
    }
    return args;
}

// "1920 × 1080  ·  1:02  ·  h264  ·  48 MB".
function resumoDaMidia(m) {
    if (!m)
        return "";
    var partes = [resolucao(m.largura, m.altura)];
    if (m.tipo === "video")
        partes.push(tempo(m.duracao_s));
    if (m.codec)
        partes.push(m.codec);
    partes.push(tamanho(m.tamanho_bytes));
    return partes.join("  ·  ");
}

// "37%  ·  3,1x  ·  faltam 41 s" (e "passada 1 de 2" quando são duas).
function textoDoProgresso(fase) {
    if (!fase || fase.fase !== "codificando")
        return "";
    var partes = [Math.floor(fase.pct) + "%"];
    if (fase.passadas > 1)
        partes.push("passada " + fase.passada + " de " + fase.passadas);
    if (valido(fase.velocidade) && fase.velocidade > 0)
        partes.push(num(fase.velocidade) + "x");
    var r = resta(fase.resta_s);
    if (r !== "")
        partes.push(r);
    return partes.join("  ·  ");
}

// "48 MB → 7,6 MB (−84%)" ou "48 MB → 51 MB: ficou maior".
function textoDoResultado(fim) {
    if (!fim)
        return "";
    var s = tamanho(fim.antes_bytes) + " → " + tamanho(fim.depois_bytes);
    if (fim.maior)
        return s + ": ficou maior que o original";
    return s + " (−" + Math.round(Math.max(0, fim.economia_pct)) + "%)";
}

// O próximo da fila: o primeiro item "na fila". -1 quando acabou.
function proximo(itens) {
    for (var i = 0; i < (itens || []).length; i++)
        if (itens[i].estado === "fila")
            return i;
    return -1;
}

// Quantos itens dá para comprimir agora (lidos e ainda não feitos).
function prontos(itens) {
    return (itens || []).filter(function (i) { return i.estado === "pronto" && !!i.midia; }).length;
}

// O total da lista: { antes, depois, n } só dos que têm estimativa (ou
// resultado, se `feitos`).
function total(itens, opcoes, feitos) {
    var antes = 0, depois = 0, n = 0;
    for (var i = 0; i < (itens || []).length; i++) {
        var it = itens[i];
        if (feitos) {
            if (it.estado === "feito" && it.fim) {
                antes += it.fim.antes_bytes;
                depois += it.fim.depois_bytes;
                n++;
            }
            continue;
        }
        if (it.estado !== "pronto" || !it.midia)
            continue;
        var e = estimar(it.midia, opcoes);
        if (e.bytes === null)
            continue;
        antes += it.midia.tamanho_bytes;
        depois += e.bytes;
        n++;
    }
    return { antes: antes, depois: depois, n: n };
}

// Há vídeo (ou imagem) entre os itens lidos? Para mostrar só as opções que valem.
function tem(itens, tipo) {
    return (itens || []).some(function (i) { return !!i.midia && i.midia.tipo === tipo; });
}
