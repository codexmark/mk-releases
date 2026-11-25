.pragma library

// Os atalhos do Mk como dados, para a página Atalhos dos Ajustes e para a
// busca. É o espelho de config/hypr/mare/binds.lua (lá cada linha tem também o
// `run`, o que o atalho faz): mesmos ids, na mesma ordem, com o mesmo grupo,
// descrição, combo de fábrica e marcas. Ao mudar um, mude o outro, e confira
// (na raiz do repositório):
//
//   node -e 'const fs=require("fs"),f=(a)=>a.join("|"),T=fs.readFileSync("config/hypr/mare/binds.lua","utf8");const L=[...T.matchAll(/^\s*\{ id = "([^"]+)", group = "([^"]+)", desc = "([^"]+)", combo = "([^"]+)"(.*)$/gm)].map(m=>f([m[1],m[2],m[3],m[4],/required = true/.test(m[5]),/fixed = true/.test(m[5]),(m[5].match(/family = "(\w+)"/)||[])[1]||""]));const K={};new Function("K",fs.readFileSync("shell/mare/Keybinds.js","utf8").replace(/^\.pragma.*$/m,"")+";K.a=actions;K.f=families;K.e=everywhere")(K);const J=K.a.map(a=>f([a.id,a.group,a.desc,a.combo,!!a.required,!!a.fixed,a.family||""]));const d=L.filter(x=>!J.includes(x)).map(x=>"só no lua: "+x).concat(J.filter(x=>!L.includes(x)).map(x=>"só no js:  "+x));const n=(T.match(/\bid = "/g)||[]).length;if(n!==L.length)d.push("binds.lua: "+n+" ações, mas só "+L.length+" no formato de uma linha (id, group, desc, combo)");const F={};for(const m of T.matchAll(/^    (\w+) = \{ (\{ ".*\}) \},$/gm))F[m[1]]=[...m[2].matchAll(/\{ "([^"]+)"/g)].map(x=>x[1]);for(const m of T.matchAll(/^for i = (\d+), (\d+) do\n\s*families\.(\w+)\[i\] = \{ tostring\(i\), i \}/gm)){F[m[3]]=[];for(let i=+m[1];i<=+m[2];i++)F[m[3]].push(String(i))}for(const k of new Set(Object.keys(F).concat(Object.keys(K.f))))if(f(F[k]||[])!==f(K.f[k]||[]))d.push("família "+k+": lua ["+(F[k]||[])+"], js ["+(K.f[k]||[])+"]");const E=((T.match(/^local everywhere = \{(.*)\}$/m)||[])[1]||"").split(",").map(x=>x.replace(/[\[\]" ]|= true/g,"")).sort().join();if(E!==K.e.slice().sort().join())d.push("teclas de emergência: lua ["+E+"], js ["+K.e.slice().sort()+"]");if(!d.length&&L.join()!==J.join())d.push("ordem diferente");console.log(d.length?d.join("\n"):"atalhos: "+L.length+" ações e "+Object.keys(F).length+" famílias, lua e js iguais");process.exit(d.length?1:0)'
//
// Marcas: required = não pode ficar sem atalho (emergência); fixed = não se
// troca aqui (mouse, teclas de mídia, toque no Super, Alt+Tab); family = a
// ação vale para várias teclas e o combo guarda só os modificadores.
// O comando confere também as teclas de cada família e as teclas que valem
// para a emergência (`everywhere`, aqui e no binds.lua).
// O que o usuário troca fica em ajustes.json, na chave "keybinds", como
// { id: "COMBO" } ("" = sem atalho), e vai para <raiz>/atalhos.lua.

