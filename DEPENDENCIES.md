# Dependencies

dfuse for EOSIO requires our instrumented `nodeos` binary, a fork
of the standard `EOSIO` software.

You will also want to build both the `dashboard` and the `eosq` apps

## Building web apps

* **NOTE: Building web apps is done as part of the ./build.sh script**

Building `eosq`:

```
cd eosq
yarn install
yarn build
```

and building the `dashboard` from dlauncher, copying the ricebox to this repo:

```
pushd ..
  git clone git@github.com/dfuse-io/dlauncher
  pushd dashboard
    pushd client
      yarn install
      yarn build
	popd
    go generate
  popd
popd
go generate ./dashboard # copies the ../dlauncher/dashboard/rice-box.go file to ./dashboard
```

## dfuse Instrumented EOSIO Prebuilt Binaries

dfuse uses a specialized EOSIO binary that contains instrumentation required
to extract the data needed to power all dfuse's features.

The current source code can be found on branch [release/2.0.x-dm](https://github.com/dfuse-io/eos/tree/release/2.0.x-dm)
under [github.com/dfuse-io/eos](https://github.com/dfuse-io/eos) fork of EOSIO software.

**Notes**:

* It is safe to use this forked version as a replacement for your current installation, all
  special instrumentations are gated around a config option (i.e. `deep-mind = true`) that is off by
  default.

* This instrumentation has been merged in the upstream develop branch,
  but is not yet in a release: https://github.com/EOSIO/eos/pull/8788

### Mac OS X:

#### Mac OS X Brew Install

```sh
brew install dfuse-io/tap/eosio
```

#### Mac OS X Brew Uninstall

```sh
brew remove eosio
```

### Ubuntu Linux:

#### Ubuntu 18.04 Package Install

```sh
curl -s https://api.github.com/repos/dfuse-io/eos/releases/latest|grep "browser_download_url.*18.04_amd64.deb"|cut -d : -f 2,3|tr -d \"|wget --show-progress -O ./eosio-dm-latest-18.04.deb -qi -
sudo apt install ./eosio-dm-latest-18.04.deb
```

#### Ubuntu 16.04 Package Install

```sh
curl -s https://api.github.com/repos/dfuse-io/eos/releases/latest|grep "browser_download_url.*16.04_amd64.deb"|cut -d : -f 2,3|tr -d \"|wget --show-progress -O ./eosio-dm-latest-16.04.deb -qi -
sudo apt install ./eosio-dm-latest-16.04.deb
```

#### Ubuntu Package Uninstall

```sh
sudo apt remove eosio
```

### RPM-based (CentOS, Amazon Linux, etc.):

#### RPM Package Install

```sh
wget curl -s https://api.github.com/repos/dfuse-io/eos/releases/latest|grep "browser_download_url.*.rpm"|cut -d : -f 2,3|tr -d \"|wget --show-progress -O ./eosio-dm-latest.rpm -qi -
sudo yum install ./eosio-dm-latest.rpm
```

#### RPM Package Uninstall

```sh
sudo yum remove eosio
```
