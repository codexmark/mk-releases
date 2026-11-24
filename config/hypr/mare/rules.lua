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

-- O MkPulso (apps/mkpulso), o monitor do sistema: um utilitário que se abre
-- por cima do trabalho, no tamanho dele. Lado a lado com outras janelas ele
-- vira uma coluna estreita e a tabela não cabe.
hl.window_rule({
    name = "float-mkpulso",
    match = { title = "^MkPulso$" },
    float = true,
    center = true,
})

hl.window_rule({
    name = "no-maximize-requests",
    match = { class = ".*" },
    suppress_event = "maximize",
})
