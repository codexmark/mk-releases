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
    ["Ctrl+3", "as máquinas (Enter liga, Ctrl+N cria)"],
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
        break;
    case "vm":
        switch (a[1]) {
        case "listar":
        case "requisitos":
        case "uso":
        case "criar":
            return "vm-" + a[1] + ".json";
        case "snapshot":
            return a[2] === "listar" ? "vm-snapshot-listar.json" : "vm-acao.json";
        case "ligar":
        case "desligar":
        case "pausar":
        case "continuar":
        case "remover":
        case "host":
            return "vm-acao.json";
        }
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

// ── as máquinas ────────────────────────────────────────────────────────
// O que a área Máquinas faz sozinha: os textos, a conta da CPU e a conferência
// da folha da máquina nova (as mesmas regras do `vm criar` do motor).

// O estado de uma máquina na bolinha dos hosts: cheia se ligada, anel de
// cobre se pausada, apagada se travada, anel fino se desligada.
function bolinhaDaMaquina(estado) {
    return ({ ligada: "responde", pausada: "salto", travada: "nao" })[estado] || "desconhecido";
}

function haQuanto(desde, agora) {
    if (typeof desde !== "number" || typeof agora !== "number" || !isFinite(desde) || !isFinite(agora) || agora < desde)
        return "";
    var s = agora - desde;
    if (s < 60)
        return "há instantes";
    if (s < 3600)
        return "há " + Math.floor(s / 60) + " min";
    if (s < 48 * 3600)
        return "há " + Math.floor(s / 3600) + " h";
    return "há " + Math.floor(s / 86400) + " d";
}

function textoDoEstadoDaMaquina(m, agora) {
    if (!m)
        return "";
    if (m.estado === "ligada") {
        var ha = haQuanto(m.desde, agora);
        return ha === "" ? "ligada" : "ligada " + ha;
    }
    if (m.estado === "travada")
        return "travada: o QEMU está de pé, mas não responde";
    return String(m.estado || "");
}

function memoria(mb) {
    if (typeof mb !== "number" || !isFinite(mb) || mb < 0)
        return "—";
    if (mb < 1024)
        return mb + " MB";
    return (mb % 1024 === 0 ? String(mb / 1024) : (mb / 1024).toFixed(1).replace(".", ",")) + " GB";
}

function tamanho(bytes) {
    if (typeof bytes !== "number" || !isFinite(bytes) || bytes < 0)
        return "—";
    var unidades = ["B", "KB", "MB", "GB", "TB"], i = 0, v = bytes;
    while (v >= 1024 && i < unidades.length - 1) {
        v /= 1024;
        i++;
    }
    if (i === 0)
        return v + " B";
    return (v < 10 ? v.toFixed(1).replace(".", ",") : String(Math.round(v))) + " " + unidades[i];
}

function resumoDaMaquina(m) {
    if (!m)
        return "";
    return memoria(m.memoria_mb) + " · " + plural(m.cpus, "CPU", "CPUs");
}

function resumoDasMaquinas(maquinas) {
    var l = maquinas || [], ligadas = 0, pausadas = 0;
    if (l.length === 0)
        return "nenhuma máquina";
    for (var i = 0; i < l.length; i++) {
        if (l[i].estado === "ligada")
            ligadas++;
        else if (l[i].estado === "pausada")
            pausadas++;
    }
    var texto = plural(l.length, "máquina", "máquinas");
    if (ligadas > 0)
        texto += " · " + plural(ligadas, "ligada", "ligadas");
    if (pausadas > 0)
        texto += " · " + plural(pausadas, "pausada", "pausadas");
    return texto;
}

// A porcentagem de CPU entre duas respostas do `vm uso` (100 é uma CPU
// cheia). null quando não dá para saber: falta uma leitura, o processo é
// outro (a máquina religou), o relógio não andou ou o contador voltou.
function cpuPct(antes, agora) {
    if (!antes || !agora || antes.pid !== agora.pid)
        return null;
    var campos = ["ticks", "hz", "agora_ms"];
    for (var i = 0; i < campos.length; i++)
        if (typeof antes[campos[i]] !== "number" || typeof agora[campos[i]] !== "number")
            return null;
    var ms = agora.agora_ms - antes.agora_ms, ticks = agora.ticks - antes.ticks;
    if (ms <= 0 || ticks < 0 || agora.hz <= 0)
        return null;
    return (ticks / agora.hz) / (ms / 1000) * 100;
}

function textoDoUso(pct, rssKb) {
    var cpu = typeof pct === "number" && isFinite(pct) ? Math.round(pct) + "%" : "—";
    var mem = typeof rssKb === "number" && isFinite(rssKb) && rssKb >= 0 ? tamanho(rssKb * 1024) : "—";
    return "CPU " + cpu + " · memória " + mem;
}

function erroDoNomeDaMaquina(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v === "")
        return "falta o nome da máquina";
    if (v.length > 40)
        return "o nome é comprido demais (até 40)";
    if (!/^[a-z0-9][a-z0-9._-]{0,39}$/.test(v) || /[\n\r]/.test(v))
        return "o nome só pode ter letras minúsculas sem acento, algarismos, ponto, hífen e sublinhado, começando por letra ou algarismo";
    return "";
}

// Um inteiro escrito só com algarismos, entre os dois limites.
function erroDeInteiro(valor, minimo, maximo, frase) {
    var v = String(valor === null || valor === undefined ? "" : valor).trim();
    if (!/^[0-9]{1,7}$/.test(v) || parseInt(v, 10) < minimo || parseInt(v, 10) > maximo)
        return frase;
    return "";
}

function erroDaMemoria(valor) {
    return erroDeInteiro(valor, 512, 65536, "a memória vai de 512 a 65536 MB");
}

