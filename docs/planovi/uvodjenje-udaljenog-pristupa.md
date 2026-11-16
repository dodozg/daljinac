# Plan uvođenja udaljenog pristupa

16. 11. 2026. · arhivirano; trenutni smjer je u [`../odluke.md`](../odluke.md)

Danas koristite Splashtop za oko 500 računala i zadovoljni ste njime; cilj je zadržati sličan doživljaj bez skupe pretplate. Preporuka je 2–4 tjedna usporedo sa Splashtopom probati MeshCentral (upravljanje, datoteke, naredbena linija) i po potrebi RustDesk (glatka slika) na vlastitom serveru, a pretplatu ugasiti tek kad ste zadovoljni. Trošak je jedan server, oko 10–30 € mjesečno; pregled zauzeća svih strojeva razvijamo nas dvoje.

## Odluka: MeshCentral sam ili uz Tactical RMM

Kreni s MeshCentralom; Tactical RMM uzmi tek ako zatrebaju upozorenja, skripte i zakrpe. Za Mac i Linux agente Tactical RMM traži plaćeno sponzorstvo od 55 USD mjesečno.

|  | MeshCentral | Tactical RMM (s MeshCentralom za pristup) |
| --- | --- | --- |
| Udaljeni prikaz, datoteke, naredbena linija | Da, Windows, macOS, Linux | Da, preko ugrađenog MeshCentrala |
| Odvojene grupe po firmi klijentu | Grupe uređaja i ovlasti po korisniku | Klijenti i lokacije |
| CPU, RAM i disk | Po stroju, na stranici uređaja; nema skupnog pregleda | Provjere i upozorenja za sve strojeve |
| Mac i Linux agent | Besplatno | Potpisani agenti tek uz sponzorstvo Tier 1 (55 USD/mj. ili 600 USD/god.) ili vlastiti build iz izvornog koda |
| Licenca | Apache-2.0 (otvoreni kod) | Tactical RMM License 1.0: nije open source, ali izričito dopušta upotrebu MSP-ovima |
| Složenost | Jedan servis (Node.js) | Više servisa (Django, PostgreSQL, NATS, MeshCentral) |

**U usporedbi sa Splashtopom:** MeshCentral šalje sliku kao komprimirane JPEG dijelove zaslona, pa je osjetno sporiji i mutniji kod brzih promjena (video, pomicanje prozora), a koliko znam nema prijenosa zvuka. Za rutinsku podršku i servere to je dovoljno. Ako vam je važna glatkoća kakvu imate u Splashtopu, RustDesk s vlastitim serverom koristi video kodek i daje sličan doživljaj. Njegova besplatna verzija nema središnju konzolu ni korisničke račune, pa bi služio samo za spajanje, uz MeshCentral za sve ostalo.

