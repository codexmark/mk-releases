# Contrato do mktunesd

O que o daemon do MkTunes (`bin/mktunesd`, código em `src/`) responde e o que a
janela (`qml/`) espera. Muda aqui primeiro; depois o código e os testes
(`tests/apps/mktunes-go.test.sh`, `tests/apps/mktunes-janela.test.sh`).

O daemon é um porte em Go do daemon em Kotlin do MkTunes: o protocolo é o
mesmo, campo por campo, e a janela não mudou.

## Linha de comando

| Chamada | O que faz | Saída |
|---|---|---|
| `mktunesd` | sobe o daemon e fica no ar | 0 ao encerrar (ou se já há um respondendo no socket); 1 se não deu para subir (sem mpv, guarda de versão mais nova) |
| `mktunesd --versao` | imprime a versão deste binário | 0 |
| `mktunesd --ping` | imprime a versão do daemon que responde no socket | 0 se alguém responde; 1 se ninguém |
| `mktunesd --encerra` | manda `{"cmd":"quit"}` e espera o daemon sair (ele guarda a sessão) | 0 se saiu ou não havia; 1 se continua no ar |

Qualquer outra opção: uso em stderr e saída 2. `SIGTERM`, `SIGINT` e `SIGHUP`
encerram como o `quit`: guardam a fila e a posição, fecham o socket e o mpv.

## Socket

`$XDG_RUNTIME_DIR/mktunes/mktunes.sock` (pasta `0700`, socket `0600`: só o
usuário). Uma linha JSON por mensagem em cada sentido, UTF-8, `\n` no fim.
Linhas da janela com mais de 8 MiB derrubam a conexão. Um arquivo de socket
deixado por um daemon que morreu é substituído; se alguém responde nele, o
segundo daemon sai.

Ao conectar, a janela recebe o estado de todas as áreas, nesta ordem: `now`,
`progress`, `search`, `home`, `library`, `downloads`, `radio`, `lyrics`,
`settings`, `volume`, `local`. Depois, cada mudança de estado (a mensagem
inteira de novo) e as mensagens avulsas. Para uma janela lenta, um estado novo
toma o lugar do antigo da mesma área, e só as 64 avulsas mais novas esperam.

### Faixa

As faixas vão e voltam inteiras: a janela devolve qualquer lista para tocar,
curtir ou baixar.

```json
{"id":"W40W7L1fhUM","provider":"YOUTUBE","title":"Até Quando","artist":"Natanzinho Lima",
 "durationMs":144000,"length":"2:24","artwork":"https://…","streamUrl":"https://…"}
```

`provider`: `YOUTUBE`, `RADIO`, `LOCAL` (e `DEEZER`, `AUDIUS`, `SOUNDCLOUD`,
`SPOTIFY`, aceitos de backups). `length` é `"AO VIVO"` nas rádios e `""` sem
duração. `streamUrl` só existe quando o catálogo já deu o link (rádio, arquivo
local). Uma faixa sem `id` ou com `provider` desconhecido é recusada.

Um **item** de feed é uma faixa com `"kind":"song"` e `subtitle`, ou
`{"kind":"album"|"playlist","id","title","subtitle","artwork","playlistId"}`,
ou `{"kind":"artist","id","title","subtitle","artwork"}`. Uma **prateleira** é
`{"title","items":[item…]}`.

## Estados (daemon → janela)