var actions = [
    { id: "emergency.terminal", group: "EMERGÊNCIA", desc: "terminal de emergência", combo: "SUPER + SHIFT + RETURN", required: true },
    { id: "emergency.exit", group: "EMERGÊNCIA", desc: "sair do Hyprland", combo: "SUPER + SHIFT + ESCAPE", required: true },

    { id: "mare.launcher.tap", group: "MARÉ", desc: "busca: apps, projetos e comandos (toque no Super)", combo: "SUPER", family: "super", fixed: true },
    { id: "mare.launcher", group: "MARÉ", desc: "busca: apps, projetos e comandos", combo: "SUPER + SPACE" },
    { id: "mare.switcher.next", group: "MARÉ", desc: "trocar de janela", combo: "ALT + TAB", fixed: true },
    { id: "mare.switcher.prev", group: "MARÉ", desc: "trocar de janela, para trás", combo: "ALT + SHIFT + TAB", fixed: true },
    { id: "mare.project", group: "MARÉ", desc: "projeto do workspace", combo: "SUPER + P" },
    { id: "mare.calendar", group: "MARÉ", desc: "calendário", combo: "SUPER + C" },
    { id: "mare.notifications", group: "MARÉ", desc: "notificações", combo: "SUPER + N" },
    { id: "mare.dnd", group: "MARÉ", desc: "não perturbe", combo: "SUPER + SHIFT + N" },
    { id: "mare.present", group: "MARÉ", desc: "modo apresentação", combo: "SUPER + SHIFT + P" },
    { id: "mare.game", group: "MARÉ", desc: "modo jogo", combo: "SUPER + SHIFT + G" },
    { id: "mare.controls", group: "MARÉ", desc: "controles rápidos", combo: "SUPER + ESCAPE" },
    { id: "mare.session", group: "MARÉ", desc: "sair, bloquear, desligar", combo: "SUPER + CTRL + ESCAPE" },
    { id: "mare.wallpaper", group: "MARÉ", desc: "papel de parede", combo: "SUPER + W" },
    { id: "mare.clipboard", group: "MARÉ", desc: "área de transferência", combo: "SUPER + SHIFT + V" },
    { id: "mare.emoji", group: "MARÉ", desc: "emoji", combo: "SUPER + PERIOD" },
    { id: "mare.settings", group: "MARÉ", desc: "estes ajustes", combo: "SUPER + COMMA" },
    { id: "mare.lock", group: "MARÉ", desc: "bloquear a tela", combo: "SUPER + L" },

    { id: "app.terminal", group: "APPS", desc: "terminal no projeto", combo: "SUPER + RETURN" },
    { id: "app.lazygit", group: "APPS", desc: "lazygit", combo: "SUPER + G" },
    { id: "app.lazydocker", group: "APPS", desc: "lazydocker", combo: "SUPER + D" },
    { id: "app.files", group: "APPS", desc: "arquivos (yazi)", combo: "SUPER + E" },
    { id: "app.browser", group: "APPS", desc: "navegador", combo: "SUPER + B" },

    { id: "win.close", group: "JANELAS", desc: "fechar", combo: "SUPER + Q" },
    { id: "win.fullscreen", group: "JANELAS", desc: "tela cheia", combo: "SUPER + F" },
    { id: "win.float", group: "JANELAS", desc: "flutuar", combo: "SUPER + V" },
    { id: "win.focus", group: "JANELAS", desc: "mover o foco", combo: "SUPER", family: "arrows" },
    { id: "win.move", group: "JANELAS", desc: "mover a janela", combo: "SUPER + SHIFT", family: "arrows" },
    { id: "win.resize", group: "JANELAS", desc: "redimensionar", combo: "SUPER + CTRL", family: "arrows" },
    { id: "win.drag", group: "JANELAS", desc: "mover com o mouse (arrastar)", combo: "SUPER + mouse:272", fixed: true },
    { id: "win.dragsize", group: "JANELAS", desc: "redimensionar com o mouse (botão direito)", combo: "SUPER + mouse:273", fixed: true },

    { id: "ws.focus", group: "WORKSPACES", desc: "ir para α β γ…", combo: "SUPER", family: "digits" },
    { id: "ws.move", group: "WORKSPACES", desc: "levar a janela", combo: "SUPER + SHIFT", family: "digits" },
    { id: "ws.previous", group: "WORKSPACES", desc: "workspace anterior", combo: "SUPER + TAB" },
    { id: "ws.next.wheel", group: "WORKSPACES", desc: "workspace seguinte (roda do mouse)", combo: "SUPER + mouse_down", fixed: true },
    { id: "ws.prev.wheel", group: "WORKSPACES", desc: "workspace anterior (roda do mouse)", combo: "SUPER + mouse_up", fixed: true },

    { id: "shot.area", group: "TELA", desc: "recortar e copiar", combo: "PRINT" },
    { id: "shot.screen", group: "TELA", desc: "salvar a tela inteira", combo: "SHIFT + PRINT" },
    { id: "shot.panel", group: "TELA", desc: "painel de captura (área, janela, vídeo)", combo: "SUPER + PRINT" },
    { id: "shot.read", group: "TELA", desc: "ler o texto ou o QR de uma área", combo: "SUPER + SHIFT + T" },
    { id: "shot.record", group: "TELA", desc: "gravar a tela / parar", combo: "SUPER + SHIFT + R" },

    { id: "media.volume.up", group: "TECLAS DE MÍDIA", desc: "aumentar o volume", combo: "XF86AudioRaiseVolume", fixed: true },
    { id: "media.volume.down", group: "TECLAS DE MÍDIA", desc: "diminuir o volume", combo: "XF86AudioLowerVolume", fixed: true },
    { id: "media.volume.mute", group: "TECLAS DE MÍDIA", desc: "silenciar", combo: "XF86AudioMute", fixed: true },
    { id: "media.mic.mute", group: "TECLAS DE MÍDIA", desc: "silenciar o microfone", combo: "XF86AudioMicMute", fixed: true },
    { id: "media.brightness.up", group: "TECLAS DE MÍDIA", desc: "aumentar o brilho", combo: "XF86MonBrightnessUp", fixed: true },
    { id: "media.brightness.down", group: "TECLAS DE MÍDIA", desc: "diminuir o brilho", combo: "XF86MonBrightnessDown", fixed: true },
    { id: "media.play", group: "TECLAS DE MÍDIA", desc: "tocar / pausar", combo: "XF86AudioPlay", fixed: true },
    { id: "media.next", group: "TECLAS DE MÍDIA", desc: "próxima faixa", combo: "XF86AudioNext", fixed: true },
    { id: "media.prev", group: "TECLAS DE MÍDIA", desc: "faixa anterior", combo: "XF86AudioPrev", fixed: true }
];

