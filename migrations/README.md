# Migrações

Mudanças que o `mk-update` precisa fazer fora de `~/.config/mk` para um Mk já
instalado ficar igual a um recém-instalado. Cada script roda **uma vez**:

- `system/*.sh` — uma vez por máquina, como root (`sudo`). Registradas em
  `/var/lib/mk/migrations/<nome>`.
- `user/*.sh` — uma vez por usuário, como ele mesmo. Registradas em
  `~/.local/state/mk/migrations/<nome>`.

Rodam em ordem de nome, então o nome começa pela data: `AAAAMMDD-assunto.sh`.
Recebem `MK_SHARE` (o Mk instalado, normalmente `/usr/share/mk`). Precisam ser
idempotentes: rodar de novo não pode estragar nada. Se uma falhar, o
`mk-update` para e ela fica pendente para a próxima vez.

Migração de sistema **não instala pacote**: elas rodam antes do `pacman -Syu`,
e instalar sobre uma lista de pacotes velha falha e trava a atualização. Quem
precisa de pacote declara no cabeçalho, numa linha `# pacotes: a b c`: o
`mk-update` instala esses junto com a atualização do sistema e roda a migração
depois dela. A migração deve aguentar o pacote não estar lá (sem rede, por
exemplo): avisa e sai com 0.

Uma instalação nova já nasce com tudo: o instalador marca todas como feitas.
Mudou algo fora de `~/.config/mk` (em `/etc`, `/usr/share`, no boot, no
`~/.bashrc`)? Faça a mudança no instalador **e** escreva a migração.

## O que uma migração nova precisa ter

- **Nome** `AAAAMMDD-assunto.sh`, em `system/` ou em `user/`. O `mk-update` só
  enxerga `*.sh` nessas duas pastas e roda em ordem de nome: primeiro todas as
  de `system/`, depois as de `user/` (as de sistema com `# pacotes:` ficam para
  depois do pacman). O nome é a marca de "feita": não renomeie
  uma migração já lançada, ou ela roda de novo em todo mundo.
- **Cabeçalho**: `#!/usr/bin/env bash`, um comentário dizendo o que muda e por
  quê, e `set -euo pipefail`. O `mk-update` chama com `bash <arquivo>`; o bit
  de execução não importa.
- **`MK_SHARE`**: leia os arquivos do Mk por
  `${MK_SHARE:-/usr/share/mk}`, nunca pelo caminho fixo. É a única variável que
  o `mk-update` passa. As de sistema rodam com `sudo env MK_SHARE=…`: `$HOME` e
  `$USER` ali não são os de quem atualiza, então nada de mexer em home a partir
  delas.
- **`MK_ROOT`**: todo caminho do sistema (`/etc`, `/usr`, `/var`, `/boot`)
  passa por uma raiz, `R=${MK_ROOT:-}`, escrita logo depois do
  `set -euo pipefail`: `"$R/etc/skel"`, não `/etc/skel`. O `mk-update` não
  passa essa variável (e o `sudo env MK_SHARE=…` não a herda), então `$R` é
  vazio e os caminhos são os de sempre. Quem a usa são os testes
  (`tests/migrations/`), que rodam a migração numa raiz falsa. As de usuário
  não precisam: seguem `$HOME` e os `XDG_*`. Comando que mexe no sistema por
  conta própria (`systemctl`, `mkinitcpio`, `limine-update`) não recebe a
  raiz: nos testes é um dublê.
- **Teste**: `tests/migrations/system/<nome>.test.sh` (ou `user/`), escrito
  antes da migração: monta o "antes", roda e confere o "depois"; roda de novo
  e nada muda; num sistema sem nada a fazer, não quebra. O
  `tests/migrations/regras.test.sh` confere as regras desta página em todas.
- **Idempotência**: confira antes de mudar (`grep -q`, `[[ -e … ]]`,
  `pacman -Q`). A marca só é gravada depois que o script termina; uma rodada
  interrompida no meio repete a migração inteira.
- **Código de saída**: 0 marca como feita, para sempre. Qualquer outro valor
  para o `mk-update` ali mesmo (o resumo fica "parou em: ajustes"), não marca,
  e o que vinha depois não roda nessa vez: os pacotes e o AUR ou, numa
  migração com `# pacotes:` (que roda depois do pacman), só o AUR. Saia com
  erro só
  quando tentar de novo na próxima rodada faz sentido; para o que não tem
  conserto automático, avise em stderr e saia com 0.
- **Pacotes** (só em `system/`): a linha `# pacotes: a b c`, no começo da
  linha, com nomes dos repositórios oficiais (entram num `pacman -Su`; AUR não
  serve). Nome que não existe mais nos repositórios é tirado do pedido com um
  aviso, e a migração roda mesmo assim. Só é aceito nome que casa com
  `^[a-z0-9@._+-]+$`; o que não casa é descartado em silêncio. Em `user/` a
  linha é ignorada.
- **Sem perguntas**: a migração roda no meio da atualização, sem ninguém
  esperando uma pergunta dela.

E, fora do script:

- **O instalador faz a mesma mudança.** `installer/mk-install` marca sozinho
  todas as migrações como feitas (as de sistema em `/var/lib/mk/migrations`, as
  de usuário em `/etc/skel/.local/state/mk/migrations`), sem rodar nenhuma. Não
  há lista para editar, mas se o instalador não fizer o que a migração faz, a
  instalação nova fica sem a mudança. `bin/new-user` também marca as de
  usuário sem rodar, mas só quando ele mesmo criou o usuário; `bin/try-session
  install`, só na primeira instalação (quando `~/.config/mk/shell` ainda não
  existia).
- **Mudança que o usuário vê tem o par no skel.** A migração de usuário só
  alcança quem roda o `mk-update`. Para quem for criado depois, escreva também
  uma de sistema que ajuste `/etc/skel` (veja `20261120-terminal-skel.sh` e
  `20261125-apps-padrao-skel.sh`). O `/etc/skel` de uma máquina já instalada só
  muda por migração: o `mk-update` não o atualiza. As marcas de usuário no skel
  são as do dia da instalação, então uma home nova ainda recebe, no primeiro
  `mk-update`, as migrações de usuário lançadas depois; se o skel já entrega a
  mudança pronta e a migração não deve rodar ali, a de sistema grava a marca no
  skel (veja `20261125-bem-vindo-skel.sh`).

O caminho completo da atualização está em
[docs/atualizacoes.md](../docs/atualizacoes.md).
