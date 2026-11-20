-- Regras de janela: diálogos e utilitários flutuam e centralizam.
hl.window_rule({
    name = "float-utilities",
    match = { class = "(pavucontrol|nm-connection-editor|blueman-manager|xdg-desktop-portal-gtk|org.gnome.Calculator)" },
    float = true,
    center = true,
})

hl.window_rule({
    name = "float-file-dialogs",
    match = { title = "(Open File|Save File|Save As|Choose Files|Open Folder|Abrir arquivo|Salvar arquivo)" },
    float = true,
    center = true,
})

hl.window_rule({
    name = "no-maximize-requests",
    match = { class = ".*" },
    suppress_event = "maximize",
})
