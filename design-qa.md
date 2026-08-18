# Design QA — 数字员工二次改版

## Evidence

- Latest user reference: `/var/folders/r5/6l6chs6x4sn8ky535l_qxsmw0000gn/T/codex-clipboard-0ecfdb5d-239a-48c3-85e8-9eff4762b46a.png`
- Main implementation screenshot: `/Users/beiluo/Documents/work/code/qiayanai.com/kageos/design-qa-digital-employees-v2.png`
- Dropdown implementation screenshot: `/Users/beiluo/Documents/work/code/qiayanai.com/kageos/design-qa-digital-employee-dialog-dropdown.png`
- Combined reference/implementation comparison: `/Users/beiluo/Documents/work/code/qiayanai.com/kageos/design-qa-digital-employee-dialog-comparison.png`
- Browser viewport: 1584 × 1024 CSS px
- Screenshot density: 1×
- Tested state: `/workspace`, `demos` workspace, 数字员工 tab, one selected employee; create dialog opened with model, overlap-policy, and datetime poppers exercised.

The supplied reference is a portrait crop rather than a full viewport. For the combined comparison, both sides were placed on equal 1584 × 1666 canvases without cropping the dialog. The focused region is the creation dialog because the reported defect was popper occlusion and the requested semantic changes are all visible there.

## Comparison

1. Global shell and service tree: preserved the existing kageos dark theme, spacing, navigation, and hierarchy. The service-tree employee mark is now a generated raster asset and remains recognizable at 24 px.
2. Employee roster: preserves real task status, schedule, next run, run count, description, and error data. The selected employee uses the generated state GIF matching the real state.
3. Employee hero and facts: title, status, role type, responsibilities, actions, schedule, next run, run count, work directory, overlap policy, model, and creator are readable without the old right column.
4. Main content and execution history: the former right-side runtime card is merged into the center facts grid. The center column gains width while responsibilities, work instructions, management actions, and execution records keep their hierarchy.
5. Creation/editing flow: visible terminology is now “新建数字员工 / 员工名称 / 员工职责 / 工作说明”. Default naming uses “demos 值守员”. Model, overlap-policy, and datetime poppers render above the dialog overlay and remain selectable.

## Asset QA

- `employee-ready.gif`: emerald/teal, 2 frames, 256 × 256.
- `employee-working.gif`: blue/cyan, 2 frames, 256 × 256.
- `employee-paused.gif`: amber, 2 frames, 256 × 256.
- `employee-failed.gif`: coral/red, 2 frames, 256 × 256.
- `service-icon.webp`: generated service-directory icon, 256 × 256.
- All character assets use transparent backgrounds, contain no text or watermark, and are bundled by the production build.

## Iteration history

- Rejected the first implementation capture because action buttons collapsed the hero title into a vertical column.
- Moved hero actions to their own row and changed facts to a responsive grid.
- Replaced the handcrafted SVG mascot with generated state-specific assets.
- Removed the persistent right aside and merged its real fields into the main column.
- Added a dialog-specific popper layer above the global overlay and verified all three popup types in the browser.

## Interaction and runtime checks

- Opened and closed the create-digital-employee dialog without creating data.
- Opened the model selector; all real model options were visible and unobstructed.
- Opened the overlap-policy selector; all three policies were visible.
- Opened the datetime picker; calendar panel was present and expanded.
- Browser console errors after the tested flow: none.
- `npm run type-check`: passed.
- Targeted Vitest suite: 2 files, 12 tests passed.
- `go test ./core/agent-server/service`: passed.
- `npm run build`: passed.
- `git diff --check`: passed.

## Final result

passed
