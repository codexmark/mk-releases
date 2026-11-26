.pragma library

// A lógica da janela do MkFiles que não depende do QML: ordenar e filtrar a
// listagem, os textos (tamanho, data, tipo), os nomes (nome livre, parte sem
// extensão), os caminhos (migalhas, completar), os URIs da área de
// transferência, o evento de pasta vigiada e o modo de exemplo.

// --- texto ------------------------------------------------------------------

var ACENTOS = {
    "á": "a", "à": "a", "â": "a", "ã": "a", "ä": "a", "å": "a", "é": "e", "è": "e", "ê": "e", "ë": "e",
    "í": "i", "ì": "i", "î": "i", "ï": "i", "ó": "o", "ò": "o", "ô": "o", "õ": "o", "ö": "o",
    "ú": "u", "ù": "u", "û": "u", "ü": "u", "ç": "c", "ñ": "n", "ý": "y", "ÿ": "y"
};

// Minúsculas e sem acento: "Ação" e "acao" são o mesmo texto para buscar e ordenar.
function semAcento(texto) {
    var s = String(texto).toLowerCase();
    var saida = "";
    for (var i = 0; i < s.length; i++) {
        var c = s.charAt(i);
        saida += ACENTOS[c] !== undefined ? ACENTOS[c] : c;
    }
    return saida;
}

// A chave da ordem natural: cada sequência de algarismos vira "tamanho +
// algarismos", para "foto 9" vir antes de "foto 10" numa comparação de texto.
function chave(nome) {
    var s = semAcento(nome);
    var saida = "";
    var i = 0;
    while (i < s.length) {
        var c = s.charCodeAt(i);
        if (c >= 48 && c <= 57) {
            var j = i;
            while (j < s.length && s.charCodeAt(j) >= 48 && s.charCodeAt(j) <= 57)
                j++;
            var numero = s.substring(i, j).replace(/^0+(?=\d)/, "");
            saida += "0" + String.fromCharCode(48 + Math.min(numero.length, 70)) + numero;
            i = j;
        } else {
            saida += s.charAt(i);
            i++;
        }
    }
    return saida;
}

// --- a listagem ---------------------------------------------------------------

// Pasta, ou link que leva a uma.
function ePasta(item) {
    return !!item && (item.tipo === "pasta" || (item.tipo === "link" && item.alvo_tipo === "pasta"));
}

function chaveDe(item) {
    if (item._k === undefined)
        item._k = chave(item.nome);
    return item._k;
}

// A comparação de dois itens: ordem = { campo, inversa }.
function comparar(a, b, ordem, pastasAntes) {
    if (pastasAntes) {
        var pa = ePasta(a), pb = ePasta(b);
        if (pa !== pb)
            return pa ? -1 : 1;
    }
    var r = 0;
    var campo = ordem && ordem.campo || "nome";
    if (campo === "tamanho")
        r = (a.tamanho || 0) - (b.tamanho || 0);
    else if (campo === "mtime")
        r = (a.mtime || 0) - (b.mtime || 0);
    else if (campo === "apagado")
        r = (a.apagado_s || 0) - (b.apagado_s || 0);
    else if (campo === "tipo") {
        var ta = tipo(a), tb = tipo(b);
        r = ta < tb ? -1 : ta > tb ? 1 : 0;
    }
    if (r === 0) {
        var ka = chaveDe(a), kb = chaveDe(b);
        r = ka < kb ? -1 : ka > kb ? 1 : a.nome < b.nome ? -1 : a.nome > b.nome ? 1 : 0;
    }
    return ordem && ordem.inversa ? -r : r;
}

// Uma cópia ordenada.
function ordenar(itens, ordem, pastasAntes) {
    return itens.slice().sort(function (a, b) { return comparar(a, b, ordem, pastasAntes); });
}

// Um padrão com * e ? (sem diferenciar maiúsculas nem acentos). Sem curinga,
// é um trecho do nome. Vazio casa com tudo.
function casaPadrao(nome, padrao) {
    var p = semAcento(String(padrao).trim());
    if (p === "")
        return true;
    var n = semAcento(nome);
    if (p.indexOf("*") < 0 && p.indexOf("?") < 0)
        return n.indexOf(p) >= 0;
    var re = "";
    for (var i = 0; i < p.length; i++) {
        var c = p.charAt(i);
        re += c === "*" ? ".*" : c === "?" ? "." : c.replace(/[.+^${}()|[\]\\]/g, "\\$&");
    }
    return new RegExp("^" + re + "$").test(n);
}

