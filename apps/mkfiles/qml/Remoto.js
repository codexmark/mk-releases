.pragma library

// A lógica pura dos lugares de rede: montar o endereço a partir do formulário,
// reconhecer um caminho que mora num remoto montado (o ponto FUSE do gvfs),
// as migalhas e o endereço dele, as linhas da seção "Rede" da barra lateral e
// os textos das perguntas que o `gio` faz. Quem monta e desmonta é o motor
// (../CONTRATO.md, "Remotos"); a senha nunca passa por aqui.

// Os tipos do formulário de conectar: { v, texto, porta padrão, com usuário }.
var TIPOS = [
    { v: "sftp", texto: "SFTP", porta: 22, usuario: true },
    { v: "smb", texto: "Windows", porta: 445, usuario: true },
    { v: "ftp", texto: "FTP", porta: 21, usuario: true },
    { v: "dav", texto: "WebDAV", porta: 80, usuario: true },
    { v: "davs", texto: "DAV seguro", porta: 443, usuario: true }
];

// Os esquemas que o motor aceita montar.
var ESQUEMAS = ["sftp", "smb", "ftp", "ftps", "dav", "davs", "mtp", "gphoto2", "nfs", "afp"];

function tipo(v) {
    for (var i = 0; i < TIPOS.length; i++)
        if (TIPOS[i].v === v)
            return TIPOS[i];
    return TIPOS[0];
}

// Codifica um pedaço de endereço (usuário, um nome de pasta): o que não é
// seguro vira %XX.
function codificar(texto) {
    return encodeURIComponent(String(texto)).replace(/[!'()*]/g, function (c) {
        return "%" + c.charCodeAt(0).toString(16).toUpperCase();
    });
}

// A pasta de um endereço: cada nome codificado, as barras ficam. Sempre
// começa com "/".
function codificarPasta(pasta) {
    var partes = String(pasta || "").split("/").filter(function (p) { return p !== "" && p !== "."; });
    return "/" + partes.map(codificar).join("/");
}

