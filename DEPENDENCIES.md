# Dependencies

dfuse for EOSIO requires our instrumented `nodeos` binary, a fork
of the standard `EOSIO` software.

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
brew tap dfuse-io/tap/eosio
```

#### Mac OS X Brew Uninstall

```sh
brew remove eosio
```

### Ubuntu Linux:

#### Ubuntu 18.04 Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm_ubuntu-18.04_amd64.deb
sudo apt install ./eosio_2.0.3-dm_ubuntu-18.04_amd64.deb
```

#### Ubuntu 16.04 Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm_ubuntu-16.04_amd64.deb
sudo apt install ./eosio_2.0.3-dm_ubuntu-16.04_amd64.deb
```

#### Ubuntu Package Uninstall

```sh
sudo apt remove eosio
```

### RPM-based (CentOS, Amazon Linux, etc.):

#### RPM Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm.el7.x86_64.rpm
sudo yum install ./eosio_2.0.3-dm.el7.x86_64.rpm
```

#### RPM Package Uninstall

```sh
sudo yum remove eosio
```
