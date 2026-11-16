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

Server i `server.json` ostavi za Probu 3.

**Ako se prozor servera odmah zatvori ili agent ne može poslati izvještaj**, port 8080 ili 8081 vjerojatno zauzima drugi program (na uredskom računalu to je bio iVMS-4200, Hikvisionov program za kamere). Tada u `server.json` zadaj druge portove, npr. `{"agent_listen":"127.0.0.1:18081","console_listen":"127.0.0.1:18080","clients":[...]}`, i svugdje u uputama umjesto 8081 i 8080 koristi 18081 i 18080.

## Proba 3: agent kao servis, oko 10 minuta

Nastavak Probe 2, na istom računalu i s istim tokenom. Treba PowerShell otvoren **kao administrator**, u mapi s datotekama. Agenta iz Probe 2 zaustavi (Ctrl+C), a server neka radi.

```powershell
# Isti token kao u Probi 2, pročitan iz server.json (novi prozor ne zna $token)
$env:DALJINAC_TOKEN = (Get-Content server.json | ConvertFrom-Json).clients[0].token
.\daljinac-agent-windows-amd64.exe -install -server http://127.0.0.1:8081
Remove-Item Env:DALJINAC_TOKEN
```

Što provjeriti:

- [ ] U `services.msc` postoji „Daljinac Agent”, pokrenut, s automatskim pokretanjem.
- [ ] Unutar minute stroj se javlja u konzoli.
- [ ] Iz običnog (ne administratorskog) PowerShella `Get-Content C:\ProgramData\Daljinac\agent.json` javlja da pristup nije dozvoljen.
- [ ] Nakon restarta računala (i ponovnog pokretanja servera) stroj se opet javlja, bez prijave korisnika.
- [ ] Zaustavi server na nekoliko minuta: u Event Vieweru (Windows Logs → Application, izvor `DaljinacAgent`) je jedno upozorenje, a nakon ponovnog pokretanja servera poruka da izvještaji opet prolaze.
- [ ] Ponovni `-install` bez tokena (nadogradnja) prolazi i zadržava stari token.
- [ ] `.\daljinac-agent-windows-amd64.exe -uninstall` uklanja servis, `C:\ProgramData\Daljinac` i `C:\Program Files\Daljinac`.

Na kraju zatvori prozor servera i obriši `server.json`.

## Nakon probe

Javi rezultate svih proba. Ako je RustDesk dovoljno dobar, sljedeći korak je VPS i prvo pravo pokretanje (`docs/plan.md`, faza 1).
