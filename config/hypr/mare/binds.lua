-- Atalhos. Uma tabela só: cada linha é uma ação (id, grupo, descrição, combo de
-- fábrica e o que ela faz), e apply() transforma a tabela em hl.bind. A mesma
-- lista, sem o `run`, está em shell/mare/Keybinds.js, para a página Atalhos dos
-- Ajustes: ao mexer aqui, mexa lá (ids, combos e marcas iguais; o comando que
-- confere os dois está no alto do Keybinds.js).
--
-- O que você troca nos Ajustes fica em <raiz>/atalhos.lua, um arquivo só de
-- dados (`return { ["win.close"] = "SUPER + W" }`; "" desliga o atalho). Este
-- módulo o lê antes de criar os binds, então cada ação nasce uma vez, já no
-- combo certo, sem desfazer nada. Se o arquivo faltar ou estiver quebrado,
-- valem os de fábrica. Na sessão aberta, os Ajustes chamam
-- `hyprctl eval 'require("mare.binds").apply({...})'`, que tira os binds
-- deste módulo e os cria de novo, sem recarregar o resto da config.
--
-- Os painéis do shell abrem pelo IPC do Maré: qs -p <shell> ipc call mare <...>
local paths = require("mare.paths")
local mare = "qs -p '" .. paths.shell .. "' ipc call mare "

local function shell(args)
    return hl.dsp.exec_cmd(mare .. args)
end

local function exec(cmd)
    return hl.dsp.exec_cmd(cmd)
end

-- Famílias: uma ação que vale para várias teclas. O combo da ação guarda só os
-- modificadores; a tecla vem daqui, com o argumento que `each` recebe.
local families = {
    -- Um toque no Super sozinho: o bind é de soltar a tecla, e o Hyprland não o
    -- dispara se outra tecla foi apertada junto (Super+Enter etc.), então os
    -- atalhos com Super continuam rodando normalmente.
    super = { { "SUPER_L" }, { "SUPER_R" } },
    arrows = { { "Left", "left" }, { "Right", "right" }, { "Up", "up" }, { "Down", "down" } },
    digits = {},
}
-- Workspaces = projetos, de 1 a 9.
for i = 1, 9 do
    families.digits[i] = { tostring(i), i }
end

local step = { left = { -40, 0 }, right = { 40, 0 }, up = { 0, -40 }, down = { 0, 40 } }
local media = { locked = true }
local held = { locked = true, repeating = true }

