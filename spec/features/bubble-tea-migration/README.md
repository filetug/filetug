---
format: https://specscore.md/feature-specification
status: Draft
---

# Feature: Bubble Tea migration

> [SpecScore.**Studio**](https://specscore.studio): | [Explore](https://specscore.studio/app/github.com/filetug/filetug/spec/features/bubble-tea-migration?op=explore) | [Edit](https://specscore.studio/app/github.com/filetug/filetug/spec/features/bubble-tea-migration?op=edit) | [Ask question](https://specscore.studio/app/github.com/filetug/filetug/spec/features/bubble-tea-migration?op=ask) | [Request change](https://specscore.studio/app/github.com/filetug/filetug/spec/features/bubble-tea-migration?op=request-change) |
**Status:** Draft
**Source Ideas:** —

## Summary

Filetug will fully cut over from `tview` and `tcell` to Charm in one release through an architectural UI and state rewrite.

The target stack is Bubble Tea v2, Bubbles v2, Lip Gloss v2, and Glamour v2.

A keyboard-driven three-panel internal milestone is estimated at 5–8 working
days for one experienced Go developer or roughly 6–12 focused agent-hours plus
terminal QA. Full production parity through the approved hard-cutover route is
estimated at 2.5–4.5 developer weeks or 2.5–5 focused agent-days plus review,
CI, release coordination, and human terminal smoke testing.

“Migrate” means full cutover: all Filetug surfaces move to Charm, the old UI
implementation is removed, and no Filetug production or test code depends on
`tview` or `tcell` when the implementation lands.

## Decisions

### D1 — Single-release hard cutover

**Approved:** 2026-08-28 by the founder.

- Filetug will not ship a public `--ui` selector or a release containing both
  UI implementations.
- The implementation starts from the latest `origin/main` in an isolated WB
  worktree. The canonical clone is read-only source and synchronization state,
  never the implementation checkout.
- The existing executable may remain temporarily callable inside the private
  implementation worktree while the Bubble Tea v2 packages compile and gain
  targeted tests. This is an implementation technique, not a product mode.
- The cutover candidate switches the production entry point once, then removes
  the old root, panels, test helpers, and module dependencies before landing.
- No Filetug consumer is retained on `tview` or `tcell` after the cutover.
- Operational rollback is the immediately previous Filetug release or a
  forward revert, not an in-product backend selector.

**Reason:** this removes the public selector, two-backend regression matrix,
rollback compatibility layer, later default switch, and separate retirement
campaign. The estimated saving is 0.5–1.5 focused agent-days relative to a
released dual-mode rollout. The accepted trade-off is a larger single release
with no runtime fallback.

## Implementation Handoff

This Feature is the authoritative plan for the next implementation agent. This
planning effort does not authorize or contain migration code.

1. Re-read the exact current `origin/main` and inventory every old UI consumer;
   the 2026-08-28 counts below are a baseline, not a substitute for a fresh
   cutover inventory.
2. Resolve Open Question 2 before changing `strongo/strongo-tui`. The
   recommended boundary is a provider-first addition of backend-neutral theme
   tokens and a Lip Gloss v2 adapter while its existing `tview` adapter remains
   for unrelated repositories; Filetug must stop consuming that old adapter.
3. Build the Bubble Tea v2 model in Filetug behind ordinary package boundaries
   while the current root keeps the worktree compilable. Do not add a public
   selector, ship an alternate binary, or merge an intermediate dual stack.
4. Treat the three-panel MVP as an internal test checkpoint. Continue through
   every parity phase before switching the production root.
5. Land only when all Acceptance Criteria are satisfied, `origin/main` contains
   the exact candidate, and every related branch and worktree is cleaned.

## End-to-End Journey

### User story

As a Filetug user, I run `ft <path>` and see the directory tree, file list,
and preview without doing anything else. I navigate directories and files with
the keyboard; each other panel reflects the current selection without stale
background work overwriting newer state. I can then continue browsing or quit,
and quitting restores the terminal.

### Start

1. The user starts `ft <path>`.
2. With no further input, the three-panel layout appears with the directory
   represented in the tree, its entries in the file list, and a truthful
   loading, empty, error, or preview state in the preview panel.

**Observable good result:** the first frame is responsive, exactly one panel is
focused, the selected path is consistent across all three panels, and slow I/O
does not block key handling.

### Middle

1. The user moves in the directory tree and opens a directory.
2. The file panel shows that directory while the preview panel reports loading
   or the selected entry.
3. The user moves rapidly from file A to file B while file A's preview is still
   loading.

**Observable good result:** the tree, file list, breadcrumbs, focus indicator,
and preview agree on the current path; file A's late result is discarded and
can never replace file B's preview.

### End

1. The user selects a supported text or Markdown file.
2. The preview shows bounded, scrollable content with syntax or Markdown
   styling.

**Observable good result:** selection remains usable while the preview loads,
content is clipped or wrapped to the panel, and errors are visible in the UI.

### Epilogue branches

- **Close:** the user quits. The application cancels owned work, exits without a
  leaked process, and restores the terminal cursor, screen, and input mode.
- **Continue or replay:** the user navigates to another directory or revisits a
  prior one. The same journey repeats with no state from the earlier directory
  leaking into the new projection.

## Problem

The current UI directly exposes `tview` primitives and `tcell` events across
navigation, panels, previewers, layout, focus, color, mouse handling, and test
seams. Callback-driven mutation and `QueueUpdateDraw` also mix application
state with rendering. That makes a direct component swap impossible and makes
asynchronous correctness harder to reason about.

The goal is not a cosmetic rewrite. The new architecture should establish one
message-driven state owner, preserve Filetug's differentiated filesystem and
Git behavior, make stale async results structurally rejectable, and finish by
removing the old UI stack.

## Feasibility Analysis

Assessment baseline: `filetug/filetug` commit
`01a1535870838d2e9722fc1fd5782f189a5a2668` on 2026-08-28.

| Dimension | Verdict | Evidence and consequence |
|---|---|---|
| Technical feasibility | High | Bubble Tea v2 supports full-window applications, commands/messages, keyboard and mouse input, and declarative terminal state. Bubbles v2 provides interactive tree, table, list, text input, spinner, help, and viewport models. |
| Mechanical replaceability | Low | Filetug has 50 production files (6,551 lines) and 37 test files (10,950 lines) directly importing `tview` or `tcell`. Approximately 66% of production Go lines are directly UI-coupled. |
| Domain reuse | High | `pkg/files`, `pkg/gitutils`, `pkg/fsutils`, persistence, masks, and much of preview data extraction are reusable without changing their product behavior. |
| Incremental development | Medium to high | A separate Bubble Tea model can initially reuse stores and Git services while the old executable keeps only the private worktree compilable. No intermediate dual-stack release is permitted. |
| Performance confidence | Medium | Bubble Tea v2 has an optimized cell renderer, but Filetug's current file list uses lazy `tview.TableContent`; Bubbles v2 tables materialize rows. A custom virtualized file-list model or measured list delegate is required before claiming parity. |
| Testability | Medium to high | Pure `Update` and `View` behavior is easier to test deterministically. Filetug must replace 12 simulation/draw test files and numerous callback mocks with reducer, render, race, and PTY journey tests. |
| Migration risk | Medium to high | The main risks are state ownership, stale async results, focus/key routing, large directories, mouse hit testing, Unicode width, terminal restoration, and maintaining the repository's 100% coverage gate. |

The current baseline passes `go test ./...`. No local or remote Filetug or
`strongo-tui` branch currently contains a Bubble Tea migration.

### Current dependency surface

- Filetug currently pins `github.com/rivo/tview v0.42.0` and
  `github.com/gdamore/tcell/v2 v2.13.8`.
- Filetug also consumes
  `github.com/strongo/strongo-tui v0.0.0-20260215000528-71bd9150836c`
  for themes, colors, and a button. That package exposes `tview` and `tcell`
  types, so Filetug cannot complete a clean cutover while using those APIs.
- Filetug already uses `github.com/charmbracelet/glamour v0.10.0` for Markdown.
  It should move to Glamour v2 so the rendering stack shares Lip Gloss v2 rather
  than retaining the older Lip Gloss module path.
- The installed Go 1.27.0 toolchain is compatible with the current Charm
  packages.

### Component mapping

| Current surface | Charm target | Migration note |
|---|---|---|
| `tview.Application`, callbacks, `QueueUpdateDraw` | Bubble Tea v2 root `Model`, `Update`, `View`, and `Cmd` | Rewrite around messages; background work returns typed results carrying request/path identity. |
| `tview.Flex`, `Grid`, `Pages`, custom boxes | Lip Gloss v2 joins, placement, borders, and optional layers | Centralize width/height allocation; do not scatter layout arithmetic through panels. |
| `tview.TreeView` | Bubbles v2 tree plus Filetug adapter | Reuse navigation/open/close/viewport behavior; add lazy children, incremental search, Git decoration, focus transfer, and mouse behavior. |
| `tview.Table` plus lazy `TableContent` | Filetug virtualized file-list model, optionally reusing Bubbles v2 table/list rendering ideas | Do not eagerly call `Info()` for every entry or materialize every styled cell. Benchmark 100, 1,000, and 10,000 entries. |
| `tview.TextView` | Bubbles v2 viewport | Use for text, Markdown, help, errors, and details with bounded content. |
| `tview.InputField`, `Form`, `Button` | Bubbles v2 text input and key bindings | The create panel is small enough to compose directly; avoid a second form framework unless its value is measured. |
| `tview` dynamic color/region tags | Lip Gloss v2 styles and ANSI-aware width helpers | Replace region strings with explicit model state and mouse hit regions. |
| Custom Chroma-to-`tview` tags | Chroma ANSI formatter rendered through Bubble Tea v2 | Delete `pkg/chroma2tcell` after parity tests. |
| `tcell` theme values in `strongo-tui` | Backend-neutral theme tokens plus a Lip Gloss v2 adapter in `strongo-tui` | Provider-first change; retain its existing `tview` adapter only for unrelated consumers during their own migrations. |
| `tcell.SimulationScreen` and primitive mocks | Direct message/reducer tests, ANSI-stripped render goldens, race tests, and compiled-binary PTY smoke | Test mechanisms and terminal restoration, not only final strings. |

## Recommended Architecture

The root Bubble Tea v2 model owns:

- current store and directory identity;
- focused panel and modal/overlay state;
- tree, file-list, and preview child models;
- window dimensions and panel proportions;
- active request IDs and cancellation boundaries;
- keymap and theme tokens.

I/O occurs only in `tea.Cmd` functions. Each result message includes enough
identity to reject stale work, for example store ID, path, entry ID, and request
sequence. Child models receive only messages relevant to them and return
commands upward. `View` is a pure projection of model state.

Filesystem stores, favorites persistence, Git services, and preview data
extractors remain UI-independent. Presenters convert their plain results to
Lip Gloss v2 output at the edge.

## Approved Delivery Strategy

1. Create or resume one isolated implementation effort from the exact latest
   `origin/main`; never edit the canonical clone.
2. Add the new root and child models as normal Go packages, leaving the current
   executable entry point unchanged while targeted model, render, and journey
   tests are established.
3. Keep all temporary coexistence private to the implementation branch. No
   public CLI flag, alternate release artifact, or dual-backend CI promise is
   created.
4. Complete every parity phase, switch the production entry point once, and
   delete all Filetug `tview`/`tcell` code in the same candidate.
5. Validate the complete candidate, release once, and retain the previous
   Filetug release as the operational rollback artifact.

The rejected alternative is a staged dual-mode rollout. It provides a runtime
fallback but costs an estimated 0.5–1.5 additional agent-days and creates a
temporary product surface and test matrix that must later be retired.

## Internal Three-Panel Milestone

This milestone is not releasable and may not land on `origin/main` without the
remaining full-cutover phases.

### Included

- local filesystem store;
- responsive three-panel shell;
- interactive lazy directory tree;
- virtualized file list with parent navigation and current filters;
- text and Markdown preview;
- loading, empty, and error states;
- keyboard focus, navigation, resize, help, and quit;
- cancellation and stale-result rejection;
- deterministic model/view tests and a compiled-binary PTY journey smoke.

### Excluded from this milestone, required before landing

- FTP and HTTP store journey parity;
- favorites, masks, create, delete, scripts, and operations panels;
- worktree detail and WB metadata panel;
- directory summary, image, JSON-specific, DS_Store, and Git preview parity;
- complete Git-status decoration and actions;
- mouse parity for breadcrumbs, lists, tabs, and footer actions;
- removal of `tview`, `tcell`, and their test helpers.

## High-Level Plan

### Phase 0 — current inventory and failing contract (1–2 days)

1. Resolve the remaining interaction, performance, and `strongo-tui` boundary
   questions below before implementation reaches the affected surface.
2. Inventory every current key, mouse action, panel, previewer, store, loading
   state, and error state in an executable parity matrix.
3. Add the end-to-end three-panel journey harness before the new UI exists and
   prove its Bubble Tea target fails for a precise missing mechanism.
4. Measure the current UI with synthetic directories of 100, 1,000, and 10,000
   entries and record input-to-view latency, allocations, and first useful
   frame.

**Verifies:** the existing `tview` journey passes; the new target fails because
there is no Bubble Tea model, not because the fixture or terminal is broken.

### Phase 1 — three-panel internal milestone (5–8 days)

1. Apply the approved `strongo-tui` boundary. Under the recommended option,
   add backend-neutral theme tokens and a Lip Gloss v2 adapter, release it, and
   consume that exact package from Filetug.
2. Add the Bubble Tea v2 program boundary, root model, window sizing, keymap,
   focus state, layout, and terminal lifecycle.
3. Adapt Bubbles v2 tree to Filetug's lazy `files.Store` reads.
4. Implement a virtualized file-list model that preserves lazy metadata and
   current filtering.
5. Convert directory reads and previews to commands with context cancellation
   and identity-bearing result messages.
6. Add Bubbles v2 viewport-based text and Glamour v2 Markdown previews.
7. Add loading, empty, error, help, resize, and quit behavior.

**Verifies:** a fake store isolation test walks tree → file list → preview;
selecting B before A finishes proves A cannot overwrite B; render tests cover
80×24, 120×40, and narrow-terminal layouts; a PTY test launches
the implementation candidate against `<fixture>`, drives the journey, quits,
and proves terminal restoration.

### Phase 2 — navigation and auxiliary parity (4–6 days)

1. Port breadcrumbs, tabs, filters, favorites, masks, new/create, delete,
   scripts, operations, and persisted navigation state.
2. Preserve focus transitions and every current keybinding unless a separately
   approved product change says otherwise.
3. Add FTP and HTTP store journey coverage using controlled fixtures.

**Verifies:** each panel has an isolation test named for its entry, action, exit,
and restored focus; the whole compiled-binary journey covers switching stores,
returning to navigation, and continuing without restart.

### Phase 3 — previews, Git, worktrees, and mouse parity (4–7 days)

1. Port directory summary, Git status, worktree/WB metadata, image metadata,
   JSON, DS_Store, and syntax-highlighted preview behavior.
2. Replace direct widget mutation with result messages and explicit panel
   state.
3. Implement mouse hit regions through Bubble Tea v2 mouse messages and the
   last rendered layout.
4. Run race, cancellation, Unicode/emoji width, and large-directory performance
   tests.

**Verifies:** delayed and cancelled Git/preview commands cannot mutate a newer
path; mouse tests activate only the rendered target; candidate benchmarks stay
within the approved regression budget relative to Phase 0.

### Phase 4 — production switch and full retirement (2–3 days)

1. Run the entire parity matrix and PTY journey on the exact candidate.
2. Switch the production root from the old application to Bubble Tea v2 once.
3. Remove the old root, old panels, `pkg/tviewmocks`,
   `pkg/chroma2tcell`, all `tview`/`tcell` imports, and obsolete
   `strongo-tui` adapter calls.
4. Remove `github.com/rivo/tview` and `github.com/gdamore/tcell/v2` from
   Filetug's module graph, update documentation, and run the exact pre-push/CI
   mechanisms.
5. Confirm the previous Filetug release can be reinstalled as the operational
   rollback, then ship and perform a human smoke in the founder-visible
   terminal.

**Verifies:** repository and module-graph searches return zero Filetug
`tview`/`tcell` consumers; `go test ./...`, the 100% coverage gate, race
checks, build/release checks, the compiled-binary PTY journey, and human terminal
smoke all pass for the exact landed commit.

## Effort and Confidence

| Deliverable | One experienced Go developer | AI-agent implementation |
|---|---:|---:|
| Static visual three-panel spike | 1–2 days | 2–4 focused hours |
| Tested keyboard-driven internal milestone | 5–8 days | 6–12 focused agent-hours plus terminal QA |
| Full approved hard cutover | 2.5–4.5 weeks | 2.5–5 focused agent-days plus review, CI, release coordination, and human QA |
| Rejected released dual-mode route | 3–5 weeks | 3–6 focused agent-days plus the later retirement pass |

Confidence is medium. The estimates should be recalibrated after Phase 1 using
measured large-directory behavior and the number of parity gaps found by the
journey harness.

## Behavior

- Filetug uses one Bubble Tea v2 state owner and declarative view.
- Every asynchronous operation returns a typed message; it does not mutate a
  component from a goroutine.
- The current path, selected entry, focus, and preview request identity are
  explicit model state.
- Window resizing produces a valid layout or an intentional compact fallback;
  it never produces negative panel sizes.
- Keyboard and mouse commands are defined in one discoverable keymap/action
  catalog and rendered by help/footer views.
- Terminal modes are declared by the Bubble Tea v2 view and are restored on
  normal quit, error, signal, and panic paths.

## Acceptance Criteria

- [ ] Starting `ft <path>` with no further input renders a truthful,
      responsive tree, file list, and preview for that path.
- [ ] The end-to-end journey in this Feature runs against the compiled
      production binary without test-only timing sleeps or direct model
      mutation.
- [ ] Rapid tree and file navigation cannot display a stale directory, Git
      status, worktree report, or preview result.
- [ ] All current Filetug panels, stores, previewers, keyboard actions, mouse
      actions, loading states, empty states, and error states are either ported
      or explicitly removed by an approved product decision recorded here.
- [ ] The landed product has one Bubble Tea v2 entry point and no public UI
      backend selector, alternate legacy binary, or dual-backend release path.
- [ ] Synthetic 100, 1,000, and 10,000-entry directory benchmarks meet the
      approved latency/allocation budget relative to the measured current UI.
- [ ] Normal quit, handled error, signal, and panic paths restore the terminal.
- [ ] Filetug production code, tests, generated mocks, module manifests, and
      documentation contain no `tview` or `tcell` dependency after the
      retirement phase.
- [ ] The exact candidate passes `go test ./...`, the repository's 100%
      coverage gate, race checks, build/release checks, PTY journey, and human
      terminal smoke before the cutover is reported complete.
- [ ] The migration is merged and pushed to `origin/main`, and every related
      branch and worktree is cleaned.

## Open Questions

1. Is keyboard-only interaction acceptable for the 5–8 day internal milestone,
   with mouse parity deferred to Phase 3?
2. Should `strongo-tui` add backend-neutral theme tokens plus parallel
   `tview` and Lip Gloss v2 adapters, or should its whole repository migrate
   in the same campaign? The former is recommended to avoid forcing unrelated
   consumers into this Filetug effort.
3. Are text and Markdown previewers sufficient for the internal milestone, with
   all remaining previewers required before the production switch?
4. What maximum measured regression is acceptable for 10,000-entry input
   latency and allocation behavior? A default proposal is no more than 10%
   slower input-to-view latency and no more than 20% additional allocations
   than the measured current baseline.

## Primary Sources

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — framework,
  message/command architecture, keyboard/mouse input, and declarative views.
- [Bubbles](https://github.com/charmbracelet/bubbles) — interactive tree,
  table, list, viewport, text input, spinner, and help components.
- [Bubbles v2 tree source](https://raw.githubusercontent.com/charmbracelet/bubbles/main/tree/tree.go)
  — selection, open/close, navigation, viewport, sizing, and style APIs.
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — layout, styling,
  borders, tables, lists, trees, and ANSI-aware width handling.
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering on
  the Lip Gloss v2 stack.

---
*This document follows the https://specscore.md/feature-specification*
