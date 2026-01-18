# Killer features of FileTug

FileTug is designed to provide a more intuitive and user-friendly experience for navigating and managing files in the
terminal. Here are some of its key features that set it apart from traditional file managers like `mc`, `ranger`, and
others:

## Implemented

- Quick navigation to your favorite directories and files
- Directory summary
    - summary by extension
    - summary by file type
- Instant smart file preview. Parses known file formats and provides a structured preview.
- Integrated git support:
    - dir status
    - file status – _to be implemented_
    - stage/unstage files & dirs – _to be implemented_

## Roadmap – to be implemented

- Caching of network resources
- pre-fetching of data
- Quick bulk-select using sets of matching patterns.
    - Example:
        - Coding files: *.(cpp|cs|js|ts)
        - Data files: *.(csv|dbf|json|xml|yaml)
- Tagging files & directories.
- Curated lists of files and directories with predefined filters.
- Bookmarks – quick jumps between multiple locations.
- History of operations – see what you've been up to.
- Bulk renaming:
    - support for regular expressions
    - dry run
    - rollback rename
    - history
- Logs viewers
    - support for common log formats
    - preview logs
    - tail-watching