function erroDasCpus(valor) {
    return erroDeInteiro(valor, 1, 32, "as CPUs vão de 1 a 32");
}

function erroDoDisco(valor) {
    return erroDeInteiro(valor, 1, 2000, "o disco vai de 1 a 2000 GB");
}

// O caminho da ISO, sem olhar o disco (se o arquivo existe, quem diz é o motor).
function erroDaIso(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v === "")
        return "";
    for (var i = 0; i < v.length; i++) {
        var c = v.charCodeAt(i);
        if (c === 10 || c === 13)
            return "o caminho da ISO não pode ter quebra de linha";
        if (c < 32 || c === 127)
            return "o caminho da ISO não pode ter caractere de controle";
        if (c === 44)
            return "o caminho da ISO não pode ter vírgula (o QEMU a lê como separador de opções): renomeie o arquivo";
    }
    if (v.charAt(0) !== "/")
        return "o caminho da ISO tem de ser inteiro, começando por /";
    if (v.length > 4096)
        return "o caminho da ISO é comprido demais";
    return "";
}

function erroDoRotulo(valor) {
    var v = String(valor === null || valor === undefined ? "" : valor);
    if (v === "")
        return "falta o rótulo";
    if (!/^[A-Za-z0-9._][A-Za-z0-9._-]{0,39}$/.test(v) || /[\n\r]/.test(v))
        return "o rótulo só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado (até 40), sem começar com hífen";
    return "";
}

function maquinaEscolhida(maquinas, nome) {
    var l = maquinas || [];
    for (var i = 0; i < l.length; i++)
        if (l[i].nome === nome)
            return l[i];
    return null;
}

function maquinaVizinha(maquinas, nome, passo) {
    return vizinho((maquinas || []).map(function (m) { return { apelido: m.nome }; }), nome, passo);
}

// Os erros da folha da máquina nova. `requisitos` é a resposta do
// `vm requisitos` (para o UEFI, que precisa do OVMF).
function validarMaquina(form, maquinas, requisitos) {
    var nome = aparado(form, "nome");
    var erros = {
        nome: erroDoNomeDaMaquina(nome),
        memoria_mb: erroDaMemoria(aparado(form, "memoria_mb")),
        cpus: erroDasCpus(aparado(form, "cpus")),
        disco_gb: erroDoDisco(aparado(form, "disco_gb")),
        iso: erroDaIso(aparado(form, "iso")),
        uefi: ""
    };
    if (erros.nome === "" && maquinaEscolhida(maquinas, nome))
        erros.nome = "já existe uma máquina com esse nome";
    if (form && form.uefi === true && requisitos && requisitos.ovmf && requisitos.ovmf.tem === false)
        erros.uefi = "UEFI precisa do OVMF: instale o pacote edk2-ovmf";
    return erros;
}

function maquinaValida(erros) {
    if (!erros)
        return false;
    var campos = ["nome", "memoria_mb", "cpus", "disco_gb", "iso", "uefi"];
    for (var i = 0; i < campos.length; i++)
        if (erros[campos[i]] !== "")
            return false;
    return true;
}

// A folha em branco: UEFI ligado só se o OVMF está instalado.
function maquinaVazia(requisitos) {
    return {
        nome: "", iso: "", memoria_mb: "4096", cpus: "2", disco_gb: "40",
        uefi: !!(requisitos && requisitos.ovmf && requisitos.ovmf.tem === true), tres_d: false
    };
}

// O objeto que vai (como JSON) para a entrada do `vm criar`.
function pedidoDeMaquina(form) {
    var inteiro = function (campo) {
        var v = aparado(form, campo);
        return /^[0-9]{1,7}$/.test(v) ? parseInt(v, 10) : 0;
    };
    return {
        nome: aparado(form, "nome"), memoria_mb: inteiro("memoria_mb"), cpus: inteiro("cpus"), disco_gb: inteiro("disco_gb"),
        iso: aparado(form, "iso"), uefi: !!(form && form.uefi === true), tres_d: !!(form && form.tres_d === true)
    };
}

// O que falta instalar, pelo `vm requisitos`.
function faltas(requisitos) {
    var l = [];
    if (!requisitos)
        return l;
    if (requisitos.qemu && requisitos.qemu.tem === false)
        l.push("o QEMU (qemu-system-x86_64)");
    if (requisitos.qemu_img && requisitos.qemu_img.tem === false)
        l.push("o qemu-img");
    return l;
}

// O que o estado da máquina deixa fazer.
function podeNaMaquina(m, acao) {
    if (!m)
        return false;
    var e = m.estado;
    switch (acao) {
    case "ligar":
    case "snapshots":
    case "remover":
        return e === "desligada";
    case "desligar":
        return e === "ligada" || e === "pausada";
    case "forcar":
        return e === "ligada" || e === "pausada" || e === "travada";
    case "pausar":
        return e === "ligada";
    case "continuar":
        return e === "pausada";
    case "host":
        return true;
    }
    return false;
}

// Por que os snapshots e o remover estão fechados ("" se não estão).
function porQueNao(m, acao) {
    if (!m || podeNaMaquina(m, acao) || (acao !== "snapshots" && acao !== "remover"))
        return "";
    return acao + " só com a máquina desligada";
}

// O seletor de arquivos devolve uma URL (file:///…): aqui vira um caminho.
// O que não é um arquivo daqui vira "".
function caminhoDeUrl(url) {
    var u = String(url === null || url === undefined ? "" : url);
    if (u.charAt(0) === "/")
        return u;
    if (u.indexOf("file://") !== 0)
        return "";
    try {
        return decodeURIComponent(u.slice(7));
    } catch (e) {
        return "";
    }
}
