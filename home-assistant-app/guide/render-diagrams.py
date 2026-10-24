"""Render the language-neutral connection diagram used by both guides."""

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent
OUTPUT = ROOT / "images" / "connection-flow.png"
MQTT_OUTPUT = ROOT / "images" / "mqtt-choice.png"
WIDTH, HEIGHT = 1800, 520
BACKGROUND = "#FFFFFF"
INK = "#17343A"
TEAL = "#087F83"
PALE = "#E9F5F3"
GREY = "#667B80"


def connection_flow() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    canvas = Image.new("RGB", (WIDTH, HEIGHT), BACKGROUND)
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default(size=40)
    small_font = ImageFont.load_default(size=25)
    label_font = ImageFont.load_default(size=29)
    draw.text((900, 48), "Commands / Befehle",
              anchor="mm", fill=INK, font=label_font)
    draw.line((1450, 88, 350, 88), fill=TEAL, width=6)
    draw.polygon(((350, 88), (372, 74), (372, 102)), fill=TEAL)
    boxes = [
        (45, 180, 370, 380, "digitalSTROM", "dSS"),
        (505, 180, 830, 380, "digitalSTROM", "App"),
        (965, 180, 1290, 380, "MQTT", "Broker"),
        (1425, 180, 1750, 380, "Home", "Assistant"),
    ]
    for left, top, right, bottom, first, second in boxes:
        draw.rounded_rectangle((left, top, right, bottom), radius=27,
                               fill=PALE, outline=TEAL, width=5)
        center = (left + right) / 2
        draw.text((center, 245), first, anchor="mm", fill=INK, font=font)
        draw.text((center, 310), second, anchor="mm", fill=GREY,
                  font=small_font)
    for start in (370, 830, 1290):
        end = start + 135
        y = 270
        draw.line((start + 17, y, end - 17, y), fill=TEAL, width=6)
        draw.polygon(((end - 27, y - 16), (end - 7, y),
                      (end - 27, y + 16)), fill=TEAL)
        draw.polygon(((start + 27, y - 16), (start + 7, y),
                      (start + 27, y + 16)), fill=TEAL)
    draw.text((900, 455), "State & events / Status & Ereignisse",
              anchor="mm", fill=INK, font=label_font)
    draw.line((350, 490, 1450, 490), fill=TEAL, width=6)
    draw.polygon(((1450, 490), (1428, 476), (1428, 504)), fill=TEAL)
    canvas.save(OUTPUT, optimize=True)
    print(OUTPUT)


def mqtt_choice() -> None:
    canvas = Image.new("RGB", (1800, 760), BACKGROUND)
    draw = ImageDraw.Draw(canvas)
    title_font = ImageFont.load_default(size=40)
    box_font = ImageFont.load_default(size=34)
    hint_font = ImageFont.load_default(size=23)
    lanes = [
        ("A", "Home Assistant MQTT service", "Mosquitto Broker App", 75),
        ("B", "Manual configuration", "Existing MQTT broker", 360),
    ]
    for letter, title, broker, top in lanes:
        draw.text((55, top + 17), letter, fill=TEAL, font=title_font)
        draw.text((145, top + 22), title, fill=INK, font=title_font)
        box_top = top + 100
        boxes = [
            (55, 525, "digitalSTROM App"),
            (670, 1140, broker),
            (1285, 1755, "Home Assistant"),
        ]
        for left, right, label in boxes:
            draw.rounded_rectangle((left, box_top, right, box_top + 105),
                                   radius=24, fill=PALE, outline=TEAL, width=4)
            center = (left + right) / 2
            draw.text((center, box_top + 52), label, anchor="mm",
                      fill=INK, font=box_font)
        for start, end in ((525, 670), (1140, 1285)):
            y = box_top + 52
            draw.line((start + 8, y, end - 8, y), fill=TEAL, width=6)
            draw.polygon(((end - 20, y - 15), (end - 2, y),
                          (end - 20, y + 15)), fill=TEAL)
            draw.polygon(((start + 20, y - 15), (start + 2, y),
                          (start + 20, y + 15)), fill=TEAL)
        draw.text((900, box_top + 135),
                  "Commands / Befehle   ·   State & events / Status & Ereignisse",
                  anchor="mm", fill=GREY, font=hint_font)
    canvas.save(MQTT_OUTPUT, optimize=True)
    print(MQTT_OUTPUT)


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    connection_flow()
    mqtt_choice()


if __name__ == "__main__":
    main()
