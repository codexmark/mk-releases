.pragma library

// Os Ajustes do Mk como dados. Cada página é uma lista de linhas; o painel
// desenha todas do mesmo jeito, e a busca procura em todas. Uma linha:
//   key     onde o valor fica em ~/.config/mk/ajustes.json
//   ctl     "switch" | "slider" | "choice"
//   def     o valor de fábrica (o mesmo que o Mk usa sem ajustes.json)
//   hypr    o campo do Hyprland que a linha muda (vai para ajustes.lua e
//           é aplicado na hora com hyprctl eval); sem hypr, é do Maré
//   kw      palavras que alguém digitaria na busca e que o texto não usa
// A ideia vem do Hub do Ryoku: a página é dado, não layout.

var groups = [
    { name: "APARÊNCIA", items: [
        { key: "ilha", name: "Ilha", num: "01" },
        { key: "fundo", name: "Fundo", num: "02" },
        { key: "movimento", name: "Movimento", num: "03" },
        { key: "notificacoes", name: "Notificações", num: "04" } ] },
    { name: "JANELAS", items: [
        { key: "janelas", name: "Janelas", num: "05" } ] },
    { name: "DISPOSITIVOS", items: [
        { key: "tela", name: "Tela", num: "06" },
        { key: "som", name: "Som", num: "07" },
        { key: "entrada", name: "Teclado e mouse", num: "08" },
        { key: "energia", name: "Energia", num: "09" } ] },
    { name: "SISTEMA", items: [
        { key: "atualizacoes", name: "Atualizações", num: "10" },
        { key: "atalhos", name: "Atalhos", num: "11" },
        { key: "sobre", name: "Sobre o Mk", num: "12" } ] }
];

var blurbs = {
    "ilha": "A barra no alto da tela: o que ela mostra e quanto do fundo deixa passar.",
    "fundo": "A folha de desenho atrás das janelas, e como o papel de parede pousa sobre ela.",
    "movimento": "A velocidade de tudo que se move: painéis, ilha, janelas e workspaces.",
    "notificacoes": "Os avisos que descem da ilha. Super + Shift + N liga e desliga o não perturbe.",
    "tela": "Tamanho das coisas na tela e a luz noturna, que esquenta as cores para descansar a vista.",
    "som": "Por onde o som sai e por onde o microfone entra. O volume fica na ilha.",
    "energia": "O que acontece quando você se afasta. Zero quer dizer nunca.",
    "janelas": "Espaços, bordas e cantos das janelas. Muda na hora, sem reiniciar.",
    "entrada": "Layout do teclado, repetição de teclas, mouse e touchpad.",
    "atualizacoes": "O Mk, os pacotes do Arch e os do AUR. Cada atualização cria um snapshot antes: se algo quebrar, dá para voltar pelo menu do boot.",
    "atalhos": "Os atalhos do Mk. Super sozinho abre a busca; Super com outra tecla faz o atalho.",
    "sobre": "Esta máquina e onde os ajustes ficam guardados."
};

