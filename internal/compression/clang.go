package compression

import (
	"context"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*CCompressor)(nil)

// CCompressor implements LanguageCompressor for C source code.
// It uses a line-by-line state machine parser to extract structural signatures.
// The compressor is stateless and safe for concurrent use.
type CCompressor struct{}

// NewCCompressor creates a C compressor.
func NewCCompressor() *CCompressor {
	return &CCompressor{}
}

// Compress parses C source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *CCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &cParser{}
	return p.parse(ctx, source)
}

// Language returns "c".
func (c *CCompressor) Language() string {
	return "c"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *CCompressor) SupportedNodeTypes() []string {
	return []string{
		"preproc_include",
		"preproc_def",
		"function_definition",
		"declaration",
		"struct_specifier",
		"enum_specifier",
		"type_definition",
	}
}

// ---------------------------------------------------------------------------
// C parser
// ---------------------------------------------------------------------------

// cParser provides C source parsing using a line-by-line state machine.
type cParser struct{}

// parse extracts structural signatures from C source code.
func (p *cParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "c",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &cParseCtx{
		state: cStateTopLevel,
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
		case cStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case cStateInBlockComment:
			p.handleBlockComment(pc, trimmed, line, lineNum)

		case cStateInFuncBody:
			p.handleFuncBody(pc, trimmed)

		case cStateInStructBody:
			p.handleStructBody(pc, trimmed, line, lineNum)

		case cStateInEnumBody:
			p.handleEnumBody(pc, trimmed, line, lineNum)

		case cStateInPreproc:
			p.handlePreproc(pc, trimmed, line, lineNum)
		}
	}

	// Flush any pending state at EOF.
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "c",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *cParser) flushPending(pc *cParseCtx, lastLine int) {
	switch pc.state {
	case cStateInStructBody:
		text := pc.accum.String()
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractCStructName(getFirstLine(text)),
			Source:    maybePrependDoc(pc.docComment, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docComment = ""

	case cStateInEnumBody:
		text := pc.accum.String()
		sig := Signature{
			Kind:      KindType,
			Name:      extractCEnumName(getFirstLine(text)),
			Source:    maybePrependDoc(pc.docComment, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docComment = ""

	case cStateInPreproc:
		// Flush pending multi-line preprocessor.
		text := pc.preprocAccum.String()
		firstLine := getFirstLine(text)
		trimFirst := strings.TrimSpace(firstLine)
		if isCDefine(trimFirst) {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindConstant,
				Name:      extractCDefineNameOnly(trimFirst),
				Source:    extractCDefineNameLine(trimFirst),
				StartLine: pc.accumStartLine,
				EndLine:   lastLine,
			})
		}
		pc.preprocAccum.Reset()
		pc.docComment = ""
	}
}

// ---------------------------------------------------------------------------
// Top-level handler
// ---------------------------------------------------------------------------

// handleTopLevel processes a line when the parser is at the top level.
func (p *cParser) handleTopLevel(pc *cParseCtx, trimmed, line string, lineNum int) {
	// Empty line resets doc comment.
	if trimmed == "" {
		pc.docComment = ""
		return
	}

	// Doc comment start: /** ...
	if isCDocCommentStart(trimmed) {
		if strings.Contains(trimmed, "*/") {
			// Single-line doc comment.
			pc.docComment = line
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = cStateInBlockComment
		return
	}

	// Block comment (non-doc) -- skip.
	if isCBlockCommentStart(trimmed) {
		if strings.Contains(trimmed[2:], "*/") {
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.docComment = ""
		pc.state = cStateInBlockComment
		return
	}

	// Single-line // comments may be doc comments if they precede a declaration.
	if isCLineComment(trimmed) {
		if pc.docComment != "" {
			pc.docComment += "\n" + line
		} else {
			pc.docComment = line
		}
		return
	}

	// Preprocessor directives.
	if isCPreprocessorDirective(trimmed) {
		p.handlePreprocessor(pc, trimmed, line, lineNum)
		return
	}

	// Typedef (before struct/enum since typedef struct is both).
	if isCTypedef(trimmed) {
		p.handleTypedef(pc, trimmed, line, lineNum)
		return
	}

	// Struct declaration.
	if isCStructDecl(trimmed) {
		p.handleStructDecl(pc, trimmed, line, lineNum)
		return
	}

	// Enum declaration.
	if isCEnumDecl(trimmed) {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

	// Forward declarations (struct Foo;, enum Bar;).
	if isCForwardDecl(trimmed) {
		sig := Signature{
			Kind:      KindType,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Function definition.
	if isCFuncDefinition(trimmed) {
		p.handleFuncDef(pc, trimmed, line, lineNum)
		return
	}

	// Function prototype.
	if isCFuncPrototype(trimmed) {
		name := extractCFuncName(trimmed)
		sig := Signature{
			Kind:      KindFunction,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Global variable declaration.
	if isCGlobalVarDecl(trimmed) {
		name := extractCGlobalVarName(trimmed)
		sig := Signature{
			Kind:      KindConstant,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Anything else -- discard doc comment.
	pc.docComment = ""
}

// ---------------------------------------------------------------------------
// Block comment handler
// ---------------------------------------------------------------------------

// handleBlockComment accumulates /* ... */ block comment lines.
func (p *cParser) handleBlockComment(pc *cParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if strings.Contains(trimmed, "*/") {
		text := pc.accum.String()
		// Check if it was a doc comment (starts with /**).
		firstLine := getFirstLine(text)
		if strings.Contains(strings.TrimSpace(firstLine), "/**") {
			pc.docComment = text
		}
		pc.accum.Reset()
		pc.state = cStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Function body handler (skipping)
// ---------------------------------------------------------------------------

// handleFuncBody skips function body by counting braces.
func (p *cParser) handleFuncBody(pc *cParseCtx, trimmed string) {
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = cStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Struct body handler (accumulating)
// ---------------------------------------------------------------------------

// handleStructBody accumulates struct body lines.
func (p *cParser) handleStructBody(pc *cParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		name := extractCStructName(getFirstLine(text))
		// Check for typedef struct -- the typedef name is after the closing brace.
		firstLine := strings.TrimSpace(getFirstLine(text))
		if strings.HasPrefix(firstLine, "typedef struct") {
			typedefName := extractCTypedefTrailingName(trimmed)
			if typedefName != "" {
				name = typedefName
			}
		}
		sig := Signature{
			Kind:      KindStruct,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docComment = ""
		pc.braceDepth = 0
		pc.state = cStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Enum body handler (accumulating)
// ---------------------------------------------------------------------------

// handleEnumBody accumulates enum body lines.
func (p *cParser) handleEnumBody(pc *cParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		name := extractCEnumName(getFirstLine(text))
		// Check for typedef enum.
		firstLine := strings.TrimSpace(getFirstLine(text))
		if strings.HasPrefix(firstLine, "typedef enum") {
			typedefName := extractCTypedefTrailingName(trimmed)
			if typedefName != "" {
				name = typedefName
			}
		}
		sig := Signature{
			Kind:      KindType,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docComment = ""
		pc.braceDepth = 0
		pc.state = cStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Preprocessor handler
// ---------------------------------------------------------------------------

// handlePreprocessor processes a preprocessor directive at the top level.
func (p *cParser) handlePreprocessor(pc *cParseCtx, trimmed, line string, lineNum int) {
	// #include -- verbatim.
	if isCInclude(trimmed) {
		sig := Signature{
			Kind:      KindImport,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// #define
	if isCDefine(trimmed) {
		// Check for multi-line define (ends with \).
		if isMultiLineDefine(trimmed) {
			pc.preprocAccum.Reset()
			pc.preprocAccum.WriteString(line)
			pc.accumStartLine = lineNum
			pc.state = cStateInPreproc
			return
		}
		// Single-line define: emit name and parameters only.
		nameLine := extractCDefineNameLine(trimmed)
		nameOnly := extractCDefineNameOnly(trimmed)
		sig := Signature{
			Kind:      KindConstant,
			Name:      nameOnly,
			Source:    nameLine,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Other preprocessor directives (#ifdef, #ifndef, #endif, #pragma, etc.) -- skip.
	pc.docComment = ""
}

// handlePreproc accumulates multi-line preprocessor directive lines.
func (p *cParser) handlePreproc(pc *cParseCtx, trimmed, line string, lineNum int) {
	pc.preprocAccum.WriteString("\n")
	pc.preprocAccum.WriteString(line)

	// If this line does NOT end with \, the directive is complete.
	if !isMultiLineDefine(trimmed) {
		text := pc.preprocAccum.String()
		firstLine := getFirstLine(text)
		trimFirst := strings.TrimSpace(firstLine)
		if isCDefine(trimFirst) {
			nameOnly := extractCDefineNameOnly(trimFirst)
			nameLine := extractCDefineNameLine(trimFirst)
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindConstant,
				Name:      nameOnly,
				Source:    nameLine,
				StartLine: pc.accumStartLine,
				EndLine:   lineNum,
			})
		}
		pc.preprocAccum.Reset()
		pc.docComment = ""
		pc.state = cStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Declaration handlers
// ---------------------------------------------------------------------------

// handleFuncDef processes a function definition line.
func (p *cParser) handleFuncDef(pc *cParseCtx, trimmed, line string, lineNum int) {
	sigText := extractCFuncSignature(line)
	name := extractCFuncName(trimmed)

	sig := Signature{
		Kind:      KindFunction,
		Name:      name,
		Source:    maybePrependDoc(pc.docComment, nil, sigText),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docComment = ""

	braces := cCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = cStateInFuncBody
	}
	// If braces == 0, the function body opened and closed on the same line.
}

// handleTypedef processes a typedef statement.
func (p *cParser) handleTypedef(pc *cParseCtx, trimmed, line string, lineNum int) {
	// typedef struct/enum with body go to their respective handlers.
	if strings.HasPrefix(trimmed, "typedef struct") {
		p.handleStructDecl(pc, trimmed, line, lineNum)
		return
	}
	if strings.HasPrefix(trimmed, "typedef enum") {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

	// Simple typedef (including function pointers): full verbatim line.
	sig := Signature{
		Kind:      KindType,
		Source:    maybePrependDoc(pc.docComment, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docComment = ""
}

// handleStructDecl processes a struct declaration.
func (p *cParser) handleStructDecl(pc *cParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

	// Forward declaration or single-line struct.
	if strings.HasSuffix(trimmed, ";") && braces == 0 {
		name := extractCStructName(trimmed)
		sig := Signature{
			Kind:      KindStruct,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Single-line struct with body: struct Foo { int x; };
	if strings.Contains(trimmed, "{") && braces == 0 {
		name := extractCStructName(trimmed)
		sig := Signature{
			Kind:      KindStruct,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Multi-line struct.
	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = cStateInStructBody
		return
	}

	// Struct without body on this line -- handle gracefully.
	name := extractCStructName(trimmed)
	sig := Signature{
		Kind:      KindStruct,
		Name:      name,
		Source:    maybePrependDoc(pc.docComment, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docComment = ""
}

// handleEnumDecl processes an enum declaration.
func (p *cParser) handleEnumDecl(pc *cParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

	// Single-line enum.
	if strings.Contains(trimmed, "{") && braces == 0 {
		name := extractCEnumName(trimmed)
		sig := Signature{
			Kind:      KindType,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Multi-line enum.
	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = cStateInEnumBody
		return
	}

	// Enum without body.
	name := extractCEnumName(trimmed)
	sig := Signature{
		Kind:      KindType,
		Name:      name,
		Source:    maybePrependDoc(pc.docComment, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docComment = ""
}

// ---------------------------------------------------------------------------
// Additional C-specific helpers
// ---------------------------------------------------------------------------

// extractCDefineNameOnly extracts just the macro name (without #define prefix or params).
func extractCDefineNameOnly(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "#define ")
	rest = strings.TrimSpace(rest)
	return extractCIdentifier(rest)
}

// extractCGlobalVarName extracts the variable name from a global declaration.
// For "static const int max_size = 100;", returns "max_size".
// For "int counter;", returns "counter".
func extractCGlobalVarName(trimmed string) string {
	s := strings.TrimSuffix(trimmed, ";")
	s = strings.TrimSpace(s)

	// Remove initialization: everything after =.
	if eqIdx := strings.Index(s, "="); eqIdx != -1 {
		s = strings.TrimSpace(s[:eqIdx])
	}

	// Remove array brackets: name[10] -> name.
	if brIdx := strings.Index(s, "["); brIdx != -1 {
		s = strings.TrimSpace(s[:brIdx])
	}

	// The variable name is the last identifier token.
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return ""
	}
	last := tokens[len(tokens)-1]
	// Strip pointer/reference markers.
	last = strings.TrimLeft(last, "*&")
	return last
}

// extractCTypedefTrailingName extracts the typedef alias name from the closing
// line of a typedef struct/enum: "} MyType;" -> "MyType".
func extractCTypedefTrailingName(trimmed string) string {
	// Find } and then the identifier before ;.
	braceIdx := strings.LastIndex(trimmed, "}")
	if braceIdx == -1 {
		return ""
	}
	after := strings.TrimSpace(trimmed[braceIdx+1:])
	after = strings.TrimSuffix(after, ";")
	after = strings.TrimSpace(after)
	if after == "" {
		return ""
	}
	// May have pointer: } *MyType;
	after = strings.TrimLeft(after, "*")
	return extractCIdentifier(after)
}