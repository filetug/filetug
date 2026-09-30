// Package tui is the Bubble Tea v2 user interface of FileTug, built on the
// tuigoff navigation shell.
//
// The layout maps onto the shell as follows: the menu panel holds the
// directory tree, the content panel holds the file list next to the preview,
// the breadcrumbs show the current directory and the actions bar lists the
// function and Alt keys.
//
// Everything follows the Elm architecture. Screens are values with Update and
// View. Input and output happen only in tea.Cmd functions, which return
// messages that carry the navigation sequence number of the request that
// started them; a message whose sequence is no longer current is dropped, so
// a slow read can never overwrite a newer directory or preview.
package tui
