-- Variáveis de ambiente da sessão.
-- Os apps GTK na paleta Folha (tema em system/mk/look). Por variável de
-- ambiente, vale só dentro do Mk.
hl.env("GTK_THEME", "Mk")
hl.env("XCURSOR_THEME", "Adwaita")
hl.env("XCURSOR_SIZE", "24")
hl.env("HYPRCURSOR_SIZE", "24")
hl.env("QT_QPA_PLATFORMTHEME", "qt6ct")
hl.env("QT_QPA_PLATFORM", "wayland;xcb")
hl.env("ELECTRON_OZONE_PLATFORM_HINT", "auto")
hl.env("TERMINAL", "ghostty")
