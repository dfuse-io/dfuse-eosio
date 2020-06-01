package tools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/go-getter"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var smCreateProject = &cobra.Command{Use: "create-smart-contract {project name}", RunE: smCreateProjectE}

func init() {
	Cmd.AddCommand(smCreateProject)
}

func smCreateProjectE(cmd *cobra.Command, args []string) error {

	prompt := promptui.Prompt{
		Label: "Project Name",
	}
	projectName, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		return fmt.Errorf("unable to ask project name: %w", err)
	}

	prompt = promptui.Prompt{
		Label: "Smart Contract Account Name",
	}
	accountName, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		return fmt.Errorf("unable to ask project name: %w", err)
	}

	configMap := map[string]string{}
	configMap["#project_name#"] = projectName
	configMap["#contract_account#"] = accountName

	err = createProjectFiles(projectName, configMap)
	if err != nil {
		return fmt.Errorf("failed to create project file: %w", err)
	}

	fmt.Println("Project created.")

	return nil
}

func createProjectFiles(projectName string, configMap map[string]string) error {

	err := os.Mkdir(projectName, os.ModeDir|(OS_USER_RWX|OS_ALL_R))
	if err != nil {
		return fmt.Errorf("failed to create project dir: %s: %w", projectName, err)
	}
	fmt.Printf("Folder created %s\n", projectName)

	err = toFile(path.Join(projectName, "CMakeLists.txt"), CMakeListstxt, configMap, OS_USER_RW|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing CMakeLists.txt: %w", err)
	}

	err = toFile(path.Join(projectName, "compile.sh"), CompileSh, configMap, OS_USER_RWX|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing compile.sh: %w", err)
	}

	srcDir := path.Join(projectName, "src")
	err = os.Mkdir(srcDir, os.ModeDir|(OS_USER_RWX|OS_ALL_R))
	if err != nil {
		return fmt.Errorf("failed to create project dir: %s: %w", projectName, err)
	}
	fmt.Printf("Folder created %s\n", srcDir)

	err = toFile(path.Join(srcDir, projectName+".cpp"), cpp, configMap, OS_USER_RW|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing %s.cpp: %w", projectName, err)
	}
	err = toFile(path.Join(srcDir, projectName+".hpp"), hpp, configMap, OS_USER_RW|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing %s.hpp: %w", projectName, err)
	}

	//bootDir := path.Join(projectName, "boot")
	//err = os.Mkdir(srcDir, os.ModeDir|(OS_USER_RWX|OS_ALL_R))
	//if err != nil {
	//	return fmt.Errorf("failed to create project dir: %s: %w", projectName, err)
	//}
	//fmt.Printf("Folder created %s\n", srcDir)

	err = toFile(path.Join(projectName, "boot_sequence.yaml"), bootseqYml, configMap, OS_USER_RW|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing boot_sequence.yaml: %w", projectName, err)
	}
	err = toFile(path.Join(projectName, "eosc-vault.json"), eosVaultJson, configMap, OS_USER_RW|OS_ALL_R)
	if err != nil {
		return fmt.Errorf("writing eosc-vault.json: %w", err)
	}

	err = toFile(path.Join(projectName, "boot.sh"), bootSh, configMap, 0777)
	if err != nil {
		return fmt.Errorf("writing boot.sh: %w", err)
	}

	fmt.Println("downloading boot smart contracts")
	err = getter.Get(path.Join(projectName, "contracts"), "github.com/dfuse-io/dfuse-eosio/bootstrapping/contracts")
	if err != nil {
		return fmt.Errorf("failed to get contracts from github: %s: %w", "github.com/dfuse-io/dfuse-eosio/bootstrapping/contracts", err)
	}

	return nil
}

func toFile(filePath string, content string, configMap map[string]string, fileMode os.FileMode) error {
	data := content
	for k, v := range configMap {
		data = strings.ReplaceAll(data, k, v)
	}

	fmt.Printf("Writing '%s'\n", filePath)
	if err := ioutil.WriteFile(filePath, []byte(data), fileMode); err != nil {
		return fmt.Errorf("writing %s: %w", CMakeListstxt, err)
	}

	return nil
}

