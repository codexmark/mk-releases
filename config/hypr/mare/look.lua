-- Aparência das janelas na direção Folha: cantos retos, uma linha fina em
-- tom de osso (26% na janela ativa, 13% nas outras), fundo quase preto e quente.
hl.config({
    general = {
        gaps_in = 6,
        gaps_out = 12,
        border_size = 1,
        layout = "dwindle",
        resize_on_border = true,
        ["col.active_border"] = "rgba(cdc4ba42)",
        ["col.inactive_border"] = "rgba(cdc4ba21)",
    },
    decoration = {
        rounding = 0,
        active_opacity = 1,
        inactive_opacity = 1,
        shadow = { enabled = false },
        blur = { enabled = false },
    },
    dwindle = {
        preserve_split = true,
    },
    -- Sem a janela "Hyprland updated to..." a cada versão nova (ela toma o
    -- foco no primeiro login) e sem o pedido de doação.
    ecosystem = {
        no_update_news = true,
        no_donation_nag = true,
    },
    misc = {
        -- Se o Maré cair com a tela bloqueada, um Maré novo pode assumir o bloqueio.
        allow_session_lock_restore = true,
        disable_hyprland_logo = true,
        disable_splash_rendering = true,
        force_default_wallpaper = 0,
        background_color = 0x0B0B0B,
        focus_on_activate = true,
        key_press_enables_dpms = true,
        mouse_move_enables_dpms = true,
    },
    xwayland = {
        force_zero_scaling = true,
    },
})
