package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"

	"github.com/dfuse-io/bstream"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type DefaultCache struct {
	// eosio.token -> [WAX, EOS]
	TokensInContract map[eos.AccountName][]*pbtokenmeta.Token `json:"tokens_in_contract"`

	// tokencontract-centric: eosio.token -> eoscanadadad -> [23 WAX, 22 EOS]
	Balances map[eos.AccountName]map[eos.AccountName][]*OwnedAsset `json:"balances"`

	AtBlock        *Block `json:"at_block"`
	blocklevelLock sync.RWMutex
	cacheFilePath  string
	EOSStake       map[eos.AccountName]*EOSStake `json:"eos_stake"`
}

type Block struct {
	Id  string `json:"id"`
	Num uint64 `json:"num"`
}
type EOSStake struct {
	TotalNet eos.Int64                          `json:"total_net"`
	TotalCpu eos.Int64                          `json:"total_cpu"`
	Entries  map[eos.AccountName]*EOSStakeEntry `json:"stake_entries"`
}

type EOSStakeEntry struct {
	To   eos.AccountName `json:"to"`
	From eos.AccountName `json:"from"`
	Net  eos.Int64       `json:"net"`
	Cpu  eos.Int64       `json:"cpu"`
}

func NewDefaultCache(cacheFilePath string) *DefaultCache {
	return &DefaultCache{
		EOSStake:         make(map[eos.AccountName]*EOSStake),
		TokensInContract: make(map[eos.AccountName][]*pbtokenmeta.Token),
		Balances:         make(map[eos.AccountName]map[eos.AccountName][]*OwnedAsset),
		cacheFilePath:    cacheFilePath,
	}
}

func NewDefaultCacheWithData(tokensInContract []*pbtokenmeta.Token, tokenBalances []*pbtokenmeta.AccountBalance, stakedEntries []*EOSStakeEntry, startBlock bstream.BlockRef, cacheFilePath string) *DefaultCache {
	c := NewDefaultCache(cacheFilePath)

	mutations := &MutationsBatch{}
	for _, token := range tokensInContract {
		mutations.SetToken(token)
	}

	for _, bal := range tokenBalances {
		mutations.SetBalance(bal)
	}

	for _, stak := range stakedEntries {
		mutations.SetStake(stak)
	}

	c.Apply(mutations, startBlock)
	return c
}

func LoadDefaultCacheFromFile(filename string) (*DefaultCache, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer f.Close()
	c := &DefaultCache{}
	dataDecoder := gob.NewDecoder(f)
	err = dataDecoder.Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("unable decode tokenmeta cache: %w", err)
	}
	c.cacheFilePath = filename
	return c, nil
}

func (c *DefaultCache) SaveToFile() error {
	if c.cacheFilePath == "" {
		return fmt.Errorf("cannot save cache no filepath specified")
	}

	tempfile := fmt.Sprintf("%s.tmp", c.cacheFilePath)
	zlog.Info("trying to save to token cache file", zap.String("filename", c.cacheFilePath), zap.String("temp_filename", tempfile))

	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()
	f, err := os.Create(tempfile)
	if err != nil {
		return err
	}
	dataEncoder := gob.NewEncoder(f)
	err = dataEncoder.Encode(c)
	f.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tempfile, c.cacheFilePath)
	return err
}

func (c *DefaultCache) AtBlockRef() bstream.BlockRef {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()

	return bstream.NewBlockRef(c.AtBlock.Id, c.AtBlock.Num)
}

func (c *DefaultCache) Tokens() (tokens []*pbtokenmeta.Token) {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()
	for _, v := range c.TokensInContract {
		tokens = append(tokens, v...)
	}
	return
}

func (c *DefaultCache) TokenContract(contract eos.AccountName, code eos.SymbolCode) *pbtokenmeta.Token {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()

	if tokens, ok := c.TokensInContract[contract]; ok {
		for _, token := range tokens {
			if token.Symbol == code.String() {
				return token
			}
		}
	}
	return nil

}

func (c *DefaultCache) IsTokenContract(contract eos.AccountName) bool {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()

	if _, ok := c.TokensInContract[contract]; ok {
		return true
	}
	return false
}

func (c *DefaultCache) hasSymbolForContract(contract eos.AccountName, symbol string) bool {
	if _, ok := c.TokensInContract[contract]; !ok {
		return false
	}

	for _, t := range c.TokensInContract[contract] {
		if t.Symbol == symbol {
			return true
		}
	}

	return false
}

