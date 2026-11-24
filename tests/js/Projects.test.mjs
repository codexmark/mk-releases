// shell/mare/Projects.qml, o trecho entre "puro: início" e "puro: fim": o
// texto que vira um argumento só para o shell, a linha que abre cada terminal
// e a lista de projetos tirada da saída do find.
import { spawnSync } from "node:child_process";
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("Projects.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["chosenLine", "dirsFrom", "knownTerminal", "lineFor", "shellQuote"];
const CONSTANTES = ["terminals"];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("Projects.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const CONFIG = "/usr/share/mk/shell/ghostty/config";
// O shell roda sem PATH: só funções e comandos internos (printf). Nenhum
// terminal de verdade abre, mesmo instalado nesta máquina.
const sh = (linha) => spawnSync("/bin/sh", ["-c", linha], { encoding: "utf8", env: { PATH: "/nonexistent" } });
const temSh = !sh("exit 0").error;
// O que o shell entende de uma palavra: deve ser o texto original, inteiro.
const peloShell = (palavra) => sh("printf '%s' " + palavra);

const TORTOS = ["", " ", "simples", "com espaço", "it's", "''", "'", "a'b'c", "\"aspas\"", "$HOME", "$(id)", "`id`",
    "a;b", "a&&b", "a|b", "a>b", "*", "~", "\\", "a\\'b", "linha\nnova", "tab\taqui", "ção ñ 日本 🙂", "-n", "--flag=x",
    "/home/u/Work/meu projeto (cópia)/it's", "C:\\Users\\mk: \"x\""];

t("shellQuote: aspas simples em volta, e a aspa simples de dentro escapada", () => {
    eq(M.shellQuote("abc"), "'abc'");
    eq(M.shellQuote(""), "''");
    eq(M.shellQuote("it's"), "'it'\\''s'");
    eq(M.shellQuote("a b"), "'a b'");
    eq(M.shellQuote(42), "'42'");
    eq(M.shellQuote(null), "'null'");
});

t("shellQuote: o shell lê de volta exatamente o texto, seja qual for", () => {
    if (!temSh)
        return;
    const ruins = [];
    for (const texto of TORTOS) {
        const r = peloShell(M.shellQuote(texto));
        if (r.status !== 0 || r.stdout !== texto)
            ruins.push(JSON.stringify(texto) + " voltou " + JSON.stringify(r.stdout) + " (saída " + r.status + ")");
    }
    vazio(ruins, "textos que o shell leu diferente");
});

t("os terminais que o Mk sabe abrir", () => {
    igual(M.terminals, ["ghostty", "kitty", "foot", "alacritty", "wezterm"]);
});

t("knownTerminal: o escolhido, se o Mk sabe abrir; senão, o ghostty", () => {
    for (const nome of ["ghostty", "kitty", "foot", "alacritty", "wezterm"])
        eq(M.knownTerminal(nome), nome);
    eq(M.knownTerminal("xterm"), "ghostty");
    eq(M.knownTerminal(""), "ghostty");
    eq(M.knownTerminal(undefined), "ghostty");
    eq(M.knownTerminal(null), "ghostty");
    eq(M.knownTerminal("Kitty"), "ghostty");
    eq(M.knownTerminal("kitty; rm -rf ~"), "ghostty");
});

t("lineFor: o ghostty leva a config do Mk, opcional, e ignora a do usuário", () => {
    eq(M.lineFor("ghostty", "/home/u/Work/loja", "lazygit", "Mk · git", CONFIG),
        "ghostty --config-default-files=false --config-file='?" + CONFIG + "' --working-directory='/home/u/Work/loja'"
        + " --title='Mk · git' -e lazygit");
    eq(M.lineFor("ghostty", "/home/u", "", "", CONFIG),
        "ghostty --config-default-files=false --config-file='?" + CONFIG + "' --working-directory='/home/u'");
});

t("lineFor: cada terminal com as suas flags", () => {
    eq(M.lineFor("kitty", "/p", "htop", "T", CONFIG), "kitty --directory='/p' --title='T' htop");
    eq(M.lineFor("foot", "/p", "htop", "T", CONFIG), "foot --working-directory='/p' --title='T' htop");
    eq(M.lineFor("alacritty", "/p", "htop", "T", CONFIG), "alacritty --working-directory '/p' --title 'T' -e htop");
    // O wezterm não tem flag de título.
    eq(M.lineFor("wezterm", "/p", "htop", "T", CONFIG), "wezterm start --always-new-process --cwd '/p' -- htop");
});

t("lineFor: sem comando e sem título, só o diretório", () => {
    eq(M.lineFor("kitty", "/p", "", "", CONFIG), "kitty --directory='/p'");
    eq(M.lineFor("foot", "/p", "", undefined, CONFIG), "foot --working-directory='/p'");
    eq(M.lineFor("alacritty", "/p", "", "", CONFIG), "alacritty --working-directory '/p'");
    eq(M.lineFor("wezterm", "/p", "", "", CONFIG), "wezterm start --always-new-process --cwd '/p'");
});

t("lineFor: só o ghostty cita a config do Mk", () => {
    for (const nome of ["kitty", "foot", "alacritty", "wezterm"])
        assert.ok(!M.lineFor(nome, "/p", "x", "T", CONFIG).includes(CONFIG), nome);
});

t("lineFor: terminal desconhecido cai no ghostty", () => {
    eq(M.lineFor("xterm", "/p", "", "", CONFIG), M.lineFor("ghostty", "/p", "", "", CONFIG));
});

t("lineFor: diretório, título e config tortos chegam inteiros ao terminal", () => {
    if (!temSh)
        return;
    const ruins = [];
    for (const nome of M.terminals) {
        for (const texto of TORTOS) {
            // No lugar do terminal, uma função que imprime cada argumento numa linha.
            const linha = M.lineFor(nome, texto, "", texto, "/cfg/" + texto);
            const r = sh(nome + "() { for a in \"$@\"; do printf '<%s>' \"$a\"; done; }; " + linha);
            const esperados = nome === "wezterm" ? ["<" + texto + ">"]
                : nome === "alacritty" ? ["<" + texto + ">"].concat(texto ? ["<--title><" + texto + ">"] : [])
                : ["=" + texto + ">"].concat(texto ? ["<--title=" + texto + ">"] : []);
            if (nome === "ghostty")
                esperados.push("<--config-file=?/cfg/" + texto + ">");
            if (r.status !== 0 || r.stderr !== "" || esperados.some((e) => !r.stdout.includes(e)))
                ruins.push(nome + " com " + JSON.stringify(texto) + ": " + JSON.stringify(r.stdout) + " " + r.stderr.trim());
        }
    }
    vazio(ruins, "linhas que o shell leu diferente");
});

t("chosenLine: com o ghostty escolhido é a linha dele, sem if", () => {
    eq(M.chosenLine("ghostty", "/p", "lazygit", "", CONFIG, "/home/u"), M.lineFor("ghostty", "/p", "lazygit", "", CONFIG));
});

t("chosenLine: outro terminal vai com o ghostty de reserva, se foi desinstalado", () => {
    eq(M.chosenLine("kitty", "/p", "lazygit", "T", CONFIG, "/home/u"),
        "if command -v kitty >/dev/null 2>&1; then " + M.lineFor("kitty", "/p", "lazygit", "T", CONFIG)
        + "; else " + M.lineFor("ghostty", "/p", "lazygit", "T", CONFIG) + "; fi");
});

t("chosenLine: sem diretório (vazio, nulo) abre na pasta pessoal", () => {
    for (const dir of ["", null, undefined]) {
        eq(M.chosenLine("ghostty", dir, "", "", CONFIG, "/home/u"), M.lineFor("ghostty", "/home/u", "", "", CONFIG));
        assert.ok(M.chosenLine("foot", dir, "", "", CONFIG, "/home/u").includes("foot --working-directory='/home/u'"));
    }
});

t("chosenLine: o shell roda o ramo certo, com e sem o terminal instalado", () => {
    if (!temSh)
        return;
    const linha = M.chosenLine("kitty", "/meu projeto", "", "", CONFIG, "/home/u");
    const eco = "() { printf '%s' \"$*\"; }; ";
    // `command -v` acha uma função do shell: com a função kitty, é o ramo do kitty.
    const com = sh("kitty" + eco + "ghostty() { printf 'reserva'; }; " + linha);
    eq(com.stdout, "--directory=/meu projeto");
    const sem = sh("ghostty() { printf 'reserva'; }; " + linha);
    eq(sem.stdout, "reserva");
});

t("dirsFrom: um caminho por linha vira { name, dir }, em ordem de nome", () => {
    igual(M.dirsFrom("/home/u/Work/zeta\n/home/u/Work/cliente/loja-api\n/home/u/Work/projetos/meu-os\n"), [
        { name: "loja-api", dir: "/home/u/Work/cliente/loja-api" },
        { name: "meu-os", dir: "/home/u/Work/projetos/meu-os" },
        { name: "zeta", dir: "/home/u/Work/zeta" }
    ]);
});

t("dirsFrom: vazio e linhas em branco não viram projeto", () => {
    igual(M.dirsFrom(""), []);
    igual(M.dirsFrom("\n\n"), []);
    igual(M.dirsFrom("/w/a\n\n/w/b\n").map((d) => d.name), ["a", "b"]);
});

t("dirsFrom: nome com espaço, acento, dois-pontos e aspas fica inteiro", () => {
    igual(M.dirsFrom("/w/meu projeto (cópia)\n/w/a:b\n/w/it's \"x\"\n").map((d) => d.name).sort(),
        ["a:b", "it's \"x\"", "meu projeto (cópia)"].sort());
    igual(M.dirsFrom("/w/ação")[0], { name: "ação", dir: "/w/ação" });
});

t("dirsFrom: dois projetos com o mesmo nome em pastas diferentes aparecem os dois", () => {
    const d = M.dirsFrom("/w/b/api\n/w/a/api\n");
    eq(d.length, 2);
    igual(d.map((x) => x.name), ["api", "api"]);
    igual(d.map((x) => x.dir).sort(), ["/w/a/api", "/w/b/api"]);
});

t("dirsFrom: a mesma saída dá a mesma lista (quem compara não refaz a busca à toa)", () => {
    const saida = "/w/b\n/w/a\n/w/c\n";
    eq(JSON.stringify(M.dirsFrom(saida)), JSON.stringify(M.dirsFrom(saida)));
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
