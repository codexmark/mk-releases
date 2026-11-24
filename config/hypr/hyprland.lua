-- Hyprland do Mk. Cada arquivo em mare/ cuida de um assunto.
-- Fica em ~/.config/mk/hypr/ (o diretório inteiro); user.lua é seu e carrega por último.

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

-- O que foi escolhido nos Ajustes (Super + ,), em arquivos gerados pelo Maré
-- na raiz (<raiz>/ajustes.lua...), fora da pasta hypr que a instalação
-- substitui. Nesta ordem: ajustes gerais, monitores. O atalhos.lua não entra
-- aqui: é só dado, e o mare/binds.lua já o leu, sem acesso a nada.
local root = require("mare.paths").root

-- Uma troca de monitores que ficou sem resposta (a máquina desligou ou o
-- Hyprland caiu durante a contagem dos Ajustes): o arquivo anterior volta.
-- A primeira linha da sobra diz que sessão começou o teste; se é esta, o
-- teste ainda está correndo e o scripts/mare-monitors cuida dele.
do
    local path = root .. "monitores.lua"
    local left = io.open(path .. ".anterior")
    if left then
        local mark = left:read("l") or ""
        local before = left:read("a") or ""
        left:close()
        if mark ~= "-- mk:teste " .. (os.getenv("HYPRLAND_INSTANCE_SIGNATURE") or "") then
            if before == "" then
                os.remove(path)
            else
                local out = io.open(path, "w")
                if out then
                    out:write(before)
                    out:close()
                end
            end
            os.remove(path .. ".anterior")
        end
    end
end

for _, name in ipairs({ "ajustes", "monitores" }) do
    if package.searchpath(name, root .. "?.lua") then
        local ok, err = pcall(dofile, root .. name .. ".lua")
        if not ok then
            print("Mk: " .. name .. ".lua falhou: " .. tostring(err))
        end
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