| `type` | Campos |
|---|---|
| `now` | `status` (`idle`, `loading`, `playing`, `paused`, `failed`), `track` (faixa, ou `{}`), `title`, `artist`, `artwork`, `live`, `liked`, `download` (`""`, `downloading`, `completed`), `upNext` (faixas), `repeat` (`off`, `all`, `one`), `stopAfterCurrent`, `speed`, `sleep` (`""`, `"14:59"`, `"fim da faixa"`) |
| `progress` | `position`, `duration` (segundos); a cada 0,5 s enquanto toca |
| `search` | `status` (`idle`, `loading`, `results`, `nothing`, `unavailable`), `query`, `filter` (`all`, `songs`, `albums`, `artists`, `playlists`), `tracks` (faixas), `items` (itens), `top` (item ou `null`), `sections` (`[{"title","filter","items"}]`), `recent` (textos), `suggestions` (textos) |
| `home` | `status` (`loading`, `ready`, `unavailable`), `chips` (`[{"title","token"}]`), `chip` (token escolhido ou `""`), `shelves` (prateleiras), `hasMore`, `loadingMore`, `moreFailed`, `recent` (faixas) |
| `library` | `liked` (faixas), `likedIds`, `playlists` (`[{"id","title","artwork","tracks"}]`), `saved` (`[{"kind","id","title","subtitle","artwork"}]`), `smart` (`[{"title","tracks"}]`: "No repeat", "Tocadas recentemente", "Redescobrir"), `period` (`day`, `week`, `month`), `stats` (`{}` ou `{"plays","minutes","artists":[{"name","plays"}],"tracks":[faixa+"plays"]}`) |
| `downloads` | `folder`, `jobs` (`[{"track","status":"queued"\|"downloading"\|"completed"\|"failed"\|"cancelled","percent","path","reason"}]`); no máximo 4 por segundo |
| `radio` | `status` (`idle`, `loading`, `ready`, `nothing`, `unavailable`), `query`, `stations` (faixas) |
| `lyrics` | `status` (`none`, `loading`, `synced`, `plain`, `unavailable`), `lines` (`[{"ms","text"}]`, em `synced`), `text` (em `plain`) |
| `settings` | `autoplay`, `normalizeVolume`, `audioQuality` (`high`, `economy`), `downloadWorkers`, `maxWorkers`, `backupsDir`, `version` |
| `volume` | `level` (0 a 100) |
| `local` | `status` (`idle`, `scanning`, `ready`, `empty`), `tracks` (faixas) |

## Avulsas (daemon → janela)

| `type` | Campos |
|---|---|
| `page` | `ref` (o de quem pediu), `status` (`ready`, `unavailable`); em `ready`, de álbum ou playlist: `kind`, `id`, `title`, `subtitle`, `owner`, `detail`, `artwork`, `playlistId`, `saved`, `tracks`; de artista: `kind":"artist"`, `id`, `title`, `subtitle`, `audience`, `radioPlaylistId`, `artwork`, `tracks`, `shelves` |
| `saved` | `id`, `saved` |
| `notice` | `text`: uma linha que a janela mostra por um instante. Os avisos da subida (migração, falta do `sqlite`) esperam a primeira janela mandar um comando |

## Comandos (janela → daemon)

`{"cmd": …}` mais os campos. Uma linha que não é JSON, sem `cmd`, com um
comando desconhecido ou com campos fora de alcance não muda nada.

| `cmd` | Campos | O que faz |
|---|---|---|
| `play` | `tracks`, `index`, `shuffle` | troca a fila e toca |
| `playCollection` | `playlistId` | toca um álbum ou playlist inteiro |
| `toggle`, `next`, `previous` | | transporte |
| `seek` | `seconds` | |
| `playUpNext`, `removeUpNext` | `index` | |
| `moveUpNext` | `from`, `to` | |
| `clearUpNext`, `shuffleUpNext`, `cycleRepeat`, `stopAfterCurrent` | | |
| `repeat` | `mode` (`off`, `all`, `one`) | o que o MPRIS usa |
| `speed` | `value` (0,5 a 2) | |
| `sleep` | `minutes`, ou `endOfTrack: true`, ou `cancel: true` | |
| `playNext`, `addToQueue` | `track` | avisam o que fizeram |
| `search` | `query`, `filter` | sem `filter`, repete o último; `all` busca tudo |
| `suggest` | `text` | sugestões depois de 300 ms sem digitar |
| `clearSearch`, `clearRecent` | | |
| `home`, `homeRetry`, `homeMore` | | |
| `homeChip` | `token` | o mesmo chip de novo o desmarca |
| `open` | `kind` (`album`, `playlist`, `artist`), `id`, `ref` | responde com `page` |
| `toggleSaved` | `id` | de uma coleção aberta há pouco |
| `library` | | refaz as listas inteligentes e o resumo |
| `statsPeriod` | `period` | |
| `like` | `track` | |
| `createPlaylist` | `title`, `track` (opcional) | |
| `addToPlaylist` | `id`, `track` | |
| `removePlaylistTrack` | `id`, `index` | |
| `movePlaylistTrack` | `id`, `from`, `to` | |
| `renamePlaylist` | `id`, `title` | |
| `deletePlaylist` | `id` | |
| `download` | `track` | |
| `downloadAll` | `tracks` | |
| `cancelDownload`, `retryDownload`, `removeDownload` | `id` | |
| `clearDownloadHistory` | | só falhas e cancelados |
| `radio` | `query` | |
| `local` | | procura as músicas da pasta de música |
| `volume` | `level` | |
| `setting` | `key` (`autoplay`, `normalizeVolume`, `downloadWorkers`, `audioQuality`), `value` | |
| `exportBackup` | | escreve `mktunes-backup-AAAA-MM-DD.json` em `backupsDir` |
| `importBackup` | `path` | junta o backup à biblioteca |
| `quit` | | interno (o lançador): guarda a sessão e encerra |