// O que está errado num host digitado ("" se nada).
function erroDoHost(host) {
    var h = String(host || "").trim();
    if (h === "")
        return "escreva o servidor";
    if (/[\s\/@?#\\]/.test(h) || /[\x00-\x1f\x7f]/.test(h) || h.charAt(0) === "-")
        return "o servidor é só o nome ou o número da máquina";
    return "";
}

// O que está errado numa porta digitada ("" se nada; vazia é a padrão).
function erroDaPorta(porta) {
    var p = String(porta === undefined || porta === null ? "" : porta).trim();
    if (p === "")
        return "";
    if (!/^[0-9]{1,5}$/.test(p) || Number(p) < 1 || Number(p) > 65535)
        return "a porta é um número de 1 a 65535";
    return "";
}

// O endereço que o formulário descreve: { tipo, host, porta, usuario, pasta,
// dominio }. null se o host ou a porta não servem. No SMB o domínio vai antes
// do usuário ("dominio;usuario"), e a primeira parte da pasta é o
// compartilhamento. Um host IPv6 ganha colchetes.
function montarUri(f) {
    f = f || {};
    var t = tipo(f.tipo);
    if (erroDoHost(f.host) !== "" || erroDaPorta(f.porta) !== "")
        return null;
    var host = String(f.host).trim();
    if (host.indexOf(":") >= 0 && host.charAt(0) !== "[")
        host = "[" + host + "]";
    var usuario = String(f.usuario || "").trim();
    var dominio = String(f.dominio || "").trim();
    var quem = "";
    if (usuario !== "")
        quem = (t.v === "smb" && dominio !== "" ? codificar(dominio) + ";" : "") + codificar(usuario) + "@";
    var porta = String(f.porta === undefined || f.porta === null ? "" : f.porta).trim();
    var comPorta = porta !== "" && Number(porta) !== t.porta ? ":" + Number(porta) : "";
    return t.v + "://" + quem + host + comPorta + codificarPasta(f.pasta);
}

// O que está errado num endereço digitado ("" se serve para o motor).
function erroDoEndereco(texto) {
    var t = String(texto || "").trim();
    if (t === "")
        return "escreva o endereço";
    if (/[\x00-\x1f\x7f\s]/.test(t))
        return "o endereço não pode ter espaço (use %20)";
    var m = /^([a-z][a-z0-9+.-]*):\/\/([^\/]*)/.exec(t);
    if (!m)
        return "o endereço começa com o tipo: sftp://, smb://, ftp://, dav://…";
    if (ESQUEMAS.indexOf(m[1]) < 0)
        return "não conheço o tipo «" + m[1] + "»";
    if (m[2] === "" || m[2].charAt(m[2].length - 1) === "@")
        return "falta o servidor";
    return "";
}

// O endereço de um host do ~/.ssh/config: o apelido basta (o ssh do sistema
// resolve o resto: nome, usuário, porta, chave).
function uriDoHost(h) {
    return h && h.apelido ? "sftp://" + codificar(h.apelido) + "/" : null;
}

// A linha de um host do ssh na lista: "apelido" e "usuario@host:porta".
function detalheDoHost(h) {
    if (!h)
        return "";
    var d = (h.usuario ? h.usuario + "@" : "") + (h.host || h.apelido || "");
    return h.porta && Number(h.porta) !== 22 ? d + ":" + h.porta : d;
}

function dentroDe(pai, filho) {
    return !!pai && (filho === pai || String(filho).indexOf(pai === "/" ? "/" : pai + "/") === 0);
}

// O caminho está dentro do ponto do gvfs (num remoto, montado ou não)?
function eRemoto(caminho, gvfs) {
    return !!gvfs && !!caminho && caminho !== gvfs && dentroDe(gvfs, caminho);
}

// O remoto montado em que um caminho mora (a raiz mais longa), ou null.
function doRemoto(caminho, remotos) {
    var achado = null;
    for (var i = 0; i < (remotos || []).length; i++) {
        var r = remotos[i];
        if (r && r.raiz && dentroDe(r.raiz, caminho) && (achado === null || r.raiz.length > achado.raiz.length))
            achado = r;
    }
    return achado;
}

// As migalhas de um caminho remoto: "servidor › pasta › sub". null se o
// caminho não é de um remoto montado (valem as migalhas comuns).
function migalhas(caminho, remotos) {
    var r = doRemoto(caminho, remotos);
    if (r === null)
        return null;
    var lista = [{ nome: r.nome || r.host || "rede", caminho: r.raiz }];
    var resto = caminho.substring(r.raiz.length).split("/").filter(function (p) { return p !== ""; });
    var atual = r.raiz;
    for (var i = 0; i < resto.length; i++) {
        atual += "/" + resto[i];
        lista.push({ nome: resto[i], caminho: atual });
    }
    return lista;
}

// O endereço (URI) de um caminho remoto: o do remoto mais o resto,
// codificado. null se não é de um remoto montado, ou se o remoto não tem
// endereço conhecido.
function endereco(caminho, remotos) {
    var r = doRemoto(caminho, remotos);
    if (r === null || !r.uri)
        return null;
    var base = String(r.uri).replace(/\/+$/, "");
    var resto = caminho.substring(r.raiz.length).split("/").filter(function (p) { return p !== ""; });
    return base + "/" + resto.map(codificar).join("/");
}

// Dois endereços são do mesmo servidor e compartilhamento? Compara sem a
// barra do fim e sem diferenciar maiúsculas no tipo e no servidor.
function mesmoLugar(a, b) {
    var arruma = function (u) {
        var m = /^([a-zA-Z][a-zA-Z0-9+.-]*:\/\/[^\/]*)(.*)$/.exec(String(u || ""));
        return m ? m[1].toLowerCase() + m[2].replace(/\/+$/, "") : String(u || "");
    };
    return a !== null && b !== null && a !== undefined && b !== undefined && arruma(a) === arruma(b);
}

// Onde um endereço está montado: { remoto, caminho } (o remoto cujo endereço
// é o começo dele, e o caminho já dentro da raiz), ou null.
function montadoEm(uri, remotos) {
    var u = String(uri || "").replace(/\/+$/, "");
    var melhor = null;
    for (var i = 0; i < (remotos || []).length; i++) {
        var r = remotos[i];
        if (!r || !r.uri || !r.raiz)
            continue;
        var base = String(r.uri).replace(/\/+$/, "");
        if (u.toLowerCase() !== base.toLowerCase() && u.toLowerCase().indexOf(base.toLowerCase() + "/") !== 0)
            continue;
        if (melhor !== null && base.length <= melhor.base.length)
            continue;
        var resto = u.substring(base.length).split("/").filter(function (p) { return p !== ""; });
        var caminho = r.raiz;
        var certo = true;
        for (var k = 0; k < resto.length; k++) {
            var nome;
            try {
                nome = decodeURIComponent(resto[k]);
            } catch (e) {
                certo = false;
                break;
            }
            // Um nome decodificado nunca pode sair da raiz.
            if (nome === "" || nome === "." || nome === ".." || nome.indexOf("/") >= 0 || nome.indexOf("\u0000") >= 0) {
                certo = false;
                break;
            }
            caminho += "/" + nome;
        }
        if (certo)
            melhor = { remoto: r, caminho: caminho, base: base };
    }
    return melhor === null ? null : { remoto: melhor.remoto, caminho: melhor.caminho };
}

// O ícone de um remoto pelo protocolo.
function icone(protocolo) {
    if (protocolo === "mtp" || protocolo === "gphoto2")
        return "phone";
    return "folder-remote";
}

// As linhas da seção "Rede" da barra lateral: os remotos montados (com o
// desmontar) e, por fim, a entrada de conectar. Os celulares e câmeras (MTP,
// gphoto2) que o gvfs monta sozinho entram aqui como os outros.
function linhas(remotos) {
    var l = [];
    for (var i = 0; i < (remotos || []).length; i++) {
        var r = remotos[i];
        if (!r || !r.raiz)
            continue;
        l.push({ tipo: "remoto", nome: r.nome || r.host || r.protocolo || "rede", caminho: r.raiz, uri: r.uri || null,
            protocolo: r.protocolo || "", icone: icone(r.protocolo), montado: r.raiz });
    }
    l.push({ tipo: "conectar", nome: "Conectar a um servidor…", caminho: null, icone: "network-server" });
    return l;
}

// A pergunta que o gio fez, para a folha: { titulo, dica, oculto, simNao }.
function pergunta(p) {
    p = p || {};
    switch (p.tipo) {
    case "senha":
        return { titulo: p.de_novo ? "A senha não serviu. Tente de novo" : "Senha", dica: "senha", oculto: true, simNao: false };
    case "usuario":
        return { titulo: "Usuário", dica: "usuário", oculto: false, simNao: false };
    case "dominio":
        return { titulo: "Domínio", dica: "domínio (pode ficar vazio)", oculto: false, simNao: false };
    case "confirmar":
        return { titulo: "O servidor pede uma confirmação", dica: "", oculto: false, simNao: true };
    }
    return { titulo: "O servidor pergunta", dica: "resposta", oculto: false, simNao: false };
}

// A frase de um erro de montagem, pelo motivo.
function erroDeMontar(r) {
    r = r || {};
    switch (r.motivo) {
    case "senha": return "O usuário ou a senha não serviram.";
    case "recusado": return "O servidor recusou a conexão.";
    case "tempo": return "O servidor não respondeu a tempo.";
    case "falta": return r.erro || "Falta o suporte a esse tipo de servidor (gvfs).";
    case "endereco": return r.erro || "Esse endereço não serve.";
    }
    return r.erro || "Não deu para conectar.";
}

// A listagem falhou porque o remoto caiu (ou demorou demais)? É quando o
// painel volta para a pasta pessoal em vez de ficar parado no erro.
function caiu(motivo, caminho, gvfs) {
    return motivo === "conexao" || (motivo === "tempo" && eRemoto(caminho, gvfs))
        || (motivo === "nao-existe" && eRemoto(caminho, gvfs));
}

// Um item de pasta remota ganha miniatura? Só se a pessoa deixa, e só os
// pequenos (baixar um vídeo inteiro pela rede para fazer um quadradinho não).
function cabeMiniatura(item, prefs) {
    prefs = prefs || {};
    if (prefs.miniaturas_remotas === false)
        return false;
    var mb = Number(prefs.miniatura_remota_mb);
    if (!(mb >= 1 && mb <= 100))
        mb = 5;
    return !!item && Number(item.tamanho) <= mb * 1048576;
}

// O próximo limite de miniatura remota, para o menu (1, 5, 20, 100 MB, e volta).
function proximoLimite(mb) {
    var passos = [1, 5, 20, 100];
    var i = passos.indexOf(Number(mb));
    return passos[(i + 1) % passos.length];
}
