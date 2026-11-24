# O que não foi testado, e os limites conhecidos

O Mk é testado de três jeitos: os testes automáticos (`tests/run`, sem
máquina nenhuma), a sessão aninhada (`bin/dev-shot`) e uma VM QEMU
(`iso/vm`). Esta página diz o que nenhum dos três alcança e o que se sabe que
não funciona. Uma coisa sai daqui quando alguém a confere numa máquina de
verdade e registra onde (issue ou commit).

## Conferido só na VM

A VM tem um disco virtio, uma tela, rede cabeada e firmware OVMF. Nela foram
conferidos:

- a instalação do começo ao fim, com o `limine.conf` gerenciado, uma entrada
  UEFI só ("Mk") e o submenu de snapshots;
- a atualização por `mk atualizar`, com migrações de sistema e de usuário, e
  o reinício depois da migração do boot;
- o pedido de senha de administrador (o Maré como agente de polkit), o
  chaveiro e o agente SSH;
- o teclado e os cliques fora dos painéis da ilha;
- as boas-vindas e a instalação das ferramentas de desenvolvimento pelo mise;
- uma taxa de atualização fora da lista do EDID (modeline aceito e a volta
  sozinha em 15 s).

## Nunca testado

| O quê | Onde está | O que falta |
|---|---|---|
| Disco criptografado (LUKS) | `installer/mk-install` | uma instalação com `ENCRYPT=sim` e o boot dela |
| Placa NVIDIA | `installer/mk-install` (drivers, parâmetros do kernel) | uma máquina com NVIDIA |
| Impressoras de verdade | `shell/mare/scripts/mare-printers`, Ajustes › Impressoras | achar, instalar e imprimir numa impressora de rede e numa USB |
| VPN de verdade | `shell/mare/scripts/mare-vpn`, Ajustes › VPN | importar um `.ovpn` e um WireGuard e conectar |
| Wi-Fi: esquecer rede, rede oculta, QR | `shell/mare/scripts/mare-wifi`, `mare-wifi-qr` | uma placa Wi-Fi (a VM só tem cabo) |
| Bluetooth: parear com código | `shell/mare/BluetoothPair.qml` | um adaptador e um aparelho |
| Aviso de bateria fraca | `shell/mare/BatteryWatch.qml` | um notebook descarregando até 20% e 8% |
| Gravação de tela | `shell/mare/scripts/mare-shot` | gravar e conferir o arquivo numa máquina com GPU |
| Driver `xone` do controle Xbox | Ajustes › Controle, `mare-gamepad` | instalar pelo botão e ouvir o headset no controle |
| VS Code, AWS, Wrangler, Vercel | `migrations/system/20261127-editor-nuvem.sh` | a instalação dos quatro numa máquina |
| Modelines nos monitores reais | `shell/mare/scripts/mare-monitors` | os números calculados diferem dos testados à mão num dos monitores |
| Regras de monitor por descrição (`desc:`) | `mare-monitors` | não existe: as regras são pelo nome da saída (DP-1...) |
| Firmware UEFI de placas reais | `finish_boot` em `installer/mk-install` | alguns firmwares recriam entradas ("UEFI OS") por conta própria |
| Dois monitores de larguras diferentes | `shell/mare/Monitor.qml` | a posição dos painéis foi conferida com uma tela virtual, não com duas físicas |

## Não funciona, de propósito ou por limite

- **Hibernação.** Retirada. Na VM a sessão era gravada e não voltava no boot
  seguinte (`systemd-hibernate-resume: Unable to resume from device`), e a
  causa não foi achada. O Mk suspende ou não dorme. Quem preparou a
  hibernação numa versão antiga desfaz com `sudo mk-hibernate remove`.
- **Suspender na AMD Navi 10** (RX 5600/5700). A placa às vezes não volta: a
  tela fica preta e só reiniciando. É um defeito do driver, não do Mk. O
  instalador semeia "não dormir" nessas máquinas e Ajustes › Energia avisa.
- **BIOS antiga (sem UEFI)**: o instalador não instala.
- **Sistemas de arquivos além do btrfs**: os snapshots e a volta atrás pelo
  menu de boot dependem dele.

## Defeitos conhecidos, achados pelos testes e ainda abertos

Cada um tem dono certo: virar um caso que falha em `tests/` e depois a
correção.

- **Três ou mais monitores.** Com duas telas o desenho da página Monitores
  fica sempre limpo (testado). Com três ou mais, pôr uma tela num lado já
  ocupado, desfazer um espelho ou desligar uma tela pode deixar duas por cima
  uma da outra ou uma solta pelo canto. A página avisa (sobreposição, tela
  solta), mas não arruma sozinha.
- **Calculadora da busca:** `-2^2` dá 4 (o menos prende mais que a potência),
  e os espaços somem antes da conta (`1 2 + 3` dá 15).
- **Ajustes:** um valor de tipo errado posto à mão no `ajustes.json` (texto
  onde vai número) segue para o Hyprland sem ser recusado.
- **`mk atualizar`:** não há trava contra duas rodadas ao mesmo tempo, e um
  download que falha deixa a pasta temporária para trás.
- **`bin/check-packages` sem rede** acusa todo pacote do AUR como inexistente,
  em vez de dizer que não alcançou o AUR.
- **`mare-gamepad`:** instalar ou remover o `xone` sai com 0 mesmo quando o
  `yay` falha.
- **`mare-wifi-qr`:** uma rede WEP é tratada como aberta.
- **Clima:** um portal cativo que responde no lugar da previsão deixa o painel
  vazio, sem dizer o motivo.

## O que os testes automáticos não veem

O desenho dos painéis, o foco do teclado, as animações e a conversa com o
Hyprland de verdade. `tests/run` confere a lógica (scripts, esquema dos
Ajustes, configuração, instalador, migrações, atualização) com o sistema
trocado por dublês. Uma mudança que mexe no que aparece na tela continua
pedindo uma olhada com o `bin/dev-shot` ou na VM.
