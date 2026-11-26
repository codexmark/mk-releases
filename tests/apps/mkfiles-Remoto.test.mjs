// apps/mkfiles/qml/Remoto.js: os lugares de rede da janela do MkFiles: o
// endereço montado do formulário, o caminho que mora num remoto (migalhas,
// endereço), as linhas da barra lateral e os textos das perguntas do gio.
import { readFileSync } from "node:fs";
import { t, eq, igual } from "../js/harness.mjs";
import { load, REPO } from "../js/load.mjs";

const R = load(REPO + "/apps/mkfiles/qml/Remoto.js");
const gvfs = "/run/user/1000/gvfs";
const remotos = [
    { uri: "sftp://ana@servidor:2222/", raiz: gvfs + "/sftp:host=servidor,port=2222,user=ana", nome: "servidor", protocolo: "sftp", host: "servidor" },
    { uri: "smb://nas/fotos", raiz: gvfs + "/smb-share:server=nas,share=fotos", nome: "fotos em nas", protocolo: "smb", host: "nas" },
    { uri: null, raiz: gvfs + "/mtp:host=Telefone", nome: "Telefone", protocolo: "mtp", host: "Telefone" }
];

t("toda função do arquivo tem caso aqui", () => {
    const cobertas = ["tipo", "codificar", "codificarPasta", "erroDoHost", "erroDaPorta", "montarUri", "erroDoEndereco", "uriDoHost",
        "detalheDoHost", "dentroDe", "eRemoto", "doRemoto", "migalhas", "endereco", "mesmoLugar", "montadoEm", "icone", "linhas",
        "pergunta", "erroDeMontar", "caiu", "cabeMiniatura", "proximoLimite"];
    const funcoes = Object.keys(R).filter((k) => typeof R[k] === "function");
    igual(funcoes.filter((f) => !cobertas.includes(f)).sort(), [], "função sem teste");
    igual(cobertas.filter((f) => !funcoes.includes(f)), [], "função que sumiu");
});

