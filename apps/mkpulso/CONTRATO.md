# MkPulso: o contrato entre o coletor e a janela

O MkPulso é o monitor de sistema do Mk: processos, desempenho, a ficha da
máquina, o que sobe no login e os serviços. Tem duas metades:

- **o coletor** (`src/`, em Go, um binário só: `bin/mkpulso-coletor`), que lê
  o sistema e executa as ações. É onde mora toda a lógica, e é todo testado
  com `/proc`, `/sys` e comandos falsos;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que só mostra o que
  o coletor diz e pede o que a pessoa clicou.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados. Todo texto que a janela mostra vem pronto em português do coletor só
quando é um estado (`energia`, `estado`); rótulos de coluna são da janela.

## Regras gerais

- O coletor escreve JSON em UTF-8 no stdout, um objeto por linha (sem quebra
  de linha dentro do objeto). Erro vai para o stderr e para o código de saída.
- Um subcomando que falha escreve no stdout `{"erro": "frase em português"}`
  e sai com código diferente de 0. A janela mostra a frase.
- Número que o coletor não conseguiu ler é `null`, nunca 0 nem ausente.
- Tamanhos em **kB**, taxas em **bytes por segundo** (`_bps`), porcentagens de
  0 a 100 por núcleo somados (um processo com dois núcleos cheios mostra 200),
  tempos em **segundos desde a época** (`_s`) ou milissegundos (`_ms`).
- Raízes trocáveis para teste, por variável de ambiente: `MKPULSO_PROC`
  (`/proc`), `MKPULSO_SYS` (`/sys`), `MKPULSO_ETC` (`/etc`), `MKPULSO_RUN`
  (`/run`), `MKPULSO_HOME` (a home), `MKPULSO_APPS` (`/usr/share/applications`),
  `MKPULSO_PACMAN` (`/var/lib/pacman/local`), `MKPULSO_MK` (`/usr/share/mk`) e
  `MKPULSO_PCIIDS` (`/usr/share/hwdata/pci.ids`, de onde sai o nome da GPU). Comandos externos (`hyprctl`,
  `systemctl`, `journalctl`, `kill` não: sinais são chamadas do sistema) são
  achados no `PATH`, então o teste põe dublês na frente.
- O coletor nunca pede senha e nunca roda como root. O que precisa de root
  (uma unidade de sistema) passa pelo `systemctl`, que chama o polkit (o Maré
  é o agente e mostra o pedido de senha).

## `mkpulso-coletor watch [--intervalo MS] [--uma-vez]`

Solta um retrato a cada intervalo (padrão 1000 ms, mínimo 250). `--uma-vez`
mede duas vezes com o intervalo no meio, solta um retrato e sai (as taxas
precisam de duas leituras). Encerra limpo com SIGTERM ou quando o stdout
fecha.

```json
{
  "t_ms": 1795480000000,
  "maquina": "nome-da-maquina",
  "nucleos": 8,
  "cpu": { "pct": 37.5, "por_nucleo": [12.0, 80.1], "freq_mhz": 2400, "temp_c": 61.0, "carga": [1.2, 0.9, 0.7] },
  "mem": { "total_kb": 16000000, "usado_kb": 9000000, "disponivel_kb": 7000000, "cache_kb": 3000000,
           "swap_total_kb": 0, "swap_usado_kb": 0 },
  "discos": [ { "nome": "nvme0n1", "leitura_bps": 0, "escrita_bps": 8192 } ],
  "montagens": [ { "onde": "/", "fs": "btrfs", "total_kb": 500000000, "usado_kb": 120000000 } ],
  "rede": [ { "nome": "wlan0", "rx_bps": 1400, "tx_bps": 300 } ],
  "gpus": [ { "nome": "AMD Radeon RX 5700", "uso_pct": 12.0, "mem_usada_kb": 800000, "mem_total_kb": 8000000, "temp_c": 55.0 } ],
  "bateria": { "pct": 75, "carregando": false, "watts": 9.4 },
  "totais": { "processos": 327, "grupos": 39, "threads": 1500 },
  "grupos": [
    {
      "id": "app:firefox",
      "tipo": "app",
      "nome": "Firefox",
      "icone": "firefox",
      "unidade": "app-firefox-1234.scope",
      "escopo": "usuario",
      "janelas": [ { "titulo": "Mk — Firefox", "workspace": "2", "endereco": "0x55d0" } ],
      "n": 14,
      "pid": 1234,
      "cpu_pct": 5.9, "mem_kb": 839000, "disco_bps": 0, "gpu_pct": null,
      "energia": "muito baixa",
      "media": { "cpu_pct_1m": 4.0, "cpu_pct_3m": 3.1, "mem_kb_1m": 830000, "mem_kb_3m": 820000,
                 "disco_bps_1m": 0, "disco_bps_3m": 0 },
      "processos": [
        { "pid": 1234, "ppid": 900, "nome": "firefox", "cmd": "/usr/lib/firefox/firefox", "usuario": "mark",
          "estado": "dormindo", "cpu_pct": 3.0, "mem_kb": 400000, "disco_bps": 0, "gpu_pct": null,
          "threads": 80, "nice": 0, "inicio_s": 1795470000 }
      ]
    }
  ]
}
```

