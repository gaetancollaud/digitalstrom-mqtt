# digitalSTROM in Home Assistant einrichten

Diese Anleitung führt dich durch die erste Einrichtung der **digitalSTROM MQTT**
App. Du brauchst keine Kommandozeile, keinen Docker-Hub-Account und keine
zusätzliche VM. Die App läuft auf deiner bestehenden Home-Assistant-OS-Installation.

Die Anleitung setzt eine veröffentlichte App-Version voraus. Solange die App
noch nicht im offiziellen Projekt veröffentlicht wurde, lässt sie sich über
diesen Weg nicht installieren. Die **PR Test**-Variante ist für Entwicklung und
Tests gedacht, nicht für diese Anleitung.

## 1. Das solltest du bereithalten

- **Home Assistant OS** mit dem Menü **Einstellungen → Apps**. Bei älteren
  Oberflächen heisst es noch **Add-ons**. Home Assistant Container hat keinen
  App Store und kann diese App nicht auf diesem Weg installieren.
- Die **IP-Adresse deines digitalSTROM-Servers** und seine Zugangsdaten.
  Gemeint ist das Passwort für die digitalSTROM-Serveroberfläche, nicht dein
  Home-Assistant-Passwort. Der Benutzer heisst häufig `dssadmin`.
- Einen MQTT-Broker. Das ist der Dienst, über den die App ihre Daten an Home
  Assistant weitergibt. Falls du noch keinen hast, nimm Weg A im nächsten Schritt.

Falls bereits eine andere `digitalstrom-mqtt`-Bridge läuft, stoppe diese, bevor
du die neue App startest. Lass die alte Installation zunächst als Rückweg bestehen.

## 2. MQTT vorbereiten: einen Weg auswählen

### Weg A: MQTT soll in Home Assistant laufen

1. Öffne **Einstellungen → Apps → App Store**.
2. Suche **Mosquitto broker**, installiere die App und starte sie.
3. Öffne **Einstellungen → Geräte & Dienste**. Falls dort MQTT als neu entdeckt
   angezeigt wird, richte es über **Konfigurieren** ein. Ist MQTT bereits
   eingerichtet, musst du es nicht nochmals hinzufügen.

In der digitalSTROM-App bleibt später **Home Assistant MQTT service** ausgewählt.
Die Broker-Adresse und die Zugangsdaten übernimmt die App selbst. Die manuellen
MQTT-Felder lässt du leer beziehungsweise auf ihrem Standardwert.

### Weg B: Du hast bereits einen MQTT-Broker auf einem anderen Gerät

Du musst Mosquitto nicht zusätzlich in Home Assistant installieren.

1. Öffne **Einstellungen → Geräte & Dienste** und suche die **MQTT**-Integration.
   Falls sie fehlt, füge sie über **Integration hinzufügen → MQTT** hinzu.
2. Verbinde Home Assistant mit deinem vorhandenen Broker. Halte dessen Adresse,
   Port, Benutzername und Passwort auch für die digitalSTROM-App bereit.
3. Wähle in der digitalSTROM-App später **Manual configuration** und trage dort
   denselben Broker ein.

Wichtig: Die App liest **nicht** die Einstellungen der MQTT-Integration aus.
Ein dort eingetragener externer Broker wird deshalb nicht automatisch übernommen.
Beide Verbindungen müssen auf denselben Broker zeigen, damit Home Assistant die
Geräte entdecken kann.

Die manuellen App-Einstellungen unterstützen normales MQTT im vertrauenswürdigen
Heimnetz, üblicherweise auf Port `1883`. Ein Broker, der TLS oder eigene
Zertifikate verlangt, wird hier noch nicht unterstützt. Schalte dafür nicht die
Verschlüsselung eines öffentlichen Brokers ab.

## 3. Die digitalSTROM-App installieren

1. Öffne **Einstellungen → Apps → App Store**.
2. Öffne oben rechts das **Dreipunkt-Menü → Repositories**.
3. Füge diese Adresse hinzu:

   ```text
   https://github.com/gaetancollaud/digitalstrom-mqtt
   ```

4. Schliesse den Dialog und suche im Store nach **digitalSTROM MQTT**.
5. Öffne die App und wähle **Installieren**. Warte, bis die Installation fertig ist.

Home Assistant lädt das passende App-Image selbst herunter. Du musst kein Image
auswählen, keinen Docker-Befehl ausführen und keine zusätzlichen Ports freigeben.

## 4. Die App konfigurieren

Öffne den Reiter **Konfiguration** der installierten App.

| Feld | Was du einträgst |
| --- | --- |
| Adresse des digitalSTROM Servers | Die IP-Adresse deines dSS, ohne `https://` und ohne Port. |
| Port des digitalSTROM Servers | Normalerweise `8080`; nur ändern, wenn dein dSS anders eingerichtet ist. |
| digitalSTROM Benutzername | Dein dSS-Benutzer, häufig `dssadmin`. |
| digitalSTROM Passwort | Dein dSS-Passwort für die erste Einrichtung. |
| MQTT-Verbindung | **Home Assistant MQTT service** für Weg A, **Manual configuration** für Weg B. |