var rows = [
    // Ilha
    { page: "ilha", group: "FORMA", key: "island.restOpacity", label: "Opacidade em repouso",
      desc: "Com um painel aberto, a ilha fica sempre sólida", ctl: "slider",
      min: 0.5, max: 1, step: 0.02, def: 0.82, unit: "%", kw: "transparência barra" },
    { page: "ilha", group: "CONTEÚDO", key: "island.showSystem", label: "Sistema",
      desc: "Temperatura, rede, processador e memória ao lado dos workspaces", ctl: "switch",
      def: true, kw: "cpu memória temperatura rede estatísticas" },
    { page: "ilha", group: "CONTEÚDO", key: "island.showWeather", label: "Clima",
      desc: "Temperatura lá fora, ao lado do relógio", ctl: "switch", def: true, kw: "tempo previsão" },

    // Fundo
    { page: "fundo", group: "DESENHO", key: "backdrop.letter", label: "Letra do workspace",
      desc: "O α grande no canto e o convite num workspace vazio", ctl: "switch", def: true,
      kw: "grego alfa convite" },
    { page: "fundo", group: "PAPEL DE PAREDE", key: "backdrop.dim", label: "Escurecer a imagem",
      desc: "Quanto o papel de parede escurece para o texto continuar legível", ctl: "slider",
      min: 0, max: 0.8, step: 0.05, def: 0.35, unit: "%", kw: "wallpaper escuro brilho" },
    { page: "fundo", group: "TEXTURA", key: "backdrop.grain", label: "Grão",
      desc: "O ruído fino que faz o preto nunca ser preto", ctl: "slider",
      min: 0, max: 0.16, step: 0.01, def: 0.06, unit: "%", kw: "ruído textura filme" },

    // Movimento
    { page: "movimento", group: "MARÉ", key: "motion.speed", label: "Velocidade",
      desc: "Painéis, ilha e trocas de workspace na barra", ctl: "choice", def: 1,
      options: [ { v: 0.7, label: "rápido" }, { v: 1, label: "normal" }, { v: 1.4, label: "calmo" }, { v: 0, label: "sem" } ],
      kw: "animação rapidez" },
    { page: "movimento", group: "JANELAS", key: "hypr.animations", label: "Animar janelas",
      desc: "Janelas e workspaces entrando e saindo", ctl: "switch", def: true,
      hypr: "animations.enabled", kw: "animação hyprland" },

    // Notificações
    { page: "notificacoes", group: "AVISOS", key: "notify.duration", label: "Tempo na tela",
      desc: "Quanto um aviso fica antes de ir para a lista (os urgentes ficam o dobro)", ctl: "slider",
      min: 2, max: 15, step: 1, def: 5, unit: "s", kw: "notificação toast duração" },

    // Tela
    { page: "tela", group: "TAMANHO", key: "hypr.scale", label: "Escala",
      desc: "Em telas pequenas de alta resolução, 1,25 ou 1,5 deixa tudo legível", ctl: "choice", def: 1,
      options: [ { v: 1, label: "100%" }, { v: 1.25, label: "125%" }, { v: 1.5, label: "150%" }, { v: 2, label: "200%" } ],
      hypr: "@monitor.scale", kw: "zoom tamanho hidpi dpi resolução monitor" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.temperature", label: "Calor",
      desc: "Quanto menor, mais quente (6500 K é a cor normal)", ctl: "slider",
      min: 2500, max: 6000, step: 100, def: 4000, unit: "K", kw: "temperatura cor azul hyprsunset" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.auto", label: "Ligar sozinha à noite",
      desc: "Entre as horas abaixo", ctl: "switch", def: false, kw: "automático horário" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.from", label: "Começa às",
      desc: "", ctl: "slider", min: 16, max: 23, step: 1, def: 19, unit: "h" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.to", label: "Termina às",
      desc: "", ctl: "slider", min: 4, max: 10, step: 1, def: 6, unit: "h" },

    // Energia
    { page: "energia", group: "AO SE AFASTAR", key: "idle.dim", label: "Escurecer a tela",
      desc: "Minutos sem mexer no teclado ou no mouse", ctl: "slider",
      min: 0, max: 30, step: 1, def: 4, unit: "min", kw: "inatividade ocioso brilho hypridle" },
    { page: "energia", group: "AO SE AFASTAR", key: "idle.lock", label: "Bloquear",
      desc: "Pede a senha para voltar", ctl: "slider",
      min: 0, max: 60, step: 1, def: 5, unit: "min", kw: "bloqueio senha hyprlock tela de bloqueio" },
    { page: "energia", group: "AO SE AFASTAR", key: "idle.off", label: "Desligar a tela",
      desc: "A tela apaga; qualquer tecla acende", ctl: "slider",
      min: 0, max: 60, step: 1, def: 6, unit: "min", kw: "dpms monitor apagar" },
    { page: "energia", group: "AO SE AFASTAR", key: "idle.suspend", label: "Suspender",
      desc: "O computador dorme e gasta quase nada", ctl: "slider",
      min: 0, max: 120, step: 5, def: 30, unit: "min", kw: "suspensão dormir sleep" },

    // Janelas
    { page: "janelas", group: "ESPAÇOS", key: "hypr.gapsIn", label: "Entre janelas",
      desc: "O espaço que separa uma janela da outra", ctl: "slider",
      min: 0, max: 24, step: 1, def: 6, unit: "px", hypr: "general.gaps_in", kw: "gaps espaçamento" },
    { page: "janelas", group: "ESPAÇOS", key: "hypr.gapsOut", label: "Até a borda da tela",
      desc: "O espaço entre as janelas e a borda", ctl: "slider",
      min: 0, max: 48, step: 1, def: 12, unit: "px", hypr: "general.gaps_out", kw: "gaps margem" },
    { page: "janelas", group: "FORMA", key: "hypr.border", label: "Borda",
      desc: "A linha fina em osso em volta de cada janela", ctl: "slider",
      min: 0, max: 4, step: 1, def: 1, unit: "px", hypr: "general.border_size", kw: "contorno linha" },
    { page: "janelas", group: "FORMA", key: "hypr.rounding", label: "Cantos",
      desc: "Zero é a folha: cantos retos", ctl: "slider",
      min: 0, max: 16, step: 1, def: 0, unit: "px", hypr: "decoration.rounding", kw: "arredondado raio" },
    { page: "janelas", group: "FORMA", key: "hypr.inactiveOpacity", label: "Janelas em segundo plano",
      desc: "Opacidade das janelas sem foco", ctl: "slider",
      min: 0.6, max: 1, step: 0.02, def: 1, unit: "%", hypr: "decoration.inactive_opacity",
      kw: "transparência inativas foco" },

    // Atualizações
    { page: "atualizacoes", group: "AVISOS", key: "updates.auto", label: "Procurar sozinho",
      desc: "A cada 3 horas, sem sudo e sem baixar nada", ctl: "switch", def: true,
      kw: "update upgrade pacman verificar automático" },
    { page: "atualizacoes", group: "AVISOS", key: "updates.island", label: "Avisar na ilha",
      desc: "Um ícone com o número de atualizações, só quando há alguma", ctl: "switch", def: true,
      kw: "update notificação barra" },

    // Teclado e mouse
    { page: "entrada", group: "TECLADO", key: "hypr.kbLayout", label: "Layout",
      desc: "Com dois layouts, Alt+Shift troca entre eles", ctl: "choice", def: "br",
      options: [ { v: "br", label: "ABNT2" }, { v: "us", label: "US" }, { v: "br,us", label: "ABNT2 + US" } ],
      hypr: "input.kb_layout", kw: "idioma teclado abnt" },
    { page: "entrada", group: "TECLADO", key: "hypr.repeatRate", label: "Repetição",
      desc: "Letras por segundo com a tecla segurada", ctl: "slider",
      min: 10, max: 60, step: 1, def: 25, unit: "/s", hypr: "input.repeat_rate", kw: "repetir velocidade" },
    { page: "entrada", group: "TECLADO", key: "hypr.repeatDelay", label: "Espera para repetir",
      desc: "Quanto segurar antes de começar a repetir", ctl: "slider",
      min: 150, max: 1000, step: 25, def: 600, unit: "ms", hypr: "input.repeat_delay", kw: "atraso delay" },
    { page: "entrada", group: "MOUSE", key: "hypr.sensitivity", label: "Sensibilidade",
      desc: "Velocidade do ponteiro", ctl: "slider",
      min: -1, max: 1, step: 0.1, def: 0, unit: "", hypr: "input.sensitivity", kw: "ponteiro cursor velocidade" },
    { page: "entrada", group: "TOUCHPAD", key: "hypr.naturalScroll", label: "Rolagem natural",
      desc: "O conteúdo segue os dedos, como no celular", ctl: "switch", def: true,
      hypr: "input.touchpad.natural_scroll", kw: "scroll inverter" },
    { page: "entrada", group: "TOUCHPAD", key: "hypr.tapToClick", label: "Tocar para clicar",
      desc: "Um toque leve conta como clique", ctl: "switch", def: true,
      hypr: "input.touchpad.tap_to_click", kw: "toque clique" }
];