- **`grupos[].tipo`**: `"app"` (tem janela, ou é um `app-*.scope` com atalho
  de menu), `"servico"` (uma unidade do systemd, de sistema ou de usuário) ou
  `"sistema"` (threads do kernel e o que não pertence a unidade nenhuma, num
  grupo só, `id` `"sistema:kernel"` e `"sistema:outros"`).
- **`cpu.pct`**: da máquina inteira, de 0 a 100. (Só o `cpu_pct` de grupos e
  processos soma por núcleo e passa de 100.)
- **`grupos[].id`**: estável entre retratos (`app:<nome>`, `servico:<unidade>`),
  para a janela manter aberto o grupo que a pessoa abriu. **Não é único
  sozinho**: a mesma unidade pode existir no sistema e na sessão
  (`dbus-broker.service`), e só o `escopo` as separa. A chave de um grupo é
  `id` + `escopo`.
- **De onde vem o grupo**: do cgroup do processo. `app-<nome>-*.scope` e
  `app-<nome>@*.service` viram `app:<nome>`; uma unidade da sessão,
  `servico:<unidade>` com escopo `"usuario"`; uma do sistema, com escopo
  `"sistema"`; o que não está em unidade nenhuma (uma sessão de terminal)
  agrupa pelo programa do topo, `app:<executável>`, sem unidade nem escopo. A
  regra inteira está no cabeçalho de `src/internal/grupos/grupos.go`.
- **`grupos[].tipo`** é `"app"` só com janela ou atalho de menu; um
  `app:<executável>` sem nenhum dos dois sai como `"servico"`.
- **`grupos[].escopo`**: `"usuario"` ou `"sistema"` (de quem é a unidade); `""`
  quando não há unidade.
- **`grupos[].pid`**: o processo principal (o `MainPID` da unidade ou o mais
  antigo do grupo).
- **`grupos[].energia`**: uma de `"nenhuma"`, `"muito baixa"`, `"baixa"`,
  `"moderada"`, `"alta"`, `"muito alta"`. A regra: impacto = CPU do grupo ÷
  núcleos + GPU ÷ 2 + disco em MiB/s (até 10); abaixo de 0,1 nenhuma, de 2
  muito baixa, de 8 baixa, de 25 moderada, de 60 alta, e daí muito alta
  (`src/internal/grupos/energia.go`, com as fronteiras testadas).
- **`grupos[].pid`** é o processo mais antigo do grupo. **`mem_kb`** é a soma
  da memória residente dos processos (a compartilhada conta mais de uma vez).
  **`cmd`** vem cortado em 300 bytes.
- **`grupos[].media`**: médias do último minuto e dos últimos três, guardadas
  pelo `watch` (por isso só existem depois de ele rodar esse tempo; antes
  disso valem a média do que há). Em `--uma-vez`, iguais ao valor atual.
- **`processos[].estado`**: `"rodando"`, `"dormindo"`, `"esperando disco"`,
  `"parado"`, `"zumbi"`, `"ocioso"`.
- **Ordem**: os grupos saem ordenados por `cpu_pct` decrescente (empate: `id`);
  os processos de um grupo também. A janela reordena como a pessoa pedir.