func (c *DefaultCache) AccountBalances(account eos.AccountName, opts ...AccountBalanceOption) (ownedAssets []*OwnedAsset) {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()

	for _, v := range c.Balances {
		if accAssets, ok := v[account]; ok {
			for _, ass := range accAssets {
				if ass.Asset.Contract == EOSTokenContract &&
					ass.Asset.Asset.Symbol.MustSymbolCode().String() == "EOS" &&
					hasAccountBalanceOption(opts, EOSIncludeStakedAccOpt) {

					value := c.getStakeForAccount(ass.Owner)

					ownedAssets = append(ownedAssets, &OwnedAsset{
						Owner: ass.Owner,
						Asset: &eos.ExtendedAsset{
							Contract: ass.Asset.Contract,
							Asset:    eos.NewEOSAsset(int64(ass.Asset.Asset.Amount) + value),
						},
					})
					continue
				}
				ownedAssets = append(ownedAssets, ass)
			}

		}
	}
	return ownedAssets
}

func hasAccountBalanceOption(opts []AccountBalanceOption, opt AccountBalanceOption) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}
func hasTokenBalanceOption(opts []TokenBalanceOption, opt TokenBalanceOption) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}

func (c *DefaultCache) TokenBalances(contract eos.AccountName, opts ...TokenBalanceOption) (tokenBalances []*OwnedAsset) {
	c.blocklevelLock.RLock()
	defer c.blocklevelLock.RUnlock()

	if contractAssets, ok := c.Balances[contract]; ok {
		for _, accAssets := range contractAssets {
			for _, ass := range accAssets {
				if ass.Asset.Contract == EOSTokenContract &&
					ass.Asset.Asset.Symbol.MustSymbolCode().String() == "EOS" &&
					hasTokenBalanceOption(opts, EOSIncludeStakedTokOpt) {
					tokenBalances = append(tokenBalances, &OwnedAsset{
						Owner: ass.Owner,
						Asset: &eos.ExtendedAsset{
							Contract: contract,
							Asset:    eos.NewEOSAsset(int64(ass.Asset.Asset.Amount) + c.getStakeForAccount(ass.Owner)),
						},
					})
					continue
				}
				tokenBalances = append(tokenBalances, ass)
			}
		}
	}
	return tokenBalances
}

func (c *DefaultCache) setBalance(ownedAsset *OwnedAsset) error {
	if _, ok := c.TokensInContract[ownedAsset.Asset.Contract]; !ok {
		return fmt.Errorf("token contract %s not found in cache", ownedAsset.Asset.Contract)
	}

	if !c.hasSymbolForContract(ownedAsset.Asset.Contract, ownedAsset.Asset.Asset.Symbol.Symbol) {
		return fmt.Errorf("token symbol %s not found in cache in token contract %s", ownedAsset.Asset.Asset.Symbol.Symbol, ownedAsset.Asset.Contract)
	}

	contractAssetsByOwner, ok := c.Balances[ownedAsset.Asset.Contract]
	if !ok {
		c.Balances[ownedAsset.Asset.Contract] = map[eos.AccountName][]*OwnedAsset{}
		contractAssetsByOwner = c.Balances[ownedAsset.Asset.Contract]
	}

	assetsForOwner, ok := contractAssetsByOwner[ownedAsset.Owner]

	// update owned assets if it exists
	for _, a := range assetsForOwner {
		if a.Asset.Asset.Symbol == ownedAsset.Asset.Asset.Symbol {
			a.Asset = ownedAsset.Asset // set it !
			return nil
		}
	}

	//adding a token to the assets of that user on that contract
	contractAssetsByOwner[ownedAsset.Owner] = append(assetsForOwner, ownedAsset)
	return c.setTokenHolders(1, ownedAsset.Asset.Asset.Symbol.Symbol, ownedAsset.Asset.Contract)
}

func (c *DefaultCache) setStake(
	stake *EOSStakeEntry) error {
	if eosStake, ok := c.EOSStake[stake.From]; ok {
		newCpu := eosStake.TotalCpu
		newNet := eosStake.TotalNet
		if oldEntry, ok := eosStake.Entries[stake.To]; ok {
			newCpu = newCpu - oldEntry.Cpu
			newNet = newNet - oldEntry.Net
		}
		eosStake.TotalCpu = newCpu + stake.Cpu
		eosStake.TotalNet = newNet + stake.Net
		eosStake.Entries[stake.To] = stake
	} else {
		c.EOSStake[stake.From] = &EOSStake{
			TotalNet: stake.Net,
			TotalCpu: stake.Cpu,
			Entries: map[eos.AccountName]*EOSStakeEntry{
				stake.To: stake,
			},
		}
	}
	return nil
}

func (c *DefaultCache) getStakeForAccount(account eos.AccountName) int64 {
	if stakeEntry, ok := c.EOSStake[account]; ok {
		return int64(stakeEntry.TotalNet + stakeEntry.TotalCpu)
	}
	return 0
}