// As teclas de cada família (as mesmas de `families` no binds.lua) e como a
// família aparece no lugar da tecla.
var families = {
    "super": ["SUPER_L", "SUPER_R"],
    "arrows": ["Left", "Right", "Up", "Down"],
    "digits": ["1", "2", "3", "4", "5", "6", "7", "8", "9"]
};
var familyLabels = { "super": "(toque)", "arrows": "setas", "digits": "1…9" };

// Modificadores, na ordem em que entram no combo.
var mods = ["SUPER", "CTRL", "ALT", "SHIFT"];
var modLabels = { "SUPER": "Super", "CTRL": "Ctrl", "ALT": "Alt", "SHIFT": "Shift" };

// Nomes de tecla que o Hyprland aceita (os do xkb; ele não liga para a
// caixa). Fora desta lista vale ainda "code:<n>", o número físico da tecla.
var keys = [
    "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
    "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
    "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
    "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12",
    "RETURN", "ESCAPE", "SPACE", "TAB", "BACKSPACE", "DELETE", "INSERT",
    "HOME", "END", "PRIOR", "NEXT", "Left", "Right", "Up", "Down",
    "PRINT", "PAUSE", "MENU", "Scroll_Lock",
    "COMMA", "PERIOD", "SLASH", "SEMICOLON", "APOSTROPHE", "MINUS", "EQUAL",
    "BRACKETLEFT", "BRACKETRIGHT", "BACKSLASH", "GRAVE", "CCEDILLA",
    "dead_tilde", "dead_acute",
    "KP_0", "KP_1", "KP_2", "KP_3", "KP_4", "KP_5", "KP_6", "KP_7", "KP_8", "KP_9",
    "KP_Enter", "KP_Add", "KP_Subtract", "KP_Multiply", "KP_Divide", "KP_Decimal",
    "XF86AudioRaiseVolume", "XF86AudioLowerVolume", "XF86AudioMute", "XF86AudioMicMute",
    "XF86AudioPlay", "XF86AudioPause", "XF86AudioStop", "XF86AudioNext", "XF86AudioPrev",
    "XF86MonBrightnessUp", "XF86MonBrightnessDown", "XF86KbdBrightnessUp", "XF86KbdBrightnessDown",
    "XF86Calculator", "XF86Mail", "XF86Search", "XF86HomePage", "XF86Explorer",
    "XF86Tools", "XF86Favorites", "XF86Display", "XF86WLAN", "XF86Bluetooth",
    "XF86TouchpadToggle", "XF86Launch1", "XF86Launch2"
];

