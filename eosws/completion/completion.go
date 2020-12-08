// Copyright 2020 dfuse Platform Inc.
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

package completion

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"

	"github.com/arpitbbhayani/tripod"
	"go.uber.org/zap"
)

const maxAccountNameLength = 13

func init() {
	initSQEIndexedFieldTypeByName()
}

type Completion interface {
	Complete(prefix string, limit int) ([]*mdl.SuggestionSection, error)

	AddAccount(account string)

	searchAccountNamesByPrefix(prefix string, limit int) []string
}

func New(ctx context.Context, db eosws.DB) (Completion, error) {
	zlog.Debug("fetching initial account names")
	accountNames, err := db.ListAccountNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("new completion: %s", err)
	}

	return newFromData(accountNames), nil
}

func newFromData(accountNames []string) Completion {
	completion := &defaultCompletion{
		writeMutex: &sync.Mutex{},
	}
	completion.initAccountNames(accountNames)
	completion.initQueryLanguageFields()

	zlog.Debug("finished initializing completion")
	return completion
}

type defaultCompletion struct {
	accountNamesTrie     *tripod.PrefixStoreByteTrie
	sqeIndexedFieldsTrie *tripod.PrefixStoreByteTrie

	writeMutex *sync.Mutex
}

// FIXME change "metrics" logging statements to proper trace's span instead so we can collect metrics
func (completion *defaultCompletion) initAccountNames(accountNames []string) {
	accountNameCount := len(accountNames)
	zlog.Info("initializing completion account names", zap.Int("count", accountNameCount))

	// Needed as the underlying trie returns results in insertion order. By sorting the
	// input array, insertion order will the sorted order of the input array
	zlog.Debug("sorting initial account names")
	sort.Strings(accountNames)

	zlog.Debug("adding all account names to completion trie")
	trie := tripod.CreatePrefixStoreByteTrie(maxAccountNameLength)
	for _, accountName := range accountNames {
		trie.Put([]byte(accountName))
	}

	completion.accountNamesTrie = trie
}

func (completion *defaultCompletion) initQueryLanguageFields() {
	zlog.Info("initializing sqe language fields", zap.Int("count", len(sqeIndexedFields)))

	maxFieldLength := 0
	for _, indexedField := range sqeIndexedFields {
		if len(indexedField.name) > maxFieldLength {
			maxFieldLength = len(indexedField.name)
		}
	}

	zlog.Debug("adding all sqe indexed fields names to completion trie")
	trie := tripod.CreatePrefixStoreByteTrie(maxFieldLength)
	for _, indexedField := range sqeIndexedFields {
		trie.Put([]byte(indexedField.name))
	}

	completion.sqeIndexedFieldsTrie = trie
}

func (completion *defaultCompletion) AddAccount(account string) {
	completion.writeMutex.Lock()
	defer completion.writeMutex.Unlock()

	zlog.Debug("adding new account to completion", zap.String("name", account))
	completion.accountNamesTrie.Put([]byte(account))
}

func (completion *defaultCompletion) Complete(prefix string, limit int) ([]*mdl.SuggestionSection, error) {
	var suggestionSections []*mdl.SuggestionSection

	accountSuggestions := completion.completeAccountNames(prefix, limit)
	if len(accountSuggestions) > 0 {
		suggestionSections = append(suggestionSections, &mdl.SuggestionSection{
			ID:          "accounts",
			Suggestions: accountSuggestions,
		})
	}

	sqeSuggestions := completion.completeSQE(prefix, limit)
	if len(sqeSuggestions) > 0 {
		suggestionSections = append(suggestionSections, &mdl.SuggestionSection{
			ID:          "query",
			Suggestions: sqeSuggestions,
		})
	}

	return suggestionSections, nil
}

