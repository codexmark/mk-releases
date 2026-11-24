.pragma library

// A lógica da janela do MkCalc que não depende do QML: o que cada tecla faz
// com o texto da expressão, a troca de sinal, o continuar do resultado, a
// arrumação do que é colado e a tradução das teclas. Nada aqui faz conta: o
// resultado é sempre do motor (../CONTRATO.md). Testado no node
// (tests/apps/mkcalc-Calc.test.mjs).
//
// O estado é { expr, fechado, cru, resultado, aproximado }:
//   expr     a expressão, no jeito de editar: sem espaços, sem milhar, com
//            vírgula decimal e os sinais × ÷ √ (o motor lê isso direto)
//   fechado  true logo depois do "=": expr é a conta feita, e cru, resultado
//            e aproximado são a resposta do motor
// Toda função devolve um estado novo; nenhuma mexe no que recebeu.

// Quantos caracteres a expressão pode ter. Cabe o maior resultado que o motor
// dá (uns 3.100 caracteres) com folga para continuar a conta; o motor aceita
// até 5.000 (o resultado mostrado, com os pontos de milhar, também cabe).
var LIMITE = 3500;

var OPERADORES = "+-×÷^";

// Os códigos do Qt (Qt::Key e Qt::KeyboardModifier) que a janela usa.
var TECLA = {
    esc: 0x01000000, apagar: 0x01000003, retorno: 0x01000004, enter: 0x01000005,
    insert: 0x01000006, del: 0x01000007, home: 0x01000010, fim: 0x01000011,
    esquerda: 0x01000012, cima: 0x01000013, direita: 0x01000014, baixo: 0x01000015,
    paginaAcima: 0x01000016, paginaAbaixo: 0x01000017, meio: 0x0100000b,
    circunflexoMorto: 0x01001252, c: 0x43, h: 0x48, v: 0x56
};
var MOD = { shift: 0x02000000, ctrl: 0x04000000, alt: 0x08000000, meta: 0x10000000, numerico: 0x20000000 };

// O teclado numérico sem NumLock: a tecla chega com o nome da função dela.
var SEM_NUMLOCK = {};
SEM_NUMLOCK[TECLA.insert] = "0";
SEM_NUMLOCK[TECLA.fim] = "1";
SEM_NUMLOCK[TECLA.baixo] = "2";
SEM_NUMLOCK[TECLA.paginaAbaixo] = "3";
SEM_NUMLOCK[TECLA.esquerda] = "4";
SEM_NUMLOCK[TECLA.meio] = "5";
SEM_NUMLOCK[TECLA.direita] = "6";
SEM_NUMLOCK[TECLA.home] = "7";
SEM_NUMLOCK[TECLA.cima] = "8";
SEM_NUMLOCK[TECLA.paginaAcima] = "9";
SEM_NUMLOCK[TECLA.del] = ",";

// ── o estado ───────────────────────────────────────────────────────────

function novo() {
    return { expr: "", fechado: false, cru: "", resultado: "", aproximado: false };
}

// Uma expressão em edição (sem resultado).
function abre(expr) {
    return { expr: expr, fechado: false, cru: "", resultado: "", aproximado: false };
}

// Nada no visor: é quando o Esc fecha a janela.
function vazio(e) {
    return e.expr === "" && !e.fechado;
}

// O "=" deu certo: a expressão arrumada pelo motor fica à vista, fechada.
function confirmado(e, resposta) {
    return {
        expr: String(resposta.expressao || e.expr).replace(/\s+/g, ""),
        fechado: true,
        cru: String(resposta.cru),
        resultado: String(resposta.resultado),
        aproximado: resposta.aproximado === true
    };
}

// ── ler o texto ────────────────────────────────────────────────────────

function eAlgarismo(c) {
    return c >= "0" && c <= "9" && c.length === 1;
}

// O número (algarismos e vírgula) em que a expressão termina, ou "".
function numeroNoFim(expr) {
    let i = expr.length;
    while (i > 0 && (eAlgarismo(expr[i - 1]) || expr[i - 1] === ","))
        i--;
    return expr.slice(i);
}

// Quantos "(" ainda esperam o ")".
function abertos(expr) {
    let n = 0;
    for (const c of expr)
        n += c === "(" ? 1 : c === ")" ? -1 : 0;
    return Math.max(n, 0);
}