// Teclas que todo teclado tem, além das letras, dos números e de F1 a F12: as
// únicas para onde as ações de emergência podem ir (a mesma lista do binds.lua).
var everywhere = ["return", "escape", "space", "tab", "backspace", "delete"];

// As teclas que ficam no mesmo lugar em qualquer layout: número físico -> nome.
// "code:36" e "RETURN" são a mesma tecla, e escritas de um jeito só o conflito
// aparece. Letras e pontuação mudam de lugar com o layout e ficam de fora.
var codes = {
    9: "ESCAPE", 10: "1", 11: "2", 12: "3", 13: "4", 14: "5", 15: "6", 16: "7", 17: "8", 18: "9", 19: "0",
    22: "BACKSPACE", 23: "TAB", 36: "RETURN", 65: "SPACE",
    67: "F1", 68: "F2", 69: "F3", 70: "F4", 71: "F5", 72: "F6", 73: "F7", 74: "F8", 75: "F9", 76: "F10",
    78: "Scroll_Lock", 95: "F11", 96: "F12", 104: "KP_Enter", 107: "PRINT",
    110: "HOME", 111: "Up", 112: "PRIOR", 113: "Left", 114: "Right", 115: "END", 116: "Down", 117: "NEXT",
    118: "INSERT", 119: "DELETE", 127: "PAUSE", 135: "MENU"
};

// Pontuação com Shift: o Qt entrega o símbolo de cima ("<"), e o Hyprland quer
// o nome da tecla (COMMA). Símbolo@número físico -> nome, para os layouts
// americano e ABNT2; fora deles a tecla vira "code:<n>".
var shifted = {
    "<@59": "COMMA", ">@60": "PERIOD", "_@20": "MINUS", "+@21": "EQUAL",
    ":@47": "SEMICOLON", "?@61": "SLASH", "\"@48": "APOSTROPHE", "{@34": "BRACKETLEFT",
    "}@35": "BRACKETRIGHT", "|@51": "BACKSLASH", "~@49": "GRAVE",
    ":@61": "SEMICOLON", "?@97": "SLASH", "\"@49": "APOSTROPHE", "{@35": "BRACKETLEFT",
    "}@51": "BRACKETRIGHT", "|@94": "BACKSLASH"
};

// O que alguém digitaria em vez do nome do xkb.
var aliases = {
    "enter": "RETURN", "esc": "ESCAPE", "espaço": "SPACE", "espaco": "SPACE", "barra de espaço": "SPACE",
    "del": "DELETE", "ins": "INSERT", "pgup": "PRIOR", "pageup": "PRIOR", "page_up": "PRIOR",
    "pgdn": "NEXT", "pgdown": "NEXT", "pagedown": "NEXT", "page_down": "NEXT",
    "esquerda": "Left", "direita": "Right", "cima": "Up", "baixo": "Down",
    "←": "Left", "→": "Right", "↑": "Up", "↓": "Down",
    "printscreen": "PRINT", "prtsc": "PRINT", "prtscr": "PRINT", "print screen": "PRINT",
    ",": "COMMA", ".": "PERIOD", "/": "SLASH", ";": "SEMICOLON", "'": "APOSTROPHE",
    "-": "MINUS", "=": "EQUAL", "[": "BRACKETLEFT", "]": "BRACKETRIGHT", "\\": "BACKSLASH",
    "`": "GRAVE", "ç": "CCEDILLA", "~": "dead_tilde", "´": "dead_acute", "backtab": "TAB"
};

