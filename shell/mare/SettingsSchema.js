.pragma library
.import "Keybinds.js" as Keybinds

// Os Ajustes do Mk como dados. Cada página é uma lista de linhas; o painel
// desenha todas do mesmo jeito, e a busca procura em todas. Uma linha:
//   key     onde o valor fica em ~/.config/mk/ajustes.json
//   ctl     "switch" | "slider" | "choice"
//   def     o valor de fábrica (o mesmo que o Mk usa sem ajustes.json)
//   hypr    o campo do Hyprland que a linha muda (vai para ajustes.lua e
//           é aplicado na hora com hyprctl eval); sem hypr, é do Maré
//           ("@..." = entra no Lua por um caminho próprio, veja hyprCode)
//   kw      palavras que alguém digitaria na busca e que o texto não usa
//   quando  { outra chave: valor ou [valores] }: a linha só aparece assim
//   precisa "touchpad" | "bateria" | "brilho": some se a
//           máquina não tem (scripts/mare-caps). Vale também numa opção de
//           um "choice" (veja optionShown)
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
        { key: "monitores", name: "Monitores", num: "07" },
        { key: "som", name: "Som", num: "08" },
        { key: "entrada", name: "Teclado e mouse", num: "09" },
        { key: "energia", name: "Energia", num: "10" },
        { key: "impressoras", name: "Impressoras", num: "11" },
        { key: "controle", name: "Controle", num: "12" } ] },
    { name: "REDE", items: [
        { key: "vpn", name: "VPN", num: "13" } ] },
    { name: "SISTEMA", items: [
        { key: "apps", name: "Apps padrão", num: "14" },
        { key: "atualizacoes", name: "Atualizações", num: "15" },
        { key: "atalhos", name: "Atalhos", num: "16" },
        { key: "sobre", name: "Sobre o Mk", num: "17" } ] }
];

var blurbs = {
    "ilha": "A barra no alto da tela: o que ela mostra e quanto do fundo deixa passar.",
    "fundo": "A folha de desenho atrás das janelas, e como o papel de parede pousa sobre ela.",
    "movimento": "A velocidade de tudo que se move: painéis, ilha, janelas e workspaces.",
    "notificacoes": "Os avisos que descem da ilha. Super + Shift + N liga e desliga o não perturbe.",
    "tela": "Tamanho das coisas na tela e a luz noturna, que esquenta as cores para descansar a vista.",
    "som": "Por onde o som sai e por onde o microfone entra. O volume fica na ilha.",
    "energia": "O que acontece quando você se afasta, e como o computador dorme. Zero quer dizer nunca.",
    "janelas": "Espaços, bordas e cantos das janelas. Muda na hora, sem reiniciar.",
    "entrada": "Layout do teclado, repetição de teclas, mouse e touchpad.",
    "atualizacoes": "O Mk, os pacotes do Arch e os do AUR. Cada atualização cria um snapshot antes: se algo quebrar, dá para voltar pelo menu do boot.",
    "monitores": "Cada tela ligada: resolução, taxa, escala, posição e rotação.",
    "impressoras": "As impressoras deste computador e as que a rede oferece.",
    "controle": "O controle Xbox (gamepad) e o fone (headset) plugado nele, pelo driver xone.",
    "vpn": "Conexões privadas (WireGuard e OpenVPN) pelo NetworkManager.",
    "apps": "Que app abre cada tipo de coisa: páginas, arquivos, texto, imagens, vídeos, música, PDF, e-mail. E o terminal do Mk.",
    "atalhos": "Os atalhos do Mk. Clique num para trocar a tecla; Super sozinho abre a busca.",
    "sobre": "Esta máquina e onde os ajustes ficam guardados."
};

