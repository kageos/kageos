# System settings navigation design QA

## Evidence

- Source visual truth: `docs/system-settings-option-1.png`
- Earlier desktop implementation: `docs/system-settings-option-1-implemented.jpg`
- Latest desktop implementation: `docs/system-settings-theme-aligned.jpg`
- Theme variants: `docs/system-settings-theme-variants.jpg` (Hub dark and Hub light)
- Mobile implementation: `docs/system-settings-option-1-mobile.jpg`
- Local route: `/system/settings?tab=appearance`
- State: Theme section selected; classic dark is the primary comparison state, with Hub dark and Hub light also rendered and checked
- Desktop viewport: 1280 × 720 CSS px, device scale factor 1
- Mobile viewport: 390 × 844 CSS px, device scale factor 1
- Source pixels: 1487 × 1058
- Desktop implementation pixels: 1280 × 720
- Mobile implementation pixels: 390 × 844
- Density normalization: for desktop full-view comparison, the source was cropped from the top to 1487 × 837 (matching the implementation aspect ratio) and resized to 1280 × 720. The focused navigation comparison used the left 236 × 720 region from both normalized images.

## Full-view comparison

The rendered page preserves the selected direction's dark full-height settings surface, compact left rail, violet active state, quiet content canvas, restrained dividers, and right-aligned utility actions. The implementation intentionally displays the real Theme section because the demo account cannot load populated system resource data; the generated source shows an illustrative Operations state. Main-content data rows were therefore not treated as pixel-comparable evidence for this navigation-focused change.

The desktop implementation uses the existing Inter-based product typography and KageOS color tokens. Heading hierarchy, neutral text contrast, icon stroke family, active violet, and panel/background balance match the source direction. The latest pass derives the page, sidebar, divider, hover, active fill, active marker, and mark colors from shared `--app-shell-*`, `--color-primary`, and text/border tokens. The implementation is slightly denser than the generated image, which is consistent with the stated goal of replacing the oversized card tabs with compact navigation.

The 390 × 844 capture shows the sidebar replaced by one labeled native category selector, followed by the selected section and its actions. There is no horizontal overflow, clipped control, or collapsed card content.

## Focused region comparison

The focused sidebar comparison confirms the same four groups and item order:

1. System operations: Operations, File assets
2. Accounts & access: Login settings, User management
3. Integrations: Data backup, Log archives (plus feature-flagged Email, OpenAPI, and Connectors when enabled)
4. Preferences: Theme, Language

Element Plus outline icons replace the generated mock's illustrative icons with the closest existing product-library equivalents. Active state uses both color and a left marker; text remains readable without relying on color alone.

## Required fidelity surfaces

- Fonts and typography: passed. Existing Inter/fallback stack retained; 11 px group labels, 14 px navigation labels, and 18–20 px headings create the intended hierarchy without wrapping or truncation in the tested Chinese state.
- Spacing and layout rhythm: passed. The 236 px rail, 40 px navigation rows, grouped vertical rhythm, 34–56 px responsive content padding, and flat full-height surface match the compact direction.
- Colors and visual tokens: passed. The earlier fixed Slate-800 sidebar was removed. The sidebar now mixes the active theme's shell background with 4% of its primary color; hover uses 6%, and the selected item uses a 14% primary mix. Classic dark, Hub dark, and Hub light all retain coherent surface hierarchy and active contrast.
- Image quality and asset fidelity: passed. The target contains no photographic or branded raster assets. UI icons use the existing Element Plus vector icon library; no placeholder, emoji, handcrafted SVG, or CSS illustration substitutes were introduced.
- Copy and content: passed. Existing localized section names and descriptions remain intact; four new group labels have Chinese and English translations.
- Responsiveness and accessibility: passed. Desktop navigation uses semantic `nav`, buttons, `aria-current`, and a labeled complementary region. Mobile uses a labeled native select with option groups and a visible focus ring. Primary controls remain at least 40–42 px high.

## Findings

No actionable P0, P1, or P2 findings remain.

- [P3] The generated source uses larger sidebar icons and looser rows than the implementation. The implementation keeps smaller existing library icons and denser rows to satisfy the original compact-navigation goal and to fit feature-flagged items without excessive scrolling.
- [P3] The implementation includes a short settings subtitle beside the sidebar mark. This adds useful context and remains visually subordinate; it does not alter navigation hierarchy.

## Comparison history

### Iteration 1 — blocked

- [P2] Data backup and Log archives were grouped under System operations, while the source grouped them under Integrations. With integration feature flags disabled, this also caused the Integrations heading to disappear.
- Fix: moved Data backup and Log archives into the Integrations group, before feature-flagged Email, OpenAPI, and Connectors.

### Iteration 2 — passed

- Revised desktop evidence: `docs/system-settings-option-1-implemented.jpg`
- Revised focused comparison confirms all four group headings, intended ordering, icon/title alignment, and active-state treatment.
- No actionable P0/P1/P2 differences remain.

### Iteration 3 — blocked after user review

- [P2] The classic-dark sidebar rendered as `#1e293b` while the main surface rendered as `#0f172a`. This full Slate-level jump made the navigation look like a separate theme, and the translucent active violet looked pasted onto it.
- Fix: introduced settings-local semantic colors derived from the active theme's shell, primary, text, and border tokens. The sidebar is now a subtle primary-tinted version of the page surface instead of a fixed secondary background.

### Iteration 4 — passed

- Latest desktop evidence: `docs/system-settings-theme-aligned.jpg`
- Additional theme evidence: `docs/system-settings-theme-variants.jpg`
- Classic dark now uses a subtle deep-blue/indigo surface relationship; Hub dark uses its near-black/brand-indigo palette; Hub light uses its fog-blue/indigo palette.
- The active fill, left marker, icon, divider, and heading mark change together with the selected theme. No actionable P0/P1/P2 color mismatch remains.

## Interaction and runtime checks

- Desktop sidebar button switching tested: Theme → Language updated the selected content and URL query to `tab=language`.
- Mobile category selector tested at 390 × 844: Theme → Language updated the URL query to `tab=language`.
- Operations route opened and its empty/permission state remained stable for the demo account.
- Console checked: one existing business-request error is emitted when the demo account reaches a system-super-admin-only resource endpoint, matching the visible permission message. No uncaught JavaScript exception or Vite error overlay was observed.
- Theme switching tested across classic dark, Hub dark, and Hub light; the original classic-dark selection was restored after verification. The final classic-dark page produced no console errors.
- ESLint passed for the changed Vue and locale files.
- Architecture boundary check passed.
- Production Vite build passed.
- Full `vue-tsc --build` is currently noisy outside this change because the workspace has duplicate TipTap/ProseMirror type installations; no error referenced the changed settings or locale files.

## Implementation checklist

- [x] Replace oversized card tabs with grouped compact navigation.
- [x] Use product icon library and stronger active affordance.
- [x] Preserve all feature-flagged settings and existing data behavior.
- [x] Convert Operations card tabs to lightweight underline tabs.
- [x] Add a mobile grouped category selector.
- [x] Verify desktop and mobile interactions.
- [x] Pass lint, architecture, build, and visual QA gates.

final result: passed
