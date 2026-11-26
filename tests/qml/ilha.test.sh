#!/usr/bin/env bash
# A ilha não estica nem encolhe quando um número muda de tamanho: cada leitura
# reserva o lugar do seu texto mais largo (IslandItem.reserve, Island.js).
. "$(dirname "$0")/../lib.sh"

item=$(<"$REPO/shell/mare/IslandItem.qml")
bar=$(<"$REPO/shell/mare/Bar.qml")

t "o item da ilha aceita um texto de reserva e mede a largura por ele"
has "$item" 'property string reserve: ""'
has "$item" 'TextMetrics {'
has "$item" 'text: item.reserve'
has "$item" 'width: Math.max(implicitWidth, Math.ceil(reserved.advanceWidth))'

t "sem reserva o item fica como era: a largura do próprio texto"
has "$item" 'implicitWidth: row.implicitWidth + (plate ? 16 : 10)'

t "a reserva usa a mesma fonte do texto (família, tamanho e algarismos de largura igual)"
like "$item" 'TextMetrics \{[^}]*font\.family: item\.textFamily[^}]*font\.pixelSize: item\.textSize[^}]*font\.features: \(\{ "tnum": 1 \}\)'

t "as leituras da ilha reservam a largura: temperatura, rede, processador, memória e bateria"
has "$bar" 'import "Island.js" as Island'
has "$bar" 'text: Island.degrees(Stats.temp)'
has "$bar" 'reserve: Island.reserve.degrees'
has "$bar" 'text: Island.rate(Stats.rxRate)'
has "$bar" 'reserve: Island.reserve.rate'
has "$bar" 'text: Island.percent(Stats.cpu)'
has "$bar" 'text: Island.percent(Stats.mem)'
has "$bar" 'text: Island.percent(bar.batteryPct / 100)'
eq 3 "$(grep -c 'reserve: Island.reserve.percent' <<<"$bar")"

t "nenhuma leitura da ilha monta o texto à mão (o que mudava de largura a cada dígito)"
hasnt "$bar" 'Math.round(Stats.cpu * 100) + "%"'
hasnt "$bar" 'Math.round(Stats.mem * 100) + "%"'
hasnt "$bar" 'Stats.rate(Stats.rxRate).replace'
hasnt "$bar" 'bar.batteryPct + "%"'