var rows = [
    // Ilha
    { page: "ilha", group: "FORMA", key: "island.restOpacity", label: "Opacidade em repouso",
      desc: "Com um painel aberto, a ilha fica sempre sólida", ctl: "slider",
      min: 0.5, max: 1, step: 0.02, def: 0.82, unit: "%", kw: "transparência barra" },
    { page: "ilha", group: "FORMA", key: "island.sound", label: "Responder ao som",
      desc: "A borda de baixo da ilha ondula com o que está tocando", ctl: "switch", def: true,
      kw: "música áudio visualizador espectro cava onda" },
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
    { page: "movimento", group: "TUDO", key: "motion.speed", label: "Velocidade",
      desc: "Painéis, ilha, janelas e workspaces", ctl: "choice", def: 1,
      options: [ { v: 0.7, label: "rápido" }, { v: 1, label: "normal" }, { v: 1.4, label: "calmo" }, { v: 0, label: "sem" } ],
      hypr: "@animations", kw: "animação rapidez animar janelas hyprland" },

    // Notificações
    { page: "notificacoes", group: "AVISOS", key: "notify.duration", label: "Tempo na tela",
      desc: "Quanto um aviso fica antes de ir para a lista (os urgentes ficam o dobro)", ctl: "slider",
      min: 2, max: 15, step: 1, def: 5, unit: "s", kw: "notificação toast duração" },

    // Tela
    { page: "tela", group: "TAMANHO", key: "ui.scale", label: "Tamanho da interface",
      desc: "Ilha, painéis e ajustes. Ao mudar, o Maré reinicia", ctl: "choice", def: 0.85,
      options: [ { v: 0.75, label: "75%" }, { v: 0.85, label: "85%" }, { v: 1, label: "100%" }, { v: 1.15, label: "115%" } ],
      kw: "tamanho fonte letra grande pequeno zoom interface barra" },
    { page: "tela", group: "TAMANHO", key: "hypr.scale", label: "Escala da tela",
      desc: "Para telas sem ajuste próprio em Monitores, onde a escala é por tela", ctl: "choice", def: 1,
      options: [ { v: 1, label: "100%" }, { v: 1.25, label: "125%" }, { v: 1.5, label: "150%" }, { v: 2, label: "200%" } ],
      hypr: "@monitor.scale", kw: "zoom tamanho hidpi dpi resolução monitor" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.temperature", label: "Calor",
      desc: "Quanto menor, mais quente (6500 K é a cor normal)", ctl: "slider",
      min: 2500, max: 6000, step: 100, def: 4000, unit: "K", kw: "temperatura cor azul hyprsunset" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.auto", label: "Ligar sozinha à noite",
      desc: "Entre as horas abaixo", ctl: "switch", def: false, kw: "automático horário" },
    { page: "tela", group: "LUZ NOTURNA", key: "night.from", label: "Começa às",
      desc: "", ctl: "slider", min: 16, max: 23, step: 1, def: 19, unit: "h",
      quando: { "night.auto": true } },
    { page: "tela", group: "LUZ NOTURNA", key: "night.to", label: "Termina às",
      desc: "", ctl: "slider", min: 4, max: 10, step: 1, def: 6, unit: "h",
      quando: { "night.auto": true } },

    // Energia
    { page: "energia", group: "AO SE AFASTAR", key: "idle.dim", label: "Escurecer a tela",
      desc: "Minutos sem mexer no teclado ou no mouse", ctl: "slider",
      min: 0, max: 30, step: 1, def: 4, unit: "min", precisa: "brilho",
      kw: "inatividade ocioso brilho hypridle" },
    { page: "energia", group: "AO SE AFASTAR", key: "idle.lock", label: "Bloquear",
      desc: "Pede a senha para voltar", ctl: "slider",
      min: 0, max: 60, step: 1, def: 5, unit: "min", kw: "bloqueio senha hyprlock tela de bloqueio" },
    { page: "energia", group: "AO SE AFASTAR", key: "idle.off", label: "Desligar a tela",
      desc: "A tela apaga; qualquer tecla acende", ctl: "slider",
      min: 0, max: 60, step: 1, def: 6, unit: "min", kw: "dpms monitor apagar" },
    { page: "energia", group: "BATERIA", key: "power.lowWarn", label: "Avisar bateria fraca",
      desc: "Um aviso quando a carga está acabando", ctl: "switch", def: true, precisa: "bateria",
      kw: "carga pouca acabando notificação" },
    // Uma escolha só, usada em todo lugar: o passo de dormir abaixo, o painel
    // de sessão e a tela de bloqueio (veja sleepMode). O Mk não hiberna.
    { page: "energia", group: "DORMIR", key: "power.sleep", label: "Ao dormir",
      desc: "Suspender guarda na memória e volta na hora",
      ctl: "choice", def: "suspender",
      options: [ { v: "suspender", label: "suspender" }, { v: "nada", label: "não dormir" } ],
      kw: "suspender suspensão sleep suspend s3" },
    { page: "energia", group: "DORMIR", key: "idle.suspend", label: "Dormir",
      desc: "Minutos sem uso até o computador dormir sozinho", ctl: "slider",
      min: 0, max: 120, step: 5, def: 30, unit: "min", quando: { "power.sleep": ["suspender"] },
      kw: "suspender suspensão inatividade ocioso sleep" },

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
    { page: "entrada", group: "TECLADO", key: "input.caps", label: "Caps Lock",
      desc: "O que a tecla faz", ctl: "choice", def: "normal",
      options: [ { v: "normal", label: "normal" }, { v: "esc", label: "Esc" }, { v: "ctrl", label: "Ctrl" } ],
      hypr: "@kb_options", kw: "maiúsculas escape control vim" },
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
      hypr: "input.touchpad.natural_scroll", precisa: "touchpad", kw: "scroll inverter" },
    { page: "entrada", group: "TOUCHPAD", key: "hypr.tapToClick", label: "Tocar para clicar",
      desc: "Um toque leve conta como clique", ctl: "switch", def: true,
      hypr: "input.touchpad.tap_to_click", precisa: "touchpad", kw: "toque clique" }
];

