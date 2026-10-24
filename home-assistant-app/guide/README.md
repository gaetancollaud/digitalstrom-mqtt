# Illustrated beginner guides

- [English: install the Home Assistant App](beginner-guide.en.md)
- [Deutsch: Home-Assistant-App installieren](beginner-guide.de.md)

GitHub displays the screenshots and diagrams in these Markdown guides. Their
images are stored in `images/`; no build step is needed to read the guides.

## Editing

Keep the English and German steps aligned. Check image links after renaming a
file. To rebuild the shared diagrams after changing `render-diagrams.py`:

```sh
python -m pip install pillow==12.2.0
python render-diagrams.py
```

Use real App screenshots with example values. Before adding an image, check it
for passwords, keys, personal data, private network details and device names.
The current screenshots show a German Home Assistant interface; both language
guides use them. Check the App configuration and current release information
before changing installation steps.
