.pragma library

// Calculadora da busca: "12,5 * 3", "2^10", "(1200 - 15%) / 4", "raiz(81)".
// Um analisador próprio (sem eval): números, + - * x / ^ %, parênteses,
// raiz/sqrt e pi. Devolve null quando o texto não é uma conta.
//
// A busca é de apps, projetos e comandos: a calculadora só responde quando o
// texto é claramente uma conta. Na dúvida, null. A regra:
//
//   Precedência  do que prende mais para o que prende menos: parênteses, raiz
//                e %; a potência (associa à direita: 2^3^2 = 2^9); o menos e
//                o mais da frente; * x /; + e -. Então -2^2 = −4, (-2)^2 = 4
//                e 2^-2 = 0,25 (o expoente pode ter o seu sinal).
//   Espaços      valem só entre um termo e outro ("1 + 2", "raiz 16",
//                "3 x 4"). Nunca juntam nada: "1 2 + 3" não é 12 + 3 nem
//                "ra iz" é raiz. Dois números lado a lado não são conta.
//   Números      algarismos com uma vírgula ou um ponto decimal ("12,5",
//                "1.5", ",5"). Zero à esquerda ("007", "01") não é número de
//                conta: é código, hora ou pedaço de data.
//   Uma operação o texto precisa fazer alguma coisa: "42", "-5", "(5)" e "pi"
//                não são conta ("15%" é: vale 0,15).
//   Data         o texto inteiro no formato de data, sem espaços e com o ano
//                de quatro algarismos numa ponta ("2024-11-15", "15/11/2024"),
//                não é conta. Com espaços ("2024 - 11 - 15") é.
//   Hexadecimal  "0x" colado num algarismo ou letra ("0x10", "0xff") não é
//                zero vezes nada. "10x10" e "0 x 10" continuam contas.
//   Resultado    que não é um número finito (1/0, raiz(-1), 10^999) é null.

function evaluate(text) {
    // Só texto: uma lista ["1+1"] viraria "1+1" no String().
    if (typeof text !== "string")
        return null;
    var s = text.toLowerCase().replace(/×/g, "*").replace(/÷/g, "/").replace(/−/g, "-").trim();
    if (s === "" || !/[0-9]/.test(s) || !/[-+*x\/^%(]|raiz|sqrt/.test(s))
        return null;
    // Data inteira, com o mesmo separador nos dois lugares.
    if (/^\d{4}([-\/])\d{1,2}\1\d{1,2}$/.test(s) || /^\d{1,2}([-\/])\d{1,2}\1\d{4}$/.test(s))
        return null;
    // Hexadecimal: um 0 que começa o número, colado no x e no que vem depois.
    if (/(^|[^0-9.,])0x\S/.test(s))
        return null;
    var i = 0;
    // Quantas operações a conta fez: sem nenhuma, é só um número.
    var ops = 0;

    // O espaço separa os termos e só é pulado entre um e outro.
    function peek() {
        while (i < s.length && /\s/.test(s[i]))
            i++;
        return s[i];
    }
    function fail() { throw new Error("conta"); }

    function number() {
        // Um pedaço basta: 400 algarismos já passam do que um double guarda.
        var m = /^(\d+(?:[.,]\d+)?|[.,]\d+)/.exec(s.slice(i, i + 400));
        if (!m || /^0\d/.test(m[0])) fail();
        i += m[0].length;
        return parseFloat(m[0].replace(",", "."));
    }
    function atom() {
        var v;
        if (peek() === "(") {
            i++;
            v = sum();
            if (peek() !== ")") fail();
            i++;
        } else if (s.slice(i, i + 4) === "raiz" || s.slice(i, i + 4) === "sqrt") {
            i += 4;
            ops++;
            v = Math.sqrt(plain(atom()));
        } else if (s.slice(i, i + 2) === "pi") {
            i += 2;
            v = Math.PI;
        } else {
            v = number();
        }
        // 15% = 0,15; em "200 - 15%" vale 15% do termo da esquerda (em sum()).
        if (peek() === "%") {
            i++;
            ops++;
            v = { percent: v };
        }
        return v;
    }
    function plain(v) { return typeof v === "object" ? v.percent / 100 : v; }
    // A potência prende mais que o sinal da frente: -2^2 é -(2^2). O expoente
    // é um unary(), então 2^-2 vale e 2^3^2 continua 2^(3^2).
    function power() {
        var base = atom();
        if (peek() === "^") { i++; ops++; return Math.pow(plain(base), plain(unary())); }
        return base;
    }
    function unary() {
        if (peek() === "-") { i++; return -plain(unary()); }
        if (peek() === "+") { i++; return plain(unary()); }
        return power();
    }
    function product() {
        var v = unary();
        while (peek() === "*" || peek() === "x" || peek() === "/") {
            var op = s[i++], r = plain(unary());
            ops++;
            v = op === "/" ? plain(v) / r : plain(v) * r;
        }
        return v;
    }
    function sum() {
        var v = plain(product());
        while (peek() === "+" || peek() === "-") {
            var op = s[i++], r = product();
            ops++;
            r = typeof r === "object" ? v * r.percent / 100 : r;
            v = op === "+" ? v + r : v - r;
        }
        return v;
    }

    try {
        var out = sum();
        peek();
        if (i !== s.length || ops === 0 || !isFinite(out))
            return null;
        return out;
    } catch (e) {
        return null;
    }
}

// 1234567.5 -> "1.234.567,5"
function format(v) {
    // Oito casas tiram o lixo do ponto flutuante (0,1 + 0,2). Acima de 1e7 a
    // conta v * 1e8 já não cabe exata e estragava o número (10^15 saía
    // 999.999.999.999.999,9): ali valem os 15 dígitos que um double guarda.
    var r = Math.abs(v) < 1e7 ? Math.round(v * 1e8) / 1e8 : Number(v.toPrecision(15));
    var parts = String(Math.abs(r)).split(".");
    if (parts[0].indexOf("e") >= 0 || (parts[1] && parts[1].indexOf("e") >= 0))
        return String(r).replace(".", ",");
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ".");
    return (r < 0 ? "−" : "") + parts.join(",");
}
