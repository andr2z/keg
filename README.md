# 🌳 KEG Commands

*🚧This project is in active preliminary development. Commits are public
to allow collaboration and exploration of different directions.*

[![GoDoc](https://godoc.org/github.com/rwxrob/keg?status.svg)](https://godoc.org/github.com/rwxrob/keg)
[![License](https://img.shields.io/badge/license-Apache2-brightgreen.svg)](LICENSE)

This `keg` [Bonzai](https://github.com/rwxrob/bonzai) branch contains
all KEG related commands, most of which are exported so they can be
composed individually if preferred (for example, `keg.MarkCmd` for
parsing KEG Mark markup language in other converter utilities).

## Install

This command can be installed as a standalone program or composed into a
Bonzai command tree.

Standalone

```
go install github.com/rwxrob/keg/cmd/keg@latest
```

Composed

```go
package z

import (
	Z "github.com/rwxrob/bonzai/z"
	example "github.com/rwxrob/keg"
)

var Cmd = &Z.Cmd{
	Name:     `z`,
	Commands: []*Z.Cmd{help.Cmd, example.Cmd, example.BazCmd},
}
```

## Tab Completion

To activate bash completion just use the `complete -C` option from your
`.bashrc` or command line. There is no messy sourcing required. All the
completion is done by the program itself.

```
complete -C keg keg
```

If you don't have bash or tab completion check use the shortcut
commands instead.

## Embedded Documentation

All documentation (like manual pages) has been embedded into the source
code of the application. See the source or run the program with help to
access it.

## Command Line Usage

```
keg help
keg set
ket get
keg conf
keg map - return YAML config for current local keg ids and directories
keg current - returns current local keg id
keg set current - sets current local keg id
keg nodes update [KEG] - update KEGNODES (and /index) for target KEG
keg update - updates own KEGNODES and follow cache
keg follow add - add an entry to FOLLOWS
keg follow cache - fetch fresh cache of all FOLLOWS
keg avoid add - add an entry to AVOID
keg nodes - print the KEGNODES file
keg copy|cp (KEG|NODE ...) KEGTARGET - copy a KEG into into another
keg move|mv (KEG|NODE ...) KEGTARGET - move a KEG or node into another
```

## Configuration

`map` - map of all local keg ids pointing to their directories (like PATH)

## Variables

`current` - current keg from `map`
