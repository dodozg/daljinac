# Plan

Stanje: **faza 1 u tijeku** (16. 11. 2026.).

## Faza 1 – Temelj (tjedan 1–2)

- [x] Repozitorij, odluke, arhitektura
- [x] `daljinac-server`: prijem izvještaja, tokeni po firmi, konzola s pragovima
- [x] `daljinac-agent`: CPU, RAM, diskovi za Windows, macOS i Linux
- [x] CI: testovi na tri sustava, build agenta za sve platforme
- [x] Docker Compose za RustDesk `hbbs`/`hbbr`
- [ ] VPS i poddomena (vlasnik)
- [ ] Prvo pokretanje na VPS-u: Caddy, `daljinac-server`, `hbbs`/`hbbr`
- [ ] Agent i službeni RustDesk klijent na 3 vlastita stroja (Windows, Mac, Linux)

## Faza 2 – Agent kao servis i instalacija (tjedan 3–4)

- [ ] Agent kao Windows servis, macOS LaunchDaemon, Linux systemd jedinica
- [ ] Konfiguracijska datoteka agenta (adresa, token) umjesto varijabli okruženja
- [ ] Instalacijski paketi: MSI (Windows), PKG (macOS), skripta (Linux)
- [ ] Skripte za GPO i Intune

## Faza 3 – Naš RustDesk klijent (tjedan 4–6)

- [ ] Javni fork RustDeska: ime Daljinac, ugrađena adresa servera i ključ
- [ ] Build u GitHub Actions za Windows, macOS i Linux
- [ ] Potpis koda (Windows) i notarizacija (macOS)
- [ ] Jedan instalacijski paket po firmi: RustDesk klijent + agent

## Faza 4 – Prijava i upravljanje (tjedan 6–10)

- [ ] Računi za tehničare s 2FA; konzola smije van localhosta tek tada
- [ ] Dnevnik spajanja
- [ ] Adresar strojeva po firmi, povezan s RustDesk klijentom
- [ ] Pilot kod prvog klijenta, zatim prelazak ostalih; Splashtop se gasi zadnji

## Kasnije

- Upozorenja e-poštom, povijest i grafovi
- Popis instaliranog softvera, skripte, zakrpe