func (completion *defaultCompletion) completeAccountNames(prefix string, limit int) []*mdl.Suggestion {
	if !isAccountLike(prefix) {
		return nil
	}

	accountNames := completion.searchAccountNamesByPrefix(prefix, limit)
	accountNameCount := len(accountNames)
	if accountNameCount <= 0 {
		return nil
	}

	accountNameSuggestions := make([]*mdl.Suggestion, accountNameCount)
	for i, accountName := range accountNames {
		accountNameSuggestions[i] = &mdl.Suggestion{Key: string(accountName), Label: string(accountName)}
	}

	return accountNameSuggestions
}

func (completion *defaultCompletion) completeSQE(prefix string, limit int) []*mdl.Suggestion {
	if isAccountLike(prefix) {
		return suggestSQEDefaultSuggestions(prefix)
	}

	usedFieldNames := extractSQEFieldNames(prefix)

	if strings.HasSuffix(prefix, "(") {
		return suggestNextSQEField(prefix, limit, usedFieldNames)
	}

	if strings.HasSuffix(prefix, ")") || strings.HasSuffix(strings.ToLower(prefix), " or") {
		return suggestNextSQEField(prefix+" ", limit, usedFieldNames)
	}

	// Not account like and not previous fields present, try to complete field if possible
	if len(usedFieldNames) <= 0 {
		return completion.completeSQEFieldName(prefix, prefix, limit)
	}

	// We had at least one field (complete or not) present
	if prefix == "" || strings.HasSuffix(prefix, " ") {
		return suggestNextSQEField(prefix, limit, usedFieldNames)
	}

	currentField := findClosestUncompletedSQEField(prefix)
	if currentField == "" {
		return suggestSQEPrefixOnly(prefix)
	}

	if shouldCompleteSQEFieldValue(currentField) {
		return completion.completeSQEFieldValue(currentField, prefix, limit)
	}

	// If we are not completing the field value, we are then completing the name
	suggestions := completion.completeSQEFieldName(currentField, prefix, limit)
	if len(suggestions) > 0 {
		return suggestions
	}

	// If we were not able to complete the field name and no previous
	// completed field were present, user is probably screwing it up, propose
	// a bunch of standard proposals in this case
	if len(usedFieldNames) == 0 {
		return suggestNextSQEField("", limit, emptyFields)
	}

	return suggestSQEPrefixOnly(prefix)
}

func (completion *defaultCompletion) completeSQEFieldName(field string, prefix string, limit int) []*mdl.Suggestion {
	fields := completion.searchSQEFieldsByPrefix(field, limit)
	fieldCount := len(fields)
	if fieldCount <= 0 {
		return nil
	}

	prefixWithoutField := strings.TrimSuffix(prefix, field)

	// A single match on the field name should automatically auto-complete the value
	if fieldCount == 1 {
		return completion.completeSQEFieldValue(fields[0]+":", prefixWithoutField+fields[0]+":", limit)
	}

	fieldSuggestions := make([]*mdl.Suggestion, fieldCount)
	for i, field := range fields {
		label := prefixWithoutField + field + ":"
		fieldSuggestions[i] = &mdl.Suggestion{Key: label, Label: label}
	}

	return fieldSuggestions
}

func (completion *defaultCompletion) completeSQEFieldValue(field string, prefix string, limit int) []*mdl.Suggestion {
	colonIndex := strings.Index(field, ":")
	fieldName := field[0:colonIndex]
	value := field[colonIndex+1:]

	valueType, ok := sqeIndexedFieldTypeByName[fieldName]
	if !ok {
		return suggestSQEPrefixOnly(prefix)
	}

	switch valueType {
	case accountType:
		accountNames := completion.searchAccountNamesByPrefix(value, limit)
		accountNameCount := len(accountNames)
		if accountNameCount <= 0 {
			return suggestSQEPrefixOnly(prefix)
		}

		prefixWithoutField := strings.TrimSuffix(prefix, field)

		suggestions := make([]*mdl.Suggestion, accountNameCount)
		for i, accountName := range accountNames {
			label := prefixWithoutField + fieldName + ":" + string(accountName)
			suggestions[i] = &mdl.Suggestion{Key: label, Label: label}
		}

		return suggestions
	case booleanType:
		var booleanValues []string
		if value == "" {
			booleanValues = []string{"true", "false"}
		} else if strings.HasPrefix("true", value) {
			booleanValues = []string{"true"}
		} else if strings.HasPrefix("false", value) {
			booleanValues = []string{"false"}
		} else {
			booleanValues = []string{"true", "false"}
		}

		prefixWithoutField := strings.TrimSuffix(prefix, field)

		suggestions := make([]*mdl.Suggestion, len(booleanValues))
		for i, booleanValue := range booleanValues {
			label := prefixWithoutField + fieldName + ":" + booleanValue
			suggestions[i] = &mdl.Suggestion{Key: label, Label: label}
		}

		return suggestions
	default:
		return suggestSQEPrefixOnly(prefix)
	}
}

