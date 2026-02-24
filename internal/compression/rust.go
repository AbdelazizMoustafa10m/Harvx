package compression

import (
	"context"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*RustCompressor)(nil)

// RustCompressor implements LanguageCompressor for Rust source code.
// It uses a line-by-line state machine parser to extract structural signatures.
// The compressor is stateless and safe for concurrent use.
type RustCompressor struct{}

// NewRustCompressor creates a Rust compressor.
func NewRustCompressor() *RustCompressor {
	return &RustCompressor{}
}

// Compress parses Rust source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *RustCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &rustParser{}
	return p.parse(ctx, source)
}

// Language returns "rust".
func (c *RustCompressor) Language() string {
	return "rust"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *RustCompressor) SupportedNodeTypes() []string {
	return []string{
		"use_declaration",
		"function_item",
		"struct_item",
		"enum_item",
		"trait_item",
		"impl_item",
		"type_alias",
		"const_item",
		"static_item",
		"mod_item",
		"macro_definition",
		"extern_block",
	}
}

// ---------------------------------------------------------------------------
// Rust parser state machine
// ---------------------------------------------------------------------------

// rustState tracks the current state of the Rust line-by-line parser.
type rustState int

const (
	rustStateTopLevel       rustState = iota // Scanning for declarations
	rustStateInBlockComment                  // Inside /* ... */ block comment
	rustStateInFnBody                        // Skipping function body by counting braces
	rustStateInStructBody                    // Accumulating struct body (all fields)
	rustStateInEnumBody                      // Accumulating enum body (all variants)
	rustStateInTraitBody                     // Extracting trait method signatures
	rustStateInImplBody                      // Extracting impl method signatures
	rustStateInExternBlock                   // Extracting extern "C" block items
	rustStateInMacroBody                     // Skipping macro_rules! body
)

// rustParseCtx holds all mutable state for the Rust line-by-line parser.
type rustParseCtx struct {
	state      rustState
	braceDepth int

	// Accumulator for multi-line constructs (struct, enum, trait header, impl header, etc.).
	accum          strings.Builder
	accumStartLine int

	// Doc comment and attribute lines pending attachment to the next declaration.
	docLines   []string
	attrLines  []string

	// For trait/impl bodies: the header line and accumulated method signatures.
	blockHeader     string
	blockName       string
	blockStartLine  int
	blockDocComment string
	blockAttrs      []string
	blockMethods    []string

	// Collected signatures.
	sigs []Signature
}

// rustParser provides Rust source parsing using a line-by-line state machine.
type rustParser struct{}

// parse extracts structural signatures from Rust source code.
func (p *rustParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "rust",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &rustParseCtx{
		state: rustStateTopLevel,
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
		case rustStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case rustStateInBlockComment:
			p.handleBlockComment(pc, trimmed, line, lineNum)

		case rustStateInFnBody:
			p.handleFnBody(pc, trimmed, lineNum)

		case rustStateInStructBody:
			p.handleStructBody(pc, trimmed, line, lineNum)

		case rustStateInEnumBody:
			p.handleEnumBody(pc, trimmed, line, lineNum)

		case rustStateInTraitBody:
			p.handleTraitBody(pc, trimmed, line, lineNum)

		case rustStateInImplBody:
			p.handleImplBody(pc, trimmed, line, lineNum)

		case rustStateInExternBlock:
			p.handleExternBlock(pc, trimmed, line, lineNum)

		case rustStateInMacroBody:
			p.handleMacroBody(pc, trimmed, lineNum)
		}
	}

	// Flush any pending accumulated block at EOF.
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "rust",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *rustParser) flushPending(pc *rustParseCtx, lastLine int) {
	switch pc.state {
	case rustStateInStructBody:
		text := pc.accum.String()
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractRustStructName(getFirstLine(text)),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.attrLines = nil

	case rustStateInEnumBody:
		text := pc.accum.String()
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindType,
			Name:      extractRustEnumName(getFirstLine(text)),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.attrLines = nil

	case rustStateInTraitBody, rustStateInImplBody:
		p.emitBlockSignature(pc, lastLine)

	case rustStateInExternBlock:
		p.emitBlockSignature(pc, lastLine)
	}
}

// ---------------------------------------------------------------------------
// Top-level handler
// ---------------------------------------------------------------------------

