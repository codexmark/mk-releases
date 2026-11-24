.pragma library

// A lógica do MkEscrita que não depende do QML nem do disco: as edições de
// Markdown (alternar marcação, continuar lista, recuo, link), a busca, a
// máquina de estados do salvamento, o que fazer ao fechar e os textos do
// rodapé. Testada no node (tests/apps/mkescrita-Escrita.test.mjs).
//
// Uma EDIÇÃO é { inicio, fim, texto, selInicio, selFim }: troque o trecho
// [inicio, fim) por `texto` e deixe a seleção em [selInicio, selFim) (iguais =
// só o cursor). As posições são as do TextEdit: índices da string.

// --- pedaços do texto ----------------------------------------------------------

function edicao(inicio, fim, texto, selInicio, selFim) {
    return { inicio: inicio, fim: fim, texto: texto, selInicio: selInicio, selFim: selFim === undefined ? selInicio : selFim };
}

// Aplica uma edição a um texto (os testes usam; a janela aplica no TextEdit,
// para a edição entrar no desfazer).
function aplicar(texto, e) {
    return texto.slice(0, e.inicio) + e.texto + texto.slice(e.fim);
}

// Onde começa e acaba (sem a quebra) a linha em que `pos` está.
function linhaEm(texto, pos) {
    const inicio = texto.lastIndexOf("\n", pos - 1) + 1;
    let fim = texto.indexOf("\n", pos);
    if (fim < 0)
        fim = texto.length;
    return { inicio: inicio, fim: fim, texto: texto.slice(inicio, fim) };
}

// O número (do 1) da linha em que `pos` está.
function linhaDe(texto, pos) {
    let n = 1;
    for (let i = texto.indexOf("\n"); i >= 0 && i < pos; i = texto.indexOf("\n", i + 1))
        n++;
    return n;
}

// A posição em que a linha `linha` (do 1) começa; o fim do texto se não há.
function posicaoDaLinha(texto, linha) {
    let pos = 0;
    for (let n = 1; n < linha; n++) {
        const i = texto.indexOf("\n", pos);
        if (i < 0)
            return texto.length;
        pos = i + 1;
    }
    return pos;
}

// O parágrafo (linhas seguidas sem linha em branco) em que `pos` está. Numa
// linha em branco, é só ela.
function paragrafoEm(texto, pos) {
    const atual = linhaEm(texto, pos);
    if (atual.texto.trim() === "")
        return { inicio: atual.inicio, fim: atual.fim };
    let inicio = atual.inicio;
    while (inicio > 0) {
        const antes = linhaEm(texto, inicio - 1);
        if (antes.texto.trim() === "")
            break;
        inicio = antes.inicio;
    }
    let fim = atual.fim;
    while (fim < texto.length) {
        const depois = linhaEm(texto, fim + 1);
        if (depois.texto.trim() === "")
            break;
        fim = depois.fim;
    }
    return { inicio: inicio, fim: fim };
}

// Letra ou algarismo (o motor de JavaScript do Qt não tem \p{L}: letra é o
// que muda entre maiúscula e minúscula).
function letra(c) {
    return c !== undefined && c !== "" && ((c >= "0" && c <= "9") || c.toLowerCase() !== c.toUpperCase());
}

// Hífen e apóstrofo fazem parte da palavra só entre duas letras ("guarda-chuva").
function liga(texto, i) {
    return (texto[i] === "-" || texto[i] === "'" || texto[i] === "’") && letra(texto[i - 1]) && letra(texto[i + 1]);
}

// A palavra sob o cursor: { inicio, fim }, ou null se o cursor não toca em nenhuma.
function palavraEm(texto, pos) {
    let inicio = pos;
    let fim = pos;
    while (inicio > 0 && (letra(texto[inicio - 1]) || liga(texto, inicio - 1)))
        inicio--;
    while (fim < texto.length && (letra(texto[fim]) || liga(texto, fim)))
        fim++;
    return fim > inicio ? { inicio: inicio, fim: fim } : null;
}

// --- alternar marcação ---------------------------------------------------------

