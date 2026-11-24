// shell/mare/Calc.js: a calculadora da busca. O que o cabeçalho promete:
// números, + - * x / ^ %, parênteses, raiz/sqrt e pi; null quando o texto não
// é uma conta; e nada de eval.
import { t, eq, perto, vazio, assert } from "./harness.mjs";
import { load, source } from "./load.mjs";

const Calc = load("Calc.js");
const calc = (s) => Calc.evaluate(s);

// Toda função do arquivo tem caso aqui; uma nova sem teste reprova.
t("o arquivo exporta evaluate e format, e mais nada", () => {
    assert.deepStrictEqual(Object.keys(Calc).sort(), ["evaluate", "format"]);
});

// --- os exemplos do cabeçalho ---

t("cabeçalho: 12,5 * 3 = 37,5", () => perto(calc("12,5 * 3"), 37.5));
t("cabeçalho: 2^10 = 1024", () => eq(calc("2^10"), 1024));
t("cabeçalho: (1200 - 15%) / 4 = 255", () => perto(calc("(1200 - 15%) / 4"), 255));
t("cabeçalho: raiz(81) = 9", () => eq(calc("raiz(81)"), 9));

// --- as quatro operações e a precedência ---

t("soma e subtração, da esquerda para a direita", () => {
    eq(calc("1+2"), 3);
    eq(calc("10-4-3"), 3);
    eq(calc("10 - 4 + 3"), 9);
});

t("multiplicação e divisão vêm antes de soma e subtração", () => {
    eq(calc("2+3*4"), 14);
    eq(calc("2*3+4"), 10);
    eq(calc("10-6/2"), 7);
    eq(calc("100/10/2"), 5);
    eq(calc("2*3*4"), 24);
});

t("a potência vem antes da multiplicação e associa à direita", () => {
    eq(calc("2*3^2"), 18);
    eq(calc("2^3^2"), 512);
    eq(calc("2^-1"), 0.5);
});

t("os parênteses mudam a ordem, e podem ir um dentro do outro", () => {
    eq(calc("(2+3)*4"), 20);
    eq(calc("2*(3+4)"), 14);
    eq(calc("((2+3)*(4-1))^2"), 225);
    eq(calc("(7)+1"), 8);
});

t("x, × e * multiplicam; ÷ e / dividem", () => {
    eq(calc("3x4"), 12);
    eq(calc("3 X 4"), 12);
    eq(calc("3×4"), 12);
    eq(calc("3*4"), 12);
    eq(calc("12÷4"), 3);
    eq(calc("12/4"), 3);
});

t("menos e mais na frente de um número", () => {
    eq(calc("-5+3"), -2);
    eq(calc("+5+3"), 8);
    eq(calc("3*-2"), -6);
    eq(calc("3--2"), 5);
    eq(calc("-(2+3)"), -5);
});

t("os espaços não importam", () => {
    eq(calc("  1 +   2  "), 3);
    eq(calc("1\t+\n2"), 3);
});

// --- números ---

t("vírgula e ponto valem como separador decimal", () => {
    perto(calc("1,5+1,5"), 3);
    perto(calc("1.5+1.5"), 3);
    perto(calc("0,1+0,2"), 0.3);
    perto(calc(",5+.5"), 1);
});

t("um número com dois separadores não é número", () => {
    eq(calc("1.234,5+1"), null);
    eq(calc("1,2,3+1"), null);
});

// --- porcentagem ---

t("somar ou tirar N% vale N% do que está à esquerda", () => {
    perto(calc("200 - 15%"), 170);
    perto(calc("200 + 15%"), 230);
    perto(calc("100 + 10% + 10%"), 121);
});

t("fora da soma, N% é N/100", () => {
    perto(calc("15%"), 0.15);
    perto(calc("200 * 15%"), 30);
    perto(calc("50% * 80"), 40);
    perto(calc("30 / 50%"), 60);
    perto(calc("(15%)*100"), 15);
});

t("% não é resto de divisão: '5%3' não é conta", () => eq(calc("5%3"), null));

