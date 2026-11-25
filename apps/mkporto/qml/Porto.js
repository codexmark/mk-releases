.pragma library

// A lógica da janela do MkPorto que não depende do QML (testada no node:
// tests/apps/mkporto-Porto.test.mjs). A janela só mostra o que o motor diz; o
// que ela faz sozinha (procurar, ordenar, a bolinha de cada host, conferir o
// formulário enquanto se digita) mora aqui. As regras do formulário são as
// mesmas do motor (../CONTRATO.md): ele confere de novo ao guardar.

// Abaixo desta largura o detalhe não cabe ao lado da lista.
var LARGURA_DE_DUAS_COLUNAS = 720;
var LARGURA_APERTADA = 400;

// A folha de atalhos (F1): [tecla, o que faz].
var ATALHOS = [
    ["Enter", "conectar ao host escolhido"],
    ["↑  ↓", "andar na lista"],
    ["Ctrl+F", "procurar"],
    ["Ctrl+N", "novo host"],
    ["Ctrl+E", "editar o host (só os do MkPorto)"],
    ["Ctrl+K", "enviar uma chave pública ao host"],
    ["Delete", "remover o host (só os do MkPorto)"],
    ["Ctrl+Shift+C", "copiar o comando (ssh apelido)"],
    ["Ctrl+T", "testar a entrada (sem pedir senha)"],
    ["F5", "sondar de novo (nas chaves, reler)"],
    ["Ctrl+G", "nova chave (na área Chaves)"],
    ["Ctrl+1", "os hosts"],
    ["Ctrl+2", "as chaves"],
    ["Esc", "fechar o que estiver aberto"],
    ["F1", "esta folha"]
];

var CAMPOS = ["apelido", "host", "usuario", "porta", "chave", "salto"];

// ── textos ─────────────────────────────────────────────────────────────

// Sem maiúsculas nem acentos: é assim que a busca compara.
function fold(s) {
    if (s === null || s === undefined)
        return "";
    return String(s).toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "");
}

function plural(n, um, muitos) {
    return n + " " + (n === 1 ? um : muitos);
}

// usuario@host:porta, do jeito que se leria numa linha de comando.
function linha(h) {
    if (!h)
        return "";
    var nome = String(h.host || h.apelido || "");
    if (nome.indexOf(":") >= 0)
        nome = "[" + nome + "]";
    return (h.usuario ? h.usuario + "@" : "") + nome + ":" + (h.porta || 22);
}

function origem(h) {
    return h && h.origem === "mkporto" ? "MkPorto" : "o seu ~/.ssh/config";
}

// Só o que o MkPorto guardou dá para editar e remover.
function editavel(h) {
    return !!h && h.origem === "mkporto";
}

function latencia(ms) {
    if (typeof ms !== "number" || !isFinite(ms) || ms < 0)
        return "—";
    if (ms < 1)
        return "<1 ms";
    if (ms < 1000)
        return Math.round(ms) + " ms";
    return (ms / 1000).toFixed(1).replace(".", ",") + " s";
}

function tipoEBits(chave) {
    if (!chave)
        return "";
    return String(chave.tipo || "") + (typeof chave.bits === "number" ? " · " + chave.bits + " bits" : "");
}

// A chave privada que acompanha uma pública: o mesmo caminho sem o ".pub". É
// só o nome, para preencher o campo; o MkPorto nunca abre esse arquivo.
function sugestaoDeChave(chave) {
    var arquivo = chave && typeof chave.arquivo === "string" ? chave.arquivo : "";
    return /\.pub$/.test(arquivo) ? arquivo.slice(0, -4) : "";
}

// No modo de exemplo a janela não abre terminal: diz o que abriria.
function abriria(resposta) {
    if (!resposta || !Array.isArray(resposta.argv))
        return "exemplo: não abriria nada";
    return "exemplo: abriria um terminal com " + resposta.argv.join(" ");
}

// ── a lista ────────────────────────────────────────────────────────────

// Todos os termos da busca aparecem no apelido, no endereço, no usuário, na
// porta ou no salto.
function casa(h, busca) {
    if (!h)
        return false;
    var onde = fold([h.apelido, h.host, h.usuario, h.porta, h.salto].join(" "));
    var termos = fold(busca).split(/\s+/);
    for (var i = 0; i < termos.length; i++)
        if (termos[i] !== "" && onde.indexOf(termos[i]) < 0)
            return false;
    return true;
}

function filtrar(hosts, busca) {
    return (hosts || []).filter(function (h) { return casa(h, busca); });
}