// A expressão termina em algo que vale um número: algarismo, ")" ou "%".
function fimDeOperando(expr) {
    const u = expr.slice(-1);
    return u !== "" && (eAlgarismo(u) || u === ")" || u === "%");
}

// O + ou - da posição i é um sinal na frente de um número (e não uma conta
// entre dois): está no começo ou depois de um operador, "(" ou "√".
function naFrente(expr, i) {
    return (expr[i] === "-" || expr[i] === "+") && (i === 0 || "+-×÷^(√".indexOf(expr[i - 1]) >= 0);
}

// Sem a vírgula solta do fim ("2," vira "2").
function semVirgula(expr) {
    return expr.slice(-1) === "," ? expr.slice(0, -1) : expr;
}

// ── as teclas ──────────────────────────────────────────────────────────

// O que a tecla t faz com o estado. t é um algarismo, "," ou um de
// + - × ÷ ^ % ( ) √, ou "apagar", "limpar", "sinal". O resto não faz nada.
function tecla(e, t) {
    let r;
    if (t === "limpar")
        return novo();
    if (t === "apagar")
        return e.fechado ? abre(e.expr) : abre(e.expr.slice(0, -1));
    if (t === "sinal")
        r = trocarSinal(e);
    else if (typeof t === "string" && eAlgarismo(t))
        r = comAlgarismo(e, t);
    else if (t === ",")
        r = comVirgula(e);
    else if (typeof t === "string" && t.length === 1 && OPERADORES.indexOf(t) >= 0)
        r = comOperador(e, t);
    else if (t === "%" || t === "(" || t === ")" || t === "√")
        r = comSimbolo(e, t);
    else
        return e;
    return r.expr.length > LIMITE ? e : r;
}

// Um algarismo: depois do "=", começa conta nova.
function comAlgarismo(e, d) {
    const base = e.fechado ? "" : e.expr;
    if (numeroNoFim(base) === "0")
        return abre(base.slice(0, -1) + d);
    const u = base.slice(-1);
    return abre(base + (u === ")" || u === "%" ? "×" : "") + d);
}

function comVirgula(e) {
    const base = e.fechado ? "" : e.expr;
    const n = numeroNoFim(base);
    if (n.indexOf(",") >= 0)
        return e;
    if (n !== "")
        return abre(base + ",");
    const u = base.slice(-1);
    return abre(base + (u === ")" || u === "%" ? "×" : "") + "0,");
}

// O resultado de antes, pronto para continuar a conta com o operador op.
function doResultado(e, op) {
    return e.cru[0] === "-" && op === "^" ? "(" + e.cru + ")" : e.cru;
}

// Um operador de dois lados: depois do "=", continua do resultado; em cima de
// outro operador, troca; o "-" depois de × ÷ ^ ( é o sinal do próximo número.
function comOperador(e, op) {
    if (e.fechado)
        return abre(doResultado(e, op) + op);
    const expr = semVirgula(e.expr);
    const u = expr.slice(-1);
    if (expr === "" || u === "(" || u === "√")
        return op === "-" ? abre(expr + "-") : e;
    if (OPERADORES.indexOf(u) < 0)
        return abre(expr + op);
    if (naFrente(expr, expr.length - 1)) {
        // Termina num sinal de menos solto: outro operador o desfaz.
        if (op === "-")
            return abre(expr);
        const antes = expr.slice(0, -1);
        const a = antes.slice(-1);
        return abre(a !== "" && OPERADORES.indexOf(a) >= 0 ? antes.slice(0, -1) + op : antes);
    }
    if (op === "-" && "×÷^".indexOf(u) >= 0)
        return abre(expr + "-");
    return abre(expr.slice(0, -1) + op);
}

function comSimbolo(e, t) {
    if (e.fechado) {
        if (t === "%")
            return abre(e.cru + "%");
        if (t === "√")
            return abre("√(" + e.cru + ")");
        return t === "(" ? abre("(") : e;
    }
    const expr = semVirgula(e.expr);
    const operando = fimDeOperando(expr);
    if (t === "%")
        return operando ? abre(expr + "%") : e;
    if (t === ")")
        return operando && abertos(expr) > 0 ? abre(expr + ")") : e;
    // "(" e "√" depois de um número multiplicam: o × fica à vista.
    return abre(expr + (operando ? "×" : "") + t);
}

