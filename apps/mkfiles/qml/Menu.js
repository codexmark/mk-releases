.pragma library
.import "Arquivos.js" as Arquivos

// Os menus de contexto e a folha de atalhos. Um menu é uma lista de entradas
// { id, texto, atalho, ligado } e de separadores { sep: true }; a janela trata
// cada id em App.acao(). As entradas das extensões (prévia, renomear em lote,
// compactar e extrair) aparecem quando o motor diz que tem a extensão
// (`sobre.extensoes`). Numa pasta de rede (c.remoto) não há lixeira, e o
// endereço (sftp://…) pode ser copiado.

var ATALHOS = [
    ["Enter", "abrir"],
    ["Backspace · Alt+↑", "subir uma pasta"],
    ["Alt+← · Alt+→", "voltar e avançar"],
    ["Ctrl+L", "digitar o caminho (Tab completa)"],
    ["Ctrl+F", "buscar nesta pasta"],
    ["Ctrl+T · Ctrl+W", "nova aba, fechar aba"],
    ["Ctrl+Tab", "trocar de aba"],
    ["F3", "painel duplo"],
    ["Tab", "alternar o painel ativo"],
    ["F5 · F6", "copiar, mover para o outro painel"],
    ["Ctrl+1 · 2 · 3", "ícones, lista, compacta"],
    ["Ctrl++ · Ctrl+−", "tamanho dos ícones"],
    ["Ctrl+H", "mostrar ocultos"],
    ["Ctrl+A", "selecionar tudo"],
    ["Ctrl+Shift+I", "inverter a seleção"],
    ["Ctrl+S", "selecionar por padrão (*.jpg)"],
    ["Ctrl+C · X · V", "copiar, recortar, colar"],
    ["Ctrl+Shift+C", "copiar o caminho"],
    ["F2", "renomear (com vários: em lote)"],
    ["Espaço", "prévia rápida (setas passam, Enter abre)"],
    ["Delete", "mover para a lixeira"],
    ["Shift+Delete", "apagar de vez"],
    ["Ctrl+Z · Ctrl+Shift+Z", "desfazer, refazer"],
    ["Ctrl+Shift+N", "nova pasta"],
    ["Alt+Enter", "propriedades"],
    ["F4", "terminal aqui"],
    ["Ctrl+N", "nova janela"],
    ["Ctrl+K", "conectar a um servidor"],
    ["Menu · Shift+F10", "menu de contexto"],
    ["Ctrl+R", "recarregar"],
    ["F1", "esta folha"],
    ["Esc", "fechar o que está aberto"]
];

function e(id, texto, atalho, ligado) {
    return { id: id, texto: texto, atalho: atalho || "", ligado: ligado !== false };
}

var SEP = { sep: true };

// Tira separadores do começo, do fim e repetidos.
function arrumar(lista) {
    var saida = [];
    for (var i = 0; i < lista.length; i++) {
        if (!lista[i])
            continue;
        if (lista[i].sep && (saida.length === 0 || saida[saida.length - 1].sep))
            continue;
        saida.push(lista[i]);
    }
    while (saida.length > 0 && saida[saida.length - 1].sep)
        saida.pop();
    return saida;
}

// As entradas das extensões para uma seleção.
function extensoes(c) {
    var x = c.extensoes || {};
    var lista = [];
    if (x.previa && c.n === 1)
        lista.push(e("previa", "Prévia rápida", "Espaço"));
    if (x.lote && c.n > 1)
        lista.push(e("renomear-lote", "Renomear em lote…", "F2"));
    if (x.compactados) {
        if (c.n === 1 && Arquivos.eCompactado(c.item)) {
            lista.push(e("extrair", "Extrair aqui", "", c.gravavel !== false));
            lista.push(e("extrair-pasta", "Extrair para «" + Arquivos.pastaDeExtracao(c.item.nome) + "»", "", c.gravavel !== false));
            lista.push(e("extrair-para", "Extrair para…"));
        }
        lista.push(e("compactar", "Compactar…"));
    }
    return lista;
}

// Dentro de um compactado (só leitura): abrir, ver e tirar de lá.
function doCompactado(alvo, c) {
    var x = c.extensoes || {};
    if (alvo === "fundo")
        return arrumar([
            e("extrair-tudo", "Extrair tudo para a pasta do arquivo"),
            e("extrair-para", "Extrair tudo para…"),
            SEP,
            e("selecionar-tudo", "Selecionar tudo", "Ctrl+A"),
            e("selecionar-padrao", "Selecionar por padrão…", "Ctrl+S")
        ]);
    return arrumar([
        e("abrir", "Abrir", "Enter"),
        x.previa && c.n === 1 ? e("previa", "Prévia rápida", "Espaço") : null,
        SEP,
        e("copiar", "Copiar", "Ctrl+C"),
        e("extrair-selecao", c.n > 1 ? "Extrair " + c.n + " itens para a pasta do arquivo" : "Extrair para a pasta do arquivo"),
        e("extrair-para", "Extrair para…")
    ]);
}

