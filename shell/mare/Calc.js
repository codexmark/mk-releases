.pragma library

// Calculadora da busca: "12,5 * 3", "2^10", "(1200 - 15%) / 4", "raiz(81)".
// Um analisador próprio (sem eval): números, + - * x / ^ %, parênteses,
// raiz/sqrt e pi. Devolve null quando o texto não é uma conta.

function evaluate(text) {
    var s = String(text).toLowerCase().replace(/\s+/g, "").replace(/×/g, "*").replace(/÷/g, "/");
    if (s === "" || !/[0-9]/.test(s) || !/[-+*x\/^%(]|raiz|sqrt/.test(s))
        return null;
    var i = 0;

    function peek() { return s[i]; }
    function fail() { throw new Error("conta"); }

    function number() {
        var m = /^(\d+(?:[.,]\d+)?|[.,]\d+)/.exec(s.slice(i));
        if (!m) fail();
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
            v = { percent: v };
        }
        return v;
    }
    function plain(v) { return typeof v === "object" ? v.percent / 100 : v; }
    function unary() {
        if (peek() === "-") { i++; return -plain(unary()); }
        if (peek() === "+") { i++; return plain(unary()); }
        return atom();
    }
    function power() {
        var base = unary();
        if (peek() === "^") { i++; return Math.pow(plain(base), plain(power())); }
        return base;
    }
    function product() {
        var v = power();
        while (peek() === "*" || peek() === "x" || peek() === "/") {
            var op = s[i++], r = plain(power());
            v = op === "/" ? plain(v) / r : plain(v) * r;
        }
        return v;
    }
    function sum() {
        var v = plain(product());
        while (peek() === "+" || peek() === "-") {
            var op = s[i++], r = product();
            r = typeof r === "object" ? v * r.percent / 100 : r;
            v = op === "+" ? v + r : v - r;
        }
        return v;
    }

    try {
        var out = sum();
        if (i !== s.length || !isFinite(out))
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
