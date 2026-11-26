-- config/hypr/mare/autostart.lua: o que sobe com a sessão, pelo caminho da
-- instalação do Mk, e o que não sobe (o agente de polkit é o próprio Maré).
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

local function started(opts)
    opts = opts or {}
    opts.only = "mare.autostart"
    local s = T.session(opts)
    T.eq(nil, not s.ok and s.err or nil, "erro")
    return s
end

T.t("nada roda ao carregar a configuração: só no início da sessão (hyprland.start)", function()
    local s = started()
    T.eq({}, s.log.execs)
    T.eq(1, #s.log.events["hyprland.start"])
    s:fire("hyprland.start")
    T.ok(#s.log.execs > 0)
end)

T.t("um recarregar da configuração não sobe tudo de novo: nenhum hl.exec_cmd fora do evento", function()
    local s = T.session()
    T.eq({}, s.log.execs)
end)

T.t("sobe o Maré pelo scripts/mare-start da instalação", function()
    local s = started()
    s:fire("hyprland.start")
    T.eq("bash '" .. s.root .. "shell/scripts/mare-start'", s.log.execs[1])
    T.ok(T.slurp(T.repo .. "/shell/mare/scripts/mare-start"), "o script existe")
end)

T.t("sem hyprpolkitagent (nem outro agente de polkit): o Maré é o agente da sessão", function()
    local s = started()
    s:fire("hyprland.start")
    local all = table.concat(s.log.execs, "\n"):lower()
    T.hasnt(all, "polkit")
    T.hasnt(all, "hyprpolkitagent")
    T.ok(T.slurp(T.repo .. "/shell/mare/PolkitDialog.qml"), "o agente do Maré existe")
end)

T.t("hypridle com a configuração de fábrica enquanto ninguém mexeu em Ajustes › Energia", function()
    local s = started()
    s:fire("hyprland.start")
    T.has(table.concat(s.log.execs, "\n"), "hypridle -c '" .. s.root .. "hypr/hypridle.conf'")
    T.ok(s:exists("hypr/hypridle.conf"))
end)

T.t("hypridle com o <raiz>/hypridle.conf quando os Ajustes já geraram um", function()
    local s = started({ files = { ["hypridle.conf"] = "general {\n}\n" } })
    s:fire("hyprland.start")
    local all = table.concat(s.log.execs, "\n")
    T.has(all, "hypridle -c '" .. s.root .. "hypridle.conf'")
    T.hasnt(all, "hypr/hypridle.conf")
end)

T.t("um só hypridle e um só Maré", function()
    local s = started()
    s:fire("hyprland.start")
    local idle, mare = 0, 0
    for _, cmd in ipairs(s.log.execs) do
        if cmd:match("^hypridle ") then idle = idle + 1 end
        if cmd:find("mare-start", 1, true) then mare = mare + 1 end
    end
    T.eq(1, idle)
    T.eq(1, mare)
end)

T.t("a área de transferência é guardada (texto e imagem) e sobrevive ao app fechar", function()
    local s = started()
    s:fire("hyprland.start")
    local all = "\n" .. table.concat(s.log.execs, "\n") .. "\n"
    T.has(all, "\nwl-clip-persist --clipboard regular\n")
    T.has(all, "\nwl-paste --type text --watch cliphist store\n")
    T.has(all, "\nwl-paste --type image --watch cliphist store\n")
end)

T.t("aplica o visual dos apps (mare-look) e monta pendrives (udiskie, sem bandeja)", function()
    local s = started()
    s:fire("hyprland.start")
    local all = "\n" .. table.concat(s.log.execs, "\n") .. "\n"
    T.has(all, "\nbash '" .. s.root .. "shell/scripts/mare-look'\n")
    T.has(all, "\nudiskie --automount --no-tray\n")
    T.ok(T.slurp(T.repo .. "/shell/mare/scripts/mare-look"), "o script existe")
end)

T.t("nada depende de ~/.config/hypr nem de ~/.config/quickshell", function()
    local s = started()
    s:fire("hyprland.start")
    local all = table.concat(s.log.execs, "\n")
    T.hasnt(all, ".config/hypr")
    T.hasnt(all, ".config/quickshell")
    T.hasnt(all, "~")
end)

T.t("todo programa chamado vem num pacote das listas do Mk", function()
    local s = started()
    s:fire("hyprland.start")
    local pkgs = "\n"
    for _, list in ipairs({ "base", "desktop", "dev", "apps" }) do
        pkgs = pkgs .. T.slurp(T.repo .. "/system/packages/" .. list .. ".packages") .. "\n"
    end
    local owner = { ["wl-paste"] = "wl-clipboard", bash = "base" }
    for _, cmd in ipairs(s.log.execs) do
        local bin = cmd:match("^(%S+)")
        local pkg = owner[bin] or bin
        T.ok(pkgs:find("\n" .. pkg .. "\n", 1, true), bin .. " (pacote " .. pkg .. ") não está nas listas")
    end
end)


T.t("a aparência vai também para o systemd e o D-Bus (scripts/mare-env): apps ativados por lá pegam o tema", function()
    local s = started()
    s:fire("hyprland.start")
    T.has(table.concat(s.log.execs, "\n"), "bash '" .. s.root .. "shell/scripts/mare-env'")
    T.ok(T.slurp(T.repo .. "/shell/mare/scripts/mare-env"), "o script existe")
end)

T.finish()
