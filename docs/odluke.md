# Odluke

Najnovije na vrhu. Svaka odluka: datum, što je odlučeno, zašto i što bi je promijenilo.

## 2026-11-16 · Agent kao Windows servis, token u zaštićenoj datoteci

**Odluka:** `daljinac-agent -install` (kao administrator) kopira program u `C:\Program Files\Daljinac`, zapisuje adresu servera i token u `C:\ProgramData\Daljinac\agent.json` i registrira servis `DaljinacAgent` (automatsko pokretanje s odgodom, ponovno pokretanje nakon pada). Datoteci i mapi mogu pristupiti samo SYSTEM i administratori, a vlasnik je grupa Administrators. Servis piše u Event Viewer (Application, izvor `DaljinacAgent`). Pad veze bilježi se kad počne, najviše jednom na sat dok traje i kad prođe. Varijable `DALJINAC_TOKEN` i `-server` i dalje rade za probu u konzoli, ali ih servis ignorira.

**Zašto:** agent mora raditi bez prijavljenog korisnika i preživjeti restart. Token ne smije biti čitljiv običnom korisniku (s njim bi mogao slati lažne izvještaje u ime firme), a ni vidljiv u popisu procesa. Zato se ne zadaje kao argument. Servis radi kao LocalSystem, kao i RustDesk.

**Nova ovisnost:** `golang.org/x/sys` (Go tim, već je bio neizravna ovisnost preko gopsutila) za Windows servis, dnevnik događaja i ovlasti nad datotekama. U standardnoj biblioteci toga nema.

**Promijenilo bi je:** MSI (sljedeći korak) preuzima kopiranje i registraciju servisa; `-install` ostaje za ručnu instalaciju i skripte.

## 2026-11-16 · Prvo samo Windows

**Odluka:** probni rad, instalacija, servis, RustDesk klijent i potpis koda prvo radimo samo za Windows. macOS i Linux dolaze nakon što Windows radi kod prvog klijenta.

**Zašto:** odluka vlasnika; jedna platforma skraćuje put do prve prave upotrebe.

**Posljedice:** kod agenta i servera ostaje višeplatformski (to ništa ne košta), a CI i dalje gradi sve platforme. Testovi u CI-ju rade na Linuxu i Windowsu; test na macOS-u je isključen dok ne krene faza za Mac, jer su macOS minute na GitHubu deset puta skuplje.

## 2026-11-16 · Nadzor preko vlastitog malog agenta, ne preko RustDesk forka

**Odluka:** CPU, RAM i diskove skuplja zaseban `daljinac-agent` (Go, biblioteka gopsutil), a ne izmjena RustDesk klijenta.

**Zašto:** RustDesk fork mijenjamo što manje (ime, adresa servera, ključ), pa se lakše usklađuje s novim verzijama. Agent je jedna mala izvršna datoteka za sve tri platforme i može se testirati neovisno.

**Promijenilo bi je:** ako dva programa na svakom stroju postanu problem pri instalaciji; tada agenta spajamo u isti instalacijski paket.

## 2026-11-16 · Server u Go-u, stanje u memoriji

**Odluka:** `daljinac-server` je u Go-u. Zadnji izvještaj svakog stroja drži u memoriji, bez baze.

**Zašto:** za trenutno zauzeće dovoljan je zadnji izvještaj; agenti javljaju svake minute, pa je nakon restarta slika potpuna za minutu. Manje dijelova za održavanje.

**Promijenilo bi je:** povijest i grafovi, korisnički računi, dnevnik spajanja. Tada dodajemo SQLite.

## 2026-11-16 · Konzola samo na localhostu dok nema prijavu

**Odluka:** konzola sluša samo na 127.0.0.1; tehničari joj pristupaju preko SSH tunela. Server odbija pokretanje ako je drugačije podešeno.

**Zašto:** konzola pokazuje sve strojeve svih klijenata. Bez prijave s 2FA ne smije biti dostupna s interneta.

**Promijenilo bi je:** kad napravimo prijavu s 2FA (sljedeća faza).

## 2026-11-16 · Agent šalje, server ne poziva agenta

**Odluka:** agent se spaja prema serveru (HTTPS, port 443) s tokenom svoje firme.

**Zašto:** kod klijenata ne treba otvarati portove ni VPN. Token određuje firmu, pa se strojevi različitih klijenata ne mogu pomiješati.

## 2026-11-16 · RustDesk kao osnova za udaljeni pristup

**Odluka:** udaljeni prikaz i upravljanje radi RustDesk (AGPL-3.0) s vlastitim `hbbs`/`hbbr` serverima; naš je backend, konzola i nadzor.

**Zašto:** kvaliteta slike blizu Splashtopovoj (video kodeci, izravna veza, zvuk), radi na Windowsu, macOS-u i Linuxu. Razvoj streamera od nule trajao bi 3–5 mjeseci.

**Posljedice:** izmijenjeni klijent koji instaliramo kod klijenata mora imati dostupan izvorni kod (AGPL), pa će fork biti javni repozitorij. Za Windows treba certifikat za potpis koda, za macOS Apple Developer račun.

**Odbačeno:** reverse engineering Splashtopa (licenca i sigurnost), MeshCentral kao glavno rješenje (slabija slika od Splashtopa).

## 2026-11-16 · Ime projekta: Daljinac

Kolokvijalno „daljinski upravljač”. Pretraga weba nije našla program s tim imenom; zaštitni znak nije provjeren (potrebno samo ako ga budemo nudili drugima).
