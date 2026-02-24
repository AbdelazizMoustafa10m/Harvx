package compression

import (
	"context"
	"strings"
	"unicode"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*GoCompressor)(nil)

// GoCompressor implements LanguageCompressor for Go source code.
// It uses a line-by-line state machine parser to extract structural signatures.
// The compressor is stateless and safe for concurrent use.
type GoCompressor struct{}

// NewGoCompressor creates a Go compressor.
func NewGoCompressor() *GoCompressor {
	return &GoCompressor{}
}

// Compress parses Go source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *GoCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &goParser{}
	return p.parse(ctx, source)
}

// Language returns "go".
func (c *GoCompressor) Language() string {
	return "go"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *GoCompressor) SupportedNodeTypes() []string {
	return []string{
		"package_clause",
		"import_declaration",
		"function_declaration",
		"method_declaration",
		"type_declaration",
		"const_declaration",
		"var_declaration",
	}
}

// ---------------------------------------------------------------------------
// Go parser state machine
// ---------------------------------------------------------------------------

// goState tracks the current state of the Go line-by-line parser.
type goState int

const (
	goStateTopLevel      goState = iota // Scanning for declarations
	goStateInLineComment                // Accumulating // doc comment lines
	goStateInBlockComment               // Accumulating /* ... */ doc comment
	goStateInImport                     // Accumulating import (...) block
	goStateInType                       // Accumulating type declaration (struct/interface body or grouped type)
	goStateInConst                      // Accumulating const (...) block
	goStateInVar                        // Accumulating var (...) block
	goStateInFunc                       // Skipping function/method body by counting braces
)

// goParseCtx holds all mutable state for the Go line-by-line parser.
type goParseCtx struct {
	state          goState
	braceDepth     int
	accum          strings.Builder
	accumStartLine int
	docLines       []string // accumulated // comment lines
	sigs           []Signature
}

// goParser provides Go source parsing using a line-by-line state machine.
type goParser struct{}

// parse extracts structural signatures from Go source code.
func (p *goParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "go",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &goParseCtx{
		state: goStateTopLevel,
	}

	for i, line := range lines {
		// Check context cancellation every 1000 lines.
		if i%1000 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		lineNum := i + 1 // 1-based
		trimmed := strings.TrimSpace(line)

		switch pc.state {
		case goStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case goStateInLineComment:
			p.handleLineComment(pc, trimmed, line, lineNum)

		case goStateInBlockComment:
			p.handleBlockComment(pc, trimmed, line, lineNum)

		case goStateInImport:
			p.handleImportBlock(pc, trimmed, line, lineNum)

		case goStateInType:
			p.handleTypeBlock(pc, trimmed, line, lineNum)

		case goStateInConst:
			p.handleConstBlock(pc, trimmed, line, lineNum)

		case goStateInVar:
			p.handleVarBlock(pc, trimmed, line, lineNum)

		case goStateInFunc:
			p.handleFuncBody(pc, trimmed, line, lineNum)
		}
	}

	// Flush any pending accumulated block (e.g., unterminated at EOF).
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "go",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *goParser) flushPending(pc *goParseCtx, lastLine int) {
	switch pc.state {
	case goStateInImport:
		// Unterminated import block -- emit what we have.
		sig := Signature{
			Kind:      KindImport,
			Source:    pc.accum.String(),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()

	case goStateInType:
		text := pc.accum.String()
		kind, name := classifyGoType(text)
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      kind,
			Name:      name,
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil

	case goStateInConst:
		text := pc.accum.String()
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractGoGroupName(text, "const"),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil

	case goStateInVar:
		text := pc.accum.String()
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractGoGroupName(text, "var"),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
	}
}

// handleTopLevel processes a line when the parser is at the top level.
func (p *goParser) handleTopLevel(pc *goParseCtx, trimmed, line string, lineNum int) {
	// Build constraint or go directive -- skip (not a doc comment).
	if strings.HasPrefix(trimmed, "//go:") {
		// Emit as part of next declaration if needed, but typically standalone.
		// Treat as doc comment line.
		pc.docLines = append(pc.docLines, line)
		return
	}

	// Line comment: could be start of doc comment block.
	if strings.HasPrefix(trimmed, "//") {
		pc.docLines = append(pc.docLines, line)
		pc.state = goStateInLineComment
		return
	}

	// Block comment start.
	if strings.HasPrefix(trimmed, "/*") {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		if strings.Contains(trimmed, "*/") {
			// Single-line block comment -- treat as doc comment.
			pc.docLines = append(pc.docLines, line)
			return
		}
		pc.state = goStateInBlockComment
		return
	}

	// Empty line resets doc comment accumulation.
	if trimmed == "" {
		pc.docLines = nil
		return
	}

	// Package clause.
	if strings.HasPrefix(trimmed, "package ") {
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      KindImport,
			Name:      extractGoPackageName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		return
	}

	// Import declarations.
	if strings.HasPrefix(trimmed, "import ") || trimmed == "import(" {
		p.handleImportStart(pc, trimmed, line, lineNum)
		return
	}

	// Function/method declarations.
	if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "func(") {
		p.handleFuncDecl(pc, trimmed, line, lineNum)
		return
	}

	// Type declarations.
	if strings.HasPrefix(trimmed, "type ") {
		p.handleTypeDecl(pc, trimmed, line, lineNum)
		return
	}

	// Const declarations.
	if strings.HasPrefix(trimmed, "const ") || trimmed == "const(" {
		p.handleConstDecl(pc, trimmed, line, lineNum)
		return
	}

	// Var declarations.
	if strings.HasPrefix(trimmed, "var ") || trimmed == "var(" {
		p.handleVarDecl(pc, trimmed, line, lineNum)
		return
	}

	// Anything else at top level -- discard doc comment.
	pc.docLines = nil
}

