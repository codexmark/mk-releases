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
| Descanso de tela pelo hypridle | `shell/mare/ScreenSaver.qml`, `config/hypr/hypridle.conf` | visto ligar e desligar por IPC na sessão aninhada; a espera real de 3 min, dois monitores e o consumo com ele horas na tela não foram vistos |
| `mk atualizar` com escolha de provedor | `bin/mk-update` (o ensaio antes da transação) | uma atualização de verdade em que o pacman precise escolher entre provedores |
| Arranjo de três ou mais telas na página | `shell/mare/Monitors.qml` | a lógica é testada com sorteios; a página nunca foi vista com três monitores de verdade |
| Dois monitores de larguras diferentes | `shell/mare/Monitor.qml` | a posição dos painéis foi conferida com uma tela virtual, não com duas físicas |

## Não funciona, de propósito ou por limite

- **Hibernação.** Retirada. Na VM a sessão era gravada e não voltava no boot
  seguinte (`systemd-hibernate-resume: Unable to resume from device`), e a
  causa não foi achada. O Mk suspende ou não dorme. Quem preparou a
  hibernação numa versão antiga desfaz com `sudo mk-hibernate remove`.
- **Dormir, de fábrica.** O Mk não suspende sozinho: no lugar fica o descanso
  de tela, e a tela apaga aos 30 minutos. Suspender é uma escolha em Ajustes ›
  Energia.
- **Suspender na AMD Navi 10** (RX 5600/5700). A placa às vezes não volta: a
  tela fica preta e só reiniciando. É um defeito do driver, não do Mk.
  Ajustes › Energia avisa quem escolher suspender nessas máquinas.
- **BIOS antiga (sem UEFI)**: o instalador não instala.
- **Sistemas de arquivos além do btrfs**: os snapshots e a volta atrás pelo
  menu de boot dependem dele.

## Defeitos e limites conhecidos, ainda abertos

Cada um tem dono certo: virar um caso que falha em `tests/` e depois a
correção.

- **Monitores vindos tortos de fora.** A página Monitores arruma sozinha tudo
  o que se faz nela, com qualquer número de telas (nenhuma sobreposta, nenhuma
  solta). O que o Hyprland já traz torto (um `hypr/monitors.lua` pessoal com
  telas sobrepostas, por exemplo) não é arrumado ao ser lido: a página só
  avisa. E o arranjo é guloso, não ótimo: um grupo solto pode encostar num
  lado que você não escolheria.
- **Ajustes, o painel.** Um valor de tipo errado ou fora da faixa posto à mão
  no `ajustes.json` não chega mais ao Hyprland nem ao `hypridle.conf` (vale o
  de fábrica), mas o painel ainda mostra o valor cru.
- **Calculadora da busca:** `1.000 + 500` dá 501. O ponto é lido como
  decimal; milhar com ponto não é aceito.
- **Atalhos:** um combo com duas teclas (`SUPER + Q + W`) fica só com a
  última.
- **`mk atualizar`:**
  - se a rodada parar entre baixar a lista de pacotes e atualizar, o sistema
    fica com a lista nova e os pacotes antigos: rode `mk atualizar` de novo
    antes de instalar qualquer pacote avulso;
  - a pergunta "Replace X with Y?" do pacman é respondida "sim" sozinha (é o
    sucessor que o Arch declarou); a escolha entre provedores não: aí a
    rodada para e mostra o comando;
  - a trava contra duas rodadas é por usuário; entre dois usuários da mesma
    máquina só a trava do pacman segura.
- **Desligar à força logo depois de atualizar, num Mk anterior a esta
  correção.** O `~/.config/mk/VERSION` pode ficar vazio, e aí `mk atualizar`
  diz "em dia ()" e não acha mais versão nova. Apague o arquivo
  (`rm ~/.config/mk/VERSION`) e rode de novo. As versões novas gravam o
  arquivo com segurança e ignoram um arquivo vazio.
- **`shellcheck` puxa uns 55 pacotes do Haskell** (as bibliotecas dele no
  Arch). Entra pela lista de desenvolvimento, para o lint do `tests/run` não
  ser pulado num Mk instalado.
- **Apps novos (MkPulso, MkPacotes, MkInstalar, MkComprimir), o que foi
  visto e o que não foi.** Os motores foram exercitados de verdade na VM:
  encerrar um processo, instalar e remover um pacote com a senha pedida pelo
  Maré, comprimir um vídeo, e as recusas (processo 1, SDDM, glibc). As janelas
  foram vistas abrindo, com dados reais, mas **os botões nunca foram
  clicados**: a folha de confirmação do MkPacotes, o seletor de arquivo e o
  arrastar do MkInstalar e do MkComprimir, e as ações de serviço do MkPulso só
  rodaram contra dublês ou pela linha de comando.
