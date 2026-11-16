# Plan: softver za udaljeni pristup

16. 11. 2026. · arhivirano; trenutni smjer je u [`../odluke.md`](../odluke.md)

Razvijamo ga nas dvoje: ja pišem većinu koda, testove i infrastrukturu, a ti testiraš na stvarnim strojevima, donosiš odluke i vodiš račune i certifikate. Realan MVP je 3–5 mjeseci uz tvojih 10–15 sati tjedno, uz trošak od oko 300–1.100 eura godišnje. Usko grlo nije pisanje koda nego testiranje na Windowsima i kroz stvarne mreže.

## Opseg funkcionalnosti

MVP pokriva ono za što korisnici zapravo plaćaju: pouzdano spajanje kroz NAT, glatku sliku, prijenos datoteka i nenadzirani pristup. Sve ostalo dolazi u kasnijim fazama.

| Područje | MVP (3–5 mj.) | Verzija 1.0 (9–12 mj.) | Kasnije |
| --- | --- | --- | --- |
| Platforme agenta (stroj na koji se spaja) | Windows 10/11 | macOS, Linux (X11) | Linux Wayland, Android |
| Platforme klijenta (tko se spaja) | Windows, macOS, web preglednik | Linux, iOS, Android | — |
| Udaljeni prikaz i upravljanje | Jedan i više monitora, tipkovnica, miš, Ctrl+Alt+Del, prikaz zaslona za prijavu i UAC | Zvuk, adaptivna kvaliteta do 60 fps, 4K | Upravljanje pisačem, USB preusmjeravanje |
| Načini pristupa | Nadzirani (kod za podršku + privola) i nenadzirani (instalirani servis) | Wake-on-LAN, ponovno pokretanje uz ponovno spajanje | — |
| Prijenos podataka | Međuspremnik (tekst), prijenos datoteka s nastavkom prekinutog prijenosa | Međuspremnik za datoteke i slike, povlačenje i ispuštanje | Sinkronizacija mapa |
| Komunikacija | Chat u sesiji | Bilješke sesije | Glas i video |
| Nadzor (monitoring) | Status online/offline, osnovni podaci o stroju | CPU, RAM, disk, servisi, upozorenja, inventar hardvera i softvera | Upravljanje zakrpama, skripte, udaljena ljuska (RMM) |
| Upravljanje i sigurnost | Računi, 2FA, grupe strojeva, dnevnik revizije | Uloge i ovlasti (RBAC), SSO (SAML/OIDC), snimanje sesija | Prilagođeni branding, on-premise instalacija |
| Implementacija | Ručni instalacijski program | MSI/PKG za masovnu instalaciju, MDM profili za macOS | API za integracije |

## Arhitektura

Podaci sesije idu izravno; oblak samo spaja strane, a relay pomaže kroz NAT.

```
 Klijent (tko se spaja)        Kontrolna ravnina (oblak)          Agent (udaljeni stroj)
 ┌──────────────────────┐      ┌────────────────────────────┐      ┌─────────────────────────┐
 │ Desktop (Tauri)      │◄┄┄┄┄►│ Signalizacija              │◄┄┄┄┄►│ Windows servis (SYSTEM) │
 │ Web preglednik       │      │ WebSocket; dogovor veze    │      │ Proces u svakoj sesiji  │
 │ Android i iOS        │      ├────────────────────────────┤      │ Snima zaslon            │
 │                      │      │ API i web konzola          │      │ Kodira H.264            │
 │ Dekodira video       │      │ Računi, 2FA, revizija      │      │ Izvodi ulaz i datoteke  │
 │ Šalje tipkovnicu,miš │      ├────────────────────────────┤      │ Šalje metrike (nadzor)  │
 │                      │      │ Baza i metrike             │      │                         │
 │                      │      └────────────────────────────┘      │                         │
 │                      │◄═════ WebRTC izravno: video i ulaz ═════►│                         │
 │                      │      ┌────────────────────────────┐      │                         │
 │                      │◄────►│ STUN / TURN relay          │◄────►│                         │
 └──────────────────────┘      │ kad izravna veza ne uspije │      └─────────────────────────┘
                               └────────────────────────────┘
 ┄┄ signalizacija   ══ / ── podaci sesije, šifrirani s kraja na kraj; relay ih ne vidi
```

