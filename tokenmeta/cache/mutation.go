package cache

import (
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
)

type MutationsBatch struct {
	mutations []*Mutation
}

type MutationType int

const (
	SetBalanceMutation MutationType = iota
	RemoveBalanceMutation
	SetTokenMutation
	SetStakeMutation
)

func (m *MutationsBatch) Mutations() []*Mutation {
	return m.mutations
}

func (m *MutationsBatch) SetStake(stake *EOSStakeEntry) {
	mut := &Mutation{
		Type: SetStakeMutation,
		Args: []interface{}{
			stake,
		},
	}
	m.mutations = append(m.mutations, mut)
}

func (m *MutationsBatch) SetBalance(bal *pbtokenmeta.AccountBalance) {
	mut := &Mutation{
		Type: SetBalanceMutation,
		Args: []interface{}{
			bal,
		},
	}
	m.mutations = append(m.mutations, mut)
}

// SetToken should be called when you change maximumSupply or Supply. It ignores the `holders` attribute from token param
func (m *MutationsBatch) SetToken(token *pbtokenmeta.Token) {
	mut := &Mutation{
		Type: SetTokenMutation,
		Args: []interface{}{
			token,
		},
	}
	m.mutations = append(m.mutations, mut)
}

// TODO THIS IS NEVER CALLED, so holders do not decrement..
// RemoveBalance removes a 0-value and decrements holders
func (m *MutationsBatch) RemoveBalance(bal *pbtokenmeta.AccountBalance) {
	mut := &Mutation{
		Type: RemoveBalanceMutation,
		Args: []interface{}{
			bal,
		},
	}
	m.mutations = append(m.mutations, mut)
}

type Mutation struct {
	Type MutationType
	Args []interface{}
}
