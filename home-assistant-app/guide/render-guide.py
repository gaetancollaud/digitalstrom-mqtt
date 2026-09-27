"""Build the PDF from beginner-guide.en.md. Requires reportlab==4.4.9.

Usage: python render-guide.py [source.md] [output.pdf]
Supported source format: headings, paragraphs, single-line lists, bold,
inline code, HTTPS Markdown links, local Markdown images and explicit
pagebreak comments. No network access is used.
"""
from pathlib import Path
import re
import sys
from html import escape

from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.units import mm
from reportlab.lib.utils import ImageReader
from reportlab.platypus import Flowable, Image, PageBreak, Paragraph, SimpleDocTemplate

ROOT = Path(__file__).resolve().parent
SOURCE = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / 'beginner-guide.en.md'
OUTPUT = Path(sys.argv[2]) if len(sys.argv) > 2 else ROOT / 'beginner-guide.en.pdf'
IS_GERMAN = SOURCE.name.endswith('.de.md')
BODY_SIZE = 10 if IS_GERMAN else 10.6
BODY_LEADING = 13.5 if IS_GERMAN else 14.7
INK = colors.HexColor('#203343')
TEAL = colors.HexColor('#007C83')
styles = {
    'body': ParagraphStyle('body', fontName='Helvetica', fontSize=BODY_SIZE,
                           leading=BODY_LEADING, textColor=INK, spaceAfter=6 if IS_GERMAN else 7),
    'h1': ParagraphStyle('h1', fontName='Helvetica-Bold', fontSize=28,
                         leading=33, textColor=TEAL, spaceAfter=13),
    'h2': ParagraphStyle('h2', fontName='Helvetica-Bold', fontSize=21,
                         leading=26, textColor=TEAL, spaceAfter=15,
                         keepWithNext=True),
    'h3': ParagraphStyle('h3', fontName='Helvetica-Bold', fontSize=12.2,
                         leading=16, textColor=INK, spaceBefore=7,
                         spaceAfter=6, keepWithNext=True),
    'list': ParagraphStyle('list', fontName='Helvetica', fontSize=BODY_SIZE,
                           leading=BODY_LEADING, textColor=INK, leftIndent=13,
                           firstLineIndent=-13, spaceAfter=5 if IS_GERMAN else 6),
}
styles['list-keep'] = styles['list'].clone('list-keep', keepWithNext=True)
styles['body-keep'] = styles['body'].clone('body-keep', keepWithNext=True)


class AppInfoViewport(Flowable):
    """Show the controls in the original screenshot without editing it."""

    def __init__(self, path):
        super().__init__()
        self.reader = ImageReader(str(path))
        self.image_width, self.image_height = self.reader.getSize()
        # Source-pixel coordinates of the right-hand Start/toggles card.
        self.left, self.top, self.right, self.bottom = 735, 75, 1220, 410
        if not (0 <= self.left < self.right <= self.image_width and
                0 <= self.top < self.bottom <= self.image_height):
            raise ValueError(f'Invalid app info viewport for {path}')
        self.width = 145*mm
        self.scale = self.width / (self.right-self.left)
        self.height = (self.bottom-self.top)*self.scale
        self.hAlign = 'CENTER'

    def draw(self):
        self.canv.saveState()
        clip = self.canv.beginPath()
        clip.rect(0, 0, self.width, self.height)
        self.canv.clipPath(clip, stroke=0, fill=0)
        self.canv.drawImage(self.reader, -self.left*self.scale,
                            -(self.image_height-self.bottom)*self.scale,
                            width=self.image_width*self.scale,
                            height=self.image_height*self.scale)
        self.canv.restoreState()

def inline(value):
    value = escape(value)
    value = re.sub(r'\[([^\]]+)\]\((https://[^\s)]+)\)',
                   r'<link href="\2" color="#007C83"><u>\1</u></link>', value)
    value = re.sub(r'`([^`]+)`', r'<font name="Courier">\1</font>', value)
    return re.sub(r'\*\*(.+?)\*\*', r'<b>\1</b>', value)

