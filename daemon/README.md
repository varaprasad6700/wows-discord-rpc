# World of Warships Discord Presence Daemon

Standalone Windows daemon in Go that monitors World of Warships, deploys telemetry mod scripts, resolves ship metadata via caching backend, and streams Rich Presence via Discord IPC (`\\.\pipe\discord-ipc-0`).