- `gpus`, `bateria`, `temp_c`, `gpu_pct`: `null` ou lista vazia quando a
  máquina não tem ou o coletor não alcança. Também podem vir `null`: `mem`,
  `cpu.pct`, `cpu.freq_mhz`, `cpu.carga`, `disco_bps` (processo de outro
  usuário), `inicio_s`, e o tamanho de uma montagem. `maquina` desconhecida é
  `""` no `watch` e `null` no `sistema`.
- `--intervalo` abaixo de 250 vira 250; acima de uma hora, uma hora.
- Um processo que some entre a listagem e a leitura é ignorado, sem erro.

## `mkpulso-coletor sistema`

A ficha da máquina, uma vez só:

```json
{
  "maquina": "nome", "modelo": "HP EliteBook 840 G3", "placa": "HP 8079", "firmware": "N75 Ver. 01.60",
  "cpu": { "nome": "Intel Core i5-6300U", "nucleos": 2, "threads": 4, "max_mhz": 3000 },
  "mem_total_kb": 8000000,
  "gpus": [ { "nome": "Intel HD Graphics 520", "driver": "i915" } ],
  "discos": [ { "nome": "nvme0n1", "modelo": "Samsung 970", "tamanho_kb": 500000000, "rotativo": false, "saude": "ok" } ],
  "bateria": { "modelo": "…", "ciclos": 984, "saude_pct": 71, "capacidade_wh": 33.1 },
  "telas": [ { "nome": "eDP-1", "descricao": "…", "largura": 1366, "altura": 768, "hz": 60.0, "escala": 1.0 } ],
  "sistema": { "nome": "Mk", "versao_mk": "4c4ea04fc26e", "data_mk": "2026-11-23", "kernel": "7.3.5-arch1-1",
               "compositor": "Hyprland 0.57.2", "ligado_ha_s": 3600, "instalado_em_s": 1795000000, "pacotes": 812 }
}
```

`saude` de um disco: `"ok"`, `"atenção"`, `"falhando"` ou `null` (sem como
saber sem root). Campos desconhecidos são `null`.

## `mkpulso-coletor sinal <pid> <sinal>`

`<sinal>`: `encerrar` (TERM), `matar` (KILL), `pausar` (STOP), `continuar`
(CONT), `recarregar` (HUP), `interromper` (INT). Recusa o pid 1, o próprio
coletor, a sessão em que ele roda (o Hyprland e o Maré) a menos que venha
`--mesmo-assim`, e processos de outro usuário. Resposta:
`{"ok": true, "pid": 1234, "sinal": "encerrar"}`.

## `mkpulso-coletor grupo <id> <sinal> [--pids 1,2,3] [--usuario] [--mesmo-assim]`

O mesmo para o grupo inteiro. `servico:<unidade>` vai pelo systemd
(`systemctl stop` para `encerrar`, `kill --signal=…` para os outros), como
unidade do sistema ou, com `--usuario`, da sessão: a janela passa `--usuario`
quando o `escopo` do grupo é `"usuario"`. `app:<nome>` acha as unidades do app
(`app-<nome>-*.scope`, `app-<nome>@*.service`); os processos de `--pids` (os
do retrato) que estiverem fora delas recebem o sinal direto, dos filhos para
os pais. Sem unidade e sem `--pids` é erro. Uma recusa em qualquer processo
aborta antes do primeiro sinal; `sistema:*` é sempre recusado. Resposta:
`{"ok": true, "id": "app:firefox", "sinal": "encerrar", "pids": [1234, 1240]}`.

## `mkpulso-coletor prioridade <pid> <nice>`

`<nice>` de -20 a 19. Baixar o nice (mais prioridade) de um processo próprio
pede privilégio: o coletor tenta e, recusado, responde com o erro
`"só dá para baixar a prioridade; para subir é preciso ser administrador"`.
Vale para todas as linhas de execução do processo. Resposta:
`{"ok": true, "pid": 1234, "nice": 5}`. As recusas são as do `sinal`.

## `mkpulso-coletor servicos [--usuario|--sistema]`

Sem opção, os dois. Uma linha com:

```json
{ "servicos": [
  { "unidade": "sshd.service", "escopo": "sistema", "descricao": "OpenSSH Daemon",
    "estado": "ativo", "sub": "running", "habilitado": "sim", "pid": 812, "mem_kb": 5400, "desde_s": 1795470000 }
] }
```

- `estado`: `"ativo"`, `"inativo"`, `"falhou"`, `"ativando"`, `"desativando"`.
- `habilitado`: `"sim"`, `"não"`, `"estático"` (não dá para habilitar),
  `"mascarado"`, `"gerado"`.
- Só unidades `.service`, `.socket` e `.timer`, inclusive as que só têm o
  arquivo (saem como inativas). Ficam de fora as não encontradas, os apelidos
  e os modelos (`nome@.service`).
- `desde_s` só vem com a unidade de pé.

## `mkpulso-coletor servico <acao> <unidade> [--usuario]`

`<acao>`: `iniciar`, `parar`, `reiniciar`, `habilitar`, `desabilitar`,
`recarregar`. Resposta: `{"ok": true, "unidade": "…", "acao": "reiniciar"}`.
Recusa parar ou desabilitar o que derruba a sessão ou a máquina
(`sddm.service`, `dbus*.service`, `systemd-*`, `user@*.service`,
`NetworkManager.service` só com `--mesmo-assim`), e parar ou reiniciar uma
unidade que contém a sessão ou o próprio coletor. O nome da unidade só aceita
letras, números e `: _ . @ - \`, não começa com `-` e termina em `.service`,
`.socket` ou `.timer`; o resto é erro, sem chegar ao `systemctl`.

## `mkpulso-coletor registro <unidade> [--usuario] [--linhas N]`

As últimas N linhas (padrão 80, máximo 500) do journal da unidade:
`{ "linhas": [ { "t_s": 1795470000, "nivel": "info", "texto": "…" } ] }`.
`nivel`: `"erro"`, `"aviso"`, `"info"`, `"depuração"`. `--linhas` fora de 1 a
500 é erro. `t_s` é `null` numa linha sem relógio.

## `mkpulso-coletor inicio`

O que sobe quando a pessoa entra:

```json
{ "itens": [
  { "id": "desktop:/etc/xdg/autostart/foo.desktop", "origem": "atalho", "nome": "Foo", "comando": "foo --tray",
    "ligado": true, "do_sistema": true, "mudavel": true },
  { "id": "unidade:syncthing.service", "origem": "serviço de usuário", "nome": "Syncthing", "comando": "syncthing",
    "ligado": true, "do_sistema": false, "mudavel": true },
  { "id": "mk:mare", "origem": "Mk", "nome": "Maré (a ilha e os painéis)", "comando": "mare-start",
    "ligado": true, "do_sistema": true, "mudavel": false }
] }
```

- `origem`: `"atalho"` (um `.desktop` em `~/.config/autostart` ou
  `/etc/xdg/autostart`), `"serviço de usuário"` (unidade de usuário
  habilitada) ou `"Mk"` (o que o `autostart.lua` do Mk sobe: só leitura).
- Um atalho do sistema é desligado do jeito do padrão XDG: uma cópia em
  `~/.config/autostart` com `Hidden=true`. Ligar de novo apaga essa cópia. O
  `id` de um atalho que existe no sistema é sempre o caminho do sistema, com
  ou sem a cópia.
- Atalhos que não valem no Hyprland (`OnlyShowIn`, `NotShowIn`, `TryExec`)
  ficam fora da lista.
- Uma unidade habilitada pelo sistema (em `/etc/systemd/user`) sai com
  `do_sistema: true` e `mudavel: false`.
- Uma unidade de usuário desligada some da lista (ela é das habilitadas): a
  janela guarda a linha para oferecer ligar de novo.

## `mkpulso-coletor inicio <ligar|desligar> <id>`

Resposta: `{"ok": true, "id": "…", "ligado": false}`. `mudavel: false` é
recusado com erro.

## Códigos de saída

`0` certo; `1` erro de uso ou de sistema (com `{"erro": …}`); `2` recusado por
segurança (pid 1, a sessão, outro usuário), também com `{"erro": …}`.
