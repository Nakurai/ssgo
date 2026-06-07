# ssgo

A small static site generator. You write **Markdown**, pick a **type** for each
page, and `ssgo` renders a complete static website — HTML, CSS, client-side search,
and a clean URL structure. You never write HTML, CSS, or JavaScript: all styling
lives in the templates that ship with the tool, and you control the look through a
config file.

---

## Table of contents

- [Quick start](#quick-start)
- [The project layout](#the-project-layout)
- [Writing content](#writing-content)
  - [Front matter](#front-matter)
  - [Layout directives](#layout-directives)
  - [Content types](#content-types)
  - [Blog posts: drafts and scheduling](#blog-posts-drafts-and-scheduling)
  - [Tags](#tags)
  - [The blog post index](#the-blog-post-index)
  - [The navigation menu](#the-navigation-menu)
  - [Images and media](#images-and-media)
- [URLs](#urls)
- [Configuration (`ssgo.json`)](#configuration-ssgojson)
  - [Colors](#colors)
  - [Styles](#styles)
- [Search](#search)
- [Commands](#commands)
  - [`ssgo init`](#ssgo-init)
  - [`ssgo generate`](#ssgo-generate)
  - [`ssgo watch`](#ssgo-watch)
  - [`ssgo host`](#ssgo-host)
  - [`ssgo deploy`](#ssgo-deploy)
- [Deploying to Firebase](#deploying-to-firebase)

---

## Quick start

```sh
# 1. Create a new site in the current directory
ssgo init --url https://example.com

# 2. Preview it locally with live reload
ssgo watch          # serves http://localhost:8088

# 3. Build the production site
ssgo generate       # writes build/prod/

# 4. (Optional) configure a host and deploy
ssgo host firebase
ssgo deploy --build
```

`ssgo init` seeds the project with example content, so `generate` and `watch` work
immediately — you can edit the samples or replace them.

---

## The project layout

After `ssgo init` your directory looks like this:

```
my-site/
  ssgo.json                  # site configuration (title, colors, style, URLs, host)
  content/                  # your Markdown — this is where you work
    index.md
    about.md
    blog/
      index.md              # the post-index listing → /blog/
      hello.md
      second-post.md
  template/
    style/
      default/              # the shipped templates (HTML + CSS); editable
        base.html  page.html  post.html  post-index.html  index.html  404.html  default.html
  build/
    dev/                    # output of `ssgo watch`  (unminified, live reload)
    prod/                   # output of `ssgo generate` (minified, production)
```

You spend almost all of your time in **`content/`**. You touch **`ssgo.json`** to
change the title, colors, or URLs. You only go into `template/` if you want to
restyle the site (see [Styles](#styles)).

---

## Writing content

Every page is a single Markdown file inside `content/`. The folder structure under
`content/` becomes the URL structure of the site (see [URLs](#urls)).

A page is plain Markdown with an optional **front matter** block at the very top:

```markdown
---
type: page
title: About Us
in-menu: yes
---

# About

We make **good things**. Here is a [link](https://example.com) and an image:

![A diagram](./diagram.png)
```

### Front matter

Front matter is a YAML block delimited by `---` lines at the start of the file. It
sets metadata for the page. Every field is optional unless noted.

| Field      | Values                      | Default          | Applies to  | Meaning                              |
|------------|-----------------------------|------------------|-------------|--------------------------------------|
| `type`     | a type name (string)        | `default`        | all pages   | Selects which template renders it    |
| `title`    | string                      | the file name    | all pages   | The page title                       |
| `in-menu`  | `yes` / `no`                | `no`             | all pages   | Add this page to the site-wide nav   |
| `status`   | `draft` / `published`       | `draft`          | blog posts  | Whether the post is published        |
| `date`     | ISO 8601, e.g. `2025-01-15` | —                | blog posts  | Publish date (**required** for posts)|
| `tags`     | comma-separated strings     | —                | blog posts  | Labels, e.g. `tags: go, web, ssg`    |

A page with no front matter at all is valid — it renders with the `default` type.

### Layout directives

Sometimes plain Markdown isn't enough to express *layout* — you want a centered
image, or a hero section at the top of your home page. Directives let you do that
**without writing any HTML or CSS**. You wrap content in a named fenced block:

```markdown
:::hero
# Welcome to my site
A short, punchy tagline.
:::

:::centered
![My photo](./photo.jpg)
:::
```

The block opens with three or more colons followed by a **name**, and closes with
a line of colons on its own. Everything inside is ordinary Markdown — headings,
images, links, lists, even other directives all work as usual.

Each directive renders as `<div class="ssgo-<name>">…</div>`, and the matching
styling lives in your style's templates. The shipped styles include two built-in
directives:

| Directive    | Effect                                              |
|--------------|-----------------------------------------------------|
| `hero`       | A large, centered, padded banner — ideal at the top of the home page |
| `centered`   | Centers its content, including images               |

You can **nest** directives by giving the outer block more colons than the inner
one:

```markdown
::::hero
# Big heading
:::centered
![logo](./logo.png)
:::
::::
```

Directive *names* are all you ever need to know — never any CSS. Adding a brand-new
directive is a styling task for whoever maintains your templates (see
[Styles](#styles)); content authors just use the name.

### Content types

The `type` field picks the template used to render the page. The shipped `default`
style includes these types:

| `type`      | Use it for                                              |
|-------------|---------------------------------------------------------|
| `default`    | Anything — the fallback when no `type` is given                    |
| `page`       | Standalone pages (About, Contact, …)                              |
| `index`      | A landing/home page or a list page                               |
| `post`       | A single blog article (alias: see `blog-post` below)             |
| `post-index` | A listing of every blog post, with a date archive and tag list  |

> **Blog posts** are written with `type: blog-post`. They render through the
> `post` template and are the only type subject to draft/scheduling rules.

If you set a `type` that has no matching template, `ssgo` falls back to the
`default` template rather than failing the build.

### Blog posts: drafts and scheduling

Pages with `type: blog-post` have a publication lifecycle controlled by two fields:

```markdown
---
type: blog-post
title: Hello World
status: published
date: 2025-01-01
---

My first post.
```

- **`status`** defaults to `draft`. A post is only published when you explicitly
  set `status: published`. This is a safety default — you can never *accidentally*
  publish a post.
- **`date`** is **required** for blog posts and must be a valid ISO 8601 date. A
  post whose `date` is in the future is treated as *scheduled*.

What happens at build time depends on the command:

- **`ssgo generate` (production):** a blog post is **excluded** from the build if it
  is a draft *or* its date is in the future. It is skipped entirely — no page, no
  URL, no search entry, no nav link.
- **`ssgo watch` (preview):** **all** posts are shown so you can preview your work.
  Drafts and scheduled posts render with a visible banner at the top of the page
  (amber for drafts, blue for scheduled) so you can tell them apart from live
  content.

### Tags

Blog posts can carry **tags** — a comma-separated list of labels in the front
matter:

```markdown
---
type: blog-post
title: Hello World
status: published
date: 2025-01-01
tags: go, web, ssg
---
```

Tags are displayed on the post itself and are aggregated into a tag list on the
[blog post index](#the-blog-post-index). Whitespace around each tag is trimmed,
so `tags: go, web` and `tags: go,web` are equivalent. A page with no `tags`
field simply has none.

### The blog post index

The `post-index` type renders a listing of **every blog post** on the site. Give
any page that type — typically `content/blog/index.md`, which becomes `/blog/`:

```markdown
---
type: post-index
title: Blog
in-menu: yes
---

Optional intro text shown above the list.
```

The page shows, newest first, each post's title (linked), date, excerpt, and
tags. The excerpt is the post's `description` front matter if present, otherwise
the beginning of its body. Alongside the list, a sidebar holds two summaries
built automatically from the posts:

- a **date archive** — a year → month → day tree with the number of posts in
  each section;
- a **tag list** — every tag with the number of posts that use it.

Both update on every build; you never maintain them by hand. The same draft and
scheduling rules apply: in production the index lists only published, past-dated
posts; in `watch` it lists everything.

### The navigation menu

Any page can opt into the site-wide navigation bar by setting `in-menu: yes`:

```markdown
---
type: page
title: About
in-menu: yes
---
```

- The nav appears on every page and lists every opted-in page.
- Items are sorted **alphabetically by title**.
- In production, draft and scheduled posts never appear in the nav, even with
  `in-menu: yes`. In preview (`watch`), everything appears.

### Images and media

Reference images (and other media) from Markdown the normal way. Paths can be:

- **Relative** to the Markdown file: `![](./diagram.png)` or `![](images/logo.svg)`
- **Absolute** filesystem paths: `![](/Users/me/pictures/photo.jpg)`

When you build, `ssgo` copies every referenced file into a single `assets/` folder
in the output, names it by content hash (so identical files are stored once), and
rewrites the link to `/assets/<hash>.<ext>`. External links (`http://`, `https://`,
`//…`) and anchors are left untouched.

You never manage the `assets/` folder yourself — it is produced by the build.

---

## URLs

`ssgo` produces clean, pretty URLs from your folder structure. The file's path under
`content/` determines its URL:

| Markdown file              | Output                       | URL          |
|----------------------------|------------------------------|--------------|
| `content/index.md`         | `index.html`                 | `/`          |
| `content/about.md`         | `about/index.html`           | `/about/`    |
| `content/blog/hello.md`    | `blog/hello/index.html`      | `/blog/hello/` |

This means there are no `.html` extensions in your URLs.

`content/index.md` is the site's **home page** — it is the only file whose URL is `/` and it outputs `index.html` at the build root.

---

## Configuration (`ssgo.json`)

`ssgo init` writes an `ssgo.json` at the project root. It controls the site's
identity, look, and deployment target:

```json
{
  "title":   "My Site",
  "style":   "default",
  "logo":    "",
  "favicon": "",
  "host":    "",
  "colors": {
    "background": "#ffffff",
    "text":       "#1a1a1a",
    "primary":    "#2563eb",
    "secondary":  "#64748b",
    "surface":    "#f5f5f5"
  },
  "dev":  { "baseURL": "http://localhost:8088" },
  "prod": { "baseURL": "https://example.com" }
}
```

`logo` and `favicon` are empty by default; fill them in to add a header logo or a
favicon (see [Logo and favicon](#logo-and-favicon) for accepted path values).

| Field    | Meaning                                                                 |
|----------|-------------------------------------------------------------------------|
| `title`  | Site title, available to every template                                 |
| `style`  | The active style folder under `template/style/` (default `default`)     |
| `logo`    | Optional site logo shown in the header (see below); omit for none      |
| `favicon` | Optional site favicon (see below); omit for none                       |
| `host`   | Hosting provider codename (`""` = none, `"firebase"` = Firebase)         |
| `colors` | The theme palette (see below)                                           |
| `dev`    | Settings used by `ssgo watch` — the local base URL                       |
| `prod`   | Settings used by `ssgo generate` — the production base URL               |

There is no "mode" flag to remember: `ssgo generate` always uses the `prod` settings
and writes to `build/prod/`; `ssgo watch` always uses `dev` and writes to
`build/dev/`.

### Logo and favicon

The optional `logo` field puts an image in the site header, next to the title.
The header handles any aspect ratio gracefully: the logo is rendered at a fixed
height with automatic width, so wide wordmarks and square icons both sit
correctly without distortion. Leave `logo` out (or empty) and the header shows
the title text alone, exactly as before.

The `favicon` field works the same way and is shared across builds. Both `logo`
and `favicon` are site-wide, top-level fields, and both accept the same kinds of
value, resolved at build time:

- **A path relative to `ssgo.json`** — e.g. `"logo.png"` or `"assets/icon.svg"`.
  The file is copied into the output `assets/` folder (content-hashed and
  deduplicated, like [images](#images-and-media)) and the link is rewritten for
  you. If the file is missing, the build fails with a clear error.
- **A site-absolute path** (leading `/`, e.g. `"/favicon.ico"`) or an **external
  URL** (`https://…`, `//…`, `data:…`) — used verbatim, on the assumption you
  serve it yourself; nothing is copied.

### Colors

The five-color palette is applied on top of *any* style, so changing colors never
means editing a template. Each is a CSS color string; omit any field to use the
default.

| Key          | Role                                                  | Default   |
|--------------|-------------------------------------------------------|-----------|
| `background` | Page background                                       | `#ffffff` |
| `text`       | Body text                                             | `#1a1a1a` |
| `primary`    | Links, headings, primary accents                     | `#2563eb` |
| `secondary`  | Subtitles, dates, borders, muted text                | `#64748b` |
| `surface`    | Code blocks, cards, sidebars                          | `#f5f5f5` |

Edit a color, run `ssgo generate` (or just save while `ssgo watch` is running), and
the whole site updates.

### Styles

A **style** is a complete, self-contained set of templates living in one folder
under `template/style/`. Switching styles swaps the entire markup and CSS at once.

To use a different style:

1. Drop a folder of templates into `template/style/`, e.g. `template/style/dark/`.
2. Set `"style": "dark"` in `ssgo.json`.

That's it — no other wiring. The default templates installed by `ssgo init` live in
`template/style/default/`; copy that folder as a starting point for your own style.

A style folder must contain `base.html` (plus the per-type templates). If the named
style is missing or has no `base.html`, the build fails with a clear error listing
the styles that *are* available. Custom styles inherit your color palette
automatically as long as their CSS references the `--color-*` variables.

---

## Search

Every site gets **client-side full-text search** with no server required. At build
time `ssgo` writes a compact trigram index (`search-index.json`) alongside a small
search script. The shipped templates wire both in, so search works in the browser
out of the box. Drafts and scheduled posts that are excluded from a production
build are also excluded from its search index.

---

## Commands

`ssgo` has five commands.

### `ssgo init`

```sh
ssgo init [--url <prodURL>] [--force]
```

Scaffolds a new project in the current directory: creates `content/`,
`template/style/default/`, and `build/{dev,prod}/`; writes `ssgo.json` (with `--url`
as the production base URL, or a placeholder if omitted); installs the default
templates; and seeds `content/` with an example home page, an about page, a
couple of tagged blog posts, and a `post-index` page at `/blog/`.

Refuses to overwrite an existing `ssgo.json` unless you pass `--force`.

### `ssgo generate`

```sh
ssgo generate
```

Builds the **production** site into `build/prod/`. Renders every page, applies
blog-post filtering (drafts and future-dated posts are excluded), collects the nav,
copies and deduplicates media into `assets/`, builds the search index, and writes a
`404.html`. Output is minified and pages are rendered concurrently.

### `ssgo watch`

```sh
ssgo watch
```

Builds a **preview** site into `build/dev/` and serves it at
`http://localhost:8088` with **live reload** — edit a `.md` file, a template, or
`ssgo.json` and the browser refreshes automatically. The preview is unminified and
shows *all* content, including drafts and scheduled posts (with warning banners).

### `ssgo host`

```sh
ssgo host [<codename>]
```

One-time, interactive setup of a hosting provider. Writes the provider's config
files and records the provider in `ssgo.json`. Currently the only provider is
`firebase`. See [Deploying to Firebase](#deploying-to-firebase).

### `ssgo deploy`

```sh
ssgo deploy [--build]
```

Pushes `build/prod/` to the configured host. With no host configured it's a
friendly no-op. By default it deploys the existing `build/prod/` (and errors if
there isn't one yet); `--build` runs a full production build first.

---

## Deploying to Firebase

Firebase Hosting is the supported provider today. Setup assumes the Firebase
**project already exists** (create one at
<https://console.firebase.google.com>).

**Prerequisites:** install the Firebase CLI and log in once.

```sh
npm install -g firebase-tools
firebase login
```

**Configure the host** (interactive — you'll be asked for your Firebase project ID,
and optionally a multi-site hosting site ID):

```sh
ssgo host firebase
```

This writes `firebase.json` and `.firebaserc`, fixes the public directory to
`build/prod`, and sets `"host": "firebase"` in `ssgo.json`.

**Deploy:**

```sh
ssgo deploy --build    # build the prod site, then push it
# or, if build/prod/ is already up to date:
ssgo deploy
```

On success the live URLs (`https://<projectID>.web.app` and
`https://<projectID>.firebaseapp.com`) are printed.

## License

ssgo is licensed under the [PolyForm Noncommercial License 1.0.0](LICENSE). You
may use, modify, and share it freely for any noncommercial purpose. Selling the
software or any commercial use requires a separate license — contact
[nakurai](https://github.com/nakurai).
