// shell/mare/Updates.qml, o trecho entre "puro: início" e "puro: fim": quando
// a ilha acende, há quanto tempo foi a procura, o dia da última rodada do
// mk-update e a linha que resume como ela foi.
import { t, eq, igual, vazio, assert } from "./harness.mjs";
import { puro, fonte } from "./qml-puro.mjs";

// Cada função pedida ao bloco fica anotada: no fim, uma que nenhum caso
// chamou reprova.
const bloco = puro("Updates.qml");
const chamadas = new Set();
const M = new Proxy(bloco, { get: (alvo, nome) => { chamadas.add(nome); return alvo[nome]; } });

const FUNCOES = ["agoText", "badgeFor", "dayWord", "daysSince", "summary"];
const CONSTANTES = [];

// Toda função do bloco tem caso aqui; uma nova sem teste reprova.
t("o bloco puro tem estas funções e constantes, e mais nada", () => {
    igual(Object.keys(bloco).sort(), FUNCOES.concat(CONSTANTES).sort());
    igual(fonte("Updates.qml").funcoes.slice().sort(), FUNCOES.slice().sort());
});

const AGORA = 1795400000;
// A resposta do scripts/mare-updates, com o que o teste quiser trocar.
function resposta(mais) {
    return Object.assign({
        mk: { available: false, local: "0.9.2", remote: "0.9.2" }, system: [], aur: [], migrations: 0, error: "",
        checked: AGORA, last: null, applied: AGORA - 3600, reboot: false, news: []
    }, mais || {});
}
const pacotes = (n) => Array.from({ length: n }, (_, i) => ({ name: "pacote" + i, from: "1.0-1", to: "1.0-2" }));
const seg = (data) => data.getTime() / 1000;

t("badgeFor: sem resposta ainda, a ilha fica apagada", () => {
    eq(M.badgeFor(null), false);
    eq(M.badgeFor(undefined), false);
});

t("badgeFor: nada a atualizar não acende", () => {
    eq(M.badgeFor(resposta()), false);
    eq(M.badgeFor(resposta({ applied: 0 })), false);
});

t("badgeFor: versão nova do Mk ou migração pendente acende na hora", () => {
    eq(M.badgeFor(resposta({ mk: { available: true } })), true);
    eq(M.badgeFor(resposta({ migrations: 2 })), true);
    // Uma resposta antiga, sem o campo das migrações, não acende por isso.
    const semCampo = resposta();
    delete semCampo.migrations;
    eq(M.badgeFor(semCampo), false);
});

t("badgeFor: pacote esperando só acende depois de uma semana da última rodada", () => {
    const dia = 86400;
    eq(M.badgeFor(resposta({ system: pacotes(40), applied: AGORA - 2 * dia })), false);
    eq(M.badgeFor(resposta({ aur: pacotes(1), applied: AGORA - 7 * dia + 1 })), false);
    eq(M.badgeFor(resposta({ aur: pacotes(1), applied: AGORA - 7 * dia })), true);
    eq(M.badgeFor(resposta({ system: pacotes(3), aur: pacotes(2), applied: AGORA - 30 * dia })), true);
});

t("badgeFor: pacote esperando e o mk-update nunca rodou acende", () => {
    eq(M.badgeFor(resposta({ system: pacotes(1), applied: 0 })), true);
    const semCampo = resposta({ system: pacotes(1) });
    delete semCampo.applied;
    eq(M.badgeFor(semCampo), true);
});

t("agoText: agora há pouco, minutos e horas", () => {
    eq(M.agoText(AGORA, AGORA), "agora há pouco");
    eq(M.agoText(AGORA - 29, AGORA), "agora há pouco");
    eq(M.agoText(AGORA - 30, AGORA), "há 1 min");
    eq(M.agoText(AGORA - 12 * 60, AGORA), "há 12 min");
    eq(M.agoText(AGORA - 59 * 60, AGORA), "há 59 min");
    eq(M.agoText(AGORA - 3600, AGORA), "há 1 h");
    eq(M.agoText(AGORA - 3 * 3600, AGORA), "há 3 h");
    // O relógio do node vem com fração de segundo.
    eq(M.agoText(AGORA - 600, AGORA + 0.734), "há 10 min");
});

