import QtQuick
import QtQuick.Controls

// O menu ⋯ de uma faixa, igual em toda tela: fila, curtir, baixar, playlists.
MkMenu {
    id: menu
    property var track: ({})
    // Numa playlist local: oferece tirar a faixa dela.
    property string playlistId: ""
    property int playlistIndex: -1
    signal newPlaylist(var track)

    readonly property bool live: track.provider === "RADIO"

    function openFor(t, anchorItem, fromPlaylist, index) {
        menu.track = t;
        menu.playlistId = fromPlaylist || "";
        menu.playlistIndex = index === undefined ? -1 : index;
        menu.popup(anchorItem, 0, anchorItem ? anchorItem.height : 0);
    }

    MenuEntry { text: "Tocar a seguir"; glyph: Theme.icon.next; enabled: !menu.live; onTriggered: daemon.send({ cmd: "playNext", track: menu.track }) }
    MenuEntry { text: "Adicionar à fila"; glyph: Theme.icon.queue; enabled: !menu.live; onTriggered: daemon.send({ cmd: "addToQueue", track: menu.track }) }
    MenuEntry {
        text: daemon.isLiked(menu.track.id) ? "Descurtir" : "Curtir"
        glyph: daemon.isLiked(menu.track.id) ? Theme.icon.heart : Theme.icon.heartOutline
        onTriggered: daemon.send({ cmd: "like", track: menu.track })
    }
    MenuEntry { text: "Baixar"; glyph: Theme.icon.download; enabled: !menu.live; onTriggered: daemon.send({ cmd: "download", track: menu.track }) }
    MkMenu {
        id: playlistMenu
        title: "Adicionar à playlist"
        enabled: !menu.live
        MenuEntry { text: "Nova playlist…"; glyph: Theme.icon.plus; onTriggered: menu.newPlaylist(menu.track) }
        Instantiator {
            model: daemon.library.playlists
            delegate: MenuEntry {
                required property var modelData
                text: modelData.title
                glyph: Theme.icon.playlistAdd
                onTriggered: daemon.send({ cmd: "addToPlaylist", id: modelData.id, track: menu.track })
            }
            onObjectAdded: (index, object) => playlistMenu.insertItem(index + 1, object)
            onObjectRemoved: (index, object) => playlistMenu.removeItem(object)
        }
    }
    MenuEntry {
        visible: menu.playlistId !== ""
        height: visible ? implicitHeight : 0
        text: "Tirar desta playlist"
        glyph: Theme.icon.remove
        onTriggered: daemon.send({ cmd: "removePlaylistTrack", id: menu.playlistId, index: menu.playlistIndex })
    }
}