func (c *DefaultCache) removeBalance(ownedAsset *OwnedAsset) error {
	contractAssetsByOwner, ok := c.Balances[ownedAsset.Asset.Contract]
	if !ok {
		return fmt.Errorf("removeBalance: token contract %s not found in cache", ownedAsset.Asset.Contract)
	}
	assetsForOwner, ok := contractAssetsByOwner[ownedAsset.Owner]
	if !ok {
		return fmt.Errorf("removeBalance: owner %s not found in cache for token contract %s", ownedAsset.Owner, ownedAsset.Asset.Contract)
	}
	var updatedAssetArray []*OwnedAsset
	for _, a := range assetsForOwner {
		if a.Asset.Asset.Symbol != ownedAsset.Asset.Asset.Symbol {
			updatedAssetArray = append(updatedAssetArray, a)
		}
	}

	newLength := len(updatedAssetArray)
	if newLength == len(assetsForOwner)-1 {
		if newLength == 0 {
			delete(contractAssetsByOwner, ownedAsset.Owner)
		} else {
			contractAssetsByOwner[ownedAsset.Owner] = updatedAssetArray
		}
		return c.setTokenHolders(-1, ownedAsset.Asset.Asset.Symbol.Symbol, ownedAsset.Asset.Contract)
	}
	return fmt.Errorf("removeBalance: token symbol %s not found in cache for owner %s in token contract %s", ownedAsset.Asset.Asset.Symbol, ownedAsset.Owner, ownedAsset.Asset.Contract)
}

func (c *DefaultCache) setTokenHolders(delta int, tokenSymbol string, tokenContract eos.AccountName) error {
	tokens, ok := c.TokensInContract[tokenContract]
	if !ok {
		return fmt.Errorf("cannot set holders, tokenContract %s does not exist in map", tokenContract)
	}
	var absDelta uint32
	if delta < 0 {
		absDelta = uint32(-delta)
	} else {
		absDelta = uint32(delta)
	}

	for _, t := range tokens {
		if t.Symbol == tokenSymbol {
			if delta < 0 {
				if t.Holders < uint64(absDelta) {
					return fmt.Errorf("cannot decrement holders for token %s/%s by %d, would go below 0", tokenContract, tokenSymbol, -delta)
				}
				t.Holders -= uint64(absDelta)
				return nil
			}
			t.Holders += uint64(absDelta)
			return nil
		}
	}
	return fmt.Errorf("setTokenHolders: token %s/%s not found", tokenContract, tokenSymbol)
}

func (c *DefaultCache) setToken(token *pbtokenmeta.Token) error {
	tokens := c.TokensInContract[eos.AccountName(token.Contract)]
	var previousTokenHolders uint64
	var updatedTokensInContract []*pbtokenmeta.Token
	for _, t := range tokens {
		if t.Symbol == token.Symbol { // match on symbol.Symbol alone
			previousTokenHolders = t.Holders
		} else {
			updatedTokensInContract = append(updatedTokensInContract, t)
		}
	}
	token.Holders = previousTokenHolders
	c.TokensInContract[eos.AccountName(token.Contract)] = append(updatedTokensInContract, token)
	return nil
}

func (c *DefaultCache) setContract(contractName eos.AccountName) error {
	_, found := c.TokensInContract[contractName]
	if found {
		return fmt.Errorf("cannot re-add a known contract: %s", contractName)
	}

	c.TokensInContract[contractName] = []*pbtokenmeta.Token{}
	return nil
}

func (c *DefaultCache) Apply(mutationsBatch *MutationsBatch, processedBlock bstream.BlockRef) (errors []error) {
	c.blocklevelLock.Lock()
	defer c.blocklevelLock.Unlock()

	for _, mut := range mutationsBatch.Mutations() {
		var err error
		switch mut.Type {
		case SetBalanceMutation:
			var a *OwnedAsset
			if bal, ok := mut.Args[0].(*pbtokenmeta.AccountBalance); ok {
				a = ProtoEOSAccountBalanceToOwnedAsset(bal)
			} else {
				a = mut.Args[0].(*OwnedAsset)
			}
			err = c.setBalance(a)
		case RemoveBalanceMutation:
			var a *OwnedAsset
			if bal, ok := mut.Args[0].(*pbtokenmeta.AccountBalance); ok {
				a = ProtoEOSAccountBalanceToOwnedAsset(bal)
			} else {
				a = mut.Args[0].(*OwnedAsset)
			}
			err = c.removeBalance(a)
		case SetTokenMutation:
			err = c.setToken(mut.Args[0].(*pbtokenmeta.Token))
		case SetStakeMutation:
			err = c.setStake(mut.Args[0].(*EOSStakeEntry))
		case SetContractMutation:
			err = c.setContract(mut.Args[0].(eos.AccountName))
		}
		if err != nil {
			errors = append(errors, err)
		}
	}
	c.AtBlock = &Block{
		Id:  processedBlock.ID(),
		Num: processedBlock.Num(),
	}
	return
}
