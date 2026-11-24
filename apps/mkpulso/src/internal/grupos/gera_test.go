package grupos

import (
	"os"
	"testing"

	"mkpulso/internal/coleta/falso"
)

const (
	usuario = "/user.slice/user-1000.slice/user@1000.service"
	sessao  = "/user.slice/user-1000.slice/session-3.scope"
)

// A árvore de testdata/desktop: uma máquina com um pouco de cada caso da
// regra de agrupamento. Para refazer os arquivos depois de mudar a lista:
//
//	MKPULSO_GERA=1 go test ./internal/grupos -run TestGeraTestdata
var arvore = []falso.Proc{
	{PID: 1, PPID: 0, Comm: "systemd", Cmd: []string{"/sbin/init"}, Inicio: 1, RSS: 12000, Cgroup: "/init.scope"},
	{PID: 2, PPID: 0, Comm: "kthreadd", Inicio: 1},
	{PID: 30, PPID: 2, Comm: "kworker/0:1", Estado: "I", Inicio: 2},
	{PID: 300, PPID: 1, Comm: "sshd", Cmd: []string{"sshd: /usr/bin/sshd -D [listener]"}, Inicio: 300, RSS: 5400, Cgroup: "/system.slice/sshd.service"},
	{PID: 310, PPID: 1, Comm: "dbus-broker-lau", Cmd: []string{"/usr/bin/dbus-broker-launch", "--scope", "system"}, Inicio: 310, RSS: 3000, Cgroup: "/system.slice/dbus-broker.service"},
	{PID: 311, PPID: 310, Comm: "dbus-broker", Cmd: []string{"dbus-broker"}, Inicio: 311, RSS: 2000, Cgroup: "/system.slice/dbus-broker.service"},
	{PID: 320, PPID: 1, Comm: "dockerd", Cmd: []string{"/usr/bin/dockerd"}, Inicio: 320, RSS: 90000, Cgroup: "/system.slice/docker.service"},
	{PID: 321, PPID: 320, Comm: "carga", Cmd: []string{"/carga"}, Inicio: 321, RSS: 100, Cgroup: "/system.slice/docker.service/carga"},
	{PID: 800, PPID: 1, Comm: "systemd", Cmd: []string{"/usr/lib/systemd/systemd", "--user"}, UID: 1000, Inicio: 800, RSS: 9000, Cgroup: usuario + "/init.scope"},
	{PID: 810, PPID: 800, Comm: "dbus-broker-lau", Cmd: []string{"/usr/bin/dbus-broker-launch", "--scope", "user"}, UID: 1000, Inicio: 810, RSS: 2500, Cgroup: usuario + "/session.slice/dbus-broker.service"},
	{PID: 820, PPID: 800, Comm: "pipewire", Cmd: []string{"/usr/bin/pipewire"}, UID: 1000, Inicio: 820, RSS: 15000, Cgroup: usuario + "/session.slice/pipewire.service"},
	{PID: 830, PPID: 800, Comm: "Hyprland", Cmd: []string{"Hyprland"}, UID: 1000, Inicio: 830, RSS: 150000, Cgroup: usuario + "/session.slice/wayland-wm@hyprland.desktop.service"},
	{PID: 1234, PPID: 800, Comm: "firefox", Cmd: []string{"/usr/lib/firefox/firefox"}, UID: 1000, Inicio: 1234, RSS: 400000, Cgroup: usuario + "/app.slice/app-firefox-1234.scope"},
	{PID: 1240, PPID: 1234, Comm: "Isolated Web Co", Cmd: []string{"/usr/lib/firefox/firefox", "-contentproc"}, UID: 1000, Inicio: 1240, RSS: 200000, Cgroup: usuario + "/app.slice/app-firefox-1234.scope"},
	{PID: 1250, PPID: 800, Comm: "firefox", Cmd: []string{"/usr/lib/firefox/firefox", "--private-window"}, UID: 1000, Inicio: 1250, RSS: 100000, Cgroup: usuario + "/app.slice/app-firefox-1250.scope"},
	{PID: 1300, PPID: 800, Comm: "claude-desktop", Cmd: []string{"/usr/bin/claude-desktop"}, UID: 1000, Inicio: 1300, RSS: 300000, Cgroup: usuario + `/app.slice/app-graphical.slice/app-claude\x2ddesktop@autostart.service`},
	{PID: 1400, PPID: 800, Comm: "nautilus", Cmd: []string{"/usr/bin/nautilus"}, UID: 1000, Inicio: 1400, RSS: 80000, Cgroup: usuario + "/app.slice/app-gnome-org.gnome.Nautilus-1400.scope"},
	{PID: 1500, PPID: 800, Comm: "meu-app", Cmd: []string{"/opt/meu-app/meu-app"}, UID: 1000, Inicio: 1500, RSS: 5000, Cgroup: usuario + "/app.slice/app-meu-app-1500.scope"},
	{PID: 1700, PPID: 800, Comm: "guarda", Cmd: []string{"/usr/bin/guarda"}, UID: 1000, Inicio: 1700, RSS: 900, Cgroup: usuario + `/app.slice/app-ryoku\x2dsession\x2dguard.slice/ryoku-session-guard@9.service`},
	{PID: 1800, PPID: 800, Comm: "emacs", Cmd: []string{"/usr/bin/emacs", "--fg-daemon"}, UID: 1000, Inicio: 1800, RSS: 70000, Cgroup: usuario + "/app.slice/emacs.service"},
	{PID: 2000, PPID: 1, Comm: "login", Cmd: []string{"login", "--", "mark"}, Inicio: 2000, RSS: 3000, Cgroup: sessao},
	{PID: 2001, PPID: 2000, Comm: "bash", Cmd: []string{"-bash"}, UID: 1000, Inicio: 2001, RSS: 4000, Cgroup: sessao},
	{PID: 2100, PPID: 1, Comm: "kitty", Cmd: []string{"kitty"}, UID: 1000, Inicio: 2100, RSS: 60000, Cgroup: sessao},
	{PID: 2101, PPID: 2100, Comm: "bash", Cmd: []string{"/bin/bash"}, UID: 1000, Inicio: 2101, RSS: 4000, Cgroup: sessao},
	{PID: 2102, PPID: 2101, Comm: "vim", Cmd: []string{"vim", "notas.txt"}, UID: 1000, Inicio: 2102, RSS: 9000, Cgroup: sessao},
	{PID: 2200, PPID: 1, Comm: "kitty", Cmd: []string{"kitty", "--class", "outra"}, UID: 1000, Inicio: 2200, RSS: 60000, Cgroup: sessao},
	{PID: 2201, PPID: 2200, Comm: "htop", Cmd: []string{"htop"}, UID: 1000, Inicio: 2201, RSS: 3000, Cgroup: sessao},
	{PID: 3100, PPID: 800, Comm: "solto", Cmd: []string{"/usr/bin/solto"}, UID: 1000, Inicio: 3100, RSS: 100, Cgroup: usuario + "/app.slice"},
	{PID: 3200, PPID: 1, Comm: "orfao", Cmd: []string{"/usr/bin/orfao"}, Inicio: 3200, RSS: 100, Cgroup: "/"},
	{PID: 3201, PPID: 1, Comm: "semcgroup", Cmd: []string{"/usr/bin/semcgroup"}, Inicio: 3201, RSS: 100},
	{PID: 3300, PPID: 1, Comm: "nginx", Cmd: []string{"nginx: master process /usr/bin/nginx"}, Inicio: 3300, RSS: 2000, Cgroup: "/docker/abc123"},
	{PID: 3301, PPID: 3300, Comm: "nginx", Cmd: []string{"nginx: worker process"}, Inicio: 3301, RSS: 1500, Cgroup: "/docker/abc123"},
}