// Compara dois textos com os números valendo como números (web2 antes de web10).
function compara(a, b) {
    var pa = a.match(/\d+|\D+/g) || [], pb = b.match(/\d+|\D+/g) || [];
    for (var i = 0; i < pa.length && i < pb.length; i++) {
        if (pa[i] === pb[i])
            continue;
        if (/^\d/.test(pa[i]) && /^\d/.test(pb[i])) {
            var d = parseInt(pa[i], 10) - parseInt(pb[i], 10);
            if (d !== 0)
                return d;
            continue;
        }
        return pa[i] < pb[i] ? -1 : 1;
    }
    return pa.length - pb.length;
}

// Por apelido, sem ligar para maiúscula nem acento. Devolve uma lista nova.
function ordenar(hosts) {
    return (hosts || []).slice().sort(function (a, b) {
        var d = compara(fold(a.apelido), fold(b.apelido));
        if (d !== 0)
            return d;
        return a.apelido < b.apelido ? -1 : a.apelido > b.apelido ? 1 : 0;
    });
}

function visiveis(hosts, busca) {
    return ordenar(filtrar(hosts, busca));
}

function escolhido(hosts, apelido) {
    var lista = hosts || [];
    for (var i = 0; i < lista.length; i++)
        if (lista[i].apelido === apelido)
            return lista[i];
    return null;
}

// O apelido de quem está `passo` linhas adiante (ou atrás) do escolhido, sem
// sair da lista. Sem escolhido na lista: o primeiro (descendo) ou o último.
function vizinho(lista, apelido, passo) {
    var l = lista || [];
    if (l.length === 0)
        return "";
    var i = -1;
    for (var k = 0; k < l.length; k++)
        if (l[k].apelido === apelido)
            i = k;
    if (i < 0)
        return l[passo < 0 ? l.length - 1 : 0].apelido;
    return l[Math.max(0, Math.min(l.length - 1, i + passo))].apelido;
}

// ── as sondas ──────────────────────────────────────────────────────────
// O mapa das sondas é { apelido: { sondando, responde, ms, motivo } }. As
// funções devolvem um mapa novo (é assim que o QML percebe a mudança).

function copia(sondas) {
    var novo = {}, k;
    for (k in (sondas || {}))
        novo[k] = sondas[k];
    return novo;
}

function comecaSonda(sondas, apelidos) {
    var novo = copia(sondas), l = apelidos || [];
    for (var i = 0; i < l.length; i++)
        novo[l[i]] = { sondando: true };
    return novo;
}

// Uma linha do `sondar`. O que não é um resultado é ignorado.
function aplicaSonda(sondas, r) {
    if (!r || typeof r.apelido !== "string")
        return sondas;
    var novo = copia(sondas);
    novo[r.apelido] = {
        sondando: false,
        responde: r.responde === true ? true : r.responde === false ? false : null,
        ms: typeof r.ms === "number" ? r.ms : null,
        motivo: typeof r.motivo === "string" ? r.motivo : null,
        servidor: typeof r.servidor === "string" ? r.servidor : null,
        via: typeof r.via === "string" ? r.via : null
    };
    return novo;
}

// O `sondar` acabou: quem ainda estava "sondando" (o motor parou sem dizer
// dele) volta a "ainda não sondado".
function encerraSonda(sondas) {
    var novo = {}, k;
    for (k in (sondas || {}))
        if (!sondas[k].sondando)
            novo[k] = sondas[k];
    return novo;
}

// "responde" | "nao" | "salto" | "sondando" | "desconhecido". Um host por
// salto é sondado pelo salto; se a sonda não pôde ir (ou ainda não foi), é "salto".
function estado(h, sondas) {
    if (!h)
        return "desconhecido";
    var s = sondas ? sondas[h.apelido] : null, semSonda = h.salto ? "salto" : "desconhecido";
    if (!s)
        return semSonda;
    if (s.sondando)
        return "sondando";
    return s.responde === true ? "responde" : s.responde === false ? "nao" : semSonda;
}

function textoDoEstado(h, sondas) {
    var s = h && sondas ? sondas[h.apelido] : null;
    switch (estado(h, sondas)) {
    case "responde":
        return "responde " + (s.via ? "por " + s.via + " " : "") + "em " + latencia(s.ms);
    case "nao":
        return "não responde" + (s.motivo ? ": " + s.motivo : "");
    case "salto":
        return "por salto: só se chega nele por " + h.salto;
    case "sondando":
        return "sondando…";
    }
    return "ainda não sondado";
}

function resumo(hosts, sondas) {
    var l = hosts || [], sim = 0, nao = 0, sondando = false;
    if (l.length === 0)
        return "nenhum host";
    for (var i = 0; i < l.length; i++) {
        var e = estado(l[i], sondas);
        if (e === "responde")
            sim++;
        else if (e === "nao")
            nao++;
        else if (e === "sondando")
            sondando = true;
    }
    var texto = plural(l.length, "host", "hosts");
    if (sondando)
        return texto + " · sondando…";
    if (sim > 0)
        texto += " · " + sim + (sim === 1 ? " responde" : " respondem");
    if (nao > 0)
        texto += " · " + nao + (nao === 1 ? " não responde" : " não respondem");
    return texto;
}