-- Marcas: required = não pode ficar sem atalho (emergência); fixed = os
-- Ajustes não trocam (mouse, teclas de mídia, toque no Super, Alt+Tab, que
-- depende de soltar o Alt). family = veja acima. opts = opções do hl.bind.
local actions = {
    -- Emergência: funcionam mesmo se o Maré não abrir (terminal e sair da sessão).
    { id = "emergency.terminal", group = "EMERGÊNCIA", desc = "terminal de emergência", combo = "SUPER + SHIFT + RETURN", required = true,
      run = exec("ghostty --config-default-files=false --config-file='?" .. paths.ghostty .. "' || kitty || foot || alacritty") },
    { id = "emergency.exit", group = "EMERGÊNCIA", desc = "sair do Hyprland", combo = "SUPER + SHIFT + ESCAPE", required = true,
      run = hl.dsp.exit() },

    -- Shell: launcher, Alt+Tab e painéis da ilha.
    { id = "mare.launcher.tap", group = "MARÉ", desc = "busca: apps, projetos e comandos (toque no Super)", combo = "SUPER", family = "super", fixed = true,
      opts = { release = true }, each = function() return shell("toggle launcher") end },
    { id = "mare.launcher", group = "MARÉ", desc = "busca: apps, projetos e comandos", combo = "SUPER + SPACE",
      run = shell("toggle launcher") },
    { id = "mare.switcher.next", group = "MARÉ", desc = "trocar de janela", combo = "ALT + TAB", fixed = true,
      run = shell("switcher next") },
    { id = "mare.switcher.prev", group = "MARÉ", desc = "trocar de janela, para trás", combo = "ALT + SHIFT + TAB", fixed = true,
      run = shell("switcher prev") },
    { id = "mare.project", group = "MARÉ", desc = "projeto do workspace", combo = "SUPER + P",
      run = shell("toggle project") },
    { id = "mare.calendar", group = "MARÉ", desc = "calendário", combo = "SUPER + C",
      run = shell("toggle calendar") },
    { id = "mare.notifications", group = "MARÉ", desc = "notificações", combo = "SUPER + N",
      run = shell("toggle notifications") },
    { id = "mare.dnd", group = "MARÉ", desc = "não perturbe", combo = "SUPER + SHIFT + N",
      run = shell("dnd") },
    { id = "mare.session", group = "MARÉ", desc = "sair, bloquear, desligar", combo = "SUPER + ESCAPE",
      run = shell("toggle session") },
    { id = "mare.wallpaper", group = "MARÉ", desc = "papel de parede", combo = "SUPER + W",
      run = shell("toggle wallpaper") },
    { id = "mare.clipboard", group = "MARÉ", desc = "área de transferência", combo = "SUPER + SHIFT + V",
      run = shell("toggle clipboard") },
    { id = "mare.emoji", group = "MARÉ", desc = "emoji", combo = "SUPER + PERIOD",
      run = shell("toggle emoji") },
    { id = "mare.settings", group = "MARÉ", desc = "estes ajustes", combo = "SUPER + COMMA",
      run = shell("toggle settings") },
    { id = "mare.lock", group = "MARÉ", desc = "bloquear a tela", combo = "SUPER + L",
      run = shell("lock") },

    -- Programas no diretório do projeto em foco
    { id = "app.terminal", group = "APPS", desc = "terminal no projeto", combo = "SUPER + RETURN",
      run = shell("here ''") },
    { id = "app.lazygit", group = "APPS", desc = "lazygit", combo = "SUPER + G",
      run = shell("here lazygit") },
    { id = "app.lazydocker", group = "APPS", desc = "lazydocker", combo = "SUPER + D",
      run = shell("here lazydocker") },
    { id = "app.files", group = "APPS", desc = "arquivos (yazi)", combo = "SUPER + E",
      run = shell("here yazi") },
    { id = "app.browser", group = "APPS", desc = "navegador", combo = "SUPER + B",
      run = exec("bash '" .. paths.shell .. "/scripts/mare-default-apps' open navegador || chromium") },

    -- Janelas
    { id = "win.close", group = "JANELAS", desc = "fechar", combo = "SUPER + Q",
      run = hl.dsp.window.close() },
    { id = "win.fullscreen", group = "JANELAS", desc = "tela cheia", combo = "SUPER + F",
      run = hl.dsp.window.fullscreen() },
    { id = "win.float", group = "JANELAS", desc = "flutuar", combo = "SUPER + V",
      run = hl.dsp.window.float({ action = "toggle" }) },
    { id = "win.focus", group = "JANELAS", desc = "mover o foco", combo = "SUPER", family = "arrows",
      each = function(dir) return hl.dsp.focus({ direction = dir }) end },
    { id = "win.move", group = "JANELAS", desc = "mover a janela", combo = "SUPER + SHIFT", family = "arrows",
      each = function(dir) return hl.dsp.window.move({ direction = dir }) end },
    { id = "win.resize", group = "JANELAS", desc = "redimensionar", combo = "SUPER + CTRL", family = "arrows",
      opts = { repeating = true }, each = function(dir) return hl.dsp.window.resize({ x = step[dir][1], y = step[dir][2], relative = true }) end },
    { id = "win.drag", group = "JANELAS", desc = "mover com o mouse (arrastar)", combo = "SUPER + mouse:272", fixed = true,
      opts = { mouse = true }, run = hl.dsp.window.drag() },
    { id = "win.dragsize", group = "JANELAS", desc = "redimensionar com o mouse (botão direito)", combo = "SUPER + mouse:273", fixed = true,
      opts = { mouse = true }, run = hl.dsp.window.resize() },

    -- Workspaces = projetos. Entrar num workspace novo e vazio abre o seletor de projeto.
    { id = "ws.focus", group = "WORKSPACES", desc = "ir para α β γ…", combo = "SUPER", family = "digits",
      each = function(i) return hl.dsp.focus({ workspace = i }) end },
    { id = "ws.move", group = "WORKSPACES", desc = "levar a janela", combo = "SUPER + SHIFT", family = "digits",
      each = function(i) return hl.dsp.window.move({ workspace = i }) end },
    { id = "ws.previous", group = "WORKSPACES", desc = "workspace anterior", combo = "SUPER + TAB",
      run = hl.dsp.focus({ workspace = "previous" }) },
    { id = "ws.next.wheel", group = "WORKSPACES", desc = "workspace seguinte (roda do mouse)", combo = "SUPER + mouse_down", fixed = true,
      run = hl.dsp.focus({ workspace = "e+1" }) },
    { id = "ws.prev.wheel", group = "WORKSPACES", desc = "workspace anterior (roda do mouse)", combo = "SUPER + mouse_up", fixed = true,
      run = hl.dsp.focus({ workspace = "e-1" }) },

    -- Captura de tela: área para a área de transferência / tela inteira para arquivo
    { id = "shot.area", group = "TELA", desc = "recortar e copiar", combo = "PRINT",
      run = shell("shot copiar") },
    { id = "shot.screen", group = "TELA", desc = "salvar a tela inteira", combo = "SHIFT + PRINT",
      run = shell("shot tela") },
    { id = "shot.panel", group = "TELA", desc = "painel de captura (área, janela, vídeo)", combo = "SUPER + PRINT",
      run = shell("toggle capture") },
    { id = "shot.record", group = "TELA", desc = "gravar a tela / parar", combo = "SUPER + SHIFT + R",
      run = shell("record") },

    -- Teclas de mídia: passam pelo Maré, que mostra o OSD na linha
    { id = "media.volume.up", group = "TECLAS DE MÍDIA", desc = "aumentar o volume", combo = "XF86AudioRaiseVolume", fixed = true,
      opts = held, run = shell("volume +5") },
    { id = "media.volume.down", group = "TECLAS DE MÍDIA", desc = "diminuir o volume", combo = "XF86AudioLowerVolume", fixed = true,
      opts = held, run = shell("volume -5") },
    { id = "media.volume.mute", group = "TECLAS DE MÍDIA", desc = "silenciar", combo = "XF86AudioMute", fixed = true,
      opts = media, run = shell("volume mute") },
    { id = "media.mic.mute", group = "TECLAS DE MÍDIA", desc = "silenciar o microfone", combo = "XF86AudioMicMute", fixed = true,
      opts = media, run = exec("wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle") },
    { id = "media.brightness.up", group = "TECLAS DE MÍDIA", desc = "aumentar o brilho", combo = "XF86MonBrightnessUp", fixed = true,
      opts = held, run = shell("brightness +5") },
    { id = "media.brightness.down", group = "TECLAS DE MÍDIA", desc = "diminuir o brilho", combo = "XF86MonBrightnessDown", fixed = true,
      opts = held, run = shell("brightness -5") },
    { id = "media.play", group = "TECLAS DE MÍDIA", desc = "tocar / pausar", combo = "XF86AudioPlay", fixed = true,
      opts = media, run = exec("playerctl play-pause") },
    { id = "media.next", group = "TECLAS DE MÍDIA", desc = "próxima faixa", combo = "XF86AudioNext", fixed = true,
      opts = media, run = exec("playerctl next") },
    { id = "media.prev", group = "TECLAS DE MÍDIA", desc = "faixa anterior", combo = "XF86AudioPrev", fixed = true,
      opts = media, run = exec("playerctl previous") },
}