// Troca o sinal do último número (ou do último parêntese) da expressão.
// 5×3 vira 5×-3, 5+3 vira 5-3, 5-3 vira 5+3, -3 vira 3; depois do "=", o
// resultado com o sinal trocado começa uma conta nova.
function trocarSinal(e) {
    if (e.fechado) {
        if (/^-?[0,]*$/.test(e.cru))
            return abre(e.cru.replace("-", ""));
        return abre(e.cru[0] === "-" ? e.cru.slice(1) : "-" + e.cru);
    }
    const expr = e.expr;
    // Onde começa o último operando: pula os %, depois o parêntese ou o número.
    let i = expr.length;
    while (i > 0 && expr[i - 1] === "%")
        i--;
    const depois = i;
    if (i > 0 && expr[i - 1] === ")") {
        let fundo = 0;
        for (i = i - 1; i >= 0; i--) {
            if (expr[i] === ")")
                fundo++;
            else if (expr[i] === "(" && --fundo === 0)
                break;
        }
        if (i < 0)
            return e;
    } else {
        while (i > 0 && (eAlgarismo(expr[i - 1]) || expr[i - 1] === ","))
            i--;
    }
    if (i === depois) {
        // Não há número ainda: liga ou desliga o sinal do que vai ser digitado.
        if (depois !== expr.length)
            return e;
        const u = expr.slice(-1);
        if (u === "-" || u === "+") {
            if (naFrente(expr, expr.length - 1))
                return abre(expr.slice(0, -1));
            return abre(expr.slice(0, -1) + (u === "-" ? "+" : "-"));
        }
        return u === "√" ? e : abre(expr + "-");
    }
    while (i > 0 && expr[i - 1] === "√")
        i--;
    const a = i > 0 ? expr[i - 1] : "";
    if (a === "-" && naFrente(expr, i - 1))
        return abre(expr.slice(0, i - 1) + expr.slice(i));
    if (a === "-" || a === "+")
        return abre(expr.slice(0, i - 1) + (a === "-" ? "+" : "-") + expr.slice(i));
    return abre(expr.slice(0, i) + "-" + expr.slice(i));
}

// Clicar numa conta do histórico traz o resultado dela: completa a conta que
// espera um número (2× vira 2×220); senão começa uma conta nova com ele.
function doHistorico(e, item) {
    const v = String(item.cru);
    if (e.fechado || e.expr === "")
        return abre(v);
    const expr = semVirgula(e.expr);
    if (fimDeOperando(expr))
        return abre(v);
    const r = expr + (v[0] === "-" ? "(" + v + ")" : v);
    return r.length > LIMITE ? e : abre(r);
}

// ── colar ──────────────────────────────────────────────────────────────

// Um número colado, pela regra do contrato (a mesma do motor, conferida com
// os mesmos casos): um separador só, uma vez, é o decimal (menos o ponto de
// 1.234, que é milhar); o mesmo separador repetido é milhar; com os dois, o
// último é o decimal. Devolve o número no jeito de editar (sem milhar, com
// vírgula) ou null se não é um número.
function normalizarNumero(s) {
    const pontos = s.split(".").length - 1;
    const virgulas = s.split(",").length - 1;
    const emGrupos = (t, sep) => {
        const partes = t.split(sep);
        return partes.length >= 2 && /^\d{1,3}$/.test(partes[0]) && partes.slice(1).every((p) => /^\d{3}$/.test(p));
    };
    let inteiro = s;
    let frac = "";
    if (pontos + virgulas === 1) {
        const i = s.search(/[.,]/);
        inteiro = s.slice(0, i);
        frac = s.slice(i + 1);
        // 1.234 é mil duzentos e trinta e quatro, como se escreve em português.
        if (s[i] === "." && /^[1-9]\d{0,2}$/.test(inteiro) && /^\d{3}$/.test(frac)) {
            inteiro += frac;
            frac = "";
        }
    } else if (pontos + virgulas > 1 && (pontos === 0 || virgulas === 0)) {
        const sep = pontos === 0 ? "," : ".";
        if (!emGrupos(s, sep))
            return null;
        inteiro = s.split(sep).join("");
    } else if (pontos + virgulas > 1) {
        const i = Math.max(s.lastIndexOf("."), s.lastIndexOf(","));
        const decimal = s[i];
        const milhar = decimal === "." ? "," : ".";
        if (s.split(decimal).length !== 2 || !emGrupos(s.slice(0, i), milhar))
            return null;
        inteiro = s.slice(0, i).split(milhar).join("");
        frac = s.slice(i + 1);
    }
    if (inteiro + frac === "" || !/^\d*$/.test(inteiro) || !/^\d*$/.test(frac))
        return null;
    inteiro = inteiro.replace(/^0+(?=\d)/, "") || "0";
    frac = frac.replace(/0+$/, "");
    return frac === "" ? inteiro : inteiro + "," + frac;
}

