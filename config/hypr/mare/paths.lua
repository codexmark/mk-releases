-- Onde o Mk está instalado. A config do Hyprland fica em <raiz>/hypr e o
-- shell Maré em <raiz>/shell (normalmente ~/.config/mk). O Maré não vai
-- para ~/.config/quickshell: aquela pasta pode pertencer a outro desktop, que
-- a recria e apaga o que não reconhece.
local hypr = debug.getinfo(1, "S").source:match("^@(.*/)mare/") or "./"
local root = hypr:match("^(.*/)hypr/$") or (hypr .. "../")

return {
    root = root,
    hypr = hypr,
    shell = root .. "shell",
    ghostty = root .. "ghostty/config",
    log = (os.getenv("XDG_CACHE_HOME") or ((os.getenv("HOME") or "") .. "/.cache")) .. "/mk",
}
