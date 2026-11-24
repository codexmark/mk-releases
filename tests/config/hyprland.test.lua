-- config/hypr/hyprland.lua: carrega com e sem os arquivos dos Ajustes, na
-- ordem certa, aguenta um arquivo quebrado e desfaz um teste de monitores
-- que ficou sem resposta.
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

local function fns(log)
    local list = {}
    for _, c in ipairs(log.calls) do
        list[#list + 1] = c.fn
    end
    return table.concat(list, " ")
end

T.t("carrega sem erro sem nenhum arquivo dos Ajustes", function()
    local s = T.session()
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("", s.prints, "avisos")
end)

T.t("pede ao Hyprland cada assunto: ambiente, monitor, aparência, movimento, teclado, atalhos, regras, início", function()
    local s = T.session()
    T.ok(s.log.env.GTK_THEME, "env")
    T.ok(#s.log.monitors >= 1, "monitor")
    T.ok(s.log.config.general and s.log.config.decoration, "aparência")
    T.ok(#s.log.animations >= 1, "animações")
    T.ok(s.log.config.input, "teclado")
    T.ok(#s.log.binds >= 1, "atalhos")
    T.ok(#s.log.rules >= 1, "regras")
    T.ok(s.log.events["hyprland.start"], "início da sessão")
end)

T.t("todo arquivo de mare/ é carregado (nenhum módulo esquecido no hyprland.lua)", function()
    local s = T.session()
    local p = io.popen("ls '" .. T.repo .. "/config/hypr/mare'")
    for file in p:lines() do
        local name = file:match("^(.*)%.lua$")
        T.ok(name, "em mare/ só há .lua: " .. file)
        T.ok(package.loaded["mare." .. name] ~= nil, "mare." .. name .. " não foi carregado")
    end
    p:close()
end)

T.t("os caminhos saem de onde o hyprland.lua está, não de ~/.config", function()
    local s = T.session()
    local paths = package.loaded["mare.paths"]
    T.eq(s.root, paths.root)
    T.eq(s.root .. "hypr/", paths.hypr)
    T.eq(s.root .. "shell", paths.shell)
    T.eq(s.root .. "ghostty/config", paths.ghostty)
end)

T.t("com ajustes.lua e monitores.lua: os dois rodam, depois dos módulos do Mk, nesta ordem", function()
    local s = T.session({ files = {
        ["ajustes.lua"] = 'hl.config({ general = { gaps_in = 2 } })\nhl.env("MK_AJUSTES", "sim")\n',
        ["monitores.lua"] = 'hl.monitor({ output = "DP-1", mode = "2560x1440@144", position = "0x0", scale = 1.25 })\n',
    } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("", s.prints, "avisos")
    T.eq(2, s.log.config.general.gaps_in, "o ajuste vale sobre o de fábrica")
    T.eq(12, s.log.config.general.gaps_out, "o que o ajuste não disse fica")
    T.eq("DP-1", s.log.monitors[#s.log.monitors].output, "a regra dos Ajustes vem por último")
    T.eq("", s.log.monitors[1].output, "a regra geral do Mk vem antes")
    T.ok(fns(s.log):match("on config env monitor$"), "ordem: " .. fns(s.log))
end)

T.t("com atalhos.lua: os atalhos nascem já trocados", function()
    local s = T.session({ files = { ["atalhos.lua"] = 'return { ["win.close"] = "SUPER + X" }\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.ok(s:bind("SUPER + X"), "SUPER + X")
    T.eq(nil, s:bind("SUPER + Q"), "SUPER + Q")
end)

T.t("o atalhos.lua é só dado: o hyprland.lua não o roda como configuração", function()
    local s = T.session({ files = { ["atalhos.lua"] = 'hl.env("MK_INVASOR", "1")\nreturn {}\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(nil, s.log.env.MK_INVASOR)
end)

T.t("com os três presentes carrega sem erro e sem aviso", function()
    local s = T.session({ files = {
        ["ajustes.lua"] = 'hl.config({ input = { kb_layout = "us" } })\n',
        ["monitores.lua"] = 'hl.monitor({ output = "eDP-1", mode = "preferred", position = "auto", scale = 1.5 })\n',
        ["atalhos.lua"] = 'return { ["mare.calendar"] = "" }\n',
    } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("", s.prints, "avisos")
    T.eq("us", s.log.config.input.kb_layout)
end)

T.t("um ajustes.lua quebrado não derruba a sessão: avisa e segue para o monitores.lua", function()
    local s = T.session({ files = {
        ["ajustes.lua"] = "hl.config({ general = ",
        ["monitores.lua"] = 'hl.monitor({ output = "DP-1", mode = "preferred", position = "auto", scale = 1 })\n',
    } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: ajustes.lua falhou")
    T.eq("DP-1", s.log.monitors[#s.log.monitors].output)
end)

T.t("um ajustes.lua que dá erro no meio (função que não existe) também só avisa", function()
    local s = T.session({ files = { ["ajustes.lua"] = 'hl.nao_existe("x")\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: ajustes.lua falhou")
end)

T.t("um monitores.lua quebrado só avisa", function()
    local s = T.session({ files = { ["monitores.lua"] = "isto não é lua" } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: monitores.lua falhou")
end)

T.t("monitors.lua e user.lua, ao lado do hyprland.lua, carregam por último e prevalecem", function()
    local s = T.session({ files = {
        ["ajustes.lua"] = 'hl.config({ general = { gaps_in = 2 } })\n',
        ["hypr/monitors.lua"] = 'hl.monitor({ output = "HDMI-A-1", mode = "1920x1080", position = "0x0", scale = 1 })\n',
        ["hypr/user.lua"] = 'hl.config({ general = { gaps_in = 9 } })\n',
    } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(9, s.log.config.general.gaps_in)
    T.eq("HDMI-A-1", s.log.monitors[#s.log.monitors].output)
    T.ok(fns(s.log):match("monitor config$"), "ordem: " .. fns(s.log))
end)

T.t("um user.lua quebrado só avisa", function()
    local s = T.session({ files = { ["hypr/user.lua"] = 'error("ops")\n' } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.has(s.prints, "Mk: user.lua falhou")
end)

-- ── a sobra de um teste de monitores (monitores.lua.anterior) ─────────
local novo = 'hl.monitor({ output = "DP-1", mode = "3840x2160@60", position = "0x0", scale = 2 })\n'
local antigo = 'hl.monitor({ output = "DP-1", mode = "1920x1080@60", position = "0x0", scale = 1 })\n'

T.t("sobra de outra sessão (a máquina caiu no meio do teste): o monitores.lua anterior volta e é o que vale", function()
    local s = T.session({
        env = { HYPRLAND_INSTANCE_SIGNATURE = "sessao-de-agora" },
        files = { ["monitores.lua"] = novo, ["monitores.lua.anterior"] = "-- mk:teste sessao-que-caiu\n" .. antigo },
    })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(antigo, s:read("monitores.lua"))
    T.eq(false, s:exists("monitores.lua.anterior"), "a sobra some")
    T.eq("1920x1080@60", s.log.monitors[#s.log.monitors].mode)
    T.eq(2, #s.log.monitors, "só a regra geral e a que voltou")
end)

T.t("sobra desta mesma sessão (o teste ainda corre): nada é desfeito", function()
    local s = T.session({
        env = { HYPRLAND_INSTANCE_SIGNATURE = "sessao-de-agora" },
        files = { ["monitores.lua"] = novo, ["monitores.lua.anterior"] = "-- mk:teste sessao-de-agora\n" .. antigo },
    })
    T.eq(novo, s:read("monitores.lua"))
    T.eq("-- mk:teste sessao-de-agora\n" .. antigo, s:read("monitores.lua.anterior"))
    T.eq("3840x2160@60", s.log.monitors[#s.log.monitors].mode)
end)

T.t("sobra de outra sessão e não havia monitores.lua antes do teste: o arquivo do teste some", function()
    local s = T.session({
        env = { HYPRLAND_INSTANCE_SIGNATURE = "sessao-de-agora" },
        files = { ["monitores.lua"] = novo, ["monitores.lua.anterior"] = "-- mk:teste sessao-que-caiu\n" },
    })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(false, s:exists("monitores.lua"))
    T.eq(false, s:exists("monitores.lua.anterior"))
    T.eq(1, #s.log.monitors, "só a regra geral do Mk")
end)

T.t("sobra com várias linhas: o anterior volta inteiro, byte a byte", function()
    local varias = "-- gerado pelos Ajustes\n" .. antigo .. 'hl.monitor({ output = "eDP-1", disabled = true })\n'
    local s = T.session({
        env = { HYPRLAND_INSTANCE_SIGNATURE = "b" },
        files = { ["monitores.lua"] = novo, ["monitores.lua.anterior"] = "-- mk:teste a\n" .. varias },
    })
    T.eq(varias, s:read("monitores.lua"))
    T.eq(3, #s.log.monitors)
end)

T.t("sobra sem a marca do teste (arquivo estranho): tratada como de outra sessão, o conteúdo depois da 1ª linha volta", function()
    local s = T.session({
        env = { HYPRLAND_INSTANCE_SIGNATURE = "b" },
        files = { ["monitores.lua"] = novo, ["monitores.lua.anterior"] = "qualquer coisa\n" .. antigo },
    })
    T.eq(antigo, s:read("monitores.lua"))
    T.eq(false, s:exists("monitores.lua.anterior"))
end)

T.t("sem a sobra, o monitores.lua não é tocado", function()
    local s = T.session({ env = { HYPRLAND_INSTANCE_SIGNATURE = "b" }, files = { ["monitores.lua"] = novo } })
    T.eq(novo, s:read("monitores.lua"))
    T.eq("3840x2160@60", s.log.monitors[#s.log.monitors].mode)
end)

T.t("a marca é a que o scripts/mare-monitors grava", function()
    T.has(T.slurp(T.repo .. "/shell/mare/scripts/mare-monitors"), "printf -- '-- mk:teste %s\\n' \"$session\"")
end)

T.finish()