// O texto colado, no jeito de editar: a primeira linha, sem espaços, sem "R$",
// sem o "=" das pontas, com os sinônimos trocados (* x / : ** raiz) e os
// números arrumados. null quando sobra algo que não é de conta.
function normalizar(texto) {
    const linha = String(texto === null || texto === undefined ? "" : texto).split(/\r?\n/)
        .map((l) => l.trim()).filter((l) => l !== "")[0];
    if (linha === undefined)
        return null;
    const s = linha.replace(/R\$|[$€£]/g, "").replace(/\s+/g, "").replace(/^=+|=+$/g, "")
        .replace(/raiz|sqrt/gi, "√").replace(/\*\*/g, "^").replace(/[−–—]/g, "-")
        .replace(/[*·xX]/g, "×").replace(/[\/:]/g, "÷");
    let saida = "";
    for (let i = 0; i < s.length;) {
        const c = s[i];
        if (eAlgarismo(c) || c === "." || c === ",") {
            let j = i;
            while (j < s.length && (eAlgarismo(s[j]) || s[j] === "." || s[j] === ","))
                j++;
            const n = normalizarNumero(s.slice(i, j));
            if (n === null)
                return null;
            saida += n;
            i = j;
        } else if ("+-×÷^%()√".indexOf(c) >= 0) {
            saida += c;
            i++;
        } else {
            return null;
        }
    }
    return saida === "" ? null : saida;
}

// Cola um número ou uma expressão. Depois do "=", o que começa com um
// operador continua do resultado; o resto começa conta nova. Devolve null
// quando o texto não é de conta (a janela avisa e não muda nada).
function colar(e, texto) {
    const n = normalizar(texto);
    if (n === null)
        return null;
    let base = e.fechado ? "" : e.expr;
    if (e.fechado && "+×÷^%".indexOf(n[0]) >= 0)
        base = doResultado(e, n[0]);
    else if (!e.fechado && (n[0] === "(" || n[0] === "√") && fimDeOperando(semVirgula(base)))
        base = semVirgula(base) + "×";
    else if (eAlgarismo(n[0]) && (base.slice(-1) === ")" || base.slice(-1) === "%"))
        base += "×";
    return (base + n).length > LIMITE ? null : abre(base + n);
}

// ── mostrar e pedir ────────────────────────────────────────────────────