// Quantos `c` seguidos acabam em `pos` (para trás) ou começam em `pos` (para a frente).
function corridaAntes(texto, pos, c) {
    let n = 0;
    while (pos - n > 0 && texto[pos - n - 1] === c)
        n++;
    return n;
}

function corridaDepois(texto, pos, c) {
    let n = 0;
    while (pos + n < texto.length && texto[pos + n] === c)
        n++;
    return n;
}

// Uma corrida de `n` marcas contém a marca pedida? "*" (itálico) está em "*" e
// em "***", mas não em "**" (que é só negrito).
function temMarca(n, marca) {
    return marca.length === 1 ? n === 1 || n >= 3 : n >= marca.length;
}

// Liga ou desliga uma marcação ("**", "*", "~~") na seleção; sem seleção, na
// palavra sob o cursor; sem palavra, abre um par vazio com o cursor no meio
// (e desfaz o par vazio se o cursor já está dentro de um).
function alternarMarca(texto, selInicio, selFim, marca) {
    const c = marca[0];
    const m = marca.length;
    let ini = Math.min(selInicio, selFim);
    let fim = Math.max(selInicio, selFim);
    const semSelecao = ini === fim;
    let cursor = ini;
    if (semSelecao) {
        // Cursor encostado nas marcas ("**palavra|**") conta como dentro.
        const p = palavraEm(texto, ini) || palavraEm(texto, ini - corridaAntes(texto, ini, c))
            || palavraEm(texto, ini + corridaDepois(texto, ini, c));
        if (!p) {
            if (temMarca(corridaAntes(texto, ini, c), marca) && temMarca(corridaDepois(texto, ini, c), marca)
                && corridaAntes(texto, ini, c) === corridaDepois(texto, ini, c))
                return edicao(ini - m, ini + m, "", ini - m);
            return edicao(ini, ini, marca + marca, ini + m);
        }
        ini = p.inicio;
        fim = p.fim;
    }
    const dentro = texto.slice(ini, fim);
    // As marcas estão dentro da seleção: "**assim**" selecionado inteiro.
    if (!semSelecao && dentro.length >= 2 * m && temMarca(corridaDepois(dentro, 0, c), marca)
        && temMarca(corridaAntes(dentro, dentro.length, c), marca)
        && dentro.length > corridaDepois(dentro, 0, c)) {
        return edicao(ini, fim, dentro.slice(m, dentro.length - m), ini, fim - 2 * m);
    }
    // As marcas estão em volta: tira.
    if (temMarca(corridaAntes(texto, ini, c), marca) && temMarca(corridaDepois(texto, fim, c), marca)) {
        if (semSelecao) {
            const novo = Math.max(ini - m, Math.min(cursor - m, fim - m));
            return edicao(ini - m, fim + m, dentro, novo);
        }
        return edicao(ini - m, fim + m, dentro, ini - m, fim - m);
    }
    // Não tem: põe. Os espaços das pontas da seleção ficam de fora das marcas
    // ("** assim **" não é negrito).
    const semPontas = dentro.trim();
    if (semPontas === "")
        return edicao(ini, fim, marca + dentro + marca, ini + m, fim + m);
    const antes = /^\s*/.exec(dentro)[0].length;
    const depois = /\s*$/.exec(dentro)[0].length;
    const novoTexto = dentro.slice(0, antes) + marca + semPontas + marca + dentro.slice(dentro.length - depois);
    if (semSelecao)
        return edicao(ini, fim, novoTexto, cursor + m);
    return edicao(ini, fim, novoTexto, ini + antes + m, fim - depois + m);
}

// Digitar "*", "_" ou "`" com um trecho selecionado envolve o trecho em vez
// de trocá-lo. Sem seleção (ou com outra tecla), null: a tecla segue normal.
function envolver(texto, selInicio, selFim, tecla) {
    const ini = Math.min(selInicio, selFim);
    const fim = Math.max(selInicio, selFim);
    if (ini === fim || ["*", "_", "`"].indexOf(tecla) < 0)
        return null;
    return edicao(ini, fim, tecla + texto.slice(ini, fim) + tecla, ini + 1, fim + 1);
}

