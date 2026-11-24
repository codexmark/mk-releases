---
camada: "03 Base do sistema"
tags:
  - mk
  - áudio
  - pipewire
fontes:
  - "system/packages/base.packages"
  - "shell/mare/Mare.qml"
  - "shell/mare/scripts/mare-cava"
  - "shell/mare/Sound.qml"
  - "config/hypr/mare/binds.lua"
---

# Áudio

O som é do PipeWire, com o WirePlumber como gerente de sessão. O Maré lê e muda o volume direto pela API do PipeWire e usa o `cava` para fazer a ilha ondular com a música.

## Para quê

Saída e microfone, volume por aplicativo, teclas de mídia e o efeito visual da ilha.

## Como funciona

- **Pacotes** (`base.packages`): `pipewire`, `pipewire-alsa`, `pipewire-pulse`, `pipewire-audio`, `wireplumber`, `alsa-utils`, `rtkit`, `pamixer`.
- **Volume pelas teclas**: os binds chamam `ipc call mare volume +5 | -5 | mute`. O `Mare.qml` muda `Pipewire.defaultAudioSink` (o serviço `Quickshell.Services.Pipewire`) e o OSD aparece na ilha ([[Notificações e OSD]]). O mudo do microfone é `wpctl set-mute @DEFAULT_AUDIO_SOURCE@ toggle`.
- **Painel de volume**: a saída padrão e um controle por app que está tocando (os fluxos de saída do PipeWire).
- **Página Som dos Ajustes**: por onde o som sai e por onde o microfone entra.
- **A onda da ilha**: `Sound.qml` roda `scripts/mare-cava`, que escreve uma config temporária do `cava` (28 faixas, 30 quadros por segundo, entrada `pipewire`, saída crua em ASCII) e imprime uma linha por quadro. Quadros repetidos (silêncio) são ignorados.
- **Player**: `playerctl` para tocar/pausar/próxima; o Maré mostra o player MPRIS em destaque ([[Painéis]]).

## Por quê assim

- **As teclas de mídia passam pelo Maré**, e não direto por `wpctl`, porque é ele quem mostra o OSD.
- **`cava` com `--pdeathsig TERM`** (`setpriv`): se o Maré morrer, o cava morre junto. E antes de subir, o script mata um cava de um Maré anterior, com um `pkill -f` cujo padrão (`cava -p <config>`) não casa com a linha de comando do próprio script ([[Convenções]]).
- **Como o PipeWire sobe**: o instalador não habilita nenhuma unidade de áudio. O repositório conta com o padrão dos pacotes (ativação por soquete na sessão do usuário), mas não o declara. **Não confirmado no código.**

## Onde está no código

- `system/packages/base.packages` (seção "Áudio (PipeWire)")
- `shell/mare/Mare.qml` (`changeVolume`, IPC `volume`), `VolumePanel.qml`, `Sound.qml`, `Media.qml`
- `shell/mare/scripts/mare-cava`

## Ligações

Depende de: [[Listas de pacotes]]. Usado por: [[Ilha]], [[Painéis]], [[Atalhos]], [[MkTunes]].