// handleLineComment continues accumulating // doc comment lines.
func (p *goParser) handleLineComment(pc *goParseCtx, trimmed, line string, lineNum int) {
	// Continue accumulating // comment lines.
	if strings.HasPrefix(trimmed, "//") {
		pc.docLines = append(pc.docLines, line)
		return
	}

	// Non-comment line encountered -- transition back to top level and process.
	pc.state = goStateTopLevel
	p.handleTopLevel(pc, trimmed, line, lineNum)
}

// handleBlockComment accumulates /* ... */ block comment lines.
func (p *goParser) handleBlockComment(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if strings.Contains(trimmed, "*/") {
		// Block comment closed -- save as doc comment.
		pc.docLines = append(pc.docLines, pc.accum.String())
		pc.accum.Reset()
		pc.state = goStateTopLevel
	}
}

// handleImportStart processes the start of an import declaration.
func (p *goParser) handleImportStart(pc *goParseCtx, trimmed, line string, lineNum int) {
	doc := p.buildDocComment(pc.docLines)
	pc.docLines = nil

	// Single-line import: import "fmt"
	if !strings.Contains(trimmed, "(") {
		sig := Signature{
			Kind:      KindImport,
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		return
	}

	// Grouped import with closing on same line: import ( "fmt" )
	if strings.Contains(trimmed, ")") {
		sig := Signature{
			Kind:      KindImport,
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		return
	}

	// Multi-line import block.
	pc.accum.Reset()
	if doc != "" {
		pc.accum.WriteString(doc)
		pc.accum.WriteString("\n")
	}
	pc.accum.WriteString(line)
	pc.accumStartLine = lineNum
	pc.state = goStateInImport
}

// handleImportBlock accumulates lines in an import (...) block.
func (p *goParser) handleImportBlock(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if trimmed == ")" {
		sig := Signature{
			Kind:      KindImport,
			Source:    pc.accum.String(),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.state = goStateTopLevel
	}
}

// handleFuncDecl processes func/method declaration lines.
func (p *goParser) handleFuncDecl(pc *goParseCtx, trimmed, line string, lineNum int) {
	doc := p.buildDocComment(pc.docLines)
	pc.docLines = nil

	// Extract the signature (everything up to the opening brace).
	sigText := extractGoFuncSignature(line)
	name := extractGoFuncName(trimmed)

	sig := Signature{
		Kind:      KindFunction,
		Name:      name,
		Source:    maybePrependDoc(doc, nil, sigText),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)

	// Count braces to determine if we need to skip a body.
	braces := goCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = goStateInFunc
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Opening and closing on same line: func init() {}
		// Already emitted signature, stay at top level.
	}
	// If no brace at all (e.g., interface method or forward decl), stay at top level.
}

// handleFuncBody skips function/method body by counting braces.
func (p *goParser) handleFuncBody(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.braceDepth += goCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = goStateTopLevel
	}
}

// handleTypeDecl processes type declarations.
func (p *goParser) handleTypeDecl(pc *goParseCtx, trimmed, line string, lineNum int) {
	doc := p.buildDocComment(pc.docLines)

	// Grouped type declaration: type (
	if isGoGroupedDecl(trimmed, "type") {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = 0
		pc.state = goStateInType
		return
	}

	// Determine if this is a struct or interface with body.
	if isGoStructOrInterfaceLine(trimmed) {
		braces := goCountBraces(trimmed)
		if braces > 0 {
			// Multi-line struct/interface -- accumulate body.
			pc.accum.Reset()
			pc.accum.WriteString(line)
			pc.accumStartLine = lineNum
			pc.braceDepth = braces
			pc.state = goStateInType
			return
		}
		// Single-line struct/interface: type Foo struct{} or type Foo struct{ X int }
		kind, name := classifyGoType(line)
		sig := Signature{
			Kind:      kind,
			Name:      name,
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		return
	}

	// Simple type alias or definition: type ID = string, type ID string
	kind, name := classifyGoType(trimmed)
	sig := Signature{
		Kind:      kind,
		Name:      name,
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docLines = nil
}

// handleTypeBlock accumulates lines in a type block (struct/interface body or grouped type).
func (p *goParser) handleTypeBlock(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)

	// For grouped type ( ... ) we track parentheses.
	firstLine := getFirstLine(pc.accum.String())
	isGrouped := isGoGroupedDecl(strings.TrimSpace(firstLine), "type")

	if isGrouped {
		if trimmed == ")" {
			text := pc.accum.String()
			doc := p.buildDocComment(pc.docLines)
			sig := Signature{
				Kind:      KindType,
				Name:      extractGoGroupName(text, "type"),
				Source:    maybePrependDoc(doc, nil, text),
				StartLine: pc.accumStartLine,
				EndLine:   lineNum,
			}
			pc.sigs = append(pc.sigs, sig)
			pc.accum.Reset()
			pc.docLines = nil
			pc.state = goStateTopLevel
		}
		return
	}

	// Struct/interface body -- track braces.
	pc.braceDepth += goCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		kind, name := classifyGoType(text)
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      kind,
			Name:      name,
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.braceDepth = 0
		pc.state = goStateTopLevel
	}
}

// handleConstDecl processes const declarations.
func (p *goParser) handleConstDecl(pc *goParseCtx, trimmed, line string, lineNum int) {
	doc := p.buildDocComment(pc.docLines)

	// Grouped const: const (
	if isGoGroupedDecl(trimmed, "const") {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = goStateInConst
		return
	}

	// Single const: const MaxSize = 1024
	name := extractGoSingleDeclName(trimmed, "const")
	sig := Signature{
		Kind:      KindConstant,
		Name:      name,
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docLines = nil
}

// handleConstBlock accumulates lines in a const (...) block.
func (p *goParser) handleConstBlock(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if trimmed == ")" {
		text := pc.accum.String()
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractGoGroupName(text, "const"),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.state = goStateTopLevel
	}
}

// handleVarDecl processes var declarations.
func (p *goParser) handleVarDecl(pc *goParseCtx, trimmed, line string, lineNum int) {
	doc := p.buildDocComment(pc.docLines)

	// Grouped var: var (
	if isGoGroupedDecl(trimmed, "var") {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = goStateInVar
		return
	}

	// Single var: var ErrNotFound = errors.New("not found")
	name := extractGoSingleDeclName(trimmed, "var")
	sig := Signature{
		Kind:      KindConstant,
		Name:      name,
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docLines = nil
}

// handleVarBlock accumulates lines in a var (...) block.
func (p *goParser) handleVarBlock(pc *goParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if trimmed == ")" {
		text := pc.accum.String()
		doc := p.buildDocComment(pc.docLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractGoGroupName(text, "var"),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.state = goStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Go-specific helper functions
// ---------------------------------------------------------------------------

// buildDocComment joins accumulated doc comment lines into a single string.
// Returns empty string if there are no doc lines.
func (p *goParser) buildDocComment(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// extractGoPackageName extracts the package name from a "package xxx" line.
func extractGoPackageName(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "package ")
	rest = strings.TrimSpace(rest)
	return extractGoIdentifier(rest)
}

// extractGoFuncSignature extracts the function/method signature up to (but not
// including) the opening brace of the function body. Handles cases like
// `func foo(v interface{}) {` where `interface{}` braces are NOT the body opener.
func extractGoFuncSignature(line string) string {
	bodyIdx := findFuncBodyBrace(line)
	if bodyIdx == -1 {
		return strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(line[:bodyIdx], " \t")
}

// findFuncBodyBrace finds the index of the `{` that opens the function body.
// After all parentheses are balanced (parameter and return lists closed),
// the next `{` at brace depth 0 is the body opener.
func findFuncBodyBrace(line string) int {
	parenDepth := 0
	braceDepth := 0
	inDoubleQuote := false
	inRawString := false
	inRune := false
	escaped := false
	seenParens := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			continue
		}
		if inRawString {
			if ch == '`' {
				inRawString = false
			}
			continue
		}
		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}
		if inRune {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inRune = false
			}
			continue
		}
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '`':
			inRawString = true
		case '\'':
			inRune = true
		case '(':
			parenDepth++
			seenParens = true
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			if seenParens && parenDepth == 0 && braceDepth == 0 {
				return i
			}
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		}
	}
	return -1
}

// extractGoFuncName extracts the function or method name from a func declaration.
// Handles both "func Name(...)" and "func (recv Type) Name(...)".
func extractGoFuncName(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "func ")

	// Method with receiver: func (recv Type) Name(...)
	if strings.HasPrefix(rest, "(") {
		// Find matching closing paren for the receiver.
		depth := 0
		closeParen := -1
		for i, ch := range rest {
			switch ch {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					closeParen = i
				}
			}
			if closeParen >= 0 {
				break
			}
		}
		if closeParen >= 0 && closeParen+1 < len(rest) {
			rest = strings.TrimSpace(rest[closeParen+1:])
		}
	}

	return extractGoIdentifier(rest)
}