// Ctrl+K: "[texto](|)" com a seleção como texto; se a seleção já é um
// endereço, "[|](endereço)"; sem seleção, "[|]()".
function inserirLink(texto, selInicio, selFim) {
    const ini = Math.min(selInicio, selFim);
    const fim = Math.max(selInicio, selFim);
    const sel = texto.slice(ini, fim);
    if (sel === "")
        return edicao(ini, fim, "[]()", ini + 1);
    if (/^(https?:\/\/|mailto:|www\.)\S+$/i.test(sel))
        return edicao(ini, fim, "[](" + sel + ")", ini + 1);
    return edicao(ini, fim, "[" + sel + "]()", ini + sel.length + 3);
}

// --- listas --------------------------------------------------------------------

// O começo de um item de lista ou de uma citação numa linha:
// { recuo, marca, numero, fecho, espaco, tarefa, prefixo }, ou null.
function itemDe(linha) {
    let m = /^(\s*)(?:([-*+])|(\d{1,9})([.)]))( +|$)(\[[ xX]\](?: +|$))?/.exec(linha);
    if (m) {
        return {
            recuo: m[1], marca: m[2] || "", numero: m[3] === undefined ? null : parseInt(m[3], 10), fecho: m[4] || "",
            espaco: m[5] || " ", tarefa: !!m[6], citacao: false, prefixo: m[0]
        };
    }
    m = /^(\s*(?:> ?)+)/.exec(linha);
    if (m)
        return { recuo: "", marca: m[1], numero: null, fecho: "", espaco: "", tarefa: false, citacao: true, prefixo: m[0] };
    return null;
}

// Enter numa linha de lista (ou de citação): continua com a próxima marca
// (o número seguinte; tarefa nova desmarcada); num item vazio, encerra a
// lista (a marca some). Fora de lista, null: o Enter segue normal.
function aoEnter(texto, selInicio, selFim) {
    if (selInicio !== selFim)
        return null;
    const pos = selInicio;
    const l = linhaEm(texto, pos);
    const it = itemDe(l.texto);
    if (!it || pos - l.inicio < it.prefixo.length)
        return null;
    if (l.texto.slice(it.prefixo.length).trim() === "") {
        // Item vazio: um nível a menos se estava aninhado, senão acaba a lista.
        if (!it.citacao && it.recuo.length >= 2) {
            const menos = it.recuo.slice(Math.min(it.recuo.length, it.numero === null ? 2 : 3));
            const nova = menos + l.texto.slice(it.recuo.length);
            return edicao(l.inicio, l.fim, nova, l.inicio + nova.length);
        }
        return edicao(l.inicio, l.fim, "", l.inicio);
    }
    let marca;
    if (it.citacao)
        marca = it.marca.replace(/\s*$/, " ");
    else if (it.numero !== null)
        marca = it.recuo + (it.numero + 1) + it.fecho + " " + (it.tarefa ? "[ ] " : "");
    else
        marca = it.recuo + it.marca + " " + (it.tarefa ? "[ ] " : "");
    // Os espaços em volta do cursor não vão para a linha nova.
    let ini = pos;
    let fim = pos;
    while (ini > l.inicio + it.prefixo.length && texto[ini - 1] === " ")
        ini--;
    while (fim < l.fim && texto[fim] === " ")
        fim++;
    return edicao(ini, fim, "\n" + marca, ini + 1 + marca.length);
}

// Tab e Shift+Tab: recua (ou volta) os itens de lista das linhas tocadas pela
// seleção. Se nenhuma é item de lista, null: a tecla segue normal.
function recuar(texto, selInicio, selFim, menos) {
    const ini = Math.min(selInicio, selFim);
    const fim = Math.max(selInicio, selFim);
    const primeira = linhaEm(texto, ini);
    // Uma seleção que acaba no começo de uma linha não a inclui.
    const ultima = linhaEm(texto, fim > ini && texto[fim - 1] === "\n" ? fim - 1 : fim);
    const linhas = texto.slice(primeira.inicio, ultima.fim).split("\n");
    let mexeu = false;
    let novoIni = ini;
    let novoFim = fim;
    let pos = primeira.inicio;
    const novas = linhas.map((linha) => {
        const it = itemDe(linha);
        let nova = linha;
        if (it && !it.citacao) {
            const passo = it.numero === null ? 2 : 3;
            if (menos)
                nova = linha.slice(Math.min(passo, it.recuo.length));
            else
                nova = " ".repeat(passo) + linha;
        }
        const d = nova.length - linha.length;
        if (d !== 0) {
            mexeu = true;
            // O cursor e a seleção andam junto com o texto deles.
            if (ini > pos || (ini === pos && d > 0 && ini === fim))
                novoIni += Math.max(d, pos - ini);
            if (fim > pos || (fim === pos && d > 0 && ini === fim))
                novoFim += Math.max(d, pos - fim);
        }
        pos += linha.length + 1;
        return nova;
    });
    if (!mexeu)
        return null;
    return edicao(primeira.inicio, ultima.fim, novas.join("\n"), novoIni, novoFim);
}

