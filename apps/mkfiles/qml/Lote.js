.pragma library
.import "Arquivos.js" as Arquivos

// A conta do renomear em lote: dos itens selecionados e das opções da folha
// sai, para cada um, o nome novo, e os conflitos (nome repetido no lote, nome
// que já existe na pasta, nome inválido). A janela mostra antes → depois ao
// vivo; o motor recebe só os pares [caminho, nome novo] e renomeia tudo como
// uma tarefa só.
//
// As opções se combinam, nesta ordem: o modelo monta o nome (sem ele, vale o
// nome de agora); depois trocar texto, aparar, tirar acentos, espaços, caixa,
// tudo sobre o nome sem a extensão; por fim a extensão.

function novo() {
    return {
        de: "", para: "", curinga: false, caixaExata: false,
        modelo: "", inicio: 1, passo: 1,
        caixa: "", acentos: false, espacos: "", aparar: false,
        extensao: null
    };
}

var CAIXAS = [{ v: "", texto: "como está" }, { v: "maiusculas", texto: "MAIÚSCULAS" }, { v: "minusculas", texto: "minúsculas" }, { v: "primeira", texto: "Primeira Maiúscula" }];
var ESPACOS = [{ v: "", texto: "como está" }, { v: "_", texto: "_" }, { v: "-", texto: "-" }];

function dois(n) { return n < 10 ? "0" + n : String(n); }

// {data:AAAA-MM-DD}: AAAA, AA, MM, DD, HH, mm, ss; o resto fica como está.
function formatarData(segundos, formato) {
    var d = new Date((segundos || 0) * 1000);
    var campos = {
        AAAA: String(d.getFullYear()), AA: dois(d.getFullYear() % 100), MM: dois(d.getMonth() + 1), DD: dois(d.getDate()),
        HH: dois(d.getHours()), mm: dois(d.getMinutes()), ss: dois(d.getSeconds())
    };
    return String(formato || "AAAA-MM-DD").replace(/AAAA|AA|MM|DD|HH|mm|ss/g, function (c) { return campos[c]; });
}

// O número do item i na numeração: {n} ou {n:3} (com zeros à esquerda).
function numero(i, inicio, passo, casas) {
    var n = (Number(inicio) || 0) + i * (Number(passo) || 0);
    var s = String(Math.abs(n));
    while (s.length < (casas || 0))
        s = "0" + s;
    return (n < 0 ? "-" : "") + s;
}

// Monta um nome pelo modelo. Campos: {nome} (sem a extensão), {ext} (sem o
// ponto), {n} e {n:3}, {data} e {data:FORMATO}, {pai}. Um campo desconhecido
// fica como está. Devolve { base, ext }: se o modelo não fala da extensão, a
// do arquivo continua.
function doModelo(modelo, item, i, o) {
    var p = Arquivos.partes(item.nome, Arquivos.ePasta(item));
    var falouDaExtensao = false;
    var texto = String(modelo).replace(/\{([a-z]+)(?::([^}]*))?\}/g, function (tudo, campo, arg) {
        if (campo === "nome")
            return p[0];
        if (campo === "ext") {
            falouDaExtensao = true;
            return p[1].substring(1);
        }
        if (campo === "n")
            return numero(i, o.inicio, o.passo, /^\d{1,2}$/.test(arg || "") ? Number(arg) : 0);
        if (campo === "data")
            return formatarData(item.mtime, arg);
        if (campo === "pai")
            return Arquivos.nomeDe(Arquivos.pai(item.caminho));
        return tudo;
    });
    if (falouDaExtensao) {
        // O nome inteiro veio do modelo: um ponto sobrando no fim (item sem
        // extensão em "{nome}.{ext}") sai.
        return { base: texto.replace(/\.$/, ""), ext: "" };
    }
    return { base: texto, ext: p[1] };
}

// Trocar texto: literal, ou com * (qualquer trecho) e ? (uma letra).
function trocar(texto, de, para, curinga, caixaExata) {
    if (de === "")
        return texto;
    var fonte = String(de).replace(/[.+^${}()|[\]\\\/-]/g, "\\$&");
    fonte = fonte.replace(/[*?]/g, function (c) { return !curinga ? "\\" + c : c === "*" ? ".*?" : "."; });
    // Um padrão que termina em * leva o resto do nome.
    if (curinga && /\*$/.test(de))
        fonte = fonte.replace(/\.\*\?$/, ".*");
    var achado = false;
    return texto.replace(new RegExp(fonte, caixaExata ? "g" : "gi"), function (m) {
        // Um padrão que casa com o vazio ("*") troca uma vez só.
        if (m === "" && achado)
            return m;
        achado = true;
        return String(para);
    });
}

function tirarAcentos(texto) {
    var saida = "";
    for (var i = 0; i < texto.length; i++) {
        var c = texto.charAt(i);
        var min = c.toLowerCase();
        var sem = Arquivos.ACENTOS[min];
        if (sem === undefined)
            saida += c;
        else
            saida += c === min ? sem : sem.toUpperCase();
    }
    return saida;
}

