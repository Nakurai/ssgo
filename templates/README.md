# Templates

Each subfolder here is a self-contained ssgo **style**: a set of `*.html`
files including a `base.html` that defines the `base`, `content`, `style`,
and `scripts` blocks. They are independent of the binary and distributed as
zip archives.

Links should always open into a new tab.

## Authoring

Copy an existing folder (e.g. `minimal/`) and edit the HTML. A style must
contain at least `base.html`; per-page-type files (`index.html`, `page.html`,
`post.html`, `default.html`, `404.html`) override the `content`/`scripts`
blocks for that type, with `default.html` used as the fallback.

**Layout directives.** Content authors can wrap Markdown in `:::name … :::`
blocks, which render as `<div class="ssgo-<name>">…</div>` (same as how draft
banners use `.ssgo-banner`). A style should define the matching `.ssgo-*` rules in
its `base.html` `<style>` block — the built-in styles ship `.ssgo-hero` and
`.ssgo-centered`. Adding support for a new directive is just a new CSS rule; no
binary changes are needed.

## Packaging

Run `scripts/package-templates.sh` to build `dist/templates/<name>.zip` for
each style. These zips are release artifacts — publish them (GitHub releases,
a static host, etc.) and they are **not** committed to the repo.

## Using a template

    ssgo init --from https://example.com/minimal.zip   # new site
    ssgo style add --from ./minimal.zip --name minimal  # existing site
    ssgo style switch --name minimal                    # activate it
    ssgo style list                                     # see installed styles

`--from` also accepts a local directory or a local `.zip`.
