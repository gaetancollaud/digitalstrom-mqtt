# digitalSTROM MQTT

## 1. Bevor du beginnst

Eine Anleitung für Einsteiger, um digitalSTROM mit Home Assistant zu verbinden

**Aktuelle Version: 2.4.0-haos.1.** Die reguläre Home-Assistant-App ist über das App-Repository des Projekts verfügbar. Dieses inoffizielle Community-Projekt wird nicht von digitalSTROM bereitgestellt oder unterstützt.

### Das brauchst du

- **Home Assistant OS** mit **Einstellungen > Apps**. Die App läuft nicht mit Home Assistant Container.
- Einen funktionierenden **digitalSTROM-Server (dSS)**, den Home Assistant im Heimnetz erreichen kann.
- Die Adresse, den Benutzernamen und das Passwort des dSS. Diese Angaben gehören zum digitalSTROM-Server, nicht zu deinem Home-Assistant-Konto oder einem digitalSTROM-Cloud-Konto. Der übliche Benutzername ist **dssadmin**.
- Einen MQTT-Broker. Wenn MQTT neu für dich ist, erklärt Abschnitt 2 die einfachste Möglichkeit.

### So arbeiten die Komponenten zusammen

![Verbindungsweg in beide Richtungen zwischen digitalSTROM-Server, App, MQTT-Broker und Home Assistant](images/connection-flow.png)

Die App und die **MQTT-Integration** von Home Assistant nutzen denselben Broker. Die App kündigt Geräte per MQTT Discovery an und veröffentlicht ihren Status. Steuerbefehle aus Home Assistant laufen über den Broker zur App und zum dSS.

### Verwendest du bereits digitalstrom-mqtt?

Stoppe die bestehende Bridge, bevor du diese App startest. Lass die alte Installation verfügbar, bis die neue App ihre Prüfungen und einen Neustart bestanden hat. Wenn deine alte Einrichtung eigene MQTT-Topic-Namen oder Discovery-Einstellungen verwendet, lies zuerst die Migrationshinweise in der ausführlichen Dokumentation.

### So geht es weiter

- **Abschnitte 2 und 3:** MQTT einrichten und App installieren.
- **Abschnitt 4:** Verbindungen und Geräte prüfen.
- **Abschnitte 5 und 6:** Updates, Wiederherstellung und Hilfe.

<!-- pagebreak -->

## 2. MQTT einrichten

Wenn du noch keinen MQTT-Broker hast, nimm **A**. Das ist der einfachste Einstieg: Der Broker läuft in Home Assistant, und die App erhält ihre Verbindungsdaten automatisch. **B** ist für einen bereits vorhandenen Broker. Verwende in der digitalSTROM-App nur einen Verbindungsmodus.

![Bei beiden Wegen tauschen App und Home Assistant Befehle und Status über denselben MQTT-Broker aus](images/mqtt-choice.png)

### A. MQTT-Dienst von Home Assistant

1. Öffne **Einstellungen > Apps > App Store**. Suche **Mosquitto broker**, installiere die App und starte sie.
2. Öffne **Einstellungen > Geräte & Dienste**. Wenn MQTT erkannt wird, wähle **Konfigurieren**. Andernfalls wähle **Integration hinzufügen > MQTT** und folge den Einrichtungsschritten.
3. Vergewissere dich, dass die MQTT-Integration mit dem Mosquitto-Broker verbunden ist.

Lass in der digitalSTROM-App **MQTT-Verbindung** auf **Home Assistant MQTT service** stehen. Dies ist die Standardeinstellung. Home Assistant stellt der App die Broker-Verbindungsdaten bereit. Lass die manuellen MQTT-Felder unverändert; die App ignoriert sie in diesem Modus. Du musst das Mosquitto-Passwort nicht kopieren.

### B. Manuelle Einrichtung für einen vorhandenen Broker

![MQTT-Verbindung in der App-Konfiguration. Im Beispiel ist Manual configuration gewählt; Home Assistant MQTT service ist die andere Möglichkeit.](images/mqtt-mode.png)

1. Öffne unter **Einstellungen > Geräte & Dienste** die Integration **MQTT**. Wenn sie fehlt, wähle **Integration hinzufügen > MQTT**.
2. Verbinde Home Assistant mit deinem vorhandenen Broker. Wenn du die Adresse, den Port oder die Zugangsdaten benötigst, frage die zuständige Person.
3. Wähle in der digitalSTROM-App **Manual configuration** und gib dieselbe Broker-Adresse, den Port und einen zugelassenen Benutzernamen ein.

