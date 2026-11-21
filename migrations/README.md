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