// handleTopLevel processes a line when the parser is at the top level.
func (p *rustParser) handleTopLevel(pc *rustParseCtx, trimmed, line string, lineNum int) {
	// Empty line resets doc comment and attribute accumulation.
	if trimmed == "" {
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Block comment start: /* ...
	if strings.HasPrefix(trimmed, "/*") {
		if strings.Contains(trimmed, "*/") {
			// Single-line block comment -- discard.
			return
		}
		pc.state = rustStateInBlockComment
		return
	}

	// Inner doc comment: //! ...
	if strings.HasPrefix(trimmed, "//!") {
		sig := Signature{
			Kind:      KindDocComment,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		return
	}

	// Outer doc comment: /// ...
	if strings.HasPrefix(trimmed, "///") {
		pc.docLines = append(pc.docLines, line)
		return
	}

	// Regular line comment: // -- skip (not a doc comment).
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Attribute: #[...] or #![...]
	if strings.HasPrefix(trimmed, "#[") || strings.HasPrefix(trimmed, "#![") {
		pc.attrLines = append(pc.attrLines, line)
		return
	}

	// use declaration.
	if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "pub use ") ||
		strings.HasPrefix(trimmed, "pub(crate) use ") || strings.HasPrefix(trimmed, "pub(super) use ") {
		sig := Signature{
			Kind:      KindImport,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// mod declaration.
	if isRustModDecl(trimmed) {
		// mod with body: mod foo { ... }
		if strings.Contains(trimmed, "{") {
			// If it closes on same line, emit verbatim.
			if rustCountBraces(trimmed) == 0 {
				sig := Signature{
					Kind:      KindImport,
					Name:      extractRustModName(trimmed),
					Source:    line,
					StartLine: lineNum,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
			} else {
				// Multi-line mod body -- skip it by counting braces.
				pc.braceDepth = rustCountBraces(trimmed)
				pc.state = rustStateInFnBody
			}
			pc.docLines = nil
			pc.attrLines = nil
			return
		}
		// Simple mod declaration: mod foo;
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindImport,
			Name:      extractRustModName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// macro_rules! declaration.
	if isRustMacroRules(trimmed) {
		name := extractRustMacroName(trimmed)
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindFunction,
			Name:      name,
			Source:    maybePrependDoc(doc, nil, "macro_rules! "+name),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		// Skip macro body.
		braces := rustCountBraces(trimmed)
		if braces > 0 {
			pc.braceDepth = braces
			pc.state = rustStateInMacroBody
		}
		return
	}

	// extern "C" block.
	if isRustExternBlock(trimmed) {
		pc.blockHeader = line
		pc.blockName = ""
		pc.blockStartLine = lineNum
		pc.blockDocComment = buildRustDoc(pc.docLines, pc.attrLines)
		pc.blockAttrs = nil
		pc.blockMethods = nil
		pc.docLines = nil
		pc.attrLines = nil

		braces := rustCountBraces(trimmed)
		if braces > 0 {
			pc.braceDepth = braces
			pc.state = rustStateInExternBlock
		}
		return
	}

	// struct declaration.
	if isRustStructDecl(trimmed) {
		p.handleStructDecl(pc, trimmed, line, lineNum)
		return
	}

	// enum declaration.
	if isRustEnumDecl(trimmed) {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

	// trait declaration.
	if isRustTraitDecl(trimmed) {
		p.handleTraitDecl(pc, trimmed, line, lineNum)
		return
	}

	// impl block.
	if isRustImplDecl(trimmed) {
		p.handleImplDecl(pc, trimmed, line, lineNum)
		return
	}

	// Function declaration (standalone, not inside impl/trait).
	if isRustFnDecl(trimmed) {
		p.handleFnDecl(pc, trimmed, line, lineNum)
		return
	}

	// type alias.
	if isRustTypeAlias(trimmed) {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindType,
			Name:      extractRustTypeAliasName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// const item.
	if isRustConstDecl(trimmed) {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractRustConstName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// static item.
	if isRustStaticDecl(trimmed) {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindConstant,
			Name:      extractRustStaticName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Anything else at top level: discard accumulated doc/attrs.
	pc.docLines = nil
	pc.attrLines = nil
}

// ---------------------------------------------------------------------------
// Block comment handler
// ---------------------------------------------------------------------------

// handleBlockComment skips lines inside a /* ... */ block comment.
func (p *rustParser) handleBlockComment(pc *rustParseCtx, trimmed, line string, lineNum int) {
	if strings.Contains(trimmed, "*/") {
		pc.state = rustStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Function body handler (skipping)
// ---------------------------------------------------------------------------

// handleFnBody skips function body by counting braces.
func (p *rustParser) handleFnBody(pc *rustParseCtx, trimmed string, lineNum int) {
	pc.braceDepth += rustCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Struct body handler (accumulating)
// ---------------------------------------------------------------------------

// handleStructBody accumulates struct body lines.
func (p *rustParser) handleStructBody(pc *rustParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += rustCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractRustStructName(getFirstLine(text)),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.attrLines = nil
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Enum body handler (accumulating)
// ---------------------------------------------------------------------------

// handleEnumBody accumulates enum body lines.
func (p *rustParser) handleEnumBody(pc *rustParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += rustCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindType,
			Name:      extractRustEnumName(getFirstLine(text)),
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.docLines = nil
		pc.attrLines = nil
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Trait body handler (extracting method signatures)
// ---------------------------------------------------------------------------

// handleTraitBody processes lines inside a trait body.
func (p *rustParser) handleTraitBody(pc *rustParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += rustCountBraces(trimmed)

	// Trait closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
		return
	}

	// We only extract members at depth 1 (the trait body level).
	if prevDepth != 1 {
		return
	}

	// Skip empty lines.
	if trimmed == "" {
		return
	}

	// Doc comments inside trait body.
	if strings.HasPrefix(trimmed, "///") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Regular comments -- skip.
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Associated type: type Error; or type Item = u32;
	if strings.HasPrefix(trimmed, "type ") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Method signature or default method.
	if isRustFnDecl(trimmed) {
		sigText := extractRustFnSignature(line)
		pc.blockMethods = append(pc.blockMethods, sigText)
		// If the method has a default body, we need to skip it.
		// The braces are already counted above.
		return
	}
}

// ---------------------------------------------------------------------------
// Impl body handler (extracting method signatures)
// ---------------------------------------------------------------------------

// handleImplBody processes lines inside an impl block.
func (p *rustParser) handleImplBody(pc *rustParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += rustCountBraces(trimmed)

	// Impl block closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
		return
	}

	// We only extract members at depth 1 (the impl body level).
	if prevDepth != 1 {
		return
	}

	// Skip empty lines.
	if trimmed == "" {
		return
	}

	// Doc comments inside impl body.
	if strings.HasPrefix(trimmed, "///") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Regular comments -- skip.
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Attributes inside impl body.
	if strings.HasPrefix(trimmed, "#[") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// const/type items in impl block.
	if isRustConstDecl(trimmed) || strings.HasPrefix(trimmed, "type ") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Method signature.
	if isRustFnDecl(trimmed) {
		sigText := extractRustFnSignature(line)
		pc.blockMethods = append(pc.blockMethods, sigText)
		return
	}
}

// ---------------------------------------------------------------------------
// Extern block handler
// ---------------------------------------------------------------------------

// handleExternBlock processes lines inside an extern "C" block.
func (p *rustParser) handleExternBlock(pc *rustParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += rustCountBraces(trimmed)

	// Block closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
		return
	}

	// Extract items at depth 1.
	if prevDepth != 1 {
		return
	}

	if trimmed == "" || strings.HasPrefix(trimmed, "//") {
		return
	}

	if isRustFnDecl(trimmed) {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Other items (static, type).
	if isRustStaticDecl(trimmed) || strings.HasPrefix(trimmed, "type ") {
		pc.blockMethods = append(pc.blockMethods, line)
	}
}

// ---------------------------------------------------------------------------
// Macro body handler (skipping)
// ---------------------------------------------------------------------------

// handleMacroBody skips macro_rules! body by counting braces.
func (p *rustParser) handleMacroBody(pc *rustParseCtx, trimmed string, lineNum int) {
	pc.braceDepth += rustCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = rustStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Declaration handlers
// ---------------------------------------------------------------------------

// handleStructDecl processes a struct declaration.
func (p *rustParser) handleStructDecl(pc *rustParseCtx, trimmed, line string, lineNum int) {
	// Unit struct: pub struct Marker;
	if strings.HasSuffix(trimmed, ";") {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractRustStructName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Tuple struct on one line: pub struct Wrapper(pub T);
	// or struct with body on one line.
	braces := rustCountBraces(trimmed)
	if strings.Contains(trimmed, "{") && braces == 0 {
		// Single-line struct with body.
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractRustStructName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Check for tuple struct with parens closing on same line: pub struct Wrapper(pub T);
	if strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")") &&
		strings.HasSuffix(trimmed, ";") {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindStruct,
			Name:      extractRustStructName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Multi-line struct.
	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = rustStateInStructBody
		return
	}

	// Tuple struct that might continue to next line -- emit what we see.
	doc := buildRustDoc(pc.docLines, pc.attrLines)
	sig := Signature{
		Kind:      KindStruct,
		Name:      extractRustStructName(trimmed),
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docLines = nil
	pc.attrLines = nil
}

// handleEnumDecl processes an enum declaration.
func (p *rustParser) handleEnumDecl(pc *rustParseCtx, trimmed, line string, lineNum int) {
	braces := rustCountBraces(trimmed)

	// Single-line enum.
	if strings.Contains(trimmed, "{") && braces == 0 {
		doc := buildRustDoc(pc.docLines, pc.attrLines)
		sig := Signature{
			Kind:      KindType,
			Name:      extractRustEnumName(trimmed),
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docLines = nil
		pc.attrLines = nil
		return
	}

	// Multi-line enum.
	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = rustStateInEnumBody
		return
	}

	// Enum without body on this line (unlikely but handle gracefully).
	doc := buildRustDoc(pc.docLines, pc.attrLines)
	sig := Signature{
		Kind:      KindType,
		Name:      extractRustEnumName(trimmed),
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.docLines = nil
	pc.attrLines = nil
}

// handleTraitDecl processes a trait declaration.
func (p *rustParser) handleTraitDecl(pc *rustParseCtx, trimmed, line string, lineNum int) {
	braces := rustCountBraces(trimmed)

	pc.blockHeader = line
	pc.blockName = extractRustTraitName(trimmed)
	pc.blockStartLine = lineNum
	pc.blockDocComment = buildRustDoc(pc.docLines, pc.attrLines)
	pc.blockAttrs = nil
	pc.blockMethods = nil
	pc.docLines = nil
	pc.attrLines = nil

	if braces > 0 {
		pc.braceDepth = braces
		pc.state = rustStateInTraitBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty trait on one line: trait Foo {}
		p.emitBlockSignature(pc, lineNum)
	}
}

// handleImplDecl processes an impl block declaration.
func (p *rustParser) handleImplDecl(pc *rustParseCtx, trimmed, line string, lineNum int) {
	braces := rustCountBraces(trimmed)

	pc.blockHeader = line
	pc.blockName = extractRustImplName(trimmed)
	pc.blockStartLine = lineNum
	pc.blockDocComment = buildRustDoc(pc.docLines, pc.attrLines)
	pc.blockAttrs = nil
	pc.blockMethods = nil
	pc.docLines = nil
	pc.attrLines = nil

	if braces > 0 {
		pc.braceDepth = braces
		pc.state = rustStateInImplBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty impl on one line: impl Foo {}
		p.emitBlockSignature(pc, lineNum)
	}
}

// handleFnDecl processes a standalone function declaration.
func (p *rustParser) handleFnDecl(pc *rustParseCtx, trimmed, line string, lineNum int) {
	doc := buildRustDoc(pc.docLines, pc.attrLines)
	pc.docLines = nil
	pc.attrLines = nil

	sigText := extractRustFnSignature(line)
	name := extractRustFnName(trimmed)

	sig := Signature{
		Kind:      KindFunction,
		Name:      name,
		Source:    maybePrependDoc(doc, nil, sigText),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)

	// Count braces to determine if we need to skip a body.
	braces := rustCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = rustStateInFnBody
	}
	// If braces == 0, either it's a single-line body or a declaration without body (trait method).
}

// ---------------------------------------------------------------------------
// Block signature emission (trait, impl, extern)
// ---------------------------------------------------------------------------

// emitBlockSignature builds and emits a signature for a trait/impl/extern block.
func (p *rustParser) emitBlockSignature(pc *rustParseCtx, endLine int) {
	var b strings.Builder
	headerTrimmed := strings.TrimRight(pc.blockHeader, " \t\n\r")

	// Ensure the header ends with {.
	if !strings.HasSuffix(headerTrimmed, "{") {
		headerTrimmed += " {"
	}
	b.WriteString(headerTrimmed)

	for _, m := range pc.blockMethods {
		b.WriteString("\n")
		b.WriteString(m)
	}
	b.WriteString("\n}")

	kind := KindClass // impl blocks
	name := pc.blockName
	if isRustTraitDeclLine(pc.blockHeader) {
		kind = KindInterface
	} else if isRustExternBlockLine(pc.blockHeader) {
		kind = KindType
	}

	sig := Signature{
		Kind:      kind,
		Name:      name,
		Source:    maybePrependDoc(pc.blockDocComment, nil, b.String()),
		StartLine: pc.blockStartLine,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)

	pc.blockHeader = ""
	pc.blockName = ""
	pc.blockMethods = nil
	pc.blockDocComment = ""
	pc.blockAttrs = nil
}

// ---------------------------------------------------------------------------
// Rust-specific detection helpers
// ---------------------------------------------------------------------------

// isRustFnDecl checks if a trimmed line starts a function declaration.
func isRustFnDecl(trimmed string) bool {
	// Strip visibility and qualifiers.
	s := stripRustFnPrefix(trimmed)
	return strings.HasPrefix(s, "fn ")
}

// stripRustFnPrefix removes visibility and qualifier prefixes from a line.
func stripRustFnPrefix(trimmed string) string {
	s := trimmed
	// Visibility prefixes.
	for _, vis := range []string{
		"pub(crate) ", "pub(super) ", "pub(in ", "pub ",
	} {
		if strings.HasPrefix(s, vis) {
			s = strings.TrimPrefix(s, vis)
			// Handle pub(in path) -- skip to closing paren.
			if vis == "pub(in " {
				idx := strings.Index(s, ") ")
				if idx >= 0 {
					s = s[idx+2:]
				}
			}
			break
		}
	}
	// Qualifier prefixes -- strip repeatedly since multiple qualifiers can appear.
	qualifiers := []string{
		"default ", "unsafe ", "async ", "const ", "extern \"C\" ",
		"extern \"system\" ", "extern \"cdecl\" ",
	}
	changed := true
	for changed {
		changed = false
		for _, qual := range qualifiers {
			if strings.HasPrefix(s, qual) {
				s = strings.TrimPrefix(s, qual)
				changed = true
				break
			}
		}
	}
	return s
}

// isRustStructDecl checks if a trimmed line declares a struct.
func isRustStructDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "struct ")
}

// isRustEnumDecl checks if a trimmed line declares an enum.
func isRustEnumDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "enum ")
}

// isRustTraitDecl checks if a trimmed line declares a trait.
func isRustTraitDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "unsafe ")
	return strings.HasPrefix(s, "trait ")
}

// isRustTraitDeclLine checks if a full line is a trait declaration header.
func isRustTraitDeclLine(line string) bool {
	return isRustTraitDecl(strings.TrimSpace(line))
}

// isRustExternBlockLine checks if a full line is an extern block header.
func isRustExternBlockLine(line string) bool {
	return isRustExternBlock(strings.TrimSpace(line))
}

// isRustImplDecl checks if a trimmed line declares an impl block.
func isRustImplDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "unsafe ")
	return strings.HasPrefix(s, "impl ") || strings.HasPrefix(s, "impl<")
}

// isRustTypeAlias checks if a trimmed line is a type alias.
func isRustTypeAlias(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "type ") && strings.Contains(s, "=")
}

// isRustConstDecl checks if a trimmed line is a const declaration.
func isRustConstDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "const ") && !strings.HasPrefix(s, "const fn ")
}

// isRustStaticDecl checks if a trimmed line is a static declaration.
func isRustStaticDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "static ") || strings.HasPrefix(s, "static mut ")
}

