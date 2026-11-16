# Daljinac – upute za rad na projektu

Daljinac je interni alat za udaljeni pristup i nadzor računala, umjesto Splashtopa. Razvijaju ga vlasnik projekta i Claude: Claude piše većinu koda, vlasnik testira na stvarnim strojevima i donosi odluke.

Claude ne pamti prethodne sesije. **Sve odluke i stanje projekta su u ovom repozitoriju**: prije rada pročitaj `docs/odluke.md` i `docs/plan.md`, a nakon važne odluke ih ažuriraj.

## Ukratko o korištenju

- Troje tehničara održava više firmi klijenata, ukupno oko 500 računala i servera.
- Windows, macOS i Linux; neki klijenti imaju Active Directory i/ili Microsoft 365, neki ništa.
- Za početak u konzoli treba vidjeti zauzeće CPU-a, RAM-a i diskova.

## Arhitektura (detalji u `docs/arhitektura.md`)

- **Udaljeni prikaz i upravljanje:** RustDesk klijent (fork s našim imenom, adresom servera i ključem) + RustDesk serveri `hbbs`/`hbbr` (`deploy/rustdesk/`). Reverse engineering tuđeg komercijalnog softvera (npr. Splashtopa) **ne radimo**.
- **Nadzor:** naš `daljinac-agent` (Go) šalje izvještaj svake minute na `daljinac-server` (Go), koji ga prikazuje u konzoli grupirano po firmi.

## Struktura

```
cmd/daljinac-server   server: prima izvještaje, poslužuje konzolu
cmd/daljinac-agent    agent: skuplja CPU, RAM, diskove i šalje ih
internal/metrics      format izvještaja, provjera, pragovi (ok/warning/critical)
internal/server       HTTP handleri, spremište u memoriji, konzola (dashboard.html)
internal/agent        prikupljanje (gopsutil) i slanje
deploy/rustdesk       Docker Compose za hbbs/hbbr
deploy/server         primjer konfiguracije i Caddyfile
docs/                 odluke, arhitektura, plan, probni rad (na hrvatskom)
docs/planovi/         raniji planski dokumenti; povijest, ne trenutni plan
```

## Naredbe

```bash
go test -race ./...          # svi testovi
go vet ./... && gofmt -l .   # provjere koje radi i CI
go run ./cmd/daljinac-server -config server.json
DALJINAC_TOKEN=... go run ./cmd/daljinac-agent -server http://127.0.0.1:8081 -once
```

CI (`.github/workflows/ci.yml`) pokreće testove na Windowsu i Linuxu (macOS je isključen do faze za Mac) i gradi agenta i server za sve platforme. Prvo radimo samo Windows (`docs/odluke.md`).

## Pravila

- Kod, identifikatori i komentari na engleskom; dokumentacija i tekst u konzoli na hrvatskom.
- Prednost standardnoj biblioteci; nova ovisnost samo uz razlog zapisan u `docs/odluke.md`.
- Svaka promjena dolazi s testovima; `go test -race ./...` mora prolaziti prije commita.
- Sigurnost se ne slabi radi praktičnosti:
  - konzola sluša samo na loopback adresi dok nema prijavu s 2FA (provjera je u `Config.Validate`),
  - agent šalje token samo preko HTTPS-a (osim na localhost) i čita ga samo iz varijable okruženja,
  - tokeni se uspoređuju u konstantnom vremenu, svaka firma ima svoj token i odvojene podatke.
- Pravi tokeni, `server.json` i RustDesk ključevi nikad ne idu u repozitorij.