// Chaves com página própria (sem linha desenhada pelo painel): só o valor de fábrica.
var hidden = {
    "apps.terminal": "ghostty",
    "emoji.recent": [],
    "emoji.tone": 0
};

function defaults() {
    var out = {};
    for (var i = 0; i < rows.length; i++)
        out[rows[i].key] = rows[i].def;
    for (var k in hidden)
        out[k] = hidden[k];
    return out;
}

// A linha aparece? `values` são os ajustes de agora e `caps` o que o
// scripts/mare-caps achou. Sem values não se olha `quando`; sem caps (ainda
// não respondeu), a máquina tem tudo.
function shown(row, values, caps) {
    if (row.precisa && caps && caps[row.precisa] === false)
        return false;
    if (row.quando && values)
        for (var k in row.quando) {
            var want = row.quando[k];
            var v = valueOf(values, k);
            if (Array.isArray(want) ? want.indexOf(v) < 0 : v !== want)
                return false;
        }
    return true;
}

// Uma opção de um "choice" vale nesta máquina? Com `precisa`, ela some como
// a linha some; a que está escolhida fica, para a pessoa ver o que vale (e
// trocar).
function optionShown(row, option, values, caps) {
    return !option.precisa || !caps || caps[option.precisa] !== false
        || (values !== undefined && values !== null && valueOf(values, row.key) === option.v);
}

function options(row, values, caps) {
    return (row.options || []).filter(function (o) { return optionShown(row, o, values, caps); });
}

// Sem values e caps: todas as linhas da página, as escondidas também.
function pageRows(page, values, caps) {
    return rows.filter(function (r) { return r.page === page && shown(r, values, caps); });
}

function rowOf(key) {
    for (var i = 0; i < rows.length; i++)
        if (rows[i].key === key)
            return rows[i];
    return null;
}

// Mudar esta chave muda o Lua do Hyprland?
function touchesHypr(key) {
    var r = rowOf(key);
    return r !== null && r.hypr !== undefined;
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

function search(query, binds, values, caps) {
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
        if (!shown(row, values, caps))
            continue;
        var page = pageName(row.page);
        if (test(row.label + " " + row.desc + " " + row.group + " " + page.name + " " + (row.kw || "")))
            out.push({ page: row.page, key: row.key, label: row.label, where: page.name + " › " + row.group.toLowerCase() });
    }
    // Os atalhos (Keybinds.js), com o combo que vale agora.
    var keybinds = Keybinds.entries(binds);
    for (var s = 0; s < keybinds.length; s++)
        if (test(keybinds[s].label + " " + keybinds[s].combo + " " + keybinds[s].group + " atalho tecla"))
            out.push({ page: "atalhos", key: "atalho:" + keybinds[s].id, label: keybinds[s].label, where: "Atalhos › " + keybinds[s].combo });
    return out.slice(0, 9);
}

// As opções do xkb, montadas dos Ajustes: Alt+Shift só existe com dois
// layouts, e o Caps Lock vira o que a pessoa escolheu.
var capsOptions = { "esc": "caps:escape", "ctrl": "ctrl:nocaps" };

function kbOptions(values) {
    var parts = [];
    if (String(valueOf(values, "hypr.kbLayout")).indexOf(",") >= 0)
        parts.push("grp:alt_shift_toggle");
    var caps = capsOptions[valueOf(values, "input.caps")];
    if (caps)
        parts.push(caps);
    return parts.join(",");
}

// As animações do config/hypr/mare/motion.lua (mesmas folhas, curvas e
// velocidades de fábrica; mudou lá, mude aqui). O speed do Hyprland é em
// 100 ms, então motion.speed multiplica: calmo = número maior.
var animations = [
    ["global", 2.4, "mare", ""],
    ["windows", 2.4, "mare", ""],
    ["windowsIn", 2.0, "mare", "popin 95%"],
    ["windowsOut", 1.6, "mareOut", "popin 94%"],
    ["border", 2.0, "mare", ""],
    ["fade", 1.7, "mare", ""],
    ["layers", 2.0, "mare", "fade"],
    ["workspaces", 2.8, "mare", "slide"],
    ["specialWorkspace", 2.4, "mare", "slidevert"]
];

