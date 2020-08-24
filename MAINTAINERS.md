# Changelog management

* Distinguish between `Public API Changes` and `System Administration Changes`
* Put `Added` section first, followed by `Changed`, `Deprecated`, `Removed`, `Fixed` and `Security`.

# Release process

* Rebuild ../dlauncher/dashboard from the and run `go generate ./...`

* Test your stuff (./devel contains a few scenarios that you can run).

* Modify the CHANGELOG.md to reflect the tag you're about to build and push

* Tag it (`git tag v0.1.2`)

* Run [GoReleaser](https://goreleaser.com/quick-start/):

    goreleaser release --rm-dist

* Remove the changelogs in `CHANGELOG.md`, and link to the GitHub
  release.  The final source of truth for the change log is the GitHUb
  `/releases` page for the release.

* Make sure any contributions from the community are RECOGNIZED in the
  Releases page.