var CMakeListstxt = `
set(CMAKE_SYSTEM_NAME Generic)
set(CMAKE_C_COMPILER_WORKS 1)
set(CMAKE_CXX_COMPILER_WORKS 1)

find_package(eosio.cdt)

cmake_minimum_required(VERSION 3.5)
project(ice VERSION 1.0.0.0)

add_contract( #project_name# #project_name# src/#project_name#.cpp )

target_include_directories(#project_name#.wasm PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)
`

var CompileSh = `
#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

BROWN='\033[0;33m'
NC='\033[0m'

BUILD_SUFFIX=${1}
CORES=` + "`getconf _NPROCESSORS_ONLN`" + `

printf "${BROWN}Compiling ${BUILD_SUFFIX}${NC}\n"

mkdir -p $ROOT/build${BUILD_SUFFIX}
eosio-cpp ./src/#project_name#.cpp -o ./build/#project_name#.wasm
`

var cpp = `
#include "#project_name#.hpp"
#include <eosio/eosio.hpp>

void #project_name#::hi( name user ) {
 print( "Hello, ", user);
}
`
var hpp = `
#include <eosio/eosio.hpp>

using namespace eosio;

class [[eosio::contract]] #project_name# : public contract {
  public:
      using contract::contract;

      [[eosio::action]]
      void hi( name user );
};
`

var bootseqYml = `
# Copyright 2019 dfuse Platform Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

keys:
  ephemeral: 5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3   # EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV

contents:
  - name: eosio.bios.abi
    url: ./contracts/eosio.bios-1.0.2.abi
  - name: eosio.bios.wasm
    url: ./contracts/eosio.bios-1.0.2.wasm

  - name: eosio.system.abi
    url: ./contracts/eosio.system-1.0.2.abi
  - name: eosio.system.wasm
    url: ./contracts/eosio.system-1.0.2.wasm

  - name: eosio.msig.abi
    url: ./contracts/eosio.msig-1.0.2.abi
  - name: eosio.msig.wasm
    url: ./contracts/eosio.msig-1.0.2.wasm

  - name: eosio.token.abi
    url: ./contracts/eosio.token-1.0.2.abi
  - name: eosio.token.wasm
    url: ./contracts/eosio.token-1.0.2.wasm

################################# BOOT SEQUENCE ###################################
boot_sequence:
- op: system.setcode
  label: Setting eosio.bios code for account eosio
  data:
    account: eosio
    contract_name_ref: eosio.bios
- op: system.newaccount
  label: Create account eosio2
  data:
    creator: eosio
    new_account: eosio2
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio3
  data:
    creator: eosio
    new_account: eosio3
    pubkey: ephemeral

- op: system.newaccount
  label: Create account eosio.msig (on-chain multi-signature helper)
  data:
    creator: eosio
    new_account: eosio.msig
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.token (main multi-currency contract, including EOS)
  data:
    creator: eosio
    new_account: eosio.token
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.ram (where buyram proceeds go)
  data:
    creator: eosio
    new_account: eosio.ram
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.ramfee (where buyram fees go)
  data:
    creator: eosio
    new_account: eosio.ramfee
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.names (where bidname revenues go)
  data:
    creator: eosio
    new_account: eosio.names
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.stake (where delegated stakes go)
  data:
    creator: eosio
    new_account: eosio.stake
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.saving (unallocated inflation)
  data:
    creator: eosio
    new_account: eosio.saving
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.bpay (fund per-block bucket)
  data:
    creator: eosio
    new_account: eosio.bpay
    pubkey: ephemeral
- op: system.newaccount
  label: Create account eosio.vpay (fund per-vote bucket)
  data:
    creator: eosio
    new_account: eosio.vpay
    pubkey: ephemeral
- op: system.setpriv
  label: Setting privileged account for eosio.msig
  data:
    account: eosio.msig
- op: system.setcode
  label: Setting eosio.msig code for account eosio.msig
  data:
    account: eosio.msig
    contract_name_ref: eosio.msig
- op: system.setcode
  label: Setting eosio.token code for account eosio.token
  data:
    account: eosio.token
    contract_name_ref: eosio.token
- op: token.create
  label: Creating the EOS currency symbol
  data:
    account: eosio
    amount: 10000000000.0000 EOS  # Should work with 5% inflation, for the next 50 years (end of uint32 block_num anyway)
- op: token.issue
  label: Issuing initial EOS monetary base
  data:
    account: eosio
    amount: 1000011821.0000 EOS  # 1B coins, as per distribution model + gift of RAM to new users.
    memo: "Creation of EOS. Credits and Acknowledgments: eosacknowledgments.io"

- op: system.setcode
  label: Replacing eosio account from eosio.bios contract to eosio.system
  data:
    account: eosio
    contract_name_ref: eosio.system


- op: system.resign_accounts
  label: Disabling authorization for system accounts, pointing ` + "`eosio+`" + ` to the ` + "`eosio.prods`" + ` account.
  data:
    accounts:
    #- eosio
    - eosio.msig
    - eosio.token
    - eosio.ram
    - eosio.ramfee
    - eosio.stake
    - eosio.names
    - eosio.saving
    - eosio.bpay
    - eosio.vpay

- op: system.newaccount
  label: Create account #contract_account#
  data:
    creator: eosio
    new_account: #contract_account#
    pubkey: ephemeral
    ram_eos_quantity: 1000000

- op: system.delegate_bw
  label: delegatebw from eosio to #contract_account#
  data:
    from: eosio
    to: #contract_account#
    stake_cpu: 1000000
    stake_net: 1000000
    Transfer: true

- op: token.transfer
  label: eosio transfering token to #contract_account#
  data:
    from: eosio
    to: #contract_account#
    quantity: 100000.0000 EOS
`

