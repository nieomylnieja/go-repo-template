# go-repo-template

Repository [template](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-repository-from-a-template)
for creating new Go projects!

## Bootstrap

Click `Use this template` button and voila!
![2024-10-05_22-53](https://github.com/user-attachments/assets/ae397fc7-5fa5-49df-94c1-314572a223d8)

After you're done, you can run the following command to bootstrap the project:

```shell
./bootstrap/init.bash <github-account-name> <repo-name>
```

If you don't intend to ship a binary with your project,
add the following flag:

```shell
./bootstrap/init.bash --no-binary <github-account-name> <repo-name>
```

If you don't want auto release notes and versioning support,
add the following flag:

```shell
./bootstrap/init.bash --no-versioning <github-account-name> <repo-name>
```

The script always expects two positional arguments in the specified order and
a combination of the supported flags (or none).

### GitHub setup

Aside from setting up required GitHub tokens as secrets (mentioned below),
it's highly recommended enable the following options:

TODO

## Devbox

This project utilizes [devbox](https://github.com/jetify-com/devbox) in order
to provide a consistent and reliable development environment.
You can however, If you choose so, install the required dependencies manually.

## Project structure

The template includes an example of
[recommended Go project layout](https://github.com/golang-standards/project-layout)
which includes `cmd`, `pkg` and `internal` directories.

## Makefile

Makefile provides all the basic utilities for the development workflow.
Feel free to extend it with additional targets as you see fit.
The same Makefile targets are used in CI, this ensures consistent results
for both CI and your local machine.

You can quickly inspect the targets of Makefile by running:

```shell
make help
```

When writing new targets, make sure you document them with double `#` character
and place the comment directly above the target, like so:

```makefile
## Document me!
new-target:
  echo "Hello"
```

## CI

Continuous integration pipelines utilize the same Makefile commands which
you run locally within reproducible `devbox` environment.
This ensures consistent behavior of the executed checks
and makes local debugging easier.

## Testing

You can run all unit tests with `make test`.
We also encourage inspecting test coverage during development, you can verify
if the paths you're interested in are covered with `make test/coverage`.

## Releasing binaries

If you decide to ship binaries with the project,
[Goreleaser](https://goreleaser.com/) will require setting up
`GORELEASER_TOKEN` secret.
Refer to [goreleaser-action](https://github.com/goreleaser/goreleaser-action)
docs for up-to-date permission requirements for the token.

## Release Drafter

If you decide to keep release automation, you will need to setup
`RELEASE_DRAFTER_TOKEN` secret.
Refer to [release-drafter](https://github.com/release-drafter/release-drafter?tab=readme-ov-file#usage)
docs for up-to-date permission requirements for the token.

## Labels

In order for some automations to work, like
[Release Drafter](https://github.com/release-drafter/release-drafter),
we need a predefined set of labels.
If you create a new repository from this template, the labels will be
automatically transferred for you.
However, if you want to use these automations in an existing repository,
you'll need to create these labels.
There's a convenience script for that,
located [here](./bootstrap/add-labels.sh).
Run the following:

```shell
./bootstrap/add-labels.sh <github-account-name> <repo-name>
```

If you wish to update existing labels, add `--force` to the `gh label create`
invocation in the script.

## License

The repository template comes with Mozilla Public License 2.0.
Feel free to change the license to any that suits you.

## Gitsync

The author of this repository also uses it as a staple/root
for other repositories to follow.
This means things like linter configs or CI/CD workflows in these repositories
are supposed to be kept in sync with this repository (with some variations).

This is achieved with a tool called [gitsync](https://github.com/nieomylnieja/gitsync).
Configuration file for the tool is [gitsync.json](./gitsync.json).
The bootstrap script removes it.

In order to see the diff between managed repositories run:

```shell
gitsync -c gitsync.json diff
```

In order to sync the changes for managed repositories run:

```shell
gitsync -c gitsync.json sync
```
