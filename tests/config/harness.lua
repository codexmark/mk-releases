-- Mini-arnês dos testes da configuração do Hyprland (tests/config/*.test.lua).
--
--   local T = dofile((arg[0]:match("^(.*)/") or ".") .. "/harness.lua")
--
--   T.t("o que se espera", function() ... end)   um caso; erro dentro dele reprova
--   T.eq(esperado, obtido [, o quê])             iguais (tabelas: por conteúdo)
--   T.ok(valor [, o quê])                        verdadeiro
--   T.has(texto, trecho) / T.hasnt(texto, trecho)
--   T.finish()                                   apaga a caixa e sai (1 se algo falhou)
--
-- A configuração roda numa caixa de areia: T.session{...} copia config/hypr
-- para uma raiz temporária (<raiz>/hypr, como em ~/.config/mk), escreve os
-- arquivos pedidos e carrega com o `hl` falso (tests/config/hl.lua). Enquanto
-- ela roda, os.getenv só vê o que a sessão disser, io.open não sai da caixa
-- (um caminho absoluto de fora vira <sessão>/sys/<caminho>) e o print fica
-- anotado. Nada do Hyprland de verdade, nem da sessão de quem roda, é tocado.
--
--   local s = T.session({
--       files = { ["ajustes.lua"] = "hl.config({ ... })" },  na raiz (<raiz>/...)
--       sys = { ["/etc/x"] = "" },                            arquivos "do sistema"
--       env = { XDG_RUNTIME_DIR = "/run/user/1000" },
--       refuse = { ["SUPER + X"] = "erro" },                  teclas que o hl.bind recusa
--       only = "mare.env",                                    só um módulo, não o hyprland.lua
--   })
--   s.ok, s.err      carregou? com que erro?
--   s.log            o que a configuração pediu (veja hl.lua)
--   s.prints         as linhas de print, juntas
--   s.root           a raiz, com a barra no fim
--   s.binds          o módulo mare.binds, quando foi carregado
--   s:read(nome), s:exists(nome), s:fire(evento), s:live() (binds não removidos)
local T = { fail = 0, n = 0 }

local real = { open = io.open, getenv = os.getenv, print = print, path = package.path }
T.dir = (arg and arg[0] or ""):match("^(.*)/[^/]*$") or "."
T.repo = T.dir .. "/../.."
package.path = T.dir .. "/?.lua;" .. package.path
local fake = require("hl")

local function out(line)
    io.stdout:write(line, "\n")
end

-- Um valor por extenso, com as chaves em ordem, para comparar e para mostrar.
local function show(v, depth)
    depth = depth or 0
    if type(v) == "string" then
        return string.format("%q", v)
    elseif type(v) ~= "table" then
        return tostring(v)
    elseif depth > 8 then
        return "{...}"
    end
    local keys = {}
    for k in pairs(v) do
        keys[#keys + 1] = k
    end
    table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
    local parts = {}
    for _, k in ipairs(keys) do
        if type(v[k]) ~= "function" then
            parts[#parts + 1] = "[" .. show(k) .. "]=" .. show(v[k], depth + 1)
        end
    end
    return "{" .. table.concat(parts, ",") .. "}"
end
T.show = show

local function flunk(what, ...)
    local lines = { ... }
    error(what .. "\n" .. table.concat(lines, "\n"), 0)
end

function T.eq(want, got, what)
    if show(want) ~= show(got) then
        flunk((what or "eq") .. ": esperado", "    " .. show(want), "  obtido", "    " .. show(got))
    end
end
function T.ok(value, what)
    if not value then
        flunk((what or "ok") .. ": devia ser verdadeiro")
    end
end
function T.has(text, piece, what)
    if not tostring(text):find(piece, 1, true) then
        flunk((what or "has") .. ": não achei", "    " .. piece, "  em", "    " .. tostring(text):sub(1, 600))
    end
end
function T.hasnt(text, piece, what)
    if tostring(text):find(piece, 1, true) then
        flunk((what or "hasnt") .. ": não devia ter", "    " .. piece, "  em", "    " .. tostring(text):sub(1, 600))
    end
end

function T.t(name, fn)
    T.n = T.n + 1
    local ok, err = xpcall(fn, function(e)
        return type(e) == "string" and e or debug.traceback(tostring(e), 2)
    end)
    if ok then
        out("ok - " .. name)
    else
        T.fail = T.fail + 1
        out("not ok - " .. name)
        for line in tostring(err):gmatch("[^\n]+") do
            io.stderr:write("    # ", line, "\n")
        end
    end
end

-- ── a caixa de areia ──────────────────────────────────────────────────
local function sh(cmd)
    local ok = os.execute(cmd)
    if ok ~= true and ok ~= 0 then
        error("falhou: " .. cmd)
    end
end
local function quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

local box, sessions = nil, 0
local function sandbox()
    if not box then
        local p = io.popen("mktemp -d \"${TMPDIR:-/tmp}/mk-lua.XXXXXX\"")
        box = p:read("l")
        p:close()
        assert(box and box:match("/mk%-lua%.[%w]+$"), "mktemp falhou")
    end
    return box
end

function T.write(path, text)
    sh("mkdir -p " .. quote(path:match("^(.*)/[^/]*$")))
    local f = assert(real.open(path, "w"))
    f:write(text)
    f:close()
end
function T.slurp(path)
    local f = real.open(path)
    if not f then
        return nil
    end
    local text = f:read("a")
    f:close()
    return text
end

local Session = {}
Session.__index = Session
function Session:read(name) return T.slurp(self.root .. name) end
function Session:exists(name) return T.slurp(self.root .. name) ~= nil end
function Session:fire(event)
    for _, fn in ipairs(self.log.events[event] or {}) do
        fn()
    end
end
function Session:live()
    local list = {}
    for _, b in ipairs(self.log.binds) do
        if not b.removed then
            list[#list + 1] = b
        end
    end
    return list
end
-- O bind vivo de um combo (comparado sem ligar para maiúsculas e espaços).
function Session:bind(keys)
    local function norm(k) return (k:lower():gsub("%s+", "")) end
    for _, b in ipairs(self:live()) do
        if norm(b.keys) == norm(keys) then
            return b
        end
    end
end

function T.session(opts)
    opts = opts or {}
    sessions = sessions + 1
    local base = sandbox() .. "/s" .. sessions
    local s = setmetatable({ root = base .. "/mk/", sys = base .. "/sys", lines = {} }, Session)
    sh("mkdir -p " .. quote(s.root) .. " " .. quote(s.sys) .. " && cp -r " .. quote(T.repo .. "/config/hypr") .. " " .. quote(s.root .. "hypr"))
    for name, text in pairs(opts.files or {}) do
        T.write(s.root .. name, text)
    end
    for name, text in pairs(opts.sys or {}) do
        T.write(s.sys .. name, text)
    end

    local hl, log = fake.new()
    s.log = log
    for keys, how in pairs(opts.refuse or {}) do
        log.refuse[keys] = how
    end
    log.no_timer = opts.no_timer

    -- O mundo que a configuração enxerga.
    local env = { HOME = base .. "/home", XDG_RUNTIME_DIR = "/run/user/1000" }
    for k, v in pairs(opts.env or {}) do
        env[k] = v or nil
    end
    _G.hl = hl
    _G.print = function(...)
        local parts = {}
        for i = 1, select("#", ...) do
            parts[#parts + 1] = tostring((select(i, ...)))
        end
        s.lines[#s.lines + 1] = table.concat(parts, "\t")
    end
    os.getenv = function(name) return env[name] end
    io.open = function(path, mode)
        if path:sub(1, #base) ~= base then
            path = s.sys .. (path:sub(1, 1) == "/" and "" or "/") .. path
        end
        return real.open(path, mode)
    end
    -- Só os módulos da caixa: nada de mare.* já carregado, nem do sistema.
    for name in pairs(package.loaded) do
        if name:match("^mare%.") or name == "monitors" or name == "user" then
            package.loaded[name] = nil
        end
    end
    package.path = base .. "/nada/?.lua"

    if opts.only then
        package.path = s.root .. "hypr/?.lua"
        s.ok, s.err = pcall(require, opts.only)
        s.module = s.ok and s.err or nil
    else
        s.ok, s.err = pcall(dofile, s.root .. "hypr/hyprland.lua")
    end
    s.binds = package.loaded["mare.binds"]
    return setmetatable(s, {
        __index = function(self, key)
            if key == "prints" then
                return table.concat(rawget(self, "lines"), "\n")
            end
            return Session[key]
        end,
    })
end

function T.finish()
    io.open, os.getenv, _G.print, package.path = real.open, real.getenv, real.print, real.path
    if box and box:match("/mk%-lua%.[%w]+$") then
        os.execute("rm -rf " .. quote(box))
    end
    if T.n == 0 then
        out("not ok - nenhum caso")
        os.exit(1)
    end
    os.exit(T.fail == 0 and 0 or 1)
end

return T
