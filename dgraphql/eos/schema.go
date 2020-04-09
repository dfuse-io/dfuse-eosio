// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Use `go generate` to pack all *.graphql files under this directory (and sub-directories) into
// a binary format.
//
//go:generate go-bindata -ignore=\.go -ignore=testdata/.* -pkg=eos -o=bindata.go ./...
package eos

import "github.com/dfuse-io/dgraphql"

// String reads the .graphql schema files from the generated _bindata.go file, concatenating the
// files together into one string.
//
// If this method complains about not finding functions AssetNames() or MustAsset(),
// run `go generate` against this package to generate the functions.
//
// We concatenate various versions of the schema that are served based on certain
// contexts (specific HTTP headers, Authentication status, etc).

func CommonSchema() *string {
	return dgraphql.BuildSchemaString("common", collectAssetNames(dgraphql.IsCommonAssetName), MustAsset)
}

func AlphaSchema() *string {
	return dgraphql.BuildSchemaString("alpha", collectAssetNames(dgraphql.IsAlphaAssetName), MustAsset)
}

func collectAssetNames(isIncluded func(name string) bool) []string {
	var assetNames []string
	for _, name := range AssetNames() {
		if isIncluded(name) {
			assetNames = append(assetNames, name)
		}
	}

	return assetNames
}
