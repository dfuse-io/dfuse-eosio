# Release process

* Test your stuff. Close the milestone.

* Modify the CHANGELOG.md to reflect the tag you're about to build and push

* Tag it (`git tag v0.1.2`)

* Run [GoReleaser](https://goreleaser.com/quick-start/):

    goreleaser release --rm-dist