function comCaixa(texto, caixa) {
    if (caixa === "maiusculas")
        return texto.toUpperCase();
    if (caixa === "minusculas")
        return texto.toLowerCase();
    if (caixa === "primeira")
        return texto.toLowerCase().replace(/(^|[\s_\-.(\[])([^\s_\-.(\[])/g, function (m, antes, letra) { return antes + letra.toUpperCase(); });
    return texto;
}

// O nome novo de um item (o i-ésimo do lote).
function nomeNovo(item, i, o) {
    var pasta = Arquivos.ePasta(item);
    var p = Arquivos.partes(item.nome, pasta);
    var base = p[0], ext = p[1];
    if (o.modelo !== undefined && String(o.modelo).trim() !== "") {
        var m = doModelo(o.modelo, item, i, o);
        base = m.base;
        ext = m.ext;
    }
    base = trocar(base, o.de || "", o.para || "", o.curinga === true, o.caixaExata === true);
    if (o.acentos === true)
        base = tirarAcentos(base);
    if (o.aparar === true)
        base = base.replace(/\s+/g, " ").trim();
    if (o.espacos === "_" || o.espacos === "-")
        base = base.replace(/\s+/g, o.espacos);
    base = comCaixa(base, o.caixa);
    // Mudar só a extensão (pasta não tem): "" tira; "txt" e ".txt" dão ".txt".
    if (!pasta && o.extensao !== null && o.extensao !== undefined) {
        var e = String(o.extensao).trim().replace(/^\.+/, "");
        ext = e === "" ? "" : "." + e;
    }
    return base + ext;
}

// Um nome serve? "" ou a frase (as regras do motor, mais o nome que só tem
// espaços).
function erroDoNome(nome) {
    if (nome.trim() === "")
        return "o nome não pode ser vazio";
    if (nome.indexOf("\u0000") >= 0)
        return "nome com byte nulo";
    return Arquivos.erroDoNome(nome);
}

var MOTIVOS = { repetido: "repetido no lote", existe: "já existe na pasta", invalido: "nome inválido" };

// A conta inteira. itens: os selecionados (na ordem em que aparecem);
// existentes: { pasta, nomes } com o que há na pasta do painel (null: não se
// sabe, como nos resultados de uma busca; o motor confere). Devolve
// { linhas: [{ caminho, antes, depois, muda, motivo, frase }], pares:
// [[caminho, depois]] (só os que mudam), mudam, conflitos }.
function calcular(itens, o, existentes) {
    var linhas = [];
    var finais = {};
    var doLote = {};
    var i, chave;
    for (i = 0; i < itens.length; i++) {
        var depois = nomeNovo(itens[i], i, o);
        var pasta = Arquivos.pai(itens[i].caminho);
        linhas.push({ caminho: itens[i].caminho, pasta: pasta, antes: itens[i].nome, depois: depois, muda: depois !== itens[i].nome, motivo: "", frase: "" });
        doLote[pasta + "/" + itens[i].nome] = true;
        chave = pasta + "/" + depois;
        finais[chave] = (finais[chave] || 0) + 1;
    }
    // O que fica na pasta sem ser do lote ocupa o nome que tem.
    var ocupados = {};
    if (existentes && existentes.nomes)
        for (i = 0; i < existentes.nomes.length; i++)
            if (doLote[existentes.pasta + "/" + existentes.nomes[i]] !== true)
                ocupados[existentes.pasta + "/" + existentes.nomes[i]] = true;
    var pares = [];
    var conflitos = 0;
    for (i = 0; i < linhas.length; i++) {
        var l = linhas[i];
        chave = l.pasta + "/" + l.depois;
        var erro = erroDoNome(l.depois);
        if (erro !== "") {
            l.motivo = "invalido";
            l.frase = erro;
        } else if (finais[chave] > 1) {
            l.motivo = "repetido";
        } else if (l.muda && ocupados[chave] === true) {
            l.motivo = "existe";
        }
        if (l.motivo !== "") {
            conflitos++;
            if (l.frase === "")
                l.frase = MOTIVOS[l.motivo];
        } else if (l.muda) {
            pares.push([l.caminho, l.depois]);
        }
    }
    return { linhas: linhas, pares: pares, mudam: pares.length, conflitos: conflitos };
}

// O motor recusou o lote ({ erro, conflitos: [{ de, para, motivo }] }): a
// frase para a folha.
function recusa(r) {
    var c = r && r.conflitos instanceof Array && r.conflitos.length > 0 ? r.conflitos[0] : null;
    if (!c)
        return (r && r.erro) || "";
    return (r.erro || "o motor recusou o lote") + ": «" + c.para + "» (" + (MOTIVOS[c.motivo] || c.motivo) + ")";
}
