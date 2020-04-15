# eosq

EOSIO block explorer

## Quick start

This alternate start assumes you have yarn and go installed on your system.

* Copy `config-example.json` to `server/config.json`

* Add a valid `dfuse_io_api_key` in `server/config.json`

* In the first terminal:

      yarn install
      yarn start

* In the second terminal, start the go server:

      go install -v ./server && server

* In your browser, connect to your instance via http://localhost:8001.

## Build as single binary within dfuse for EOSIO

First, you will need to grab https://github.com/GeertJohan/go.rice with `go get github.com/GeertJohan/go.rice/rice`

Then,
* yarn build # will create or update the `build` folder

Then either:
* go generate

or

* goreleaser (from root /)

## File structure

./     small Go server to inject configuration in dfuse-box mode
src    JS code
server another small Go server to run stand-alone eosq server

## JS File structure

* `src/atoms`: UI-only components, those components are not application specific and would eventually move to some `styleguide` library.
* `src/clients`: API-layer objects
* `src/components`: Application specific components
    * `src/components/action-pills`: action pill related components, this folder contains also all the templates. Note that the translations are encapsulated inside the templates.
    * `src/components/app-container`: Main layout, this folder would need to move to `src/layouts` to add clarity.
* `src/helpers`: Misc helper functions
* `src/i18n`: Translation folder for the application except for the Pill Templates.
* `src/models`: Object definitions, some objects are still in other folders, they should be moved here.
* `src/pages`: Page components + their component dependencies for complex pages.
* `src/routes`: Route definitions
* `src/services`: Misc classes/objects. Some of it needs to be refactored/moved
* `src/stores`: `Mobx` or plain stores. Mainly classes that store things
* `src/streams`: Websocket listener registration
* `src/theme`: Theme initialisation

### React APP

#### Main dependencies:

* Material-ui
* Emotion
* Fortawesome PRO
* @dfuse/client
* i18next
* nvd3/d3
* mobx
* react-scripts
* Typescript

