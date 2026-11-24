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

## mkpulso

O monitor de sistema do Mk: processos agrupados por app e por serviço,
desempenho, a ficha da máquina, o que sobe no login e os serviços do systemd.
Ao contrário do MkTunes, **o código mora aqui**:

| Pasta | O que é |
|---|---|
| `mkpulso/src/` | o coletor, em Go (só a biblioteca padrão): lê `/proc`, `/sys`, os cgroups e o systemd, e executa as ações (sinais, prioridade, serviços, inicialização) |
| `mkpulso/qml/` | a janela, do Quickshell, com os tokens do Maré; só mostra o que o coletor diz |
| `mkpulso/bin/mkpulso` | o lançador (uma janela só; `mkpulso servicos` abre direto numa área) |
| `mkpulso/bin/mkpulso-coletor` | o coletor compilado (estático), que vai junto com o Mk |
| `mkpulso/CONTRATO.md` | o que o coletor responde e a janela espera: muda lá primeiro |

Uma máquina instalada não compila nada: o binário vem pronto. Depois de mexer
em `src/`, recompile e faça o commit do binário junto:

```bash
apps/mkpulso/pack
```

O `pack` formata, roda o `go vet` e o `go test`, compila e anota ao lado do
binário de que código ele saiu; `tests/run` reprova um binário que não é o do
código que está no repositório. Os testes do coletor usam `/proc`, `/sys` e
comandos de mentira: nenhum manda sinal nem mexe em serviço de verdade.

Para ver a janela sem o coletor, com respostas gravadas:

```bash
MKPULSO_EXEMPLO=tests/apps/mkpulso-exemplo qs -p apps/mkpulso/qml
```

O sistema liga o app por `system/mk/mkpulso-links` (comando `mkpulso`, atalho
no menu e ícone). Não pede pacote além do que o Mk já tem.

## mkpacotes

O gerenciador de pacotes do Mk, no mesmo molde do MkPulso (coletor em Go em
`mkpacotes/src/`, janela em `mkpacotes/qml/`, contrato em
`mkpacotes/CONTRATO.md`, `mkpacotes/pack` para compilar): o que está
instalado, com filtros (pedidos por você, do AUR, órfãos, com atualização),
busca nos repositórios e no AUR, a ficha de cada pacote e o histórico do
`pacman.log`. Toda ação mostra antes o plano (o que entra, o que sai, os
comandos) e só roda depois da confirmação, com a senha pedida pelo polkit (o
Maré). Recusa remover o que derruba o sistema. "Atualizar tudo" é do
`mk atualizar`: o app só o abre num terminal.

## mkutil

Dois utilitários com um motor em Go (`mkutil/src/`, `mkutil/bin/mkutil-motor`):

- **MkInstalar** (`mkinstalar [arquivos]`): instala um AppImage (copia para
  `~/Aplicativos`, cria o atalho de menu; o arquivo nunca é executado para ser
  inspecionado), um pacote do Arch (`pacman -U` pelo polkit) ou um arquivo
  compactado (recusa caminhos que saem da pasta e bombas), e desinstala o que
  instalou.
- **MkComprimir** (`mkcomprimir [arquivos]`): encolhe vídeos e imagens com o
  ffmpeg, com predefinições (inclusive "caber em N MB"), a estimativa antes de
  começar, progresso, cancelar e fila. Nunca sobrescreve o original.

O sistema liga os três por `system/mk/apps-links`.