Izvori: [MeshCentral – kartice uređaja](https://docs.meshcentral.com/meshcentral/devicetabs/), [Tactical RMM – sponzorstvo](https://docs.tacticalrmm.com/sponsor/), [Tactical RMM – cijene](https://www.tacticalrmm.com/pricing), [Tactical RMM License](https://license.tacticalrmm.com). Licencu MeshCentrala navodim po sjećanju; provjerit ću je pri instalaciji.

## Koraci uvođenja

Od praznog servera do prvog klijenta je oko tjedan dana rada, većinom testiranja. Agenti se spajaju prema van, na port 443 vašeg servera, pa kod klijenata ne treba VPN ni otvarati portove.

1. **Server (1 dan).** VPS u EU (za oko 500 strojeva: 4 vCPU, 8 GB RAM, MongoDB umjesto ugrađene baze), poddomena tipa *remote.vasafirma.hr*, MeshCentral u Dockeru s automatskim HTTPS certifikatom. Ja pripremam konfiguraciju u repozitoriju; ti otvaraš VPS i DNS zapis.
2. **Postavke (pola dana).** Računi za troje tehničara s obaveznim 2FA, isključena javna registracija, dnevna sigurnosna kopija na drugu lokaciju.
3. **Probni rad (3–5 dana).** Agenti na vašim vlastitim strojevima: barem jedan Windows, jedan Mac i jedan Linux. Provjeravamo sliku, datoteke, naredbenu liniju, zaslon za prijavu i ponovno spajanje nakon restarta.
4. **Prvi klijent.** Najmanji ili najprijateljskiji klijent, jedna grupa uređaja, instalacija prema tablici ispod.
5. **Ostali klijenti.** Jedna grupa uređaja po firmi; postojeći pristup (Splashtop i sl.) gasi se tek kad su svi strojevi te firme vidljivi u MeshCentralu. Za oko 500 strojeva računajte 4–8 tjedana; kod klijenata bez AD-a i Intunea agent možete instalirati spajanjem preko Splashtopa dok ga još imate.

## Instalacija agenta po vrsti klijenta

Način instalacije ovisi o tome čime klijent upravlja računalima, a ne o tome ima li Microsoft 365. Za Intune treba licenca koja ga uključuje (npr. Business Premium), a obični 365 ga nema.

| Klijent | Windows | macOS | Linux |
| --- | --- | --- | --- |
| Ima Active Directory | GPO: instalacijski paket ili skripta pri pokretanju, za cijelu domenu odjednom | Ručno ili skriptom (AD ne upravlja Macovima) | Skripta preko SSH-a |
| Ima Intune (365 s upravljanjem uređajima) | Intune Win32 aplikacija ili PowerShell skripta | Intune PKG + profil s dozvolama (PPPC) | Skripta preko SSH-a |
| Nema ničega | Jednokratno: preuzimanje s poveznice za tu grupu, ili preko postojećeg alata dok ga još imate | Isto, ručno | Skripta preko SSH-a |

Dvije stvari treba znati unaprijed:

- Na Macu korisnik jednom ručno odobri snimanje zaslona i pristupačnost; ni profil s dozvolama to ne može potpuno preskočiti.
- Agent je vezan uz grupu uređaja u trenutku instalacije, pa za svaku firmu radimo zaseban instalacijski paket ili poveznicu.

## Sigurnost

Server će držati ključeve za sva računala svih vaših klijenata, pa je on najvrjednija meta. Ove postavke idu u prvu verziju konfiguracije.

- Obavezan 2FA za sva tri računa; bez javne registracije novih korisnika.
- Grupe za računala zaposlenika: korisnik vidi obavijest dok je netko spojen, a po želji klijenta mora i odobriti spajanje. Grupe za servere: bez upita.
- Dnevnik tko se kada i na koji stroj spojio, koji se ne briše ručno.
- Redovite nadogradnje MeshCentrala i operacijskog sustava servera; dnevna šifrirana sigurnosna kopija izvan servera.
- Pristup web konzoli po želji ograničen na IP adrese vašeg ureda.
- GDPR: kao izvršitelj obrade za klijente trebate ugovor o obradi podataka; ako ga već imate za održavanje, provjerite pokriva li udaljeni pristup.

## Što razvijamo sami: pregled zauzeća

MeshCentral pokazuje CPU i memoriju tek kad otvorite pojedini stroj. Mi dodajemo jednu stranicu sa svim strojevima, grupiranu po firmi, s bojom za strojeve koji su blizu granice. Procjena: 1–2 tjedna, većinom moj rad.

- **Podaci:** preko MeshCentralovog API-ja, bez dodatnog agenta na strojevima; svakih nekoliko minuta skupljamo CPU, RAM i zauzeće svakog diska.
- **Prikaz:** tablica po firmi (stroj, OS, online, CPU %, RAM %, najpuniji disk %), sortiranje i filtar; prijava istim računima kao u MeshCentral.
- **Pragovi:** npr. disk iznad 85 % žuto, iznad 95 % crveno; kasnije iz toga lako nastaju upozorenja e-poštom.
- **Prvi korak:** provjerim točno što API vraća na Windowsu, Macu i Linuxu; ako nešto nedostaje, podatak skupljamo kratkom skriptom koju MeshCentral pokreće na stroju.

Ako kasnije zatreba ozbiljan nadzor (povijest, grafovi, upozorenja), tada je bolje uzeti Tactical RMM ili Zabbix nego širiti ovaj dodatak.

## Troškovi i otvorena pitanja

Trošak je oko 120–360 € godišnje dok ste samo na MeshCentralu.

| Stavka | Trošak |
| --- | --- |
| VPS u EU | 10–30 € mjesečno |
| Poddomena | 0 € ako već imate domenu |
| MeshCentral i naš dodatak | 0 € |
| Tactical RMM Tier 1 (samo ako ga uzmete) | 55 USD mjesečno ili 600 USD godišnje |

- [ ] VPS ili server u vašem uredu? Kod kojeg pružatelja?
- [ ] Koju poddomenu koristimo za konzolu?
- [ ] Novi GitHub repozitorij za konfiguraciju i dodatak: kako se zove?
- [ ] Koji klijent ide prvi? Koliko danas godišnje plaćate Splashtop (za izračun uštede)?
- [ ] Trebaju li zaposlenici klijenata odobravati spajanje ili je dovoljna obavijest?
