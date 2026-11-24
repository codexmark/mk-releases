// Carrega o bloco puro de um .qml do shell (shell/mare/*.qml) no node.
//
//   import { puro } from "./qml-puro.mjs";
//   const M = puro("Monitors.qml");
//   M.timing(1920, 1080, 75, { vmin: 48, vmax: 75, hmin: 30, hmax: 83, clock: 170 });
//
// Um Singleton do Maré pode separar, entre os comentários
//
//     // --- puro: início ---
//     ...
//     // --- puro: fim ---
//
// as funções que não tocam no Maré nem no Hyprland. Só esse trecho é lido
// aqui: cada `function nome(...) {` com 4 espaços de recuo vira uma função do
// objeto devolvido, e cada `readonly property <tipo> nome: valor` do trecho
// vira uma constante (também devolvida). O resto do arquivo (Process, Timer,
// propriedades com estado) não é avaliado.
//
// Uma constante de fora do trecho (`readonly property int minVblank: 25` lá
// em cima) entra junto se alguma função pura a usa e o valor é um literal.
// Qualquer outra coisa de fora que o trecho use (uma propriedade com estado,
// uma função do painel, um id, o `root`, um singleton como Settings) é um
// defeito do "puro": o puro() lança dizendo quem usa o quê. A procura é pelo
// texto, função por função (um parâmetro ou variável local com o mesmo nome
// não conta); e, como rede de baixo, ler um desses nomes durante um teste
// também lança.
//
// As linhas ficam no mesmo número do .qml: o erro de um teste aponta para a
// linha certa do Monitors.qml. Cada puro() é uma cópia nova. O código roda no
// mesmo mundo do teste (vm.runInThisContext), então um array que ele devolve
// é um Array daqui e o deepStrictEqual funciona.
import { readFileSync } from "node:fs";
import { isAbsolute, resolve } from "node:path";
import vm from "node:vm";
import { SHELL } from "./load.mjs";

