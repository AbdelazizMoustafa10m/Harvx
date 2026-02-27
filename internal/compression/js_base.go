package compression

import (
	"context"
	"strings"
	"unicode"
)

// jsParserConfig controls which language features are extracted.
type jsParserConfig struct {
	extractInterfaces      bool   // TypeScript only
	extractTypeAliases     bool   // TypeScript only
	extractEnums           bool   // TypeScript only
	extractTypeAnnotations bool   // TypeScript only
	language               string // "typescript" or "javascript"
}

// parserState tracks the current state of the line-by-line state machine.
type parserState int

const (
	stateTopLevel      parserState = iota // Scanning for declarations at brace depth 0
	stateInDocComment                     // Accumulating /** ... */ block
	stateInBlockSkip                      // Skipping function/method body (counting braces)
	stateInClassBody                      // Extracting class members at depth 1
	stateInInterfaceBody                  // Extracting interface body (TS only)
	stateInEnumBody                       // Extracting enum body (TS only)
	stateInImport                         // Accumulating multi-line import
	stateInExport                         // Accumulating multi-line export
	stateInTypeAlias                      // Accumulating multi-line type alias
)

// parseCtx holds all mutable state for the line-by-line parser.
// Passed by pointer to avoid copying strings.Builder values.
type parseCtx struct {
	state           parserState
	braceDepth      int
	skipTargetDepth int
	accum           strings.Builder
	accumStartLine  int
	docComment      string
	decorators      []string
	sigs            []Signature
	classHeader     strings.Builder
	classFields     []string
	classMethods    []string
	classStartLine  int
	classDocComment string   // doc comment saved for the class declaration
	classDecorators []string // decorators saved for the class declaration
}

// jsParser provides shared JS/TS parsing using a line-by-line state machine.
type jsParser struct {
	config jsParserConfig
}

// newJSParser creates a new jsParser with the given configuration.
func newJSParser(config jsParserConfig) *jsParser {
	return &jsParser{config: config}
}

