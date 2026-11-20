# Controller icon resources

This directory includes every icon from the **generic**, **Xbox** and
**PlayStation** families of the supplied Flat Gamepad Icons 1.0 ZIP archive.

The filenames are the original icon IDs from the distribution, with the
source `.150dpi.png` suffix normalized to `.png` (pixels unchanged).
PNG transparency is preserved. **38 icons** are available offline.

Use the shared registry in `../controller-icons.js`:

```js
VoyagerGamepadIcons.icon("xb_a", "Botão A");
VoyagerGamepadIcons.icon("ps_circle", "Círculo");
VoyagerGamepadIcons.button(0, "playstation"); // Face South (×)
VoyagerGamepadIcons.button(1, "xbox");        // Face East (B)
VoyagerGamepadIcons.stick("left", "up");
```

To add future Big Picture actions, use the standard Gamepad API indices
(0–16) and `VoyagerGamepadIcons.button(index, family)`.
Controllers detected as DualShock or DualSense use PlayStation glyphs;
all others use Xbox glyphs, while directional icons remain generic.

See LICENSE.md for attribution and CC BY-SA 4.0 conditions.