// O que aparece: sem os ocultos (a não ser que se peça) e só o que casa com o texto.
function filtrar(itens, opcoes) {
    var ocultos = !!(opcoes && opcoes.ocultos);
    var texto = opcoes && opcoes.texto || "";
    return itens.filter(function (it) {
        return (ocultos || !it.oculto) && casaPadrao(it.nome, texto);
    });
}

// A listagem como a visão mostra: filtrada e ordenada.
function visiveis(todos, opcoes) {
    return ordenar(filtrar(todos, opcoes), opcoes && opcoes.ordem, !opcoes || opcoes.pastas_antes !== false);
}

// O evento "pasta" do motor aplicado à listagem crua: os que mudaram entram
// (ou trocam o que havia com o mesmo nome), os que saíram somem.
function aplicarEvento(todos, evento) {
    var fora = {};
    var novos = {};
    var i;
    for (i = 0; i < (evento.saiu || []).length; i++)
        fora[evento.saiu[i]] = true;
    for (i = 0; i < (evento.mudou || []).length; i++)
        novos[evento.mudou[i].nome] = evento.mudou[i];
    var saida = [];
    for (i = 0; i < todos.length; i++) {
        var nome = todos[i].nome;
        if (novos[nome] !== undefined) {
            saida.push(novos[nome]);
            delete novos[nome];
        } else if (!fora[nome]) {
            saida.push(todos[i]);
        }
    }
    for (i = 0; i < (evento.mudou || []).length; i++)
        if (novos[evento.mudou[i].nome] !== undefined) {
            saida.push(evento.mudou[i]);
            delete novos[evento.mudou[i].nome];
        }
    return saida;
}

// O primeiro item cujo nome começa com o texto digitado, procurando a partir
// de `de` e dando a volta. -1 se nenhum.
function pular(itens, texto, de) {
    var t = semAcento(texto);
    if (t === "" || itens.length === 0)
        return -1;
    var inicio = de >= 0 && de < itens.length ? de : 0;
    for (var k = 0; k < itens.length; k++) {
        var i = (inicio + k) % itens.length;
        if (semAcento(itens[i].nome).indexOf(t) === 0)
            return i;
    }
    return -1;
}

// --- textos -------------------------------------------------------------------

function virgula(numero, casas) {
    return numero.toFixed(casas).replace(".", ",");
}

// 1536 -> "1,5 kB" (múltiplos de 1000, como os discos são vendidos).
function tamanho(bytes) {
    if (bytes === null || bytes === undefined || isNaN(bytes))
        return "—";
    var n = Number(bytes);
    if (n < 1000)
        return Math.round(n) + " B";
    var unidades = ["kB", "MB", "GB", "TB", "PB"];
    var u = -1;
    while (n >= 999.5 && u < unidades.length - 1) {
        n /= 1000;
        u++;
    }
    return (n >= 99.5 ? String(Math.round(n)) : virgula(n, 1).replace(/,0$/, "")) + " " + unidades[u];
}

var MESES = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];

function dois(n) { return n < 10 ? "0" + n : String(n); }

// "hoje, 14:05", "ontem, 09:00", "3 de mar., 08:15", "3 de mar. de 2021".
function data(segundos, agora) {
    if (!segundos)
        return "—";
    var d = new Date(segundos * 1000);
    var a = agora ? new Date(agora * 1000) : new Date();
    var hora = dois(d.getHours()) + ":" + dois(d.getMinutes());
    var dia = function (x) { return new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime(); };
    var dias = Math.round((dia(a) - dia(d)) / 86400000);
    if (dias === 0)
        return "hoje, " + hora;
    if (dias === 1)
        return "ontem, " + hora;
    var base = d.getDate() + " de " + MESES[d.getMonth()] + ".";
    return d.getFullYear() === a.getFullYear() ? base + ", " + hora : base + " de " + d.getFullYear();
}