// ── o formulário ───────────────────────────────────────────────────────
// As mesmas regras do `guardar` do motor. Cada função devolve a frase do erro
// ou "" se o valor serve.

// Um valor que vai numa linha do config como uma palavra só.
function erroDePalavra(valor, oQue, maximo) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v.charAt(0) === "-")
        return oQue + " não pode começar com \"-\"";
    if (v.length > maximo)
        return oQue + " é comprido demais (até " + maximo + ")";
    if (/[\n\r]/.test(v))
        return oQue + " não pode ter quebra de linha";
    if (/[\u0000-\u001f\u007f]/.test(v))
        return oQue + " não pode ter caractere de controle";
    if (/[ \u00a0]/.test(v))
        return oQue + " não pode ter espaço";
    var m = /["#]/.exec(v);
    if (m)
        return oQue + " não pode ter " + m[0];
    return "";
}

function erroDoApelido(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v === "")
        return "falta o apelido";
    if (v.charAt(0) === "-")
        return "o apelido não pode começar com \"-\"";
    if (/[\n\r]/.test(v))
        return "o apelido não pode ter quebra de linha";
    if (v.length > 64)
        return "o apelido é comprido demais (até 64)";
    if (!/^[A-Za-z0-9._][A-Za-z0-9._-]*$/.test(v))
        return "o apelido só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado";
    return "";
}

function erroDoHost(valor) {
    if (valor === "" || valor === null || valor === undefined)
        return "falta o endereço";
    return erroDePalavra(valor, "o endereço", 253);
}

function erroDoUsuario(valor) {
    return erroDePalavra(valor, "o usuário", 64);
}

function erroDoSalto(valor) {
    return erroDePalavra(valor, "o salto", 1024);
}

// Vazio é a porta 22.
function erroDaPorta(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor).trim();
    if (v === "")
        return "";
    if (!/^[0-9]{1,5}$/.test(v) || parseInt(v, 10) < 1 || parseInt(v, 10) > 65535)
        return "a porta vai de 1 a 65535";
    return "";
}

function erroDaChave(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v.charAt(0) === "-")
        return "a chave não pode começar com \"-\"";
    if (v.length > 4096)
        return "o caminho da chave é comprido demais";
    if (/[\n\r]/.test(v))
        return "a chave não pode ter quebra de linha";
    if (/[\u0000-\u001f\u007f]/.test(v))
        return "a chave não pode ter caractere de controle";
    var m = /["\\]/.exec(v);
    if (m)
        return "a chave não pode ter " + m[0];
    return "";
}

function aparado(form, campo) {
    var v = form ? form[campo] : "";
    return String(v === null || v === undefined ? "" : v).trim();
}

// Os erros de cada campo do formulário ("" onde está certo). `hosts` são os
// que já existem; `anterior`, o apelido de quem está sendo editado ("" num
// host novo).
function validar(form, hosts, anterior) {
    var erros = {
        apelido: erroDoApelido(aparado(form, "apelido")),
        host: erroDoHost(aparado(form, "host")),
        usuario: erroDoUsuario(aparado(form, "usuario")),
        porta: erroDaPorta(aparado(form, "porta")),
        chave: erroDaChave(aparado(form, "chave")),
        salto: erroDoSalto(aparado(form, "salto"))
    };
    var apelido = aparado(form, "apelido");
    if (erros.apelido === "" && apelido !== anterior) {
        var ja = escolhido(hosts, apelido);
        if (ja)
            erros.apelido = editavel(ja) ? "já existe um host com esse apelido" : apelido + " já está no seu ~/.ssh/config";
    }
    return erros;
}

function valido(erros) {
    if (!erros)
        return false;
    for (var i = 0; i < CAMPOS.length; i++)
        if (erros[CAMPOS[i]] !== "")
            return false;
    return true;
}

function formularioVazio() {
    return { apelido: "", host: "", usuario: "", porta: "22", chave: "", salto: "" };
}

function formularioDe(h) {
    if (!h)
        return formularioVazio();
    // O que está escrito no arquivo do app, quando o motor diz: o que o ssh
    // completou (o usuário local, um Host *) não é para ser gravado no host.
    var e = h.escrito || h;
    return {
        apelido: String(h.apelido || ""),
        host: String(e.host || ""),
        usuario: String(e.usuario || ""),
        porta: String(e.porta || 22),
        chave: String(e.chave || ""),
        salto: String(e.salto || "")
    };
}