const INICIO = /^\s*\/\/\s*-+\s*puro:\s*início\b/;
const FIM = /^\s*\/\/\s*-+\s*puro:\s*fim\b/;
const FUNCAO = /^ {4}function\s+([A-Za-z_$][\w$]*)\s*\(/;
const PROPRIEDADE = /^(\s*)(readonly\s+)?(?:default\s+|required\s+)*property\s+(?:[\w.]+(?:<[\w.]+>)?)\s+([A-Za-z_$][\w$]*)\s*(?::\s*(.*))?$/;
const ID = /^\s*id:\s*([A-Za-z_$][\w$]*)\s*$/;

// O que o QML põe ao alcance de qualquer função e o node não tem.
const DO_QML = ["root", "Qt", "Quickshell", "Hyprland", "Settings", "qsTr", "print", "gc"];

// Tira comentários, textos e expressões regulares, para sobrar só o código.
function soCodigo(texto) {
    return texto
        .replace(/"(?:\\.|[^"\\\n])*"|'(?:\\.|[^'\\\n])*'|`(?:\\.|[^`\\])*`/g, '""')
        .replace(/\/\/.*$/gm, "")
        .replace(/\/\*[\s\S]*?\*\//g, "")
        .replace(/(?<=[(,=:!&|?]\s*)\/(?![\/*\s])(?:\\.|[^\/\n\\])+\/[gimsuy]*/g, '""');
}

// O nome aparece como variável (não como `x.nome` nem como chave `{ nome: ... }`)?
function usa(codigo, nome) {
    const re = new RegExp("(?<![.\\w$])" + nome.replace(/\$/g, "\\$") + "(?![\\w$])", "g");
    for (const m of codigo.matchAll(re)) {
        const antes = codigo.slice(0, m.index).trimEnd().slice(-1);
        const depois = codigo.slice(m.index + nome.length).trimStart();
        const chave = (antes === "{" || antes === ",") && /^:/.test(depois);
        if (!chave)
            return true;
    }
    return false;
}

// O nome é parâmetro ou variável declarada neste trecho?
function declara(codigo, nome) {
    const n = nome.replace(/\$/g, "\\$");
    return new RegExp("\\b(?:const|let|var)\\s+(?:[\\w$]+\\s*(?:=[^,;]*)?,\\s*)*" + n + "(?![\\w$])").test(codigo)
        || new RegExp("(?<![.\\w$])" + n + "\\s*=>").test(codigo)
        || [...codigo.matchAll(/\(([^()]*)\)\s*(?:=>|\{)/g)].some((m) =>
            m[1].split(",").some((p) => p.trim().replace(/\s*=.*$/, "") === nome));
}

// Um valor que não depende de nada: número, texto, lista ou objeto de literais.
function literal(valor) {
    const v = soCodigo(valor).replace(/\b(?:true|false|null|Infinity|NaN)\b/g, "");
    return valor.trim() !== "" && !/[A-Za-z_$]/.test(v.replace(/\b\d[\d.]*e[+-]?\d+\b/gi, ""))
        && !/[{]/.test(v);
}

// As partes do arquivo: { linhas, de, ate, funcoes, constantes, fora, constantesDeFora }.
export function fonte(name) {
    const path = isAbsolute(name) ? name : resolve(SHELL, name);
    const linhas = readFileSync(path, "utf8").split("\n");
    const de = linhas.findIndex((l) => INICIO.test(l));
    const ate = linhas.findIndex((l) => FIM.test(l));
    if (de < 0 || ate < 0 || ate < de)
        throw new Error("puro: " + path + " não tem o trecho entre '// --- puro: início ---' e '// --- puro: fim ---'");

    const funcoes = [], constantes = [], comEstado = [];
    const fora = [], constantesDeFora = {};
    linhas.forEach((linha, n) => {
        const dentro = n > de && n < ate;
        const f = FUNCAO.exec(linha), p = PROPRIEDADE.exec(linha), id = ID.exec(linha);
        if (dentro) {
            if (f)
                funcoes.push(f[1]);
            if (p && p[1].length === 4)
                (p[2] ? constantes : comEstado).push(p[3]);
            return;
        }
        if (p && p[2] && literal(p[4] || ""))
            constantesDeFora[p[3]] = p[4];
        const nome = (/^\s*function\s+([A-Za-z_$][\w$]*)/.exec(linha) || [])[1] || (p && p[3]) || (id && id[1]);
        if (nome && !fora.includes(nome))
            fora.push(nome);
    });
    if (comEstado.length > 0)
        throw new Error("puro: " + path + ": o trecho puro declara propriedade com estado (não é readonly): "
            + comEstado.join(", "));
    return { path, linhas, de, ate, funcoes, constantes, fora, constantesDeFora };
}

export function puro(name) {
    const f = fonte(name);
    const { path, linhas, de, ate, funcoes, constantes } = f;
    const proprios = funcoes.concat(constantes);
    const proibidos = f.fora.concat(DO_QML).filter((n) => !proprios.includes(n));

    // Cada membro do trecho (função ou constante) com o seu código, para
    // dizer quem usa o que é de fora.
    const membros = [];
    for (let n = de + 1; n < ate; n++) {
        const fn = FUNCAO.exec(linhas[n]), p = PROPRIEDADE.exec(linhas[n]);
        if (fn || (p && p[1].length === 4))
            membros.push({ nome: fn ? fn[1] : p[3], linha: n + 1, texto: [] });
        if (membros.length > 0)
            membros[membros.length - 1].texto.push(linhas[n]);
    }
    const precisa = [], defeitos = [];
    for (const m of membros) {
        const codigo = soCodigo(m.texto.join("\n"));
        for (const nome of proibidos) {
            if (!usa(codigo, nome) || declara(codigo, nome))
                continue;
            if (nome in f.constantesDeFora) {
                if (!precisa.includes(nome))
                    precisa.push(nome);
            } else {
                defeitos.push(m.nome + " (linha " + m.linha + ") usa '" + nome + "'");
            }
        }
        // Um tipo ou singleton do QML que não está na lista: começa com
        // maiúscula e o node não conhece.
        for (const x of codigo.matchAll(/(?<![.\w$])([A-Z][\w$]*)\s*\./g))
            if (!(x[1] in globalThis) && !proibidos.includes(x[1]) && !declara(codigo, x[1]))
                defeitos.push(m.nome + " (linha " + m.linha + ") usa '" + x[1] + "'");
    }
    if (defeitos.length > 0)
        throw new Error("puro: " + path + ": o trecho puro usa o que está fora dele e não é constante:\n  "
            + defeitos.join("\n  "));

    // O trecho, nas mesmas linhas do arquivo. `readonly property T nome: v`
    // vira `const nome = v`.
    const corpo = linhas.map((linha, n) => {
        if (n <= de || n >= ate)
            return "";
        const p = PROPRIEDADE.exec(linha);
        if (p && p[1].length === 4) {
            if (p[4] === undefined || p[4].trim() === "")
                throw new Error("puro: " + path + ":" + (n + 1) + ": constante sem valor: " + linha.trim());
            return p[1] + "const " + p[3] + " = " + p[4];
        }
        return linha;
    }).join("\n");

    const deFora = precisa.map((n) => "const " + n + " = " + f.constantesDeFora[n] + "; ").join("");
    const nomes = funcoes.concat(constantes, precisa);
    const getters = nomes.map((n) => "get " + JSON.stringify(n) + "() { return " + n + "; }").join(", ");
    // A rede de baixo: quem ler um nome de fora durante o teste cai aqui.
    const guarda = new Proxy({}, {
        has: (_, nome) => typeof nome === "string" && proibidos.includes(nome) && !precisa.includes(nome),
        get: (_, nome) => {
            if (typeof nome !== "string")
                return undefined;
            throw new Error("puro: " + path + ": o trecho puro leu '" + nome + "', que está fora dele");
        },
        set: (_, nome) => {
            throw new Error("puro: " + path + ": o trecho puro escreveu em '" + String(nome) + "', que está fora dele");
        }
    });
    // Tudo na primeira linha do invólucro: o corpo fica nas linhas do arquivo.
    const embrulho = "(function (__fora) { with (__fora) { return (function () { " + deFora + corpo
        + "\n;return { " + getters + " };\n})(); } })";
    return vm.runInThisContext(embrulho, { filename: path })(guarda);
}
