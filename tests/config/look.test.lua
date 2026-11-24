-- config/hypr/mare/look.lua: a aparência Folha e as janelas que o Hyprland
-- não deve abrir sozinho (novidades da versão, pedido de doação).
local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")

T.t("carrega sem erro", function()
    local s = T.session({ only = "mare.look" })
    T.eq(nil, not s.ok and s.err or nil, "erro")
end)

T.t("sem a janela de novidades a cada versão nova (ecosystem.no_update_news)", function()
    local s = T.session({ only = "mare.look" })
    T.eq(true, s.log.config.ecosystem.no_update_news)
end)

T.t("sem o pedido de doação (ecosystem.no_donation_nag)", function()
    local s = T.session({ only = "mare.look" })
    T.eq(true, s.log.config.ecosystem.no_donation_nag)
end)

T.t("direção Folha: cantos retos, linha de 1 px, sem sombra nem desfoque", function()
    local c = T.session({ only = "mare.look" }).log.config
    T.eq(0, c.decoration.rounding)
    T.eq(1, c.general.border_size)
    T.eq(false, c.decoration.shadow.enabled)
    T.eq(false, c.decoration.blur.enabled)
end)

T.t("as bordas são o osso da paleta (cdc4ba), mais forte na janela ativa", function()
    local g = T.session({ only = "mare.look" }).log.config.general
    local on, off = g["col.active_border"]:match("^rgba%(cdc4ba(%x%x)%)$"), g["col.inactive_border"]:match("^rgba%(cdc4ba(%x%x)%)$")
    T.ok(on and off, "rgba(cdc4baXX)")
    T.ok(tonumber(on, 16) > tonumber(off, 16), "ativa mais forte")
end)

T.t("sem o logo, o splash e o papel de parede do Hyprland; fundo quase preto", function()
    local m = T.session({ only = "mare.look" }).log.config.misc
    T.eq(true, m.disable_hyprland_logo)
    T.eq(true, m.disable_splash_rendering)
    T.eq(0, m.force_default_wallpaper)
    T.eq(0x0B0B0B, m.background_color)
end)

T.t("um Maré novo pode assumir o bloqueio se o antigo cair, e mexer acorda a tela", function()
    local m = T.session({ only = "mare.look" }).log.config.misc
    T.eq(true, m.allow_session_lock_restore)
    T.eq(true, m.key_press_enables_dpms)
    T.eq(true, m.mouse_move_enables_dpms)
end)

T.t("só fala de seções conhecidas", function()
    local known = { general = true, decoration = true, dwindle = true, ecosystem = true, misc = true, xwayland = true }
    for section in pairs(T.session({ only = "mare.look" }).log.config) do
        T.ok(known[section], "seção inesperada: " .. tostring(section))
    end
end)

T.finish()
