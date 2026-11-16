# Probni rad

Dvije neovisne probe, obje bez vlastitog servera. Prva pokazuje je li RustDesk dovoljno dobar u odnosu na Splashtop, druga kako izgleda naš nadzor.

## Proba 1: kvaliteta udaljenog pristupa (RustDesk), oko 30 minuta

Ovo je najvažnija proba: ako RustDesk ne zadovolji, ostatak plana treba promijeniti.

1. Na dva računala instaliraj službeni RustDesk s [rustdesk.com](https://rustdesk.com). Najbolje da su na **različitim mrežama** (npr. ured i kuća, ili jedno na mobilnom hotspotu), kao kod pravog klijenta.
2. Na računalu na koje se spajaš u RustDesku pročitaj ID i jednokratnu lozinku, pa se s drugog računala spoji na taj ID.
3. Za probu se koriste RustDeskovi javni serveri; na njima može biti sporije nego na našem. Zato ocjenjuj prvenstveno sliku i upravljanje, a ne vrijeme spajanja.

Usporedi sa Splashtopom:

- [ ] Glatkoća pri pomicanju prozora i listanju dokumenta
- [ ] Oštrina teksta
- [ ] Rad s više monitora
- [ ] Prijenos datoteke od nekoliko stotina MB
- [ ] Kopiranje teksta između računala (međuspremnik)
- [ ] Nenadzirani pristup: postavi stalnu lozinku, odjavi korisnika i spoji se na zaslon za prijavu
- [ ] Windows: pokreni nešto „kao administrator" i provjeri vidiš li UAC prozor i možeš li kliknuti
- [ ] Mac: dozvole za snimanje zaslona i pristupačnost (jednom ručno)
- [ ] Linux: radi li na vašoj distribuciji (X11 radi; Wayland je ograničen)

Zapiši što je lošije nego u Splashtopu. To su stvari koje moramo riješiti prije prelaska.

## Proba 2: nadzor zauzeća (Daljinac), oko 10 minuta

Server i agent rade na **istom** računalu. Za nadzor više računala treba pravi server s HTTPS-om, a to dolazi nakon probe.

### Preuzimanje

1. Otvori [GitHub Actions za daljinac](https://github.com/dodozg/daljinac/actions) i klikni zadnji zeleni build.
2. Na dnu stranice, pod *Artifacts*, preuzmi `daljinac-...` i raspakiraj ga.
3. Trebaju ti dvije datoteke za tvoj sustav:

| Sustav | Server | Agent |
| --- | --- | --- |
| Windows | `daljinac-server-windows-amd64.exe` | `daljinac-agent-windows-amd64.exe` |
| Mac (M1 i noviji) | `daljinac-server-darwin-arm64` | `daljinac-agent-darwin-arm64` |
| Mac (Intel) | `daljinac-server-darwin-amd64` | `daljinac-agent-darwin-amd64` |
| Linux | `daljinac-server-linux-amd64` | `daljinac-agent-linux-amd64` |

Programi još nisu potpisani, pa će Windows SmartScreen ili antivirus možda upozoriti. Na Windowsu odaberi *More info* → *Run anyway*. Na Macu prvo pokreni: `xattr -d com.apple.quarantine daljinac-*`, pa `chmod +x daljinac-*`.

### Windows (PowerShell, u mapi s datotekama)

```powershell
# Nasumični token za probnu firmu
$b = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
$token = -join ($b | ForEach-Object { $_.ToString('x2') })
'{"clients":[{"name":"Probna firma","token":"' + $token + '"}]}' | Set-Content -Encoding ascii server.json

# Server u zasebnom prozoru
Start-Process .\daljinac-server-windows-amd64.exe -ArgumentList '-config','server.json'

# Agent javlja svakih 10 sekundi (za probu; inače svake minute)
$env:DALJINAC_TOKEN = $token
.\daljinac-agent-windows-amd64.exe -server http://127.0.0.1:8081 -interval 10s
```

### Mac i Linux (Terminal, u mapi s datotekama)

```bash
token=$(openssl rand -hex 32)
printf '{"clients":[{"name":"Probna firma","token":"%s"}]}' "$token" > server.json
./daljinac-server-darwin-arm64 -config server.json &      # Linux: daljinac-server-linux-amd64
DALJINAC_TOKEN=$token ./daljinac-agent-darwin-arm64 -server http://127.0.0.1:8081 -interval 10s
```

### Što gledati

Otvori http://127.0.0.1:8080 (stranica se sama osvježava svake minute).

- Vidiš li svoje računalo pod „Probna firma", s CPU-om, RAM-om i najpunijim diskom?
- Pokreni nešto zahtjevno (npr. više kartica s videom): raste li CPU ili RAM, i postaje li ćelija žuta ili crvena?
- Zaustavi agenta (Ctrl+C) i pričekaj 5 minuta: prelazi li stroj u „offline"?
- Što nedostaje ili smeta?

Nakon probe zatvori server (Windows: zatvori njegov prozor; Mac i Linux: `kill %1`) i obriši `server.json`.

## Nakon probe

Javi rezultate obje probe. Ako je RustDesk dovoljno dobar, sljedeći korak je VPS i prvo pravo pokretanje (`docs/plan.md`, faza 1).