var TIPOS = {
    "inode/directory": "Pasta", "inode/symlink": "Link quebrado", "application/pdf": "Documento PDF",
    "application/zip": "Arquivo ZIP", "application/gzip": "Arquivo gzip", "application/x-compressed-tar": "Arquivo tar.gz",
    "application/x-tar": "Arquivo tar", "application/x-7z-compressed": "Arquivo 7z", "application/x-shellscript": "Script de shell",
    "application/x-executable": "Programa", "application/octet-stream": "Dados", "application/x-zerosize": "Arquivo vazio",
    "application/json": "Dados JSON", "text/plain": "Texto", "text/markdown": "Texto Markdown", "text/html": "Página HTML",
    "text/csv": "Planilha CSV", "image/svg+xml": "Desenho SVG", "inode/fifo": "Canal", "inode/socket": "Soquete",
    "inode/chardevice": "Dispositivo", "inode/blockdevice": "Dispositivo de disco"
};
var FAMILIAS = { image: "Imagem", video: "Vídeo", audio: "Áudio", text: "Texto", font: "Fonte" };

// O tipo como a pessoa lê: "Imagem JPEG", "Pasta", "Link para pasta".
function tipo(item) {
    if (!item)
        return "";
    if (item.tipo === "link" && item.quebrado)
        return "Link quebrado";
    var m = item.mime || "";
    var texto = TIPOS[m];
    if (texto === undefined) {
        var partes = m.split("/");
        var sub = (partes[1] || "").replace(/^x-/, "").replace(/^vnd\./, "");
        var familia = FAMILIAS[partes[0]];
        if (familia !== undefined)
            texto = sub !== "" ? familia + " " + sub.toUpperCase() : familia;
        else if (m.indexOf("opendocument") >= 0 || m.indexOf("officedocument") >= 0 || m.indexOf("msword") >= 0)
            texto = "Documento de escritório";
        else
            texto = sub !== "" ? "Arquivo " + sub : "Arquivo";
    }
    if (item.tipo === "link")
        return "Link para " + texto.charAt(0).toLowerCase() + texto.substring(1);
    return texto;
}

// --- nomes ----------------------------------------------------------------------

// [base, extensão]: "foto.jpg" -> ["foto", ".jpg"]; "a.tar.gz" -> ["a",
// ".tar.gz"]; ".bashrc" e pasta não têm extensão.
function partes(nome, pasta) {
    if (pasta)
        return [nome, ""];
    var m = /^(.+?)(\.tar\.[A-Za-z0-9]{1,5})$/.exec(nome);
    if (m)
        return [m[1], m[2]];
    var i = nome.lastIndexOf(".");
    if (i <= 0 || i === nome.length - 1)
        return [nome, ""];
    return [nome.substring(0, i), nome.substring(i)];
}

// Quantos caracteres o F2 seleciona: o nome sem a extensão.
function semExtensao(nome, pasta) {
    return partes(nome, pasta)[0].length;
}

// O primeiro nome livre: "x.ext", "x (2).ext", "x (3).ext"… `ocupado(nome)`
// diz se já existe.
function nomeLivre(nome, ocupado, pasta) {
    if (!ocupado(nome))
        return nome;
    var p = partes(nome, pasta);
    var m = /^(.*) \((\d+)\)$/.exec(p[0]);
    var base = m ? m[1] : p[0];
    for (var n = m ? Number(m[2]) + 1 : 2; n < 100000; n++) {
        var tentativa = base + " (" + n + ")" + p[1];
        if (!ocupado(tentativa))
            return tentativa;
    }
    return nome;
}

// Um nome novo serve? Devolve a frase do erro, ou "".
function erroDoNome(nome) {
    if (nome === "")
        return "o nome não pode ser vazio";
    if (nome === "." || nome === "..")
        return "«" + nome + "» não serve de nome";
    if (nome.indexOf("/") >= 0)
        return "o nome não pode ter «/»";
    if (nome.length > 255)
        return "nome comprido demais";
    return "";
}

// --- caminhos -----------------------------------------------------------------