var eosVaultJson = `
{
  "kind": "eosc-vault-wallet",
  "version": 1,
  "comment": "",
  "secretbox_wrap": "passphrase",
  "secretbox_ciphertext": "SP1c+Tj8VWKd/PI/b05KvnG4DzpcHxHNl0cSrB4kF6kopheXO/YHjhbWzhXQVNU3l4dejIjmb8Jyt1CuX7ze2jLAl6wXWsM0RPpmk0ycl0okqpRi3KGgStU8OIy5L1AoseoF8vnqVguKKYgVfLTKauLOXPJebaFI"
}
`
var bootSh = `#!/bin/bash -xe

## I'm so sorry...
`

const (
	OS_READ        = 04
	OS_WRITE       = 02
	OS_EX          = 01
	OS_USER_SHIFT  = 6
	OS_GROUP_SHIFT = 3
	OS_OTH_SHIFT   = 0

	OS_USER_R   = OS_READ << OS_USER_SHIFT
	OS_USER_W   = OS_WRITE << OS_USER_SHIFT
	OS_USER_X   = OS_EX << OS_USER_SHIFT
	OS_USER_RW  = OS_USER_R | OS_USER_W
	OS_USER_RWX = OS_USER_RW | OS_USER_X

	OS_GROUP_R   = OS_READ << OS_GROUP_SHIFT
	OS_GROUP_W   = OS_WRITE << OS_GROUP_SHIFT
	OS_GROUP_X   = OS_EX << OS_GROUP_SHIFT
	OS_GROUP_RW  = OS_GROUP_R | OS_GROUP_W
	OS_GROUP_RWX = OS_GROUP_RW | OS_GROUP_X

	OS_OTH_R   = OS_READ << OS_OTH_SHIFT
	OS_OTH_W   = OS_WRITE << OS_OTH_SHIFT
	OS_OTH_X   = OS_EX << OS_OTH_SHIFT
	OS_OTH_RW  = OS_OTH_R | OS_OTH_W
	OS_OTH_RWX = OS_OTH_RW | OS_OTH_X

	OS_ALL_R   = OS_USER_R | OS_GROUP_R | OS_OTH_R
	OS_ALL_W   = OS_USER_W | OS_GROUP_W | OS_OTH_W
	OS_ALL_X   = OS_USER_X | OS_GROUP_X | OS_OTH_X
	OS_ALL_RW  = OS_ALL_R | OS_ALL_W
	OS_ALL_RWX = OS_ALL_RW | OS_GROUP_X
)