## Arquivos

Em `$XDG_DATA_HOME/mktunes` (`~/.local/share/mktunes`), só do usuário, escritos
num temporário e trocados de nome; todos trazem `"versao": 1`. Um arquivo
estragado é posto de lado (`.danificado-…`); um de versão mais nova impede o
daemon de subir, sem ser tocado.

| Arquivo | O que guarda |
|---|---|
| `biblioteca.json` | curtidas, playlists e coleções salvas |
| `preferencias.json` | ajustes e buscas recentes |
| `sessao.json` | a fila, o modo de repetir e a posição |
| `downloads.json` | o histórico de downloads |
| `historico.jsonl` | uma tocada por linha (os últimos 30 dias) |
| `mktunes.db.migrado` | o banco SQLite da versão em Kotlin, depois de migrado |

Fora dali: `$XDG_CACHE_HOME/mktunes/home-feed.json` (o último Início) e
`$XDG_STATE_HOME/mktunes/volume`. As músicas baixadas vão para
`<Música>/MkTunes/<artista>/<título> [id].m4a`; os backups, para `Documentos`
(as pastas do usuário vêm de `user-dirs.dirs`).

**Migração**: na subida, se `mktunes.db` existe, o daemon exporta as tabelas
com `sqlite3 -readonly -json`, junta à guarda (sem repetir) e renomeia o banco
para `mktunes.db.migrado`. Sem `sqlite3`: avisa o pacote (`sqlite`) e tenta na
próxima subida. O backup (exportar/importar) tem o mesmo formato da versão em
Kotlin (`"format":"mktunes-backup","version":1`).

## Mundo de fora

| Variável | Padrão | Para quê |
|---|---|---|
| `MKTUNES_MPV` | `mpv` | o tocador: `--config=no --load-scripts=no --idle=yes --no-video --no-terminal --audio-display=no --input-ipc-server=<pasta do socket>/mpv.sock --keep-open=no` |
| `MKTUNES_MPV_ARGS` | | opções a mais do mpv, separadas por espaço (`--ao=null`: sem som) |
| `MKTUNES_YTDLP` | `yt-dlp` | o áudio do YouTube: `-f <seletor> -g --no-playlist --no-warnings -- https://music.youtube.com/watch?v=<id>`; só ids `[A-Za-z0-9_-]{11}`, só endereços https de `googlevideo.com` e `youtube.com` |
| `MKTUNES_SQLITE3` | `sqlite3` | a migração |
| `MKTUNES_FFPROBE` | `ffprobe` | as etiquetas das músicas locais |
| `MKTUNES_LAUNCHER` | `mktunes` | o que o "Raise" do MPRIS roda |
| `MKTUNES_MUSIC_DIR` | `<Música>/MkTunes` | a pasta dos downloads |
| `MKTUNES_MPRIS` | | `0`: fica fora do barramento |
| `MKTUNES_INNERTUBE_URL`, `MKTUNES_LRCLIB_URL`, `MKTUNES_RADIO_URL` | os serviços de verdade | outro endereço (os testes) |

Nenhum comando passa por shell. Sem `yt-dlp`, a faixa fica `failed` e a janela
recebe um aviso com o pacote. Sem barramento de sessão
(`DBUS_SESSION_BUS_ADDRESS`), o daemon sobe sem MPRIS, sem erro.

## MPRIS

`org.mpris.MediaPlayer2.mktunes` em `/org/mpris/MediaPlayer2`, por um cliente
D-Bus próprio (`src/internal/dbus`). É mais um cliente deste protocolo: as
propriedades vêm de `now`, `progress` e `volume`; os métodos viram `toggle`,
`next`, `previous`, `seek`, `volume`, `repeat` e `shuffleUpNext`; `Raise` roda o
lançador. `PropertiesChanged` a cada mudança (menos `Position`) e `Seeked`
quando a posição salta.