Nur für **Weg B** füllst du ausserdem die manuellen MQTT-Felder aus:
Serveradresse ohne Protokoll oder Port, separater Port, Benutzername und Passwort.
Die Anmeldung bleibt nur dann leer, wenn dein Broker ausdrücklich ohne Anmeldung
betrieben wird. Das MQTT-Passwort ist nicht automatisch dein dSS-Passwort.

Alle anderen Optionen kannst du für den Anfang unverändert lassen.
**API-Schlüssel erneuern** bleibt beim ersten Start ausgeschaltet; die App
erstellt den benötigten Schlüssel von selbst.

Wähle **Speichern**, wechsle zu **Informationen** und drücke **Starten**.

## 5. Prüfen, ob es funktioniert

1. Öffne den Reiter **Protokoll**. Warte auf Meldungen über die Verbindung zum
   MQTT-Server und zum digitalSTROM-WebSocket. Eine kurze Wartezeit ist normal.
2. Nach erfolgreicher Einrichtung wird das **digitalSTROM-Passwortfeld leer**.
   Das ist beabsichtigt: Die App hat einen eigenen Zugangsschlüssel gespeichert.
   Du musst das Passwort nicht wieder eintragen.
3. Öffne **Einstellungen → Geräte & Dienste → MQTT**. Dort sollten die von der
   Bridge unterstützten digitalSTROM-Geräte und ihre Funktionen erscheinen.
4. Starte die App einmal neu und prüfe nochmals das Protokoll. Das gespeicherte
   Passwortfeld soll dabei leer bleiben; die App verwendet ihren Schlüssel weiter.

Nicht jedes digitalSTROM-Gerät und jede Funktion wird von der Bridge unterstützt.
Die App ist ein inoffizielles Community-Projekt.

## 6. Was danach automatisch passiert

- **Beim Booten starten:** Bei der regulären App normalerweise bereits an.
- **Watchdog:** Wird nach dem ersten vollständigen Start eingeschaltet. Erkennt
  Home Assistant einen Hänger der App, kann es sie neu starten.
- **Automatische Updates:** Werden bei der regulären App nach dem ersten
  vollständigen Start einmalig eingeschaltet. Neue veröffentlichte App-Versionen
  können dann ohne weitere Rückfrage installiert werden.

Wenn du Watchdog oder automatische Updates später ausschaltest, bleibt deine
Wahl auch nach Neustarts und Updates erhalten. Du darfst diese Einstellungen
also an deine Bedürfnisse anpassen. Der Watchdog erkennt nicht jeden denkbaren
Fehler; falsche Zugangsdaten musst du weiterhin korrigieren.

Bei einem normalen Neustart oder Update musst du das dSS-Passwort nicht erneut
eingeben. Das manuell eingetragene **MQTT-Passwort** bleibt dagegen gespeichert,
weil die App es für neue Broker-Verbindungen benötigt.

## Wenn etwas nicht klappt

| Beobachtung | Nächster Schritt |
| --- | --- |
| Die App ist nicht im Store sichtbar | Repository-Adresse und Veröffentlichungsstatus prüfen. Die PR-Testkopie ist kein reguläres Release. |
| Das dSS-Passwortfeld fehlt | Prüfe die App-Version und öffne die Konfiguration neu. In einer älteren Testversion gegebenenfalls die ungenutzten optionalen Felder einblenden. |
| „Enter the digitalSTROM password …“ | dSS-Passwort eintragen, speichern und die App erneut starten. |
| „MQTT service is unavailable“ | Bei Weg A Mosquitto starten. Bei Weg B **Manual configuration** auswählen und die Broker-Daten eintragen. |
| Der dSS-Hostname wird nicht gefunden | Statt eines kurzen Hostnamens die IP-Adresse des dSS verwenden; speichern und neu starten. |
| Die Anmeldung wird abgelehnt | Die Zugangsdaten des betroffenen Dienstes korrigieren, speichern und die App erneut starten. Nicht einfach immer wieder starten. |
| Die App läuft, aber es erscheinen keine Geräte | Prüfen, ob MQTT-Integration und App denselben Broker verwenden. Danach das App-Protokoll ansehen. |
| Änderungen am Gerät kommen nicht in HA an | Die dSS-Verbindung prüfen lassen: Neben dem HTTPS-Port, normalerweise `8080`, wird der WebSocket-Port `8090` benötigt. Keine Router-Portfreigabe ins Internet anlegen. |

Vor einer Deinstallation oder grösseren Änderung ein Home-Assistant-Backup
erstellen. Bei einer Fehlermeldung App-Version und relevante Protokollzeilen
angeben; Passwörter und Zugangsschlüssel niemals mitsenden.

Mehr zu den einzelnen Optionen, Migration und Deinstallation steht in der
[ausführlichen Dokumentation](DOCS.de.md). Die
[MQTT-Dokumentation von Home Assistant](https://www.home-assistant.io/integrations/mqtt/)
erklärt die Einrichtung der MQTT-Integration.
