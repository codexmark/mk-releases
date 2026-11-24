-- O `hl` falso: a API do Hyprland que a configuração do Mk usa, sem Hyprland
-- nenhum. Nada acontece; cada pedido fica anotado para o teste conferir.
--
--   local hl, log = require("hl").new()
--
--   log.calls        tudo, na ordem: { fn = "config", args = { ... } }
--   log.config       hl.config(...) fundidos numa tabela só (log.config.general.gaps_in)
--   log.env          { NOME = "valor" }
--   log.monitors     os argumentos de cada hl.monitor
--   log.curves       { nome = { ... } }
--   log.animations   os argumentos de cada hl.animation
--   log.rules        os argumentos de cada hl.window_rule
--   log.execs        os comandos de hl.exec_cmd
--   log.events       { ["hyprland.start"] = { fn, ... } }
--   log.binds        cada hl.bind: { keys, action, opts, removed, enabled }
--   log.timers       cada hl.timer: { fn, opts, enabled }
--   log.refuse       teclas que o hl.bind recusa: { ["SUPER + X"] = "erro" | "nil" }
--
-- Um despacho (hl.dsp.window.close(), hl.dsp.focus({ ... })) vira
-- { dsp = "window.close", args = { ... } }. Chamar uma função que não está na
-- lista abaixo é erro, como no Hyprland: a configuração não pode depender de
-- algo que este dublê não conhece sem o teste saber.
local M = {}

local function merge(into, from)
    for k, v in pairs(from) do
        if type(v) == "table" and type(into[k]) == "table" then
            merge(into[k], v)
        elseif type(v) == "table" then
            into[k] = {}
            merge(into[k], v)
        else
            into[k] = v
        end
    end
end

-- hl.dsp.<caminho>(...): só os despachos que o Mk usa.
local dispatchers = {
    ["exec_cmd"] = true, ["exit"] = true, ["focus"] = true, ["dpms"] = true,
    ["window.close"] = true, ["window.fullscreen"] = true, ["window.float"] = true,
    ["window.move"] = true, ["window.resize"] = true, ["window.drag"] = true,
}

local function dsp(path)
    return setmetatable({}, {
        __index = function(_, key)
            return dsp(path == "" and key or (path .. "." .. key))
        end,
        __call = function(_, ...)
            if not dispatchers[path] then
                error("hl.dsp." .. path .. " não existe no hl falso (tests/config/hl.lua)", 2)
            end
            return { dsp = path, args = { ... } }
        end,
    })
end

function M.new()
    local log = {
        calls = {}, config = {}, env = {}, monitors = {}, curves = {}, animations = {},
        rules = {}, execs = {}, events = {}, binds = {}, timers = {}, refuse = {},
    }
    local function note(fn, ...)
        log.calls[#log.calls + 1] = { fn = fn, args = { ... } }
    end
    local hl = { dsp = dsp("") }

    function hl.config(t)
        note("config", t)
        assert(type(t) == "table", "hl.config espera uma tabela")
        merge(log.config, t)
    end
    function hl.env(name, value)
        note("env", name, value)
        assert(type(name) == "string" and type(value) == "string", "hl.env espera nome e valor em texto")
        log.env[name] = value
    end
    function hl.monitor(t)
        note("monitor", t)
        assert(type(t) == "table", "hl.monitor espera uma tabela")
        log.monitors[#log.monitors + 1] = t
    end
    function hl.curve(name, t)
        note("curve", name, t)
        log.curves[name] = t
    end
    function hl.animation(t)
        note("animation", t)
        assert(type(t) == "table" and type(t.leaf) == "string", "hl.animation espera { leaf = ... }")
        log.animations[#log.animations + 1] = t
    end
    function hl.window_rule(t)
        note("window_rule", t)
        assert(type(t) == "table", "hl.window_rule espera uma tabela")
        log.rules[#log.rules + 1] = t
    end
    function hl.exec_cmd(cmd)
        note("exec_cmd", cmd)
        assert(type(cmd) == "string", "hl.exec_cmd espera um comando em texto")
        log.execs[#log.execs + 1] = cmd
    end
    function hl.on(event, fn)
        note("on", event, fn)
        assert(type(fn) == "function", "hl.on espera uma função")
        log.events[event] = log.events[event] or {}
        table.insert(log.events[event], fn)
    end
    function hl.get_monitors()
        note("get_monitors")
        return {}
    end
    function hl.bind(keys, action, opts)
        note("bind", keys, action, opts)
        local refuse = log.refuse[keys]
        if refuse == "erro" then
            error("tecla desconhecida: " .. tostring(keys))
        elseif refuse then
            return nil
        end
        assert(type(keys) == "string", "hl.bind espera o combo em texto")
        assert(type(action) == "table" or type(action) == "function", "hl.bind espera um despacho")
        local bind = { keys = keys, action = action, opts = opts or {}, removed = false, enabled = true }
        function bind.remove(self)
            self.removed = true
        end
        function bind.set_enabled(self, on)
            self.enabled = on and true or false
        end
        log.binds[#log.binds + 1] = bind
        return bind
    end
    function hl.timer(fn, opts)
        note("timer", fn, opts)
        if log.no_timer then
            error("sem relógio")
        end
        local timer = { fn = fn, opts = opts or {}, enabled = true }
        function timer.set_enabled(self, on)
            self.enabled = on and true or false
        end
        log.timers[#log.timers + 1] = timer
        return timer
    end

    setmetatable(hl, {
        __index = function(_, key)
            error("hl." .. tostring(key) .. " não existe no hl falso (tests/config/hl.lua)", 2)
        end,
    })
    return hl, log
end

return M