function agrupar(inteiro) {
    return inteiro.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

// A expressão como aparece no visor e no histórico: milhar com ponto e um
// espaço de cada lado de + - × ÷ (como o motor a arruma). Só para ler: o
// texto que vai para o motor é o de editar.
function mostrar(expr) {
    let saida = "";
    for (let i = 0; i < expr.length;) {
        const c = expr[i];
        if (eAlgarismo(c)) {
            let j = i;
            while (j < expr.length && eAlgarismo(expr[j]))
                j++;
            saida += agrupar(expr.slice(i, j));
            // A parte decimal vai como está, sem agrupar.
            i = j;
            if (expr[i] === ",") {
                j = i + 1;
                while (j < expr.length && eAlgarismo(expr[j]))
                    j++;
                saida += expr.slice(i, j);
                i = j;
            }
        } else {
            saida += "+-×÷".indexOf(c) >= 0 && !naFrente(expr, i) ? " " + c + " " : c;
            i++;
        }
    }
    return saida.replace(/\s+$/, "");
}

// O que vai para o motor: sem a vírgula solta do fim e com os parênteses que
// faltam fechados ("(2+3" calcula como "(2+3)").
function paraMotor(expr) {
    const s = semVirgula(String(expr).replace(/\s+/g, ""));
    return s + ")".repeat(abertos(s));
}

// Vale pedir a prévia: há uma conta, e não só um número.
function temConta(expr) {
    return /[+\-×÷^%√]/.test(expr.replace(/^-/, ""));
}

// A linha de baixo do visor. tipo: "resultado" (depois do "="), "previa"
// (enquanto se digita), "aviso" (a prévia não sai: divisão por zero…), "erro"
// (o "=" foi recusado) ou "vazio". previa é a resposta do motor para a
// expressão de agora, ou null; erro é a frase do "=" recusado, ou "".
function visor(e, previa, erro) {
    if (erro)
        return { tipo: "erro", texto: String(erro), sinal: "" };
    if (e.fechado)
        return { tipo: "resultado", texto: e.resultado, sinal: e.aproximado ? "≈" : "=" };
    if (previa && previa.erro)
        return previa.incompleta ? { tipo: "vazio", texto: "", sinal: "" } : { tipo: "aviso", texto: String(previa.erro), sinal: "" };
    if (previa && previa.resultado !== undefined)
        return { tipo: "previa", texto: String(previa.resultado), sinal: previa.aproximado ? "≈" : "=" };
    return { tipo: "vazio", texto: "", sinal: "" };
}

// O que o Ctrl+C copia: o resultado cru; antes do "=", o da prévia; sem
// conta nenhuma, o número que está escrito. "" quando não há o que copiar.
function paraCopiar(e, previa) {
    if (e.fechado)
        return e.cru;
    if (previa && !previa.erro && previa.cru !== undefined)
        return String(previa.cru);
    return /^-?\d+(,\d*)?$/.test(e.expr) ? semVirgula(e.expr) : "";
}

// ── o teclado físico ───────────────────────────────────────────────────

// Um caractere digitado vira uma tecla da calculadora (ou "").
function deTexto(c) {
    if (typeof c !== "string" || c.length !== 1)
        return "";
    if (eAlgarismo(c) || "+-^%()".indexOf(c) >= 0)
        return c;
    switch (c) {
    case ",": case ".":
        return ",";
    case "*": case "x": case "X": case "×":
        return "×";
    case "/": case ":": case "÷":
        return "÷";
    case "=":
        return "igual";
    case "s": case "S": case "±":
        return "sinal";
    case "c": case "C":
        return "limpar";
    case "r": case "R": case "√":
        return "√";
    }
    return "";
}

// Um evento de teclado (a tecla, o texto e os modificadores do Qt) vira a
// lista do que fazer: teclas da calculadora (as de tecla()) e "igual",
// "esc", "copiar", "colar", "historico". Lista vazia: a tecla não é nossa.
function acoes(codigo, texto, mods) {
    mods = mods || 0;
    if (mods & (MOD.alt | MOD.meta))
        return [];
    if (mods & MOD.ctrl) {
        if (codigo === TECLA.c || codigo === TECLA.insert)
            return ["copiar"];
        return codigo === TECLA.v ? ["colar"] : codigo === TECLA.h ? ["historico"] : [];
    }
    if (codigo === TECLA.retorno || codigo === TECLA.enter)
        return ["igual"];
    if (codigo === TECLA.apagar)
        return ["apagar"];
    if (codigo === TECLA.esc)
        return ["esc"];
    if ((mods & MOD.numerico) && SEM_NUMLOCK[codigo] !== undefined)
        return [SEM_NUMLOCK[codigo]];
    if (codigo === TECLA.insert && (mods & MOD.shift))
        return ["colar"];
    if (codigo === TECLA.del)
        return ["limpar"];
    // Num teclado em que o ^ é tecla morta ele chega sem texto.
    if (codigo === TECLA.circunflexoMorto)
        return ["^"];
    const lista = [];
    for (const c of String(texto || "")) {
        const a = deTexto(c);
        if (a !== "")
            lista.push(a);
    }
    return lista;
}

// ── o motor ────────────────────────────────────────────────────────────

// Com MKCALC_EXEMPLO a janela não fala com o motor: cada pedido vira um jq
// numa resposta gravada (calcular.json leva da expressão à resposta, com "*"
// para as outras; historico.json, apagar.json e limpar.json são a resposta).
function comandoDeExemplo(pasta, pedido) {
    const id = String(Math.floor(Number(pedido.id)) || 0);
    if (pedido.op === "calcular")
        return ["jq", "-c", "--argjson", "id", id, "--arg", "e", String(pedido.expr), "(.[$e] // .[\"*\"]) + {id: $id}", pasta + "/calcular.json"];
    const arquivo = ["historico", "apagar", "limpar"].indexOf(pedido.op) >= 0 ? pedido.op : "historico";
    return ["jq", "-c", "--argjson", "id", id, ". + {id: $id}", pasta + "/" + arquivo + ".json"];
}