// Dentro de um compactado o lugar é "arquivo//interno" (veja "compactados",
// mais abaixo): subir da raiz dele sai para a pasta do arquivo.
function pai(caminho) {
    var c = partesDoCompactado(caminho);
    if (c !== null) {
        if (c.interno === "")
            return pai(c.arquivo);
        var k = c.interno.lastIndexOf("/");
        return noCompactado(c.arquivo, k < 0 ? "" : c.interno.substring(0, k));
    }
    var i = caminho.lastIndexOf("/");
    return i <= 0 ? "/" : caminho.substring(0, i);
}

function nomeDe(caminho) {
    var c = partesDoCompactado(caminho);
    if (c !== null && c.interno === "")
        return nomeDe(c.arquivo);
    return caminho === "/" ? "/" : caminho.substring(caminho.lastIndexOf("/") + 1);
}

function juntar(pasta, nome) {
    return pasta === "/" || pasta.substring(pasta.length - 2) === SEPARADOR ? pasta + nome : pasta + "/" + nome;
}

// filho está dentro de pasta (ou é ela)?
function dentro(pasta, filho) {
    return filho === pasta || pasta === "/" || filho.indexOf(pasta + "/") === 0;
}

// O caminho que a pessoa digitou, limpo e absoluto ("~" é a pasta pessoal;
// ".." sobe). null se não dá um caminho absoluto.
function limpar(texto, home) {
    var t = String(texto).trim();
    if (t.indexOf("file://") === 0) {
        var c = deUri(t);
        if (c === null)
            return null;
        t = c;
    }
    if (t === "~" || t.indexOf("~/") === 0)
        t = home + t.substring(1);
    if (t.charAt(0) !== "/" || t.indexOf("\u0000") >= 0)
        return null;
    var saida = [];
    var pedacos = t.split("/");
    for (var i = 0; i < pedacos.length; i++) {
        if (pedacos[i] === "" || pedacos[i] === ".")
            continue;
        if (pedacos[i] === "..")
            saida.pop();
        else
            saida.push(pedacos[i]);
    }
    return "/" + saida.join("/");
}

// As migalhas de um caminho: [{ nome, caminho }]. A pasta pessoal vira uma
// migalha só.
function migalhas(caminho, home) {
    var c = partesDoCompactado(caminho);
    if (c !== null) {
        // "arquivo.zip › pasta": as migalhas do arquivo, e depois as de dentro.
        var fora = migalhas(c.arquivo, home);
        fora[fora.length - 1].caminho = noCompactado(c.arquivo, "");
        var por = "";
        var dentroDele = c.interno === "" ? [] : c.interno.split("/");
        for (var d = 0; d < dentroDele.length; d++) {
            por = por === "" ? dentroDele[d] : por + "/" + dentroDele[d];
            fora.push({ nome: dentroDele[d], caminho: noCompactado(c.arquivo, por) });
        }
        return fora;
    }
    var lista = [];
    var resto = caminho;
    if (home && home !== "/" && dentro(home, caminho)) {
        lista.push({ nome: "Pasta pessoal", caminho: home });
        resto = caminho.substring(home.length);
    } else {
        lista.push({ nome: "/", caminho: "/" });
    }
    var atual = lista[0].caminho;
    var pedacos = resto.split("/");
    for (var i = 0; i < pedacos.length; i++) {
        if (pedacos[i] === "")
            continue;
        atual = juntar(atual, pedacos[i]);
        lista.push({ nome: pedacos[i], caminho: atual });
    }
    return lista;
}

// Como o caminho aparece no campo: a pasta pessoal vira "~".
function til(caminho, home) {
    if (home && home !== "/" && dentro(home, caminho))
        return "~" + caminho.substring(home.length);
    return caminho;
}

