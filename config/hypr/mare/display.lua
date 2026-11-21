-- Monitores: escala 1 por padrão. Sem regra, o Hyprland escolhe a escala
-- sozinho e pode dobrar tudo (2x) numa tela 1366x768, deixando a barra e os
-- painéis do Maré enormes e cortados. A escala de cada tela se escolhe em
-- Ajustes > Monitores (<raiz>/monitores.lua, regras pelo nome da saída); o
-- que o painel não tem vai em monitors.lua, ao lado de hyprland.lua (carrega
-- por último e prevalece).
hl.monitor({ output = "", mode = "preferred", position = "auto", scale = 1 })