// extractGoIdentifier extracts the first Go identifier from a string.
func extractGoIdentifier(s string) string {
	var b strings.Builder
	for i, ch := range s {
		if i == 0 {
			if !unicode.IsLetter(ch) && ch != '_' {
				break
			}
		} else {
			if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
				break
			}
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// isGoGroupedDecl checks if a trimmed line starts a grouped declaration:
// "const (", "var (", "type (".
func isGoGroupedDecl(trimmed, keyword string) bool {
	// "const (" or "const("
	return trimmed == keyword+" (" ||
		trimmed == keyword+"(" ||
		strings.HasPrefix(trimmed, keyword+" (") ||
		strings.HasPrefix(trimmed, keyword+"(")
}

// isGoStructOrInterfaceLine checks if a type declaration line contains struct or interface.
func isGoStructOrInterfaceLine(trimmed string) bool {
	// type Foo struct { ... } or type Foo interface { ... }
	// Also handles generics: type Set[T comparable] struct { ... }
	return strings.Contains(trimmed, " struct") || strings.Contains(trimmed, " interface")
}

// classifyGoType determines the SignatureKind and name for a type declaration.
func classifyGoType(text string) (SignatureKind, string) {
	firstLine := getFirstLine(text)
	trimmed := strings.TrimSpace(firstLine)
	name := extractGoTypeName(trimmed)

	if strings.Contains(trimmed, " struct") {
		return KindStruct, name
	}
	if strings.Contains(trimmed, " interface") {
		return KindInterface, name
	}
	return KindType, name
}

// extractGoTypeName extracts the type name from a type declaration.
// Handles: type Name ..., type Name[T any] ...
func extractGoTypeName(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "type ")
	rest = strings.TrimSpace(rest)
	return extractGoIdentifier(rest)
}

// extractGoSingleDeclName extracts the name from a single-line const or var declaration.
// "const MaxSize = 1024" -> "MaxSize", "var x int" -> "x".
func extractGoSingleDeclName(trimmed, keyword string) string {
	rest := strings.TrimPrefix(trimmed, keyword+" ")
	rest = strings.TrimSpace(rest)
	return extractGoIdentifier(rest)
}

// extractGoGroupName extracts the first name from a grouped declaration for labeling.
// For "const (\n  Foo = 1\n  Bar = 2\n)", returns "Foo".
func extractGoGroupName(text, keyword string) string {
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		t := strings.TrimSpace(l)
		// Skip the keyword line and closing paren.
		if t == "" || strings.HasPrefix(t, keyword) || t == ")" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") {
			continue
		}
		return extractGoIdentifier(t)
	}
	return ""
}