// O Tab do campo de caminho: `texto` é o que está digitado e `itens` a
// listagem da pasta de pai(texto). Devolve { texto, opcoes }: o texto
// completado até onde as pastas concordam, e as pastas que ainda cabem.
function completar(texto, itens) {
    var corte = texto.lastIndexOf("/");
    if (corte < 0)
        return { texto: texto, opcoes: [] };
    var prefixo = texto.substring(corte + 1);
    var p = semAcento(prefixo);
    var nomes = [];
    for (var i = 0; i < itens.length; i++)
        if (ePasta(itens[i]) && semAcento(itens[i].nome).indexOf(p) === 0 && (p !== "" ? true : !itens[i].oculto))
            nomes.push(itens[i].nome);
    nomes.sort(function (a, b) { return chave(a) < chave(b) ? -1 : 1; });
    if (nomes.length === 0)
        return { texto: texto, opcoes: [] };
    if (nomes.length === 1)
        return { texto: texto.substring(0, corte + 1) + nomes[0] + "/", opcoes: [] };
    var comum = nomes[0];
    for (var j = 1; j < nomes.length; j++) {
        var k = 0;
        while (k < comum.length && k < nomes[j].length && semAcento(comum.charAt(k)) === semAcento(nomes[j].charAt(k)))
            k++;
        comum = comum.substring(0, k);
    }
    return { texto: texto.substring(0, corte + 1) + (comum.length > prefixo.length ? comum : prefixo), opcoes: nomes };
}

// --- a área de transferência ------------------------------------------------------

// Um caminho como URI, do jeito do motor: cada pedaço codificado, as barras ficam.
function uri(caminho) {
    return "file://" + caminho.split("/").map(function (p) {
        return encodeURIComponent(p).replace(/[!'()*]/g, function (c) {
            return "%" + c.charCodeAt(0).toString(16).toUpperCase();
        }).replace(/%(24|26|2B|3A|3D|40)/g, function (m) { return decodeURIComponent(m); });
    }).join("/");
}

// O caminho de um file://… (null se não é um daqui).
function deUri(texto) {
    var t = String(texto).trim();
    if (t.indexOf("file://") !== 0)
        return null;
    var resto = t.substring(7);
    if (resto.indexOf("localhost/") === 0)
        resto = resto.substring(9);
    if (resto.charAt(0) !== "/")
        return null;
    try {
        resto = decodeURIComponent(resto);
    } catch (e) {
        return null;
    }
    if (resto.indexOf("\u0000") >= 0)
        return null;
    return limparSimples(resto);
}

function limparSimples(c) {
    var saida = [];
    var pedacos = c.split("/");
    for (var i = 0; i < pedacos.length; i++) {
        if (pedacos[i] === "" || pedacos[i] === ".")
            continue;
        if (pedacos[i] === "..")
            saida.pop();
        else
            saida.push(pedacos[i]);
    }
    return "/" + saida.join("/");
}

// text/uri-list: um URI por linha, com \r\n.
function listaDeUris(caminhos) {
    return caminhos.map(uri).join("\r\n") + "\r\n";
}

function lerListaDeUris(texto) {
    var caminhos = [];
    var linhas = String(texto || "").split(/\r?\n/);
    for (var i = 0; i < linhas.length; i++) {
        if (linhas[i] === "" || linhas[i].charAt(0) === "#")
            continue;
        var c = deUri(linhas[i]);
        if (c !== null)
            caminhos.push(c);
    }
    return caminhos;
}

// x-special/gnome-copied-files: "copy" ou "cut", e um URI por linha.
function copiados(caminhos, recortar) {
    return (recortar ? "cut" : "copy") + "\n" + caminhos.map(uri).join("\n");
}

function lerCopiados(texto) {
    var linhas = String(texto || "").split(/\r?\n/);
    var acao = linhas.shift();
    if (acao !== "cut" && acao !== "copy")
        return null;
    return { recortar: acao === "cut", caminhos: lerListaDeUris(linhas.join("\n")) };
}

// O pedido area-por para copiar ou recortar: o wl-copy oferece um tipo por
// vez, e é o gnome-copied-files ("copy" ou "cut" e os URIs) que os
// gerenciadores de arquivos leem; vai sempre ele.
function pedidoDeCopia(caminhos, recortar) {
    return { op: "area-por", tipo: "x-special/gnome-copied-files", texto: copiados(caminhos, recortar === true) };
}

// Um texto que é só caminhos ou file://…, um por linha: os caminhos (ou []
// se alguma linha não é um).
function caminhosDeTexto(texto) {
    var linhas = String(texto || "").split(/\r?\n/).filter(function (l) { return l.trim() !== ""; });
    var caminhos = [];
    for (var i = 0; i < linhas.length; i++) {
        var l = linhas[i].trim();
        var c = l.charAt(0) === "/" && l.indexOf("\u0000") < 0 ? limparSimples(l) : deUri(l);
        if (c === null)
            return [];
        caminhos.push(c);
    }
    return caminhos;
}

