# pc-resource-monitor
my take on htop with the help of claude code

a retro terminal monitor for windows. shows cpu, temperature, ram, wifi, disk and the gpu engines task manager has (3d, copy, video decode, video processing), plus a process list.

![screenshot](docs/screenshot.png)

## run it

grab `pc-resource-monitor.exe` from [releases](https://github.com/glimmercharger/pc-resource-monitor/releases) and run it. looks best in windows terminal.

| key | does |
|---|---|
| F1 / h | help |
| F2 / t | change theme (htop, phosphor, amber, c64) |
| F3 / g | graphs on/off |
| F4 / p | process list on/off |
| F6 / s | sort by cpu, mem, pid or name |
| ↑ ↓ | scroll the process list |
| F10 / q | quit |

`-theme amber` starts on a theme, `-delay 2s` updates less often.

## build

needs go 1.27

```
go build -ldflags "-s -w" -o pc-resource-monitor.exe .
```