// --- raiz e pi ---

t("raiz e sqrt são a mesma coisa, com ou sem parênteses", () => {
    eq(calc("raiz(16)"), 4);
    eq(calc("sqrt(16)"), 4);
    eq(calc("RAIZ(16)"), 4);
    eq(calc("raiz 16"), 4);
    eq(calc("sqrt(16)+1"), 5);
    eq(calc("2*raiz(9)"), 6);
    eq(calc("raiz(raiz(16))"), 2);
    eq(calc("raiz(3^2+4^2)"), 5);
});

t("raiz de uma porcentagem: raiz 25% = 0,5, com ou sem parênteses", () => {
    perto(calc("raiz(25%)"), 0.5);
    perto(calc("raiz 25%"), 0.5);
});

t("raiz de número negativo não tem resultado: null", () => eq(calc("raiz(-1)"), null));

t("pi entra numa conta", () => {
    perto(calc("2*pi"), 2 * Math.PI);
    perto(calc("pi*2"), 2 * Math.PI);
    perto(calc("PI/2"), Math.PI / 2);
    perto(calc("pi^2+1"), Math.PI * Math.PI + 1);
});

// --- o que não tem resultado ---

t("divisão por zero devolve null, não Infinity nem NaN", () => {
    eq(calc("1/0"), null);
    eq(calc("-1/0"), null);
    eq(calc("0/0"), null);
    eq(calc("5/(2-2)"), null);
});

t("um resultado grande demais para caber devolve null", () => eq(calc("10^999"), null));

// --- texto que não é conta ---

t("texto, vazio e um número sozinho não são conta: null", () => {
    const naoEConta = ["", "   ", "firefox", "terminal", "42", "3,14", "pi", "raiz", "sqrt",
        "abc+def", "+", "-", "()", "%", "linux 6", "x11", "ipv4 10.0.0.1", "foo(1)", "vscode-2"];
    vazio(naoEConta.filter((s) => calc(s) !== null).map((s) => JSON.stringify(s) + " -> " + calc(s)),
        "deviam dar null");
});

t("conta pela metade ou torta devolve null", () => {
    const tortas = ["1+", "*2", "2**3", "(1+2", "1+2)", "2x", "1//2", "2^", "1+()", "raiz()",
        "1e3+1", "2pi", "1+2=", "1+2;", "1,+2", "3 + quatro"];
    vazio(tortas.filter((s) => calc(s) !== null).map((s) => JSON.stringify(s) + " -> " + calc(s)),
        "deviam dar null");
});

t("o que não é texto não derruba: null", () => {
    eq(calc(null), null);
    eq(calc(undefined), null);
    eq(calc({}), null);
    eq(calc([]), null);
    eq(calc(42), null);
});

// --- sem eval ---

t("a fonte não usa eval, Function nem nada que rode texto como código", () => {
    const semComentario = source("Calc.js").replace(/\/\/.*$/gm, "");
    for (const proibido of [/\beval\b/, /\bFunction\b/, /\bconstructor\b/, /\bimport\b/, /\brequire\b/,
        /\bsetTimeout\b/, /\bQt\./])
        assert.ok(!proibido.test(semComentario), "Calc.js usa " + proibido);
});

