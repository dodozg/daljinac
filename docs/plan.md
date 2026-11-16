# Plan

Stanje: **faza 1 u tijeku**, a uz nju je započeta faza 2 jer ne ovisi o probnom radu (16. 11. 2026.).

## Faza 1 – Temelj (tjedan 1–2)

- [x] Repozitorij, odluke, arhitektura
- [x] `daljinac-server`: prijem izvještaja, tokeni po firmi, konzola s pragovima
- [x] `daljinac-agent`: CPU, RAM, diskovi za Windows, macOS i Linux
- [x] CI: testovi na Windowsu i Linuxu, build agenta za sve platforme
- [x] Docker Compose za RustDesk `hbbs`/`hbbr`
- [x] Upute za probni rad bez servera (`docs/probni-rad.md`)
- [x] Proba 1: kvaliteta RustDeska u odnosu na Splashtop zadovoljava (vlasnik, 16. 11. 2026.)
- [x] Proba 2 i 3: nadzor i agent kao servis na jednom računalu (Claude na uredskom računalu, 16. 11. 2026.; sve točke prošle osim restarta računala)
- [ ] Proba 3, restart računala: servis se javlja bez prijave korisnika (vlasnik)
- [ ] VPS i poddomena (vlasnik)
- [ ] Prvo pokretanje na VPS-u: Caddy, `daljinac-server`, `hbbs`/`hbbr`
- [ ] Agent i službeni RustDesk klijent na 3 vlastita Windows stroja

## Faza 2 – Agent kao servis i instalacija (tjedan 3–4)

- [x] Agent kao Windows servis (`daljinac-agent -install`)
- [x] Konfiguracijska datoteka agenta (adresa, token) umjesto varijabli okruženja
- [ ] Instalacijski paket: MSI
- [ ] Skripte za GPO i Intune

## Faza 3 – Naš RustDesk klijent (tjedan 4–6)

- [ ] Javni fork RustDeska: ime Daljinac, ugrađena adresa servera i ključ
- [ ] Build u GitHub Actions za Windows
- [ ] Potpis koda za Windows
- [ ] Jedan instalacijski paket po firmi: RustDesk klijent + agent

## Faza 4 – Prijava i upravljanje (tjedan 6–10)

- [ ] Računi za tehničare s 2FA; konzola smije van localhosta tek tada
- [ ] Dnevnik spajanja
- [ ] Adresar strojeva po firmi, povezan s RustDesk klijentom
- [ ] Pilot kod prvog klijenta, zatim prelazak ostalih; Splashtop se gasi zadnji

## Kasnije

- macOS i Linux: agent kao LaunchDaemon i systemd servis, PKG i skripta, RustDesk klijent, notarizacija za macOS

- Upozorenja e-poštom, povijest i grafovi
- Popis instaliranog softvera, skripte, zakrpe