// Uma destas chaves saiu da fábrica, ou foi mexida nesta sessão (`changed`,
// um mapa de chaves)? Só então o Lua fala do assunto: com tudo de fábrica,
// o kb_options e as animações dos arquivos da pessoa ficam em paz.
function ours(values, changed, keys) {
    for (var i = 0; i < keys.length; i++)
        if ((changed && changed[keys[i]]) || valueOf(values, keys[i]) !== rowOf(keys[i]).def)
            return true;
    return false;
}

// "sem" (0) não emite nada: animations.enabled = false já desliga tudo.
function animationCode(values, changed) {
    var speed = Number(valueOf(values, "motion.speed"));
    if (!(speed > 0) || !ours(values, changed, ["motion.speed"]))
        return "";
    var out = "";
    for (var i = 0; i < animations.length; i++) {
        var a = animations[i];
        out += "hl.animation({ leaf = " + luaValue(a[0]) + ", enabled = true, speed = "
            + luaValue(Math.round(a[1] * speed * 100) / 100) + ", bezier = " + luaValue(a[2])
            + (a[3] ? ", style = " + luaValue(a[3]) : "") + " })\n";
    }
    return out;
}

// O ajustes.lua que o Hyprland carrega no login: só as linhas com hypr.
function luaValue(v) {
    if (typeof v === "boolean")
        return v ? "true" : "false";
    if (typeof v === "number")
        return String(Math.round(v * 1000) / 1000);
    return "\"" + String(v).replace(/\\/g, "\\\\").replace(/"/g, "\\\"") + "\"";
}

function hyprTable(values, changed) {
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
    if (ours(values, changed, ["hypr.kbLayout", "input.caps"]))
        tree.input.kb_options = kbOptions(values);
    if (ours(values, changed, ["motion.speed"]))
        tree.animations = { enabled: valueOf(values, "motion.speed") > 0 };
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

// O Lua que aplica tudo: hl.config, as animações e a escala dos monitores.
// `changed` (as chaves mexidas nesta sessão) só entra no eval; o ajustes.lua
// é gerado sem ele.
function hyprCode(values, changed) {
    return "hl.config(" + hyprTable(values, changed) + ")\n"
        + animationCode(values, changed)
        + "hl.monitor({ output = \"\", mode = \"preferred\", position = \"auto\", scale = "
        + luaValue(valueOf(values, "hypr.scale")) + " })\n";
}

function hyprLua(values) {
    return "-- Gerado pelos Ajustes do Mk (Super + ,). Não edite: o painel reescreve.\n"
        + hyprCode(values);
}

// Como a máquina dorme de fato: a escolha de power.sleep, se a máquina puder.
// Suspender com o suspend proibido (sleep.conf) vira "nada". Um "hibernar"
// que sobrou de quando o Mk hibernava também: não vira suspender sozinho.
// Sem caps (o mare-caps ainda não respondeu), vale a escolha.
function sleepMode(values, caps) {
    var mode = valueOf(values, "power.sleep");
    if (mode === "suspender")
        return caps && caps.sono && caps.sono.canSuspend === false ? "nada" : "suspender";
    return "nada";
}

// O comando de cada jeito de dormir; "" = não dorme.
var sleepCommands = { "suspender": "systemctl suspend" };

function sleepCommand(values, caps) {
    return sleepCommands[sleepMode(values, caps)] || "";
}

// O passo de dormir que um hypridle.conf tem ("" = nenhum). O hibernate é de
// um arquivo antigo: achá-lo faz o syncIdle reescrever sem ele.
function sleepStep(text) {
    var m = /on-timeout = (systemctl (?:suspend|hibernate))\b/.exec(text);
    return m ? m[1] : "";
}

// O hypridle.conf dos Ajustes (Energia). Minutos 0 = aquele passo não acontece.
// Sem brilho para mexer (caps), o passo de escurecer também não; o de dormir
// segue power.sleep.
function hypridleConf(values, hyprDir, caps) {
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
        ["idle.suspend", sleepCommand(values, caps), ""]
    ];
    for (var i = 0; i < steps.length; i++) {
        var min = valueOf(values, steps[i][0]);
        if (!min || !steps[i][1] || (steps[i][0] === "idle.dim" && caps && caps.brilho === false))
            continue;
        out += "\nlistener {\n    timeout = " + Math.round(min * 60) + "\n    on-timeout = " + steps[i][1] + "\n"
            + (steps[i][2] ? "    on-resume = " + steps[i][2] + "\n" : "") + "}\n";
    }
    return out;
}