// isRustModDecl checks if a trimmed line is a mod declaration.
func isRustModDecl(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	return strings.HasPrefix(s, "mod ")
}

// isRustMacroRules checks if a trimmed line starts a macro_rules! declaration.
func isRustMacroRules(trimmed string) bool {
	return strings.HasPrefix(trimmed, "macro_rules!")
}

// isRustExternBlock checks if a trimmed line starts an extern block (not an extern fn).
func isRustExternBlock(trimmed string) bool {
	s := stripRustVisibility(trimmed)
	// extern "C" { or extern { or extern "system" {
	if !strings.HasPrefix(s, "extern") {
		return false
	}
	rest := strings.TrimPrefix(s, "extern")
	rest = strings.TrimSpace(rest)

	// extern { ... }
	if strings.HasPrefix(rest, "{") {
		return true
	}

	// extern "C" { ... } or extern "system" { ... }
	// Must not be extern "C" fn (that is a function, not a block).
	if strings.HasPrefix(rest, "\"") {
		// Find closing quote.
		closeQuote := strings.Index(rest[1:], "\"")
		if closeQuote == -1 {
			return false
		}
		afterABI := strings.TrimSpace(rest[closeQuote+2:])
		// An extern block has { right after the ABI string.
		// An extern fn has "fn" after the ABI string.
		return strings.HasPrefix(afterABI, "{")
	}
	return false
}