Agent drži trajnu vezu prema signalizaciji. Kad se klijent spoji, strane dogovore izravnu WebRTC vezu; relay prenosi šifrirani promet samo kad NAT ili vatrozid to spriječe.

## Tehnologije

Preporuka je Rust za agent i relay te WebRTC za prijenos, jer daje NAT traversal, enkripciju i web klijent bez dodatnog rada.

| Komponenta | Preporuka | Zašto |
| --- | --- | --- |
| Agent (host) | Rust; Windows servis + korisnički proces po sesiji | Memorijska sigurnost, jedna baza koda za Windows, macOS i Linux |
| Snimanje zaslona | Windows: DXGI Desktop Duplication; macOS: ScreenCaptureKit; Linux: PipeWire / X11 | Nativni API-ji s najmanjom latencijom |
| Kodiranje videa | H.264 hardverski (NVENC, Quick Sync, AMF, VideoToolbox, VA-API), softverski fallback (openh264 ili x264) | Niska latencija i opterećenje CPU-a |
| Prijenos | WebRTC (ICE, STUN, TURN, DTLS-SRTP); podatkovni kanali za ulaz, međuspremnik i datoteke | Prolaz kroz NAT, enkripcija, rad u pregledniku |
| Relay | coturn ili vlastiti TURN u Rustu, više regija (EU prvo) | Kad izravna veza ne uspije (obično 10–20 % veza) |
| Signalizacija i API | Go ili Rust, WebSocket za signalizaciju, REST/gRPC za API | Mnogo istovremenih trajnih veza |
| Baza i red poruka | PostgreSQL, Redis; vremenske serije za metrike (TimescaleDB ili ClickHouse) | Pouzdano i poznato; metrike rastu brzo |
| Web konzola i web klijent | TypeScript, React; WebCodecs / WebRTC u pregledniku | Pristup bez instalacije |
| Desktop klijent | Tauri (Rust + web sučelje) | Dijeli kod s web klijentom, mali instalacijski paket |
| Mobilne aplikacije | Kotlin (Android), Swift (iOS) | Nativni API-ji za prikaz i dodir |
| Infrastruktura | Kubernetes ili Nomad, Terraform, Prometheus + Grafana | Više regija, nadzor vlastitog sustava |

## Odluka: od nule ili na open-source temelju

Preporuka je vlastita jezgra (agent, prijenos, relay), uz proučavanje postojećih projekata. Temelj s copyleft licencom ograničava komercijalni model, a temelj bez nje (MeshCentral) daje brži start, ali uz teško zamjenjivu arhitekturu.

| Opcija | Licenca (provjeriti s pravnikom) | Prednost | Nedostatak | Procjena do MVP-a |
| --- | --- | --- | --- | --- |
| Od nule (preporuka) | Vlastita | Puna kontrola, čista licenca, razlikovanje od konkurencije | Najduže i najskuplje | 3–5 mj. |
| Fork RustDeska | AGPL-3.0 | Gotov agent, relay i klijenti u Rustu | AGPL traži objavu izmijenjenog koda, i kad se koristi preko mreže | 1–2 mj. |
| Fork MeshCentrala | Apache-2.0 | Web konzola, agent i RMM funkcije već postoje | Node.js monolit, slabija kvaliteta slike od komercijalnih rješenja | 1–2 mj. |
| Apache Guacamole kao web gateway | Apache-2.0 | Web pristup RDP/VNC/SSH bez klijenta | Nema vlastitog agenta ni rada kroz NAT bez VPN-a | 2–3 mj. (samo dio funkcija) |

Licence su navedene iz poznavanja projekata, ne iz provjere izvora; prije odluke treba ih potvrditi.

## Faze rada

Četiri faze; svaka završava provjerom na tvom stvarnom računalu. Trajanje pretpostavlja tvojih 10–15 sati tjedno.