// Como cada tecla aparece na página.
var keyLabels = {
    "return": "Enter", "escape": "Esc", "space": "Espaço", "tab": "Tab", "backspace": "Backspace",
    "delete": "Delete", "insert": "Insert", "home": "Home", "end": "End", "prior": "PgUp", "next": "PgDn",
    "left": "←", "right": "→", "up": "↑", "down": "↓", "print": "Print", "pause": "Pause", "menu": "Menu",
    "scroll_lock": "Scroll Lock", "comma": ",", "period": ".", "slash": "/", "semicolon": ";",
    "apostrophe": "'", "minus": "-", "equal": "=", "bracketleft": "[", "bracketright": "]",
    "backslash": "\\", "grave": "`", "ccedilla": "Ç", "dead_tilde": "~", "dead_acute": "´",
    "mouse:272": "arrastar", "mouse:273": "arrastar (botão direito)",
    "mouse_down": "roda para baixo", "mouse_up": "roda para cima"
};

var keyIndex = null;

// Nome digitado ou capturado -> o nome que vai para o Hyprland; "" se não existe.
function normalizeKey(text) {
    if (keyIndex === null) {
        keyIndex = {};
        for (var i = 0; i < keys.length; i++)
            keyIndex[keys[i].toLowerCase()] = keys[i];
    }
    var t = String(text === undefined || text === null ? "" : text).trim().toLowerCase();
    if (t === "")
        return "";
    // Só o que está escrito nas tabelas: "constructor" e "__proto__" existem em
    // todo objeto e não são teclas.
    var own = Object.prototype.hasOwnProperty;
    if (own.call(aliases, t))
        return aliases[t];
    if (own.call(keyIndex, t))
        return keyIndex[t];
    if (/^code:[0-9]{1,3}$/.test(t)) {
        var n = parseInt(t.slice(5), 10);
        return codes[n] !== undefined ? codes[n] : "code:" + n;
    }
    return "";
}

function byId(id) {
    for (var i = 0; i < actions.length; i++)
        if (actions[i].id === id)
            return actions[i];
    return null;
}

// "SUPER + SHIFT + K" -> { mods: ["SUPER", "SHIFT"], key: "K" }. Numa família
// o combo é só de modificadores e a tecla fica vazia.
function parse(combo) {
    var out = { mods: [], key: "" };
    var parts = String(combo || "").split("+");
    for (var i = 0; i < parts.length; i++) {
        var p = parts[i].trim();
        if (p === "")
            continue;
        if (mods.indexOf(p.toUpperCase()) >= 0)
            out.mods.push(p.toUpperCase());
        else
            out.key = p;
    }
    out.mods = mods.filter(function (m) { return out.mods.indexOf(m) >= 0; });
    return out;
}

function format(modList, key) {
    var parts = mods.filter(function (m) { return modList.indexOf(m) >= 0; });
    if (key)
        parts.push(key);
    return parts.join(" + ");
}

function same(a, b) {
    var pa = parse(a), pb = parse(b);
    return format(pa.mods, pa.key).toLowerCase() === format(pb.mods, pb.key).toLowerCase();
}

// O combo escrito do nosso jeito (modificadores em ordem, tecla pelo nome), ou
// null se não serve para a ação: tecla que não existe, família com tecla, etc.
function canonical(action, combo) {
    if (combo === "")
        return "";
    var p = parse(combo);
    if (action.family && p.key !== "")
        return null;
    var r = validate(action, p.mods, p.key);
    return r.error === "" ? r.combo : null;
}

// Só o que é troca de verdade: ids que existem e podem mudar, com um combo
// válido e diferente do de fábrica. O que veio torto (ajustes.json ou
// atalhos.lua mexidos à mão) cai fora em vez de ir para o Hyprland. Uma de
// emergência também não fica com o combo de fábrica de quem não foi mexido
// (o apply() do binds.lua faz o mesmo).
function clean(overrides) {
    var out = {};
    if (!overrides || typeof overrides !== "object")
        return out;
    for (var i = 0; i < actions.length; i++) {
        var a = actions[i];
        var v = overrides[a.id];
        if (a.fixed || typeof v !== "string")
            continue;
        v = canonical(a, v.trim());
        if (v === null || (v === "" ? (a.required || a.family) : same(v, a.combo)))
            continue;
        out[a.id] = v;
    }
    for (var again = true; again;) {
        again = false;
        var home = {};
        for (var h = 0; h < actions.length; h++)
            if (out[actions[h].id] === undefined)
                expand(actions[h], actions[h].combo).forEach(function (k) { home[k] = true; });
        for (var r = 0; r < actions.length && !again; r++) {
            var e = actions[r];
            if (e.required && out[e.id] !== undefined && home[expand(e, out[e.id])[0]] === true) {
                delete out[e.id];
                again = true;
            }
        }
    }
    return out;
}