// O que a resposta do area-ler quer dizer: { tipo: "arquivos" | "imagem" |
// "texto" | "vazia", recortar, caminhos }.
function lerArea(r) {
    var vazia = { tipo: "vazia", recortar: false, caminhos: [] };
    if (!r || r.erro || !r.tipo)
        return vazia;
    if (r.tipo === "x-special/gnome-copied-files") {
        var c = lerCopiados(r.texto);
        return c && c.caminhos.length > 0 ? { tipo: "arquivos", recortar: c.recortar, caminhos: c.caminhos } : vazia;
    }
    if (r.tipo === "text/uri-list") {
        var lista = lerListaDeUris(r.texto);
        return lista.length > 0 ? { tipo: "arquivos", recortar: false, caminhos: lista } : vazia;
    }
    if (r.tipo === "image/png")
        return { tipo: "imagem", recortar: false, caminhos: [] };
    if (r.tipo === "text/plain" && r.texto) {
        // Caminhos ou file://, um por linha, colados como texto também são
        // arquivos (é o que um terminal ou outro gerenciador deixa).
        var soCaminhos = caminhosDeTexto(r.texto);
        if (soCaminhos.length > 0)
            return { tipo: "arquivos", recortar: false, caminhos: soCaminhos };
        return { tipo: "texto", recortar: false, caminhos: [] };
    }
    return vazia;
}

// --- estado, tarefas, erros -------------------------------------------------------

// A soma do que está selecionado: { n, pastas, tamanho }.
function soma(itens) {
    var s = { n: itens.length, pastas: 0, tamanho: 0 };
    for (var i = 0; i < itens.length; i++) {
        if (ePasta(itens[i]))
            s.pastas++;
        else
            s.tamanho += itens[i].tamanho || 0;
    }
    return s;
}

function plural(n, um, varios) {
    return n + " " + (n === 1 ? um : varios);
}

// A frase da barra de estado.
function estado(total, selecionados, espaco, carregando) {
    var partesDaFrase = [carregando ? "lendo… " + plural(total, "item", "itens") : plural(total, "item", "itens")];
    if (selecionados && selecionados.n > 0) {
        var t = plural(selecionados.n, "selecionado", "selecionados");
        if (selecionados.n > selecionados.pastas)
            t += " (" + tamanho(selecionados.tamanho) + ")";
        partesDaFrase.push(t);
    }
    if (espaco && espaco.total)
        partesDaFrase.push(tamanho(espaco.livre) + " livres de " + tamanho(espaco.total));
    return partesDaFrase.join("  ·  ");
}

// Quanto de uma tarefa já foi (0 a 1).
function progresso(tarefa) {
    if (!tarefa)
        return 0;
    if (tarefa.bytes_total > 0)
        return Math.min(1, tarefa.bytes / tarefa.bytes_total);
    if (tarefa.itens_total > 0)
        return Math.min(1, tarefa.itens / tarefa.itens_total);
    return 0;
}

function tarefaAcabou(tarefa) {
    return tarefa.estado === "feita" || tarefa.estado === "cancelada" || tarefa.estado === "erro";
}

// A linha de detalhe de uma tarefa na área de tarefas.
function detalheDaTarefa(t) {
    var textos = { fila: "na fila", pausada: "pausada", conflito: "esperando resposta", cancelada: "cancelada", feita: "pronto" };
    if (t.estado === "erro")
        return plural((t.erros || []).length, "erro", "erros") + ": " + ((t.erros || [])[0] ? t.erros[0].erro : "");
    if (textos[t.estado] !== undefined)
        return textos[t.estado];
    var texto = (t.itens || 0) + " de " + (t.itens_total || 0);
    if (t.bytes_total > 0)
        texto += "  ·  " + tamanho(t.bytes) + " de " + tamanho(t.bytes_total);
    if (t.velocidade > 0)
        texto += "  ·  " + tamanho(t.velocidade) + "/s";
    return texto;
}

