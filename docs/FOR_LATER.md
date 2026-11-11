# For later

Ideas to design later, each in its own ADR when its time comes.

## A two-part skybox

Zoomed out (zoom 2) or with a free camera, the screen shows past the chunks you hold, and today that is the flat
`background` color. It should become a two-part skybox, drawn behind the world. The two parts are to be designed.

## Exploration remembered by the server

The map screen clears its gray veil only over the chunks held during this session, and the explored % starts again
on each load. The server should remember what each player explored, which is also what fog of war needs.

## Icons on screen

The camera lock gets an icon (a camera, with or without a padlock). Icons on screen need a proper design: where they
go, their size and style, how they group, and how other icons (health, chat, status…) join them.

## Choosing your robot

Each guest gets a random robot (ADR 0006). A menu to pick or change it, and the server picking it once accounts
exist, are to be designed.

## Item icons from the Pixel Art Atlas

[pixel-art-atlas-1.4m](https://huggingface.co/datasets/Angais/pixel-art-atlas-1.4m) holds 1.4M AI-made 16×16 sprites
(transparent, with a caption and a subject), good for inventory, equipment, shop and loot icons. Not for the world:
they are front or 3/4 views, not isometric.

- **Fit**: recolor them to our style (black outlines to our dark purple-gray, colors onto our palette). Captions are
  approximate, so look at each sprite picked.
- **How**: a script outside the client filters by `subject`, recolors, and exports one spritesheet + JSON that PixiJS
  loads. No need to download the whole 694 MB.
- **License**: the dataset is CC0, but the sprites are also Krea 2 "Outputs": commercial use only under $1M yearly
  revenue, an enterprise license above. AI-made art has little copyright (others may reuse it), may resemble
  someone's IP, and must be disclosed as AI (Steam asks).

## Rocks and players hidden behind hills

Rocks and players are drawn above all the ground (ADR 0007), so one standing just behind a raised cell isn't hidden by
it: up to 8 px wrong at their feet. Fixing it means sorting the ground with them, for example baking each chunk in
diagonal strips, one per depth line. It matters once taller art comes (trees, houses, taller cliffs), and how the
ground is drawn may change then.

Since ADR 0011, massifs and the glacier ring rise to +6, so a player just behind one is drawn over it, up to 48 px
wrong. Once players are sorted with the ground, a player hidden behind terrain should show as a silhouette through
it.

## Hand edits that survive the generator

The starter map is generated once, then edited by hand (ADR 0007). Generating it again overwrites the hand edits. Hand
edits could live in overlay files applied over the generated map, so the generator can run again, once there are real
hand-made places like the dome city.
