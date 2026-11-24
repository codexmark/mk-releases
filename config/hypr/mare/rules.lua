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

-- Os apps do Mk (apps/): utilitários que se abrem por cima do trabalho, no
-- tamanho deles. Lado a lado com outras janelas viram uma coluna estreita e
-- as tabelas não cabem. O MkCorte e o MkEscrita são janelas de trabalho:
-- entram lado a lado como qualquer outra.
hl.window_rule({
    name = "float-mkpulso",
    match = { title = "^(MkPulso|MkPacotes|MkInstalar|MkComprimir|MkCalc)$" },
    float = true,
    center = true,
})

hl.window_rule({
    name = "no-maximize-requests",
    match = { class = ".*" },
    suppress_event = "maximize",
})