// getFirstLine returns the first line of a multi-line string.
func getFirstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx != -1 {
		return s[:idx]
	}
	return s
}

// goCountBraces counts net brace depth for a Go source line, ignoring braces
// inside string literals (double-quoted, backtick/raw), rune literals, and comments.
func goCountBraces(line string) int {
	depth := 0
	inDoubleQuote := false
	inRawString := false
	inRune := false
	escaped := false
	inLineComment := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			continue
		}

		// Inside a line comment -- ignore rest of line.
		if inLineComment {
			break
		}

		// Inside a raw string literal (backtick).
		if inRawString {
			if ch == '`' {
				inRawString = false
			}
			continue
		}

		// Inside a double-quoted string.
		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		// Inside a rune literal.
		if inRune {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inRune = false
			}
			continue
		}

		// Check for line comment start.
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			inLineComment = true
			break
		}

		// Check for block comment on same line -- simplified handling.
		if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
			// Skip until */ on same line.
			end := strings.Index(line[i+2:], "*/")
			if end >= 0 {
				i = i + 2 + end + 1 // skip past */
				continue
			}
			// Block comment doesn't close on this line; ignore rest.
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '`':
			inRawString = true
		case '\'':
			inRune = true
		case '{':
			depth++
		case '}':
			depth--
		}
	}

	return depth
}
