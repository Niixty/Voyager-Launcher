"use strict";

/**
 * Flat Gamepad Icons by RobTheFiveNine (CC BY-SA 4.0).
 * Registry shared by every future Big Picture screen. All 38 original icon
 * variants (generic, Xbox and PlayStation) are bundled offline at 150 DPI.
 *
 * Examples:
 *   VoyagerGamepadIcons.icon("xb_a", "A");
 *   VoyagerGamepadIcons.button(0, "playstation"); // PS Cross
 *   VoyagerGamepadIcons.button(1, "xbox");        // Xbox B
 *   VoyagerGamepadIcons.stick("left", "up");      // Generic left-stick up
 */
const VoyagerGamepadIcons = (() => {
  const families = Object.freeze({
    generic: Object.freeze([
      "dpad_down", "dpad_left", "dpad_right", "dpad_up",
      "l3", "left_stick", "left_stick_down", "left_stick_left",
      "left_stick_right", "left_stick_up", "r3", "right_stick",
      "right_stick_down", "right_stick_left", "right_stick_right",
      "right_stick_up", "select", "start",
    ]),
    xbox: Object.freeze([
      "a", "b", "lb", "lt", "rb", "rt",
      "select", "start", "super", "x", "y",
    ]),
    playstation: Object.freeze([
      "circle", "l1", "l2", "r1", "r2",
      "square", "super", "triangle", "x",
    ]),
  });
  const prefixes = Object.freeze({ generic: "generic", xbox: "xb", playstation: "ps" });
  const names = new Set(
    Object.entries(families).flatMap(([family, items]) =>
      items.map(name => prefixes[family] + "_" + name),
    ),
  );
  const standardButtons = Object.freeze({
    0: ["xb_a", "ps_x"],
    1: ["xb_b", "ps_circle"],
    2: ["xb_x", "ps_square"],
    3: ["xb_y", "ps_triangle"],
    4: ["xb_lb", "ps_l1"],
    5: ["xb_rb", "ps_r1"],
    6: ["xb_lt", "ps_l2"],
    7: ["xb_rt", "ps_r2"],
    8: ["xb_select", "generic_select"],
    9: ["xb_start", "generic_start"],
    10: ["generic_l3", "generic_l3"],
    11: ["generic_r3", "generic_r3"],
    12: ["generic_dpad_up", "generic_dpad_up"],
    13: ["generic_dpad_down", "generic_dpad_down"],
    14: ["generic_dpad_left", "generic_dpad_left"],
    15: ["generic_dpad_right", "generic_dpad_right"],
    16: ["xb_super", "ps_super"],
  });

  function icon(name, alt = "") {
    if (!names.has(name)) return null;
    const img = document.createElement("img");
    img.src = "/controller-icons/" + name + ".png";
    img.alt = alt;
    img.className = "gp-icon";
    img.decoding = "async";
    img.draggable = false;
    return img;
  }
  function buttonName(index, family = "xbox") {
    const match = standardButtons[index];
    if (!match) return null;
    if (family === "generic" && index <= 3) return null;
    return family === "playstation" ? match[1] : match[0];
  }
  function button(index, family = "xbox", alt = "") {
    const name = buttonName(index, family);
    return name ? icon(name, alt) : null;
  }
  function stick(side = "left", direction = "") {
    const suffix = side === "right" ? "right_stick" : "left_stick";
    const dir = ["up", "down", "left", "right"].includes(direction) ? "_" + direction : "";
    return icon("generic_" + suffix + dir);
  }
  function detect(pad) {
    // The Web Gamepad API uses the same "standard" button indices for Xbox
    // and PlayStation, but its pad.id can still identify the glyph family.
    const id = String(pad?.id || "");
    if (/playstation|dualshock|dualsense|wireless controller|sony|054c|0ce6/i.test(id))
      return "playstation";
    return "xbox";
  }
  return Object.freeze({
    families,
    names: Object.freeze([...names]),
    standardButtons,
    icon, buttonName, button, stick, detect,
    has(name) { return names.has(name); },
  });
})();
window.VoyagerGamepadIcons = VoyagerGamepadIcons;