// O combo que vale para cada ação, como o apply() do binds.lua decide.
function effective(overrides) {
    var o = clean(overrides);
    var out = {};
    for (var i = 0; i < actions.length; i++)
        out[actions[i].id] = o[actions[i].id] !== undefined ? o[actions[i].id] : actions[i].combo;
    return out;
}

// As combinações que uma ação ocupa, em minúsculas, para comparar.
function expand(action, combo) {
    if (combo === "")
        return [];
    var p = parse(combo);
    if (!action.family)
        return [format(p.mods, p.key).toLowerCase()];
    return families[action.family].map(function (k) { return format(p.mods, k).toLowerCase(); });
}

// Todos os que já usam alguma combinação que `id` passaria a ocupar com
// `combo` (uma família ocupa várias e pode bater em mais de um); [] se ninguém.
function conflicts(id, combo, overrides) {
    var out = [];
    var me = byId(id);
    if (!me || combo === "")
        return out;
    var mine = expand(me, combo);
    var all = effective(overrides);
    for (var i = 0; i < actions.length; i++) {
        var other = actions[i];
        if (other.id === id)
            continue;
        if (expand(other, all[other.id]).some(function (k) { return mine.indexOf(k) >= 0; }))
            out.push(other);
    }
    return out;
}

// O primeiro deles; null se ninguém.
function conflict(id, combo, overrides) {
    var all = conflicts(id, combo, overrides);
    return all.length > 0 ? all[0] : null;
}

// Confere o que está no editor. Devolve { combo, error }: o combo pronto, ou
// o motivo de não dar.
function validate(action, modList, keyText) {
    var strong = modList.indexOf("SUPER") >= 0 || modList.indexOf("CTRL") >= 0 || modList.indexOf("ALT") >= 0;
    if (action.family) {
        if (!strong)
            return { combo: "", error: "escolha Super, Ctrl ou Alt" };
        return { combo: format(modList, ""), error: "" };
    }
    if (String(keyText || "").trim() === "")
        return { combo: "", error: "falta a tecla" };
    var key = normalizeKey(keyText);
    if (key === "")
        return { combo: "", error: "tecla desconhecida: " + String(keyText).trim() };
    // Emergência: só onde qualquer teclado alcança.
    if (action.required && (!strong || !(/^([a-z0-9]|f([1-9]|1[0-2]))$/i.test(key) || everywhere.indexOf(key.toLowerCase()) >= 0)))
        return { combo: "", error: "emergência: Super, Ctrl ou Alt com " + reachableText() };
    // Sem Super, Ctrl ou Alt, só teclas que não escrevem nada.
    if (!strong && !/^(f[0-9]+|print|pause|menu|scroll_lock|xf86.*)$/i.test(key))
        return { combo: "", error: "essa tecla sozinha pararia de digitar: junte Super, Ctrl ou Alt" };
    return { combo: format(modList, key), error: "" };
}

// Para onde uma ação de emergência pode ir, por extenso. As teclas saem de
// `everywhere`: a mensagem não tem lista própria para ficar para trás.
function reachableText() {
    var names = ["letra", "número", "F1–F12"].concat(everywhere.map(keyLabel));
    return names.slice(0, -1).join(", ") + " ou " + names[names.length - 1];
}

function keyLabel(key) {
    var k = String(key).toLowerCase();
    // Só o que está escrito na tabela ("constructor" existe em todo objeto).
    if (Object.prototype.hasOwnProperty.call(keyLabels, k))
        return keyLabels[k];
    if (k.indexOf("xf86") === 0)
        return String(key).slice(4).replace(/([a-z])([A-Z])/g, "$1 $2");
    if (k.indexOf("code:") === 0)
        return "tecla " + k.slice(5);
    return String(key).length === 1 ? String(key).toUpperCase() : String(key);
}

