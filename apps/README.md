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

## mkcalc

A calculadora (`mkcalc`): contas **exatas** (`0,1 + 0,2` dá `0,3`; o motor usa
frações, não ponto flutuante), parênteses, potência, raiz e o percentual de
calculadora de mesa (`200 + 10%` = `220`). O resultado aparece enquanto se
digita, tudo funciona pelo teclado, e as contas confirmadas ficam num
histórico (`Ctrl+H`). O motor (`mkcalc-motor servir`) fica de pé e responde
uma linha por pedido.

## mkcorte

O editor de cortes de vídeo (`mkcorte [arquivos]`): aparar as pontas, dividir,
remover trechos, juntar outros vídeos, reordenar, desfazer, ver o resultado e
exportar em MP4. Dois modos de exportar: **exato** (recodifica; o corte cai no
quadro marcado) e **rápido** (copia o vídeo como está, em segundos; o corte
cai no quadro-chave anterior). Nunca escreve por cima de um arquivo. A janela
mostra o vídeo pelo QtMultimedia (`qt6-multimedia`, `qt6-multimedia-ffmpeg`);
o motor (`mkcorte-motor`) monta e roda o ffmpeg. `?` mostra os atalhos.

## mkescrita

O editor de Markdown (`mkescrita [arquivo]`): uma coluna de texto, salvamento
automático depois que o documento tem nome (escrita atômica; se o arquivo
mudou por fora, para e pergunta), rascunho do que ainda não foi salvo
recuperado na abertura seguinte, pré-visualização ao lado (`Ctrl+Shift+P`),
sumário dos títulos, busca e substituição, exportar HTML. O conversor de
Markdown é do próprio motor (`mkescrita-motor`). `F1` mostra os atalhos.

## mkporto

O catálogo das conexões SSH (`mkporto`): os hosts do `~/.ssh/config` (com os
`Include`, com os valores que o `ssh -G` diz), se cada um responde (lê a
saudação do servidor, sem autenticar; host atrás de um salto é sondado pelo
salto) e em quantos milissegundos, conectar num terminal (`Enter`), testar se
a sua chave entra (`Ctrl+T`, sem pedir senha), copiar o comando, enviar uma
chave pública (`Ctrl+K`, `ssh-copy-id`), gerar uma chave nova (`Ctrl+G`,
`ssh-keygen` num terminal) e as chaves públicas com a impressão digital. **Não guarda nem pede senha, e nunca lê chave privada**: quem
autentica é o `ssh`, no terminal. Os hosts criados por ele ficam em
`~/.ssh/config.d/mkporto.conf`; na primeira gravação entra uma linha
`Include` no topo do `~/.ssh/config`, e o resto do arquivo fica como estava.
Os hosts do seu config aparecem, mas só você os edita. `F1` mostra os atalhos.

O sistema liga todos estes (do MkPacotes em diante) por `system/mk/apps-links`.