// parse extracts structural signatures from JS/TS source code.
func (p *jsParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     p.config.language,
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &parseCtx{
		state: stateTopLevel,
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
		case stateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case stateInDocComment:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			if strings.Contains(trimmed, "*/") {
				pc.docComment = pc.accum.String()
				pc.accum.Reset()
				pc.state = stateTopLevel
			}

		case stateInBlockSkip:
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth {
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			}

		case stateInClassBody:
			prevDepth := pc.braceDepth
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth {
				// Class closed.
				sig := p.buildClassSignature(pc.classHeader.String(), pc.classFields,
					pc.classMethods, pc.classStartLine, lineNum, pc.classDocComment, pc.classDecorators)
				pc.sigs = append(pc.sigs, sig)
				pc.classDocComment = ""
				pc.classDecorators = nil
				pc.docComment = ""
				pc.decorators = nil
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			} else if prevDepth == pc.skipTargetDepth+1 {
				// We were at class body level before counting braces on this line.
				p.handleClassMember(trimmed, line, &pc.classFields, &pc.classMethods,
					&pc.docComment)
			}
			// If prevDepth > skipTargetDepth+1, we are inside a method body -- skip.

		case stateInInterfaceBody:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth {
				sig := Signature{
					Kind:      KindInterface,
					Name:      extractInterfaceName(pc.accum.String()),
					Source:    maybePrependDoc(pc.docComment, pc.decorators, pc.accum.String()),
					StartLine: pc.accumStartLine,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
				pc.accum.Reset()
				pc.docComment = ""
				pc.decorators = nil
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			}

		case stateInEnumBody:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth {
				sig := Signature{
					Kind:      KindType,
					Name:      extractEnumName(pc.accum.String()),
					Source:    maybePrependDoc(pc.docComment, pc.decorators, pc.accum.String()),
					StartLine: pc.accumStartLine,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
				pc.accum.Reset()
				pc.docComment = ""
				pc.decorators = nil
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			}

		case stateInImport:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			if isImportComplete(pc.accum.String()) {
				sig := Signature{
					Kind:      KindImport,
					Source:    pc.accum.String(),
					StartLine: pc.accumStartLine,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
				pc.accum.Reset()
				pc.state = stateTopLevel
			}

		case stateInExport:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth && isExportComplete(pc.accum.String()) {
				sig := Signature{
					Kind:      KindExport,
					Source:    pc.accum.String(),
					StartLine: pc.accumStartLine,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
				pc.accum.Reset()
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			}

		case stateInTypeAlias:
			pc.accum.WriteString("\n")
			pc.accum.WriteString(line)
			pc.braceDepth += countBraces(trimmed)
			if pc.braceDepth <= pc.skipTargetDepth && isTypeAliasComplete(pc.accum.String()) {
				sig := Signature{
					Kind:      KindType,
					Name:      extractTypeAliasName(pc.accum.String()),
					Source:    maybePrependDoc(pc.docComment, pc.decorators, pc.accum.String()),
					StartLine: pc.accumStartLine,
					EndLine:   lineNum,
				}
				pc.sigs = append(pc.sigs, sig)
				pc.accum.Reset()
				pc.docComment = ""
				pc.decorators = nil
				pc.state = stateTopLevel
				pc.braceDepth = pc.skipTargetDepth
			}
		}
	}

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     p.config.language,
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// handleTopLevel processes a line when the parser is at the top level.
func (p *jsParser) handleTopLevel(pc *parseCtx, trimmed, line string, lineNum int) {
	// Skip empty lines and single-line comments.
	if trimmed == "" || strings.HasPrefix(trimmed, "//") {
		return
	}

	// Doc comment start: /** ... */
	if strings.HasPrefix(trimmed, "/**") {
		if strings.Contains(trimmed, "*/") {
			// Single-line doc comment.
			pc.docComment = line
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = stateInDocComment
		return
	}

	// Block comment (non-doc) -- skip.
	if strings.HasPrefix(trimmed, "/*") {
		if strings.Contains(trimmed[2:], "*/") {
			return
		}
		// Multi-line non-doc comment -- use doc comment state but discard.
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.docComment = ""
		pc.state = stateInDocComment
		return
	}

	// Decorators: @Something or @Something(...)
	if strings.HasPrefix(trimmed, "@") {
		pc.decorators = append(pc.decorators, line)
		return
	}

	// Import statement.
	if isImportLine(trimmed) {
		if isImportComplete(line) {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindImport,
				Source:    line,
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = stateInImport
		return
	}

	// Pure export/re-export statements (not export of a declaration).
	if isExportStatement(trimmed) {
		pc.braceDepth += countBraces(trimmed)
		if isExportComplete(line) && pc.braceDepth == 0 {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindExport,
				Source:    line,
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInExport
		return
	}

	// Interface declaration (TS only).
	if p.config.extractInterfaces && isInterfaceDeclaration(trimmed) {
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth == 0 {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindInterface,
				Name:      extractInterfaceName(line),
				Source:    maybePrependDoc(pc.docComment, pc.decorators, line),
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			pc.docComment = ""
			pc.decorators = nil
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInInterfaceBody
		return
	}

	// Enum declaration (TS only).
	if p.config.extractEnums && isEnumDeclaration(trimmed) {
		pc.braceDepth += countBraces(trimmed)
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInEnumBody
		return
	}

	// Type alias declaration (TS only).
	if p.config.extractTypeAliases && isTypeAliasDeclaration(trimmed) {
		pc.braceDepth += countBraces(trimmed)
		if isTypeAliasComplete(line) && pc.braceDepth == 0 {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindType,
				Name:      extractTypeAliasName(line),
				Source:    maybePrependDoc(pc.docComment, pc.decorators, line),
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			pc.docComment = ""
			pc.decorators = nil
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInTypeAlias
		return
	}

	// Class declaration (including export default class, abstract class).
	if isClassDeclaration(trimmed) {
		pc.classHeader.Reset()
		pc.classHeader.WriteString(line)
		pc.classFields = nil
		pc.classMethods = nil
		pc.classStartLine = lineNum
		// Save and clear the doc comment/decorators for the class itself.
		pc.classDocComment = pc.docComment
		pc.classDecorators = pc.decorators
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInClassBody
		}
		return
	}

	// Export with declaration: export function, export class, export default function, etc.
	if isExportDeclaration(trimmed) {
		p.handleExportDeclaration(pc, trimmed, line, lineNum)
		return
	}

	// Function declaration.
	if isFunctionDeclaration(trimmed) {
		sig := p.extractFunctionSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// Arrow function assigned to const/let/var.
	if isArrowFunctionDeclaration(trimmed) {
		sig := p.extractArrowFunctionSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// Top-level const/let/var declarations.
	if isTopLevelVarDeclaration(trimmed) {
		sig := p.extractConstSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		if sig != nil {
			pc.sigs = append(pc.sigs, *sig)
			pc.docComment = ""
			pc.decorators = nil
		}
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// Anything else at top level with unmatched braces -- track them.
	pc.braceDepth += countBraces(trimmed)
	if pc.braceDepth > 0 {
		pc.skipTargetDepth = 0
		pc.state = stateInBlockSkip
		return
	}

	// Reset doc comment if we encounter a non-declaration without consuming it.
	if pc.docComment != "" && !strings.HasPrefix(trimmed, "@") {
		pc.docComment = ""
		pc.decorators = nil
	}
}

// handleExportDeclaration processes exported declarations (export function, export class, etc.).
func (p *jsParser) handleExportDeclaration(pc *parseCtx, trimmed, line string, lineNum int) {
	inner := stripExportPrefix(trimmed)

	// export class / export default class / export abstract class
	if isClassDeclaration(inner) || (strings.HasPrefix(inner, "abstract ") && isClassDeclaration(strings.TrimPrefix(inner, "abstract "))) {
		pc.classHeader.Reset()
		pc.classHeader.WriteString(line)
		pc.classFields = nil
		pc.classMethods = nil
		pc.classStartLine = lineNum
		// Save and clear the doc comment/decorators for the class itself.
		pc.classDocComment = pc.docComment
		pc.classDecorators = pc.decorators
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInClassBody
		}
		return
	}

	// export function / export default function / export async function
	if isFunctionDeclaration(inner) {
		sig := p.extractFunctionSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// export const with arrow function
	if isArrowFunctionDeclaration(inner) {
		sig := p.extractArrowFunctionSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		pc.decorators = nil
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// export interface (TS only)
	if p.config.extractInterfaces && isInterfaceDeclaration(inner) {
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth == 0 {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindInterface,
				Name:      extractInterfaceName(line),
				Source:    maybePrependDoc(pc.docComment, pc.decorators, line),
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			pc.docComment = ""
			pc.decorators = nil
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInInterfaceBody
		return
	}

	// export enum (TS only)
	if p.config.extractEnums && isEnumDeclaration(inner) {
		pc.braceDepth += countBraces(trimmed)
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInEnumBody
		return
	}

	// export type alias (TS only)
	if p.config.extractTypeAliases && isTypeAliasDeclaration(inner) {
		pc.braceDepth += countBraces(trimmed)
		if isTypeAliasComplete(line) && pc.braceDepth == 0 {
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindType,
				Name:      extractTypeAliasName(line),
				Source:    maybePrependDoc(pc.docComment, pc.decorators, line),
				StartLine: lineNum,
				EndLine:   lineNum,
			})
			pc.docComment = ""
			pc.decorators = nil
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.skipTargetDepth = 0
		pc.state = stateInTypeAlias
		return
	}

	// export const/let/var (not arrow function)
	if isTopLevelVarDeclaration(inner) {
		sig := p.extractConstSignature(line, trimmed, lineNum, pc.docComment, pc.decorators)
		if sig != nil {
			pc.sigs = append(pc.sigs, *sig)
			pc.docComment = ""
			pc.decorators = nil
		}
		pc.braceDepth += countBraces(trimmed)
		if pc.braceDepth > 0 {
			pc.skipTargetDepth = 0
			pc.state = stateInBlockSkip
		}
		return
	}

	// Fallback: treat as generic export.
	pc.sigs = append(pc.sigs, Signature{
		Kind:      KindExport,
		Source:    line,
		StartLine: lineNum,
		EndLine:   lineNum,
	})
	pc.braceDepth += countBraces(trimmed)
	if pc.braceDepth > 0 {
		pc.skipTargetDepth = 0
		pc.state = stateInBlockSkip
	}
}

// handleClassMember processes a line within a class body at the class-body
// brace depth. It extracts field declarations and method signatures.
func (p *jsParser) handleClassMember(
	trimmed, line string,
	classFields *[]string, classMethods *[]string,
	docComment *string,
) {
	if trimmed == "" || trimmed == "}" {
		return
	}

	// Skip single-line comments inside class body.
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Doc comments inside class body -- keep for next member.
	if strings.HasPrefix(trimmed, "/**") {
		if strings.Contains(trimmed, "*/") {
			*docComment = line
		}
		return
	}

	// Field declarations: visibility? readonly? name: Type;
	if isFieldDeclaration(trimmed) {
		fieldLine := line
		if *docComment != "" {
			fieldLine = *docComment + "\n" + line
			*docComment = ""
		}
		*classFields = append(*classFields, fieldLine)
		return
	}

	// Method declarations: visibility? async? name(params): RetType {
	if isMethodDeclaration(trimmed) {
		methodSig := extractMethodSignature(line, trimmed)
		if *docComment != "" {
			methodSig = *docComment + "\n" + methodSig
			*docComment = ""
		}
		*classMethods = append(*classMethods, methodSig)
		// If the method body opens on this line, skip it.
		if strings.Contains(trimmed, "{") {
			// The opening brace is already counted in the caller.
			// We need to enter block-skip to skip the method body,
			// but remain in class body state once the method closes.
			// We handle this by just tracking braces -- the class body handler
			// already ignores lines at depth > 1.
		}
		return
	}

	// Constructor, getter, setter, static methods, abstract methods.
	if isConstructorOrAccessor(trimmed) {
		methodSig := extractMethodSignature(line, trimmed)
		if *docComment != "" {
			methodSig = *docComment + "\n" + methodSig
			*docComment = ""
		}
		*classMethods = append(*classMethods, methodSig)
		return
	}

	// Decorator inside class.
	if strings.HasPrefix(trimmed, "@") {
		*classFields = append(*classFields, line)
		return
	}
}

// buildClassSignature constructs a class Signature from accumulated parts.
func (p *jsParser) buildClassSignature(
	header string, fields, methods []string,
	startLine, endLine int, docComment string, decorators []string,
) Signature {
	var b strings.Builder
	// Write the class header (e.g., "class Foo extends Bar {").
	headerTrimmed := strings.TrimRight(header, " \t\n\r")
	if !strings.HasSuffix(headerTrimmed, "{") {
		headerTrimmed += " {"
	}
	b.WriteString(headerTrimmed)

	for _, f := range fields {
		b.WriteString("\n")
		b.WriteString(f)
	}
	for _, m := range methods {
		b.WriteString("\n\n")
		b.WriteString(m)
	}
	b.WriteString("\n}")

	name := extractClassName(header)

	return Signature{
		Kind:      KindClass,
		Name:      name,
		Source:    maybePrependDoc(docComment, decorators, b.String()),
		StartLine: startLine,
		EndLine:   endLine,
	}
}

// extractFunctionSignature extracts a function declaration signature.
func (p *jsParser) extractFunctionSignature(line, trimmed string, lineNum int, docComment string, decorators []string) Signature {
	sig := extractSignatureBeforeBrace(line)
	name := extractFunctionName(trimmed)

	return Signature{
		Kind:      KindFunction,
		Name:      name,
		Source:    maybePrependDoc(docComment, decorators, sig),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
}

// extractArrowFunctionSignature extracts an arrow function assigned to a variable.
func (p *jsParser) extractArrowFunctionSignature(line, trimmed string, lineNum int, docComment string, decorators []string) Signature {
	sig := extractArrowSignature(line)
	name := extractVarName(trimmed)

	return Signature{
		Kind:      KindFunction,
		Name:      name,
		Source:    maybePrependDoc(docComment, decorators, sig),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
}

// extractConstSignature extracts a top-level const/let/var declaration.
func (p *jsParser) extractConstSignature(line, trimmed string, lineNum int, docComment string, decorators []string) *Signature {
	name := extractVarName(trimmed)
	if name == "" {
		return nil
	}
	constSig := extractConstDeclaration(line, trimmed)

	return &Signature{
		Kind:      KindConstant,
		Name:      name,
		Source:    maybePrependDoc(docComment, decorators, constSig),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
}

// ---------------------------------------------------------------------------
// Detection helpers
// ---------------------------------------------------------------------------

// isImportLine checks if a trimmed line starts an import statement.
func isImportLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "import ")
}

// isImportComplete checks if the accumulated import text forms a complete statement.
func isImportComplete(text string) bool {
	t := strings.TrimSpace(text)
	// An import is complete when it ends with a quote + optional semicolon,
	// or ends with a semicolon.
	return strings.HasSuffix(t, ";") ||
		strings.HasSuffix(t, "'") ||
		strings.HasSuffix(t, "\"")
}

// isExportStatement checks if a trimmed line is a pure export/re-export (not an export declaration).
func isExportStatement(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "export ") {
		return false
	}
	rest := strings.TrimPrefix(trimmed, "export ")

	// Re-exports: export * from, export { ... } from
	if strings.HasPrefix(rest, "* ") || strings.HasPrefix(rest, "{ ") || strings.HasPrefix(rest, "{") {
		return true
	}
	// export type { ... } from (TS re-export)
	if strings.HasPrefix(rest, "type {") || strings.HasPrefix(rest, "type { ") {
		return true
	}
	// export default <identifier>; (not a declaration keyword)
	if strings.HasPrefix(rest, "default ") {
		afterDefault := strings.TrimPrefix(rest, "default ")
		// If the next token is not a declaration keyword, it's a value export.
		for _, kw := range []string{
			"function", "function*", "function(",
			"class ", "abstract ", "async ",
			"const ", "let ", "var ",
			"interface ", "enum ", "type ",
		} {
			if strings.HasPrefix(afterDefault, kw) {
				return false
			}
		}
		return true
	}
	return false
}

// isExportDeclaration checks if a trimmed line is an export followed by a declaration.
func isExportDeclaration(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "export ") {
		return false
	}
	inner := stripExportPrefix(trimmed)
	return isFunctionDeclaration(inner) ||
		isClassDeclaration(inner) ||
		isArrowFunctionDeclaration(inner) ||
		isInterfaceDeclaration(inner) ||
		isEnumDeclaration(inner) ||
		isTypeAliasDeclaration(inner) ||
		isTopLevelVarDeclaration(inner) ||
		strings.HasPrefix(inner, "abstract ")
}

// isExportComplete checks if accumulated export text is complete.
func isExportComplete(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasSuffix(t, ";") ||
		strings.HasSuffix(t, "'") ||
		strings.HasSuffix(t, "\"")
}

// stripExportPrefix removes export/export default prefix.
func stripExportPrefix(trimmed string) string {
	s := strings.TrimPrefix(trimmed, "export ")
	s = strings.TrimPrefix(s, "default ")
	return s
}

// isInterfaceDeclaration checks for interface keyword.
func isInterfaceDeclaration(trimmed string) bool {
	return strings.HasPrefix(trimmed, "interface ") ||
		strings.HasPrefix(trimmed, "declare interface ")
}

// isEnumDeclaration checks for enum keyword.
func isEnumDeclaration(trimmed string) bool {
	return strings.HasPrefix(trimmed, "enum ") ||
		strings.HasPrefix(trimmed, "const enum ") ||
		strings.HasPrefix(trimmed, "declare enum ") ||
		strings.HasPrefix(trimmed, "declare const enum ")
}

// isTypeAliasDeclaration checks for type keyword.
func isTypeAliasDeclaration(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "type ") && !strings.HasPrefix(trimmed, "declare type ") {
		return false
	}
	// Exclude "type { ... } from" which is a re-export.
	rest := strings.TrimPrefix(trimmed, "declare ")
	rest = strings.TrimPrefix(rest, "type ")
	return !strings.HasPrefix(rest, "{")
}

// isTypeAliasComplete checks if accumulated type alias text is complete.
func isTypeAliasComplete(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasSuffix(t, ";") ||
		// Type aliases may end without semicolons in some styles.
		(!strings.HasSuffix(t, "|") &&
			!strings.HasSuffix(t, "&") &&
			!strings.HasSuffix(t, ",") &&
			!strings.HasSuffix(t, "=") &&
			countBraces(t) == 0 &&
			countParens(t) == 0 &&
			countAngleBrackets(t) == 0)
}

// isClassDeclaration checks for class keyword.
func isClassDeclaration(trimmed string) bool {
	return strings.HasPrefix(trimmed, "class ") ||
		strings.HasPrefix(trimmed, "abstract class ") ||
		strings.HasPrefix(trimmed, "declare class ") ||
		strings.HasPrefix(trimmed, "declare abstract class ")
}

// isFunctionDeclaration checks for function keyword.
func isFunctionDeclaration(trimmed string) bool {
	return strings.HasPrefix(trimmed, "function ") ||
		strings.HasPrefix(trimmed, "function(") ||
		strings.HasPrefix(trimmed, "function*(") ||
		strings.HasPrefix(trimmed, "async function ") ||
		strings.HasPrefix(trimmed, "async function(") ||
		strings.HasPrefix(trimmed, "declare function ")
}

// isArrowFunctionDeclaration checks for arrow function patterns assigned to variables.
func isArrowFunctionDeclaration(trimmed string) bool {
	if !isVarKeyword(trimmed) {
		return false
	}
	return strings.Contains(trimmed, "=>")
}

// isTopLevelVarDeclaration checks for const/let/var at top level.
func isTopLevelVarDeclaration(trimmed string) bool {
	return isVarKeyword(trimmed) && !isArrowFunctionDeclaration(trimmed)
}

// isVarKeyword checks if line starts with const/let/var.
func isVarKeyword(trimmed string) bool {
	return strings.HasPrefix(trimmed, "const ") ||
		strings.HasPrefix(trimmed, "let ") ||
		strings.HasPrefix(trimmed, "var ") ||
		strings.HasPrefix(trimmed, "declare const ") ||
		strings.HasPrefix(trimmed, "declare let ") ||
		strings.HasPrefix(trimmed, "declare var ")
}

// isFieldDeclaration checks if a line in a class body is a field declaration.
func isFieldDeclaration(trimmed string) bool {
	// Field patterns: name: Type;  private name: Type;  readonly name: Type;
	// Also: #privateName: Type;  static name: Type;
	if strings.HasSuffix(trimmed, ";") || strings.HasSuffix(trimmed, ",") {
		// Likely a field if it doesn't have parentheses (method call) and has a colon or equals.
		if !strings.Contains(trimmed, "(") {
			return true
		}
		// Could be a field with a default value that's a function call.
		colonIdx := strings.Index(trimmed, ":")
		parenIdx := strings.Index(trimmed, "(")
		if colonIdx != -1 && colonIdx < parenIdx {
			return true
		}
	}
	// Field without semicolon (some styles).
	if !strings.Contains(trimmed, "(") && !strings.Contains(trimmed, "{") {
		if strings.Contains(trimmed, ":") || strings.Contains(trimmed, "=") {
			// Check it's not a method/getter/setter.
			firstWord := firstToken(trimmed)
			if firstWord != "get" && firstWord != "set" && firstWord != "async" &&
				firstWord != "static" && firstWord != "abstract" && firstWord != "constructor" &&
				firstWord != "*" {
				return true
			}
			// Could be "static fieldName: Type" -- check second word.
			if firstWord == "static" || firstWord == "abstract" {
				rest := strings.TrimPrefix(trimmed, firstWord+" ")
				rest = strings.TrimPrefix(rest, "readonly ")
				if !strings.Contains(rest, "(") {
					return true
				}
			}
		}
	}
	return false
}

// isMethodDeclaration checks if a line in a class body is a method.
func isMethodDeclaration(trimmed string) bool {
	if strings.Contains(trimmed, "(") {
		first := firstToken(trimmed)
		// Direct method: name(
		if strings.Contains(first, "(") {
			return true
		}
		switch first {
		case "public", "private", "protected", "static", "async", "abstract", "override", "*", "readonly":
			return true
		}
		// Check for pattern like: methodName<T>(
		if strings.Contains(first, "<") || (len(first) > 0 && isIdentChar(first[0])) {
			return true
		}
	}
	return false
}

// isConstructorOrAccessor checks for constructor, get, set in class body.
func isConstructorOrAccessor(trimmed string) bool {
	first := firstToken(trimmed)
	return first == "constructor" || first == "get" || first == "set"
}

// ---------------------------------------------------------------------------
// Extraction helpers
// ---------------------------------------------------------------------------

// extractSignatureBeforeBrace extracts text up to but not including the opening brace.
func extractSignatureBeforeBrace(line string) string {
	idx := strings.Index(line, "{")
	if idx == -1 {
		return strings.TrimRight(line, " \t;")
	}
	return strings.TrimRight(line[:idx], " \t")
}

// extractArrowSignature extracts an arrow function's signature.
func extractArrowSignature(line string) string {
	arrowIdx := strings.Index(line, "=>")
	if arrowIdx == -1 {
		return strings.TrimRight(line, " \t;")
	}
	// Include the arrow.
	afterArrow := strings.TrimSpace(line[arrowIdx+2:])
	if strings.HasPrefix(afterArrow, "{") {
		// Arrow with block body: extract up to and including => then add { ... }
		return strings.TrimRight(line[:arrowIdx+2], " ") + " { ... }"
	}
	// Arrow with expression body -- include the whole line (expression is the signature).
	return strings.TrimRight(line, " \t;")
}

// extractMethodSignature extracts a method signature from within a class body.
func extractMethodSignature(line, trimmed string) string {
	// If the method has a body on the same line, extract up to { and add { ... }
	braceIdx := strings.Index(trimmed, "{")
	if braceIdx != -1 {
		// Find the brace in the original line (preserving indentation).
		lineBraceIdx := strings.Index(line, "{")
		if lineBraceIdx != -1 {
			sig := strings.TrimRight(line[:lineBraceIdx], " \t")
			return sig + " { ... }"
		}
	}
	// Abstract or declaration-only method -- preserve verbatim.
	return strings.TrimRight(line, " \t")
}

// extractConstDeclaration extracts a constant declaration without the value.
func extractConstDeclaration(line, trimmed string) string {
	// For typed declarations: const name: Type = value -> const name: Type
	colonIdx := strings.Index(trimmed, ":")
	equalsIdx := strings.Index(trimmed, "=")

	if colonIdx != -1 && (equalsIdx == -1 || colonIdx < equalsIdx) {
		// Has type annotation. Find equals in the original line.
		lineEqualsIdx := findEqualsOutsideTypeAnnotation(line)
		if lineEqualsIdx != -1 {
			return strings.TrimRight(line[:lineEqualsIdx], " \t")
		}
		// No equals -- return as is (declare const style).
		return strings.TrimRight(line, " \t;")
	}

	// No type annotation: const name = value -> const name
	if equalsIdx != -1 {
		lineEqualsIdx := strings.Index(line, "=")
		if lineEqualsIdx != -1 {
			return strings.TrimRight(line[:lineEqualsIdx], " \t")
		}
	}

	return strings.TrimRight(line, " \t;")
}

// findEqualsOutsideTypeAnnotation finds the assignment = that's not inside a type annotation.
func findEqualsOutsideTypeAnnotation(line string) int {
	angleDepth := 0
	parenDepth := 0
	for i, ch := range line {
		switch ch {
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '=':
			if angleDepth == 0 && parenDepth == 0 {
				// Make sure it's not == or ===
				if i+1 < len(line) && line[i+1] == '=' {
					continue
				}
				// Make sure it's not => (arrow)
				if i+1 < len(line) && line[i+1] == '>' {
					continue
				}
				return i
			}
		}
	}
	return -1
}

// extractFunctionName extracts the name from a function declaration.
func extractFunctionName(trimmed string) string {
	// Remove prefixes: export default async function name(
	s := trimmed
	for _, prefix := range []string{"export ", "default ", "async ", "declare ", "function ", "function*"} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimSpace(s)

	// Extract identifier before ( or <
	name := extractIdentifier(s)
	if name == "function" {
		// "export default function" with no name.
		return ""
	}
	return name
}

// extractVarName extracts the variable name from a const/let/var declaration.
func extractVarName(trimmed string) string {
	s := trimmed
	for _, prefix := range []string{"export ", "default ", "declare "} {
		s = strings.TrimPrefix(s, prefix)
	}
	// Remove const/let/var.
	for _, kw := range []string{"const ", "let ", "var "} {
		if strings.HasPrefix(s, kw) {
			s = strings.TrimPrefix(s, kw)
			break
		}
	}
	s = strings.TrimSpace(s)
	return extractIdentifier(s)
}

// extractClassName extracts the class name from a class declaration line.
func extractClassName(line string) string {
	s := strings.TrimSpace(line)
	for _, prefix := range []string{"export ", "default ", "abstract ", "declare "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "class ")
	s = strings.TrimSpace(s)
	return extractIdentifier(s)
}

// extractInterfaceName extracts the interface name from a declaration.
func extractInterfaceName(text string) string {
	// Find the first line.
	line := text
	if idx := strings.Index(text, "\n"); idx != -1 {
		line = text[:idx]
	}
	s := strings.TrimSpace(line)
	for _, prefix := range []string{"export ", "default ", "declare "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "interface ")
	s = strings.TrimSpace(s)
	return extractIdentifier(s)
}

// extractEnumName extracts the enum name from a declaration.
func extractEnumName(text string) string {
	line := text
	if idx := strings.Index(text, "\n"); idx != -1 {
		line = text[:idx]
	}
	s := strings.TrimSpace(line)
	for _, prefix := range []string{"export ", "default ", "declare ", "const "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "enum ")
	s = strings.TrimSpace(s)
	return extractIdentifier(s)
}

// extractTypeAliasName extracts the type alias name.
func extractTypeAliasName(text string) string {
	line := text
	if idx := strings.Index(text, "\n"); idx != -1 {
		line = text[:idx]
	}
	s := strings.TrimSpace(line)
	for _, prefix := range []string{"export ", "default ", "declare "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "type ")
	s = strings.TrimSpace(s)
	return extractIdentifier(s)
}

// extractIdentifier extracts the first identifier from a string.
func extractIdentifier(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if isIdentRune(ch) {
			b.WriteRune(ch)
		} else {
			break
		}
	}
	return b.String()
}

// isIdentRune checks if a rune is valid in a JS/TS identifier.
func isIdentRune(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '$'
}

// isIdentChar checks if a byte is valid as the start of a JS/TS identifier.
func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || b == '$'
}

// firstToken returns the first whitespace-delimited token from a string.
func firstToken(s string) string {
	idx := strings.IndexFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '(' || r == '<' || r == ':'
	})
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// ---------------------------------------------------------------------------
// Brace/paren counting helpers
// ---------------------------------------------------------------------------

// countBraces counts the net brace depth change for a line, ignoring braces in strings.
func countBraces(line string) int {
	depth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	escaped := false
	for _, ch := range line {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		switch {
		case inSingleQuote:
			if ch == '\'' {
				inSingleQuote = false
			}
		case inDoubleQuote:
			if ch == '"' {
				inDoubleQuote = false
			}
		case inBacktick:
			if ch == '`' {
				inBacktick = false
			}
		default:
			switch ch {
			case '\'':
				inSingleQuote = true
			case '"':
				inDoubleQuote = true
			case '`':
				inBacktick = true
			case '{':
				depth++
			case '}':
				depth--
			}
		}
	}
	return depth
}

// countParens counts net parenthesis depth.
func countParens(text string) int {
	depth := 0
	for _, ch := range text {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		}
	}
	return depth
}

// countAngleBrackets counts net angle bracket depth (approximate).
func countAngleBrackets(text string) int {
	depth := 0
	for _, ch := range text {
		switch ch {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth
}

// maybePrependDoc prepends a doc comment and/or decorators to source text.
func maybePrependDoc(docComment string, decorators []string, source string) string {
	var parts []string
	if docComment != "" {
		parts = append(parts, docComment)
	}
	parts = append(parts, decorators...)
	parts = append(parts, source)
	return strings.Join(parts, "\n")
}
