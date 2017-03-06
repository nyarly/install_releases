# Install Releases

install_releases is a command line tool
to aid the installation of software
for automated agents.
The original use case was
to get up-to-date versions of a Go project onto TeamCity agents,
but it may be flexible enough for others uses.

## Install

It's pithy to suggest that

```
go get github.com/nyarly/install_releases
```

will do everything you need,
since the intended install target is probably not easily amenable to `go get`.
You'll need to bridge that gap yourself, though.

## Use

Arrange for

```
install_releases <organization/project> <asset_pattern> /usr/local/bin
```

to get run on your server.
`organization/project` is the github name for the project
(i.e. `https://github.com/organization/project`).
The asset pattern is a substring
that will distinguish the name of release assets that matter
for your project and target.
It's treated as a regex, so `.*` would choose everything.
`/usr/local/bin` is a reasonable example of an install path.

## What happens

Install releases will make a query to Github
for the releases associated with your project,
and will look for releases that are named
with a sematic version
(possibly as a package-name suffix, like "package-1.2.3").
It will skip "prerelease" (i.e. `1.2.3-rc1`) versions.
Then it looks for tarballs attached as an asset to the release.
It pulls those down,
unpacks them into a "store" directory
and symlinks executables into the install directory provided
(e.g. `/usr/local/bin`).
These symlinks are named with the version,
so if there's `program` in the `1.2.3` tarball,
it'll be linked as `program-1.2.3`.
Additionally,
the most advanced subversion within a minor version will get an extra link,
the most advanced minor version will get an extra link,
and the most advanced version will get a link.
For example, if `1.2.3` is the most recent release,
then its `program` will get a total of 4 links:
`program-1.2.3`, `program-1.2`, `program-1`, and `program`.
This means that consumers of these executables can optimistically use `program`,
and if changes to the interface break their use,
they can pin to an earlier versionin a cheap and cheerful way.

Finally, each of the unpacked archives get a copy of their release JSON,
which `install_releases` uses to determine if their archive needs to be downloaded in the future,
so a second install_releases call immediately after a first one
should incur only an HTTP round-trip to github and return quickly.
