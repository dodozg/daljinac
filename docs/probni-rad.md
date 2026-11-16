# Probni rad

Dvije neovisne probe, obje bez vlastitog servera i za početak samo na Windowsima. Prva pokazuje je li RustDesk dovoljno dobar u odnosu na Splashtop, druga kako izgleda naš nadzor.

## Proba 1: kvaliteta udaljenog pristupa (RustDesk), oko 30 minuta

Ovo je najvažnija proba: ako RustDesk ne zadovolji, ostatak plana treba promijeniti.

1. Na dva Windows računala instaliraj službeni RustDesk s [rustdesk.com](https://rustdesk.com). Najbolje da su na **različitim mrežama** (npr. ured i kuća, ili jedno na mobilnom hotspotu), kao kod pravog klijenta.
2. Na računalu na koje se spajaš u RustDesku pročitaj ID i jednokratnu lozinku, pa se s drugog računala spoji na taj ID.
3. Za probu se koriste RustDeskovi javni serveri; na njima može biti sporije nego na našem. Zato ocjenjuj prvenstveno sliku i upravljanje, a ne vrijeme spajanja.

Usporedi sa Splashtopom:

- [ ] Glatkoća pri pomicanju prozora i listanju dokumenta
- [ ] Oštrina teksta
- [ ] Rad s više monitora
- [ ] Prijenos datoteke od nekoliko stotina MB
- [ ] Kopiranje teksta između računala (međuspremnik)
- [ ] Nenadzirani pristup: postavi stalnu lozinku, odjavi korisnika i spoji se na zaslon za prijavu
- [ ] Pokreni nešto „kao administrator" i provjeri vidiš li UAC prozor i možeš li kliknuti

Zapiši što je lošije nego u Splashtopu. To su stvari koje moramo riješiti prije prelaska.

## Proba 2: nadzor zauzeća (Daljinac), oko 10 minuta

Server i agent rade na **istom** računalu. Za nadzor više računala treba pravi server s HTTPS-om, a to dolazi nakon probe.

### Preuzimanje

1. Otvori [GitHub Actions za daljinac](https://github.com/dodozg/daljinac/actions) i klikni zadnji zeleni build.
2. Na dnu stranice, pod *Artifacts*, preuzmi `daljinac-...` i raspakiraj ga.
3. Trebaju ti dvije datoteke: `daljinac-server-windows-amd64.exe` i `daljinac-agent-windows-amd64.exe`.

Programi još nisu potpisani, pa će SmartScreen ili antivirus možda upozoriti: odaberi *More info* → *Run anyway*.

### Pokretanje (PowerShell, u mapi s datotekama)

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

### Što gledati

Otvori http://127.0.0.1:8080 (stranica se sama osvježava svake minute).

- Vidiš li svoje računalo pod „Probna firma", s CPU-om, RAM-om i najpunijim diskom?
- Pokreni nešto zahtjevno (npr. više kartica s videom): raste li CPU ili RAM, i postaje li ćelija žuta ili crvena?
- Zaustavi agenta (Ctrl+C) i pričekaj 5 minuta: prelazi li stroj u „offline"?
- Što nedostaje ili smeta?

Nakon probe zatvori prozor servera i obriši `server.json`.

## Nakon probe

Javi rezultate obje probe. Ako je RustDesk dovoljno dobar, sljedeći korak je VPS i prvo pravo pokretanje (`docs/plan.md`, faza 1).
