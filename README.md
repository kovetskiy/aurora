# aurora

Aurora watches for changes in Arch Linux User Repositories, builds packages,
so you can download it without compiling stuff on your own machines just by
adding a repository to /etc/pacman.conf, for example:

```
[aurora]
Server = https://aurora.reconquest.io/$repo
```

# How does it work

*aurorad* is a daemon that runs on a host and creates Docker
containers with very simple script that clones package from AUR, runs
makepkg and publshes to arch repository.

For adding/removing/listing or even watching build processes there is a client —
**aurora** which communicates with RCP service provided by aurorad.

# Motivation

Well, you know that case when you have installed a `program-git` package and
then you forget about and you never receive any updates. Aurora solves it
because it will literally re-build that package and upload to repo if it's
changed. I guess, you also experienced a sort of frustration when you've
compiled a big AUR package that took like 20 minutes and then you need to do it
again on another machine. Aurora solves that problem too.

# Security

None. No warranties. It's the same as building stuff on your own machine. Maybe
it's a bit safer because at least nobody can `rm -rf` your $HOME directory while
building a package. But it doesn't save you from `rm -rf` in install scripts or
any malicious activity that a program in a package still can do.

# Daemon Deployment

Currently there is no cool way to deploy it except:
`make release@aurorad HOST=yourhost` which builds Go code and runs `scp` and
then starts systemd services. There should be a package.

During the first run you will have to run `aurorad --generate-config` to
generate the default config file.

There are two systemd services — aurora (package builder/processor) and
aurora-web (serves packages as http server).

# Client Installation

You can get it with Go:
```
go get github.com/kovetskiy/aurora/cmd/aurora
```

## Client Usage

```
Usage:
  aurora [options] get [<package>]
  aurora [options] add <package>
  aurora [options] rm <package>
  aurora [options] log <package>
  aurora [options] watch <package> [-w]
  aurora [options] whoami
  aurora -h | --help
  aurora --version

Options:
  get                            Query specified package or query a list of packages.
  add                            Add a package to the queue.
  remove                         Remove a package from the queue.
  log                            Retrieve logs of a package.
  watch                          Watch build process.
  whoami                         Retrieves information about current using in the aurora.
  -a --address <rpc>             Address of aurorad rpc server. [default: https://aurora.reconquest.io/rpc/]
  -k --key <path>                Path to private RSA key. [default: /home/operator/.config/aurora/id_rsa]
  --i-use-insecure-address       By default, aurora doesn't allow to use http:// schema in address.
                                  Use this flag to override this behavior.
  -w --wait                      Wait for a resulting status.
  -h --help                      Show this screen.
  --version                      Show version.
```

# Workflow

I use this beautiful (_no_) script to add package to the queue, wait for its
build and install on local machine:

```
#!/bin/bash

set -euo pipefail

package="${1}"

aurora rm "$package" || echo "Package was not in queue"
aurora add "$package"
aurora watch -w "$package"

aurora get "$package" | tee /dev/stderr | grep -q success

sudo pacman -Sy "aurora/${package}"
```


# State of the project

The project started in 2016 and I've been using it daily for 4 years now. It's
pretty much stable and doesn't ask for any maintenance. All contributions
including documentation are welcome.