Die App übernimmt keine Einstellungen aus der MQTT-Integration von Home Assistant. Wenn Home Assistant einen externen Broker verwendet, trägst du diesen Broker auch separat in der App ein. Für die manuelle Einrichtung ist Mosquitto in Home Assistant nicht erforderlich; bei einem Fehler wechselt die App nicht zum MQTT-Dienst von Home Assistant.

### Grenzen der manuellen Verbindung

Die manuellen Einstellungen verwenden unverschlüsseltes MQTT in einem vertrauenswürdigen lokalen Netzwerk, normalerweise über Port **1883**. TLS und eigene Zertifikate werden von diesen App-Optionen nicht unterstützt. Deaktiviere die Verschlüsselung eines öffentlichen Brokers nicht, um diesen Modus zu verwenden. Bitte stattdessen um Hilfe bei einer kompatiblen lokalen Einrichtung.

**Bereit für den nächsten Schritt:** Die MQTT-Integration von Home Assistant ist mit demselben Broker verbunden, den auch die digitalSTROM-App verwenden wird.

## 3. App installieren und einrichten

### App-Repository des Projekts hinzufügen

![Repository-Dialog mit der öffentlichen Projektadresse.](images/add-repository.png)

1. Öffne **Einstellungen > Apps > App Store**.
2. Öffne das **Drei-Punkte-Menü > Repositories**, wähle **Hinzufügen** und gib folgende Adresse ein:

https://github.com/gaetancollaud/digitalstrom-mqtt

3. Kehre nach dem Hinzufügen zum App Store zurück und suche **digitalSTROM MQTT**.
4. Wähle **Installieren** und warte, bis der Vorgang abgeschlossen ist. Die aktuelle reguläre Version ist **2.4.0-haos.1**.

![App-Bedienelemente nach der Installation: Starten, Beim Systemstart starten, Watchdog und Automatische Updates.](images/app-info.png)

Du kannst auch die Schaltfläche [Repository hinzufügen](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fgaetancollaud%2Fdigitalstrom-mqtt) von Home Assistant verwenden.

### dSS-Daten eingeben

![App-Konfiguration mit beispielhafter dSS-Adresse, Standard-Port, Benutzername und leerem temporärem Passwortfeld.](images/dss-config.png)

Öffne den Tab **Konfiguration** der App:

- **Adresse des digitalSTROM Servers:** IP-Adresse oder Hostname, ohne `https://`, Pfad oder Port.
- **Port des digitalSTROM Servers:** Lass **8080** stehen, sofern dein dSS keinen anderen HTTPS-API-Port verwendet.
- **digitalSTROM Benutzername:** Dein dSS-Konto; der übliche Standardwert ist **dssadmin**.
- **digitalSTROM Passwort:** Gib für die erste Einrichtung das dSS-Passwort ein.
- **MQTT-Verbindung:** Wähle **Home Assistant MQTT service** für Weg A in Abschnitt 2 oder **Manual configuration** für Weg B.

### Nur bei manueller Einrichtung

- **Manuelle MQTT-Serveradresse:** Broker-Hostname oder IP-Adresse, ohne Protokollpräfix oder Port. IPv6-Adressen ohne Klammern eingeben.
- **Manueller MQTT-Serverport:** Üblicherweise **1883**.
- **Manueller MQTT-Benutzername** und **Manuelles MQTT-Passwort:** Verwende den Broker-Zugang. Lass die Felder nur leer, wenn der Broker Verbindungen ohne Anmeldung erlaubt. Dieses Passwort ist unabhängig vom dSS-Passwort und bleibt für erneute Verbindungen gespeichert.

Lass die übrigen Optionen vorerst auf ihren Standardwerten. Lass **API-Schlüssel erneuern** beim ersten Start ausgeschaltet. Wähle **Speichern**, öffne den Tab **Info** und wähle **Starten**.

## 4. Funktion prüfen

### App-Log prüfen

<!-- figure-slot: App connection log | Alt text: App log showing the successful MQTT and digitalSTROM WebSocket messages without private broker names. -->

Öffne den Tab **Protokoll** und warte, bis der Start abgeschlossen ist. Achte auf diese Meldungen:

- **Connected to MQTT server.**
- **Connected to websocket for notifications**
- Nur bei der ersten Einrichtung oder nach einer Schlüsselerneuerung: **The digitalSTROM API key setup is complete; the temporary password option is clear.**

Die ersten beiden Meldungen bestätigen zwei getrennte Verbindungen. MQTT überträgt Gerätedaten an Home Assistant; der dSS-WebSocket übermittelt aktuelle Änderungen von digitalSTROM. Ein laufender App-Status allein bestätigt keine dieser Verbindungen. Vorübergehende Verbindungsfehler können erneut versucht werden. Warte daher etwas, bis der Start abgeschlossen ist.

### Ein leeres dSS-Passwortfeld ist normal

Nach erfolgreicher Einrichtung erstellt und speichert die App ihren eigenen dSS-API-Key und leert danach das temporäre Passwortfeld. Das Feld **digitalSTROM Passwort** bleibt sichtbar, ist aber leer. Lass es bei späteren Starts leer. Den API-Key musst du weder kopieren noch einfügen.

### Geräte prüfen

<!-- figure-slot: Discovered devices | Alt text: Home Assistant MQTT integration listing discovered digitalSTROM devices and entities, with personal names removed. -->

1. Öffne **Einstellungen > Geräte & Dienste > MQTT** und suche nach unterstützten digitalSTROM-Geräten und ihren Steuerelementen oder Messwerten. Home Assistant nennt ein einzelnes Steuerelement oder einen Messwert eine **Entität**.
2. Prüfe, ob Messwerte verfügbar sind. Nicht jedes digitalSTROM-Gerät und jede Funktion wird unterstützt.
3. Wähle im Tab **Info** der App **Neustart**. Prüfe, ob die Verbindungsmeldungen wieder erscheinen und die Geräte erneut verfügbar sind, ohne dass du das dSS-Passwort eingeben musst.

Die Erkennung bestätigt, dass Geräte angekündigt wurden. Sie beweist nicht, dass jede Funktion funktioniert. Wenn du eine Steuerung ausprobieren möchtest, verwende ein vertrautes Licht und prüfe dessen tatsächliche Reaktion. Betätige keine Storen oder andere Geräte nur, um die Einrichtung abzuschliessen.

**Die Einrichtung ist abgeschlossen, wenn** das Protokoll beide Verbindungen bestätigt, unterstützte Geräte unter MQTT erscheinen und ein normaler Neustart gelingt.

## 5. Wiederherstellung, Updates und Deinstallation

### App automatisch starten

Aktiviere im Tab **Info** der App **Beim Systemstart starten**, wenn die App zusammen mit Home Assistant starten soll. Diese Einstellung ist unabhängig von Watchdog und Update-Einstellungen.

### Watchdog und Wiederherstellung der Verbindung

Nach dem ersten vollständigen Start aktiviert die App automatisch den **Watchdog** von Home Assistant. Home Assistant kann die App damit neu starten, wenn die Gesundheitsprüfung feststellt, dass die Verarbeitung hängt. Während des Starts pausiert die App den Watchdog, solange sie die Einstellungen prüft und den dSS-API-Key erstellt. Ungültige Zugangsdaten musst du selbst korrigieren. Ein Broker-Ausfall allein löst die Gesundheitsprüfung nicht aus, und der Watchdog erkennt nicht jedes Geräte- oder Protokollproblem.

Wenn du den Watchdog später ausschaltest, behält die App diese Einstellung über Neustarts und Updates hinweg bei. Die App versucht vorübergehende Verbindungsfehler erneut, auch wenn der MQTT-Dienst noch nicht bereit ist.

### Updates

Nach dem ersten vollständigen Start aktiviert die reguläre App einmalig **Automatische Updates** von Home Assistant. Wenn du Veröffentlichungen lieber zuerst prüfen möchtest, schalte **Automatische Updates** nach der Einrichtung auf der App-Seite aus. Home Assistant behält deine Auswahl bei. Erstelle vor einem manuellen Update oder einer grösseren Änderung ein Home-Assistant-Backup und lies die Versionshinweise. Wiederhole nach dem Update die Prüfungen in Abschnitt 4.

Ein normaler Neustart oder ein Update behält den gespeicherten dSS-API-Key bei. Auch ein manuell eingerichtetes MQTT-Passwort bleibt für erneute Verbindungen gespeichert.