func suggestNextSQEField(prefix string, limit int, alreadyUsedFields []string) []*mdl.Suggestion {
	var suggestions []*mdl.Suggestion
	for _, sqeIndexedField := range sqeIndexedFields {
		if len(suggestions) >= limit {
			break
		}

		if contains(alreadyUsedFields, sqeIndexedField.name) {
			continue
		}

		label := prefix + sqeIndexedField.name + ":"
		suggestions = append(suggestions, &mdl.Suggestion{Key: label, Label: label})
	}

	return suggestions
}
func suggestSQEDefaultSuggestions(prefix string) []*mdl.Suggestion {
	accountHistory := fmt.Sprintf("(auth:%s OR receiver:%s)", prefix, prefix)
	signedBy := fmt.Sprintf("auth:%s", prefix)
	eosTokenTransfer := fmt.Sprintf("receiver:eosio.token account:eosio.token action:transfer (data.from:%s OR data.to:%s)", prefix, prefix)
	fuzzyTokenSearch := fmt.Sprintf("data.to:%s", prefix)

	return []*mdl.Suggestion{
		&mdl.Suggestion{Key: accountHistory, Label: accountHistory, Summary: "account_history"},
		&mdl.Suggestion{Key: signedBy, Label: signedBy, Summary: "signed_by"},
		&mdl.Suggestion{Key: eosTokenTransfer, Label: eosTokenTransfer, Summary: "eos_token_transfer"},
		&mdl.Suggestion{Key: fuzzyTokenSearch, Label: fuzzyTokenSearch, Summary: "fuzzy_token_search"},
	}
}

func suggestSQEPrefixOnly(prefix string) []*mdl.Suggestion {
	return []*mdl.Suggestion{&mdl.Suggestion{Key: prefix, Label: prefix}}
}

func (completion *defaultCompletion) searchAccountNamesByPrefix(prefix string, limit int) []string {
	results := completion.accountNamesTrie.PrefixSearch([]byte(prefix))

	length := limit
	if results.Len() < limit {
		length = results.Len()
	}

	index := 0
	matchingAccountNames := make([]string, length)
	for e := results.Front(); e != nil; e = e.Next() {
		matchingAccountNames[index] = string(e.Value.([]byte))
		index++

		if index >= limit {
			break
		}
	}

	return matchingAccountNames
}

func (completion *defaultCompletion) searchSQEFieldsByPrefix(prefix string, limit int) []string {
	results := completion.sqeIndexedFieldsTrie.PrefixSearch([]byte(prefix))

	length := limit
	if results.Len() < limit {
		length = results.Len()
	}

	index := 0
	matchingFields := make([]string, length)
	for e := results.Front(); e != nil; e = e.Next() {
		matchingFields[index] = string(e.Value.([]byte))
		index++

		if index >= limit {
			break
		}
	}

	return matchingFields
}

var sqeFieldRegexp = regexp.MustCompile("([a-z\\._]+):(.+)?")

func extractSQEFieldNames(prefix string) []string {
	parts := strings.Split(prefix, " ")

	var fieldNames []string
	for _, part := range parts {
		match := sqeFieldRegexp.FindAllStringSubmatch(part, -1)
		if len(match) <= 0 {
			continue
		}

		fieldNames = append(fieldNames, match[0][1])
	}

	return fieldNames
}