// --- buscar e substituir -------------------------------------------------------

// O máximo de ocorrências que se conta (uma letra só num texto grande).
var MAXIMO_DE_OCORRENCIAS = 20000;

function expressao(termo, diferenciar) {
    return new RegExp(termo.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), diferenciar ? "gu" : "giu");
}

// As posições em que `termo` aparece no texto, na ordem, sem sobreposição.
// Sem diferenciar maiúsculas, a menos que `diferenciar`. Cada item é
// { inicio, fim }.
function ocorrencias(texto, termo, diferenciar) {
    const achadas = [];
    if (!termo)
        return achadas;
    const re = expressao(termo, !!diferenciar);
    let m;
    while ((m = re.exec(texto)) !== null && achadas.length < MAXIMO_DE_OCORRENCIAS) {
        if (m[0].length === 0) {
            re.lastIndex++;
            continue;
        }
        achadas.push({ inicio: m.index, fim: m.index + m[0].length });
    }
    return achadas;
}

// O índice da ocorrência para onde ir a partir de `pos`: a primeira que começa
// em `pos` ou depois (dando a volta); `paraTras`, a última que acaba em `pos`
// ou antes. -1 se não há nenhuma.
function proxima(achadas, pos, paraTras) {
    if (achadas.length === 0)
        return -1;
    if (paraTras) {
        for (let i = achadas.length - 1; i >= 0; i--)
            if (achadas[i].fim <= pos)
                return i;
        return achadas.length - 1;
    }
    for (let i = 0; i < achadas.length; i++)
        if (achadas[i].inicio >= pos)
            return i;
    return 0;
}

// Qual ocorrência é exatamente a seleção (-1 se nenhuma).
function atual(achadas, selInicio, selFim) {
    const ini = Math.min(selInicio, selFim);
    const fim = Math.max(selInicio, selFim);
    for (let i = 0; i < achadas.length; i++)
        if (achadas[i].inicio === ini && achadas[i].fim === fim)
            return i;
    return -1;
}

// Substituir uma: se a seleção é uma ocorrência, troca; senão null (a janela
// só anda para a próxima). O texto de troca entra ao pé da letra.
function substituirUma(texto, termo, por, diferenciar, selInicio, selFim) {
    const achadas = ocorrencias(texto, termo, diferenciar);
    const i = atual(achadas, selInicio, selFim);
    if (i < 0)
        return null;
    return edicao(achadas[i].inicio, achadas[i].fim, por, achadas[i].inicio + por.length);
}

// Substituir todas: { edicao, quantas }. A edição cobre da primeira à última
// ocorrência (uma edição só: um desfazer só). Nada a trocar: edicao null.
function substituirTodas(texto, termo, por, diferenciar) {
    const achadas = ocorrencias(texto, termo, diferenciar);
    if (achadas.length === 0)
        return { edicao: null, quantas: 0 };
    const primeiro = achadas[0].inicio;
    const ultimo = achadas[achadas.length - 1].fim;
    let novo = "";
    let pos = primeiro;
    for (const a of achadas) {
        novo += texto.slice(pos, a.inicio) + por;
        pos = a.fim;
    }
    return { edicao: edicao(primeiro, ultimo, novo, primeiro + novo.length), quantas: achadas.length };
}

