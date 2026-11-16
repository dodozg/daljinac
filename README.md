# Daljinac

Interni alat za udaljeni pristup i nadzor računala kod klijenata.

- **Udaljeni pristup:** RustDesk klijent s našim imenom i vlastitim serverima (`deploy/rustdesk/`).
- **Nadzor:** `daljinac-agent` na svakom stroju svake minute javlja zauzeće CPU-a, RAM-a i diskova; `daljinac-server` ih prikazuje u konzoli, grupirano po firmi, žuto i crveno blizu granice.

**Prvi put? Kreni od [probnog rada](docs/probni-rad.md).**

Arhitektura: [docs/arhitektura.md](docs/arhitektura.md) · Odluke: [docs/odluke.md](docs/odluke.md) · Plan: [docs/plan.md](docs/plan.md)

## Brzi probni rad na jednom računalu

Treba Go 1.26 ili noviji.

```bash
# 1. Konfiguracija s jednom firmom
TOKEN=$(openssl rand -hex 32)
cat > server.json <<EOF
{"clients": [{"name": "Probna firma", "token": "$TOKEN"}]}
EOF

# 2. Server (konzola na http://127.0.0.1:8080, agenti na 127.0.0.1:8081)
go run ./cmd/daljinac-server -config server.json &

# 3. Agent šalje jedan izvještaj
DALJINAC_TOKEN=$TOKEN go run ./cmd/daljinac-agent -server http://127.0.0.1:8081 -once
```

Otvori http://127.0.0.1:8080.

## Na serveru

1. RustDesk serveri: vidi [deploy/rustdesk/README.md](deploy/rustdesk/README.md).
2. `daljinac-server` s konfiguracijom po uzoru na [deploy/server/server.example.json](deploy/server/server.example.json); token za svaku firmu: `openssl rand -hex 32`.
3. Caddy ispred servera za HTTPS: [deploy/server/Caddyfile.example](deploy/server/Caddyfile.example).
4. Konzola sluša samo na localhostu servera; otvori je SSH tunelom:
   `ssh -L 8080:127.0.0.1:8080 korisnik@server` pa http://127.0.0.1:8080.

## Agent

```bash
DALJINAC_TOKEN=<token firme> daljinac-agent -server https://remote.example.hr
```

Token se namjerno čita samo iz varijable okruženja, da se ne vidi u popisu procesa. Gotove izvršne datoteke za Windows, macOS i Linux gradi CI (artefakt uz svaki build).
