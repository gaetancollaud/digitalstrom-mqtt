# digitalSTROM MQTT

[English](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/home-assistant-app/DOCS.md) |
[Deutsch](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/home-assistant-app/DOCS.de.md)

Diese inoffizielle Community-App betreibt die bestehende
`digitalstrom-mqtt`-Bridge direkt auf Home Assistant OS. Sie wird nicht von
digitalSTROM bereitgestellt oder unterstützt. Die App verwendet den
Home-Assistant-MQTT-Dienst oder einen manuell konfigurierten Broker und MQTT
Discovery. Unterstützte digitalSTROM-Geräte erscheinen dadurch automatisch in Home Assistant.

## Kompatibilität

- Home Assistant OS mit App-Unterstützung wird benötigt.
- Vorgefertigte Images werden für `amd64` und `aarch64` veröffentlicht.
- Installation, Einrichtung und Neustart wurden in einer frischen
  `amd64`-Home-Assistant-OS-VM mit einem echten dSS vollständig getestet.
- Das `aarch64`-Image wird in CI gebaut. Ein Live-Start auf physischer
  ARM-Hardware wurde von den Maintainern bisher nicht getestet.

## App-Repository installieren

[![Home Assistant öffnen und dieses App-Repository hinzufügen.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fgaetancollaud%2Fdigitalstrom-mqtt)

Alternativ unter **Einstellungen -> Apps -> App Store -> Dreipunkt-Menü ->
Repositories** folgende Adresse hinzufügen:

```text
https://github.com/gaetancollaud/digitalstrom-mqtt
```

Sobald das Repository erscheint, **digitalSTROM MQTT** installieren.

## Voraussetzungen

- Ein erreichbarer digitalSTROM-Server (dSS) wird benötigt.
- Die App **Mosquitto Broker** oder einen vorhandenen MQTT-Broker verwenden.
- Prüfen, ob die MQTT-Integration in Home Assistant mit demselben Broker verbunden ist.
- Eine bestehende `digitalstrom-mqtt`-Instanz mit demselben dSS und
  MQTT-Topic-Präfix zuerst stoppen. Zwei aktive Bridges können widersprüchliche
  Zustände und Discovery-Nachrichten veröffentlichen.

## Erster Start

1. Die Adresse des dSS eingeben, normalerweise eine IP-Adresse oder einen
   lokalen Hostnamen. Den Standardport `8080` beibehalten, sofern der dSS keinen
   anderen HTTPS-API-Port verwendet.
2. Den Standardbenutzer `dssadmin` beibehalten, sofern auf dem dSS kein anderes
   Konto verwendet wird.
3. Das dSS-Passwort im sichtbaren Passwortfeld eingeben.
4. Für die Mosquitto-Broker-App **Home Assistant MQTT service** beibehalten.
   Sonst **Manual configuration** wählen und die Broker-Daten unten eintragen.
5. Die Optionen speichern und die App starten.

Die App erstellt einen eigenen dSS-API-Key und speichert ihn dauerhaft in ihrem
privaten `/data`-Verzeichnis. Nach erfolgreicher Einrichtung entfernt sie das
temporäre Passwort aus dem Feld; das leere Feld bleibt sichtbar. Kann Home
Assistant die Optionen nicht sofort aktualisieren, wartet die App und versucht
die Bereinigung automatisch erneut, bevor sie die Bridge startet. Dabei wird
kein weiterer API-Key angelegt. Normale Neustarts verwenden den
gespeicherten API-Key und benötigen das Passwort nicht erneut.

## Konfiguration

### digitalSTROM-Serveradresse

IP-Adresse oder Hostname des dSS. Diese Angabe ist zwingend.

### digitalSTROM-Serverport

HTTPS-API-Port des dSS. Standardwert ist `8080`.

### digitalSTROM-Benutzername

dSS-Konto, das einmalig zum Erstellen des eigenen API-Keys verwendet wird.
Standardwert ist `dssadmin`.

### digitalSTROM-Passwort

Nur beim Erstellen oder Erneuern des gespeicherten API-Keys nötig. Nach
erfolgreicher Erstellung leert die App das Feld. Bei späteren Starts leer lassen.

### MQTT-Verbindung

- **Home Assistant MQTT service** (Standard): MQTT-Dienst von Home Assistant.
  Bezieht Broker-Adresse und Zugangsdaten vom Supervisor, normalerweise von der
  Mosquitto-Broker-App. Die manuellen MQTT-Felder darunter werden ignoriert.
- **Manual configuration**: Manuell konfigurieren. Hostname oder IP-Adresse,
  Port (Standard `1883`), Benutzername und Passwort des Brokers eingeben.
  Zugangsdaten nur leer lassen, wenn der Broker anonyme Verbindungen erlaubt.
  IPv6-Adressen ohne Klammern eingeben. Dieser Modus benötigt keine Mosquitto-
  Broker-App und wechselt auch bei einem Fehler nicht zu ihr.

Die App übernimmt nicht die Verbindungseinstellungen der MQTT-Integration von
Home Assistant. Verwendet diese einen externen Broker, denselben Broker auch
hier eintragen. Beide müssen denselben Broker verwenden, sofern keine eigene
MQTT-Bridge zwischen Brokern eingerichtet ist. Das MQTT-Passwort bleibt für
erneute Verbindungen gespeichert; nur das temporäre dSS-Passwort wird geleert.
Manuelle Verbindungen verwenden unverschlüsseltes TCP im vertrauenswürdigen
lokalen Netz. TLS und eigene Zertifikate werden über diese App-Optionen nicht unterstützt.

### Storenposition invertieren

Aktivieren, wenn Home Assistant `100 %` als vollständig geschlossen statt
vollständig geöffnet interpretieren soll. Dies ändert angezeigte und gesendete
Storenpositionen.

### Verbrauchssensoren aktivieren

Veröffentlicht verfügbare Verbrauchs- und Leistungswerte über MQTT. Diese Werte
werden periodisch beim dSS abgefragt, weil sie nicht über den normalen
Event-Datenstrom bereitgestellt werden.

### Messintervall

Anzahl Sekunden zwischen den Messwertabfragen. Standardwert ist `10`. Ein
kürzeres Intervall liefert aktuellere Werte, belastet den dSS jedoch stärker.

### Log-Level

Steuert die Ausführlichkeit des App-Logs. Für den Normalbetrieb `INFO` und
`DEBUG` nur zur Fehlersuche verwenden. Passwörter und API-Keys werden nicht
absichtlich ins Log geschrieben.

### API-Key neu erstellen

Erstellt beim nächsten App-Start einen Ersatz-Key. Zuerst das dSS-Passwort
eingeben. Schlägt die Erneuerung fehl, bleibt der bisherige Key erhalten.

## Bestehende Bridge migrieren

1. Von der bestehenden Installation abweichende MQTT-Topics,
   Discovery-Präfixe oder Einstellungen zur Namensnormalisierung notieren.
2. Die bisherige Bridge stoppen, bevor diese App gestartet wird.
3. Die App installieren und konfigurieren. Danach die erkannten Geräte und
   einige Befehle in Home Assistant prüfen.
4. Die alte Installation gestoppt lassen, bis die App auch einen Neustart
   erfolgreich überstanden hat.
5. Die alte Installation entfernen und ihren dSS-API-Key widerrufen, sobald sie
   nicht mehr benötigt wird.

Die App verwendet momentan die normalen `digitalstrom-mqtt`-Vorgaben für MQTT
und Discovery. Eine Installation mit eigenen Topic-Präfixen kann über die
App-Optionen noch nicht identisch migriert werden.

## API-Key wiederherstellen

Wurde der API-Key im dSS widerrufen, das dSS-Passwort erneut eingeben,
**API-Key neu erstellen** aktivieren und die App starten. Nach dem Speichern des
neuen Keys und dem Entfernen des temporären Passworts wird die Option
automatisch zurückgesetzt.

## Automatische Wiederherstellung

Nach dem ersten vollständigen Start aktiviert die App den Home-Assistant-
Watchdog automatisch. Dafür muss kein zusätzlicher Schalter betätigt werden.
Wird der Watchdog später manuell ausgeschaltet, behält die App diese Wahl auch
nach Neustarts und Updates bei.

Während des Starts und der API-Key-Einrichtung ist der Watchdog pausiert.
Abgelehnte Zugangsdaten oder eine ungültige Konfiguration stoppen die App mit
einer Erklärung im Log. Einstellungen korrigieren und die App erneut starten;
nach erfolgreichem Start wird der Schutz wiederhergestellt. Vorübergehende
Verbindungsfehler werden mit wachsenden Wartezeiten von 15 bis 60 Sekunden
erneut versucht.
Das gilt auch, wenn der Home-Assistant-MQTT-Dienst noch nicht bereit ist. Nach
einem unerwarteten Startabsturz stellt die App einen zuvor aktivierten Watchdog
wieder her, damit Home Assistant sie neu starten kann. Ein manuell
ausgeschalteter Watchdog bleibt aus.

Der Container-Healthcheck erkennt unbeantwortete Gesundheitsabfragen und
Verarbeitungsschritte, die länger als zwei Minuten blockieren, einschliesslich
Ereignis- und Befehlsverarbeitung. Home Assistant kann die App dann neu starten.
Ein ruhiges Zuhause oder allein ein getrennter MQTT-Broker gilt nicht als
Hänger. Die Prüfung erkennt nicht jeden möglichen Geräte- oder Protokollfehler.

## Automatische Updates

Nach dem ersten vollständigen Start aktiviert die reguläre App die automatischen
Updates von Home Assistant einmalig. Neue veröffentlichte App-Versionen können
dann ohne weitere Bestätigung installiert werden. Wird der Schalter später
ausgeschaltet, bleibt er aus, auch nach einem manuellen Update oder Neustart.
Lokal gebaute Apps, einschliesslich der PR-Testkopie, aktivieren automatische
Updates nicht selbst. Deinstallation mit Löschen der App-Daten setzt diesen
Erststart-Zustand zurück.

## App entfernen

Die App vor der Deinstallation stoppen. Die Deinstallation widerruft den
eigenen API-Key im dSS nicht automatisch und entfernt keine persistenten MQTT-
Discovery-Nachrichten. Den Integrations-Key
`digitalstrom-mqtt-home-assistant` in der Zugriffsverwaltung des dSS widerrufen,
wenn er nicht mehr verwendet wird. Persistente Discovery-Nachrichten erst
entfernen, wenn keine andere Bridge davon abhängt.

## Fehlerbehebung

- **MQTT-Dienst nicht verfügbar**: Mosquitto Broker starten oder für einen
  vorhandenen Broker **Manual configuration** wählen. Ein externer Broker, der
  nur in der MQTT-Integration von Home Assistant eingetragen ist, wird nicht
  automatisch von der App übernommen.
- **API-Key kann nicht erstellt werden**: dSS-Adresse, Benutzername und Passwort
  prüfen. Bei einem Fehler ersetzt die App den bestehenden Key nicht.
- **Home Assistant schaltet ein Gerät, manuelle Änderungen werden aber nicht
  angezeigt**: Prüfen, ob der dSS-Notification-WebSocket auf Port `8090` von
  Home Assistant erreichbar ist. Befehle verwenden die dSS-HTTPS-API,
  normalerweise auf Port `8080`.
- **Keine Geräte erscheinen**: Im App-Log auf die Verbindungen zum dSS und zu
  MQTT warten. Prüfen, ob nicht eine zweite Bridge dieselben MQTT-Entitäten
  veröffentlicht.
- **MQTT bleibt getrennt**: MQTT-Broker und Zugangsdaten prüfen. Sobald die
  Bridge läuft, bleibt ihr Container-Healthcheck unabhängig von MQTT; ein
  Broker-Ausfall markiert den Container daher nicht als fehlerhaft. Die App
  unterstützt den standardmässigen internen Nicht-TLS-Dienst der Mosquitto-App.
  Ein Dienst, der ein eigenes TLS-Zertifikat voraussetzt, wird mit einer klaren
  Logmeldung abgelehnt.
- **Die App stoppt nach einer Optionsänderung**: Das App-Log prüfen.
  Konfigurations- und Verbindungsfehler werden ohne Containerzugriff gemeldet.

Für eine Fehlermeldung App-Version, dSS-Version, relevante App-Logzeilen und das
Verhalten nach einem Neustart angeben. IP-Adressen oder Gerätekennungen vor dem
Veröffentlichen bei Bedarf entfernen.

## Lizenz

Diese App und `digitalstrom-mqtt` stehen unter der
[GNU Affero General Public License v3.0 oder neuer](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/LICENSE).