- **MkPacotes chama `pkexec pacman` pelo nome.** O pkexec acha o programa no
  `PATH` de quem chamou; o pedido de senha mostra o caminho inteiro
  (`/usr/bin/pacman`), mas o certo é o motor passar o caminho fixo, como o
  MkInstalar já faz.
- **MkPacotes não remove pacote do AUR com letra maiúscula no nome** (a
  validação só aceita minúsculas).
- **MkComprimir:** fechar a janela no meio de uma codificação pode deixar um
  arquivo `.mkcomprimir-*.parcial.*` escondido ao lado do vídeo.
- **MkCalc, MkCorte e MkEscrita: o que foi visto.** Na sessão aninhada, pelo
  teclado: a conta com prévia, o histórico e o erro do MkCalc; abrir, dividir,
  remover e exportar (modo exato) no MkCorte, com o arquivo conferido; abrir,
  digitar, continuar lista, salvar sozinho e a pré-visualização no MkEscrita.
  **Não visto:** nada com o mouse (arrastar alça e clipe na linha do tempo do
  MkCorte, as teclas da tela do MkCalc), tocar a edição com troca de vídeo
  entre clipes, o modo rápido pela janela, o seletor de arquivo, copiar e
  colar, o conflito e o rascunho recuperado do MkEscrita na tela, e nenhum dos
  três com o mouse numa instalação. Na VM, depois de um `mk-update` de
  verdade (que instalou o `qt6-multimedia`), os três abriram e fizeram o mesmo
  pelo teclado, e um clique no histórico do MkCalc funcionou.
- **MkCorte:** fechar a janela pelo compositor não pergunta pela edição não
  exportada (só o `Q` pergunta). A pré-visualização para no fim de cada clipe
  com precisão de dezenas de milissegundos; a exportação é exata. Aparar o
  início arrastando a alça funciona, mas a alça não acompanha o ponteiro: o
  caminho firme é `Ctrl+Espaço`.
- **MkEscrita:** não imprime (exporte HTML e imprima pelo navegador); cada
  marcação aplicada por atalho conta como dois passos de desfazer; arquivo
  que não é UTF-8 ou passa de 4 MB é recusado.
- **MkPorto:** só SSH; máquinas virtuais ficaram para depois. Os dados de
  cada host são os que o `ssh -G` diz (com `Match` e o config do sistema); se
  ele falhar, valem os do leitor próprio e o detalhe avisa. Salto em cadeia
  (com vírgula) não é sondado. O `ssh -G` roda os `Match exec` do seu config
  a cada leitura da lista. **Visto na VM**, pelo teclado, contra um `sshd` de
  verdade: criar e editar host, a sonda com a versão do servidor, conectar,
  testar a entrada antes e depois de enviar a chave, enviar a chave, um host
  por salto (sondado e conectado), gerar uma chave e copiar o comando. **Não
  visto:** nada com o mouse, e uma máquina de fora (só a própria VM).
- **MkTunes continua em Java.** O porte para Go espera o repositório dele, que
  não está na máquina onde os outros apps foram feitos.
- **Controles (Super + Esc):** vistos abrindo na sessão aninhada, com Esc
  fechando e a troca para outro painel; os botões e as chaves não foram
  clicados, e a cafeína segurando de fato o descanso de tela não foi vista. A
  ordem dos blocos é fixa (dá para escondê-los em Ajustes › Ilha, não
  reordenar). Quem tinha Super + Esc para o painel Sessão passa a abrir os
  Controles; a Sessão foi para Super + Ctrl + Esc.
- **`mare-wifi-qr`:** duas conexões salvas com o mesmo nome se confundem, e
  uma rede WEP por frase-senha gera um QR que o celular pode não aceitar.
- **`mare-shot`:** duas capturas no mesmo segundo usam o mesmo nome de arquivo.
- **Ler a tela (`mare-shot ler`):** o texto sai tão bom quanto o tesseract
  consegue: letra muito pequena, texto sobre foto e fontes de ícone erram. Em
  letra de terminal ele perde acentos ("música" sai "musica") e troca `#` e
  crase; ampliar pelo ImageMagick (quando instalado) melhora pouco: medido na
  VM, uma palavra a mais certa em dez linhas. Só
  português e inglês. Um código vale antes do texto: numa área com os dois, o
  texto não é lido.
- **Descanso de tela:** ele desliga quando a tela é bloqueada (aos 5 min, de
  fábrica), e daí em diante quem aparece é a tela de bloqueio, até a tela
  apagar aos 30. Com "Bloquear" em 0 ele fica até a tela apagar.

## O que os testes automáticos não veem

O desenho dos painéis, o foco do teclado, as animações e a conversa com o
Hyprland de verdade. `tests/run` confere a lógica (scripts, esquema dos
Ajustes, configuração, instalador, migrações, atualização) com o sistema
trocado por dublês. Uma mudança que mexe no que aparece na tela continua
pedindo uma olhada com o `bin/dev-shot` ou na VM.
