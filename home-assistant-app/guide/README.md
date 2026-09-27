# Illustrated beginner guides

- [English: install the Home Assistant App](beginner-guide.en.md)
- [Deutsch: Home-Assistant-App installieren](beginner-guide.de.md)

GitHub displays the screenshots and diagrams directly in these Markdown pages.
The images are stored in `images/` and linked with relative paths. Reading or
editing the guides does not require a PDF, a separate website or a build step.

## Editing

Edit the English or German Markdown file and check that the matching section in
the other language still says the same thing. Keep the image files and their
relative links together in this directory. If a shared diagram changes, rebuild
the two diagrams with:

```sh
python -m pip install pillow==12.2.0
python render-diagrams.py
```

Each Markdown file is the content source for its language and is also a
browser-readable help page. These commands are documentation tooling; they are
not part of installing the Home Assistant App.

## Optional local PDF export

The PDFs are not needed to read the guides on GitHub and are not tracked in this
repository. To make printable copies locally:

```sh
python -m pip install reportlab==4.4.9 pillow==12.2.0
python render-guide.py beginner-guide.en.md beginner-guide.en.pdf
python render-guide.py beginner-guide.de.md beginner-guide.de.pdf
```

The PDFs contain selectable text and links. The renderer works offline.

The export helper supports headings, paragraphs, single-line lists, bold and
inline code, HTTPS Markdown links, local Markdown images, explicit
`<!-- pagebreak -->` comments and non-rendering `<!-- figure-slot: ... -->`
comments. Figure slots record a caption and alt text for a later real capture;
they do not create an image or reserve PDF space. Image paths are relative to
this directory and must point to real files. Keep guide images free of
passwords, keys, personal data, private network details and identifiable device
names. Do not use mock screenshots. The current screenshots show the released
App with example addresses and a German interface. Both language guides use
the same images. The optional PDF enlarges the Start and switch controls from
the unmodified `images/app-info.png`; Markdown shows the complete screenshot.
If that image is replaced, review the
source-pixel viewport in `render-guide.py` and inspect the resulting PDF. The
connection log and discovered devices figure slots remain open until safe
captures are available.

Use section references when a section spans multiple PDF pages. After edits,
extract the PDF text and render all pages to inspect
legibility, margins, page breaks, images and links.

Before publication, recheck the guide against the current App `DOCS.md`,
`config.yaml`, English UI translations, release information and current Home
Assistant documentation. The 27 September 2026 edition describes the regular
released App **2.4.0-haos.1**, published as Docker Hub image tag
`gaetancollaud/digitalstrom-mqtt:2.4.0-haos.1`. A local production or test
variant is a separate build and is not the public end-user release. A source
version alone is not proof of release availability; verify the published App
repository and image before changing that statement.