### Wenn der dSS-API-Key widerrufen wurde

Stoppe die App. Gib unter **Konfiguration** das dSS-Passwort ein und aktiviere **API-Schlüssel erneuern**. Wähle **Speichern** und starte dann die App. Nach erfolgreichem Ersetzen wird das Passwort geleert und die Option zurückgesetzt. Schlägt das Ersetzen fehl, behält die App den bisherigen Key bei. Lies das Protokoll und behebe das Problem, bevor du es erneut versuchst.

### Vor dem Entfernen der App

Erstelle ein Backup und stoppe die App, bevor du sie deinstallierst. Die Deinstallation widerruft den API-Key nicht auf dem dSS und entfernt auch keine gespeicherten MQTT-Discovery-Nachrichten. Wenn der Key nicht mehr verwendet wird, widerrufe **digitalstrom-mqtt-home-assistant** in den dSS-Berechtigungseinstellungen. Entferne gespeicherte Discovery-Nachrichten erst, wenn du bestätigt hast, dass keine andere Bridge davon abhängt. Für eine neue Installation wird das dSS-Passwort benötigt, um einen neuen Key zu erstellen.

<!-- pagebreak -->

## 6. Fehlerbehebung und Hilfe

### Die App fehlt oder die Installation schlägt fehl

Prüfe die Repository-Adresse in Abschnitt 3, deine Internetverbindung und das Home-Assistant-App-Protokoll. Vergewissere dich, dass **digitalSTROM MQTT 2.4.0-haos.1** angeboten wird. Wenn das Image nicht heruntergeladen werden kann, prüfe die Release- und Issue-Seiten des Projekts, bevor du die Installation wiederholst.

### Die App fragt nach dem dSS-Passwort

Gib das dSS-Passwort unter **Konfiguration** ein, wähle **Speichern** und starte die App. Es handelt sich um die Anmeldung am digitalSTROM-Server, nicht um dein Home-Assistant-Passwort. Prüfe bei einem Anmeldefehler die Adresse, den Port und den Benutzernamen.

### MQTT verbindet sich nicht

Bei **Home Assistant MQTT service** prüfst du, ob Mosquitto broker installiert ist und läuft und ob die MQTT-Integration von Home Assistant diesen Dienst verwendet. Bei **Manual configuration** prüfst du den ausgewählten Modus sowie die separaten Broker-Daten und Zugangsdaten der App. Die App übernimmt keine Einstellungen eines externen Brokers aus der Integration.

### Die App verbindet sich, aber es erscheinen keine Geräte

Prüfe beide Verbindungsmeldungen in Abschnitt 4. Vergewissere dich, dass App und Home Assistant denselben Broker verwenden, und stoppe eine ältere Bridge, die dieselben Geräte veröffentlicht. Eigene Topic-Präfixe oder Discovery-Einstellungen müssen vor der Migration geprüft werden.

### Befehle funktionieren, aber Änderungen am Wandschalter erscheinen nicht

Die dSS-HTTPS-API verwendet normalerweise Port **8080**. Live-Benachrichtigungen verwenden den dSS-WebSocket auf Port **8090**, der von Home Assistant erreichbar sein muss. Richte keine Router-Portweiterleitungen aus dem Internet ein.

### Sicher um Hilfe bitten

Nutze den [Issue-Tracker des Projekts](https://github.com/gaetancollaud/digitalstrom-mqtt/issues). Nenne die Versionen der App, von Home Assistant und des dSS, deinen MQTT-Verbindungsweg, relevante Fehlermeldungen und das Verhalten nach einem App-Neustart. Entferne Passwörter, API-Keys, private Adressen und Gerätekennungen aus Logs und Screenshots, bevor du sie teilst.

### Weitere Hilfe

- [Ausführliche App-Dokumentation](https://github.com/gaetancollaud/digitalstrom-mqtt/blob/master/home-assistant-app/DOCS.md) - Einrichtung, Migration und Deinstallation.
- [Home-Assistant-Hilfe zu MQTT](https://www.home-assistant.io/integrations/mqtt/) - Einrichtung von Broker und Integration.
- [Home-Assistant-Backups und häufige Aufgaben](https://www.home-assistant.io/common-tasks/os/) - Einrichtung vor Änderungen schützen.

Überarbeitung der Anleitung: 27. September 2026. Menünamen können je nach Home-Assistant-Version abweichen.
