// shell/mare/CalendarPanel.qml, o trecho entre "puro: início" e "puro: fim":
// a grade do mês (42 células, a semana começando no domingo), a troca de mês
// e a linha do dia por extenso.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("CalendarPanel.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["dayLine", "monthCells", "shifted"];
const CONSTANTES = ["months", "weekdays"];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("CalendarPanel.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const LONGE = new Date(1999, 0, 1);
const dias = (cells) => cells.map((c) => c.day);

t("monthCells: novembro de 2026 começa no domingo e tem 30 dias", () => {
    const c = M.monthCells(2026, 10, LONGE);
    eq(c.length, 42);
    igual(dias(c).slice(0, 7), [1, 2, 3, 4, 5, 6, 7]);
    igual(c[0], { day: 1, inMonth: true, today: false, weekend: true });
    igual(c[29], { day: 30, inMonth: true, today: false, weekend: false });
    // O que sobra é dezembro, apagado.
    igual(dias(c).slice(30), [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
    assert.ok(c.slice(30).every((x) => !x.inMonth));
});

t("monthCells: fevereiro de 2024 (bissexto) começa na quinta e vai até 29", () => {
    const c = M.monthCells(2024, 1, LONGE);
    // Domingo a quarta são o fim de janeiro.
    igual(dias(c).slice(0, 5), [28, 29, 30, 31, 1]);
    igual(c.slice(0, 5).map((x) => x.inMonth), [false, false, false, false, true]);
    igual(c[32], { day: 29, inMonth: true, today: false, weekend: false });
    igual(c[33], { day: 1, inMonth: false, today: false, weekend: false });
    eq(c.filter((x) => x.inMonth).length, 29);
});

t("monthCells: fevereiro tem 28 dias em 2025 e em 2100, e 29 em 2000", () => {
    eq(M.monthCells(2025, 1, LONGE).filter((x) => x.inMonth).length, 28);
    eq(M.monthCells(2100, 1, LONGE).filter((x) => x.inMonth).length, 28);
    eq(M.monthCells(2000, 1, LONGE).filter((x) => x.inMonth).length, 29);
});

t("monthCells: janeiro de 2027 começa na sexta, depois do fim de dezembro de 2026", () => {
    const c = M.monthCells(2027, 0, LONGE);
    igual(dias(c).slice(0, 7), [27, 28, 29, 30, 31, 1, 2]);
    igual(c.slice(0, 7).map((x) => x.inMonth), [false, false, false, false, false, true, true]);
    // 5 + 31 = 36: sobram 6 dias de fevereiro.
    igual(dias(c).slice(36), [1, 2, 3, 4, 5, 6]);
});

t("monthCells: março depois de fevereiro bissexto mostra o 29 apagado", () => {
    // 1º de março de 2024 é sexta.
    const c = M.monthCells(2024, 2, LONGE);
    igual(dias(c).slice(0, 6), [25, 26, 27, 28, 29, 1]);
    eq(c[4].inMonth, false);
});

t("monthCells: só o dia de hoje, no mês e no ano de hoje, vem marcado", () => {
    const hoje = new Date(2026, 10, 23, 14, 30);
    const c = M.monthCells(2026, 10, hoje);
    igual(c.filter((x) => x.today).map((x) => x.day), [23]);
    // O mesmo dia noutro mês ou noutro ano não é hoje.
    eq(M.monthCells(2026, 9, hoje).filter((x) => x.today).length, 0);
    eq(M.monthCells(2025, 10, hoje).filter((x) => x.today).length, 0);
    // O 23 de outubro que aparece apagado numa grade vizinha também não: dezembro de 2026 começa na terça.
    const dez = M.monthCells(2026, 11, new Date(2026, 10, 30));
    igual(dias(dez).slice(0, 3), [29, 30, 1]);
    eq(dez.filter((x) => x.today).length, 0);
});

t("monthCells: a primeira e a última coluna são o fim de semana", () => {
    const c = M.monthCells(2026, 5, LONGE);
    c.forEach((x, i) => eq(x.weekend, i % 7 === 0 || i % 7 === 6, "célula " + i));
});

t("monthCells: em todo mês de 1990 a 2110 a grade bate com o calendário", () => {
    const ruins = [];
    for (let y = 1990; y <= 2110; y++) {
        for (let m = 0; m < 12; m++) {
            const c = M.monthCells(y, m, LONGE);
            const quantos = new Date(y, m + 1, 0).getDate();
            const dentro = c.filter((x) => x.inMonth);
            if (c.length !== 42 || dentro.length !== quantos)
                ruins.push(y + "/" + (m + 1) + ": " + dentro.length + " dias em " + c.length + " células");
            // Os dias do mês vão de 1 ao último, seguidos, e começam na primeira semana.
            const primeiro = c.findIndex((x) => x.inMonth);
            if (primeiro > 6 || dentro.some((x, i) => x.day !== i + 1))
                ruins.push(y + "/" + (m + 1) + ": dias fora de ordem");
            // Cada célula cai na coluna do seu dia da semana (domingo = 0).
            for (let i = 0; i < 42; i++) {
                const data = new Date(y, m, i - primeiro + 1);
                if (data.getDay() !== i % 7 || data.getDate() !== c[i].day)
                    ruins.push(y + "/" + (m + 1) + ": célula " + i + " mostra " + c[i].day);
            }
        }
    }
    vazio(ruins, "meses errados");
});

t("shifted: anda um mês e vira o ano nas pontas", () => {
    igual(M.shifted(2026, 10, 1), { year: 2026, month: 11 });
    igual(M.shifted(2026, 11, 1), { year: 2027, month: 0 });
    igual(M.shifted(2027, 0, -1), { year: 2026, month: 11 });
    igual(M.shifted(2026, 5, 0), { year: 2026, month: 5 });
    igual(M.shifted(2026, 5, 12), { year: 2027, month: 5 });
    igual(M.shifted(2026, 0, -13), { year: 2024, month: 11 });
});

t("shifted: doze passos para a frente e doze para trás voltam ao mesmo mês", () => {
    let p = { year: 2024, month: 1 };
    for (let i = 0; i < 12; i++)
        p = M.shifted(p.year, p.month, 1);
    igual(p, { year: 2025, month: 1 });
    for (let i = 0; i < 12; i++)
        p = M.shifted(p.year, p.month, -1);
    igual(p, { year: 2024, month: 1 });
});

t("dayLine: o dia por extenso, com os acentos", () => {
    eq(M.dayLine(new Date(2026, 10, 23)), "segunda, 23 de novembro");
    eq(M.dayLine(new Date(2026, 2, 3)), "terça, 3 de março");
    eq(M.dayLine(new Date(2024, 1, 29)), "quinta, 29 de fevereiro");
    eq(M.dayLine(new Date(2027, 0, 2)), "sábado, 2 de janeiro");
    eq(M.dayLine(new Date(2026, 11, 31, 23, 59)), "quinta, 31 de dezembro");
});

t("months e weekdays: doze meses e sete dias, começando em janeiro e no domingo", () => {
    eq(M.months.length, 12);
    eq(M.months[0], "janeiro");
    eq(M.months[11], "dezembro");
    eq(M.weekdays.length, 7);
    eq(M.weekdays[0], "domingo");
    eq(M.weekdays[6], "sábado");
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
