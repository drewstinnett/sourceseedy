# Source Seedy (Source CD)

Quickly navigate and control your source code, using defined paths

Do you have a lot of source code projects on your computer? You may get annoyed
like me that it's sort of slow navigating around, yet still keeping them
organized. This project is meant to make that process more simple and quicker.

This project works by organizing your repositories in to a standard structure:

```
~/$base/$host/$namespace/$project
```

This could expand to:

```
~/src/github.com/drewstinnett/mycoolproject
```

## Configuration

The base directory defaults to `~/src`. Change it with the `SOURCESEEDY_BASE`
environment variable, or per-command with `-b`/`--base`.

## Navigation

You can use the `fzf` subcommand to load all of your projects in to an fzf list, and quickly filter, then cd to your selection. The `init` subcommand prints a shell function, `scd`, that does the `cd` for you. Add this to your shell config:

```bash
# bash, zsh
eval "$(sourceseedy init zsh)"
```

```fish
# fish
sourceseedy init fish | source
```

```powershell
# PowerShell
sourceseedy init powershell | Out-String | Invoke-Expression
```

Then `scd` opens the fzf list, and `scd myproj` starts it filtered to `myproj`.
Backing out of fzf with Esc leaves you where you are. Use `--name` to call the
function something other than `scd`.

## Windows

Windows 10 and up works. Put `git` and [`fzf`](https://github.com/junegunn/fzf)
on your `PATH` (`winget install fzf` or `scoop install fzf`), and add the
PowerShell line above to your profile (`notepad $PROFILE`). The default base
directory is `~/src`, which is `C:\Users\you\src`. Releases for Windows are
`.zip` files.

## Cloning Projects

Clone a remote straight in to the right place:

```
$ sourceseedy clone git@github.com:drewstinnett/sourceseedy.git
Cloning into '/Users/drew/src/github.com/drewstinnett/sourceseedy'...
✓ Cloned   ~/src/github.com/drewstinnett/sourceseedy
```

This ends up in `~/src/github.com/drewstinnett/sourceseedy`. Cloning something
you already have is fine, it just says `Exists`.

## Importing Projects

You can quickly import local or remote git repositories directly in to your structure with:

```
$ sourceseedy import /tmp/local_dir
✓ Moved    /tmp/local_dir → ~/src/github.com/drewstinnett/sourceseedy
```

or 

```
$ sourceseedy import https://github.com/drewstinnett/sourceseedy.git
```

Remote URLs are cloned exactly like `clone`. Repos that can't be placed (no git
remote) or whose place is already taken are skipped with a `Skipped` line and
left where they are. Add `-d` to see what would happen without doing it.

## Output

Commands tell you what happened on stderr, one line per repo. Symbols and color
show up when it's a terminal, and are left out when it isn't or `NO_COLOR` is
set. Use `-v` for the debug logging behind it.

`clone`, `import` and `archive` also print the absolute path of the result to
stdout, but only when stdout isn't a terminal, so you don't see it twice. That
makes this work, including for a repo you already had:

```
cd "$(sourceseedy clone git@github.com:drewstinnett/sourceseedy.git)"
```

## Listing Projects

Probably not super useful interactively, but good for scripts. List them all to stdout with

```
$ sourceseedy list
```

Add `--full-path` for absolute directories, or `--json` for an array of
`{id, host, namespace, name, path}`:

```
$ sourceseedy list --json | jq -r '.[] | select(.host == "github.com") | .path'
```

## Project Status

Which of your projects have work that isn't safe yet?

```
$ sourceseedy status
git.example.com/a/api       main     1 changed, 2 untracked, ahead 1
git.example.com/a/tools     feature  no upstream
git.example.com/a/website   (detached)  detached
```

It runs `git status` in every project, several at once, and only shows the ones
that need attention: uncommitted or untracked files, unpushed commits, a branch
with no upstream or one that's been deleted, a detached HEAD, or a repo with no
commits. Pass `--all` to include clean ones too, or `--json` for scripts.

This doesn't fetch, so "behind" is as of your last fetch.

## Archiving Projects

```
$ sourceseedy archive [project]
```

This will just create a tar.gz of the project you select, and put it in `$base/archive`.
The path of the archive is printed to stdout when it isn't a terminal

I use this when I'm about to do something wonky in git that I'm worried will bust my copy