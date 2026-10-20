#!/usr/bin/env python3
"""Build schemas/gear-map.json: which real-world gear each Line 6 model emulates.

Line 6 renames every model for trademark reasons — an Ampeg SVT ships as
HD2_AmpSVBeastNrm — and publishes the decoder ring in their own manual. Both
halves ship inside HX Edit:

    Contents/Resources/*.models              symbolicID  <->  name
    Contents/Resources/HX Edit Pilot's Guide.pdf   name  <->  based on

Joining them on `name` yields symbolicID -> real gear, which is what makes
artist knowledge usable: knowing someone plays an SVT is worthless until you
know which of 681 strings to write.

This is bootstrap data. It changes only when Line 6 ships new models, so the
result is committed rather than rebuilt. Run it after an HX Edit update:

    just gear-map

Python rather than Go because the model-name column uses a subset-embedded font
whose CMap rsc.io/pdf cannot resolve. pypdf copes. The frequently regenerated
artifact — the catalog — is Go.
"""

from __future__ import annotations

import collections
import glob
import json
import os
import pathlib
import re
import sys
import warnings

warnings.filterwarnings("ignore")

RESOURCES = "/Applications/Line6/HX Edit.app/Contents/Resources"
# Beside this script, so it does not matter which directory you run it from.
OUT = pathlib.Path(__file__).with_name("gear-map.json")

# The model lists Line 6 keep from an older release, which the guide gives a
# separate section and a "Legacy" prefix.
#
# Only the cabinets, measured: of the 661 models every family's entries are
# current except cab's 36, which are all legacy. The same speaker also ships as
# cabmicirs and cabmicirswithpan, which are the current ones, so a name in this
# section belongs to those and its Legacy twin belongs here.
LEGACY_FAMILIES = {"cab"}
# Column two of the manual's table. It is a phrase, not a word — "Mono,
# Stereo" and "Single, Dual" both occur — so it is consumed as a whole or the
# tail of it leaks into the gear name ("Stereo Klon Centaur").
SUBCATEGORY = re.compile(
    r"^((?:Guitar|Bass|Preamp|Legacy|Pedal|Chorus|Mono|Stereo|Single|Dual)"
    r"(?:\s*[,/]\s*(?:Mono|Stereo|Single|Dual|Preamp))*)\s+(.+)$"
)


def shown_names(resources: str) -> dict[str, str]:
    """Map each symbolic ID to the name HX Edit shows a person.

    Line 6 ship every name twice and the copies disagree for 25 of the 654
    models in both files. The guide is written against this one: it prints
    "8x10 Ampeg SVT E" where the .models file says "8x10 Ampeg SVT-E", so a
    join on the .models name alone misses the row.
    """
    path = os.path.join(resources, "HX_ModelCatalog.json")
    if not os.path.exists(path):
        return {}

    with open(path, encoding="utf-8", errors="replace") as fh:
        tree = json.load(fh)

    out: dict[str, str] = {}

    def walk(node):
        if isinstance(node, dict):
            model_id, name = node.get("id"), node.get("name")
            if isinstance(model_id, str) and isinstance(name, str) and model_id and name:
                out.setdefault(model_id, name)
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(tree)
    return out