// stripRustVisibility removes pub/pub(crate)/pub(super) prefix.
func stripRustVisibility(trimmed string) string {
	s := trimmed
	for _, vis := range []string{
		"pub(crate) ", "pub(super) ", "pub(in ",
	} {
		if strings.HasPrefix(s, vis) {
			s = strings.TrimPrefix(s, vis)
			if vis == "pub(in " {
				idx := strings.Index(s, ") ")
				if idx >= 0 {
					s = s[idx+2:]
				}
			}
			return s
		}
	}
	s = strings.TrimPrefix(s, "pub ")
	return s
}

// ---------------------------------------------------------------------------
// Rust-specific name extraction
// ---------------------------------------------------------------------------

// extractRustFnName extracts the function name from a function declaration line.
func extractRustFnName(trimmed string) string {
	s := stripRustFnPrefix(trimmed)
	s = strings.TrimPrefix(s, "fn ")
	return extractRustIdentifier(s)
}

// extractRustFnSignature extracts a function signature up to (not including) the
// opening brace of the function body. Preserves the original indentation.
func extractRustFnSignature(line string) string {
	trimmed := strings.TrimSpace(line)

	// If no brace at all, return the whole line trimmed of trailing whitespace.
	if !strings.Contains(trimmed, "{") {
		return strings.TrimRight(line, " \t")
	}

	// Find the opening brace that starts the body (not inside generics or strings).
	bodyIdx := findRustBodyBrace(line)
	if bodyIdx == -1 {
		return strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(line[:bodyIdx], " \t")
}

// findRustBodyBrace finds the index of the `{` that opens the function body.
// After all parentheses and angle brackets are balanced, the next `{` at
// brace depth 0 is the body opener.
func findRustBodyBrace(line string) int {
	parenDepth := 0
	angleDepth := 0
	inDoubleQuote := false
	inChar := false
	escaped := false
	seenParens := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
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

		if inChar {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inChar = false
			}
			continue
		}

		// Line comment.
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			// Could be a lifetime or a char literal. Peek ahead.
			if i+2 < len(line) && line[i+2] == '\'' {
				// Likely a char literal like 'a'.
				inChar = true
			} else if i+3 < len(line) && line[i+1] == '\\' && line[i+3] == '\'' {
				// Escaped char literal like '\n'.
				inChar = true
			}
			// Otherwise it is a lifetime annotation -- not a string context.
		case '(':
			parenDepth++
			seenParens = true
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '{':
			if seenParens && parenDepth == 0 && angleDepth == 0 {
				return i
			}
		}
	}
	return -1
}

