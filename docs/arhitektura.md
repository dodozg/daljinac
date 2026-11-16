# Arhitektura

```
 Tehničar                                    Računalo ili server kod klijenta
 ┌────────────────────┐                      ┌──────────────────────────────┐
 │ RustDesk klijent   │◄── izravno (P2P) ───►│ RustDesk klijent (Daljinac)  │
 │ (Daljinac)         │◄── ili preko hbbr ──►│                              │
 └─────────┬──────────┘                      │ daljinac-agent               │
           │ SSH tunel                       └───────┬──────────────┬───────┘
           ▼                                         │ HTTPS 443    │ 21115–21117
 ┌──────────────────────────── Naš server (VPS) ─────┼──────────────┼───────┐
 │ daljinac-server: konzola 127.0.0.1:8080 ◄─ Caddy ─┘              │       │
 │                  agenti  127.0.0.1:8081                          │       │
 │ hbbs (ID, dogovor veze), hbbr (relay)  ◄─────────────────────────┘       │
 └──────────────────────────────────────────────────────────────────────────┘
```

## Dijelovi

| Dio | Što radi | Gdje |
| --- | --- | --- |
| RustDesk klijent (fork) | Prikaz zaslona, upravljanje, datoteke | Na svim strojevima i kod tehničara |
| `hbbs` | Registracija strojeva, dogovor izravne veze | Naš server, portovi 21115–21116 |
| `hbbr` | Relay kad izravna veza ne uspije | Naš server, port 21117 |
| `daljinac-agent` | Svake minute šalje CPU, RAM i diskove | Na svim strojevima |
| `daljinac-server` | Prima izvještaje, konzola po firmama | Naš server, iza Caddyja |
| Caddy | HTTPS za agente; javno samo `/api/v1/report` i `/healthz` | Naš server, port 443 |

## Izvještaj agenta

`POST /api/v1/report` s `Authorization: Bearer <token firme>`; tijelo je JSON (`internal/metrics.Report`): ID stroja, ime, OS, CPU %, ukupna i zauzeta memorija, popis diskova, vrijeme rada, verzija agenta. Najviše 64 KB.

Server izvještaj sprema pod (firma, ID stroja). Stroj je **offline** ako se nije javio 5 minuta (`offline_after`).

## Pragovi

| | Žuto od | Crveno od |
| --- | --- | --- |
| CPU | 75 % | 90 % |
| RAM | 85 % | 95 % |
| Najpuniji disk | 85 % | 95 % |

Mijenjaju se u `thresholds` u konfiguraciji servera.

## Poznata ograničenja

- **macOS:** sistemski i podatkovni APFS volumen dijele isti prostor; prikazujemo oba, pa postotak podatkovnog volumena može podcijeniti stvarnu popunjenost.
- **CPU:** jedno mjerenje od 2 s po izvještaju, ne prosjek kroz minutu.
- Agent se zasad pokreće ručno; instalacija kao servis dolazi u sljedećoj fazi.
