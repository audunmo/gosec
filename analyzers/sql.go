package analyzers

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

func newSQLInjectionAnalyzer(id, description string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     id,
		Doc:      description,
		Run:      runSQL,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

func getInstructions(result *SSAAnalyzerResult) []ssa.Instruction {
	var instrs []ssa.Instruction
	for _, fn := range result.SSA.SrcFuncs {
		for _, block := range fn.DomPreorder() {
			instrs = append(instrs, block.Instrs...)
		}
	}

	return instrs
}

var sqlCallIdents = map[string]map[string]int{
	"*database/sql.DB": {
		"Exec":            0,
		"ExecContext":     1,
		"Query":           0,
		"QueryContext":    1,
		"QueryRow":        0,
		"QueryRowContext": 1,
		"Prepare":         0,
		"PrepareContext":  1,
	},
	"*database/sql.Tx": {
		"Exec":            0,
		"ExecContext":     1,
		"Query":           0,
		"QueryContext":    1,
		"QueryRow":        0,
		"QueryRowContext": 1,
		"Prepare":         0,
		"PrepareContext":  1,
	},
}

func findQueryArg(call *ssa.Function) (*ssa.Parameter, error) {
	funcName := call.Name()
	pkgName := call.Pkg.Pkg.Name()
	i := -1
	if ni, ok := sqlCallIdents[pkgName]; ok {
		if i, ok = ni[funcName]; !ok {
			i = -1
		}
	}
	if i == -1 {
		return nil, fmt.Errorf("SQL argument index not found for %s.%s", pkgName, funcName)
	}
	if i >= len(call.Params) {
		return nil, nil
	}
	query := call.Params[i]
	return query, nil
}

func getStringParameters(params []*ssa.Parameter) []string {
	var strings []string
	for _, param := range params {
		switch p := param.Object().Type().(type) {
		case *types.Basic:
			if p.Kind() == types.String {
				strings = append(strings, p.String())
			}
		}
	}

	return strings
}

func runSQL(pass *analysis.Pass) (any, error) {
	result, err := getSSAResult(pass)
	if err != nil {
		return nil, err
	}

	for _, instr := range getInstructions(result) {
		switch instr := instr.(type) {
		case *ssa.Call:
			call := instr.Call.StaticCallee()
			query, err := findQueryArg(call)
			if err != nil {
				return nil, err
			}

			stringParams := getStringParameters(call.Params)
		}
	}

	return nil, nil
}