t("o arquivo não guarda nem escreve senha: nenhum campo de senha no endereço, nada de log", () => {
    const fonte = readFileSync(REPO + "/apps/mkfiles/qml/Remoto.js", "utf8").split("\n").filter((l) => !/^\s*\/\//.test(l)).join("\n");
    eq(/console\.|f\.senha|\.senha\b|password/i.test(fonte), false);
});

t("tipo: os cinco do formulário, com a porta padrão; o desconhecido cai no sftp", () => {
    igual(R.TIPOS.map((x) => x.v), ["sftp", "smb", "ftp", "dav", "davs"]);
    eq(R.tipo("smb").porta, 445);
    eq(R.tipo("inventado").v, "sftp");
});

t("codificar e codificarPasta: espaço, acento, # e ? viram %XX; as barras ficam", () => {
    eq(R.codificar("ana maria"), "ana%20maria");
    eq(R.codificar("a@b:c/d"), "a%40b%3Ac%2Fd");
    eq(R.codificar("(x)!*'"), "%28x%29%21%2A%27");
    eq(R.codificarPasta("Meus Documentos/ação #1"), "/Meus%20Documentos/a%C3%A7%C3%A3o%20%231");
    eq(R.codificarPasta(""), "/");
    eq(R.codificarPasta("//a/./b/"), "/a/b");
});

t("erroDoHost e erroDaPorta", () => {
    eq(R.erroDoHost("servidor.local"), "");
    eq(R.erroDoHost("192.168.0.10"), "");
    eq(R.erroDoHost("  "), "escreva o servidor");
    for (const ruim of ["a b", "a/b", "ana@host", "-oProxyCommand=x", "a\\b", "a?b"])
        eq(R.erroDoHost(ruim) !== "", true, ruim);
    eq(R.erroDaPorta(""), "");
    eq(R.erroDaPorta("2222"), "");
    eq(R.erroDaPorta(22), "");
    for (const ruim of ["0", "70000", "abc", "22;x", "-1"])
        eq(R.erroDaPorta(ruim) !== "", true, ruim);
});

t("montarUri: sftp com usuário, porta e pasta; a porta padrão some", () => {
    eq(R.montarUri({ tipo: "sftp", host: "servidor", porta: "2222", usuario: "ana", pasta: "/srv/meus dados" }), "sftp://ana@servidor:2222/srv/meus%20dados");
    eq(R.montarUri({ tipo: "sftp", host: "servidor", porta: "22", usuario: "", pasta: "" }), "sftp://servidor/");
    eq(R.montarUri({ tipo: "ftp", host: " ftp.exemplo.org ", pasta: "pub" }), "ftp://ftp.exemplo.org/pub");
    eq(R.montarUri({ tipo: "davs", host: "nuvem", usuario: "ana@casa", pasta: "remote.php/dav" }), "davs://ana%40casa@nuvem/remote.php/dav");
    eq(R.montarUri({ tipo: "sftp", host: "fe80::1", usuario: "ana" }), "sftp://ana@[fe80::1]/");
});

t("montarUri: no SMB o domínio vai antes do usuário e a primeira pasta é o compartilhamento", () => {
    eq(R.montarUri({ tipo: "smb", host: "nas", usuario: "ana", dominio: "CASA", pasta: "fotos/2024" }), "smb://CASA;ana@nas/fotos/2024");
    eq(R.montarUri({ tipo: "smb", host: "nas", dominio: "CASA", pasta: "fotos" }), "smb://nas/fotos", "domínio sem usuário não entra");
    // Fora do SMB o domínio é ignorado.
    eq(R.montarUri({ tipo: "sftp", host: "x", usuario: "ana", dominio: "CASA" }), "sftp://ana@x/");
});

t("montarUri: host ou porta que não servem não viram endereço", () => {
    eq(R.montarUri({ tipo: "sftp", host: "" }), null);
    eq(R.montarUri({ tipo: "sftp", host: "-oProxyCommand=x" }), null);
    eq(R.montarUri({ tipo: "sftp", host: "x", porta: "99999" }), null);
    eq(R.montarUri(null), null);
});

t("erroDoEndereco: só os tipos que o motor monta, com servidor, sem espaço", () => {
    for (const bom of ["sftp://ana@host:2222/pasta", "smb://nas/fotos", "ftp://x", "davs://x/y", "mtp://[usb:001,004]/"])
        eq(R.erroDoEndereco(bom), "", bom);
    eq(R.erroDoEndereco(""), "escreva o endereço");
    eq(R.erroDoEndereco("http://x") !== "", true);
    eq(R.erroDoEndereco("file:///etc") !== "", true);
    eq(R.erroDoEndereco("servidor/pasta") !== "", true);
    eq(R.erroDoEndereco("sftp://") !== "", true);
    eq(R.erroDoEndereco("sftp://ana@") !== "", true);
    eq(R.erroDoEndereco("sftp://a b/c") !== "", true);
    eq(R.erroDoEndereco("-u sftp://x") !== "", true);
});

t("uriDoHost e detalheDoHost: um host do ~/.ssh/config vira sftp://apelido/", () => {
    eq(R.uriDoHost({ apelido: "casa", host: "192.168.0.2", usuario: "ana", porta: 2222 }), "sftp://casa/");
    eq(R.uriDoHost({}), null);
    eq(R.detalheDoHost({ apelido: "casa", host: "192.168.0.2", usuario: "ana", porta: 2222 }), "ana@192.168.0.2:2222");
    eq(R.detalheDoHost({ apelido: "casa", host: null, usuario: null, porta: null }), "casa");
    eq(R.detalheDoHost({ apelido: "x", host: "h", usuario: null, porta: 22 }), "h");
    eq(R.detalheDoHost(null), "");
});

t("dentroDe e eRemoto: o que mora no ponto do gvfs", () => {
    eq(R.dentroDe("/a", "/a/b"), true);
    eq(R.dentroDe("/a", "/ab"), false);
    eq(R.dentroDe("", "/a"), false);
    eq(R.eRemoto(gvfs + "/sftp:host=x/pasta", gvfs), true);
    eq(R.eRemoto(gvfs, gvfs), false, "a raiz do gvfs em si não é um remoto");
    eq(R.eRemoto("/home/ana", gvfs), false);
    eq(R.eRemoto("/home/ana", null), false);
    eq(R.eRemoto("", gvfs), false);
});

t("doRemoto: o remoto montado em que o caminho mora", () => {
    eq(R.doRemoto(remotos[0].raiz + "/srv/x", remotos).nome, "servidor");
    eq(R.doRemoto(remotos[1].raiz, remotos).nome, "fotos em nas");
    eq(R.doRemoto("/home/ana", remotos), null);
    eq(R.doRemoto(gvfs + "/sftp:host=servidor", remotos), null, "prefixo de nome não é dentro");
    eq(R.doRemoto("/x", null), null);
});

t("migalhas: servidor › pasta, em vez do caminho do gvfs", () => {
    igual(R.migalhas(remotos[0].raiz + "/srv/meus dados", remotos), [
        { nome: "servidor", caminho: remotos[0].raiz },
        { nome: "srv", caminho: remotos[0].raiz + "/srv" },
        { nome: "meus dados", caminho: remotos[0].raiz + "/srv/meus dados" }
    ]);
    igual(R.migalhas(remotos[1].raiz, remotos), [{ nome: "fotos em nas", caminho: remotos[1].raiz }]);
    eq(R.migalhas("/home/ana/Documentos", remotos), null);
});

t("endereco: a URI do remoto mais o resto codificado; sem URI conhecida, null", () => {
    eq(R.endereco(remotos[0].raiz + "/srv/meus dados/ação #1.txt", remotos), "sftp://ana@servidor:2222/srv/meus%20dados/a%C3%A7%C3%A3o%20%231.txt");
    eq(R.endereco(remotos[0].raiz, remotos), "sftp://ana@servidor:2222/");
    eq(R.endereco(remotos[1].raiz + "/2024", remotos), "smb://nas/fotos/2024");
    eq(R.endereco(remotos[2].raiz + "/DCIM", remotos), null);
    eq(R.endereco("/home/ana", remotos), null);
});

t("mesmoLugar: com e sem a barra do fim, sem diferenciar maiúsculas no servidor", () => {
    eq(R.mesmoLugar("smb://NAS/fotos/", "smb://nas/fotos"), true);
    eq(R.mesmoLugar("smb://nas/Fotos", "smb://nas/fotos"), false, "a pasta diferencia");
    eq(R.mesmoLugar("sftp://a/", "sftp://b/"), false);
    eq(R.mesmoLugar(null, null), false);
});

t("montadoEm: um favorito sftp://… já montado vira o caminho dentro da raiz", () => {
    igual(R.montadoEm("sftp://ana@servidor:2222/srv/meus%20dados", remotos), { remoto: remotos[0], caminho: remotos[0].raiz + "/srv/meus dados" });
    eq(R.montadoEm("smb://nas/fotos/", remotos).caminho, remotos[1].raiz);
    eq(R.montadoEm("SMB://NAS/fotos/2024", remotos).caminho, remotos[1].raiz + "/2024");
    eq(R.montadoEm("sftp://outro/", remotos), null);
    eq(R.montadoEm("smb://nas/fotosdafamilia", remotos), null, "prefixo de nome não é o mesmo compartilhamento");
    // Um endereço que tenta sair da raiz não vira caminho.
    eq(R.montadoEm("smb://nas/fotos/%2E%2E/x", remotos), null);
    eq(R.montadoEm("smb://nas/fotos/a%2Fb", remotos), null);
    eq(R.montadoEm("smb://nas/fotos/%zz", remotos), null);
});

t("linhas: os remotos montados e, no fim, conectar", () => {
    const l = R.linhas(remotos);
    igual(l.map((x) => x.tipo), ["remoto", "remoto", "remoto", "conectar"]);
    eq(l[0].caminho, remotos[0].raiz);
    eq(l[0].uri, "sftp://ana@servidor:2222/");
    eq(l[2].icone, "phone");
    eq(l[2].uri, null);
    eq(l[3].caminho, null);
    igual(R.linhas([]).map((x) => x.tipo), ["conectar"]);
    igual(R.linhas(null).map((x) => x.tipo), ["conectar"]);
    eq(R.icone("sftp"), "folder-remote");
    eq(R.icone("gphoto2"), "phone");
});

t("pergunta: a senha é o único campo oculto; confirmar é sim ou não", () => {
    igual(R.pergunta({ tipo: "senha" }), { titulo: "Senha", dica: "senha", oculto: true, simNao: false });
    eq(R.pergunta({ tipo: "senha", de_novo: true }).titulo, "A senha não serviu. Tente de novo");
    eq(R.pergunta({ tipo: "usuario" }).oculto, false);
    eq(R.pergunta({ tipo: "dominio" }).oculto, false);
    eq(R.pergunta({ tipo: "confirmar" }).simNao, true);
    eq(R.pergunta({ tipo: "outro" }).simNao, false);
    eq(R.pergunta(null).oculto, false);
});

t("erroDeMontar: uma frase por motivo; o resto é a frase do motor", () => {
    eq(R.erroDeMontar({ motivo: "senha", erro: "x" }), "O usuário ou a senha não serviram.");
    eq(R.erroDeMontar({ motivo: "tempo" }), "O servidor não respondeu a tempo.");
    eq(R.erroDeMontar({ motivo: "recusado" }), "O servidor recusou a conexão.");
    eq(R.erroDeMontar({ motivo: "falta", erro: "falta o gvfs-smb" }), "falta o gvfs-smb");
    eq(R.erroDeMontar({ motivo: "endereco", erro: "endereço torto" }), "endereço torto");
    eq(R.erroDeMontar({ motivo: "erro", erro: "deu ruim" }), "deu ruim");
    eq(R.erroDeMontar({}), "Não deu para conectar.");
});

t("caiu: conexão perdida, ou remoto que demorou ou sumiu; uma pasta local que sumiu não", () => {
    const c = remotos[0].raiz + "/srv";
    eq(R.caiu("conexao", c, gvfs), true);
    eq(R.caiu("conexao", "/home/ana", gvfs), true);
    eq(R.caiu("tempo", c, gvfs), true);
    eq(R.caiu("nao-existe", c, gvfs), true);
    eq(R.caiu("tempo", "/home/ana/x", gvfs), false);
    eq(R.caiu("nao-existe", "/home/ana/x", gvfs), false);
    eq(R.caiu("sem-permissao", c, gvfs), false);
});

t("cabeMiniatura e proximoLimite: em remoto só os pequenos, e só se a pessoa deixa", () => {
    eq(R.cabeMiniatura({ tamanho: 1048576 }, { miniaturas_remotas: true, miniatura_remota_mb: 5 }), true);
    eq(R.cabeMiniatura({ tamanho: 5 * 1048576 + 1 }, { miniaturas_remotas: true, miniatura_remota_mb: 5 }), false);
    eq(R.cabeMiniatura({ tamanho: 10 }, { miniaturas_remotas: false, miniatura_remota_mb: 5 }), false);
    eq(R.cabeMiniatura({ tamanho: 4 * 1048576 }, {}), true, "o padrão é ligado, 5 MB");
    eq(R.cabeMiniatura({ tamanho: 6 * 1048576 }, { miniatura_remota_mb: 999 }), false, "valor torto cai no padrão");
    eq(R.cabeMiniatura(null, {}), false);
    igual([1, 5, 20, 100, 7].map(R.proximoLimite), [5, 20, 100, 1, 1]);
});
