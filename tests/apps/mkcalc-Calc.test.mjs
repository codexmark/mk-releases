// apps/mkcalc/qml/Calc.js: a lógica da janela do MkCalc que não depende do
// QML: o que cada tecla faz com a expressão, a troca de sinal, o continuar do
// resultado, a arrumação do que é colado e a tradução do teclado físico.
import { readFileSync } from "node:fs";
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const C = load(REPO + "/apps/mkcalc/qml/Calc.js");
// Os mesmos casos que o teste em Go do motor usa (a regra é do contrato).
const numeros = JSON.parse(readFileSync(REPO + "/apps/mkcalc/src/internal/conta/testdata/numeros.json", "utf8"));

// Digita uma sequência de teclas (separadas por espaço) a partir de um estado.
const digita = (teclas, de) => teclas.split(" ").filter((x) => x !== "").reduce((e, k) => C.tecla(e, k), de || C.novo());
// O estado depois de um "=" que o motor respondeu.
const feito = (expressao, cru, resultado, aproximado) =>
    C.confirmado(C.abre(expressao.replace(/ /g, "")), { expressao, cru, resultado: resultado || cru, aproximado: !!aproximado });
const expr = (teclas, de) => digita(teclas, de).expr;

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["novo", "abre", "vazio", "confirmado", "eAlgarismo", "numeroNoFim", "abertos", "fimDeOperando", "naFrente",
        "semVirgula", "tecla", "comAlgarismo", "comVirgula", "doResultado", "comOperador", "comSimbolo", "trocarSinal",
        "doHistorico", "normalizarNumero", "normalizar", "colar", "agrupar", "mostrar", "paraMotor", "temConta", "visor",
        "paraCopiar", "deTexto", "acoes", "comandoDeExemplo"];
    const funcoes = Object.keys(C).filter((k) => typeof C[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("o arquivo não faz conta: nada de eval, Function, parseFloat nem Number sobre a expressão", () => {
    const fonte = readFileSync(REPO + "/apps/mkcalc/qml/Calc.js", "utf8").split("\n").filter((l) => !/^\s*\/\//.test(l)).join("\n");
    for (const proibido of [/\beval\b/, /\bFunction\(/, /parseFloat/, /parseInt/, /Math\.(pow|sqrt|round)/, /toFixed/])
        eq(proibido.test(fonte), false, String(proibido));
    // O único Number() é o do id do pedido de exemplo.
    eq((fonte.match(/\bNumber\(/g) || []).length, 1);
});

t("novo, abre e vazio: o estado de partida e o de edição", () => {
    igual(C.novo(), { expr: "", fechado: false, cru: "", resultado: "", aproximado: false });
    igual(C.abre("2+"), { expr: "2+", fechado: false, cru: "", resultado: "", aproximado: false });
    eq(C.vazio(C.novo()), true);
    eq(C.vazio(C.abre("2")), false);
    eq(C.vazio(feito("0 × 5", "0")), false);
});

t("confirmado: a expressão arrumada pelo motor fica, sem os espaços, e fechada", () => {
    const e = C.confirmado(C.abre("200+10%"), { expressao: "200 + 10%", resultado: "220", cru: "220", aproximado: false });
    igual(e, { expr: "200+10%", fechado: true, cru: "220", resultado: "220", aproximado: false });
    eq(C.confirmado(C.abre("2÷3"), { expressao: "2 ÷ 3", resultado: "0,666666666667", cru: "0,666666666667", aproximado: true }).aproximado, true);
    // Uma resposta sem a expressão (não devia acontecer) não apaga a que está à vista.
    eq(C.confirmado(C.abre("2+2"), { resultado: "4", cru: "4" }).expr, "2+2");
});

t("as leituras do texto: eAlgarismo, numeroNoFim, abertos, fimDeOperando, naFrente, semVirgula", () => {
    eq(C.eAlgarismo("7"), true);
    for (const c of ["", "a", ",", "12", "×"])
        eq(C.eAlgarismo(c), false, c);
    eq(C.numeroNoFim("12+3,45"), "3,45");
    eq(C.numeroNoFim("12+"), "");
    eq(C.numeroNoFim(""), "");
    eq(C.abertos("((1+2)×(3"), 2);
    eq(C.abertos("(1))"), 0);
    for (const s of ["2", "(2)", "2%"])
        eq(C.fimDeOperando(s), true, s);
    for (const s of ["", "2+", "(", "√", "2,"])
        eq(C.fimDeOperando(s), false, s);
    eq(C.naFrente("-3", 0), true);
    eq(C.naFrente("2×-3", 2), true);
    eq(C.naFrente("(-3", 1), true);
    eq(C.naFrente("√-3", 1), true);
    eq(C.naFrente("2-3", 1), false);
    eq(C.naFrente("2%-3", 2), false);
    eq(C.naFrente("2×3", 1), false);
    eq(C.semVirgula("2,"), "2");
    eq(C.semVirgula("2,5"), "2,5");
});

t("algarismos: escrevem o número; o zero sozinho na frente é trocado", () => {
    eq(expr("1 2 3"), "123");
    eq(expr("0 5"), "5");
    eq(expr("0 0 0"), "0");
    eq(expr("1 0 0"), "100");
    eq(expr("2 + 0 7"), "2+7");
    eq(expr("0 , 0 5"), "0,05");
    // Depois de ")" e de "%" o número multiplica, com o × à vista.
    eq(expr("( 2 ) 3"), "(2)×3");
    eq(expr("5 % 3"), "5%×3");
});

t("vírgula: uma por número, e sozinha vira 0,", () => {
    eq(expr(","), "0,");
    eq(expr("1 , 5"), "1,5");
    eq(expr("1 , 5 , 2"), "1,52");
    eq(expr("1 , ,"), "1,");
    eq(expr("2 + ,"), "2+0,");
    eq(expr("1 , 5 + 2 , 5"), "1,5+2,5");
    eq(expr("( 2 ) , 5"), "(2)×0,5");
});

t("operadores: um em cima do outro troca; o - depois de × ÷ ^ ( é o sinal do número", () => {
    eq(expr("2 + 3"), "2+3");
    eq(expr("2 + ×"), "2×");
    eq(expr("2 × ÷ ^ +"), "2+");
    eq(expr("2 + -"), "2-");
    eq(expr("2 - -"), "2-");
    eq(expr("2 × - 3"), "2×-3");
    eq(expr("2 ÷ -"), "2÷-");
    eq(expr("2 ^ - 3"), "2^-3");
    eq(expr("2 × - -"), "2×-");
    // Outro operador em cima do sinal solto desfaz os dois.
    eq(expr("2 × - +"), "2+");
    eq(expr("2 × - ÷"), "2÷");
    // No começo só o menos vale; os outros não fazem nada.
    eq(expr("+"), "");
    eq(expr("×"), "");
    eq(expr("-"), "-");
    eq(expr("- -"), "-");
    eq(expr("- +"), "");
    eq(expr("- 5 + 2"), "-5+2");
    eq(expr("( ×"), "(");
    eq(expr("( - 2"), "(-2");
    eq(expr("( - +"), "(");
    eq(expr("√ +"), "√");
    // A vírgula solta sai antes do operador.
    eq(expr("2 , + 3"), "2+3");
    eq(expr("2 ) "), "2");
});

t("% ( ) √: só onde fazem sentido", () => {
    eq(expr("2 0 0 + 1 0 %"), "200+10%");
    eq(expr("%"), "");
    eq(expr("2 + %"), "2+");
    eq(expr("5 % %"), "5%%");
    eq(expr("5 , %"), "5%");
    eq(expr("( 2 + 3 )"), "(2+3)");
    eq(expr(")"), "");
    eq(expr("2 )"), "2");
    eq(expr("( )"), "(");
    eq(expr("( 2 + )"), "(2+");
    eq(expr("( 2 ) )"), "(2)");
    eq(expr("( ( 2 ) )"), "((2))");
    eq(expr("2 ("), "2×(");
    eq(expr("2 , ("), "2×(");
    eq(expr("( 2 ) ("), "(2)×(");
    eq(expr("2 + ("), "2+(");
    eq(expr("√ 9"), "√9");
    eq(expr("2 √ 9"), "2×√9");
    eq(expr("2 × √ 9"), "2×√9");
    eq(expr("√ √ 1 6"), "√√16");
});

t("apagar e limpar", () => {
    eq(expr("1 2 3 apagar"), "12");
    eq(expr("2 × - apagar"), "2×");
    eq(expr("2 √ apagar apagar apagar"), "");
    eq(expr("apagar"), "");
    igual(digita("1 2 + 3 limpar"), C.novo());
    igual(C.tecla(feito("2 + 2", "4"), "limpar"), C.novo());
    // Depois do "=", apagar volta a editar a conta, inteira.
    igual(C.tecla(feito("200 + 10%", "220"), "apagar"), C.abre("200+10%"));
});

t("continuar a conta: depois do =, um operador começa do resultado e um algarismo começa conta nova", () => {
    const e = feito("200 + 10%", "220");
    eq(expr("+ 5", e), "220+5");
    eq(expr("×", e), "220×");
    eq(expr("÷ 2", e), "220÷2");
    eq(expr("-", e), "220-");
    eq(expr("^ 2", e), "220^2");
    eq(expr("%", e), "220%");
    eq(expr("7", e), "7");
    eq(expr(",", e), "0,");
    eq(expr("(", e), "(");
    eq(expr(")", e), "200+10%");
    eq(C.tecla(e, ")").fechado, true);
    eq(expr("√", e), "√(220)");
    eq(C.tecla(e, "+").fechado, false);
    // O resultado com vírgula e o aproximado continuam com o cru (sem milhar).
    eq(expr("× 2", feito("1234,5 × 1000", "1234500", "1.234.500")), "1234500×2");
    eq(expr("+ 1", feito("2 ÷ 3", "0,666666666667", "0,666666666667", true)), "0,666666666667+1");
    // Resultado negativo: -5^2 seria -(5^2), então vai entre parênteses.
    const negativo = feito("2 - 7", "-5");
    eq(expr("^ 2", negativo), "(-5)^2");
    eq(expr("× 2", negativo), "-5×2");
    eq(expr("-", negativo), "-5-");
    eq(C.doResultado(negativo, "^"), "(-5)");
    eq(C.doResultado(negativo, "+"), "-5");
    eq(C.doResultado(e, "^"), "220");
});

t("trocar o sinal: do último número, do parêntese, e do resultado", () => {
    eq(expr("3 sinal"), "-3");
    eq(expr("3 sinal sinal"), "3");
    eq(expr("5 × 3 sinal"), "5×-3");
    eq(expr("5 × 3 sinal sinal"), "5×3");
    eq(expr("5 + 3 sinal"), "5-3");
    eq(expr("5 - 3 sinal"), "5+3");
    eq(expr("2 ^ 3 sinal"), "2^-3");
    eq(expr("1 , 5 sinal"), "-1,5");
    eq(expr("1 , sinal 5"), "-1,5");
    eq(expr("( 3 sinal"), "(-3");
    eq(expr("( 2 + 3 ) sinal"), "-(2+3)");
    eq(expr("( 2 + 3 ) sinal sinal"), "(2+3)");
    eq(expr("4 × ( 2 + 3 ) sinal"), "4×-(2+3)");
    eq(expr("4 + ( ( 2 ) × 3 ) sinal"), "4-((2)×3)");
    eq(expr("5 0 % sinal"), "-50%");
    eq(expr("2 + 5 0 % sinal"), "2-50%");
    eq(expr("√ 9 sinal"), "-√9");
    eq(expr("2 + √ 9 sinal"), "2-√9");
    eq(expr("2 × √ √ 9 sinal"), "2×-√√9");
    // Sem número ainda: liga e desliga o sinal do que vai ser digitado.
    eq(expr("sinal"), "-");
    eq(expr("sinal sinal"), "");
    eq(expr("sinal 5"), "-5");
    eq(expr("2 × sinal"), "2×-");
    eq(expr("2 × sinal sinal"), "2×");
    eq(expr("2 + sinal"), "2-");
    eq(expr("2 - sinal"), "2+");
    eq(expr("( sinal 3"), "(-3");
    eq(expr("√ sinal"), "√");
    // Um ")" sem par (só vem colado) não quebra nada.
    igual(C.trocarSinal(C.abre("2)")), C.abre("2)"));
    // Depois do "=": o resultado com o sinal trocado começa conta nova.
    igual(C.tecla(feito("2 + 2", "4"), "sinal"), C.abre("-4"));
    igual(C.tecla(feito("2 - 7", "-5"), "sinal"), C.abre("5"));
    igual(C.tecla(feito("2 - 2", "0"), "sinal"), C.abre("0"));
    eq(expr("sinal + 1", feito("0,5 × 1", "0,5")), "-0,5+1");
});

t("tecla: o que não é tecla não faz nada, e a expressão tem um tamanho máximo", () => {
    const e = C.abre("2+");
    for (const k of ["", "igual", "esc", "copiar", "x", "12", "++", null, undefined, 5])
        eq(C.tecla(e, k), e, String(k));
    const cheia = C.abre("9".repeat(C.LIMITE));
    eq(C.tecla(cheia, "9"), cheia);
    eq(C.tecla(cheia, "+"), cheia);
    eq(C.tecla(cheia, "sinal"), cheia);
    eq(C.tecla(cheia, "apagar").expr.length, C.LIMITE - 1);
    // Nenhuma tecla mexe no estado que recebeu.
    const fixo = Object.freeze(C.abre("2+3"));
    for (const k of ["1", ",", "+", "-", "%", "(", ")", "√", "sinal", "apagar", "limpar"])
        C.tecla(fixo, k);
    eq(fixo.expr, "2+3");
});

t("normalizarNumero: a regra do ponto e da vírgula, com os casos do contrato (os mesmos do motor)", () => {
    eq(numeros.length >= 30, true);
    for (const c of numeros)
        eq(C.normalizarNumero(c.texto), c.le, c.texto);
});

t("normalizar: o que é colado vira texto de editar", () => {
    const casos = {
        "1.234,56": "1234,56",
        "1,234.56": "1234,56",
        "1.234": "1234",
        "1,234": "1,234",
        "3.14": "3,14",
        "1.234.567": "1234567",
        "R$ 1.234,56": "1234,56",
        "  42\n": "42",
        "\n\n3 + 4\n5 + 6": "3+4",
        "2 * 3": "2×3",
        "2 x 3": "2×3",
        "2X3": "2×3",
        "6 / 2": "6÷2",
        "6 : 2": "6÷2",
        "2 ** 8": "2^8",
        "5 − 3": "5-3",
        "5 – 3": "5-3",
        "raiz(9)": "√(9)",
        "SQRT(16) + 1": "√(16)+1",
        "= 2 + 2": "2+2",
        "2 + 2 =": "2+2",
        "(1.000 + 2.000,5) × 10%": "(1000+2000,5)×10%",
        "1 234,5": "1234,5",
        "-0,50": "-0,5",
        "007": "7",
        "√2": "√2"
    };
    for (const [colado, esperado] of Object.entries(casos))
        eq(C.normalizar(colado), esperado, JSON.stringify(colado));
    for (const lixo of ["", "   ", "\n", "abc", "2 + dois", "1.2.3", "1,,2", "2 + 2 = 4", "http://exemplo.test", "rm -rf", "$(touch x)", "2;3", "=", null, undefined])
        eq(C.normalizar(lixo), null, JSON.stringify(lixo));
});

t("colar: junta na expressão, continua do resultado ou começa conta nova", () => {
    eq(C.colar(C.novo(), "1.234,56").expr, "1234,56");
    eq(C.colar(C.abre("2+"), "R$ 3,50").expr, "2+3,5");
    eq(C.colar(C.abre("2"), "5").expr, "25");
    eq(C.colar(C.abre("2"), "(3+4)").expr, "2×(3+4)");
    eq(C.colar(C.abre("2,"), "raiz(9)").expr, "2×√(9)");
    eq(C.colar(C.abre("(2)"), "3").expr, "(2)×3");
    eq(C.colar(C.abre("5%"), "3").expr, "5%×3");
    eq(C.colar(C.abre("2+"), "(3)").expr, "2+(3)");
    // Depois do "=": o que começa com operador continua do resultado.
    const e = feito("200 + 10%", "220");
    eq(C.colar(e, "+ 5").expr, "220+5");
    eq(C.colar(e, "* 2").expr, "220×2");
    eq(C.colar(e, "42").expr, "42");
    eq(C.colar(e, "-3").expr, "-3");
    eq(C.colar(e, "42").fechado, false);
    eq(C.colar(feito("2 - 7", "-5"), "^2").expr, "(-5)^2");
    // O que não é conta não muda nada: a janela recebe null e avisa.
    eq(C.colar(C.abre("2+"), "abc"), null);
    eq(C.colar(e, ""), null);
    eq(C.colar(C.abre("9".repeat(C.LIMITE - 1)), "99"), null);
    eq(C.colar(C.novo(), "9".repeat(C.LIMITE)).expr.length, C.LIMITE);
});

t("doHistorico: clicar numa conta traz o resultado dela para a expressão", () => {
    const item = { id: 2, expressao: "200 + 10%", resultado: "220", cru: "220", aproximado: false, quando_s: 1 };
    const negativo = { id: 3, expressao: "2 - 7", resultado: "-5", cru: "-5", aproximado: false, quando_s: 2 };
    igual(C.doHistorico(C.novo(), item), C.abre("220"));
    igual(C.doHistorico(feito("1 + 1", "2"), item), C.abre("220"));
    // Completa a conta que espera um número…
    eq(C.doHistorico(C.abre("2×"), item).expr, "2×220");
    eq(C.doHistorico(C.abre("2×("), item).expr, "2×(220");
    eq(C.doHistorico(C.abre("√"), item).expr, "√220");
    eq(C.doHistorico(C.abre("2-"), negativo).expr, "2-(-5)");
    // …e, se a expressão já termina num número, começa outra com ele.
    eq(C.doHistorico(C.abre("2×3"), item).expr, "220");
    eq(C.doHistorico(C.abre("2,"), item).expr, "220");
    eq(C.doHistorico(C.novo(), negativo).expr, "-5");
    const cheia = C.abre("9".repeat(C.LIMITE - 1) + "+");
    eq(C.doHistorico(cheia, item), cheia);
});

t("agrupar e mostrar: a expressão do visor tem milhar e espaços, como a do motor", () => {
    eq(C.agrupar("1"), "1");
    eq(C.agrupar("123"), "123");
    eq(C.agrupar("1234"), "1.234");
    eq(C.agrupar("1234567"), "1.234.567");
    eq(C.mostrar(""), "");
    eq(C.mostrar("1234567,891234"), "1.234.567,891234");
    eq(C.mostrar("1000+2000,5"), "1.000 + 2.000,5");
    eq(C.mostrar("200+10%"), "200 + 10%");
    eq(C.mostrar("5--3"), "5 - -3");
    eq(C.mostrar("-5×-3"), "-5 × -3");
    eq(C.mostrar("2^-3"), "2^-3");
    eq(C.mostrar("2×(3+4)%"), "2 × (3 + 4)%");
    eq(C.mostrar("√(9)÷2"), "√(9) ÷ 2");
    eq(C.mostrar("1000+"), "1.000 +");
    eq(C.mostrar("1000,"), "1.000,");
    eq(C.mostrar("0,0001234"), "0,0001234");
    // O que o motor arruma, sem os espaços, é mostrado igual ao que ele arrumou.
    for (const arrumada of ["2 + 2", "200 + 10%", "5 - -3", "2^-3", "√(9) × 2", "2 × (3 + 4)%", "-8^(1 ÷ 3)", "7 ÷ 0,5"])
        eq(C.mostrar(arrumada.replace(/ /g, "")), arrumada);
});

t("paraMotor: fecha os parênteses que faltam e tira a vírgula solta", () => {
    eq(C.paraMotor("2+3"), "2+3");
    eq(C.paraMotor("(2+3"), "(2+3)");
    eq(C.paraMotor("2×((1+2)×(3"), "2×((1+2)×(3))");
    eq(C.paraMotor("2,"), "2");
    eq(C.paraMotor("(2,"), "(2)");
    eq(C.paraMotor("2 + 2"), "2+2");
    eq(C.paraMotor(""), "");
});

t("temConta: só pede a prévia quando há uma conta, e não um número só", () => {
    for (const s of ["2+3", "2+", "50%", "√9", "-5×2", "2^3", "(2)-1", "6÷2"])
        eq(C.temConta(s), true, s);
    for (const s of ["", "5", "-5", "1234,5", "(5)", "-", "("])
        eq(C.temConta(s), false, s);
});

t("visor: o que a linha de baixo mostra em cada situação", () => {
    const aberto = C.abre("2+3");
    igual(C.visor(C.novo(), null, ""), { tipo: "vazio", texto: "", sinal: "" });
    igual(C.visor(aberto, { id: 1, expressao: "2 + 3", resultado: "5", cru: "5", aproximado: false }, ""), { tipo: "previa", texto: "5", sinal: "=" });
    igual(C.visor(C.abre("2÷3"), { resultado: "0,666666666667", cru: "0,666666666667", aproximado: true }, ""),
        { tipo: "previa", texto: "0,666666666667", sinal: "≈" });
    // Expressão ainda incompleta: nada, sem reclamar.
    igual(C.visor(C.abre("2+"), { erro: "expressão incompleta", incompleta: true }, ""), { tipo: "vazio", texto: "", sinal: "" });
    // A prévia que não sai avisa baixinho; o "=" recusado é erro.
    igual(C.visor(C.abre("1÷0"), { erro: "divisão por zero", incompleta: false }, ""), { tipo: "aviso", texto: "divisão por zero", sinal: "" });
    igual(C.visor(C.abre("1÷0"), { erro: "divisão por zero", incompleta: false }, "divisão por zero"), { tipo: "erro", texto: "divisão por zero", sinal: "" });
    igual(C.visor(C.abre("2+"), null, "expressão incompleta"), { tipo: "erro", texto: "expressão incompleta", sinal: "" });
    igual(C.visor(feito("200 + 10%", "220"), null, ""), { tipo: "resultado", texto: "220", sinal: "=" });
    igual(C.visor(feito("1234,5 × 1000", "1234500", "1.234.500"), null, ""), { tipo: "resultado", texto: "1.234.500", sinal: "=" });
    igual(C.visor(feito("2 ÷ 3", "0,666666666667", "0,666666666667", true), null, ""), { tipo: "resultado", texto: "0,666666666667", sinal: "≈" });
});

t("paraCopiar: o resultado cru, o da prévia, ou o número escrito", () => {
    eq(C.paraCopiar(feito("1234,5 × 1000", "1234500", "1.234.500"), null), "1234500");
    eq(C.paraCopiar(feito("2 ÷ 3", "0,666666666667", "0,666666666667", true), null), "0,666666666667");
    eq(C.paraCopiar(C.abre("1000×2"), { resultado: "2.000", cru: "2000", aproximado: false }), "2000");
    eq(C.paraCopiar(C.abre("1÷0"), { erro: "divisão por zero", incompleta: false }), "");
    eq(C.paraCopiar(C.abre("1234,5"), null), "1234,5");
    eq(C.paraCopiar(C.abre("-12,"), null), "-12");
    eq(C.paraCopiar(C.abre("2+"), null), "");
    eq(C.paraCopiar(C.novo(), null), "");
});

t("deTexto: cada caractere do teclado vira a tecla certa", () => {
    for (const d of "0123456789")
        eq(C.deTexto(d), d);
    const mapa = {
        ",": ",", ".": ",", "+": "+", "-": "-", "*": "×", "x": "×", "X": "×", "×": "×", "/": "÷", ":": "÷", "÷": "÷",
        "^": "^", "%": "%", "(": "(", ")": ")", "=": "igual", "s": "sinal", "S": "sinal", "±": "sinal", "c": "limpar", "C": "limpar",
        "r": "√", "R": "√", "√": "√"
    };
    for (const [c, tecla] of Object.entries(mapa))
        eq(C.deTexto(c), tecla, c);
    for (const c of ["", "a", "q", " ", "\t", "\r", "é", "12", "++", null, undefined, 7])
        eq(C.deTexto(c), "", String(c));
    // Tudo que o deTexto devolve (menos "igual") é tecla que o tecla() conhece.
    const partida = C.abre("(2+3");
    for (const tecla of new Set(Object.values(mapa)))
        if (tecla !== "igual")
            eq(C.tecla(partida, tecla) !== partida, true, tecla);
});

t("acoes: o teclado físico inteiro, com Ctrl, Enter, Esc e Backspace", () => {
    const K = C.TECLA;
    const M = C.MOD;
    igual(C.acoes(0x37, "7", 0), ["7"]);
    igual(C.acoes(0x2b, "+", M.shift), ["+"]);
    igual(C.acoes(0x2a, "*", M.shift), ["×"]);
    igual(C.acoes(0x58, "x", 0), ["×"]);
    igual(C.acoes(0x2f, "/", 0), ["÷"]);
    igual(C.acoes(0x2e, ".", 0), [","]);
    igual(C.acoes(0x53, "s", 0), ["sinal"]);
    igual(C.acoes(K.c, "c", 0), ["limpar"]);
    igual(C.acoes(K.c, "C", M.shift), ["limpar"]);
    igual(C.acoes(0x3d, "=", 0), ["igual"]);
    igual(C.acoes(K.retorno, "\r", 0), ["igual"]);
    igual(C.acoes(K.enter, "\r", M.numerico), ["igual"]);
    igual(C.acoes(K.apagar, "\b", 0), ["apagar"]);
    igual(C.acoes(K.esc, "\u001b", 0), ["esc"]);
    igual(C.acoes(K.del, "\u007f", 0), ["limpar"]);
    igual(C.acoes(K.c, "\u0003", M.ctrl), ["copiar"]);
    igual(C.acoes(K.v, "\u0016", M.ctrl), ["colar"]);
    igual(C.acoes(K.h, "\b", M.ctrl), ["historico"]);
    igual(C.acoes(K.insert, "", M.ctrl), ["copiar"]);
    igual(C.acoes(K.insert, "", M.shift), ["colar"]);
    // Ctrl com outra tecla, Alt e Super não são da calculadora.
    igual(C.acoes(0x53, "\u0013", M.ctrl), []);
    igual(C.acoes(0x37, "7", M.ctrl), []);
    igual(C.acoes(0x37, "7", M.alt), []);
    igual(C.acoes(K.c, "c", M.meta), []);
    // Setas e teclas de função fora do teclado numérico não fazem nada.
    for (const k of [K.esquerda, K.direita, K.cima, K.baixo, K.home, K.fim, K.paginaAcima, K.paginaAbaixo, K.insert, 0x01000030, 0x01000001])
        igual(C.acoes(k, "", 0), [], "tecla " + k.toString(16));
    igual(C.acoes(0x41, "a", 0), []);
    // O ^ de tecla morta chega sem texto; e o texto composto vem inteiro.
    igual(C.acoes(K.circunflexoMorto, "", 0), ["^"]);
    igual(C.acoes(0x32, "^2", 0), ["^", "2"]);
    igual(C.acoes(0, "", 0), []);
    igual(C.acoes(0x37, "7"), ["7"]);
});

t("acoes: o teclado numérico vale com e sem NumLock", () => {
    const K = C.TECLA;
    const N = C.MOD.numerico;
    // Com NumLock: chega o texto.
    for (const d of "0123456789")
        igual(C.acoes(0x30 + Number.parseInt(d, 10), d, N), [d]);
    igual(C.acoes(0x2c, ",", N), [","]);
    igual(C.acoes(0x2e, ".", N), [","]);
    igual(C.acoes(0x2b, "+", N), ["+"]);
    igual(C.acoes(0x2d, "-", N), ["-"]);
    igual(C.acoes(0x2a, "*", N), ["×"]);
    igual(C.acoes(0x2f, "/", N), ["÷"]);
    // Sem NumLock: chega a função da tecla, com a marca do teclado numérico.
    const sem = [[K.insert, "0"], [K.fim, "1"], [K.baixo, "2"], [K.paginaAbaixo, "3"], [K.esquerda, "4"], [K.meio, "5"],
        [K.direita, "6"], [K.home, "7"], [K.cima, "8"], [K.paginaAcima, "9"], [K.del, ","]];
    for (const [k, tecla] of sem)
        igual(C.acoes(k, "", N), [tecla], tecla);
    igual(C.acoes(K.fim, "", N | C.MOD.shift), ["1"]);
    // Os códigos são os do Qt (Qt::Key e Qt::KeyboardModifier).
    igual([K.esc, K.apagar, K.retorno, K.enter, K.insert, K.del, K.home, K.fim, K.esquerda, K.cima, K.direita, K.baixo, K.paginaAcima, K.paginaAbaixo, K.meio],
        [0x01000000, 0x01000003, 0x01000004, 0x01000005, 0x01000006, 0x01000007, 0x01000010, 0x01000011, 0x01000012, 0x01000013,
            0x01000014, 0x01000015, 0x01000016, 0x01000017, 0x0100000b]);
    igual([C.MOD.shift, C.MOD.ctrl, C.MOD.alt, C.MOD.meta, C.MOD.numerico], [0x02000000, 0x04000000, 0x08000000, 0x10000000, 0x20000000]);
});

t("uma conta inteira pelo teclado: digitar, prever, confirmar, continuar", () => {
    let e = C.novo();
    const aperta = (codigo, texto, mods) => {
        for (const a of C.acoes(codigo, texto, mods))
            e = C.tecla(e, a);
    };
    for (const c of "200+10%")
        aperta(c.charCodeAt(0), c, 0);
    eq(e.expr, "200+10%");
    eq(C.temConta(e.expr), true);
    eq(C.paraMotor(e.expr), "200+10%");
    e = C.confirmado(e, { expressao: "200 + 10%", resultado: "220", cru: "220", aproximado: false });
    eq(C.mostrar(e.expr), "200 + 10%");
    eq(C.visor(e, null, "").texto, "220");
    aperta(0x2a, "*", 0);
    aperta(C.TECLA.fim, "", C.MOD.numerico);
    aperta(0x2e, ".", 0);
    aperta(0x35, "5", 0);
    eq(e.expr, "220×1,5");
    aperta(0x53, "s", 0);
    eq(e.expr, "220×-1,5");
    aperta(C.TECLA.apagar, "", 0);
    aperta(C.TECLA.apagar, "", 0);
    eq(e.expr, "220×-1");
    aperta(C.TECLA.c, "c", 0);
    eq(C.vazio(e), true);
});

t("comandoDeExemplo: cada pedido vira um jq na resposta gravada, com o id do pedido", () => {
    igual(C.comandoDeExemplo("/ex", { id: 7, op: "calcular", expr: "200+10%" }),
        ["jq", "-c", "--argjson", "id", "7", "--arg", "e", "200+10%", "(.[$e] // .[\"*\"]) + {id: $id}", "/ex/calcular.json"]);
    igual(C.comandoDeExemplo("/ex", { id: 8, op: "historico" }), ["jq", "-c", "--argjson", "id", "8", ". + {id: $id}", "/ex/historico.json"]);
    eq(C.comandoDeExemplo("/ex", { id: 9, op: "apagar", item: 2 }).pop(), "/ex/apagar.json");
    eq(C.comandoDeExemplo("/ex", { id: 10, op: "limpar" }).pop(), "/ex/limpar.json");
    // Um pedido torto não escolhe arquivo nem vira argumento do jq.
    eq(C.comandoDeExemplo("/ex", { id: "1; rm", op: "../../etc/passwd" }).pop(), "/ex/historico.json");
    eq(C.comandoDeExemplo("/ex", { id: "1; rm", op: "limpar" })[4], "0");
    // A expressão vai como argumento (--arg), nunca dentro do filtro.
    const c = C.comandoDeExemplo("/ex", { id: 1, op: "calcular", expr: "\") | halt" });
    eq(c[7], "\") | halt");
    eq(c[8].includes("halt"), false);
});