// extractRustStructName extracts the struct name from a declaration line.
func extractRustStructName(trimmed string) string {
	s := stripRustVisibility(strings.TrimSpace(trimmed))
	s = strings.TrimPrefix(s, "struct ")
	return extractRustIdentifier(s)
}

// extractRustEnumName extracts the enum name from a declaration line.
func extractRustEnumName(trimmed string) string {
	s := stripRustVisibility(strings.TrimSpace(trimmed))
	s = strings.TrimPrefix(s, "enum ")
	return extractRustIdentifier(s)
}

// extractRustTraitName extracts the trait name from a declaration line.
func extractRustTraitName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "unsafe ")
	s = strings.TrimPrefix(s, "trait ")
	return extractRustIdentifier(s)
}

// extractRustImplName extracts the impl target from an impl declaration.
// For "impl Trait for Type", returns "Trait for Type".
// For "impl Type", returns "Type".
func extractRustImplName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "unsafe ")
	s = strings.TrimPrefix(s, "impl")
	s = strings.TrimSpace(s)

	// Skip generic params: impl<T: Display> ...
	s = skipRustGenericParams(s)

	// Extract up to the opening brace.
	if idx := strings.Index(s, "{"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	return s
}

// skipRustGenericParams skips a leading <...> generic parameter list from a string.
// Returns the string unchanged if it does not start with '<'.
func skipRustGenericParams(s string) string {
	if !strings.HasPrefix(s, "<") {
		return s
	}
	depth := 0
	for i, ch := range s {
		switch ch {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[i+1:])
			}
		}
	}
	// Unbalanced -- return as-is.
	return s
}

