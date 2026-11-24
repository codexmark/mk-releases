---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - atalhos
  - ajustes
fontes:
  - "config/hypr/mare/binds.lua"
  - "shell/mare/SettingsKeybindsPage.qml"
  - "shell/mare/Keybinds.js"
  - "shell/mare/README.md"
---

# Atalhos do usuário

O que o usuário troca em Ajustes › Atalhos fica em `<raiz>/atalhos.lua`, um arquivo **só de dados** que o `binds.lua` lê antes de criar os binds.

## Para quê

Trocar a tecla de qualquer ação não fixa, com efeito imediato e sem editar a config.

## Como funciona

- **Formato**: `return { ["win.close"] = "SUPER + W" }`. Uma string vazia desliga o atalho.
- **Origem**: a chave `keybinds` do `ajustes.json`; a página grava o `atalhos.lua` a partir dela.
- **Leitura** (`M.saved()`): `loadfile(file, "t", {})`, isto é, só texto e com **ambiente vazio**; só vale se devolver uma tabela. Faltando ou quebrado, valem os de fábrica.
- **No login**: `M.apply(M.saved())` no fim do `binds.lua`. Cada ação nasce uma vez, já no combo certo.
- **Na sessão aberta**: a página chama `hyprctl eval 'require("mare.binds").apply({...})'`, que recria só os binds deste módulo, sem recarregar o resto da config. Depois confere com `hyprctl -j binds` se o Hyprland ficou com o que foi pedido.
- **Se o `eval` não responde `ok`** (sessão cuja config ainda não tem o `apply()`): `hyprctl reload`; se isso também falhar, o estado fica "gravado; vale no próximo login".
- **Editor**: os modificadores são botões e a tecla é um campo (pode ser digitada pelo nome: `k`, `f5`, `enter`, `print`); o "capturar" usa o `suspend()` do `binds.lua`.

Regras que o `apply()` impõe:

- ações `fixed` ignoram a troca;
- ações `required` (emergência) nunca ficam sem atalho nem vão para uma tecla que nem todo teclado tem, e não podem pedir o combo de fábrica de quem não foi mexido;
- quem pede um combo ocupado ou inválido volta para o de fábrica;
- `M.current` guarda o combo com que cada ação acabou, `M.unbound` quem ficou sem nenhum sem ter pedido.

## Por quê assim

- **Arquivo de dados, não de código**: uma versão anterior carregava o `atalhos.lua` junto com os outros arquivos gerados. O commit da revisão registra a mudança ("atalhos.lua não é mais executado como código") e o `binds.lua` descreve o resultado: o arquivo "roda sem acesso a nada (ambiente vazio) e só vale se devolver uma tabela". Um arquivo que o shell escreve deixa de ter o poder da config inteira.
- **`eval` do `apply`, não `reload`**: o `apply` "tira os binds deste módulo e os cria de novo, sem recarregar o resto da config".
- **Conferir depois**: o Hyprland pode recusar uma tecla em silêncio; a página só diz "valeu" depois de ler os binds de volta.
- **Dois nomes de arquivo parecidos**: `atalhos.lua` (português, na raiz, gerado) é isto; não há `binds.lua` do usuário.

## Onde está no código

- `config/hypr/mare/binds.lua`: `M.saved`, `M.apply`, `M.suspend`
- `shell/mare/SettingsKeybindsPage.qml`
- `shell/mare/Keybinds.js`: `validate`, `clean`, `entries`

## Ligações

Depende de: [[Atalhos]], [[Pasta do usuário]]. Usado por: [[Páginas dos Ajustes]], [[Ordem de carga]].