// Os atalhos de config/hypr/mare/binds.lua, para ler.
var shortcuts = [
    { group: "MARÉ", items: [
        ["Super", "busca: apps, projetos e comandos"],
        ["Super + ,", "estes ajustes"],
        ["Alt + Tab", "trocar de janela"],
        ["Super + P", "projeto do workspace"],
        ["Super + C", "calendário"],
        ["Super + N", "notificações"],
        ["Super + Shift + N", "não perturbe"],
        ["Super + W", "papel de parede"],
        ["Super + Shift + V", "área de transferência"],
        ["Super + Esc", "sair, bloquear, desligar"] ] },
    { group: "APPS", items: [
        ["Super + Enter", "terminal no projeto"],
        ["Super + G", "lazygit"],
        ["Super + D", "lazydocker"],
        ["Super + E", "arquivos (yazi)"],
        ["Super + B", "navegador"],
        ["Super + L", "bloquear a tela"] ] },
    { group: "JANELAS", items: [
        ["Super + Q", "fechar"],
        ["Super + F", "tela cheia"],
        ["Super + V", "flutuar"],
        ["Super + setas", "mover o foco"],
        ["Super + Shift + setas", "mover a janela"],
        ["Super + Ctrl + setas", "redimensionar"],
        ["Super + arrastar", "mover / redimensionar com o mouse"] ] },
    { group: "WORKSPACES", items: [
        ["Super + 1…9", "ir para α β γ…"],
        ["Super + Shift + 1…9", "levar a janela"],
        ["Super + Tab", "workspace anterior"],
        ["Super + roda", "workspace ao lado"] ] },
    { group: "TELA", items: [
        ["Print", "recortar e copiar"],
        ["Shift + Print", "salvar a tela inteira"],
        ["Super + Shift + Enter", "terminal de emergência"],
        ["Super + Shift + Esc", "sair do Hyprland"] ] }
];