// O que dizer quando uma pasta não abre: { titulo, frase }.
function erroDePasta(motivo, frase) {
    var titulos = {
        "nao-existe": "Esta pasta não existe mais",
        "sem-permissao": "Você não pode abrir esta pasta",
        "nao-e-pasta": "Isto não é uma pasta",
        "sumiu": "A pasta sumiu",
        "senha": "Este compactado pede uma senha",
        "falta": "Falta uma ferramenta para abrir este compactado",
        "formato": "Não sei abrir este compactado",
        "motor": "O motor parou de responder"
    };
    var frases = {
        "nao-existe": "Ela foi apagada, movida, ou o disco em que estava foi desmontado.",
        "sem-permissao": "As permissões dela não deixam você ver o que há dentro.",
        "sumiu": "Ela foi apagada ou movida, ou o disco saiu, enquanto estava aberta."
    };
    return { titulo: titulos[motivo] || "Não deu para abrir esta pasta", frase: frases[motivo] || frase || "" };
}

// Os bits de permissão: quem = 0 (dono), 1 (grupo), 2 (outros); qual = "r", "w", "x".
function bit(quem, qual) {
    return ({ r: 4, w: 2, x: 1 })[qual] << (3 * (2 - quem));
}

function temBit(modo, quem, qual) {
    return (modo & bit(quem, qual)) !== 0;
}

function trocarBit(modo, quem, qual) {
    return (modo ^ bit(quem, qual)) & 4095;
}

function octal(modo) {
    var s = (modo & 4095).toString(8);
    while (s.length < 3)
        s = "0" + s;
    return s;
}

// --- os filtros da busca ---------------------------------------------------------

var UNIDADES = { "KB": 1024, "MB": 1024 * 1024, "GB": 1024 * 1024 * 1024 };
var PERIODOS = [["", "qualquer data"], ["hoje", "hoje"], ["7", "7 dias"], ["30", "30 dias"], ["ano", "este ano"], ["datas", "entre datas"]];

// "1,5" MB -> bytes (null se não é um número).
function tamanhoEmBytes(texto, unidade) {
    var t = String(texto || "").trim().replace(",", ".");
    if (!/^\d+(\.\d+)?$/.test(t))
        return null;
    return Math.round(Number(t) * (UNIDADES[unidade] || 1));
}

// "2026-01-31" (ou "31/01/2026") -> segundos do começo desse dia, na hora
// local; null se não é uma data.
function dataDeTexto(texto) {
    var t = String(texto || "").trim();
    var m = /^(\d{4})-(\d{1,2})-(\d{1,2})$/.exec(t);
    var ano, mes, dia;
    if (m) {
        ano = Number(m[1]); mes = Number(m[2]); dia = Number(m[3]);
    } else {
        m = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(t);
        if (!m)
            return null;
        ano = Number(m[3]); mes = Number(m[2]); dia = Number(m[1]);
    }
    var d = new Date(ano, mes - 1, dia);
    if (d.getFullYear() !== ano || d.getMonth() !== mes - 1 || d.getDate() !== dia)
        return null;
    return Math.floor(d.getTime() / 1000);
}

// Os quatro campos de filtro do pedido buscar, a partir do que a pessoa pôs:
// f = { min, max, unidade, periodo, de, ate }; agora em segundos.
function filtrosDaBusca(f, agora) {
    f = f || {};
    var r = { tamanho_min: tamanhoEmBytes(f.min, f.unidade), tamanho_max: tamanhoEmBytes(f.max, f.unidade), depois_s: null, antes_s: null };
    var hoje = new Date(agora * 1000);
    var meiaNoite = Math.floor(new Date(hoje.getFullYear(), hoje.getMonth(), hoje.getDate()).getTime() / 1000);
    if (f.periodo === "hoje") {
        r.depois_s = meiaNoite;
    } else if (f.periodo === "7" || f.periodo === "30") {
        r.depois_s = meiaNoite - (Number(f.periodo) - 1) * 86400;
    } else if (f.periodo === "ano") {
        r.depois_s = Math.floor(new Date(hoje.getFullYear(), 0, 1).getTime() / 1000);
    } else if (f.periodo === "datas") {
        r.depois_s = dataDeTexto(f.de);
        var ate = dataDeTexto(f.ate);
        // "até" inclui o dia inteiro.
        r.antes_s = ate === null ? null : ate + 86400;
    }
    return r;
}

