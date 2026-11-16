# RustDesk serveri (hbbs i hbbr)

`hbbs` povezuje klijente i agente (ID i dogovor veze), `hbbr` prenosi promet kad izravna veza ne uspije. Oba su otvorenog koda (AGPL-3.0) i ne mijenjamo ih.

## Pokretanje

```bash
cd deploy/rustdesk
docker compose up -d
docker compose logs hbbs   # ispisuje javni ključ servera
```

Javni ključ je u `data/id_ed25519.pub`. Taj ključ i adresa servera ugrađuju se u naš klijent, tako da se on spaja samo na naš server.

**Privatni ključ `data/id_ed25519` čuvaj u sigurnosnoj kopiji.** Ako se izgubi, svi instalirani klijenti moraju dobiti novi ključ.

## Portovi u vatrozidu

| Port | Protokol | Za što |
| --- | --- | --- |
| 21115 | TCP | test vrste NAT-a |
| 21116 | TCP i UDP | registracija ID-a, heartbeat, uspostava veze |
| 21117 | TCP | relay |
| 21118, 21119 | TCP | samo za web klijent; **drži zatvoreno** dok ga ne koristimo |

Portove 21118 i 21119 otvaraj samo iza reverse proxyja koji sam postavlja `X-Real-IP`: RustDesk tim zaglavljima vjeruje bez provjere ([dokumentacija](https://rustdesk.com/docs/en/self-host/rustdesk-server-oss/docker/)).

## Za provjeriti na prvoj instalaciji

- Odbija li `hbbs` klijente bez ispravnog ključa u verziji 1.1.16 (u starijim verzijama trebalo je pokrenuti `hbbs -k _`).
