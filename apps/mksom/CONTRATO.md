# MkSom: o contrato entre o motor e a janela

O MkSom é o equalizador da saída de som e o teste de microfone do Mk. Usa só
o que vem no PipeWire e no WirePlumber (pacotes `pipewire`, `pipewire-pulse`,
`wireplumber`): o equalizador é o módulo `libpipewire-module-filter-chain` com
os filtros `builtin` (`bq_lowshelf`, `bq_peaking`, `bq_highshelf` e o
`linear`, do pré-ganho). Nada de EasyEffects, LADSPA ou LV2.

Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mksom-motor`), que escreve o
  arquivo do filter-chain e a unidade do systemd, liga e desliga, aplica os
  ganhos ao vivo, lê o `pw-dump`, mexe na entrada padrão e mede o microfone.
  É todo testado com pastas e comandos de mentira: nenhum teste roda o
  PipeWire, toca ou grava som;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que mostra o que o
  motor diz e pede o que a pessoa mexeu. A lógica que ela faz sozinha (a
  curva, os passos dos controles, o aviso de distorção, o ritmo das
  aplicações, o diagnóstico do microfone, os textos) mora em `qml/Som.js`,
  testado no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. Um subcomando
  que falha escreve `{"erro": "frase em português"}` e sai com código
  diferente de 0. A janela mostra a frase.
- O que o motor não soube é `null`, nunca `""`.
- Todo comando externo vem por variável de ambiente (é por elas que os testes
  trocam o sistema), roda como lista de argumentos (nunca por shell) e tem
  tempo limite:

  | Variável | Padrão | Para quê |
  |---|---|---|
  | `MKSOM_SYSTEMCTL` | `systemctl` | `--user daemon-reload`, `enable --now`, `disable --now` |
  | `MKSOM_PW_DUMP` | `pw-dump` | o estado de verdade (JSON) |
  | `MKSOM_PW_CLI` | `pw-cli` | `set-param`, o ajuste ao vivo |
  | `MKSOM_WPCTL` | `wpctl` | `set-default`, `set-volume`, `set-mute` |
  | `MKSOM_PIPEWIRE` | `pipewire` | o que vai no `ExecStart` da unidade (o motor não o roda) |
  | `MKSOM_PW_RECORD` | `pw-record` | o medidor e a gravação de teste |
  | `MKSOM_PW_PLAY` | `pw-play` | ouvir a gravação de teste |

- As pastas também: `MKSOM_CONFIG` (padrão `~/.config/mk/mksom`) e
  `MKSOM_UNIDADES` (padrão `~/.config/systemd/user`). A gravação de teste fica
  em `$XDG_RUNTIME_DIR/mksom/` (memória, não disco).
- **O motor nunca escreve em `~/.config/pipewire/`** (recusa gravar ali mesmo
  que uma variável aponte para lá) e não mexe na configuração do PipeWire nem
  do WirePlumber da pessoa: o equalizador é um processo à parte.
- Só vira argumento de comando o que passou pela validação: ids são números
  (até 7 algarismos) e, no microfone, têm de ser de uma entrada que o
  `pw-dump` mostra; nomes de predefinição não vão para comando nenhum.
- No motor, o que vem depois de `--` é sempre nome, nunca opção.

## Os arquivos do app

Em `~/.config/mk/mksom/` (a pasta com `700`, os arquivos com `600`, sempre
gravados inteiros e de forma atômica: temporário na mesma pasta e `rename`):

- `estado.json`: `{"ligado": bool, "predefinicao": "nome" | "", "ganhos":
  [10 números], "pre": número, "saida_anterior": "node.name" | ""}`. Ilegível
  ou ausente vale plano, desligado.
- `predefinicoes.json`: as da pessoa, `[{"nome", "ganhos", "pre"}]`. Entrada
  torta é ignorada na leitura.
- `eq.conf`: a configuração do filter-chain (abaixo).

Em `~/.config/systemd/user/`: `mksom-eq.service`.

## O equalizador

- 10 bandas fixas: 31, 62, 125, 250, 500, 1000, 2000, 4000, 8000 e 16000 Hz.
  A primeira é `bq_lowshelf`, a última `bq_highshelf`, as do meio
  `bq_peaking`; todas com Q 1.0.
- Ganho de cada banda: de −12 a +12 dB, em passos de 0,5. Pré-ganho: de −12 a
  0 dB, em passos de 0,5. Fora disso o motor recusa (código 1).
- O `eq.conf` é uma configuração inteira para `pipewire -c <eq.conf>`: os
  módulos de base (`rt`, `protocol-native`, `client-node`, `adapter`) e o
  `filter-chain`, que cria o sink virtual **Mk Equalizador** (`node.name =
  "mksom.eq"`, `media.class = Audio/Sink`, estéreo `FL FR`) e toca no sink de
  verdade (`node.name = "mksom.eq.saida"`, com `target.object` = o nome da
  saída padrão que havia ao ligar, para o equalizador nunca tocar em si
  mesmo; sem saída conhecida, sem `target.object`).
- O grafo, por canal (`L` e `R`): `pre_<C>` (`linear`, controle `Mult` =
  10^(pré/20)) → `eq_<C>_0` → … → `eq_<C>_9`. **Nomes dos controles** (são
  estes que o ajuste ao vivo usa): `eq_L_0:Gain` … `eq_L_9:Gain`,
  `eq_R_0:Gain` … `eq_R_9:Gain`, `pre_L:Mult` e `pre_R:Mult`.
- A unidade `mksom-eq.service`: `ExecStart=pipewire -c "<eq.conf>"`, depois
  do `pipewire.service` e do `wireplumber.service`, presa ao
  `pipewire.service`, `WantedBy=default.target`.
- **Ajuste ao vivo**: com o sink no ar, nada reinicia. Para cada banda que
  mudou, uma chamada
  `pw-cli set-param <id do mksom.eq> Props '{ params = [ "eq_L_3:Gain" 3.5 "eq_R_3:Gain" 3.5 ] }'`
  (os dois canais juntos); para o pré-ganho, o mesmo com `pre_L:Mult` e
  `pre_R:Mult`. Banda que não mudou não gera chamada. Depois o estado é
  gravado e o `eq.conf` regravado, para a próxima subida.
- Predefinições de fábrica: `plano`, `graves`, `agudos`, `voz`, `noite`
  (corta graves e agudos) e `fone pequeno`. As da pessoa têm nome de 1 a 40
  caracteres (os espaços das pontas saem), sem `/`, sem `\`, sem caractere de
  controle e sem começar com `-`, diferente (sem olhar maiúsculas) dos de
  fábrica; até 50.

## `mksom-motor estado`

Tudo que a janela precisa para abrir. Sai com 0 mesmo se o PipeWire não
responde (`som.responde` é `false` e `som.erro` diz por quê).

```json
{
  "som": {"responde": true, "erro": null},
  "eq": {
    "ligado": true,
    "na_saida": true,
    "bandas": [31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000],
    "ganhos": [4, 3.5, 2.5, 1, 0, 0, 0, 0, 0, 0],
    "pre": -6,
    "predefinicao": "graves",
    "saida_anterior": {"nome": "alsa_output.exemplo.analog-stereo", "descricao": "Alto-falantes", "existe": true},
    "predefinicoes": [{"nome": "plano", "fabrica": true, "ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0}]
  },
  "saida_padrao": {"id": 71, "nome": "mksom.eq", "descricao": "Mk Equalizador"},
  "entradas": [
    {"id": 52, "nome": "alsa_input.exemplo.analog-stereo", "descricao": "Microfone interno", "padrao": true, "volume": 74, "mudo": false}
  ]
}
```

- `eq.ligado`: o sink `mksom.eq` existe no `pw-dump` (o estado de verdade,
  não o que o `estado.json` diz). `eq.na_saida`: ele é a saída padrão.
- `eq.predefinicao`: o nome da que está aplicada, ou `null`.
- `eq.saida_anterior`: a saída que era a padrão ao ligar, ou `null`.
- `saida_padrao`: `null` se não há.
- `entradas`: os nós `Audio/Source` (e `Audio/Source/Virtual`), sem
  monitores. `volume` em % (0 a 150, na escala do `wpctl`: raiz cúbica do
  volume linear), `null` se o `pw-dump` não diz.

## `mksom-motor eq ligar`

1. lê o `pw-dump` (se não responde: erro, nada é escrito);
2. se o sink já está no ar, só o torna a saída padrão; senão escreve o
   `eq.conf` e a unidade, `systemctl --user daemon-reload`, `systemctl --user
   enable --now mksom-eq.service` e espera o sink aparecer no `pw-dump` (até
   5 s);
3. `wpctl set-default <id do sink>`;
4. grava o estado (`ligado`, e `saida_anterior` = a padrão de antes);
5. olha o `pw-dump` até o equalizador ser a saída padrão de verdade (até
   1,5 s: o WirePlumber troca um instante depois de aceitar o pedido). Se não
   virar, nada é desfeito: a resposta vem com `na_saida: false` e um `aviso`.

Se o sink não aparece, ou o `enable` ou o `set-default` falham, **desfaz**
(`systemctl --user disable --now mksom-eq.service`), grava `ligado: false` e
responde o erro (código 1).

```json
{"ok": true, "ligado": true, "na_saida": true, "saida_anterior": {"nome": "alsa_output.exemplo.analog-stereo", "descricao": "Alto-falantes", "existe": true}, "aviso": null}
```

## `mksom-motor eq desligar`

Se a saída padrão é o equalizador e a anterior ainda existe, `wpctl
set-default <id dela>`; se sumiu, não inventa (o WirePlumber escolhe). Depois
`systemctl --user disable --now mksom-eq.service` e `ligado: false`. Se o
`pw-dump` não responde, a unidade é desligada do mesmo jeito e `aviso` conta.
Mesma resposta do `ligar`, com `ligado` e `na_saida` em `false`.

## `mksom-motor eq ajustar`

O pedido vem pela entrada padrão: `{"ganhos": [10 números], "pre": número}`.

```json
{"ok": true, "ao_vivo": true, "aplicados": ["eq_L_3:Gain", "eq_R_3:Gain"], "predefinicao": null}
```

`ao_vivo` é `false` quando o sink não está no ar (só o estado é gravado).
`predefinicao` é o nome da que tem exatamente esses valores, ou `null`. Se um
`pw-cli` falha: erro (código 1) e o estado fica como estava.

## `mksom-motor eq predefinicao`

- `aplicar -- <nome>`: como o `ajustar`, com os valores dela.
- `guardar -- <nome>`: guarda os valores de agora com esse nome (troca a que
  tiver o mesmo nome).
- `renomear -- <de> <para>` e `apagar -- <nome>`.

```json
{"ok": true, "predefinicao": "minha", "predefinicoes": [], "ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0, "ao_vivo": false, "aplicados": []}
```

(`predefinicoes` é a lista inteira, as de fábrica primeiro, como no
`estado`.) Nome inválido, que já existe (no `renomear`) ou que não existe:
código 1. Mexer numa de fábrica (guardar por cima, renomear, apagar): código 2.

## `mksom-motor mic listar`

`{"entradas": [...]}`, como no `estado`. Se o PipeWire não responde: erro.

## `mksom-motor mic padrao <id>`, `mic volume <id> <0-150>`, `mic mudo <id> <sim|nao>`

`wpctl set-default <id>`, `wpctl set-volume <id> <0.00 a 1.50>`, `wpctl
set-mute <id> <1|0>`. Respondem `{"ok": true}`. Id que não é número: código
1; id que não é de uma entrada do `pw-dump`: código 2.

## `mksom-motor mic nivel <id>`

Um fluxo. O motor roda `pw-record --raw --target <alvo> --format s16 --rate 16000
--channels 1 -`, lê o PCM da saída dele e escreve umas 15 linhas por segundo
(uma a cada 1066 amostras):

```json
{"pico": -6, "rms": -9}
```

em dBFS, com uma casa, piso −90. `<alvo>` é o `object.serial` do nó quando o
`pw-dump` o diz (é o que o `--target` do `pw-record` de hoje entende por um
número), senão o id. Para quando a entrada padrão do motor fecha, quando
chega a linha `parar` ou um SIGTERM: o `pw-record` é morto e a última linha é
`{"fim": true}` (código 0). Se o `pw-record` morre sozinho com erro:
`{"erro": …}` e código 1. Um cabeçalho WAV no começo do fluxo é pulado.

## `mksom-motor mic gravar <id>`, `mic ouvir`, `mic limpar`

- `gravar`: 5 s de `pw-record --target <alvo> --format s16 --rate 48000
  --channels 1 <arquivo>`, com `<arquivo>` = `$XDG_RUNTIME_DIR/mksom/teste.wav`
  (pasta `700`). Sem `XDG_RUNTIME_DIR` o motor recusa (código 2): a gravação
  nunca vai para disco. Solta `{"falta": 5}` … `{"falta": 1}` (uma por
  segundo) e, no fim, `{"ok": true, "segundos": 5}`. Interrompido (entrada
  fechada, `parar`, SIGTERM): apaga o arquivo e termina com `{"fim": true}`.
- `ouvir`: `pw-play <arquivo>`, e **apaga o arquivo** quando termina (ou
  quando é interrompido). `{"ok": true}`. Sem gravação: código 2.
- `limpar`: apaga a gravação, se houver. A janela chama ao fechar.

O som gravado não sai da máquina nem vai a lugar nenhum além do `pw-play`.

## Códigos de saída

`0` certo; `1` erro de uso, de validação ou de sistema; `2` recusado (a
predefinição é de fábrica, o id não é de uma entrada, não há onde ou o que
gravar/ouvir). Sempre com `{"erro": …}`.

## A janela

- Título e classe: `MkSom`. Pequena e flutuante (nasce por volta de 560×520;
  a regra de flutuar do Hyprland casa pelo título). Cabe em 420×420: abaixo de
  480 de largura os controles estreitam e os rótulos encurtam (`1 kHz` vira
  `1k`); abaixo de 470 de altura a curva some.
- Duas áreas: **Equalizador** e **Microfone**.
- Um processo do motor por pedido: `estado` ao abrir; os outros quando a
  pessoa mexe. Arrastar um controle aplica ao soltar e, durante o arraste, no
  máximo 10 vezes por segundo, nunca com dois `eq ajustar` ao mesmo tempo.
- A curva desenhada é a soma, em dB, da resposta de cada filtro (as fórmulas
  de biquad do Audio EQ Cookbook, a 48 kHz) mais o pré-ganho: uma
  aproximação do que o PipeWire faz. Quando o pico dessa soma passa de 0 dB
  (os ganhos positivos passam do pré-ganho), a janela avisa "pode distorcer"
  e oferece o pré-ganho que resolve.
- O medidor só roda com a área Microfone à vista; a janela fecha a entrada
  padrão do `mic nivel` ao sair dela.
- O diagnóstico sai do `Som.js`, a partir dos níveis dos últimos 2 s. As
  frases:
  - "nenhuma entrada";
  - "a entrada está muda";
  - "está saturando (baixe o volume)", assim que um pico passa de −1 dBFS;
  - "o nível está muito baixo (fale perto / suba o volume)", quando nenhum
    pico chega a −40 dBFS;
  - "o nível está bom".
- `MKSOM_MOTOR` troca o binário. `MKSOM_EXEMPLO=pasta` faz a janela usar
  respostas gravadas (`estado.json`, `eq-ligar.json`, `eq-desligar.json`,
  `eq-ajustar.json`, `eq-predefinicao-aplicar.json`,
  `eq-predefinicao-guardar.json`, `eq-predefinicao-renomear.json`,
  `eq-predefinicao-apagar.json`, `mic-listar.json`, `mic-padrao.json`,
  `mic-volume.json`, `mic-mudo.json`, `mic-nivel.json` e `mic-gravar.json` —
  listas, uma linha por item —, `mic-ouvir.json`, `mic-limpar.json`) em vez do
  motor: nesse modo ela não chama o motor nem toca ou grava som.
- O lançador (`bin/mksom`) abre uma janela só; `mksom microfone` abre direto
  na área Microfone (`MKSOM_AREA`). `MKSOM_RAIZ` diz onde estão `bin/` e
  `qml/`.
- Atalhos: `Tab` anda entre as bandas, `↑`/`↓` meio dB, `Page Up`/`Page Down`
  3 dB, `0` zera a banda, `Espaço` liga e desliga o equalizador, `Ctrl+1`/
  `Ctrl+2` trocam de área, `Ctrl+S` guarda como…, `Ctrl+0` deixa plano, `F1`
  mostra os atalhos, `Esc` fecha a folha aberta.
