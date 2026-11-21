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
-- Chaves SSH com senha lembradas pelo chaveiro: o agente é o gcr-ssh-agent
-- (o gnome-keyring não traz mais o seu). Só quando o soquete está ligado
-- para todos os usuários, que é o que o instalador do Mk faz.
do
    local on = io.open("/etc/systemd/user/sockets.target.wants/gcr-ssh-agent.socket")
    local run = os.getenv("XDG_RUNTIME_DIR")
    if on and run then
        on:close()
        hl.env("SSH_AUTH_SOCK", run .. "/gcr/ssh")
    end
end
