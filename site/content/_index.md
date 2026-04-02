---
layout: hextra-home
title: nd
---

{{< hextra/hero-badge link="https://GitHub.com/armstrongl/nd/releases" >}}
  Latest release
{{< /hextra/hero-badge >}}

{{< hextra/hero-headline >}}
  nd
{{< /hextra/hero-headline >}}

{{< hextra/hero-subtitle >}}
  Coding agent asset manager
{{< /hextra/hero-subtitle >}}

```shell
brew install --cask armstrongl/tap/nd
```

{{< hextra/hero-button text="Get started" link="docs/guide/getting-started/" >}}

<div style="margin-top: 4rem;"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Symlink-based deployment"
    subtitle="Deploy assets as symlinks. Edit the source, changes show up instantly — no redeploy needed."
    icon="link"
    link="docs/guide/how-nd-works/"
  >}}
  {{< hextra/feature-card
    title="Profiles & snapshots"
    subtitle="Group assets into named profiles and switch between them. Snapshots let you bookmark and restore any state."
    icon="collection"
    link="docs/guide/profiles-and-snapshots/"
  >}}
  {{< hextra/feature-card
    title="Multiple sources"
    subtitle="Mix local directories and git repositories. Sync, scan, and manage assets from anywhere."
    icon="database"
    link="docs/guide/creating-sources/"
  >}}
{{< /hextra/feature-grid >}}
