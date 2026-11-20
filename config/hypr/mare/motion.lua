-- Movimento: a mesma curva do shell e do perfil (cubic-bezier(.215, .61, .355, 1)),
-- rápida no início e assentando no fim. Janelas entram de 95% para 100%; workspaces
-- deslizam. A unidade de speed do Hyprland é 100 ms: 2.4 = 240 ms.
hl.config({ animations = { enabled = true } })

hl.curve("mare", { type = "bezier", points = { { 0.215, 0.61 }, { 0.355, 1 } } })
hl.curve("mareOut", { type = "bezier", points = { { 0.4, 0 }, { 1, 1 } } })

hl.animation({ leaf = "global", enabled = true, speed = 2.4, bezier = "mare" })
hl.animation({ leaf = "windows", enabled = true, speed = 2.4, bezier = "mare" })
hl.animation({ leaf = "windowsIn", enabled = true, speed = 2.0, bezier = "mare", style = "popin 95%" })
hl.animation({ leaf = "windowsOut", enabled = true, speed = 1.6, bezier = "mareOut", style = "popin 94%" })
hl.animation({ leaf = "border", enabled = true, speed = 2.0, bezier = "mare" })
hl.animation({ leaf = "fade", enabled = true, speed = 1.7, bezier = "mare" })
hl.animation({ leaf = "layers", enabled = true, speed = 2.0, bezier = "mare", style = "fade" })
hl.animation({ leaf = "workspaces", enabled = true, speed = 2.8, bezier = "mare", style = "slide" })
hl.animation({ leaf = "specialWorkspace", enabled = true, speed = 2.4, bezier = "mare", style = "slidevert" })
