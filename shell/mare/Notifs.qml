pragma Singleton
import QtQuick
import Quickshell
import Quickshell.Services.Notifications

// Servidor de notificações do Maré. Cada notificação nova aparece como um
// aviso que nasce da linha; todas ficam no histórico (sino da barra).
Singleton {
    id: root

    property bool dnd: false
    property var toastItem: null
    // Hora de chegada por id: o protocolo não traz o horário.
    property var arrivedAt: ({})

    readonly property var list: server.trackedNotifications.values.slice().reverse()
    readonly property int count: list.length

    function clearAll() {
        server.trackedNotifications.values.slice().forEach(n => n.dismiss());
    }

    // Ação padrão (clicar no aviso), se o app oferecer uma; senão, só fecha.
    function activate(n) {
        const action = n.actions.find(a => a.identifier === "default") || null;
        if (action)
            action.invoke();
        else
            n.dismiss();
    }

    function ago(n) {
        const t = arrivedAt[n.id];
        if (!t)
            return "";
        const s = Math.floor((Date.now() - t) / 1000);
        if (s < 60)
            return "agora";
        if (s < 3600)
            return Math.floor(s / 60) + " min";
        return Math.floor(s / 3600) + " h";
    }

    NotificationServer {
        id: server
        keepOnReload: true
        persistenceSupported: true
        bodySupported: true
        actionsSupported: true
        imageSupported: true

        onNotification: n => {
            n.tracked = true;
            const times = Object.assign({}, root.arrivedAt);
            times[n.id] = Date.now();
            root.arrivedAt = times;
            if (!root.dnd || n.urgency === NotificationUrgency.Critical) {
                root.toastItem = n;
                const ms = Settings.get("notify.duration") * 1000;
                Mare.flash("toast", n.urgency === NotificationUrgency.Critical ? ms * 2 : ms);
            }
        }
    }
}