1. **Prototip (2–4 tjedna).** Agent na Windowsu snima zaslon, kodira H.264 i šalje sliku preko WebRTC-a u preglednik; mišem i tipkovnicom upravljaš iz preglednika.
    - Uvjet: s mobitela na hotspotu upravljaš kućnim računalom, latencija ispod 150 ms.
2. **Upotrebljivi MVP (2–3 mjeseca).** Signalizacijski server, računi i 2FA, relay, Windows servis za nenadzirani pristup, zaslon za prijavu i UAC, prijenos datoteka, međuspremnik, web konzola s popisom strojeva.
    - Uvjet: 2 tjedna svakodnevnog korištenja na 3 stroja bez kritičnih grešaka.
3. **Pilot (1–2 mjeseca).** Potpisani instalacijski paket, automatske nadogradnje, dnevnik revizije, osnovni nadzor (CPU, RAM, disk, online status), 3–5 pilot korisnika.
    - Uvjet: pilot korisnici ga koriste bez tvoje pomoći; vanjski sigurnosni pregled bez kritičnih nalaza.
4. **Proširenje (nakon toga).** macOS i Linux agent, mobilni klijenti, zvuk, SSO, napredni nadzor i RMM, naplata.

## Način rada i troškovi

Ja pišem kod; ti si moje oči i ruke na stvarnim strojevima. Repozitorij je naša zajednička memorija, jer ja ne pamtim ništa između sesija.

| Posao | Ja (Claude) | Ti |
| --- | --- | --- |
| Arhitektura i specifikacija protokola | Predlažem i zapisujem u repozitorij | Odobravaš ili mijenjaš |
| Kod: agent, server, web konzola, klijent | Pišem, uz automatske testove | Pregledavaš i spajaš promjene |
| Build i automatski testovi | Linux u svom okruženju; Windows i macOS preko GitHub Actions | — |
| Testiranje na Windowsu (snimanje, ulaz, servis, UAC) | Izravno, ako spojiš svoje računalo na moju sesiju (Remote Control) | Inače pokrećeš i šalješ logove i snimke zaslona |
| Mreža i NAT | Pišem dijagnostiku i logiranje | Spajaš se s dvije stvarne lokacije (npr. kuća i mobilni hotspot) |
| Server i relay | Pišem Docker i Terraform konfiguraciju | Otvaraš račun kod pružatelja, plaćaš, upisuješ ključeve |
| Certifikati, Apple račun, tvrtka, ugovori | — | Ti |
| Sigurnost | Sigurnosni pregled koda kod svake veće promjene | Vanjski penetracijski test prije prodaje |

Moja ograničenja:

- Ne pamtim prethodne sesije: odluke i stanje zapisujemo u repozitorij (CLAUDE.md, docs/, issues).
- Iz svog okruženja ne vidim Windows zaslon i nemam GPU; snimanje i hardversko kodiranje provjeravaju se na tvom računalu.
- Ne mogu ništa kupiti, registrirati ni potpisati u tvoje ime.

| Stavka | Trošak (procjena) |
| --- | --- |
| VPS za signalizaciju i relay (1–2 manja servera u EU) | 100–300 € godišnje |
| Domena | 10–20 € godišnje |
| Certifikat za potpis koda (Windows) | 100–700 € godišnje, ovisno o vrsti (provjeriti) |
| GitHub (privatni repozitorij, Actions minute za Windows i macOS) | 0–50 € godišnje |
| Apple Developer Program (tek u fazi macOS) | 99 USD godišnje |
| Vanjski penetracijski test (prije prodaje) | 5.000–15.000 € jednokratno |

Iznosi su okvirni, iz iskustva, ne iz ponuda. Bez penetracijskog testa prva godina košta oko 300–1.100 eura.

## Sigurnost i zakonska usklađenost

Alat za udaljeni pristup je ključ za tuđe računalo: jedan propust znači kompromitaciju svih klijenata. Sigurnost mora biti dio MVP-a, ne dodatak.

**Tehničke mjere (MVP)**

