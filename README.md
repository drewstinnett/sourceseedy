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

## Navigation

You can use the `fzf` subcommand to load all of your projects in to an fzf list, and quickly filter, then cd to your selection using something like this in your shell:

```bash
scd () {
  target=$(/usr/local/bin/sourceseedy fzf $1)
  cd $target
}
```

## Importing Projects

You can quickly import local or remote git repositories directly in to your structure with:

```
$ sourceseedy import /tmp/local_dir
```

or 

```
$ sourceseedy import https://github.com/drewstinnett/sourceseedy.git
```

## Listing Projects

Probably not super useful, but you can just list them all to stdout with

```
$ sourceseedy list
```

## Archiving Projects

```
$ sourcseedy archive [project]
```

This will just create a tar.gz of the project you select, and put it in `$base/archive`

I use this when I'm about to do somethink wonky in git that I'm worried will bust my copy