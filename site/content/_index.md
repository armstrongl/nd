---
title: nd
layout: hextra-home
---

{{< hextra/hero-badge link="https://GitHub.com/armstrongl/nd" >}}
  Open Source
{{< /hextra/hero-badge >}}

{{< hextra/hero-headline >}}
  nd
{{< /hextra/hero-headline >}}

{{< hextra/hero-subtitle >}}
  Coding agent asset manager. Deploy skills, commands, rules, and more via symlinks.
{{< /hextra/hero-subtitle >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-subtitle >}}
  <code>brew install armstrongl/tap/nd</code>
{{< /hextra/hero-subtitle >}}
</div>

{{< hextra/hero-button text="Get Started" link="docs/guide/getting-started/" >}}
{{< hextra/hero-button text="Command Reference" link="docs/reference/nd/" style="background: transparent; border: 1px solid #e5e7eb; color: inherit;" >}}

<div class="hx:mt-12"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Symlink-based deployment"
    subtitle="Assets stay in your source directories. nd creates symlinks so your coding agent finds them."
    link="docs/guide/how-nd-works/"
  >}}
  {{< hextra/feature-card
    title="Profiles & snapshots"
    subtitle="Switch between asset configurations instantly. Snapshots capture and restore your setup."
    link="docs/guide/profiles-and-snapshots/"
  >}}
  {{< hextra/feature-card
    title="Multiple sources"
    subtitle="Manage assets from local directories and git repositories. Sync them with one command."
    link="docs/guide/creating-sources/"
  >}}
{{< /hextra/feature-grid >}}
