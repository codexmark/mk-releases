---
camada: "12 Decisões e limites"
tags:
  - mk
  - limites
  - testes
  - hardware
fontes:
  - "tests/README.md"
  - "README.md"
  - "docs/limites.md"
  - "installer/README.md"
  - "iso/README.md"
  - "docs/atualizacoes.md"
  - "shell/mare/README.md"
---

# Limitações conhecidas

O que o Mk não faz, o que nunca foi testado em hardware real e onde a documentação e o código divergem.

## Para quê

Esta é a nota a ler antes de confiar no Mk numa máquina que importa.

## Como funciona

### Não existe, de propósito ou por limite

- **BIOS legado**: o instalador para se o live não foi iniciado em UEFI.
- **Dual boot**: o instalador usa o disco inteiro.
- **Discos com menos de 24 GiB.**
- **Sistemas de arquivos além do btrfs**: os snapshots e a volta atrás dependem dele.
- **Hibernação**: retirada ([[D05 Hibernação retirada]]).
- **Voltar só o Mk para uma versão anterior**: não há; o conserto é outro lançamento.
- **Regras de monitor por descrição (`desc:`)**: as regras são pelo nome da saída.
- **Agente de pareamento Bluetooth**: aparelhos que pedem código ou confirmação falham.
- **Agente gráfico de segredos para VPN**: a que pede senha conecta num terminal.
- **Placas NVIDIA anteriores à geração Turing**: o perfil usa `nvidia-open-dkms`.

### Nunca testado em hardware real

| O quê | Onde está |
|---|---|
| disco criptografado (LUKS), inclusive a senha no plymouth e o autologin | [[LUKS]] |
| perfil `nvidia` | [[Microcódigo e perfis de hardware]] |
| escolha de Wi-Fi no instalador; esquecer rede, rede oculta, QR | [[Rede]] |
| impressoras de verdade | [[Impressão]] |
| importar e conectar uma VPN | [[Rede]] |
| parear Bluetooth | [[Bluetooth]] |
| aviso de bateria fraca | [[Notificações e OSD]] |
| gravação de tela numa máquina com GPU | [[Painéis]] |
| driver `xone` do controle Xbox | [[Páginas dos Ajustes]] |
| VS Code, AWS, Wrangler, Vercel instalados numa máquina | [[Ferramentas de desenvolvimento]] |
| dois monitores físicos de larguras diferentes | [[Monitor por tela]] |
| modelines nos monitores reais (os números calculados diferem dos testados à mão num deles) | [[Monitores]] |
| firmwares UEFI variados (alguns recriam entradas como "UEFI OS") | [[Entrada UEFI única]] |
| iniciar por um snapshot e tornar a volta permanente | [[Voltar atrás pelo menu de boot]] |
| SSH para dentro da VM, de ponta a ponta | [[VM de teste]] |

### Defeitos conhecidos de terceiros

- **Suspender na AMD Navi 10** (RX 5600/5700): a placa às vezes não volta. O Mk semeia "não dormir" na instalação nova e avisa nos Ajustes ([[Energia e suspensão]]).

### O que os testes automáticos não veem

O desenho dos painéis, o foco do teclado, as animações e a conversa com o Hyprland de verdade; a instalação num disco e o boot. Uma mudança que mexe no que aparece na tela continua pedindo o `dev-shot` ou a VM ([[Testes e TDD]]).

### Dívidas declaradas

- `monitors.lua` e `user.lua` ficam dentro de `hypr/`, que a atualização troca: **se perdem a cada atualização** ([[Pasta do usuário]]).
- As listas `needed` de `try-session` e `new-user` são mantidas à mão e não são iguais; a do `try-session` não tem o `uwsm`.
- O `ufw-docker install` não é rodado pelo instalador ([[Firewall]]).
- O `grain.png` tem cópias que o gerador não atualiza ([[Arte e geradores]]).
- A paleta é repetida em vários arquivos sem nada que confira as cópias ([[Direção Folha]]).
- O `ipc call mare set` não confere faixa nem opções ([[IPC do Maré]]).
- O `hyprlock.conf` de reserva está fora da paleta ([[Ociosidade e bloqueio]]).

### Onde a documentação e o código divergem

Encontrado ao escrever esta vault, lendo os dois lados. Várias divergências da primeira leitura já foram corrigidas no repositório (o seletor de projeto, os testes no lançamento, os cabeçalhos de `desktop.packages`, `aur.packages` e `bin/mk-session`); restam:

As duas últimas (o que o `mk-update` reinstala e onde fica o `hyprland.lua`) também foram corrigidas: hoje não há divergência conhecida. O `tests/docs/docs.test.sh` confere parte disso a cada rodada.

## Por quê assim

- **O projeto declara os próprios limites.** O `README.md` abre com "Estado do projeto" e aponta para `docs/limites.md`, a lista completa "com o que falta para cada item sair dela". A regra de lá é que uma coisa só sai da lista "quando alguém a confere numa máquina de verdade e registra onde".
- **Três meios de teste, nenhum com hardware**: `tests/run` (a lógica, com o sistema trocado por dublês), o `dev-shot` (a sessão aninhada) e a VM, que não tem Wi-Fi, placa de vídeo real, segundo monitor, impressora, Bluetooth, bateria nem suspensão confiável.
- **As divergências entre documentação e código** são o custo de um projeto que mudou muito em poucos dias. Nenhuma delas muda o comportamento; mudam o que alguém espera ao ler.

> [!warning] O que não foi possível confirmar no código
> Três coisas que o sistema faz na prática mas que o repositório não mostra como: quem inicia o daemon do `gnome-keyring` ([[Chaveiro e agente SSH]]); como o PipeWire e o WirePlumber sobem na sessão ([[Áudio]]); e se o NetworkManager usa o `iwd` como backend ([[Rede]]).

## Onde está no código

`README.md` ("Estado do projeto"), `docs/limites.md`, `installer/README.md` ("O que foi testado"), `iso/README.md` ("O que a VM não mostra"), `shell/mare/README.md` (o fim da seção de scripts).

## Ligações

[[O que foi testado]] · [[Registro de decisões]] · [[Motivos não registrados]] · [[VM de teste]] · [[Quando a atualização para]]