// Há algum filtro posto?
function temFiltro(f) {
    var r = filtrosDaBusca(f, 0);
    return r.tamanho_min !== null || r.tamanho_max !== null || (!!f && !!f.periodo);
}

// --- compactados -------------------------------------------------------------------

// Dentro de um compactado o "caminho" da aba é o do arquivo, "//", e o
// caminho de dentro ("/x/a.zip//pasta/sub"; a raiz é "/x/a.zip//"). Um
// caminho limpo nunca tem "//": não há como confundir com uma pasta.
var SEPARADOR = "//";

function noCompactado(arquivo, interno) {
    return arquivo + SEPARADOR + String(interno || "").replace(/^\/+|\/+$/g, "");
}

// { arquivo, interno } de um lugar dentro de um compactado; null se é uma
// pasta comum.
function partesDoCompactado(caminho) {
    var i = String(caminho).indexOf(SEPARADOR, 1);
    if (i < 0)
        return null;
    return { arquivo: caminho.substring(0, i), interno: caminho.substring(i + 2) };
}

// A pasta de verdade em que se está (a do arquivo, dentro de um compactado).
function pastaReal(caminho) {
    var c = partesDoCompactado(caminho);
    return c === null ? caminho : pai(c.arquivo);
}

var EXTENSOES_DE_COMPACTADO = /\.(zip|tar|tgz|tbz2?|txz|tzst|gz|bz2|xz|zst|7z|rar)$/i;

// O item é um compactado que o MkFiles abre como pasta e extrai?
function eCompactado(item) {
    if (!item || item.tipo === "pasta" || (item.tipo !== "arquivo" && item.tipo !== "link"))
        return false;
    var m = item.mime || "";
    if (/^application\/(zip|gzip|x-tar|x-compressed-tar|x-bzip2?|x-bzip2?-compressed-tar|x-xz|x-xz-compressed-tar|zstd|x-zstd-compressed-tar|x-7z-compressed|vnd\.rar|x-rar)$/.test(m))
        return true;
    return EXTENSOES_DE_COMPACTADO.test(item.nome || "");
}

// O nome da pasta em que um compactado é extraído: o dele, sem a extensão.
function pastaDeExtracao(nome) {
    var n = String(nome).replace(/\.tar\.[A-Za-z0-9]{1,5}$/i, "").replace(EXTENSOES_DE_COMPACTADO, "");
    return n === "" || n === nome ? nome + " (extraído)" : n;
}

// O nome do arquivo que "compactar" sugere para uma seleção.
function nomeDoCompactado(itens, pasta, formato) {
    var base = itens.length === 1 ? partes(itens[0].nome, ePasta(itens[0]))[0] : nomeDe(pasta);
    if (base === "" || base === "/")
        base = "Arquivos";
    return base + "." + formato;
}

// Troca a extensão de compactado de um nome pela do formato escolhido.
function comFormato(nome, formato) {
    var n = String(nome).replace(/\.(zip|7z|tar\.gz|tar\.zst|tar\.xz|tgz)$/i, "");
    return (n === "" ? "Arquivos" : n) + "." + formato;
}

// A marca que vai para a área de transferência quando se copia de dentro de
// um compactado, e a leitura dela: colar extrai só o que foi copiado.
var MARCA = "mkfiles-compactado:";

function marcaDoCompactado(arquivo, internos) {
    return MARCA + arquivo + "\n" + internos.join("\n");
}

function lerMarca(texto) {
    var t = String(texto || "");
    if (t.indexOf(MARCA) !== 0)
        return null;
    var linhas = t.substring(MARCA.length).split("\n");
    var arquivo = linhas.shift();
    var internos = linhas.filter(function (l) { return l !== ""; });
    return arquivo.charAt(0) === "/" && internos.length > 0 ? { arquivo: arquivo, internos: internos } : null;
}

// --- o modo de exemplo ------------------------------------------------------------

// O arquivo com as linhas gravadas de um pedido.
function arquivoDeExemplo(pedido) {
    return String(pedido.op).replace(/[^a-z-]/g, "") + ".json";
}

// O comando que solta essas linhas, uma por linha.
function comandoDeExemplo(pasta, pedido) {
    return ["jq", "-c", ".[]", pasta + "/" + arquivoDeExemplo(pedido)];
}
