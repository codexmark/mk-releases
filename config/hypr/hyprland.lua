-- Hyprland do Mk. Cada arquivo em mare/ cuida de um assunto.
-- Instale em ~/.config/hypr/ (o diretório inteiro); user.lua é seu e carrega por último.

-- Os módulos ficam ao lado deste arquivo, onde quer que ele esteja.
local here = debug.getinfo(1, "S").source:match("^@(.*/)") or "./"
package.path = here .. "?.lua;" .. package.path

require("mare.env")
require("mare.display")
require("mare.look")
require("mare.motion")
require("mare.input")
require("mare.binds")
require("mare.rules")
require("mare.autostart")

-- O que foi escolhido nos Ajustes (Super + ,): <raiz>/ajustes.lua, gerado
-- pelo Maré, fora da pasta hypr que a instalação substitui.
local ajustes = require("mare.paths").root .. "ajustes.lua"
if package.searchpath("ajustes", (ajustes:gsub("ajustes%.lua$", "?.lua"))) then
    local ok, err = pcall(dofile, ajustes)
    if not ok then
        print("Mk: ajustes.lua falhou: " .. tostring(err))
    end
end

-- Monitores e ajustes pessoais: opcionais, nunca sobrescritos pelo Mk.
for _, mod in ipairs({ "monitors", "user" }) do
    if package.searchpath(mod, package.path) then
        local ok, err = pcall(require, mod)
        if not ok then
            print("Mk: " .. mod .. ".lua falhou: " .. tostring(err))
        end
    end
end
