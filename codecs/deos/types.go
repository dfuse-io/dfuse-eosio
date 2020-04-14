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

package deos

import (
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/eoscanada/eos-go"
)

//
// CreationFlatTree represents the creation order tree
// in a flatten manners. The flat list is built by doing
// a deep-first walk of the creational tree, outputting
// at each traversal the `CreationNode` triplet
// `(index, creatorParentIndex, executionIndex)` where a parent of
// `-1` represents a root node.
//
// For example, assuming a `CreationFlatTree` of the form:
//
// [
//   [0, -1, 0],
//   [1, 0, 1],
//   [2, 0, 2],
//   [3, 2, 3],
// ]
//
// Represents the following creational tree:
//
// ```
//   0
//   ├── 1
//   └── 2
//       └── 3
// ```
//
// The tree can be reconstructed using the following quick Python.
//
type CreationFlatTree = []CreationFlatNode

// CreationFlatNode represents a flat node in a flat tree.
// It's a triplet slice where elements reprensents the following
// values, assuming `(<depthFirstWalkIndex>, <parentDepthFirstWalkIndex>, <executionActionIndex>)`:
//
// The first value of the node is it's id, derived by doing a depth-first walk
// of the creation tree and incrementing an index at each node visited.
//
// The second value is the parent index of the current node, the index is the
// index of the initial element of the `CreationFlatNode` slice.
//
// The third value is the execution action index to get the actual execution traces
// from the actual execution tree (deep-first walking index in the execution
// tree).
//
type CreationFlatNode = [3]int

type Specification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type SubjectiveRestrictions struct {
	Enable                        bool         `json:"enable"`
	PreactivationRequired         bool         `json:"preactivation_required"`
	EarliestAllowedActivationTime eos.JSONTime `json:"earliest_allowed_activation_time"`
}

//
/// Permission OP
//

type PermOp struct {
	Operation   string            `json:"op"`
	ActionIndex int               `json:"action_idx"`
	OldPerm     *permissionObject `json:"old,omitempty"`
	NewPerm     *permissionObject `json:"new,omitempty"`
}

// permissionObject represent the `nodeos` permission object that is stored
// in chainbase. Used to deserialize deep mind JSON to a correct `PermOp`.
type permissionObject struct {
	Owner       eos.AccountName `json:"owner"`
	Name        string          `json:"name"`
	LastUpdated eos.JSONTime    `json:"last_updated"`
	Auth        *eos.Authority  `json:"auth"`
}

func (p *permissionObject) ToProto() *pbdeos.PermissionObject {
	return &pbdeos.PermissionObject{
		Owner:       string(p.Owner),
		Name:        p.Name,
		LastUpdated: mustProtoTimestamp(p.LastUpdated.Time),
		Authority:   AuthoritiesToDEOS(p.Auth),
	}
}

type rlimitState struct {
	ID                   uint32            `json:"id"`
	AverageBlockNetUsage *usageAccumulator `json:"average_block_net_usage"`
	AverageBlockCpuUsage *usageAccumulator `json:"average_block_cpu_usage"`
	PendingNetUsage      eos.Uint64        `json:"pending_net_usage"`
	PendingCpuUsage      eos.Uint64        `json:"pending_cpu_usage"`
	TotalNetWeight       eos.Uint64        `json:"total_net_weight"`
	TotalCpuWeight       eos.Uint64        `json:"total_cpu_weight"`
	TotalRamBytes        eos.Uint64        `json:"total_ram_bytes"`
	VirtualNetLimit      eos.Uint64        `json:"virtual_net_limit"`
	VirtualCpuLimit      eos.Uint64        `json:"virtual_cpu_limit"`
}

func (s *rlimitState) ToProto() *pbdeos.RlimitOp_State {
	return &pbdeos.RlimitOp_State{
		State: &pbdeos.RlimitState{
			AverageBlockNetUsage: s.AverageBlockNetUsage.ToProto(),
			AverageBlockCpuUsage: s.AverageBlockCpuUsage.ToProto(),
			PendingNetUsage:      uint64(s.PendingNetUsage),
			PendingCpuUsage:      uint64(s.PendingCpuUsage),
			TotalNetWeight:       uint64(s.TotalNetWeight),
			TotalCpuWeight:       uint64(s.TotalCpuWeight),
			TotalRamBytes:        uint64(s.TotalRamBytes),
			VirtualNetLimit:      uint64(s.VirtualNetLimit),
			VirtualCpuLimit:      uint64(s.VirtualCpuLimit),
		},
	}
}