- Enkripcija s kraja na kraj; relay ne vidi sadržaj sesije.
- 2FA obavezan za sve račune; uređaji se odobravaju pri prvoj prijavi.
- Nadzirani pristup samo uz izričitu privolu korisnika na stroju i stalno vidljiv indikator sesije.
- Dnevnik revizije koji se ne može mijenjati: tko, kada, kojem stroju, što je preneseno.
- Potpisane i provjerene nadogradnje agenta; agent ne prima naredbe koje nisu potpisane.
- Program prijave ranjivosti i vanjski penetracijski test prije svake veće verzije.

**Zloupotreba**

Prevaranti lažno predstavljaju „tehničku podršku" i koriste upravo ovakve alate. Potrebni su provjera identiteta novih računa, upozorenje korisniku prije davanja pristupa, ograničenja za besplatne račune i brzo blokiranje prijavljenih računa.

**Zakonski okvir i platforme**

- GDPR: sadržaj zaslona i datoteke su osobni podaci; treba ugovor o obradi podataka (DPA), podaci u EU i rokovi čuvanja snimki.
- NIS2 (u Hrvatskoj Zakon o kibernetičkoj sigurnosti): kupci iz kritičnih sektora tražit će dokaze o sigurnosti dobavljača; ISO 27001 vjerojatno postaje uvjet za veće kupce.
- Windows: certifikat za potpis koda (EV) i reputacija u SmartScreenu; bez njih antivirusi agent označavaju kao zloćudan.
- macOS: notarizacija i ručno odobrenje dozvola Screen Recording i Accessibility (u tvrtkama preko MDM profila).
- iOS ne dopušta upravljanje drugim aplikacijama; moguć je samo prikaz zaslona.

## Rizici i najteži tehnički dijelovi

Najveći rizik nije funkcionalnost nego kvaliteta: korisnici uspoređuju latenciju i pouzdanost spajanja sa Splashtopom i odustaju nakon jedne loše sesije.

| Rizik | Zašto je teško | Mjera |
| --- | --- | --- |
| Zaslon za prijavu, UAC i Ctrl+Alt+Del na Windowsima | Servis radi u sesiji 0; za secure desktop treba poseban proces pod SYSTEM računom | Riješiti u prototipu, prije ostalih funkcija |
| Latencija i kvaliteta slike | Kodiranje, mreža i dekodiranje zajedno moraju ostati ispod ~100 ms | Hardverski koderi, adaptivni bitrate, mjerenje od prvog dana |
| Pouzdanost spajanja | Korporativni vatrozidi, simetrični NAT, proxyji | TURN preko TCP/TLS na portu 443, relay u više regija |
| Antivirusi agent označavaju kao zloćudan | Ponašanje agenta nalikuje alatima napadača | EV potpis, prijava agenta proizvođačima antivirusa, stabilna imena datoteka |
| Linux Wayland i macOS dozvole | Wayland ograničava snimanje i ulaz; macOS traži ručno odobrenje | Wayland preko PipeWire portala tek nakon v1.0; jasne upute za macOS |
| Raznolikost hardvera | Različiti GPU-i, upravljački programi i rezolucije | Farma testnih strojeva i automatski testovi |
| Konkurencija i cijena | LogMeIn, Splashtop, TeamViewer, AnyDesk i besplatni RustDesk | Fokus na nišu (npr. lokalni IT servisi, EU smještaj podataka, cijena) |

## Otvorena pitanja

- [ ] Tko je ciljni kupac: IT servisi (MSP), interna IT podrška tvrtki ili pojedinci?
- [ ] Gradimo od nule ili na open-source temelju (vidi odluku gore)?
- [ ] Je li prioritet udaljena podrška ili nadzor i upravljanje strojevima (RMM)? To mijenja redoslijed faza.
- [ ] Samo SaaS ili i on-premise instalacija za kupce koji ne smiju koristiti oblak?
- [ ] Imaš li Windows računalo koje možeš spojiti na moju sesiju za testiranje? Kako ćemo nazvati projekt i repozitorij?
