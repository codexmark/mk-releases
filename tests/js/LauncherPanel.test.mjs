// shell/mare/LauncherPanel.qml, o trecho entre "puro: início" e "puro: fim":
// a nota de cada item na busca (prefixo > início de palavra > trecho > letras
// na ordem), a lista ordenada e o primeiro workspace livre para um projeto.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("LauncherPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["freeWorkspace", "ranked", "score"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("LauncherPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const item = (title, detail) => ({ title: title, detail: detail || "", icon: "", glyph: "", run: () => title });
const titulos = (lista) => lista.map((i) => i.title);

// Uma lista como a do painel: comandos do projeto, do Maré, projetos e apps.
const TUDO = [
    item("lazygit", "git · loja-api"), item("lazydocker", "containers · loja-api"), item("terminal", "terminal · loja-api"),
    item("ajustes", "ilha, fundo, janelas, teclado e mouse"), item("capturar a tela", "área, janela, tela inteira ou vídeo"),
    item("área de transferência", "histórico do que você copiou"), item("rede", "Wi-Fi e Ethernet"),
    item("loja-api", "projeto · workspace 2"), item("meu-os", "projeto · abrir em workspace novo"),
    item("Arquivos", "Gerenciador de arquivos"), item("Firefox", "Navegador Web"), item("GNU Image Manipulation Program", "Editor de imagens"),
    item("LibreOffice Writer", "Editor de texto"), item("Visual Studio Code", "Editor de código"), item("Zen Browser", "")
];

t("score: prefixo vale 100", () => {
    eq(M.score("fire", "Firefox"), 100);
    eq(M.score("firefox", "Firefox"), 100);
    eq(M.score("á", "Área de transferência"), 100);
});

t("score: início de palavra (depois de espaço, hífen, sublinhado ou ponto) vale 80", () => {
    eq(M.score("code", "Visual Studio Code"), 80);
    eq(M.score("api", "loja-api"), 80);
    eq(M.score("os", "meu_os"), 80);
    eq(M.score("desktop", "org.gnome.desktop"), 80);
});

t("score: trecho no meio de uma palavra vale 60", () => {
    eq(M.score("fox", "Firefox"), 60);
    eq(M.score("zy", "lazygit"), 60);
});

t("score: letras na ordem, salteadas, valem 30", () => {
    eq(M.score("vsc", "Visual Studio Code"), 30);
    eq(M.score("lzg", "lazygit"), 30);
    eq(M.score("ffx", "Firefox"), 30);
});

t("score: fora de ordem, faltando letra ou maior que o texto vale 0", () => {
    eq(M.score("xof", "Firefox"), 0);
    eq(M.score("firefoxx", "Firefox"), 0);
    eq(M.score("z", "Firefox"), 0);
    eq(M.score("ff", "f"), 0);
});

t("score: texto vazio, nulo ou ausente vale 0", () => {
    eq(M.score("a", ""), 0);
    eq(M.score("a", null), 0);
    eq(M.score("a", undefined), 0);
});

t("score: a ordem das notas é a prometida", () => {
    const notas = [M.score("co", "code"), M.score("co", "vs code"), M.score("co", "disco"), M.score("co", "cubo")];
    igual(notas, [100, 80, 60, 30]);
});

t("score: o texto digitado é texto, não expressão regular", () => {
    eq(M.score(".*", "Firefox"), 0);
    eq(M.score("c++", "C++ (gcc)"), 100);
    eq(M.score("(", "GIMP (beta)"), 80);
    eq(M.score("\\", "a\\b"), 60);
    eq(M.score("\"", "diz \"oi\""), 80);
    eq(M.score(":", "a:b"), 60);
});

t("ranked: sem texto (ou só espaços), os primeiros da lista na ordem", () => {
    igual(titulos(M.ranked("", TUDO, 7)), titulos(TUDO.slice(0, 7)));
    igual(titulos(M.ranked("   ", TUDO, 3)), ["lazygit", "lazydocker", "terminal"]);
    igual(M.ranked("", [], 7), []);
});

t("ranked: devolve os próprios itens (com o run), não cópias", () => {
    const achados = M.ranked("firefox", TUDO, 7);
    assert.ok(achados[0] === TUDO[10]);
    eq(achados[0].run(), "Firefox");
});

t("ranked: maiúsculas e espaços nas pontas do que se digita não importam", () => {
    eq(titulos(M.ranked("  FIRE ", TUDO, 7))[0], "Firefox");
    igual(titulos(M.ranked("  FIREF ", TUDO, 7)), ["Firefox"]);
    igual(titulos(M.ranked("Área", TUDO, 1)), ["área de transferência"]);
});

t("ranked: o prefixo no título ganha do início de palavra, que ganha do trecho", () => {
    const lista = [item("disco"), item("vs code"), item("cubo"), item("code")];
    igual(titulos(M.ranked("co", lista, 7)), ["code", "vs code", "disco", "cubo"]);
});

t("ranked: o detalhe vale metade do título", () => {
    // "editor" é prefixo do detalhe de três apps (50) e não aparece em título nenhum.
    igual(titulos(M.ranked("editor", TUDO, 7)).sort(), ["GNU Image Manipulation Program", "LibreOffice Writer", "Visual Studio Code"]);
    // Prefixo só no detalhe (50) perde para trecho no título (60) e ganha de letras salteadas no título (30).
    const lista = [item("abc", "git do projeto"), item("legit"), item("g-i-t")];
    igual(titulos(M.ranked("git", lista, 7)), ["legit", "abc", "g-i-t"]);
});

t("ranked: quem não combina fica de fora, e nada combinando dá lista vazia", () => {
    igual(titulos(M.ranked("lazy", TUDO, 7)), ["lazygit", "lazydocker"]);
    igual(M.ranked("qqqq", TUDO, 7), []);
});

t("ranked: corta no máximo de linhas, ficando com os melhores", () => {
    const lista = [item("xa"), item("a1"), item("a2"), item("a3"), item("a4")];
    igual(titulos(M.ranked("a", lista, 4)), ["a1", "a2", "a3", "a4"]);
    eq(M.ranked("e", TUDO, 7).length, 7);
});

t("ranked: item sem detalhe (nulo ou ausente) não quebra a busca", () => {
    const lista = [{ title: "Zen Browser" }, { title: "outro", detail: null }, { title: null, detail: "zen de detalhe" }];
    igual(M.ranked("zen", lista, 7).map((i) => i.title), ["Zen Browser", null]);
});

t("ranked não mexe na lista que recebeu", () => {
    const antes = titulos(TUDO).join("|");
    M.ranked("e", TUDO, 3);
    eq(titulos(TUDO).join("|"), antes);
});

t("freeWorkspace: o primeiro id sem janela e sem projeto", () => {
    eq(M.freeWorkspace([], {}), 1);
    eq(M.freeWorkspace([1, 2, 3], {}), 4);
    eq(M.freeWorkspace([1, 3], {}), 2);
    eq(M.freeWorkspace([], { "1": { name: "loja-api", dir: "/w/loja-api" } }), 2);
    eq(M.freeWorkspace([2], { "1": { name: "a", dir: "/w/a" }, "3": { name: "livre", dir: "" } }), 4);
    eq(M.freeWorkspace([3, 1, 2], { "4": { name: "a", dir: "/w/a" }, "6": { name: "b", dir: "/w/b" } }), 5);
});

t("freeWorkspace: workspace especial (id negativo) não ocupa lugar", () => {
    eq(M.freeWorkspace([-98, -99], {}), 1);
    eq(M.freeWorkspace([-98, 1], {}), 2);
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
