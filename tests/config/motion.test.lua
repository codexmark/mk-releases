-- config/hypr/mare/motion.lua: as animações de fábrica, iguais à lista que a
-- Velocidade dos Ajustes repete (shell/mare/SettingsSchema.js, `animations`).
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

-- A lista do SettingsSchema.js: ["folha", velocidade, "curva", "estilo"].
local function schema()
    local js = assert(T.slurp(T.repo .. "/shell/mare/SettingsSchema.js"))
    local block = assert(js:match("var animations = %[(.-)\n%];"), "não achei var animations no SettingsSchema.js")
    local list = {}
    for leaf, speed, bezier, style in block:gmatch('%["([^"]+)",%s*([%d.]+),%s*"([^"]*)",%s*"([^"]*)"%]') do
        list[#list + 1] = { leaf = leaf, speed = tonumber(speed), bezier = bezier, style = style ~= "" and style or nil }
    end
    return list
end

T.t("carrega sem erro, com as animações ligadas", function()
    local s = T.session({ only = "mare.motion" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(true, s.log.config.animations.enabled)
end)

T.t("as curvas do Maré: cubic-bezier(.215, .61, .355, 1) e a de saída", function()
    local s = T.session({ only = "mare.motion" })
    T.eq({ type = "bezier", points = { { 0.215, 0.61 }, { 0.355, 1 } } }, s.log.curves.mare)
    T.eq({ type = "bezier", points = { { 0.4, 0 }, { 1, 1 } } }, s.log.curves.mareOut)
end)

T.t("toda animação usa uma curva que foi definida antes dela", function()
    local s = T.session({ only = "mare.motion" })
    local defined = {}
    for _, c in ipairs(s.log.calls) do
        if c.fn == "curve" then
            defined[c.args[1]] = true
        elseif c.fn == "animation" then
            T.ok(defined[c.args[1].bezier], c.args[1].leaf .. ": curva " .. tostring(c.args[1].bezier))
        end
    end
end)

T.t("cada folha aparece uma vez, ligada, com velocidade positiva", function()
    local s = T.session({ only = "mare.motion" })
    local seen = {}
    for _, a in ipairs(s.log.animations) do
        T.ok(not seen[a.leaf], "folha repetida: " .. a.leaf)
        seen[a.leaf] = true
        T.eq(true, a.enabled, a.leaf)
        T.ok(type(a.speed) == "number" and a.speed > 0, a.leaf .. ": speed")
    end
end)

T.t("a lista é a mesma do SettingsSchema.js (folha, velocidade, curva e estilo, na mesma ordem)", function()
    local s = T.session({ only = "mare.motion" })
    local got = {}
    for _, a in ipairs(s.log.animations) do
        got[#got + 1] = { leaf = a.leaf, speed = a.speed, bezier = a.bezier, style = a.style }
    end
    local want = schema()
    T.ok(#want >= 5, "li a lista do SettingsSchema.js")
    T.eq(want, got)
end)

T.finish()
