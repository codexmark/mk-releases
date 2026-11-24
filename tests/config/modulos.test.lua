-- config/hypr/mare/{display,input,rules,paths}.lua: os módulos pequenos. A
-- regra geral de monitor, o teclado de fábrica, as regras de janela e os
-- caminhos da instalação.
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

-- ── display.lua ───────────────────────────────────────────────────────
T.t("display: uma regra para qualquer tela, escala 1 (o Hyprland não dobra sozinho numa 1366x768)", function()
    local s = T.session({ only = "mare.display" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq({ { output = "", mode = "preferred", position = "auto", scale = 1 } }, s.log.monitors)
end)

-- ── input.lua ─────────────────────────────────────────────────────────
T.t("input: teclado português do Brasil de fábrica, sem variante", function()
    local s = T.session({ only = "mare.input" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("br", s.log.config.input.kb_layout)
    T.eq("", s.log.config.input.kb_variant)
end)

T.t("input: touchpad com toque, rolagem natural e pausa ao digitar", function()
    local t = T.session({ only = "mare.input" }).log.config.input.touchpad
    T.eq({ natural_scroll = true, tap_to_click = true, disable_while_typing = true }, t)
end)

T.t("input: só fala de input (o kb_options da pessoa fica em paz)", function()
    local c = T.session({ only = "mare.input" }).log.config
    for section in pairs(c) do
        T.eq("input", section)
    end
    T.eq(nil, c.input.kb_options)
end)

-- ── rules.lua ─────────────────────────────────────────────────────────
T.t("rules: cada regra tem nome único e um match", function()
    local s = T.session({ only = "mare.rules" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.ok(#s.log.rules >= 1)
    local seen = {}
    for _, r in ipairs(s.log.rules) do
        T.ok(type(r.name) == "string" and r.name ~= "", "nome")
        T.ok(not seen[r.name], "nome repetido: " .. r.name)
        seen[r.name] = true
        T.ok(type(r.match) == "table" and next(r.match), r.name .. ": match")
    end
end)

local function rule(s, name)
    for _, r in ipairs(s.log.rules) do
        if r.name == name then
            return r
        end
    end
    error("sem a regra " .. name)
end

T.t("rules: utilitários (som, rede, bluetooth, calculadora) flutuam no centro", function()
    local r = rule(T.session({ only = "mare.rules" }), "float-utilities")
    T.eq(true, r.float)
    T.eq(true, r.center)
    for _, class in ipairs({ "pavucontrol", "nm-connection-editor", "blueman-manager", "org.gnome.Calculator" }) do
        T.has(r.match.class, class)
    end
end)

T.t("rules: diálogos de arquivo flutuam no centro, em inglês e em português", function()
    local r = rule(T.session({ only = "mare.rules" }), "float-file-dialogs")
    T.eq(true, r.float)
    T.eq(true, r.center)
    T.has(r.match.title, "Open File")
    T.has(r.match.title, "Abrir arquivo")
    T.has(r.match.title, "Salvar arquivo")
end)

T.t("rules: o MkPulso flutua no centro, no tamanho dele (lado a lado ele não cabe)", function()
    -- Visto na VM: aberto num workspace com dois terminais, virou uma coluna
    -- estreita e a tabela ficou cortada. É um utilitário: flutua.
    local r = rule(T.session({ only = "mare.rules" }), "float-mkpulso")
    T.eq(true, r.float)
    T.eq(true, r.center)
    for _, app in ipairs({ "MkPulso", "MkPacotes", "MkInstalar", "MkComprimir", "MkCalc" }) do
        T.has(r.match.title, app)
    end
end)

T.t("rules: nenhuma janela se maximiza sozinha", function()
    local r = rule(T.session({ only = "mare.rules" }), "no-maximize-requests")
    T.eq(".*", r.match.class)
    T.eq("maximize", r.suppress_event)
end)

-- ── paths.lua ─────────────────────────────────────────────────────────
T.t("paths: a raiz é a pasta acima de hypr/, com shell e ghostty ao lado", function()
    local s = T.session({ only = "mare.paths" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    local p = s.module
    T.eq(s.root, p.root)
    T.eq(s.root .. "hypr/", p.hypr)
    T.eq(s.root .. "shell", p.shell)
    T.eq(s.root .. "ghostty/config", p.ghostty)
end)

T.t("paths: o log vai para $XDG_CACHE_HOME/mk, ou ~/.cache/mk sem ela", function()
    local a = T.session({ only = "mare.paths", env = { XDG_CACHE_HOME = "/cache/x", HOME = "/home/ana" } })
    T.eq("/cache/x/mk", a.module.log)
    local b = T.session({ only = "mare.paths", env = { HOME = "/home/ana" } })
    T.eq("/home/ana/.cache/mk", b.module.log)
end)

T.t("paths: não pede nada ao Hyprland", function()
    T.eq({}, T.session({ only = "mare.paths" }).log.calls)
end)

T.finish()