var atalhosDaArvore = map[string]string{
	"apps/firefox.desktop":                           "[Desktop Entry]\nType=Application\nName=Firefox do sistema\nIcon=firefox-do-sistema\nExec=firefox %u\n",
	"home/.local/share/applications/firefox.desktop": "# o da home ganha do sistema\n[Desktop Entry]\nName=Firefox\nName[pt]=Raposa\nName[pt_BR]=Navegador Firefox\nName[de]=Feuerfuchs\nIcon=firefox\n\n[Desktop Action new-window]\nName=Nova janela\nIcon=outro\n",
	"apps/kitty.desktop":                             "[Desktop Entry]\nName=kitty\nGenericName=Terminal\nIcon=kitty\n",
	"xdg/applications/org.gnome.Nautilus.desktop":    "[Desktop Entry]\nName=Files\nName[pt]=Ficheiros\nIcon=org.gnome.Nautilus\n",
	"apps/emacs.desktop":                             "[Desktop Entry]\nName=Emacs\nIcon=emacs\n",
	"apps/vazio.desktop":                             "",
	"etc/passwd":                                     "root:x:0:0::/root:/usr/bin/bash\nmark:x:1000:1000:Mark:/home/mark:/usr/bin/bash\n",
}

func TestGeraTestdata(t *testing.T) {
	if os.Getenv("MKPULSO_GERA") == "" {
		t.Skip("só roda com MKPULSO_GERA=1, para refazer testdata/desktop")
	}
	raiz := "testdata/desktop"
	if err := os.RemoveAll(raiz); err != nil {
		t.Fatal(err)
	}
	for _, p := range arvore {
		p.Grava(t, raiz)
	}
	for caminho, texto := range atalhosDaArvore {
		falso.Escreve(t, raiz, caminho, texto)
	}
}
