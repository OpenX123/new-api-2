# New API Interface Context

This context distinguishes the public-facing site from the product interface available after authentication.

## Language

**Public site**:
The unauthenticated landing and informational experience.
_Avoid_: Homepage, console

**Authenticated console**:
The complete product interface available to signed-in users, including navigation, API keys, channels, and account management.
_Avoid_: Dashboard, admin panel

**Claude-inspired theme**:
A warm editorial visual language applied consistently to both light and dark interface modes.
_Avoid_: Dark-only theme, copied interface

**YiYongAI visual identity**:
An original product identity informed by the Claude-inspired theme rather than a reproduction of another product's interface.
_Avoid_: Claude clone, direct visual copy

**Visual redesign**:
A change to presentation and interaction that preserves existing routes, permissions, APIs, and business behavior.
_Avoid_: Product rewrite, workflow change

**Layout-preserving restyle**:
A visual redesign that retains each page's existing layout, information hierarchy, and data density.
_Avoid_: Layout rewrite, information-architecture change

**Operator branding**:
The runtime-configurable system name and logo shown across the public site and authenticated console.
_Avoid_: Source-coded brand, fixed operator identity

**Brand mark**:
An original warm-orange abstract Y symbol paired with the runtime-configured system name.
_Avoid_: Mascot, copied provider mark

**Fixed visual palette**:
One warm, editorial color system shared by the entire product while people retain light/dark, density, scale, and layout preferences.
_Avoid_: User-selectable color presets, global font overrides

## Relationships

- The **Public site** leads users into the **Authenticated console** after sign-in.
- The **Authenticated console** contains both user-facing and administrator-authorized areas.
- The **Claude-inspired theme** spans both the **Public site** and the **Authenticated console**.
- The **YiYongAI visual identity** expresses the **Claude-inspired theme** through original layouts and components.
- The **Visual redesign** spans the **Public site** and **Authenticated console** without changing business behavior.
- The **Layout-preserving restyle** constrains the **Visual redesign** to styling and micro-interactions.
- The **Operator branding** supplies the identity used by the **YiYongAI visual identity**.
- The **Brand mark** is one visual expression of the **Operator branding**.
- The **Fixed visual palette** keeps the **Layout-preserving restyle** visually coherent across every page.

## Example dialogue

> **Dev:** "Does this visual change apply to the **Public site** only?"
> **Operator:** "No, it must also apply to the **Authenticated console**, including API key and channel management."
