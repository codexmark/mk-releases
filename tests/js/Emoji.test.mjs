// shell/mare/Emoji.js: os emoji do painel. O arquivo é gerado
// (scripts/mare-emoji gerar); aqui se confere a forma dos dados e as funções
// que vão junto: o tom de pele, as listas e a busca.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { load, source } from "./load.mjs";

const E = load("Emoji.js");
// As linhas cruas, antes de o load() do arquivo soltá-las (raw vira null).
const cru = {};
for (const g of Object.keys(E.raw))
    cru[g] = E.raw[g].slice();
const linhasCruas = [].concat(...E.groups.map((g) => cru[g.key] || []));
const um = (s) => {
    const e = E.fromChars([s])[0];
    assert.ok(e, "não existe o emoji " + s);
    return e;
};
const chars = (lista) => lista.map((e) => e.s);
// Os cinco tons de pele do Unicode.
const TOM = /[\u{1F3FB}-\u{1F3FF}]/u;

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["fold", "withTone", "load", "list", "fromChars", "hit", "leads", "search"];
    const funcoes = Object.keys(E).filter((k) => typeof E[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste: acrescente o caso e ponha na lista");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

// --- a forma dos dados ---

t("categorias: chaves únicas, com nome, e cada uma com a sua lista de emoji", () => {
    const chaves = E.groups.map((g) => g.key);
    igual(chaves.filter((k, i) => chaves.indexOf(k) !== i), []);
    for (const g of E.groups) {
        assert.match(g.key, /^[a-z]+$/);
        assert.ok(typeof g.name === "string" && g.name.trim() !== "", g.key);
        assert.ok(Array.isArray(cru[g.key]) && cru[g.key].length > 0, g.key + " sem emoji");
    }
    igual(Object.keys(cru).filter((k) => !chaves.includes(k)), [], "lista sem categoria");
});

t("linhas: emoji|nome|palavras|nome em inglês, e às vezes um quinto campo", () => {
    const ruins = [];
    for (const l of linhasCruas) {
        const p = l.split("|");
        if (p.length !== 4 && p.length !== 5)
            ruins.push(p.length + " campos: " + l);
        else if (p[0] === "" || /\s/.test(p[0]))
            ruins.push("emoji: " + l);
        else if (p[3].trim() === "" || p[3] !== p[3].trim())
            ruins.push("nome em inglês: " + l);
        else if (p[1] !== p[1].trim() || p[2] !== p[2].trim())
            ruins.push("espaço sobrando: " + l);
        else if (p.length === 5 && p[4].trim() === "")
            ruins.push("quinto campo vazio: " + l);
        if (/[\n\r\t]/.test(l))
            ruins.push("quebra de linha ou tab: " + JSON.stringify(l));
    }
    vazio(ruins, "linhas ruins");
});

t("linhas: os números do cabeçalho são os do arquivo", () => {
    const m = /^\/\/ (\d+) emoji; (\d+) só com o nome em inglês/m.exec(source("Emoji.js"));
    assert.ok(m, "o cabeçalho não diz quantos emoji há");
    eq(linhasCruas.length, Number(m[1]));
    eq(linhasCruas.filter((l) => l.split("|")[1] === "").length, Number(m[2]));
});

t("linhas: todo emoji é um emoji (pictograma, bandeira ou tecla)", () => {
    const ruins = linhasCruas.map((l) => l.split("|")[0])
        .filter((c) => !/[\p{Extended_Pictographic}\p{Regional_Indicator}⃣]/u.test(c) || /[A-Za-z]/.test(c));
    vazio(ruins, "não são emoji");
});

t("linhas: nenhum emoji repetido, nem dentro de uma categoria nem entre elas", () => {
    const visto = {};
    const ruins = [];
    for (const l of linhasCruas) {
        const s = l.split("|")[0].replace(/~/g, "");
        if (Object.prototype.hasOwnProperty.call(visto, s))
            ruins.push(s + ": " + visto[s] + " e " + l.split("|")[1]);
        visto[s] = l.split("|")[1];
    }
    vazio(ruins, "emoji repetidos");
});

t("linhas: nenhum nome em português repetido", () => {
    const nomes = linhasCruas.map((l) => l.split("|")[1]).filter((n) => n !== "");
    const visto = new Set();
    const ruins = nomes.filter((n) => visto.has(n) || !visto.add(n));
    vazio(ruins, "nomes repetidos");
});

t("tom de pele: o '~' vem logo depois de um pictograma, e o emoji já não traz tom", () => {
    const ruins = [];
    for (const l of linhasCruas) {
        const c = l.split("|")[0];
        if (TOM.test(c))
            ruins.push("já tem tom: " + c);
        if (/^~|~~/.test(c) || /~/.test(c) && !/\p{Extended_Pictographic}~/u.test(c))
            ruins.push("~ fora do lugar: " + c);
    }
    vazio(ruins, "tons ruins");
    assert.ok(linhasCruas.some((l) => l.split("|")[0].includes("~")), "nenhum emoji aceita tom");
});

t("tones: sem tom e os cinco tons do Unicode, do claro ao escuro", () => {
    igual(E.tones, ["", "\u{1F3FB}", "\u{1F3FC}", "\u{1F3FD}", "\u{1F3FE}", "\u{1F3FF}"]);
});

// --- fold, withTone ---

t("fold: minúsculas e sem acento, como a busca dos Ajustes", () => {
    eq(E.fold("Coração"), "coracao");
    eq(E.fold("MÃO À OBRA é ü"), "mao a obra e u");
    eq(E.fold(""), "");
    const S = load("SettingsSchema.js");
    for (const texto of ["Coração Ácido", "bandeira: Brasil", "ÀÉÎÕÜ ç", "😀 a~b"])
        eq(E.fold(texto), S.fold(texto));
});

t("withTone: tom 0 tira a marca; 1 a 5 põem o tom no lugar dela", () => {
    eq(E.withTone("👋~", 0), "👋");
    eq(E.withTone("👋~", 1), "👋\u{1F3FB}");
    eq(E.withTone("👋~", 3), "👋\u{1F3FD}");
    eq(E.withTone("👋~", 5), "👋\u{1F3FF}");
    eq(E.withTone("😀", 3), "😀");
    eq(E.withTone("😀", 0), "😀");
});

t("withTone: com tom, o U+FE0F que vinha depois da base sai", () => {
    eq(E.withTone("\u{1F590}~️", 2), "\u{1F590}\u{1F3FC}");
    eq(E.withTone("\u{1F590}~️", 0), "\u{1F590}️");
    // No meio de uma sequência, o resto fica.
    eq(E.withTone("\u{1F575}~️‍♂️", 4), "\u{1F575}\u{1F3FE}‍♂️");
});

t("withTone: um emoji com duas pessoas ganha o tom nas duas", () => {
    eq(E.withTone("a~b~c", 2), "a\u{1F3FC}b\u{1F3FC}c");
    eq(E.withTone("a~b~c", 0), "abc");
});

t("withTone: tom fora de 1 a 5 é sem tom", () => {
    for (const ruim of [6, -1, 99, undefined, null, 2.5, "x"])
        eq(E.withTone("👋~", ruim), "👋", String(ruim));
});

t("withTone: todo emoji da lista sai pronto para copiar, com e sem tom", () => {
    const ruins = [];
    for (const g of E.groups)
        for (const e of E.list(g.key)) {
            if (E.withTone(e.c, 0) !== e.s)
                ruins.push(e.s + ": sem tom diferente de s");
            for (let tom = 1; tom <= 5; tom++) {
                const pronto = E.withTone(e.c, tom);
                if (pronto.includes("~"))
                    ruins.push(e.s + ": sobrou ~");
                if (e.tone !== TOM.test(pronto))
                    ruins.push(e.s + ": tom " + tom);
                if (/[\u{1F3FB}-\u{1F3FF}]️/u.test(pronto))
                    ruins.push(e.s + ": U+FE0F depois do tom");
                if (!e.tone && pronto !== e.s)
                    ruins.push(e.s + ": mudou sem aceitar tom");
            }
        }
    vazio(ruins, "emoji que não saem prontos");
});

// --- load, list, fromChars ---

t("load: as linhas viram objetos uma vez só, e as cruas são soltas", () => {
    const novo = load("Emoji.js");
    eq(novo.all, null);
    assert.ok(novo.raw !== null);
    novo.load();
    eq(novo.all.length, linhasCruas.length);
    eq(novo.raw, null);
    const antes = novo.all;
    novo.load();
    eq(novo.all, antes);
    eq(novo.all.length, linhasCruas.length);
});

t("load: cada objeto tem c, s, n, g, tone e os textos da busca (fn, fk, fe, fx)", () => {
    E.load();
    const ruins = [];
    E.all.forEach((e, i) => {
        const p = linhasCruas[i].split("|");
        igual(Object.keys(e), ["c", "s", "n", "g", "tone", "fn", "fk", "fe", "fx"]);
        if (e.c !== p[0] || e.s !== p[0].replace(/~/g, "") || e.s.includes("~"))
            ruins.push(e.c + ": c/s");
        if (e.n !== (p[1] || p[3]) || e.n === "")
            ruins.push(e.c + ": n");
        if (e.tone !== p[0].includes("~"))
            ruins.push(e.c + ": tone");
        if (!E.groups.some((g) => g.key === e.g))
            ruins.push(e.c + ": g " + e.g);
        if (e.fn !== E.fold(p[1]) || e.fe !== p[3].toLowerCase())
            ruins.push(e.c + ": fn/fe");
        if (!e.fk.startsWith(E.fold(p[2])) || (p[4] && !e.fk.endsWith(E.fold(p[4]))))
            ruins.push(e.c + ": fk");
        // As palavras postas à mão ficam entre espaços, para achar uma inteira.
        if (e.fx !== (p[4] ? " " + E.fold(p[4]) + " " : ""))
            ruins.push(e.c + ": fx");
    });
    vazio(ruins, "objetos ruins");
});

t("list: os emoji da categoria, na ordem do arquivo", () => {
    for (const g of E.groups) {
        const l = E.list(g.key);
        igual(l.map((e) => e.c), cru[g.key].map((x) => x.split("|")[0]));
        assert.ok(l.every((e) => e.g === g.key));
    }
    eq(E.list("rostos")[0].s, "😀");
    eq(E.list("rostos")[0].n, "rosto risonho");
    eq(E.groups.reduce((n, g) => n + E.list(g.key).length, 0), linhasCruas.length);
});

t("list: categoria que não existe (ou 'recentes') dá []", () => {
    igual(E.list("nao-existe"), []);
    igual(E.list("recentes"), []);
    igual(E.list(""), []);
    igual(E.list(undefined), []);
});

t("list: funciona sem ninguém ter chamado load()", () => {
    const novo = load("Emoji.js");
    eq(novo.list("rostos").length, cru.rostos.length);
});

t("fromChars: os recentes voltam a ser linhas, na ordem dada", () => {
    const l = E.fromChars(["👋", "😀", "🇧🇷"]);
    igual(chars(l), ["👋", "😀", "🇧🇷"]);
    eq(l[0].c, "👋~");
    eq(l[0].tone, true);
    eq(l[1], E.list("rostos")[0]);
    eq(l[2].n, "bandeira: Brasil");
});

t("fromChars: o que não é da lista cai fora; vazio, null e undefined dão []", () => {
    igual(chars(E.fromChars(["😀", "banana", "", "👋\u{1F3FD}", "👋"])), ["😀", "👋"]);
    igual(E.fromChars([]), []);
    igual(E.fromChars(null), []);
    igual(E.fromChars(undefined), []);
});

t("fromChars: todo emoji da lista se acha pelo próprio caractere", () => {
    E.load();
    const de = E.fromChars(E.all.map((e) => e.s));
    eq(de.length, E.all.length);
    assert.ok(de.every((e, i) => e === E.all[i]));
});

// --- hit, leads ---

t("hit: quanto a palavra casa: no começo do texto, no começo de uma palavra, ou no meio", () => {
    eq(E.hit("cafe com leite", "cafe", 100, 80, 50), 100);
    eq(E.hit("xicara de cafe", "cafe", 100, 80, 50), 80);
    eq(E.hit("descafeinado", "cafe", 100, 80, 50), 50);
    eq(E.hit("cha", "cafe", 100, 80, 50), 0);
    eq(E.hit("", "cafe", 100, 80, 50), 0);
});

t("hit: depois de hífen também é começo de palavra", () => {
    eq(E.hit("guarda-chuva", "chuva", 100, 80, 50), 80);
});

t("hit: vale o melhor lugar, não o primeiro", () => {
    // A primeira vez é no meio ("descafe"), a segunda no começo de uma palavra.
    eq(E.hit("descafe e cafe", "cafe", 100, 80, 50), 80);
    eq(E.hit("cafe e descafe", "cafe", 100, 80, 50), 100);
});

t("leads: o nome começa com a palavra inteira ('café', e não 'cafeteira')", () => {
    eq(E.leads("cafe", "cafe"), true);
    eq(E.leads("cafe com leite", "cafe"), true);
    eq(E.leads("cafeteira", "cafe"), false);
    eq(E.leads("um cafe", "cafe"), false);
    eq(E.leads("", "cafe"), false);
});

// --- search ---

t("search: busca vazia, ou sem nada parecido, não acha nada", () => {
    igual(E.search(""), []);
    igual(E.search("   "), []);
    igual(E.search("xyzzyquux"), []);
});

t("search: acha pelo nome em português, sem acento e sem caixa", () => {
    eq(E.search("café")[0].s, "☕");
    igual(chars(E.search("cafe")), chars(E.search("café")));
    igual(chars(E.search("CAFÉ")), chars(E.search("café")));
    igual(chars(E.search("coracao vermelho")), chars(E.search("Coração Vermelho")));
    eq(E.search("coração vermelho")[0].n, "coração vermelho");
});

t("search: acha pelas palavras de busca e pelo nome em inglês", () => {
    assert.ok(chars(E.search("tchau")).includes("👋"));
    assert.ok(chars(E.search("grinning face")).includes("😀"));
    assert.ok(chars(E.search("dog")).includes("🐕"));
});

t("search: todas as palavras digitadas têm de casar", () => {
    const r = E.search("gato preto");
    eq(r[0].n, "gato preto");
    assert.ok(r.length < E.search("gato").length);
    igual(E.search("gato xyzzyquux"), []);
    igual(chars(E.search("preto gato")).sort(), chars(r).sort());
});

t("search: o nome que começa com a palavra inteira vem antes do que só a contém", () => {
    const r = E.search("gato");
    igual(r.slice(0, 2).map((e) => e.n), ["gato", "gato preto"]);
    assert.ok(r.length > 2);
    eq(E.search("sol")[0].n, "sol");
    const guarda = E.search("guarda").map((e) => e.n);
    assert.ok(guarda.indexOf("guarda") < guarda.indexOf("guarda-chuva"));
});

t("search: uma palavra posta à mão e digitada inteira ganha de um começo de nome ('br')", () => {
    const r = E.search("br").map((e) => e.n);
    eq(r[0], "bandeira: Brasil");
    assert.ok(r.includes("braço mecânico"));
    assert.ok(r.indexOf("bandeira: Brasil") < r.indexOf("braço mecânico"));
    eq(E.search("eua")[0].n, "bandeira: Estados Unidos");
    eq(E.search("joinha")[0].s, "👍");
});

t("search: mas a palavra à mão perde para o nome que começa com a palavra inteira ('cd')", () => {
    const r = E.search("cd").map((e) => e.n);
    eq(r[0], "cd");
    eq(r[1], "bandeira: Congo - Kinshasa");
});

t("search: no empate, vale a ordem do Unicode (a do arquivo)", () => {
    E.load();
    for (const q of ["rosto", "bandeira", "mão", "kkk"]) {
        const r = E.search(q);
        assert.ok(r.length > 3, q);
        // Dentro do mesmo nome inicial o peso é igual: a ordem tem de ser a da lista.
        const lideram = r.filter((e) => E.leads(e.fn, E.fold(q)));
        const ordem = lideram.map((e) => E.all.indexOf(e));
        igual(ordem, ordem.slice().sort((a, b) => a - b), q);
    }
    igual(chars(E.search("rosto risonho")).slice(0, 3), ["😀", "😃", "😄"]);
});

t("search: com `limit`, só as primeiras; sem ele (ou 0), todas", () => {
    const todas = E.search("mão");
    assert.ok(todas.length > 5);
    igual(E.search("mão", 3), todas.slice(0, 3));
    igual(E.search("mão", 1), todas.slice(0, 1));
    igual(E.search("mão", 100000), todas);
    igual(E.search("mão", 0), todas);
    igual(E.search("mão", undefined), todas);
});

t("search: devolve as mesmas linhas das listas, sem repetir nenhuma", () => {
    E.load();
    for (const q of ["a", "rosto", "br", "flag", "mão coração"]) {
        const r = E.search(q);
        assert.ok(r.every((e) => E.all.includes(e)), q);
        eq(new Set(r).size, r.length, q);
    }
});

t("search: todo emoji se acha pelo próprio nome", () => {
    E.load();
    // Um em cada cinco: a busca varre a lista inteira a cada vez.
    const ruins = E.all.filter((e, i) => i % 5 === 0 && !E.search(e.n).includes(e)).map((e) => e.s + " " + e.n);
    vazio(ruins, "emoji que a busca não acha");
});

t("search: funciona sem ninguém ter chamado load()", () => {
    const novo = load("Emoji.js");
    eq(novo.search("café")[0].s, "☕");
});

// --- valores tortos vindos do ajustes.json (emoji.recent, emoji.tone) ---

t("fromChars e list: um nome de Object.prototype não vira emoji", () => {
    igual(E.fromChars(["constructor", "toString", "__proto__", "hasOwnProperty"]), []);
    igual(E.list("constructor"), []);
    igual(E.list("__proto__"), []);
});

t("withTone: um tom que não é 1 a 5 vale como sem tom", () => {
    eq(E.withTone("👋~", "length"), "👋");
    eq(E.withTone("👋~", 6), "👋");
    eq(E.withTone("👋~", -1), "👋");
    eq(E.withTone("👋~", 1.5), "👋");
    eq(E.withTone("👋~", null), "👋");
});