// O menu de um alvo: "item" (um selecionado), "varios", "fundo" (o vazio da
// pasta) ou "lugar" (uma linha da barra lateral). c é o contexto:
// { item, n, lixeira, busca, compactado, colar, gravavel, duplo, extensoes, lugar }.
function menu(alvo, c) {
    c = c || {};
    var grava = c.gravavel !== false;
    if (alvo === "lugar")
        return doLugar(c);
    if (c.compactado)
        return doCompactado(alvo, c);
    if (c.lixeira) {
        if (alvo === "fundo")
            return arrumar([e("esvaziar", "Esvaziar a lixeira"), SEP, e("selecionar-tudo", "Selecionar tudo", "Ctrl+A")]);
        return arrumar([
            e("restaurar", c.n > 1 ? "Restaurar " + c.n + " itens" : "Restaurar"),
            e("lixeira-apagar", "Apagar de vez…", "Shift+Delete"),
            SEP,
            e("propriedades", "Propriedades", "Alt+Enter")
        ]);
    }
    if (alvo === "fundo")
        return arrumar([
            e("nova-pasta", "Nova pasta", "Ctrl+Shift+N", grava && !c.busca),
            e("novo-arquivo", "Novo arquivo", "", grava && !c.busca),
            e("novo-modelo", "Novo a partir de modelo…", "", grava && !c.busca),
            SEP,
            e("colar", "Colar", "Ctrl+V", grava && !c.busca),
            SEP,
            e("selecionar-tudo", "Selecionar tudo", "Ctrl+A"),
            e("selecionar-padrao", "Selecionar por padrão…", "Ctrl+S"),
            SEP,
            e("favorito", "Adicionar aos favoritos", "", !c.busca),
            e("terminal", "Abrir terminal aqui", "F4", !c.busca),
            c.remoto ? e("copiar-endereco", "Copiar o endereço desta pasta") : null,
            SEP,
            e("propriedades", "Propriedades", "Alt+Enter", !c.busca)
        ]);
    var um = alvo === "item";
    var pasta = um && Arquivos.ePasta(c.item);
    return arrumar([
        e("abrir", "Abrir", "Enter"),
        um && !pasta ? e("abrir-com", "Abrir com…") : null,
        um && pasta ? e("abrir-aba", "Abrir em nova aba") : null,
        um && pasta ? e("abrir-outro", "Abrir no outro painel") : null,
        um && pasta ? e("abrir-janela", "Abrir em nova janela") : null,
        um && c.busca ? e("abrir-pasta", "Abrir a pasta em que está") : null,
        SEP
    ].concat(extensoes(c)).concat([
        SEP,
        e("recortar", "Recortar", "Ctrl+X"),
        e("copiar", "Copiar", "Ctrl+C"),
        um && pasta ? e("colar-dentro", "Colar dentro", "", c.colar !== false) : null,
        e("copiar-caminho", um ? "Copiar o caminho" : "Copiar os caminhos", "Ctrl+Shift+C"),
        c.remoto ? e("copiar-endereco", um ? "Copiar o endereço" : "Copiar os endereços") : null,
        SEP,
        um ? e("renomear", "Renomear", "F2") : null,
        e("ligar", um ? "Criar link" : "Criar links", "", grava),
        // Numa pasta de rede não há lixeira: Delete pergunta e apaga de vez.
        c.remoto ? null : e("lixeira", "Mover para a lixeira", "Delete"),
        e("apagar", "Apagar de vez…", c.remoto ? "Delete" : "Shift+Delete"),
        SEP,
        um && pasta ? e("favorito", "Adicionar aos favoritos") : null,
        um && pasta ? e("terminal", "Abrir terminal aqui", "F4") : null,
        um && !pasta && c.item && c.item.tipo === "arquivo" ? e("executavel", (c.item.modo & 64) ? "Tirar a permissão de executar" : "Tornar executável") : null,
        SEP,
        e("propriedades", "Propriedades", "Alt+Enter")
    ]));
}

