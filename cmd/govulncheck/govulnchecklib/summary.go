// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package govulnchecklib

import (
	"golang.org/x/vuln/cmd/govulncheck/internal/govulncheck"
	"golang.org/x/vuln/osv"
	"golang.org/x/vuln/vulncheck"
)

// Summary is the govulncheck result.
type Summary struct {
	// Vulnerabilities affecting the analysis target binary or source code.
	Affecting []Vuln
	// Vulnerabilities that may be imported but the vulnerable symbols are
	// not called. For binary analysis, this will be always empty.
	NonAffecting []Vuln
}

// Vuln represents a vulnerability relevant to a (module, package).
type Vuln struct {
	OSV     *osv.Entry
	PkgPath string // Package path.
	ModPath string // Module path.
	FoundIn string // <package path>@<version> if we know when it was introduced. Empty otherwise.
	FixedIn string // <package path>@<version> if fix is available. Empty otherwise.
	// Trace contains a call stack for each affecting symbol.
	// For vulnerabilities found from binary analysis, and vulnerabilities
	// that are reported as Unaffecting ones, this will be always empty.
	Trace []Trace
}

// Trace represents a sample trace for a vulnerable symbol.
type Trace struct {
	Symbol string       // Name of the detected vulnerable function or method.
	Desc   string       // One-line description of the callstack.
	Stack  []StackEntry // Call stack.
	Seen   int          // Number of similar call stacks.
}

// StackEntry represents a call stack entry.
type StackEntry struct {
	FuncName string // Function name is the function name, adjusted to remove pointer annotation.
	CallSite string // Position of the call/reference site. It is one of the formats token.Pos.String() returns or empty if unknown.
}

// summary summarize the analysis result.
func summary(ci *govulncheck.CallInfo, unaffected []*vulncheck.Vuln) Summary {
	var affecting, unaffecting []Vuln
	for _, vg := range ci.VulnGroups {
		// All the vulns in vg have the same PkgPath, ModPath and OSV.
		// All have a non-zero CallSink.
		v0 := vg[0]
		stacks := summarizeCallStacks(vg, ci)

		affecting = append(affecting, Vuln{
			OSV:     vg[0].OSV,
			PkgPath: v0.PkgPath,
			ModPath: v0.ModPath,
			FoundIn: foundVersion(v0.ModPath, v0.PkgPath, ci),
			FixedIn: fixedVersion(v0.PkgPath, v0.OSV.Affected),
			Trace:   stacks,
		})
	}
	for _, vuln := range unaffected {
		unaffecting = append(unaffecting, Vuln{
			OSV:     vuln.OSV,
			PkgPath: vuln.PkgPath,
			ModPath: vuln.ModPath,
			FoundIn: foundVersion(vuln.ModPath, vuln.PkgPath, ci),
			FixedIn: fixedVersion(vuln.PkgPath, vuln.OSV.Affected),
		})
	}
	return Summary{
		Affecting:    affecting,
		NonAffecting: unaffecting,
	}
}

func summarizeCallStacks(vg []*vulncheck.Vuln, ci *govulncheck.CallInfo) []Trace {
	cs := make([]Trace, 0, len(vg))
	// report one full call stack for each vuln.
	for _, v := range vg {
		css := ci.CallStacks[v]
		if len(css) == 0 {
			continue
		}
		stack := make([]StackEntry, 0, len(css))
		for _, e := range css[0] {
			stack = append(stack, StackEntry{
				FuncName: govulncheck.FuncName(e.Function),
				CallSite: govulncheck.FuncPos(e.Call),
			})
		}
		cs = append(cs, Trace{
			Symbol: v.Symbol,
			Desc:   govulncheck.SummarizeCallStack(css[0], ci.TopPackages, v.PkgPath),
			Stack:  stack,
			Seen:   len(css),
		})
	}
	return cs
}