// O objeto que vai (como JSON) para a entrada do `guardar`.
function pedido(form, anterior) {
    var porta = aparado(form, "porta");
    return {
        apelido: aparado(form, "apelido"),
        host: aparado(form, "host"),
        usuario: aparado(form, "usuario"),
        porta: porta === "" ? 22 : parseInt(porta, 10),
        chave: aparado(form, "chave"),
        salto: aparado(form, "salto"),
        anterior: String(anterior || "")
    };
}

// ── a janela ───────────────────────────────────────────────────────────

function estreita(largura) {
    return largura < LARGURA_DE_DUAS_COLUNAS;
}

// Numa coluna de um terço de tela nem o alto cabe inteiro: a marca e o
// "sondar" saem (o F5 continua) para o "novo" não ficar cortado.
function apertada(largura) {
    return largura < LARGURA_APERTADA;
}

// O arquivo de resposta gravada (MKPORTO_EXEMPLO) de um pedido ao motor. Só
// os nomes que existem: nada do que a pessoa digitou entra no caminho.
function arquivoDeExemplo(args) {
    var a = args || [];
    switch (a[0]) {
    case "listar":
    case "sondar":
    case "chaves":
    case "guardar":
    case "remover":
    case "entrar":
        return a[0] + ".json";
    case "comando":
        if (a[1] === "conectar" || a[1] === "chave" || a[1] === "gerar")
            return "comando-" + a[1] + ".json";
    }
    return "nada.json";
}

// ── os valores, a entrada e a chave nova ───────────────────────────────

// De onde vêm os valores de um host: do próprio ssh (`ssh -G`) ou, se ele não
// respondeu, do leitor do config.
function fonteDosValores(h) {
    if (!h)
        return "";
    return h.efetivo === true ? "como o ssh os vê" : "lidos do config (o ssh não respondeu)";
}

// A linha do "testar entrada" no detalhe. `entrada` é { apelido, testando }
// enquanto o motor não volta, e depois a resposta do `entrar` (ou { apelido,
// erro }). Só aparece no host a que pertence.
function textoDaEntrada(entrada, h) {
    if (!entrada || !h || entrada.apelido !== h.apelido)
        return "";
    if (entrada.testando)
        return "testando a entrada…";
    if (entrada.erro)
        return String(entrada.erro);
    if (entrada.entra === true)
        return "entra: a máquina aceitou uma das suas chaves (" + latencia(entrada.ms) + ")";
    return "não entra" + (entrada.motivo ? ": " + entrada.motivo : "");
}

// As mesmas regras do `comando gerar` do motor.
function erroDoNomeDaChave(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v === "")
        return "falta o nome da chave";
    if (v.charAt(0) === "-")
        return "o nome não pode começar com \"-\"";
    if (v.charAt(0) === ".")
        return "o nome não pode começar com ponto";
    if (/[\n\r]/.test(v))
        return "o nome não pode ter quebra de linha";
    if (v.length > 64)
        return "o nome é comprido demais (até 64)";
    if (!/^[A-Za-z0-9_][A-Za-z0-9._-]*$/.test(v))
        return "o nome só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado";
    if (/\.pub$/.test(v))
        return "o nome não leva .pub: esse é o da metade pública, que nasce junto";
    return "";
}

function erroDoComentario(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v.charAt(0) === "-")
        return "o comentário não pode começar com \"-\"";
    if (Array.from(v).length > 128)
        return "o comentário é comprido demais (até 128)";
    if (/[\n\r]/.test(v))
        return "o comentário não pode ter quebra de linha";
    if (/[\u0000-\u001f\u007f]/.test(v))
        return "o comentário não pode ter caractere de controle";
    return "";
}

// Os erros da folha da chave nova. `chaves` são as públicas que já existem
// (o motor confere também a privada, que a janela não vê).
function validarChave(form, chaves) {
    var nome = aparado(form, "nome");
    var erros = { nome: erroDoNomeDaChave(nome), comentario: erroDoComentario(aparado(form, "comentario")) };
    var l = chaves || [];
    for (var i = 0; erros.nome === "" && i < l.length; i++)
        if (l[i].nome === nome + ".pub")
            erros.nome = "já existe uma chave com esse nome";
    return erros;
}

function chaveValida(erros) {
    return !!erros && erros.nome === "" && erros.comentario === "";
}

// O que vai para o motor depois de `comando gerar --`: o nome e, se houver, o
// comentário (sem ele, o motor põe usuario@maquina).
function argsDaChave(form) {
    var comentario = aparado(form, "comentario");
    return comentario === "" ? [aparado(form, "nome")] : [aparado(form, "nome"), comentario];
}