// O menu de uma linha da barra lateral. c.lugar = { tipo: "lugar" |
// "favorito" | "disco" | "lixeira" | "remoto" | "conectar", montado, primeiro,
// ultimo, remoto }. Um favorito de rede (remoto) abre montando; um remoto
// montado tem o desconectar.
function doLugar(c) {
    var l = c.lugar || {};
    if (l.tipo === "lixeira")
        return arrumar([e("lugar-abrir", "Abrir"), e("lugar-aba", "Abrir em nova aba"), SEP, e("esvaziar", "Esvaziar a lixeira")]);
    if (l.tipo === "conectar")
        return arrumar([e("conectar", "Conectar a um servidor…", "Ctrl+K")]);
    var abre = l.tipo !== "disco" || !!l.montado;
    var lista = [
        e("lugar-abrir", l.remoto ? "Conectar e abrir" : "Abrir", "", abre),
        e("lugar-aba", "Abrir em nova aba", "", abre),
        e("lugar-outro", "Abrir no outro painel", "", abre),
        // Outra janela é outro motor: só abre o que já é uma pasta daqui.
        e("lugar-janela", "Abrir em nova janela", "", abre && !l.remoto),
        SEP
    ];
    if (l.tipo === "remoto")
        lista = lista.concat([
            e("remoto-desmontar", "Desconectar"),
            l.uri ? e("lugar-endereco", "Copiar o endereço") : null,
            l.uri ? e("lugar-favorito", "Adicionar aos favoritos") : null
        ]);
    if (l.tipo === "favorito")
        lista = lista.concat([
            e("favorito-renomear", "Renomear o favorito…"),
            e("favorito-subir", "Mover para cima", "", !l.primeiro),
            e("favorito-descer", "Mover para baixo", "", !l.ultimo),
            e("favorito-tirar", "Tirar dos favoritos")
        ]);
    if (l.tipo === "disco")
        lista = lista.concat([
            l.montado ? e("desmontar", "Desmontar") : e("montar", "Montar"),
            l.removivel ? e("ejetar", "Ejetar") : null
        ]);
    if (l.tipo !== "disco" || l.montado)
        lista = lista.concat([SEP, e("lugar-terminal", "Abrir terminal aqui", "", !l.remoto), e("lugar-propriedades", "Propriedades", "", !l.remoto)]);
    return arrumar(lista);
}

// Anda no menu com as setas, pulando separadores e entradas desligadas.
function proxima(lista, i, delta) {
    var n = lista.length;
    for (var k = 1; k <= n; k++) {
        var j = ((i + delta * k) % n + n) % n;
        if (!lista[j].sep && lista[j].ligado)
            return j;
    }
    return i;
}

// As entradas do menu principal (o botão ☰ da barra de cima).
function principal(c) {
    c = c || {};
    return arrumar([
        e("nova-janela", "Nova janela", "Ctrl+N"),
        e("nova-aba", "Nova aba", "Ctrl+T"),
        e("duplo", c.duplo ? "Desligar o painel duplo" : "Painel duplo", "F3"),
        SEP,
        e("ocultos", c.ocultos ? "Esconder os ocultos" : "Mostrar os ocultos", "Ctrl+H"),
        e("pastas-antes", c.pastas_antes ? "Misturar pastas e arquivos" : "Pastas antes dos arquivos"),
        e("clique", c.clique === "simples" ? "Abrir com clique duplo" : "Abrir com um clique"),
        e("confirmar-lixeira", c.confirmar_lixeira ? "Não confirmar antes da lixeira" : "Confirmar antes da lixeira"),
        SEP,
        e("conectar", "Conectar a um servidor…", "Ctrl+K"),
        e("miniaturas-remotas", c.miniaturas_remotas === false ? "Miniaturas em pastas de rede" : "Sem miniaturas em pastas de rede"),
        e("miniatura-remota-mb", "Miniaturas de rede: arquivos de até " + (c.miniatura_remota_mb || 5) + " MB", "", c.miniaturas_remotas !== false),
        SEP,
        e("desfazer", c.desfazer ? "Desfazer: " + c.desfazer : "Desfazer", "Ctrl+Z", !!c.desfazer),
        e("refazer", c.refazer ? "Refazer: " + c.refazer : "Refazer", "Ctrl+Shift+Z", !!c.refazer),
        SEP,
        e("recarregar", "Recarregar", "Ctrl+R"),
        e("atalhos", "Atalhos", "F1")
    ]);
}

// O menu de uma aba (botão direito nela).
function daAba(c) {
    c = c || {};
    return arrumar([
        e("aba-duplicar", "Duplicar a aba", "Ctrl+Shift+D"),
        e("aba-outro", "Abrir no outro painel"),
        SEP,
        e("aba-fechar", "Fechar a aba", "Ctrl+W", c.n > 1),
        e("aba-outras", "Fechar as outras", "", c.n > 1)
    ]);
}
