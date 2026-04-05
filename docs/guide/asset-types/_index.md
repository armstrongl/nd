---
title: "Asset types"
description: "Reference pages for each asset type nd recognizes: skills, agents, commands, rules, context, output styles, plugins, and hooks."
weight: 55
---

nd recognizes eight asset types, each with its own directory convention, file format, and deploy behavior.

{{< cards >}}
  {{< card link="skills" title="Skills" subtitle="Multi-file directory assets that package reusable coding-agent behaviors" >}}
  {{< card link="agents" title="Agents" subtitle="Single-file assets that define the behavior or persona of a named agent" >}}
  {{< card link="commands" title="Commands" subtitle="Single-file assets that register custom slash commands for agent sessions" >}}
  {{< card link="rules" title="Rules" subtitle="Single-file assets that define behavioral constraints an agent must follow" >}}
  {{< card link="context" title="Context" subtitle="Persistent instructions or project conventions deployed to fixed paths" >}}
  {{< card link="output-styles" title="Output styles" subtitle="Formatting instructions for agent output, activated via settings.json" >}}
  {{< card link="plugins" title="Plugins" subtitle="Plugin packages that bundle multiple assets for export and distribution" >}}
  {{< card link="hooks" title="Hooks" subtitle="Event-driven automation triggered by agent lifecycle events" >}}
{{< /cards >}}