func findClosestUncompletedSQEField(prefix string) string {
	parts := strings.Split(prefix, " ")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.Trim(parts[i], " ") == "" {
			continue
		}

		return parts[i]
	}

	return ""
}

func shouldCompleteSQEFieldValue(field string) bool {
	return strings.Contains(field, ":")
}

type valueType int32

const (
	accountType valueType = iota
	actionType
	actionIndexType
	assetType
	booleanType
	blockNumType
	hexType
	freeFormType
	nameType
	permissionType
	transactionIDType
)

type indexedField struct {
	name      string
	valueType valueType
}

// Keep them in order of importance for now until some scoring is implemented
var sqeIndexedFields = []indexedField{
	{"account", accountType},
	{"receiver", accountType},
	{"action", actionType},
	{"data.to", accountType},
	{"data.from", accountType},
	{"auth", permissionType},
	{"block_num", blockNumType},
	{"trx_idx", transactionIDType},

	{"data.account", accountType},
	{"data.active", freeFormType},
	{"data.active_key", freeFormType},
	{"data.actor", freeFormType},
	{"data.amount", assetType},
	{"data.authority", freeFormType},
	{"data.bid", freeFormType},
	{"data.bidder", accountType},
	{"data.canceler", accountType},
	{"data.creator", accountType},
	{"data.executer", accountType},
	{"data.is_active", booleanType},
	{"data.is_priv", booleanType},
	{"data.isproxy", booleanType},
	{"data.issuer", accountType},
	{"data.level", freeFormType},
	{"data.location", freeFormType},
	{"data.maximum_supply", assetType},
	{"data.name", nameType},
	{"data.newname", nameType},
	{"data.owner", accountType},
	{"data.parent", accountType},
	{"data.payer", accountType},
	{"data.permission", permissionType},
	{"data.producer", accountType},
	{"data.producer_key", freeFormType},
	{"data.proposal_name", nameType},
	{"data.proposal_hash", freeFormType},
	{"data.proposer", accountType},
	{"data.proxy", freeFormType},
	{"data.public_key", freeFormType},
	{"data.producers", freeFormType},
	{"data.quant", freeFormType},
	{"data.quantity", freeFormType},
	{"data.ram_payer", accountType},
	{"data.receiver", accountType},
	{"data.requirement", freeFormType},
	{"data.symbol", freeFormType},
	{"data.threshold", freeFormType},

	{"data.transfer", freeFormType},
	{"data.voter", accountType},
	{"data.voter_name", nameType},
	{"data.weight", freeFormType},

	{"act_digest", hexType},
	{"act_idx", actionIndexType},
	{"scheduled", booleanType},
}

var sqeIndexedFieldTypeByName map[string]valueType

func initSQEIndexedFieldTypeByName() {
	sqeIndexedFieldTypeByName = map[string]valueType{}
	for _, sqeIndexedField := range sqeIndexedFields {
		sqeIndexedFieldTypeByName[sqeIndexedField.name] = sqeIndexedField.valueType
	}
}

var accountNameRegex = regexp.MustCompile("^[a-z0-5\\.]{1,13}$")
var blockNumLikeRegex = regexp.MustCompile("^[0-9]+$")
var hexLikeRegex = regexp.MustCompile("^[0-9a-f]{6,}$")

func isAccountLike(input string) bool {
	return accountNameRegex.Match([]byte(input))
}

func isBlockNumLike(input string) bool {
	return !isAccountLike(input) && blockNumLikeRegex.Match([]byte(input))
}

func isHexdecimalLike(input string) bool {
	return !isAccountLike(input) && hexLikeRegex.Match([]byte(input))
}

var emptyFields []string

func contains(elements []string, candidate string) bool {
	for _, element := range elements {
		if element == candidate {
			return true
		}
	}
	return false
}