// extractRustTypeAliasName extracts the type alias name.
func extractRustTypeAliasName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "type ")
	return extractRustIdentifier(s)
}

// extractRustConstName extracts the const item name.
func extractRustConstName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "const ")
	return extractRustIdentifier(s)
}

// extractRustStaticName extracts the static item name.
func extractRustStaticName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "static mut ")
	s = strings.TrimPrefix(s, "static ")
	return extractRustIdentifier(s)
}

// extractRustModName extracts the module name from a mod declaration.
func extractRustModName(trimmed string) string {
	s := stripRustVisibility(trimmed)
	s = strings.TrimPrefix(s, "mod ")
	return extractRustIdentifier(s)
}

// extractRustMacroName extracts the macro name from a macro_rules! declaration.
func extractRustMacroName(trimmed string) string {
	s := strings.TrimPrefix(trimmed, "macro_rules!")
	s = strings.TrimSpace(s)
	return extractRustIdentifier(s)
}

// extractRustIdentifier extracts the first Rust identifier from a string.
func extractRustIdentifier(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for i, ch := range s {
		if i == 0 {
			if ch != '_' && !isRustLetter(ch) {
				break
			}
		} else {
			if ch != '_' && !isRustLetter(ch) && !isRustDigit(ch) {
				break
			}
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// isRustLetter checks if a rune is a letter for Rust identifiers.
func isRustLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// isRustDigit checks if a rune is a digit for Rust identifiers.
func isRustDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// ---------------------------------------------------------------------------
// Rust doc/attr helpers
// ---------------------------------------------------------------------------

// buildRustDoc combines accumulated doc comment lines and attribute lines
// into a single doc string. Returns empty string if there are none.
func buildRustDoc(docLines, attrLines []string) string {
	if len(docLines) == 0 && len(attrLines) == 0 {
		return ""
	}
	var parts []string
	parts = append(parts, docLines...)
	parts = append(parts, attrLines...)
	return strings.Join(parts, "\n")
}

// ---------------------------------------------------------------------------
// Rust brace counting
// ---------------------------------------------------------------------------

// rustCountBraces counts the net brace depth for a Rust source line,
// ignoring braces inside string literals, raw strings, char literals,
// and comments.
func rustCountBraces(line string) int {
	depth := 0
	inDoubleQuote := false
	inRawString := false
	rawHashCount := 0
	inChar := false
	escaped := false
	inLineComment := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			continue
		}

		if inLineComment {
			break
		}

		// Inside a raw string: r#"..."#
		if inRawString {
			if ch == '"' {
				// Count trailing # characters.
				hashes := 0
				for j := i + 1; j < len(line) && line[j] == '#'; j++ {
					hashes++
				}
				if hashes >= rawHashCount {
					inRawString = false
					i += rawHashCount // skip the closing hashes
				}
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

		// Inside a char literal.
		if inChar {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inChar = false
			}
			continue
		}

		// Check for line comment.
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			inLineComment = true
			break
		}

		// Check for block comment on same line.
		if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "*/")
			if end >= 0 {
				i = i + 2 + end + 1 // skip past */
				continue
			}
			// Block comment doesn't close on this line -- ignore rest.
			break
		}

		// Check for raw string: r"...", r#"..."#, r##"..."##, etc.
		if ch == 'r' && i+1 < len(line) {
			hashes := 0
			j := i + 1
			for j < len(line) && line[j] == '#' {
				hashes++
				j++
			}
			if j < len(line) && line[j] == '"' {
				inRawString = true
				rawHashCount = hashes
				i = j // skip to the opening quote
				continue
			}
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			// Distinguish char literal from lifetime annotation.
			// A char literal is 'x' or '\x' pattern.
			if i+2 < len(line) && line[i+2] == '\'' && line[i+1] != '\\' {
				inChar = true
			} else if i+3 < len(line) && line[i+1] == '\\' && line[i+3] == '\'' {
				inChar = true
			}
			// Otherwise, it is a lifetime annotation -- not a string context.
		case '{':
			depth++
		case '}':
			depth--
		}
	}

	return depth
}