// "SUPER + SHIFT + RETURN" -> "Super + Shift + Enter"; numa família, a tecla
// vira o nome dela ("Super + setas"). "" -> "sem atalho".
function pretty(action, combo) {
    if (combo === "")
        return "sem atalho";
    var p = parse(combo);
    var parts = p.mods.map(function (m) { return modLabels[m]; });
    if (action && action.family === "super")
        return "Super (toque)";
    if (action && action.family)
        parts.push(familyLabels[action.family]);
    else if (p.key !== "")
        parts.push(keyLabel(p.key));
    return parts.join(" + ");
}

// Tecla apertada (event.key, event.nativeScanCode e event.modifiers do Qt) ->
// nome do Hyprland. "" para modificadores sozinhos. O que não tem nome vira
// "code:<n>".
function fromQt(key, scan, modifiers) {
    // Shift, Ctrl, Meta, Alt, CapsLock, NumLock, Super, AltGr.
    if ((key >= 0x01000020 && key <= 0x01000025) || key === 0x01000053 || key === 0x01000054
            || key === 0x01001103)
        return "";
    // O teclado numérico: o Qt entrega "1" e "+" com a marca de keypad, e para
    // o Hyprland KP_1 não é 1.
    if (modifiers & 0x20000000) {
        if (key >= 0x30 && key <= 0x39)
            return "KP_" + String.fromCharCode(key);
        var pad = { 0x2b: "KP_Add", 0x2d: "KP_Subtract", 0x2a: "KP_Multiply", 0x2f: "KP_Divide",
            0x2e: "KP_Decimal", 0x01000005: "KP_Enter", 0x01000004: "KP_Enter" };
        if (pad[key] !== undefined)
            return pad[key];
        return scan > 8 ? "code:" + scan : "";
    }
    // A fileira de números, pela posição: com Shift o Qt entrega "!" e "@".
    if (scan >= 10 && scan <= 19)
        return String((scan - 9) % 10);
    if (key > 0x20 && key < 0x7f && shifted[String.fromCharCode(key) + "@" + scan] !== undefined)
        return shifted[String.fromCharCode(key) + "@" + scan];
    if (key >= 0x41 && key <= 0x5a)
        return String.fromCharCode(key);
    if (key >= 0x30 && key <= 0x39)
        return String.fromCharCode(key);
    if (key >= 0x01000030 && key <= 0x0100003b)
        return "F" + (key - 0x01000030 + 1);
    var named = {
        0x01000000: "ESCAPE", 0x01000001: "TAB", 0x01000002: "TAB", 0x01000003: "BACKSPACE",
        0x01000004: "RETURN", 0x01000005: "KP_Enter", 0x01000006: "INSERT", 0x01000007: "DELETE",
        0x01000008: "PAUSE", 0x01000009: "PRINT", 0x01000010: "HOME", 0x01000011: "END",
        0x01000012: "Left", 0x01000013: "Up", 0x01000014: "Right", 0x01000015: "Down",
        0x01000016: "PRIOR", 0x01000017: "NEXT", 0x01000026: "Scroll_Lock", 0x01000055: "MENU",
        0x20: "SPACE", 0x27: "APOSTROPHE", 0x2c: "COMMA", 0x2d: "MINUS", 0x2e: "PERIOD",
        0x2f: "SLASH", 0x3b: "SEMICOLON", 0x3d: "EQUAL", 0x5b: "BRACKETLEFT", 0x5c: "BACKSLASH",
        0x5d: "BRACKETRIGHT", 0x60: "GRAVE", 0xc7: "CCEDILLA"
    };
    if (named[key] !== undefined)
        return named[key];
    return scan > 8 ? normalizeKey("code:" + scan) : "";
}

// Modificadores segurados (event.modifiers do Qt) -> lista na nossa ordem.
function modsFromQt(modifiers) {
    var out = [];
    if (modifiers & 0x10000000)
        out.push("SUPER");
    if (modifiers & 0x04000000)
        out.push("CTRL");
    if (modifiers & 0x08000000)
        out.push("ALT");
    if (modifiers & 0x02000000)
        out.push("SHIFT");
    return out;
}

// Para a página: [{ name, items: [ação...] }], na ordem da lista.
function groups() {
    var out = [];
    for (var i = 0; i < actions.length; i++) {
        if (out.length === 0 || out[out.length - 1].name !== actions[i].group)
            out.push({ name: actions[i].group, items: [] });
        out[out.length - 1].items.push(actions[i]);
    }
    return out;
}