// O que a barra de busca diz: "3 de 12", "12 ocorrências", "nada encontrado".
function textoDaBusca(termo, quantas, indice) {
    if (!termo)
        return "";
    if (quantas === 0)
        return "nada encontrado";
    const total = quantas >= MAXIMO_DE_OCORRENCIAS ? "muitas" : milhar(quantas);
    if (indice >= 0)
        return milhar(indice + 1) + " de " + total;
    return quantas === 1 ? "1 ocorrência" : total + " ocorrências";
}

// --- o salvamento --------------------------------------------------------------

// As fases de um documento:
//   limpo     igual ao disco (ou novo e intocado)
//   sujo      há mudança que não está no disco
//   salvando  o motor está gravando; `pendente` diz se a pessoa digitou no meio
//   salvo     acabou de gravar
//   conflito  o arquivo mudou por fora: nada grava até a pessoa decidir
//   erro      o salvar falhou (sem permissão, disco cheio): o texto continua só aqui
function estadoInicial() {
    return { fase: "limpo", pendente: false };
}

// A máquina de estados: transicao(estado, evento) devolve o estado novo.
// Eventos: "editou", "salvar" (o salvamento começou), "salvou", "conflito",
// "falhou", "abriu" (documento novo, aberto ou recarregado do disco),
// "manter" (a pessoa escolheu ficar com o texto dela depois do conflito).
function transicao(estado, evento) {
    const fase = estado.fase;
    switch (evento) {
    case "abriu":
        return { fase: "limpo", pendente: false };
    case "editou":
        if (fase === "salvando")
            return { fase: "salvando", pendente: true };
        if (fase === "conflito")
            return estado;
        return { fase: "sujo", pendente: false };
    case "salvar":
        // Salvar o que já está no disco não faz nada; durante um salvamento, também não.
        if (fase === "limpo" || fase === "salvo" || fase === "salvando")
            return estado;
        return { fase: "salvando", pendente: false };
    case "salvou":
        if (fase !== "salvando")
            return estado;
        return { fase: estado.pendente ? "sujo" : "salvo", pendente: false };
    case "conflito":
        return { fase: "conflito", pendente: false };
    case "falhou":
        if (fase !== "salvando")
            return estado;
        return { fase: "erro", pendente: false };
    case "manter":
        return fase === "conflito" ? { fase: "sujo", pendente: false } : estado;
    }
    return estado;
}

// Há texto que não está no disco?
function temMudancas(estado) {
    return estado.fase !== "limpo" && estado.fase !== "salvo";
}

// Um DOCUMENTO, para as decisões abaixo, é { fase, pendente, caminho,
// recuperado, soLeitura, vazio }: `recuperado` = o texto veio de um rascunho e
// ainda não foi salvo pela pessoa; `vazio` = não há texto nenhum.

// O salvamento automático pode gravar agora? Só documento com nome, sujo, sem
// conflito. Um rascunho recuperado nunca é gravado sozinho por cima do
// arquivo, e depois de um erro só a pessoa (ou a próxima mudança) tenta de novo.
function deveSalvarSozinho(doc) {
    return doc.fase === "sujo" && !!doc.caminho && !doc.recuperado && !doc.soLeitura;
}

// O texto deve ir para um rascunho agora? Tudo que não está no disco e não é vazio.
function deveGuardarRascunho(doc) {
    return temMudancas(doc) && !doc.vazio;
}

// O que fazer quando a pessoa fecha a janela:
//   "fechar"     nada a perder
//   "salvar"     documento com nome e salvamento automático valendo: grava e fecha
//   "perguntar"  Salvar / Descartar / Cancelar
function aoFechar(doc) {
    if (!temMudancas(doc))
        return "fechar";
    if (!doc.caminho && doc.vazio)
        return "fechar";
    if (doc.fase === "salvando" && !doc.recuperado)
        return "salvar";
    if (deveSalvarSozinho(doc))
        return "salvar";
    return "perguntar";
}

// A tecla que responde à pergunta de fechar: S ou Enter salva, D descarta, C
// ou Esc cancela. "" para qualquer outra.
function respostaDaTecla(tecla) {
    switch (String(tecla).toLowerCase()) {
    case "s":
    case "enter":
    case "return":
        return "salvar";
    case "d":
        return "descartar";
    case "c":
    case "escape":
    case "esc":
        return "cancelar";
    }
    return "";
}

