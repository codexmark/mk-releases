-- config/hypr/mare/env.lua: as variáveis da sessão, e o SSH_AUTH_SOCK só
-- quando o soquete do gcr-ssh-agent está ligado para todos os usuários.
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

local socket = "/etc/systemd/user/sockets.target.wants/gcr-ssh-agent.socket"

T.t("as variáveis da sessão do Mk", function()
    local s = T.session({ only = "mare.env" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq({
        GTK_THEME = "Mk", XCURSOR_THEME = "Adwaita", XCURSOR_SIZE = "24", HYPRCURSOR_SIZE = "24",
        QT_QPA_PLATFORMTHEME = "qt6ct", QT_QPA_PLATFORM = "wayland;xcb",
        ELECTRON_OZONE_PLATFORM_HINT = "auto", TERMINAL = "ghostty",
    }, s.log.env)
end)

T.t("sem o soquete do gcr ligado: não mexe no SSH_AUTH_SOCK", function()
    local s = T.session({ only = "mare.env", env = { XDG_RUNTIME_DIR = "/run/user/1000" } })
    T.eq(nil, s.log.env.SSH_AUTH_SOCK)
end)

T.t("com o soquete do gcr ligado: SSH_AUTH_SOCK aponta para $XDG_RUNTIME_DIR/gcr/ssh", function()
    local s = T.session({ only = "mare.env", sys = { [socket] = "" }, env = { XDG_RUNTIME_DIR = "/run/user/1234" } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq("/run/user/1234/gcr/ssh", s.log.env.SSH_AUTH_SOCK)
end)

T.t("com o soquete mas sem XDG_RUNTIME_DIR: não inventa um caminho", function()
    local s = T.session({ only = "mare.env", sys = { [socket] = "" }, env = { XDG_RUNTIME_DIR = false } })
    T.eq(nil, not s.ok and s.err or nil, "erro")
    T.eq(nil, s.log.env.SSH_AUTH_SOCK)
end)

T.t("um soquete do gcr só do usuário (~/.config/systemd) não conta: é o de todos, o que o Mk liga", function()
    local s = T.session({ only = "mare.env", sys = { ["/home/x/.config/systemd/user/sockets.target.wants/gcr-ssh-agent.socket"] = "" } })
    T.eq(nil, s.log.env.SSH_AUTH_SOCK)
end)

T.t("o tema, o qt6ct e o terminal vêm com o Mk (pacotes e arquivos existem)", function()
    T.ok(T.slurp(T.repo .. "/system/mk/look/themes/Mk/index.theme"), "tema GTK Mk")
    local pkgs = "\n" .. T.slurp(T.repo .. "/system/packages/desktop.packages") .. T.slurp(T.repo .. "/system/packages/dev.packages")
    for _, p in ipairs({ "qt6ct", "ghostty", "gcr-4" }) do
        T.ok(pkgs:find("\n" .. p .. "\n", 1, true), p .. " nas listas de pacotes")
    end
end)

T.finish()