// Para a busca dos Ajustes: [{ id, label, combo, group }] com o combo que vale agora.
function entries(overrides) {
    var all = effective(overrides);
    return actions.map(function (a) {
        return { id: a.id, label: a.desc, combo: pretty(a, all[a.id]), group: a.group };
    });
}

// A tabela Lua das trocas: { ["win.close"] = "SUPER + W" }.
function luaTable(overrides, indent) {
    var o = clean(overrides);
    var items = [];
    for (var id in o)
        items.push("[" + JSON.stringify(id) + "] = " + JSON.stringify(o[id]));
    if (items.length === 0)
        return "{}";
    return indent ? "{\n    " + items.join(",\n    ") + ",\n}" : "{ " + items.join(", ") + " }";
}

// O caminho de volta: as trocas que estão num atalhos.lua (o que lua() gera,
// ou a mesma tabela escrita à mão), já passadas por clean().
function fromLua(text) {
    var out = {};
    var body = String(text || "").replace(/^\s*--.*$/gm, "");
    var re = /\[\s*"([^"\\]+)"\s*\]\s*=\s*"([^"\\]*)"/g;
    var m;
    while ((m = re.exec(body)) !== null)
        out[m[1]] = m[2];
    return clean(out);
}

var masks = { "SHIFT": 1, "CTRL": 4, "ALT": 8, "SUPER": 64 };

// Confere o que o Hyprland tem de fato (`hyprctl -j binds`, casando pela
// descrição) com o que as trocas pedem. Devolve os ids das ações que ficaram
// diferentes: o apply() do binds.lua pode recusar um combo e voltar ao de fábrica.
function diverged(binds, overrides) {
    var all = effective(overrides);
    var out = [];
    if (!binds || binds.length === undefined)
        return out;
    // Uma sessão com o binds.lua antigo não põe descrição nos binds: não dá
    // para saber, e então não se acusa ninguém.
    var known = {};
    for (var d = 0; d < actions.length; d++)
        known[actions[d].desc] = true;
    if (!binds.some(function (b) { return known[b.description] === true; }))
        return out;
    for (var i = 0; i < actions.length; i++) {
        var a = actions[i];
        if (a.fixed)
            continue;
        var mine = binds.filter(function (b) { return b.description === a.desc; });
        var p = parse(all[a.id]);
        var mask = p.mods.reduce(function (sum, m) { return sum + masks[m]; }, 0);
        var want = all[a.id] === "" ? [] : a.family ? families[a.family] : [p.key];
        var ok = mine.length === want.length && want.every(function (k) {
            return mine.some(function (b) {
                return b.modmask === mask && (String(b.key).toLowerCase() === k.toLowerCase()
                    || "code:" + b.keycode === k.toLowerCase());
            });
        });
        if (!ok)
            out.push(a.id);
    }
    return out;
}

// <raiz>/atalhos.lua: só dados, lidos pelo binds.lua quando ele cria os atalhos.
function lua(overrides) {
    return "-- Gerado pelos Ajustes do Mk (Super + ,), página Atalhos. Não edite: o painel reescreve.\n"
        + "-- Só dados: config/hypr/mare/binds.lua lê esta tabela ao criar os atalhos.\n"
        + "return " + luaTable(overrides, true) + "\n";
}

// O Lua que troca os atalhos na sessão aberta (hyprctl eval).
function evalCode(overrides) {
    return "require(\"mare.binds\").apply(" + luaTable(overrides, false) + ")";
}

// Para a captura: os atalhos do Mk dormem por `seconds` (ou até o resume) e a
// janela dos Ajustes recebe o combo inteiro. Só um número finito de zero para
// cima entra no Lua; qualquer outra coisa sai fora, e o suspend() sem
// argumento usa o tempo do binds.lua (o relógio que acorda os atalhos existe
// sempre).
function suspendCode(seconds) {
    var ok = typeof seconds === "number" && isFinite(seconds) && seconds >= 0;
    return "require(\"mare.binds\").suspend(" + (ok ? String(seconds) : "") + ")";
}

function resumeCode() {
    return "require(\"mare.binds\").resume()";
}