function defaults() {
    var out = {};
    for (var i = 0; i < rows.length; i++)
        out[rows[i].key] = rows[i].def;
    return out;
}

function pageRows(page) {
    return rows.filter(function (r) { return r.page === page; });
}

function pageName(key) {
    for (var g = 0; g < groups.length; g++)
        for (var i = 0; i < groups[g].items.length; i++)
            if (groups[g].items[i].key === key)
                return groups[g].items[i];
    return null;
}

function groupOf(key) {
    for (var g = 0; g < groups.length; g++)
        for (var i = 0; i < groups[g].items.length; i++)
            if (groups[g].items[i].key === key)
                return groups[g].name;
    return "";
}

// Busca sem acento e sem caixa; cada palavra precisa aparecer em algum lugar.
function fold(s) {
    return String(s).toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "");
}

function search(query) {
    var words = fold(query).split(/\s+/).filter(function (w) { return w.length > 0; });
    if (words.length === 0)
        return [];
    var out = [];
    var test = function (text) {
        var t = fold(text);
        return words.every(function (w) { return t.indexOf(w) >= 0; });
    };
    for (var g = 0; g < groups.length; g++)
        for (var i = 0; i < groups[g].items.length; i++) {
            var p = groups[g].items[i];
            if (test(p.name + " " + groups[g].name + " " + (blurbs[p.key] || "")))
                out.push({ page: p.key, key: "", label: p.name, where: "página" });
        }
    for (var r = 0; r < rows.length; r++) {
        var row = rows[r];
        var page = pageName(row.page);
        if (test(row.label + " " + row.desc + " " + row.group + " " + page.name + " " + (row.kw || "")))
            out.push({ page: row.page, key: row.key, label: row.label, where: page.name + " › " + row.group.toLowerCase() });
    }
    for (var s = 0; s < shortcuts.length; s++)
        for (var k = 0; k < shortcuts[s].items.length; k++) {
            var sc = shortcuts[s].items[k];
            if (test(sc[0] + " " + sc[1] + " atalho tecla"))
                out.push({ page: "atalhos", key: "", label: sc[1], where: "Atalhos › " + sc[0] });
        }
    return out.slice(0, 9);
}

