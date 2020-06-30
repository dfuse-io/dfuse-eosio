package filtering

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/google/cel-go/cel"
)

type BlockFilter struct {
	InProgram  *CELFilter
	OutProgram *CELFilter
}

func NewBlockFilter(inProgramCode, outProgramCode string) (*BlockFilter, error) {
	inFilter, err := NewCELFilterIn(inProgramCode)
	if err != nil {
		return nil, fmt.Errorf("in filter: %w", err)
	}

	outFilter, err := NewCELFilterOut(outProgramCode)
	if err != nil {
		return nil, fmt.Errorf("out filter: %w", err)
	}

	return &BlockFilter{
		InProgram:  inFilter,
		OutProgram: outFilter,
	}, nil
}

func (f *BlockFilter) TransformInPlace(block *pbcodec.Block) {
	block.FilteringKind = pbcodec.FilteringKind_FILTERINGKIND_FILTERED
	block.FilteringMetadata = &pbcodec.FilteringMetadata{
		CelFilterIn:  f.InProgram.Code,
		CelFilterOut: f.OutProgram.Code,
	}
}

type CELFilter struct {
	Code    string
	Program cel.Program
}

func NewCELFilterIn(code string) (*CELFilter, error) {
	inProgram, err := buildCELProgram("true", code)
	if err != nil {
		return nil, err
	}

	return &CELFilter{
		Code:    code,
		Program: inProgram,
	}, nil
}

func NewCELFilterOut(code string) (*CELFilter, error) {
	outProgram, err := buildCELProgram("false", code)
	if err != nil {
		return nil, err
	}

	return &CELFilter{
		Code:    code,
		Program: outProgram,
	}, nil
}
