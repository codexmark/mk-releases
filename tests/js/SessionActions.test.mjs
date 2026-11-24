// shell/mare/SessionActions.js: as ações de sessão (bloquear, suspender, sair,
// reiniciar, desligar), uma lista só para o painel Sessão e para os Controles.
import { t, eq, igual } from "./harness.mjs";
import { load } from "./load.mjs";

const S = load("SessionActions.js");

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["actions"];
    const funcoes = Object.keys(S).filter((k) => typeof S[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("de fábrica (o Mk não dorme): bloquear, sair, reiniciar e desligar, sem suspender", () => {
    igual(S.actions("nada").map((a) => a.id), ["lock", "logout", "reboot", "poweroff"]);
});

t("com suspender escolhido em Ajustes › Energia, o botão aparece depois do bloquear", () => {
    igual(S.actions("suspender").map((a) => a.id), ["lock", "suspend", "logout", "reboot", "poweroff"]);
});

t("um modo de dormir que não existe (hibernar antigo) não ganha botão", () => {
    igual(S.actions("hibernar").map((a) => a.id), ["lock", "logout", "reboot", "poweroff"]);
    igual(S.actions(undefined).map((a) => a.id), ["lock", "logout", "reboot", "poweroff"]);
});

t("sair, reiniciar e desligar pedem um segundo clique; bloquear e suspender agem na hora", () => {
    const por = Object.fromEntries(S.actions("suspender").map((a) => [a.id, a.confirm]));
    igual(por, { lock: false, suspend: false, logout: true, reboot: true, poweroff: true });
});

t("cada ação tem rótulo em português, ícone e um comando como lista de argumentos", () => {
    for (const a of S.actions("suspender")) {
        eq(typeof a.label, "string");
        eq(a.label.length > 0 && a.icon.length > 0, true, a.id);
        eq(Array.isArray(a.cmd) && a.cmd.length > 0 && a.cmd.every((x) => typeof x === "string"), true, a.id);
    }
    igual(S.actions("suspender").find((a) => a.id === "suspend").cmd, ["systemctl", "suspend"]);
    igual(S.actions("nada").find((a) => a.id === "lock").cmd, ["loginctl", "lock-session"]);
});

t("cada chamada devolve uma lista nova: mexer nela não estraga a próxima", () => {
    const a = S.actions("nada");
    a.pop();
    a[0].cmd.push("lixo");
    eq(S.actions("nada").length, 4);
    igual(S.actions("nada")[0].cmd, ["loginctl", "lock-session"]);
});
