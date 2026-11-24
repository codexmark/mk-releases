// Arnês dos testes em node (tests/js/*.test.mjs). O mesmo contrato do
// tests/lib.sh: uma linha "ok - nome" ou "not ok - nome" por caso, o detalhe
// de quem falhou logo abaixo, e saída 1 se algo falhou.
//
//   import { t, eq, igual, assert } from "./harness.mjs";
//   t("o que se espera", () => { eq(obtido, esperado); });
//
//   t(nome, fn)            um caso; passa se fn não lançar
//   eq(obtido, esperado)   idênticos (===)
//   igual(obtido, esperado) mesma estrutura (arrays e objetos por dentro)
//   perto(obtido, esperado) números iguais a menos do erro de ponto flutuante
//   vazio(lista, oQue)     a lista de problemas está vazia (mostra todos)
//   assert                 o node:assert/strict inteiro, para o resto
//
// Um arquivo sem nenhum caso reprova, como no lib.sh.
import assert from "node:assert/strict";

let casos = 0;
let falhas = 0;

export { assert };

export function t(nome, fn) {
    casos++;
    try {
        fn();
        console.log("ok - " + nome);
    } catch (e) {
        falhas++;
        process.exitCode = 1;
        console.log("not ok - " + nome);
        const texto = e instanceof Error ? (e.message || String(e)) : String(e);
        // Onde no arquivo de teste a verificação estourou.
        const onde = e instanceof Error && e.stack
            ? (e.stack.split("\n").find((l) => /\.test\.mjs:\d+/.test(l)) || "").trim() : "";
        const linhas = texto.split("\n").slice(0, 40);
        if (!(e instanceof assert.AssertionError) && e instanceof Error)
            linhas.unshift(e.name + " (o código lançou):");
        if (onde)
            linhas.push(onde);
        for (const l of linhas)
            console.log("    # " + l);
    }
}

export function eq(obtido, esperado, msg) {
    assert.strictEqual(obtido, esperado, msg);
}

export function igual(obtido, esperado, msg) {
    assert.deepStrictEqual(obtido, esperado, msg);
}

export function perto(obtido, esperado, msg) {
    assert.ok(typeof obtido === "number" && Math.abs(obtido - esperado) <= 1e-9 * Math.max(1, Math.abs(esperado)),
        (msg ? msg + ": " : "") + "esperado " + esperado + ", obtido " + obtido);
}

// Para as invariantes: junte os problemas numa lista e mostre todos de uma vez.
export function vazio(lista, oQue) {
    assert.ok(lista.length === 0, (oQue || "problemas") + " (" + lista.length + "):\n  " + lista.slice(0, 30).join("\n  "));
}

process.on("exit", () => {
    if (casos === 0) {
        console.log("not ok - " + (process.argv[1] || "o arquivo") + " não tem nenhum caso");
        process.exitCode = 1;
    }
    if (falhas > 0)
        process.exitCode = 1;
});
