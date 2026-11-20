-- Atalhos. Os painéis do shell abrem pelo IPC do Maré: qs -p <shell> ipc call mare <...>
local paths = require("mare.paths")
local mod = "SUPER"
local mare = "qs -p '" .. paths.shell .. "' ipc call mare "

local function shell(args)
    return hl.dsp.exec_cmd(mare .. args)
end

-- Emergência: funcionam mesmo se o Maré não abrir (terminal e sair da sessão).
hl.bind(mod .. " + SHIFT + RETURN", hl.dsp.exec_cmd("ghostty --config-default-files=false --config-file='?" .. paths.ghostty .. "' || kitty || foot || alacritty"))
hl.bind(mod .. " + SHIFT + ESCAPE", hl.dsp.exit())

-- Shell: launcher, Alt+Tab e painéis da ilha.
-- Um toque no Super sozinho abre o launcher: o bind é de soltar a tecla, e o
-- Hyprland não o dispara se outra tecla foi apertada junto (Super+Enter etc.),
-- então os atalhos com Super continuam rodando normalmente.
hl.bind(mod .. " + SUPER_L", shell("toggle launcher"), { release = true })
hl.bind(mod .. " + SUPER_R", shell("toggle launcher"), { release = true })
hl.bind(mod .. " + SPACE", shell("toggle launcher"))
hl.bind("ALT + TAB", shell("switcher next"))
hl.bind("ALT + SHIFT + TAB", shell("switcher prev"))
hl.bind(mod .. " + P", shell("toggle project"))
hl.bind(mod .. " + C", shell("toggle calendar"))
hl.bind(mod .. " + N", shell("toggle notifications"))
hl.bind(mod .. " + SHIFT + N", shell("dnd"))
hl.bind(mod .. " + ESCAPE", shell("toggle session"))
hl.bind(mod .. " + W", shell("toggle wallpaper"))
hl.bind(mod .. " + SHIFT + V", shell("toggle clipboard"))
hl.bind(mod .. " + COMMA", shell("toggle settings"))
hl.bind(mod .. " + L", shell("lock"))

-- Programas no diretório do projeto em foco
hl.bind(mod .. " + RETURN", shell("here ''"))
hl.bind(mod .. " + G", shell("here lazygit"))
hl.bind(mod .. " + D", shell("here lazydocker"))
hl.bind(mod .. " + E", shell("here yazi"))
hl.bind(mod .. " + B", hl.dsp.exec_cmd("chromium"))

-- Janelas
hl.bind(mod .. " + Q", hl.dsp.window.close())
hl.bind(mod .. " + F", hl.dsp.window.fullscreen())
hl.bind(mod .. " + V", hl.dsp.window.float({ action = "toggle" }))
hl.bind(mod .. " + Left", hl.dsp.focus({ direction = "left" }))
hl.bind(mod .. " + Right", hl.dsp.focus({ direction = "right" }))
hl.bind(mod .. " + Up", hl.dsp.focus({ direction = "up" }))
hl.bind(mod .. " + Down", hl.dsp.focus({ direction = "down" }))
hl.bind(mod .. " + SHIFT + Left", hl.dsp.window.move({ direction = "left" }))
hl.bind(mod .. " + SHIFT + Right", hl.dsp.window.move({ direction = "right" }))
hl.bind(mod .. " + SHIFT + Up", hl.dsp.window.move({ direction = "up" }))
hl.bind(mod .. " + SHIFT + Down", hl.dsp.window.move({ direction = "down" }))
hl.bind(mod .. " + CTRL + Left", hl.dsp.window.resize({ x = -40, y = 0, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Right", hl.dsp.window.resize({ x = 40, y = 0, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Up", hl.dsp.window.resize({ x = 0, y = -40, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Down", hl.dsp.window.resize({ x = 0, y = 40, relative = true }), { repeating = true })
hl.bind(mod .. " + mouse:272", hl.dsp.window.drag(), { mouse = true })
hl.bind(mod .. " + mouse:273", hl.dsp.window.resize(), { mouse = true })

-- Workspaces = projetos. Entrar num workspace novo e vazio abre o seletor de projeto.
for i = 1, 9 do
    hl.bind(mod .. " + " .. i, hl.dsp.focus({ workspace = i }))
    hl.bind(mod .. " + SHIFT + " .. i, hl.dsp.window.move({ workspace = i }))
end
hl.bind(mod .. " + TAB", hl.dsp.focus({ workspace = "previous" }))
hl.bind(mod .. " + mouse_down", hl.dsp.focus({ workspace = "e+1" }))
hl.bind(mod .. " + mouse_up", hl.dsp.focus({ workspace = "e-1" }))

-- Captura de tela: área para a área de transferência / tela inteira para arquivo
hl.bind("PRINT", hl.dsp.exec_cmd("grim -g \"$(slurp)\" - | wl-copy"))
hl.bind("SHIFT + PRINT", hl.dsp.exec_cmd("grim \"$(xdg-user-dir PICTURES)/$(date +%F_%H-%M-%S).png\""))

-- Teclas de mídia: passam pelo Maré, que mostra o OSD na linha
hl.bind("XF86AudioRaiseVolume", shell("volume +5"), { locked = true, repeating = true })
hl.bind("XF86AudioLowerVolume", shell("volume -5"), { locked = true, repeating = true })
hl.bind("XF86AudioMute", shell("volume mute"), { locked = true })
hl.bind("XF86AudioMicMute", hl.dsp.exec_cmd("wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle"), { locked = true })
hl.bind("XF86MonBrightnessUp", shell("brightness +5"), { locked = true, repeating = true })
hl.bind("XF86MonBrightnessDown", shell("brightness -5"), { locked = true, repeating = true })
hl.bind("XF86AudioPlay", hl.dsp.exec_cmd("playerctl play-pause"), { locked = true })
hl.bind("XF86AudioNext", hl.dsp.exec_cmd("playerctl next"), { locked = true })
hl.bind("XF86AudioPrev", hl.dsp.exec_cmd("playerctl previous"), { locked = true })
