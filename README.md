# Vzlom Poche

WebSocket relay & microservice proxy pour Agora/Bridge.

## Architecture

```
Mobile Client ←→ Vzlom Poche (port 3457) ←→ Agora Admin / Bridge (port 3456)
                        ↕
                  WebSocket relay
```

## Installation

```bash
# Go 1.22+ requis
go build -o vzlom-poche .
```

## Lancement

```bash
# Par défaut : écoute sur :3457, bridge sur :3456
./vzlom-poche

# Personnalisé
./vzlom-poche -addr :9999 -bridge http://localhost:3456
```

## Endpoints

| Route | Méthode | Description |
|-------|---------|-------------|
| `/health` | GET | Health check |
| `/ws` | WS | WebSocket relay (token requis) |
| `/proxy/*` | GET | Proxy HTTP vers bridge |

## WebSocket

Connexion : `ws://host:3457/ws?token=<auth_token>`

- Relaie les messages en broadcast à tous les clients connectés
- Authentification par token dans la query string
- Ping/pong automatique (gorilla/websocket)

## Déploiement

Service systemd : `/etc/systemd/system/vzlom-poche.service`

```ini
[Unit]
Description=Vzlom Poche WebSocket Relay
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vzlom-poche
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```