t("entrada maliciosa devolve null e não roda nada", () => {
    // Se o analisador passasse o texto a eval ou a Function, estas armadilhas disparavam.
    const evalDeVerdade = globalThis.eval, FunctionDeVerdade = globalThis.Function;
    let rodou = [];
    globalThis.eval = (s) => { rodou.push("eval: " + s); };
    globalThis.Function = function (...a) { rodou.push("Function: " + a.join(",")); return () => {}; };
    globalThis.__calcInvadido = 0;
    const ataques = [
        "1+process.exit(1)", "process.exit(1)", "1;globalThis.__calcInvadido=1", "globalThis.__calcInvadido=1",
        "(function(){globalThis.__calcInvadido=1})()+1", "1+constructor.constructor('return 1')()",
        "[].constructor.constructor('globalThis.__calcInvadido=1')()", "1+this", "1+__proto__", "(1).toString()",
        "1+`${globalThis.__calcInvadido=1}`", "1+require('fs')", "import('fs')+1", "eval('1+1')", "1+eval('1')",
        "Qt.quit()+1", "1+/a/", "0x10+1&&1", "1+1//2", "1+alert(1)", "1,__calcInvadido=1",
        "2*(globalThis.__calcInvadido=1)", "toString+1", "valueOf()+1", "1+\u0000", "1+'1'", "1+\"1\""
    ];
    let resultados;
    try {
        resultados = ataques.map((s) => [s, calc(s)]);
    } finally {
        globalThis.eval = evalDeVerdade;
        globalThis.Function = FunctionDeVerdade;
    }
    const invadido = globalThis.__calcInvadido;
    delete globalThis.__calcInvadido;
    vazio(rodou, "código rodado");
    eq(invadido, 0);
    vazio(resultados.filter((r) => r[1] !== null).map((r) => JSON.stringify(r[0]) + " -> " + r[1]), "deviam dar null");
});

t("uma conta enorme ou funda demais não derruba o shell", () => {
    const funda = "(".repeat(200000) + "1" + ")".repeat(200000);
    assert.doesNotThrow(() => calc(funda));
    assert.doesNotThrow(() => calc("-".repeat(200000) + "1"));
    assert.doesNotThrow(() => calc("raiz".repeat(50000) + "1"));
    eq(calc(new Array(5001).fill("1").join("+")), 5001);
    eq(calc("(".repeat(50) + "1+1" + ")".repeat(50)), 2);
});

// --- format ---

t("format: o exemplo do cabeçalho, 1234567.5 -> 1.234.567,5", () => eq(Calc.format(1234567.5), "1.234.567,5"));

t("format: ponto de milhar e vírgula decimal", () => {
    eq(Calc.format(0), "0");
    eq(Calc.format(7), "7");
    eq(Calc.format(999), "999");
    eq(Calc.format(1000), "1.000");
    eq(Calc.format(1000000), "1.000.000");
    eq(Calc.format(0.5), "0,5");
    eq(Calc.format(1234.5678), "1.234,5678");
});

t("format: os decimais não ganham ponto de milhar", () => eq(Calc.format(0.12345678), "0,12345678"));

t("format: o lixo do ponto flutuante some (8 casas)", () => {
    eq(Calc.format(0.1 + 0.2), "0,3");
    eq(Calc.format(1 / 3), "0,33333333");
    eq(Calc.format(2.675 * 100), "267,5");
});

t("format: negativo com o sinal de menos tipográfico, e nunca '-0'", () => {
    eq(Calc.format(-1234.5), "−1.234,5");
    eq(Calc.format(-0), "0");
    eq(Calc.format(-0.000000001), "0");
});

t("format: número grande sai como é, sem o erro da conta de arredondar", () => {
    eq(Calc.format(1e15), "1.000.000.000.000.000");
    eq(Calc.format(1e17), "100.000.000.000.000.000");
    eq(Calc.format(123456789012345), "123.456.789.012.345");
    eq(Calc.format(calc("10^15")), "1.000.000.000.000.000");
    eq(Calc.format(1234567890.12), "1.234.567.890,12");
    eq(Calc.format(100000000.1 + 0.2), "100.000.000,3");
    eq(Calc.format(-1e15), "−1.000.000.000.000.000");
});

t("format: número em notação científica sai inteiro, com vírgula", () => {
    eq(Calc.format(1e21), "1e+21");
    eq(Calc.format(1.5e21), "1,5e+21");
});

t("format mostra o que evaluate calcula", () => {
    eq(Calc.format(calc("(1200 - 15%) / 4")), "255");
    eq(Calc.format(calc("0,1 + 0,2")), "0,3");
    eq(Calc.format(calc("10 / 4")), "2,5");
    eq(Calc.format(calc("1000 * 1000")), "1.000.000");
});