def model_names(resources: str) -> dict[str, list[tuple[str, str]]]:
    """Map each model's display name to every symbolic ID that carries it.

    A name is not unique: the same amp ships as a full amp and as a preamp, so
    "Ampeg SVT Nrm" is both HD2_AmpSVBeastNrm and HD2_PreampSVBeastNrm. Both
    are real models and both need the mapping.

    Indexed under both of Line 6's spellings where they differ, because the
    guide follows the app's and this used to follow the model file's. 17 models
    carried no real-world gear for that reason alone, among them the 8x10
    Ampeg the guide documents on page 69 as a `Legacy 8x10" Ampeg SVT`. A model
    with no gear is a model nothing can ask for by the name of the thing it is.

    Both rather than only the app's, because the app's is the one that moved: a
    guide printed against an older release still matches the model file.
    """
    shown = shown_names(resources)
    by_name: dict[str, list[tuple[str, str]]] = collections.defaultdict(list)

    for path in sorted(glob.glob(os.path.join(resources, "*.models"))):
        family = os.path.basename(path).replace(".models", "")
        with open(path, encoding="utf-8", errors="replace") as fh:
            for model in json.load(fh):
                model_id = model["symbolicID"]
                names = {(model.get("name") or "").strip(), shown.get(model_id, "").strip()}

                for name in sorted(names):
                    if name:
                        by_name[name].append((model_id, family))

    return by_name


def based_on(resources: str, by_name) -> dict[str, dict]:
    """Read the manual's model table and join it to symbolic IDs."""
    import pypdf

    guide = os.path.join(resources, "HX Edit Pilot's Guide.pdf")
    reader = pypdf.PdfReader(guide)

    # Every page, not only the ones carrying the "Based On" header. A table runs
    # over several pages and the header is printed once, so filtering on it read
    # the first page of each table and dropped the rest. The cabinets lost three
    # pages that way: the current models' rows live there, the Legacy section's
    # header is on the page after them, and the 58 mic'd cabinets ended up with
    # their Legacy twin's entry because it was the only line anything saw.
    #
    # Safe to widen because a row has to be recognised anyway: it starts with a
    # model's own display name and is followed by a subcategory phrase from a
    # closed list. Prose does not do that.
    found: dict[str, dict] = {}
    for page in reader.pages:
        text = page.extract_text() or ""
        for line in text.split("\n"):
            line = line.strip()
            if not line or "Based On" in line:
                continue
            # Longest matching model name wins: "A30 Fawn Brt" before "A30 Fawn".
            candidates = [n for n in by_name if line.startswith(n + " ")]
            if not candidates:
                continue
            name = max(candidates, key=len)
            rest = SUBCATEGORY.match(line[len(name):].strip())
            if not rest:
                continue
            based = rest.group(2).strip()
            for symbolic_id, family in by_name[name]:
                # A name can appear twice in the guide, once in the current
                # section and once under Legacy, and the same cabinet ships in
                # three families under one name. Pairing them by whether the
                # entry says Legacy keeps each model naming the gear it
                # emulates; taking whichever line came last gave all three the
                # Legacy one, because the legacy section is further into the
                # book. 58 models claimed to be legacy models and were not.
                if based.startswith("Legacy ") != (family in LEGACY_FAMILIES):
                    continue
                found[symbolic_id] = {
                    "symbolic_id": symbolic_id,
                    "name": name,
                    "family": family,
                    "subcategory": rest.group(1).rstrip(","),
                    "based_on": based,
                }
    return found


def main() -> int:
    if not os.path.isdir(RESOURCES):
        print(
            f"HX Edit not found at {RESOURCES}.\n"
            "This data comes from a licensed HX Edit installation and cannot be "
            f"derived any other way. Install HX Edit, or keep the committed "
            f"{OUT.name}.",
            file=sys.stderr,
        )
        return 1

    by_name = model_names(RESOURCES)
    models = based_on(RESOURCES, by_name)
    if not models:
        print("no models mapped — the manual's table format may have changed", file=sys.stderr)
        return 1

    payload = {
        "source": {
            "model_names": "HX Edit.app/Contents/Resources/*.models",
            "based_on": "HX Edit.app/Contents/Resources/HX Edit Pilot's Guide.pdf",
        },
        "entries": len(models),
        "models": models,
    }
    with open(OUT, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, indent=2, sort_keys=True)
        fh.write("\n")

    families = collections.Counter(m["family"] for m in models.values())
    print(f"wrote {OUT}: {len(models)} models")
    for family, count in families.most_common(6):
        print(f"  {count:4}  {family}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