local M = { actions = actions, families = families, handles = {}, current = {}, unbound = {}, suspended = false }

-- Os modificadores, na ordem em que o combo é escrito (a mesma do Keybinds.js),
-- e os outros nomes que o Hyprland aceita para eles.
local modorder = { "super", "ctrl", "alt", "shift" }
local modnames = {
    super = "super", win = "super", logo = "super", mod4 = "super",
    ctrl = "ctrl", control = "ctrl",
    alt = "alt", mod1 = "alt",
    shift = "shift",
}

-- "Shift + SUPER + Return" -> { super = true, shift = true }, "return".
local function split(combo)
    local mods, key = {}, ""
    for part in combo:gmatch("[^+]+") do
        local p = part:match("^%s*(.-)%s*$"):lower()
        if modnames[p] then
            mods[modnames[p]] = true
        elseif p ~= "" then
            key = p
        end
    end
    return mods, key
end

-- O combo numa forma só, para comparar: minúsculas, modificadores em ordem.
-- Sem isso "SHIFT + SUPER + RETURN" passaria por livre ao lado de
-- "SUPER + SHIFT + RETURN" e as duas ações rodariam juntas.
local function canon(combo)
    local mods, key = split(combo)
    local out = {}
    for _, m in ipairs(modorder) do
        if mods[m] then
            out[#out + 1] = m
        end
    end
    if key ~= "" then
        out[#out + 1] = key
    end
    return table.concat(out, "+")
end

-- As de emergência só mudam para uma tecla que todo teclado tem, junto de
-- Super, Ctrl ou Alt: numa tecla que o teclado não tem (XF86Launch2, code:255)
-- o atalho existiria e ninguém conseguiria apertá-lo. A mesma regra está em
-- validate(), no Keybinds.js.
local everywhere = { ["return"] = true, escape = true, space = true, tab = true, backspace = true, delete = true }
local function reachable(combo)
    local mods, key = split(combo)
    if not (mods.super or mods.ctrl or mods.alt) then
        return false
    end
    if everywhere[key] or key:match("^[a-z0-9]$") then
        return true
    end
    local f = tonumber(key:match("^f(%d%d?)$"))
    return f ~= nil and f >= 1 and f <= 12
end

-- As combinações que uma ação ocupa: uma só, ou uma por tecla da família.
local function expand(action, combo)
    if not action.family then
        return { { combo } }
    end
    local out = {}
    for _, key in ipairs(families[action.family]) do
        out[#out + 1] = { combo .. " + " .. key[1], key[2] }
    end
    return out
end

-- Tenta dar o combo à ação. Não dá se alguma combinação já tem dono (duas
-- ações na mesma tecla rodariam juntas) ou se o Hyprland recusar a tecla;
-- nesse caso desfaz o que criou e devolve false.
local function try(action, combo, taken)
    local keys = expand(action, combo)
    for _, k in ipairs(keys) do
        if taken[canon(k[1])] then
            print("Mk: atalho " .. action.id .. " (" .. k[1] .. ") já é de " .. taken[canon(k[1])])
            return false
        end
    end
    local made = {}
    for _, k in ipairs(keys) do
        local opts = { description = action.desc }
        for name, value in pairs(action.opts or {}) do
            opts[name] = value
        end
        -- Uma tecla desconhecida pode vir como erro ou como bind vazio.
        local ok, handle = pcall(hl.bind, k[1], action.each and action.each(k[2]) or action.run, opts)
        if not ok or not handle then
            for _, h in ipairs(made) do
                pcall(h.remove, h)
            end
            print("Mk: atalho " .. action.id .. " (" .. k[1] .. ") recusado: " .. tostring(handle))
            return false
        end
        made[#made + 1] = handle
    end
    for i, k in ipairs(keys) do
        taken[canon(k[1])] = action.id
        M.handles[#M.handles + 1] = made[i]
    end
    M.current[action.id] = combo
    return true
end

-- (Re)cria todos os binds. `overrides` é { [id] = "COMBO" }; "" desliga a ação.
-- As fixas ignoram a troca; as de emergência nunca ficam sem atalho nem vão
-- para uma tecla que nem todo teclado tem. Quem pede um combo que não dá
-- (ocupado ou inválido) volta para o de fábrica. Em M.current fica o combo
-- com que cada ação acabou, e em M.unbound quem ficou sem nenhum sem ter pedido.
function M.apply(overrides)
    overrides = type(overrides) == "table" and overrides or {}
    for _, handle in ipairs(M.handles) do
        pcall(handle.remove, handle)
    end
    M.handles, M.current, M.unbound, M.suspended = {}, {}, {}, false
    if M.timer then
        pcall(M.timer.set_enabled, M.timer, false)
        M.timer = nil
    end

    local wanted = {}
    for _, action in ipairs(actions) do
        local combo = overrides[action.id]
        if action.fixed or type(combo) ~= "string" or (combo == "" and (action.required or action.family))
            or (action.required and not reachable(combo)) then
            combo = action.combo
        end
        wanted[action.id] = combo
    end
    local function stock(a) return canon(wanted[a.id]) == canon(a.combo) end

    -- Uma de emergência escolhe antes dos outros, então não pode pedir o combo
    -- de fábrica de quem não foi mexido: tiraria o atalho dele. Se pede, volta
    -- para o dela, e confere-se de novo, porque isso muda quem não foi mexido.
    -- A mesma regra está em clean(), no Keybinds.js.
    local again = true
    while again do
        again = false
        local home = {}
        for _, action in ipairs(actions) do
            if stock(action) then
                for _, k in ipairs(expand(action, action.combo)) do
                    home[canon(k[1])] = true
                end
            end
        end
        for _, action in ipairs(actions) do
            if action.required and not action.fixed and not stock(action) and home[canon(wanted[action.id])] then
                print("Mk: atalho " .. action.id .. " (" .. wanted[action.id] .. ") é de outra ação, fica o de fábrica")
                wanted[action.id] = action.combo
                again = true
                break
            end
        end
    end

    -- Em quatro voltas, para a ordem de quem escolhe primeiro: as fixas, as de
    -- emergência, quem está no combo de fábrica e, por último, quem trocou.
    -- Assim uma troca nunca tira o atalho de quem não foi mexido: se ela bate
    -- em alguém, é ela que volta para o de fábrica.
    local taken = {}
    local function pass(filter)
        for _, action in ipairs(actions) do
            if filter(action) and wanted[action.id] ~= "" then
                local ok = try(action, wanted[action.id], taken)
                if not ok and wanted[action.id] ~= action.combo then
                    ok = try(action, action.combo, taken)
                end
                if not ok then
                    M.unbound[#M.unbound + 1] = action.id
                    print("Mk: atalho " .. action.id .. " ficou sem tecla")
                end
            end
        end
    end
    pass(function(a) return a.fixed end)
    pass(function(a) return a.required and not a.fixed end)
    pass(function(a) return not a.required and not a.fixed and stock(a) end)
    pass(function(a) return not a.required and not a.fixed and not stock(a) end)
end

-- Para os Ajustes capturarem um combo: o Hyprland engole os atalhos que já
-- existem antes de a janela vê-los, então os do Mk dormem enquanto a página
-- espera a tecla (continuam na lista, só desligados). Voltam com resume(),
-- com um apply() ou sozinhos depois de `seconds` (8 se não vier), para a
-- sessão nunca ficar sem atalhos se o Maré cair no meio da captura.
local function enable(on)
    for _, handle in ipairs(M.handles) do
        pcall(handle.set_enabled, handle, on)
    end
    M.suspended = not on
end

function M.resume()
    if M.timer then
        pcall(M.timer.set_enabled, M.timer, false)
        M.timer = nil
    end
    enable(true)
end

function M.suspend(seconds)
    M.resume()
    enable(false)
    local ok, timer = pcall(hl.timer, M.resume, { timeout = math.floor((tonumber(seconds) or 8) * 1000), type = "oneshot" })
    if ok and timer then
        M.timer = timer
    else
        -- Sem o relógio de segurança não vale o risco.
        enable(true)
        error("Mk: sem hl.timer, os atalhos não foram suspensos")
    end
end

-- <raiz>/atalhos.lua, lido como dado: roda sem acesso a nada (ambiente vazio)
-- e só vale se devolver uma tabela.
function M.saved()
    local file = paths.root .. "atalhos.lua"
    if not package.searchpath("atalhos", paths.root .. "?.lua") then
        return {}
    end
    local chunk, err = loadfile(file, "t", {})
    if not chunk then
        print("Mk: atalhos.lua falhou: " .. tostring(err))
        return {}
    end
    local ok, result = pcall(chunk)
    if not ok then
        print("Mk: atalhos.lua falhou: " .. tostring(result))
        return {}
    end
    return type(result) == "table" and result or {}
end

M.apply(M.saved())

return M
