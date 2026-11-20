-- O que sobe junto com a sessão. Cada linha é um processo independente.
-- Tudo é chamado pelo caminho da instalação do Mk, para a sessão não
-- depender de ~/.config/hypr nem de ~/.config/quickshell (que podem ser de
-- outro desktop). A saída do Maré vai para ~/.cache/mk/mare.log.
local paths = require("mare.paths")

hl.on("hyprland.start", function()
    -- O Maré, no tamanho escolhido nos Ajustes (scripts/mare-start lê e aplica).
    hl.exec_cmd("bash '" .. paths.shell .. "/scripts/mare-start'")
    -- Os tempos da página Energia dos Ajustes, quando você já mexeu neles.
    local idle = paths.root .. "hypridle.conf"
    if not package.searchpath("hypridle", paths.root .. "?.conf") then
        idle = paths.hypr .. "hypridle.conf"
    end
    hl.exec_cmd("hypridle -c '" .. idle .. "'")
    hl.exec_cmd("systemctl --user start hyprpolkitagent")
    hl.exec_cmd("gnome-keyring-daemon --start --components=secrets,pkcs11")
    hl.exec_cmd("wl-clip-persist --clipboard regular")
    hl.exec_cmd("wl-paste --type text --watch cliphist store")
    hl.exec_cmd("wl-paste --type image --watch cliphist store")
    hl.exec_cmd("gsettings set org.gnome.desktop.interface color-scheme prefer-dark")
    hl.exec_cmd("gsettings set org.gnome.desktop.interface gtk-theme Adwaita-dark")
    hl.exec_cmd("udiskie --automount --no-tray")
end)