t("agoText: relógio que andou para trás não vira tempo negativo", () => {
    eq(M.agoText(AGORA + 500, AGORA), "agora há pouco");
});

t("daysSince: conta as viradas de meia-noite, não blocos de 24 horas", () => {
    const agora = new Date(2026, 10, 23, 0, 5);
    eq(M.daysSince(seg(new Date(2026, 10, 23, 0, 1)), agora), 0);
    eq(M.daysSince(seg(new Date(2026, 10, 22, 23, 59)), agora), 1);
    eq(M.daysSince(seg(new Date(2026, 10, 22, 0, 0)), agora), 1);
    eq(M.daysSince(seg(new Date(2026, 10, 18, 12, 0)), agora), 5);
    // Fim do dia: ainda é o mesmo dia.
    eq(M.daysSince(seg(new Date(2026, 10, 23, 0, 0)), new Date(2026, 10, 23, 23, 59)), 0);
});

t("daysSince: atravessa mês, ano e 29 de fevereiro", () => {
    eq(M.daysSince(seg(new Date(2026, 9, 31, 20, 0)), new Date(2026, 10, 1, 8, 0)), 1);
    eq(M.daysSince(seg(new Date(2026, 11, 31, 23, 0)), new Date(2027, 0, 1, 1, 0)), 1);
    eq(M.daysSince(seg(new Date(2024, 1, 28, 12, 0)), new Date(2024, 2, 1, 12, 0)), 2);
    eq(M.daysSince(seg(new Date(2025, 1, 28, 12, 0)), new Date(2025, 2, 1, 12, 0)), 1);
    eq(M.daysSince(seg(new Date(2025, 10, 23, 12, 0)), new Date(2026, 10, 23, 12, 0)), 365);
    // Uma data no futuro dá negativo (o dayWord trata como hoje).
    eq(M.daysSince(seg(new Date(2026, 10, 25, 12, 0)), new Date(2026, 10, 23, 12, 0)), -2);
});

t("dayWord: hoje, ontem, há N dias; de um mês em diante, vazio (vai a data)", () => {
    eq(M.dayWord(0), "hoje");
    eq(M.dayWord(-2), "hoje");
    eq(M.dayWord(1), "ontem");
    eq(M.dayWord(2), "há 2 dias");
    eq(M.dayWord(29), "há 29 dias");
    eq(M.dayWord(30), "");
    eq(M.dayWord(400), "");
});

// O resumo que o mk-update grava em ultima-atualizacao.json.
const BOM = { when: AGORA - 90000, ok: true, step: "", from: "0.9.1", to: "0.9.2", log: "/home/u/.local/state/mk/update.log", applied: AGORA - 90000 };

t("summary: rodada que deu certo e trocou a versão do Mk", () => {
    eq(M.summary(BOM, "ontem"), "última vez: ontem, tudo certo · mk 0.9.2");
});

t("summary: deu certo sem versão nova do Mk não fala em versão", () => {
    eq(M.summary(Object.assign({}, BOM, { to: "0.9.1" }), "hoje"), "última vez: hoje, tudo certo");
    eq(M.summary(Object.assign({}, BOM, { to: "" }), "hoje"), "última vez: hoje, tudo certo");
    eq(M.summary({ when: AGORA, ok: true }, "hoje"), "última vez: hoje, tudo certo");
});

t("summary: rodada que parou diz em que passo, e não fala em versão", () => {
    eq(M.summary(Object.assign({}, BOM, { ok: false, step: "pacotes do AUR" }), "há 3 dias"),
        "última vez: há 3 dias, parou em: pacotes do AUR");
    eq(M.summary(Object.assign({}, BOM, { ok: false, step: "" }), "23/09/2026"), "última vez: 23/09/2026, parou em: início");
    eq(M.summary({ when: AGORA, ok: false }, "hoje"), "última vez: hoje, parou em: início");
});

t("summary: o passo vai como veio, com aspas, dois-pontos, barra e acento", () => {
    const passo = "migração \"2026-11: ajustes\\novo\" <b>";
    eq(M.summary({ ok: false, step: passo }, "hoje"), "última vez: hoje, parou em: " + passo);
});

// --- por último ---

t("toda função do bloco foi chamada por algum caso deste arquivo", () => {
    vazio(FUNCOES.concat(CONSTANTES).filter((n) => !chamadas.has(n)), "funções sem caso");
});