// O vigia avisou que o arquivo mudou por fora (evento "mudou" ou "sumiu", com
// a impressão do que está lá). `conhecida` é o sha256 que a janela tem do
// disco ("ausente" se o arquivo não existia).
//   "ignorar"     é o que eu mesmo gravei (ou nada mudou)
//   "recarregar"  não há mudança minha: mostra o texto novo
//   "conflito"    há mudança minha (ou o arquivo sumiu): a pessoa decide
function aoMudarPorFora(doc, evento, conhecida) {
    if (evento.evento === "sumiu")
        return conhecida === "ausente" ? "ignorar" : "conflito";
    const sha = evento.impressao ? evento.impressao.sha256 : "";
    if (sha !== "" && sha === conhecida)
        return "ignorar";
    if (doc.fase === "conflito")
        return "conflito";
    return temMudancas(doc) || doc.recuperado ? "conflito" : "recarregar";
}

// Ctrl+O com um documento aberto: abre "aqui" se não há nada a perder (limpo
// ou salvo), senão numa "nova" janela. Nunca troca um texto não salvo por outro.
function ondeAbrir(doc) {
    return !temMudancas(doc) || (!doc.caminho && doc.vazio) ? "aqui" : "nova";
}

// --- textos do rodapé e da janela ----------------------------------------------

function nome(caminho) {
    const partes = String(caminho || "").split("/");
    return partes[partes.length - 1] || "";
}

function pasta(caminho) {
    const i = String(caminho || "").lastIndexOf("/");
    return i <= 0 ? (i === 0 ? "/" : "") : caminho.slice(0, i);
}

// O estado, como o rodapé mostra.
function textoDoEstado(doc) {
    switch (doc.fase) {
    case "salvando":
        return "salvando…";
    case "conflito":
        return "conflito";
    case "erro":
        return "não salvou";
    case "sujo":
        return doc.recuperado ? "rascunho recuperado" : "não salvo";
    case "salvo":
        return "salvo";
    }
    if (doc.soLeitura)
        return "só leitura";
    return doc.caminho ? "salvo" : "novo";
}

// O título da janela: um ponto quando há mudança fora do disco, o nome do
// arquivo e o do app.
function tituloDaJanela(doc) {
    return (temMudancas(doc) ? "● " : "") + (nome(doc.caminho) || "sem título") + " — MkEscrita";
}

