# FileTug Project Structure

- `pkg/tui`: The Bubble Tea v2 user interface, built on the tuigoff navigation shell (directory tree, file list,
  previews, panels).
- `pkg/filetug/*`: UI-free parts of the application: persisted state, favorites, masks, filters, settings.
- `pkg/gitutils`: Git integration helpers.
- `pkg/files`: File system abstraction and storage implementations (OS, FTP, HTTP).
