# eosq

EOSIO block explorer

## Quick start

This guide assumes you have `yarn` and `go` installed on your system.

- In the first terminal, build the dfuse single binary:

      go install ../cmd/dfuseeos

  then run the binary with:

      dfuseeos start

- In the second terminal, install dependencies for the `eosq` React app:

      yarn install

  then run the React app in development mode:

      yarn start

- In your browser, connect to your instance via http://localhost:3000.

## Build as single binary within dfuse for EOSIO

- First, you will need to grab https://github.com/GeertJohan/go.rice with `go get github.com/GeertJohan/go.rice/rice`

- Then create the React build with:

      yarn build

- Next, run either:

      go generate

  OR

      goreleaser (from root /)

### Development on existing network

Our deployed APIs for eosq is currently restricted to special API keys, so currently, you
need to obtain a special API key and those are currently restricted to dfuse employes, so
does the instruction given here.

You can easily connect to EOS Mainnet by simply exporting some environment variables and
launching `yarn start` straight. Here the minimum one you need to launch eosq for
EOS Mainnet:

```
export REACT_APP_EOSQ_CURRENT_NETWORK="eos-mainnet"
export REACT_APP_DFUSE_API_KEY="<API Key>"
export REACT_APP_DFUSE_API_NETWORK="mainnet.eos.dfuse.io"
```

Launching `yarn start` with those exported will correctly launch `eosq` and make it
point to EOS Mainnet.

Here a full list and what they control:

- `REACT_APP_DFUSE_AUTH_URL` - The dfuse API Authentication URl to pass to `@dfuse/client`.
- `REACT_APP_DFUSE_API_KEY` - The dfuse API key to pass to `@dfuse/client`.
- `REACT_APP_DFUSE_API_NETWORK` - The dfuse API network to pass to `@dfuse/client`.
- `REACT_APP_EOSQ_CURRENT_NETWORK` - The actual current network to select in the list of available networks.
- `REACT_APP_EOSQ_DISPLAY_PRICE` - Wether to display the price info or not.
- `REACT_APP_EOSQ_ON_DEMAND` - Wether this network is an on-demand network or not.
- `REACT_APP_EOSQ_AVAILABLE_NETWORK` - A valid JSON string representing the valid config of available networks to display in the main menu.

**Note** Those are valid for development purposes only, they are not picked on production usage and they are injected in the
HTML `index.html` page directly on production.

## File structure

- `src` JS code
- `app` Go code which wraps the React build into a `go` module to be run by `dfuseeos`

## JS File structure

- `src/atoms`: UI-only components, those components are not application specific and would eventually move to some `styleguide` library.
- `src/clients`: API-layer objects
- `src/components`: Application specific components
  - `src/components/action-pills`: action pill related components, this folder contains also all the templates. Note that the translations are encapsulated inside the templates.
  - `src/components/app-container`: Main layout, this folder would need to move to `src/layouts` to add clarity.
- `src/helpers`: Misc helper functions
- `src/i18n`: Translation folder for the application except for the Pill Templates.
- `src/models`: Object definitions, some objects are still in other folders, they should be moved here.
- `src/pages`: Page components + their component dependencies for complex pages.
- `src/routes`: Route definitions
- `src/services`: Misc classes/objects. Some of it needs to be refactored/moved
- `src/stores`: `Mobx` or plain stores. Mainly classes that store things
- `src/streams`: Websocket listener registration
- `src/theme`: Theme initialisation

### React APP

#### Main dependencies:

- Material-ui
- antd
- Emotion
- Fortawesome Free
- @dfuse/client
- i18next
- nvd3/d3
- mobx
- react-scripts
- Typescript