function milhar(n) {
    return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

function textoDasPalavras(n) {
    if (typeof n !== "number" || !isFinite(n) || n < 0)
        return "";
    return n === 1 ? "1 palavra" : milhar(n) + " palavras";
}

// O tempo de leitura, a partir dos segundos que o motor calcula.
function tempoDeLeitura(segundos) {
    if (typeof segundos !== "number" || !isFinite(segundos) || segundos <= 0)
        return "";
    if (segundos < 60)
        return "menos de 1 min";
    const minutos = Math.round(segundos / 60);
    if (minutos < 60)
        return minutos + " min";
    const resto = minutos % 60;
    return Math.floor(minutos / 60) + " h" + (resto ? " " + resto + " min" : "");
}

// "há 5 min", para o rascunho oferecido.
function haQuanto(agora_s, entao_s) {
    const d = Math.max(0, agora_s - entao_s);
    if (d < 60)
        return "agora há pouco";
    if (d < 3600)
        return "há " + Math.floor(d / 60) + " min";
    if (d < 86400)
        return "há " + Math.floor(d / 3600) + " h";
    const dias = Math.floor(d / 86400);
    return dias === 1 ? "ontem" : "há " + dias + " dias";
}

// A frase de um conflito, pelo motivo que o motor dá.
function fraseDoConflito(motivo, nomeDoArquivo) {
    const n = nomeDoArquivo || "o arquivo";
    if (motivo === "sumiu")
        return n + " sumiu do disco. O seu texto continua aqui.";
    if (motivo === "apareceu")
        return "Já existe um " + n + " no disco, criado por fora.";
    return n + " mudou por fora desde que foi aberto.";
}

// --- o tamanho do texto --------------------------------------------------------

var ZOOM_MINIMO = 50;
var ZOOM_MAXIMO = 300;
var ZOOM_PASSO = 10;

// O zoom depois de `passos` (positivos aumentam), preso entre 50% e 300%.
function zoom(atualPct, passos) {
    const base = typeof atualPct === "number" && isFinite(atualPct) ? atualPct : 100;
    const novo = Math.round((base + passos * ZOOM_PASSO) / ZOOM_PASSO) * ZOOM_PASSO;
    return Math.max(ZOOM_MINIMO, Math.min(ZOOM_MAXIMO, novo));
}

// --- a conversa com o motor ----------------------------------------------------

// Quantos bytes o texto tem em UTF-8: vai no --bytes, para o motor recusar um
// texto que chegou cortado. Meio par substituto solto vira U+FFFD (3 bytes).
function bytesUtf8(texto) {
    let n = 0;
    for (let i = 0; i < texto.length; i++) {
        const c = texto.charCodeAt(i);
        if (c < 0x80)
            n += 1;
        else if (c < 0x800)
            n += 2;
        else if (c >= 0xd800 && c < 0xdc00 && i + 1 < texto.length
            && texto.charCodeAt(i + 1) >= 0xdc00 && texto.charCodeAt(i + 1) < 0xe000) {
            n += 4;
            i++;
        } else
            n += 3;
    }
    return n;
}

// Os argumentos do `salvar`. `conhecida` é o sha256 que a janela tem do disco
// ("ausente" para arquivo que não existia; "" só com forcar).
function argsSalvar(caminho, texto, conhecida, opcoes) {
    const o = opcoes || {};
    const args = ["salvar", "--bytes", String(bytesUtf8(texto))];
    if (o.forcar)
        args.push("--forcar");
    else
        args.push("--esperado", conhecida || "ausente");
    if (o.fimDeLinha === "crlf")
        args.push("--fim-de-linha", "crlf");
    if (o.quebraFinal)
        args.push("--quebra-final");
    return args.concat(["--", caminho]);
}

// Os pedidos que só mandam o texto: "html", "contar", "estrutura".
function argsTexto(sub, texto, extras) {
    return [sub, "--bytes", String(bytesUtf8(texto))].concat(extras || []);
}

function argsRascunho(id, texto, caminho, conhecida) {
    const args = ["rascunho", "guardar", id, "--bytes", String(bytesUtf8(texto))];
    if (caminho)
        args.push("--origem", caminho);
    if (conhecida)
        args.push("--base", conhecida);
    return args;
}

function argsExportar(saida, texto, forcar) {
    const args = ["exportar", "html", "--bytes", String(bytesUtf8(texto))];
    if (forcar)
        args.push("--forcar");
    return args.concat(["--", saida]);
}

// O nome sugerido para o HTML exportado: o do documento, com .html.
function nomeDoExportado(caminho) {
    const n = nome(caminho);
    if (!n)
        return "documento.html";
    const semExtensao = n.replace(/\.(md|markdown|mkd|txt|text)$/i, "");
    return (semExtensao || n) + ".html";
}

// O identificador de um rascunho novo: o instante e um sorteio, só com o que
// o motor aceita ([a-z0-9-]).
function idDeRascunho(agora_ms, sorteio) {
    const s = Math.floor(Math.abs(sorteio || 0) * 0xffffff).toString(36);
    return "r-" + Math.floor(agora_ms).toString(36) + "-" + ("0000" + s).slice(-5);
}

// Qual rascunho oferecer ao abrir: nenhum de janela viva; com arquivo, só o
// daquele arquivo; sem arquivo, o mais recente (a lista já vem nessa ordem).
function rascunhoParaOferecer(itens, caminho) {
    for (const r of itens || []) {
        if (r.vivo)
            continue;
        if (caminho ? r.origem === caminho : true)
            return r;
    }
    return null;
}

// O caminho de um arquivo local a partir da URL que o seletor devolve; ""
// para o que não é arquivo local.
function caminhoDeUrl(url) {
    const m = /^file:\/\/(?:localhost)?(\/.*)$/.exec(String(url || ""));
    if (!m)
        return "";
    try {
        return decodeURIComponent(m[1]);
    } catch (e) {
        return "";
    }
}

// A URL de uma pasta, para o seletor abrir nela e as imagens relativas da
// pré-visualização acharem o caminho.
function urlDaPasta(caminho) {
    const p = pasta(caminho);
    if (!p)
        return "";
    return "file://" + p.split("/").map(encodeURIComponent).join("/") + (p === "/" ? "" : "/");
}

// Só http e https abrem no navegador quando a pessoa clica num link da prévia.
function linkAbrivel(url) {
    return /^https?:\/\/[^\s]+$/i.test(String(url || ""));
}

// Com MKESCRITA_EXEMPLO, o arquivo de resposta gravada de cada pedido.
function arquivoDeExemplo(args) {
    const sub = args[0];
    if (sub === "rascunho" || sub === "preferencias")
        return sub + "-" + args[1] + ".json";
    if (sub === "html")
        return args.indexOf("--pagina") >= 0 ? "html-pagina.json" : "html.json";
    return sub + ".json";
}

// --- sumário, fontes, prévia, atalhos ------------------------------------------

// O índice do título em que a linha `linha` está (o último que começa nela ou antes).
function tituloAtual(titulos, linha) {
    let i = -1;
    for (let k = 0; k < (titulos || []).length; k++)
        if (titulos[k].linha <= linha)
            i = k;
    return i;
}

// As famílias de fonte que casam com o que foi digitado, sem repetir, em ordem.
function filtrarFontes(familias, termo) {
    const t = String(termo || "").trim().toLowerCase();
    const vistas = {};
    const lista = [];
    for (const f of familias || []) {
        if (!f || vistas[f] || (t && f.toLowerCase().indexOf(t) < 0))
            continue;
        vistas[f] = true;
        lista.push(f);
    }
    return lista.sort((a, b) => a.toLowerCase() < b.toLowerCase() ? -1 : a.toLowerCase() > b.toLowerCase() ? 1 : 0);
}

// A folha de estilo que vai na frente do HTML do motor na pré-visualização
// (o Text do Qt entende este pedaço de CSS). As cores vêm do tema.
function estiloDaPrevia(cores, mono) {
    return "<style>"
        + "a { color: " + cores.destaque + "; }"
        + "h1, h2, h3, h4, h5, h6 { color: " + cores.texto + "; }"
        + "code, pre { font-family: '" + String(mono).replace(/['\\<>]/g, "") + "'; color: " + cores.suave + "; }"
        + "blockquote { color: " + cores.apagado + "; }"
        + "th { color: " + cores.suave + "; }"
        + "td, th { padding: 4px 10px; }"
        + "</style>";
}

// Os atalhos, para a folha do F1.
function atalhos() {
    return [
        { grupo: "Arquivo", itens: [
            ["Ctrl+S", "salvar"], ["Ctrl+Shift+S", "salvar como"], ["Ctrl+O", "abrir"], ["Ctrl+N", "nova janela"],
            ["Ctrl+Shift+E", "exportar HTML"], ["Ctrl+Shift+C", "copiar como HTML"], ["Ctrl+W", "fechar"]] },
        { grupo: "Texto", itens: [
            ["Ctrl+B", "negrito"], ["Ctrl+I", "itálico"], ["Ctrl+Shift+X", "tachado"], ["Ctrl+K", "link"],
            ["Tab / Shift+Tab", "recuar item de lista"], ["Ctrl+Z", "desfazer"], ["Ctrl+Shift+Z / Ctrl+Y", "refazer"]] },
        { grupo: "Buscar", itens: [
            ["Ctrl+F", "buscar"], ["Enter / Ctrl+G", "próxima"], ["Shift+Enter", "anterior"], ["Ctrl+H", "substituir"]] },
        { grupo: "Ver", itens: [
            ["Ctrl+Shift+P", "pré-visualização"], ["Ctrl+Shift+O", "sumário"], ["Ctrl+Shift+F", "modo foco"],
            ["F11", "tela cheia"], ["Ctrl+= / Ctrl+-", "tamanho do texto"], ["Ctrl+0", "tamanho normal"],
            ["F1 / Ctrl+/", "esta folha"]] }
    ];
}