// O ajustes.lua que o Hyprland carrega no login: só as linhas com hypr.
function luaValue(v) {
    if (typeof v === "boolean")
        return v ? "true" : "false";
    if (typeof v === "number")
        return String(Math.round(v * 1000) / 1000);
    return "\"" + String(v).replace(/\\/g, "\\\\").replace(/"/g, "\\\"") + "\"";
}

function hyprTable(values) {
    var tree = {};
    for (var i = 0; i < rows.length; i++) {
        var row = rows[i];
        if (!row.hypr || row.hypr[0] === "@")
            continue;
        var v = values[row.key] !== undefined ? values[row.key] : row.def;
        var parts = row.hypr.split(".");
        var node = tree;
        for (var p = 0; p < parts.length - 1; p++) {
            node[parts[p]] = node[parts[p]] || {};
            node = node[parts[p]];
        }
        node[parts[parts.length - 1]] = v;
    }
    var emit = function (obj) {
        var items = [];
        for (var k in obj)
            items.push(k + " = " + (typeof obj[k] === "object" ? emit(obj[k]) : luaValue(obj[k])));
        return "{ " + items.join(", ") + " }";
    };
    return emit(tree);
}

function valueOf(values, key) {
    for (var i = 0; i < rows.length; i++)
        if (rows[i].key === key)
            return values[key] !== undefined ? values[key] : rows[i].def;
    return undefined;
}

// O Lua que aplica tudo: hl.config e a escala dos monitores.
function hyprCode(values) {
    return "hl.config(" + hyprTable(values) + ")\n"
        + "hl.monitor({ output = \"\", mode = \"preferred\", position = \"auto\", scale = "
        + luaValue(valueOf(values, "hypr.scale")) + " })\n";
}

function hyprLua(values) {
    return "-- Gerado pelos Ajustes do Mk (Super + ,). Não edite: o painel reescreve.\n"
        + hyprCode(values);
}

// O hypridle.conf dos Ajustes (Energia). Minutos 0 = aquele passo não acontece.
function hypridleConf(values, hyprDir) {
    var out = "# Gerado pelos Ajustes do Mk (Super + ,), página Energia. Não edite.\n\n"
        + "general {\n"
        + "    lock_cmd = qs -p " + hyprDir.replace(/\/hypr$/, "/shell") + " ipc call mare lock || pidof hyprlock || hyprlock -c " + hyprDir + "/hyprlock.conf\n"
        + "    before_sleep_cmd = loginctl lock-session\n"
        + "    after_sleep_cmd = hyprctl dispatch 'hl.dsp.dpms({ action = \"on\" })'\n"
        + "}\n";
    var steps = [
        ["idle.dim", "brightnessctl -s set 15%", "brightnessctl -r"],
        ["idle.lock", "loginctl lock-session", ""],
        ["idle.off", "hyprctl dispatch 'hl.dsp.dpms({ action = \"off\" })'", "hyprctl dispatch 'hl.dsp.dpms({ action = \"on\" })'"],
        ["idle.suspend", "systemctl suspend", ""]
    ];
    for (var i = 0; i < steps.length; i++) {
        var min = valueOf(values, steps[i][0]);
        if (!min)
            continue;
        out += "\nlistener {\n    timeout = " + Math.round(min * 60) + "\n    on-timeout = " + steps[i][1] + "\n"
            + (steps[i][2] ? "    on-resume = " + steps[i][2] + "\n" : "") + "}\n";
    }
    return out;
}