type rlimitConfig struct {
	ID                           uint32                 `json:"id"`
	CPULimitParameters           elasticLimitParameters `json:"cpu_limit_parameters"`
	NetLimitParameters           elasticLimitParameters `json:"net_limit_parameters"`
	AccountCpuUsageAverageWindow uint32                 `json:"account_cpu_usage_average_window"`
	AccountNetUsageAverageWindow uint32                 `json:"account_net_usage_average_window"`
}

func (c *rlimitConfig) ToProto() *pbdeos.RlimitOp_Config {
	return &pbdeos.RlimitOp_Config{
		Config: &pbdeos.RlimitConfig{
			CpuLimitParameters:           c.CPULimitParameters.ToProto(),
			NetLimitParameters:           c.NetLimitParameters.ToProto(),
			AccountCpuUsageAverageWindow: c.AccountCpuUsageAverageWindow,
			AccountNetUsageAverageWindow: c.AccountNetUsageAverageWindow,
		},
	}
}

type rlimitAccountLimits struct {
	Owner     eos.AccountName `json:"owner"`
	NetWeight eos.Int64       `json:"net_weight"`
	CpuWeight eos.Int64       `json:"cpu_weight"`
	RamBytes  eos.Int64       `json:"ram_bytes"`
}

func (u *rlimitAccountLimits) ToProto() *pbdeos.RlimitOp_AccountLimits {
	return &pbdeos.RlimitOp_AccountLimits{
		AccountLimits: &pbdeos.RlimitAccountLimits{
			Owner:     string(u.Owner),
			NetWeight: int64(u.NetWeight),
			CpuWeight: int64(u.CpuWeight),
			RamBytes:  int64(u.RamBytes),
		},
	}
}

type rlimitAccountUsage struct {
	Owner    eos.AccountName   `json:"owner"`
	NetUsage *usageAccumulator `json:"net_usage"`
	CpuUsage *usageAccumulator `json:"cpu_usage"`
	RamUsage eos.Uint64        `json:"ram_usage"`
}

func (c *rlimitAccountUsage) ToProto() *pbdeos.RlimitOp_AccountUsage {
	return &pbdeos.RlimitOp_AccountUsage{
		AccountUsage: &pbdeos.RlimitAccountUsage{
			Owner:    string(c.Owner),
			NetUsage: c.NetUsage.ToProto(),
			CpuUsage: c.CpuUsage.ToProto(),
			RamUsage: uint64(c.RamUsage),
		},
	}
}

type usageAccumulator struct {
	LastOrdinal uint32     `json:"last_ordinal"`
	ValueEx     eos.Uint64 `json:"value_ex"`
	Consumed    eos.Uint64 `json:"consumed"`
}

func (a *usageAccumulator) ToProto() *pbdeos.UsageAccumulator {
	return &pbdeos.UsageAccumulator{
		LastOrdinal: a.LastOrdinal,
		ValueEx:     uint64(a.ValueEx),
		Consumed:    uint64(a.Consumed),
	}
}

type elasticLimitParameters struct {
	Target        eos.Uint64 `json:"target"`
	Max           eos.Uint64 `json:"max"`
	Periods       uint32     `json:"periods"`
	MaxMultiplier uint32     `json:"max_multiplier"`
	ContractRate  ratio      `json:"contract_rate"`
	ExpandRate    ratio      `json:"expand_rate"`
}

func (p *elasticLimitParameters) ToProto() *pbdeos.ElasticLimitParameters {
	return &pbdeos.ElasticLimitParameters{
		Target:        uint64(p.Target),
		Max:           uint64(p.Max),
		Periods:       p.Periods,
		MaxMultiplier: p.MaxMultiplier,
		ContractRate:  p.ContractRate.ToProto(),
		ExpandRate:    p.ExpandRate.ToProto(),
	}
}

type ratio struct {
	Numerator   eos.Uint64 `json:"numerator"`
	Denominator eos.Uint64 `json:"denominator"`
}

func (r *ratio) ToProto() *pbdeos.Ratio {
	return &pbdeos.Ratio{
		Numerator:   uint64(r.Numerator),
		Denominator: uint64(r.Denominator),
	}
}
