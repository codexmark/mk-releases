-- config/hypr/mare/binds.lua: a tabela de ações vira hl.bind, o atalhos.lua
-- troca e desliga, as de emergência nunca ficam sem tecla alcançável e duas
-- ações nunca dividem o mesmo combo.
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

local function with(overrides)
    local lines = {}
    for id, combo in pairs(overrides) do
        lines[#lines + 1] = string.format("[%q] = %q,", id, combo)
    end
    return T.session({ only = "mare.binds", files = { ["atalhos.lua"] = "return {\n" .. table.concat(lines, "\n") .. "\n}\n" } })
end
local function action(s, id)
    for _, a in ipairs(s.binds.actions) do
        if a.id == id then
            return a
        end
    end
    error("ação desconhecida: " .. id)
end
-- Quantos binds vivos têm a descrição desta ação.
local function count(s, id)
    local n, desc = 0, action(s, id).desc
    for _, b in ipairs(s:live()) do
        if b.opts.description == desc then
            n = n + 1
        end
    end
    return n
end
-- A forma única de um combo, como o Hyprland o entende: sem ordem nos modificadores.
local function canon(keys)
    local parts = {}
    for part in keys:gmatch("[^+]+") do
        parts[#parts + 1] = part:match("^%s*(.-)%s*$"):lower()
    end
    table.sort(parts)
    return table.concat(parts, "+")
end
local function stock_count(s)
    local n = 0
    for _, a in ipairs(s.binds.actions) do
        n = n + (a.family and #s.binds.families[a.family] or 1)
    end
    return n
end

-- ── de fábrica ────────────────────────────────────────────────────────
T.t("sem atalhos.lua: carrega sem erro nem aviso", function()
    local s = T.session({ only = "mare.binds" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("", s.prints)
end)

T.t("toda ação vira bind: uma por ação, uma por tecla nas famílias", function()
    local s = T.session({ only = "mare.binds" })
    T.eq(stock_count(s), #s:live())
    for _, a in ipairs(s.binds.actions) do
        T.eq(a.combo, s.binds.current[a.id], a.id)
    end
    T.eq({}, s.binds.unbound, "ninguém sem tecla")
end)

T.t("cada bind leva a descrição da ação e um despacho", function()
    local s = T.session({ only = "mare.binds" })
    local descs = {}
    for _, a in ipairs(s.binds.actions) do
        descs[a.desc] = true
    end
    for _, b in ipairs(s:live()) do
        T.ok(descs[b.opts.description], "descrição de " .. b.keys)
        T.ok(type(b.action) == "table" and b.action.dsp, "despacho de " .. b.keys)
    end
end)

T.t("as ações têm id único, grupo, descrição e combo", function()
    local s = T.session({ only = "mare.binds" })
    local seen = {}
    for _, a in ipairs(s.binds.actions) do
        T.ok(not seen[a.id], "id repetido: " .. tostring(a.id))
        seen[a.id] = true
        T.ok(type(a.group) == "string" and a.group ~= "", a.id .. ": grupo")
        T.ok(type(a.desc) == "string" and a.desc ~= "", a.id .. ": descrição")
        T.ok(type(a.combo) == "string" and a.combo ~= "", a.id .. ": combo")
        T.ok(a.run or a.each, a.id .. ": o que faz")
        T.ok(not a.family or s.binds.families[a.family], a.id .. ": família")
    end
end)

T.t("de fábrica, nenhum combo é de duas ações", function()
    local s = T.session({ only = "mare.binds" })
    local owner = {}
    for _, b in ipairs(s:live()) do
        local c = canon(b.keys)
        T.ok(not owner[c], b.keys .. " é de '" .. tostring(owner[c]) .. "' e de '" .. b.opts.description .. "'")
        owner[c] = b.opts.description
    end
end)

T.t("as famílias: setas para foco/mover/redimensionar, 1 a 9 para os workspaces", function()
    local s = T.session({ only = "mare.binds" })
    T.eq({ dsp = "focus", args = { { direction = "left" } } }, s:bind("SUPER + Left").action)
    T.eq({ dsp = "window.move", args = { { direction = "down" } } }, s:bind("SUPER + SHIFT + Down").action)
    T.eq({ dsp = "window.resize", args = { { x = 40, y = 0, relative = true } } }, s:bind("SUPER + CTRL + Right").action)
    T.eq(true, s:bind("SUPER + CTRL + Right").opts.repeating)
    for i = 1, 9 do
        T.eq({ dsp = "focus", args = { { workspace = i } } }, s:bind("SUPER + " .. i).action)
        T.eq({ dsp = "window.move", args = { { workspace = i } } }, s:bind("SUPER + SHIFT + " .. i).action)
    end
    T.eq(nil, s:bind("SUPER + 0"))
end)

T.t("o toque no Super abre a busca ao soltar, nas duas teclas Super", function()
    local s = T.session({ only = "mare.binds" })
    for _, key in ipairs({ "SUPER + SUPER_L", "SUPER + SUPER_R" }) do
        local b = s:bind(key)
        T.ok(b, key)
        T.eq(true, b.opts.release)
        T.has(b.action.args[1], "ipc call mare toggle launcher")
    end
end)

T.t("os painéis abrem pelo IPC do Maré instalado ao lado (não o de ~/.config/quickshell)", function()
    local s = T.session({ only = "mare.binds" })
    T.eq("qs -p '" .. s.root .. "shell' ipc call mare toggle settings", s:bind("SUPER + COMMA").action.args[1])
    T.eq("qs -p '" .. s.root .. "shell' ipc call mare lock", s:bind("SUPER + L").action.args[1])
end)

T.t("as teclas de mídia valem com a tela bloqueada, e volume e brilho repetem", function()
    local s = T.session({ only = "mare.binds" })
    for _, key in ipairs({ "XF86AudioMute", "XF86AudioMicMute", "XF86AudioPlay", "XF86AudioNext", "XF86AudioPrev" }) do
        T.eq(true, s:bind(key).opts.locked, key)
        T.eq(nil, s:bind(key).opts.repeating, key)
    end
    for _, key in ipairs({ "XF86AudioRaiseVolume", "XF86AudioLowerVolume", "XF86MonBrightnessUp", "XF86MonBrightnessDown" }) do
        T.eq(true, s:bind(key).opts.locked, key)
        T.eq(true, s:bind(key).opts.repeating, key)
    end
end)

T.t("os atalhos do mouse são binds de mouse", function()
    local s = T.session({ only = "mare.binds" })
    T.eq(true, s:bind("SUPER + mouse:272").opts.mouse)
    T.eq("window.drag", s:bind("SUPER + mouse:272").action.dsp)
    T.eq("window.resize", s:bind("SUPER + mouse:273").action.dsp)
end)

T.t("emergência de fábrica: terminal (sem depender do Maré) e sair do Hyprland", function()
    local s = T.session({ only = "mare.binds" })
    local term = s:bind("SUPER + SHIFT + RETURN").action
    T.eq("exec_cmd", term.dsp)
    T.has(term.args[1], "ghostty")
    T.has(term.args[1], "|| kitty || foot || alacritty")
    T.hasnt(term.args[1], "qs ")
    T.eq({ dsp = "exit", args = {} }, s:bind("SUPER + SHIFT + ESCAPE").action)
    for _, id in ipairs({ "emergency.terminal", "emergency.exit" }) do
        T.eq(true, action(s, id).required, id)
    end
end)

-- ── atalhos.lua ───────────────────────────────────────────────────────
T.t('{ id = "COMBO" } troca: o combo novo existe e o de fábrica deixa de existir', function()
    local s = with({ ["win.close"] = "SUPER + X" })
    T.eq("", s.prints)
    T.eq("window.close", s:bind("SUPER + X").action.dsp)
    T.eq(nil, s:bind("SUPER + Q"))
    T.eq("SUPER + X", s.binds.current["win.close"])
    T.eq(stock_count(s), #s:live(), "o total não muda")
end)

T.t('"" desliga o atalho: a ação fica sem bind e ninguém reclama', function()
    local s = with({ ["mare.calendar"] = "" })
    T.eq("", s.prints)
    T.eq(nil, s:bind("SUPER + C"))
    T.eq(0, count(s, "mare.calendar"))
    T.eq(nil, s.binds.current["mare.calendar"])
    T.eq({}, s.binds.unbound, "desligada a pedido não é 'sem tecla'")
    T.eq(stock_count(s) - 1, #s:live())
end)

T.t("um combo liberado por \"\" pode ser dado a outra ação", function()
    local s = with({ ["mare.calendar"] = "", ["win.close"] = "SUPER + C" })
    T.eq("window.close", s:bind("SUPER + C").action.dsp)
    T.eq(0, count(s, "mare.calendar"))
end)

T.t("duas ações podem trocar de combo entre si", function()
    local s = with({ ["win.close"] = "SUPER + W", ["mare.wallpaper"] = "SUPER + Q" })
    T.eq("", s.prints)
    T.eq("window.close", s:bind("SUPER + W").action.dsp)
    T.has(s:bind("SUPER + Q").action.args[1], "toggle wallpaper")
end)

T.t("família: a troca muda os modificadores de todas as teclas", function()
    local s = with({ ["win.focus"] = "ALT" })
    T.eq({ dsp = "focus", args = { { direction = "up" } } }, s:bind("ALT + Up").action)
    T.eq(nil, s:bind("SUPER + Up"))
    T.eq(4, count(s, "win.focus"))
end)

T.t('família não desliga com "": fica a de fábrica', function()
    local s = with({ ["ws.focus"] = "" })
    T.eq(9, count(s, "ws.focus"))
    T.ok(s:bind("SUPER + 1"))
end)

T.t("as fixas (mouse, mídia, toque no Super, Alt+Tab) ignoram a troca e o \"\"", function()
    local s = with({ ["mare.switcher.next"] = "SUPER + J", ["media.play"] = "", ["win.drag"] = "ALT + mouse:272", ["mare.launcher.tap"] = "" })
    T.ok(s:bind("ALT + TAB"))
    T.eq(nil, s:bind("SUPER + J"))
    T.ok(s:bind("XF86AudioPlay"))
    T.ok(s:bind("SUPER + mouse:272"))
    T.eq(nil, s:bind("ALT + mouse:272"))
    T.ok(s:bind("SUPER + SUPER_L"))
end)

T.t("um id que não existe e um valor que não é texto são ignorados", function()
    local s = T.session({ only = "mare.binds", files = { ["atalhos.lua"] = 'return { ["nao.existe"] = "SUPER + Z", ["win.close"] = 42, ["win.float"] = { "SUPER + Y" } }\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(nil, s:bind("SUPER + Z"))
    T.ok(s:bind("SUPER + Q"))
    T.ok(s:bind("SUPER + V"))
    T.eq(stock_count(s), #s:live())
end)

-- ── emergência ────────────────────────────────────────────────────────
T.t('emergência não desliga com "": fica a de fábrica', function()
    local s = with({ ["emergency.terminal"] = "", ["emergency.exit"] = "" })
    T.ok(s:bind("SUPER + SHIFT + RETURN"))
    T.ok(s:bind("SUPER + SHIFT + ESCAPE"))
end)

T.t("emergência aceita letra, número, F1 a F12 e as teclas que todo teclado tem, com Super, Ctrl ou Alt", function()
    for _, combo in ipairs({ "SUPER + T", "CTRL + ALT + T", "ALT + F4", "SUPER + F12", "SUPER + SHIFT + BACKSPACE",
        "CTRL + ALT + DELETE", "SUPER + ALT + SPACE", "CTRL + ALT + 0", "SUPER + CTRL + TAB" }) do
        local s = with({ ["emergency.terminal"] = combo })
        T.ok(s:bind(combo), combo .. " devia valer")
        T.eq(nil, s:bind("SUPER + SHIFT + RETURN"), combo .. ": a de fábrica sai")
        T.eq(combo, s.binds.current["emergency.terminal"])
    end
end)

T.t("emergência recusa tecla que nem todo teclado tem: fica a de fábrica", function()
    for _, combo in ipairs({ "SUPER + XF86Launch2", "SUPER + code:255", "SUPER + F13", "SUPER + F0", "SUPER + PRINT",
        "SUPER + mouse:274", "SUPER + XF86AudioPlay", "SUPER + Left", "SUPER + ç", "SUPER + PERIOD" }) do
        local s = with({ ["emergency.exit"] = combo })
        -- (o combo pode ser de outra ação, como SUPER + PRINT; só não pode sair da sessão)
        local b = s:bind(combo)
        T.ok(not b or b.action.dsp ~= "exit", combo .. " não devia valer")
        T.ok(s:bind("SUPER + SHIFT + ESCAPE"), combo .. ": fica a de fábrica")
        T.eq("SUPER + SHIFT + ESCAPE", s.binds.current["emergency.exit"])
    end
end)

T.t("emergência recusa combo sem Super, Ctrl nem Alt (só Shift, ou tecla sozinha)", function()
    for _, combo in ipairs({ "SHIFT + T", "T", "ESCAPE", "SHIFT + F5", "SUPER" }) do
        local s = with({ ["emergency.terminal"] = combo })
        T.ok(s:bind("SUPER + SHIFT + RETURN"), combo .. ": fica a de fábrica")
        T.eq("SUPER + SHIFT + RETURN", s.binds.current["emergency.terminal"])
        T.eq(stock_count(s), #s:live(), combo .. ": ninguém perde o atalho")
    end
end)

T.t("emergência não toma o combo de fábrica de outra ação: volta para o dela, e a outra fica", function()
    local s = with({ ["emergency.exit"] = "SUPER + Q" })
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    T.eq("exit", s:bind("SUPER + SHIFT + ESCAPE").action.dsp)
    T.has(s.prints, "emergency.exit")
end)

T.t("emergência pode ficar com o combo de uma ação que saiu dele", function()
    local s = with({ ["emergency.exit"] = "SUPER + Q", ["win.close"] = "SUPER + X" })
    T.eq("exit", s:bind("SUPER + Q").action.dsp)
    T.eq("window.close", s:bind("SUPER + X").action.dsp)
end)

T.t("as duas de emergência pedindo o mesmo combo: a primeira leva, a segunda fica na de fábrica", function()
    local s = with({ ["emergency.terminal"] = "CTRL + ALT + T", ["emergency.exit"] = "CTRL + ALT + T" })
    T.eq("exec_cmd", s:bind("CTRL + ALT + T").action.dsp)
    T.eq("exit", s:bind("SUPER + SHIFT + ESCAPE").action.dsp)
    T.eq({}, s.binds.unbound)
end)

T.t("uma de emergência pedindo o combo de fábrica da outra: cada uma fica com o seu", function()
    local s = with({ ["emergency.terminal"] = "SUPER + SHIFT + ESCAPE" })
    T.eq("exec_cmd", s:bind("SUPER + SHIFT + RETURN").action.dsp)
    T.eq("exit", s:bind("SUPER + SHIFT + ESCAPE").action.dsp)
end)

-- ── combos repetidos ──────────────────────────────────────────────────
local function no_duplicates(s, what)
    local owner = {}
    for _, b in ipairs(s:live()) do
        local c = canon(b.keys)
        T.ok(not owner[c], what .. ": " .. b.keys .. " é de duas ações")
        owner[c] = true
    end
end

T.t("pedir o combo de fábrica de outra ação: quem pediu volta para o seu, a outra não perde nada", function()
    local s = with({ ["win.close"] = "SUPER + W" })
    T.has(s:bind("SUPER + W").action.args[1], "toggle wallpaper")
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    T.eq("SUPER + Q", s.binds.current["win.close"])
    T.has(s.prints, "win.close")
    no_duplicates(s, "troca")
    T.eq(stock_count(s), #s:live())
end)

T.t("o mesmo combo com os modificadores em outra ordem (ou outra caixa) também é repetido", function()
    local s = with({ ["win.close"] = "shift + super + v" })
    T.has(s:bind("SUPER + SHIFT + V").action.args[1], "toggle clipboard")
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    no_duplicates(s, "ordem")
end)

T.t("os outros nomes dos modificadores (WIN, CONTROL, MOD1) contam como os mesmos", function()
    local s = with({ ["win.close"] = "WIN + SHIFT + V", ["win.float"] = "MOD4 + CONTROL + Left" })
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    T.eq("window.float", s:bind("SUPER + V").action.dsp)
    T.eq(stock_count(s), #s:live())
end)

T.t("duas ações pedindo o mesmo combo livre: a primeira da tabela leva, a outra volta para o seu", function()
    local s = with({ ["win.close"] = "SUPER + Z", ["win.fullscreen"] = "SUPER + Z" })
    T.eq("window.close", s:bind("SUPER + Z").action.dsp)
    T.eq("window.fullscreen", s:bind("SUPER + F").action.dsp)
    no_duplicates(s, "livre")
    T.eq({}, s.binds.unbound)
end)

T.t("pedir uma tecla de uma família (SUPER + 3, SUPER + Left) não tira a tecla da família", function()
    local s = with({ ["win.close"] = "SUPER + 3", ["win.float"] = "SUPER + Left" })
    T.eq({ dsp = "focus", args = { { workspace = 3 } } }, s:bind("SUPER + 3").action)
    T.eq({ dsp = "focus", args = { { direction = "left" } } }, s:bind("SUPER + Left").action)
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    no_duplicates(s, "família")
end)

T.t("uma família trocada para os modificadores de outra volta inteira para os seus", function()
    local s = with({ ["win.focus"] = "SUPER + SHIFT" })
    T.eq("focus", s:bind("SUPER + Up").action.dsp)
    T.eq("window.move", s:bind("SUPER + SHIFT + Up").action.dsp)
    T.eq(4, count(s, "win.focus"))
    no_duplicates(s, "famílias")
end)

T.t("pedir o combo de uma fixa (ALT + TAB, tecla de mídia) não a derruba", function()
    local s = with({ ["win.close"] = "ALT + TAB", ["win.float"] = "XF86AudioMute" })
    T.has(s:bind("ALT + TAB").action.args[1], "switcher next")
    T.has(s:bind("XF86AudioMute").action.args[1], "volume mute")
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    T.eq("window.float", s:bind("SUPER + V").action.dsp)
end)

T.t("qualquer que seja o atalhos.lua, nunca há dois binds no mesmo combo", function()
    local casos = {
        { ["win.close"] = "SUPER + F", ["win.fullscreen"] = "SUPER + Q", ["win.float"] = "SUPER + Q" },
        { ["mare.launcher"] = "SUPER + RETURN", ["app.terminal"] = "SUPER + SPACE", ["emergency.terminal"] = "SUPER + SPACE" },
        { ["ws.focus"] = "SUPER + SHIFT", ["ws.move"] = "SUPER", ["win.focus"] = "SUPER + CTRL", ["win.resize"] = "SUPER" },
        { ["shot.area"] = "SUPER + PRINT", ["shot.panel"] = "PRINT", ["shot.screen"] = "PRINT" },
    }
    for i, overrides in ipairs(casos) do
        local s = with(overrides)
        no_duplicates(s, "caso " .. i)
        T.ok(s:bind("SUPER + SHIFT + RETURN") or s.binds.current["emergency.terminal"] ~= "SUPER + SHIFT + RETURN", "caso " .. i .. ": emergência")
        for _, id in ipairs({ "emergency.terminal", "emergency.exit" }) do
            T.eq(1, count(s, id), "caso " .. i .. ": " .. id)
        end
    end
end)

-- ── teclas que o Hyprland recusa ──────────────────────────────────────
T.t("o Hyprland recusa a tecla nova (erro): a ação volta para a de fábrica, com aviso", function()
    local s = T.session({ only = "mare.binds", refuse = { ["SUPER + Bobagem"] = "erro" },
        files = { ["atalhos.lua"] = 'return { ["win.close"] = "SUPER + Bobagem" }\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
    T.has(s.prints, "recusado")
    T.eq({}, s.binds.unbound)
end)

T.t("o Hyprland devolve um bind vazio para a tecla nova: idem", function()
    local s = T.session({ only = "mare.binds", refuse = { ["SUPER + Bobagem"] = "nil" },
        files = { ["atalhos.lua"] = 'return { ["win.close"] = "SUPER + Bobagem" }\n' } })
    T.eq("window.close", s:bind("SUPER + Q").action.dsp)
end)

T.t("família com uma tecla recusada: desfaz as que já criou e volta inteira para a de fábrica", function()
    local s = T.session({ only = "mare.binds", refuse = { ["ALT + Up"] = "erro" },
        files = { ["atalhos.lua"] = 'return { ["win.focus"] = "ALT" }\n' } })
    T.eq(nil, s:bind("ALT + Left"), "a metade criada foi desfeita")
    T.eq(4, count(s, "win.focus"))
    T.ok(s:bind("SUPER + Up"))
end)

T.t("até a de fábrica recusada: a ação entra em unbound, com aviso, e as outras seguem", function()
    local s = T.session({ only = "mare.binds", refuse = { ["SUPER + Q"] = "erro" } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq({ "win.close" }, s.binds.unbound)
    T.has(s.prints, "win.close ficou sem tecla")
    T.eq(stock_count(s) - 1, #s:live())
end)

-- ── atalhos.lua quebrado ──────────────────────────────────────────────
T.t("atalhos.lua com erro de sintaxe: valem os de fábrica, com aviso", function()
    local s = T.session({ only = "mare.binds", files = { ["atalhos.lua"] = 'return { ["win.close"] = ' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: atalhos.lua falhou")
    T.ok(s:bind("SUPER + Q"))
    T.eq(stock_count(s), #s:live())
end)

T.t("atalhos.lua que não devolve uma tabela: valem os de fábrica", function()
    for _, text in ipairs({ 'return "SUPER + X"\n', "return 42\n", "", "return nil\n" }) do
        local s = T.session({ only = "mare.binds", files = { ["atalhos.lua"] = text } })
        T.eq(nil, not s.ok and s.err or nil, "erro")
        T.eq(stock_count(s), #s:live())
        T.ok(s:bind("SUPER + Q"))
    end
end)

T.t("atalhos.lua roda sem acesso a nada: não alcança os, io, hl nem require", function()
    for _, text in ipairs({
        'os.execute("true")\nreturn { ["win.close"] = "SUPER + X" }\n',
        'io.open("/etc/passwd")\nreturn { ["win.close"] = "SUPER + X" }\n',
        'hl.exec_cmd("true")\nreturn { ["win.close"] = "SUPER + X" }\n',
        'require("os")\nreturn { ["win.close"] = "SUPER + X" }\n',
    }) do
        local s = T.session({ only = "mare.binds", files = { ["atalhos.lua"] = text } })
        T.eq(nil, not s.ok and s.err or nil, "erro")
        T.has(s.prints, "Mk: atalhos.lua falhou")
        T.eq({}, s.log.execs)
        T.ok(s:bind("SUPER + Q"), "valem os de fábrica")
    end
end)

T.t("atalhos.lua em bytecode não é aceito (só texto)", function()
    local s = T.session({ only = "mare.binds", files = { ["atalhos.lua"] = "\27Lua qualquer coisa" } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: atalhos.lua falhou")
    T.ok(s:bind("SUPER + Q"))
end)

-- ── apply / suspend / resume (o que os Ajustes chamam na sessão aberta) ─
T.t("apply() de novo tira os binds antigos e cria os novos, sem sobrar nenhum", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.apply({ ["win.close"] = "SUPER + X" })
    T.eq(stock_count(s), #s:live())
    T.ok(s:bind("SUPER + X"))
    T.eq(nil, s:bind("SUPER + Q"))
    s.binds.apply({})
    T.eq(stock_count(s), #s:live())
    T.ok(s:bind("SUPER + Q"))
    T.eq(nil, s:bind("SUPER + X"))
    T.eq(3 * stock_count(s), #s.log.binds, "três levas criadas, duas removidas")
end)

T.t("apply() com algo que não é tabela vale como sem trocas", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.apply("bobagem")
    T.eq(stock_count(s), #s:live())
    s.binds.apply(nil)
    T.eq(stock_count(s), #s:live())
end)

T.t("suspend() desliga todos os binds e arma um relógio para religar (8 s se não disser)", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.suspend()
    for _, b in ipairs(s:live()) do
        T.eq(false, b.enabled, b.keys)
    end
    T.eq(true, s.binds.suspended)
    T.eq(1, #s.log.timers)
    T.eq({ timeout = 8000, type = "oneshot" }, s.log.timers[1].opts)
    s.log.timers[1].fn() -- o relógio dispara
    for _, b in ipairs(s:live()) do
        T.eq(true, b.enabled, b.keys)
    end
    T.eq(false, s.binds.suspended)
end)

T.t("suspend(segundos) usa o tempo pedido", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.suspend(2.5)
    T.eq(2500, s.log.timers[1].opts.timeout)
    s.binds.suspend("3")
    T.eq(3000, s.log.timers[2].opts.timeout)
    T.eq(false, s.log.timers[1].enabled, "o relógio anterior é desarmado")
end)

T.t("resume() religa os binds e desarma o relógio", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.suspend(5)
    s.binds.resume()
    for _, b in ipairs(s:live()) do
        T.eq(true, b.enabled, b.keys)
    end
    T.eq(false, s.log.timers[1].enabled)
    T.eq(false, s.binds.suspended)
end)

T.t("apply() durante a suspensão devolve os atalhos ligados e desarma o relógio", function()
    local s = T.session({ only = "mare.binds" })
    s.binds.suspend(5)
    s.binds.apply({})
    for _, b in ipairs(s:live()) do
        T.eq(true, b.enabled, b.keys)
    end
    T.eq(false, s.binds.suspended)
    T.eq(false, s.log.timers[1].enabled)
end)

T.t("sem hl.timer não suspende: dá erro e os atalhos continuam ligados", function()
    local s = T.session({ only = "mare.binds", no_timer = true })
    local ok, err = pcall(s.binds.suspend, 5)
    T.eq(false, ok)
    T.has(tostring(err), "os atalhos não foram suspensos")
    for _, b in ipairs(s:live()) do
        T.eq(true, b.enabled, b.keys)
    end
    T.eq(false, s.binds.suspended)
end)

T.finish()
