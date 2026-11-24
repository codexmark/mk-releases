// Carrega um .js do shell (shell/mare/*.js) como módulo no node.
//
//   import { load } from "./load.mjs";
//   const Calc = load("Calc.js");
//   Calc.evaluate("2+2");
//
// Esses arquivos são bibliotecas do QML, não módulos: começam com
// `.pragma library`, podem ter `.import "Outro.js" as Outro` e declaram
// `var` e `function` soltos no topo. Aqui as linhas com ponto saem (viram
// linhas vazias, para o número da linha de um erro continuar certo), cada
// .import é carregado do mesmo jeito e entra com o nome dado, e o que volta é
// um objeto com tudo que o arquivo declara no topo (coluna 0).
//
// As propriedades são vivas: se uma função do arquivo troca uma variável do
// topo (`all = []` no Emoji.js), o objeto mostra o valor novo. Cada load() é
// uma cópia nova do arquivo: um teste que estraga os dados não suja o outro.
// O código roda no mesmo mundo do teste (vm.runInThisContext), então um array
// que ele devolve é um Array daqui e o deepStrictEqual funciona.
import { readFileSync } from "node:fs";
import { dirname, resolve, isAbsolute } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const here = dirname(fileURLToPath(import.meta.url));

// Onde os .js do shell moram.
export const SHELL = resolve(here, "../../shell/mare");
export const REPO = resolve(here, "../..");

// O texto do arquivo, para os testes que olham a fonte (sem eval, cabeçalho).
export function source(name) {
    return readFileSync(isAbsolute(name) ? name : resolve(SHELL, name), "utf8");
}

export function load(name, loading = []) {
    const path = isAbsolute(name) ? name : resolve(SHELL, name);
    if (loading.includes(path))
        throw new Error("load: .import em círculo: " + loading.concat(path).join(" -> "));
    const imports = {};
    const lines = readFileSync(path, "utf8").split("\n").map((line, n) => {
        // Só `.pragma` e `.import` são diretivas; uma linha de código que
        // continua com ".replace(...)" fica como está.
        if (!/^\s*\.(pragma|import)\b/.test(line))
            return line;
        if (/^\s*\.pragma\s+library\s*$/.test(line))
            return "";
        const m = /^\s*\.import\s+"([^"]+)"\s+as\s+([A-Za-z_$][\w$]*)\s*$/.exec(line);
        if (!m)
            throw new Error("load: " + path + ":" + (n + 1) + ": não sei o que fazer com: " + line.trim());
        imports[m[2]] = load(resolve(dirname(path), m[1]), loading.concat(path));
        return "";
    });
    const body = lines.join("\n");

    // O que o arquivo declara no topo: var e function na coluna 0.
    const names = [];
    for (const m of body.matchAll(/^(?:var|let|const)\s+([A-Za-z_$][\w$]*)|^function\s+([A-Za-z_$][\w$]*)/gm))
        if (!names.includes(m[1] || m[2]))
            names.push(m[1] || m[2]);
    const getters = names.map((n) => "get " + JSON.stringify(n) + "() { return " + n + "; }").join(", ");

    // Tudo na primeira linha do invólucro: o corpo começa na linha 1, como no arquivo.
    const wrapped = "(function (" + Object.keys(imports).join(", ") + ") { " + body
        + "\n;return { " + getters + " };\n})";
    const fn = vm.runInThisContext(wrapped, { filename: path });
    return fn(...Object.values(imports));
}
