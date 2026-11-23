# Apps do Mk

Apps feitos para o Mk que vêm com ele, já compilados. O `mk-update` troca esta
pasta junto com o resto do Mk, então uma versão nova de um app chega como
qualquer outra mudança do Mk.

## mktunes

O player de música do Mk: músicas, álbuns, rádios e letras, sem conta. A
janela é do Quickshell (tokens do Maré) e o núcleo é um daemon Java
(`mktunesd`) que toca pelo mpv e aparece na ilha pelo MPRIS.

O código mora no repositório do MkTunes; esta pasta é gerada lá, nunca
editada aqui:

```bash
desktop/bin/check-pack                 # testa o pacote inteiro
desktop/bin/pack ~/Work/mk/apps/mktunes
```

`apps/mktunes/VERSION` diz de qual commit do MkTunes veio. O sistema liga o
app por `system/mk/mktunes-links` (comando `mktunes`, atalho no menu e ícone,
todos apontando para cá) e precisa de `jre-openjdk-headless`, `mpv` e
`ffmpeg` (em `system/packages/desktop.packages`).
