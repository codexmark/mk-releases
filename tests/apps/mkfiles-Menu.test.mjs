// apps/mkfiles/qml/Menu.js: os menus de contexto (do item, de vários, do
// fundo, de um lugar), os pontos de extensão das fases seguintes e a folha de
// atalhos.
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const M = load(REPO + "/apps/mkfiles/qml/Menu.js");
const ids = (l) => l.filter((x) => !x.sep).map((x) => x.id);
const arquivo = { nome: "a.zip", tipo: "arquivo", mime: "application/zip", modo: 0o644 };
const pasta = { nome: "p", tipo: "pasta", mime: "inode/directory", modo: 0o755 };
const semExt = { lote: false, compactados: false, previa: false, remotos: false };

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["e", "arrumar", "extensoes", "doCompactado", "menu", "doLugar", "proxima", "principal", "daAba"];
    const funcoes = Object.keys(M).filter((k) => typeof M[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("e e arrumar: uma entrada, e os separadores só entre grupos", () => {
    igual(M.e("x", "Xis"), { id: "x", texto: "Xis", atalho: "", ligado: true });
    eq(M.e("x", "Xis", "F9", false).ligado, false);
    const l = M.arrumar([M.SEP, null, M.e("a", "A"), M.SEP, M.SEP, null, M.e("b", "B"), M.SEP]);
    igual(l.map((x) => (x.sep ? "-" : x.id)), ["a", "-", "b"]);
    igual(M.arrumar([M.SEP]), []);
});

t("menu de um arquivo: tudo o que a tarefa pede", () => {
    const m = ids(M.menu("item", { item: arquivo, n: 1, extensoes: semExt }));
    for (const id of ["abrir", "abrir-com", "recortar", "copiar", "copiar-caminho", "renomear", "ligar", "lixeira", "apagar", "executavel", "propriedades"])
        eq(m.includes(id), true, id);
    for (const id of ["abrir-aba", "abrir-outro", "abrir-janela", "favorito", "terminal", "compactar", "extrair", "previa", "renomear-lote"])
        eq(m.includes(id), false, id);
    eq(M.menu("item", { item: Object.assign({}, arquivo, { modo: 0o755 }), n: 1 }).find((x) => x.id === "executavel").texto, "Tirar a permissão de executar");
});

t("menu de uma pasta: nova aba, outro painel, nova janela, favoritos e terminal", () => {
    const m = ids(M.menu("item", { item: pasta, n: 1, extensoes: semExt }));
    for (const id of ["abrir", "abrir-aba", "abrir-outro", "abrir-janela", "colar-dentro", "favorito", "terminal", "renomear", "propriedades"])
        eq(m.includes(id), true, id);
    eq(m.includes("abrir-com"), false);
    eq(m.includes("executavel"), false);
});

t("menu de vários: sem renomear nem abrir com", () => {
    const m = ids(M.menu("varios", { n: 3, extensoes: semExt }));
    igual(m, ["abrir", "recortar", "copiar", "copiar-caminho", "ligar", "lixeira", "apagar", "propriedades"]);
    eq(M.menu("varios", { n: 3 }).find((x) => x.id === "ligar").texto, "Criar links");
});

t("menu do fundo: novo, colar, selecionar, favoritos, terminal, propriedades", () => {
    const m = M.menu("fundo", { gravavel: true, extensoes: semExt });
    igual(ids(m), ["nova-pasta", "novo-arquivo", "novo-modelo", "colar", "selecionar-tudo", "selecionar-padrao", "favorito", "terminal", "propriedades"]);
    eq(m.every((x) => x.sep || x.ligado), true);
    // Pasta só de leitura: criar e colar ficam desligados, mas à vista.
    const leitura = M.menu("fundo", { gravavel: false });
    for (const id of ["nova-pasta", "novo-arquivo", "novo-modelo", "colar"])
        eq(leitura.find((x) => x.id === id).ligado, false, id);
    eq(leitura.find((x) => x.id === "terminal").ligado, true);
    // Nos resultados da busca não há "aqui".
    eq(M.menu("fundo", { busca: true }).find((x) => x.id === "nova-pasta").ligado, false);
    eq(ids(M.menu("item", { item: arquivo, n: 1, busca: true })).includes("abrir-pasta"), true);
});

t("na lixeira: restaurar, apagar de vez e esvaziar", () => {
    igual(ids(M.menu("item", { item: arquivo, n: 1, lixeira: true })), ["restaurar", "lixeira-apagar", "propriedades"]);
    eq(M.menu("varios", { n: 3, lixeira: true })[0].texto, "Restaurar 3 itens");
    igual(ids(M.menu("fundo", { lixeira: true })), ["esvaziar", "selecionar-tudo"]);
});

t("dentro de um compactado: só abrir, ver e tirar de lá", () => {
    const tudo = { lote: true, compactados: true, previa: true, remotos: false };
    const texto = { nome: "a.txt", tipo: "arquivo", mime: "text/plain", modo: 0o644 };
    igual(ids(M.menu("item", { item: texto, n: 1, compactado: true, extensoes: tudo })), ["abrir", "previa", "copiar", "extrair-selecao", "extrair-para"]);
    igual(ids(M.menu("varios", { n: 3, compactado: true, extensoes: tudo })), ["abrir", "copiar", "extrair-selecao", "extrair-para"]);
    eq(M.menu("varios", { n: 3, compactado: true }).find((x) => x.id === "extrair-selecao").texto, "Extrair 3 itens para a pasta do arquivo");
    igual(ids(M.menu("fundo", { compactado: true })), ["extrair-tudo", "extrair-para", "selecionar-tudo", "selecionar-padrao"]);
    igual(ids(M.doCompactado("item", { n: 1 })), ["abrir", "copiar", "extrair-selecao", "extrair-para"]);
    // Nada que escreva aparece lá dentro.
    for (const id of ["colar", "renomear", "lixeira", "apagar", "nova-pasta", "recortar", "compactar"])
        eq(ids(M.menu("item", { item: texto, n: 1, compactado: true, extensoes: tudo })).includes(id), false, id);
});

t("os pontos de extensão: as entradas das extensões só aparecem com a extensão ligada", () => {
    const tudo = { lote: true, compactados: true, previa: true, remotos: false };
    igual(ids(M.extensoes({ item: arquivo, n: 1, extensoes: tudo })), ["previa", "extrair", "extrair-pasta", "extrair-para", "compactar"]);
    eq(M.extensoes({ item: arquivo, n: 1, extensoes: tudo }).find((x) => x.id === "extrair-pasta").texto, "Extrair para «a»");
    eq(M.extensoes({ item: arquivo, n: 1, gravavel: false, extensoes: tudo }).find((x) => x.id === "extrair").ligado, false);
    igual(ids(M.extensoes({ n: 3, extensoes: tudo })), ["renomear-lote", "compactar"]);
    igual(ids(M.extensoes({ item: pasta, n: 1, extensoes: tudo })), ["previa", "compactar"]);
    igual(M.extensoes({ item: arquivo, n: 1, extensoes: semExt }), []);
    igual(M.extensoes({ item: arquivo, n: 1 }), []);
    const m = ids(M.menu("item", { item: arquivo, n: 1, extensoes: tudo }));
    for (const id of ["previa", "compactar", "extrair"])
        eq(m.includes(id), true, id);
    eq(ids(M.menu("varios", { n: 2, extensoes: tudo })).includes("renomear-lote"), true);
});

t("menu de um lugar da barra lateral: lugar fixo, favorito, disco e lixeira", () => {
    igual(ids(M.menu("lugar", { lugar: { tipo: "lugar" } })), ["lugar-abrir", "lugar-aba", "lugar-outro", "lugar-janela", "lugar-terminal", "lugar-propriedades"]);
    const fav = M.menu("lugar", { lugar: { tipo: "favorito", primeiro: true, ultimo: false } });
    for (const id of ["favorito-renomear", "favorito-subir", "favorito-descer", "favorito-tirar"])
        eq(ids(fav).includes(id), true, id);
    eq(fav.find((x) => x.id === "favorito-subir").ligado, false);
    eq(fav.find((x) => x.id === "favorito-descer").ligado, true);
    // Um favorito de rede abre montando; outra janela (outro motor) e o terminal não.
    const rede = M.menu("lugar", { lugar: { tipo: "favorito", remoto: true } });
    eq(rede.find((x) => x.id === "lugar-abrir").ligado, true);
    eq(rede.find((x) => x.id === "lugar-abrir").texto, "Conectar e abrir");
    eq(rede.find((x) => x.id === "lugar-janela").ligado, false);
    eq(rede.find((x) => x.id === "lugar-terminal").ligado, false);
    // Um remoto montado tem o desconectar; a entrada de conectar só conecta.
    const remoto = ids(M.menu("lugar", { lugar: { tipo: "remoto", montado: "/run/user/1000/gvfs/sftp:host=x", uri: "sftp://x/" } }));
    for (const id of ["lugar-abrir", "remoto-desmontar", "lugar-endereco", "lugar-favorito"])
        eq(remoto.includes(id), true, id);
    eq(ids(M.menu("lugar", { lugar: { tipo: "remoto", montado: "/x", uri: null } })).includes("lugar-endereco"), false);
    igual(ids(M.menu("lugar", { lugar: { tipo: "conectar" } })), ["conectar"]);
    const montado = ids(M.menu("lugar", { lugar: { tipo: "disco", montado: "/run/media/ana/X", removivel: true } }));
    eq(montado.includes("desmontar") && montado.includes("ejetar") && !montado.includes("montar"), true);
    const solto = M.menu("lugar", { lugar: { tipo: "disco", montado: null, removivel: false } });
    eq(ids(solto).includes("montar") && !ids(solto).includes("ejetar") && !ids(solto).includes("lugar-terminal"), true);
    eq(solto.find((x) => x.id === "lugar-abrir").ligado, false);
    igual(ids(M.menu("lugar", { lugar: { tipo: "lixeira" } })), ["lugar-abrir", "lugar-aba", "esvaziar"]);
    igual(ids(M.doLugar({})).slice(0, 2), ["lugar-abrir", "lugar-aba"]);
});

t("proxima: as setas pulam separadores e entradas desligadas, dando a volta", () => {
    const l = [M.e("a", "A"), M.SEP, M.e("b", "B", "", false), M.e("c", "C")];
    eq(M.proxima(l, 0, 1), 3);
    eq(M.proxima(l, 3, 1), 0);
    eq(M.proxima(l, 0, -1), 3);
    eq(M.proxima(l, -1, 1), 0);
    eq(M.proxima([M.SEP], 0, 1), 0);
});

t("o menu principal: janela, visão, preferências, desfazer", () => {
    const m = M.principal({ duplo: false, ocultos: true, pastas_antes: true, clique: "duplo", confirmar_lixeira: false, desfazer: "Mover 3 itens", refazer: null });
    igual(ids(m), ["nova-janela", "nova-aba", "duplo", "ocultos", "pastas-antes", "clique", "confirmar-lixeira", "conectar", "miniaturas-remotas",
        "miniatura-remota-mb", "desfazer", "refazer", "recarregar", "atalhos"]);
    eq(m.find((x) => x.id === "miniatura-remota-mb").texto, "Miniaturas de rede: arquivos de até 5 MB");
    eq(M.principal({ miniaturas_remotas: false, miniatura_remota_mb: 20 }).find((x) => x.id === "miniatura-remota-mb").ligado, false);
    eq(M.principal({ miniaturas_remotas: false }).find((x) => x.id === "miniaturas-remotas").texto, "Miniaturas em pastas de rede");
    eq(m.find((x) => x.id === "desfazer").texto, "Desfazer: Mover 3 itens");
    eq(m.find((x) => x.id === "refazer").ligado, false);
    eq(m.find((x) => x.id === "ocultos").texto, "Esconder os ocultos");
    eq(M.principal().find((x) => x.id === "desfazer").ligado, false);
});

t("a folha de atalhos tem todo atalho que a tarefa pede", () => {
    const teclas = M.ATALHOS.map((a) => a[0]).join(" | ");
    for (const tecla of ["Enter", "Backspace", "Alt+↑", "Alt+←", "Alt+→", "Ctrl+L", "Ctrl+F", "Ctrl+T", "Ctrl+W", "Ctrl+Tab", "F3", "Tab", "F5", "F6",
        "Ctrl+H", "Ctrl+A", "Ctrl+S", "Ctrl+C", "F2", "Delete", "Shift+Delete", "Ctrl+Z", "Ctrl+Shift+Z", "Ctrl+Shift+N", "Alt+Enter", "F4", "F1",
        "Menu", "Shift+F10", "Ctrl++", "Esc"])
        eq(teclas.includes(tecla), true, tecla);
    eq(M.ATALHOS.every((a) => a.length === 2 && a[0] !== "" && a[1] !== ""), true);
});

t("o menu de uma aba: duplicar, levar ao outro painel, fechar", () => {
    igual(ids(M.daAba({ n: 3 })), ["aba-duplicar", "aba-outro", "aba-fechar", "aba-outras"]);
    eq(M.daAba({ n: 1 }).find((x) => x.id === "aba-fechar").ligado, false);
    eq(M.daAba().find((x) => x.id === "aba-duplicar").ligado, true);
});

t("numa pasta de rede: sem lixeira (Delete apaga de vez) e com o endereço para copiar", () => {
    const item = { nome: "a.txt", caminho: "/run/user/1000/gvfs/sftp:host=x/a.txt", tipo: "arquivo", modo: 420 };
    const m = M.menu("item", { item, n: 1, remoto: true });
    eq(ids(m).includes("lixeira"), false);
    eq(m.find((x) => x.id === "apagar").atalho, "Delete");
    eq(ids(m).includes("copiar-endereco"), true);
    eq(ids(M.menu("varios", { n: 2, remoto: true })).includes("copiar-endereco"), true);
    eq(ids(M.menu("fundo", { n: 0, remoto: true })).includes("copiar-endereco"), true);
    // Fora da rede, nada muda.
    const local = M.menu("item", { item, n: 1 });
    eq(ids(local).includes("lixeira"), true);
    eq(ids(local).includes("copiar-endereco"), false);
    eq(local.find((x) => x.id === "apagar").atalho, "Shift+Delete");
    eq(M.ATALHOS.some((a) => a[0] === "Ctrl+K"), true);
});