def footer(canvas, doc):
    canvas.saveState()
    width, height = A4
    canvas.setStrokeColor(TEAL)
    canvas.setLineWidth(1)
    canvas.line(20*mm, height-15*mm, width-20*mm, height-15*mm)
    canvas.setFont('Helvetica', 8)
    canvas.setFillColor(INK)
    edition = '27. September 2026' if IS_GERMAN else '27 September 2026'
    canvas.drawString(20*mm, 13*mm,
                      f'digitalSTROM MQTT | App 2.4.0-haos.1 | {edition}')
    canvas.drawRightString(width-20*mm, 13*mm, str(doc.page))
    canvas.restoreState()

story = []
blocks = re.split(r'\n\s*\n', SOURCE.read_text(encoding='utf-8').strip())


def is_list(block):
    return bool(re.match(r'(- |\d+\. )', block))


for block_index, block in enumerate(blocks):
    if block == '<!-- pagebreak -->':
        story.append(PageBreak())
    elif re.fullmatch(r'<!-- figure-slot: .+ -->', block, flags=re.DOTALL):
        continue
    elif (match := re.fullmatch(r'!\[([^\]]*)\]\(([^)]+)\)', block)):
        image_root = SOURCE.parent.resolve()
        image_path = (image_root / match.group(2)).resolve()
        try:
            image_path.relative_to(image_root)
        except ValueError as error:
            raise ValueError('Guide images must stay inside the source directory') from error
        if not image_path.is_file():
            raise FileNotFoundError(f'Guide image not found: {image_path}')
        if image_path == (image_root / 'images' / 'app-info.png').resolve():
            # The PDF shows the controls; Markdown keeps the full PNG.
            picture = AppInfoViewport(image_path)
        else:
            picture = Image(str(image_path))
            scale = min((145*mm) / picture.imageWidth,
                        (100*mm) / picture.imageHeight, 1)
            picture.drawWidth = picture.imageWidth * scale
            picture.drawHeight = picture.imageHeight * scale
            picture.hAlign = 'CENTER'
        # If a heading introduces an illustrated step/options list, keep the
        # image and its caption with the following intro and list.
        next_blocks = blocks[block_index + 1:block_index + 3]
        keep_image = (block_index > 0 and blocks[block_index - 1].startswith('### ')
                      and bool(next_blocks) and
                      (is_list(next_blocks[0]) or
                       (len(next_blocks) > 1 and is_list(next_blocks[1]))))
        picture.keepWithNext = keep_image
        story.append(picture)
        if match.group(1):
            caption_style = styles['list-keep'] if keep_image else styles['list']
            story.append(Paragraph(inline(match.group(1)), caption_style))
    elif block.startswith('# '):
        story.append(Paragraph(inline(block[2:]), styles['h1']))
    elif block.startswith('## '):
        story.append(Paragraph(inline(block[3:]), styles['h2']))
    elif block.startswith('### '):
        story.append(Paragraph(inline(block[4:]), styles['h3']))
    elif is_list(block):
        lines = block.splitlines()
        # Keep short lists intact. A list immediately before an explicit page
        # break uses the remaining space on that page (the intro navigation).
        keep_list = len(lines) <= 5 and (
            block_index + 1 == len(blocks) or blocks[block_index + 1] != '<!-- pagebreak -->')
        for index, line in enumerate(lines):
            if not re.match(r'(- |\d+\. )', line):
                raise ValueError('List items must be single lines')
            text = ('&#8226; ' + inline(line[2:])) if line.startswith('- ') else inline(line)
            # Chained items also keep the preceding h3 with the complete list.
            style = styles['list-keep'] if keep_list and index < len(lines) - 1 else styles['list']
            story.append(Paragraph(text, style))
    else:
        keep_intro = block_index + 1 < len(blocks) and is_list(blocks[block_index + 1])
        style = styles['body-keep'] if keep_intro else styles['body']
        story.append(Paragraph(inline(block.replace('\n', ' ')), style))

SimpleDocTemplate(str(OUTPUT), pagesize=A4, rightMargin=20*mm, leftMargin=20*mm,
                  topMargin=23*mm, bottomMargin=23*mm,
                  title='digitalSTROM MQTT - Anleitung für Einsteiger' if IS_GERMAN else 'digitalSTROM MQTT - Beginner guide',
                  author='digitalSTROM MQTT community documentation',
                  subject='Home Assistant App installation and first setup').build(
                      story, onFirstPage=footer, onLaterPages=footer)
print(OUTPUT.resolve())
