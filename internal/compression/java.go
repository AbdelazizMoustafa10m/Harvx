package compression

import (
	"context"
	"strings"
	"unicode"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*JavaCompressor)(nil)

// JavaCompressor implements LanguageCompressor for Java source code.
// It uses a line-by-line state machine parser to extract structural signatures.
// The compressor is stateless and safe for concurrent use.
type JavaCompressor struct{}

// NewJavaCompressor creates a Java compressor.
func NewJavaCompressor() *JavaCompressor {
	return &JavaCompressor{}
}

// Compress parses Java source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *JavaCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &javaParser{}
	return p.parse(ctx, source)
}

// Language returns "java".
func (c *JavaCompressor) Language() string {
	return "java"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *JavaCompressor) SupportedNodeTypes() []string {
	return []string{
		"package_declaration",
		"import_declaration",
		"class_declaration",
		"interface_declaration",
		"method_declaration",
		"constructor_declaration",
		"enum_declaration",
		"annotation_type_declaration",
		"record_declaration",
	}
}

// ---------------------------------------------------------------------------
// Java parser state machine
// ---------------------------------------------------------------------------

// javaState tracks the current state of the Java line-by-line parser.
type javaState int

const (
	javaStateTopLevel         javaState = iota // Scanning for declarations
	javaStateInBlockComment                    // Inside /* ... */ or /** ... */ comment
	javaStateInClassBody                       // Extracting class members at appropriate depth
	javaStateInInterfaceBody                   // Extracting interface members (all structural)
	javaStateInEnumBody                        // Accumulating enum body
	javaStateInAnnotationBody                  // Extracting annotation type body
	javaStateInMethodBody                      // Skipping method/constructor body by counting braces
)

// javaParseCtx holds all mutable state for the Java line-by-line parser.
type javaParseCtx struct {
	state      javaState
	braceDepth int

	// Accumulator for multi-line constructs.
	accum          strings.Builder
	accumStartLine int

	// Javadoc and annotation lines pending attachment to the next declaration.
	javadocLines    []string
	annotationLines []string

	// For class/interface/enum/annotation bodies: the header and accumulated members.
	blockHeader     string
	blockName       string
	blockStartLine  int
	blockDocComment string
	blockAttrs      []string
	blockFields     []string
	blockMethods    []string
	blockKind       SignatureKind

	// Collected signatures.
	sigs []Signature
}

// javaParser provides Java source parsing using a line-by-line state machine.
type javaParser struct{}

// parse extracts structural signatures from Java source code.
func (p *javaParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "java",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &javaParseCtx{
		state: javaStateTopLevel,
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
		case javaStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case javaStateInBlockComment:
			p.handleBlockComment(pc, trimmed, line, lineNum)

		case javaStateInClassBody:
			p.handleClassBody(pc, trimmed, line, lineNum)

		case javaStateInInterfaceBody:
			p.handleInterfaceBody(pc, trimmed, line, lineNum)

		case javaStateInEnumBody:
			p.handleEnumBody(pc, trimmed, line, lineNum)

		case javaStateInAnnotationBody:
			p.handleAnnotationBody(pc, trimmed, line, lineNum)

		case javaStateInMethodBody:
			p.handleMethodBody(pc, trimmed, lineNum)
		}
	}

	// Flush any pending accumulated block at EOF.
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "java",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *javaParser) flushPending(pc *javaParseCtx, lastLine int) {
	switch pc.state {
	case javaStateInClassBody, javaStateInInterfaceBody, javaStateInAnnotationBody:
		p.emitBlockSignature(pc, lastLine)

	case javaStateInEnumBody:
		text := pc.accum.String()
		doc := buildJavaDoc(pc.javadocLines, pc.annotationLines)
		sig := Signature{
			Kind:      KindType,
			Name:      pc.blockName,
			Source:    maybePrependDoc(doc, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lastLine,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.javadocLines = nil
		pc.annotationLines = nil
	}
}

// ---------------------------------------------------------------------------
// Top-level handler
// ---------------------------------------------------------------------------

// handleTopLevel processes a line when the parser is at the top level.
func (p *javaParser) handleTopLevel(pc *javaParseCtx, trimmed, line string, lineNum int) {
	// Empty line resets Javadoc and annotation accumulation.
	if trimmed == "" {
		pc.javadocLines = nil
		pc.annotationLines = nil
		return
	}

	// Block comment start: /* ... or /** ...
	if strings.HasPrefix(trimmed, "/*") {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		if strings.Contains(trimmed[2:], "*/") {
			// Single-line block comment -- treat as Javadoc if it starts with /**.
			if strings.HasPrefix(trimmed, "/**") {
				pc.javadocLines = append(pc.javadocLines, line)
			}
			return
		}
		pc.state = javaStateInBlockComment
		return
	}

	// Single-line comment: // -- skip (not Javadoc).
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Annotation: @Something or @Something(...)
	if strings.HasPrefix(trimmed, "@") && !strings.HasPrefix(trimmed, "@interface") {
		pc.annotationLines = append(pc.annotationLines, line)
		return
	}

	// Package declaration.
	if strings.HasPrefix(trimmed, "package ") {
		sig := Signature{
			Kind:      KindImport,
			Name:      extractJavaPackageName(trimmed),
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.javadocLines = nil
		pc.annotationLines = nil
		return
	}

	// Import declaration.
	if strings.HasPrefix(trimmed, "import ") {
		sig := Signature{
			Kind:      KindImport,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.javadocLines = nil
		pc.annotationLines = nil
		return
	}

	// Annotation type declaration: @interface Name { ... }
	if isJavaAnnotationTypeDecl(trimmed) {
		p.handleAnnotationTypeDecl(pc, trimmed, line, lineNum)
		return
	}

	// Enum declaration.
	if isJavaEnumDecl(trimmed) {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

	// Interface declaration.
	if isJavaInterfaceDecl(trimmed) {
		p.handleInterfaceDecl(pc, trimmed, line, lineNum)
		return
	}

	// Class declaration (including record).
	if isJavaClassDecl(trimmed) || isJavaRecordDecl(trimmed) {
		p.handleClassDecl(pc, trimmed, line, lineNum)
		return
	}

	// Anything else at top level -- discard accumulated Javadoc/annotations.
	pc.javadocLines = nil
	pc.annotationLines = nil
}

// ---------------------------------------------------------------------------
// Block comment handler
// ---------------------------------------------------------------------------

// handleBlockComment accumulates /* ... */ or /** ... */ block comment lines.
func (p *javaParser) handleBlockComment(pc *javaParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if strings.Contains(trimmed, "*/") {
		text := pc.accum.String()
		// If it was a Javadoc comment (starts with /**), save for next declaration.
		firstLine := getFirstLine(text)
		if strings.Contains(strings.TrimSpace(firstLine), "/**") {
			pc.javadocLines = append(pc.javadocLines, text)
		}
		pc.accum.Reset()
		pc.state = javaStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Method body handler (skipping)
// ---------------------------------------------------------------------------

// handleMethodBody skips method/constructor body by counting braces.
func (p *javaParser) handleMethodBody(pc *javaParseCtx, trimmed string, lineNum int) {
	pc.braceDepth += javaCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = javaStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Class declaration handling
// ---------------------------------------------------------------------------

// handleClassDecl processes a class or record declaration.
func (p *javaParser) handleClassDecl(pc *javaParseCtx, trimmed, line string, lineNum int) {
	doc := buildJavaDoc(pc.javadocLines, pc.annotationLines)
	pc.javadocLines = nil
	pc.annotationLines = nil

	pc.blockHeader = line
	if isJavaRecordDecl(trimmed) {
		pc.blockName = extractJavaRecordName(trimmed)
	} else {
		pc.blockName = extractJavaClassName(trimmed)
	}
	pc.blockStartLine = lineNum
	pc.blockDocComment = doc
	pc.blockAttrs = nil
	pc.blockFields = nil
	pc.blockMethods = nil
	pc.blockKind = KindClass

	braces := javaCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = javaStateInClassBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty class on one line: class Foo {}
		p.emitBlockSignature(pc, lineNum)
	}
}

// handleClassBody processes lines inside a class body.
func (p *javaParser) handleClassBody(pc *javaParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += javaCountBraces(trimmed)

	// Class closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = javaStateTopLevel
		return
	}

	// We only extract members at depth 1 (the class body level).
	if prevDepth != 1 {
		return
	}

	// Skip empty lines.
	if trimmed == "" {
		return
	}

	// Javadoc inside class body.
	if strings.HasPrefix(trimmed, "/**") {
		if strings.Contains(trimmed[3:], "*/") {
			// Single-line Javadoc.
			pc.blockMethods = append(pc.blockMethods, line)
		}
		// Multi-line Javadoc inside class -- we just track it as a method line.
		// Since we are only extracting at depth 1, multi-line Javadocs
		// that don't close on the same line will be collected line by line
		// until the closing */ is found, but this simplified approach
		// captures single-line Javadoc before members.
		return
	}

	// Single-line comments inside class body -- skip.
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Block comments inside class body -- skip.
	if strings.HasPrefix(trimmed, "/*") {
		return
	}

	// Annotations inside class body.
	if strings.HasPrefix(trimmed, "@") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Nested class/interface/enum -- extract header only.
	if isJavaClassDecl(trimmed) || isJavaInterfaceDecl(trimmed) ||
		isJavaEnumDecl(trimmed) || isJavaAnnotationTypeDecl(trimmed) || isJavaRecordDecl(trimmed) {
		sigText := extractJavaSignatureBeforeBrace(line)
		pc.blockMethods = append(pc.blockMethods, sigText+" { ... }")
		return
	}

	// Constructor or method declaration.
	if isJavaMethodOrConstructor(trimmed, pc.blockName) {
		sigText := extractJavaMethodSignature(line, trimmed)
		pc.blockMethods = append(pc.blockMethods, sigText)
		return
	}

	// Field declaration (ends with ; or has no opening brace).
	if isJavaFieldDeclaration(trimmed) {
		pc.blockFields = append(pc.blockFields, line)
		return
	}

	// Static/instance initializer blocks -- skip.
	if trimmed == "static {" || trimmed == "{" {
		return
	}
}

// ---------------------------------------------------------------------------
// Interface declaration handling
// ---------------------------------------------------------------------------

// handleInterfaceDecl processes an interface declaration.
func (p *javaParser) handleInterfaceDecl(pc *javaParseCtx, trimmed, line string, lineNum int) {
	doc := buildJavaDoc(pc.javadocLines, pc.annotationLines)
	pc.javadocLines = nil
	pc.annotationLines = nil

	pc.blockHeader = line
	pc.blockName = extractJavaInterfaceName(trimmed)
	pc.blockStartLine = lineNum
	pc.blockDocComment = doc
	pc.blockAttrs = nil
	pc.blockFields = nil
	pc.blockMethods = nil
	pc.blockKind = KindInterface

	braces := javaCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = javaStateInInterfaceBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty interface on one line.
		p.emitBlockSignature(pc, lineNum)
	}
}

// handleInterfaceBody processes lines inside an interface body.
// All members in an interface are structural (method signatures, constants, default methods).
func (p *javaParser) handleInterfaceBody(pc *javaParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += javaCountBraces(trimmed)

	// Interface closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = javaStateTopLevel
		return
	}

	// Extract members at depth 1.
	if prevDepth != 1 {
		return
	}

	if trimmed == "" {
		return
	}

	// Skip comments.
	if strings.HasPrefix(trimmed, "//") {
		return
	}

	// Annotations inside interface body.
	if strings.HasPrefix(trimmed, "@") {
		pc.blockMethods = append(pc.blockMethods, line)
		return
	}

	// Javadoc inside interface body.
	if strings.HasPrefix(trimmed, "/**") || strings.HasPrefix(trimmed, "/*") {
		return
	}

	// Default or static methods -- extract signature only.
	if strings.Contains(trimmed, "{") {
		sigText := extractJavaMethodSignature(line, trimmed)
		pc.blockMethods = append(pc.blockMethods, sigText)
		return
	}

	// Abstract method signatures (no body), constants.
	pc.blockMethods = append(pc.blockMethods, line)
}

// ---------------------------------------------------------------------------
// Enum declaration handling
// ---------------------------------------------------------------------------

// handleEnumDecl processes an enum declaration.
func (p *javaParser) handleEnumDecl(pc *javaParseCtx, trimmed, line string, lineNum int) {
	doc := buildJavaDoc(pc.javadocLines, pc.annotationLines)
	pc.javadocLines = nil
	pc.annotationLines = nil

	braces := javaCountBraces(trimmed)

	pc.blockName = extractJavaEnumName(trimmed)

	// Single-line enum.
	if strings.Contains(trimmed, "{") && braces == 0 {
		sig := Signature{
			Kind:      KindType,
			Name:      pc.blockName,
			Source:    maybePrependDoc(doc, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		return
	}

	// Multi-line enum -- accumulate entire body.
	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.blockDocComment = doc
		pc.state = javaStateInEnumBody
		return
	}

	// Enum without body on this line (unlikely but handle gracefully).
	sig := Signature{
		Kind:      KindType,
		Name:      pc.blockName,
		Source:    maybePrependDoc(doc, nil, line),
		StartLine: lineNum,
		EndLine:   lineNum,
	}
	pc.sigs = append(pc.sigs, sig)
}

// handleEnumBody accumulates enum body lines.
func (p *javaParser) handleEnumBody(pc *javaParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += javaCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		sig := Signature{
			Kind:      KindType,
			Name:      pc.blockName,
			Source:    maybePrependDoc(pc.blockDocComment, nil, text),
			StartLine: pc.accumStartLine,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.accum.Reset()
		pc.javadocLines = nil
		pc.annotationLines = nil
		pc.blockDocComment = ""
		pc.braceDepth = 0
		pc.state = javaStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Annotation type declaration handling
// ---------------------------------------------------------------------------

// handleAnnotationTypeDecl processes an @interface (annotation type) declaration.
func (p *javaParser) handleAnnotationTypeDecl(pc *javaParseCtx, trimmed, line string, lineNum int) {
	doc := buildJavaDoc(pc.javadocLines, pc.annotationLines)
	pc.javadocLines = nil
	pc.annotationLines = nil

	pc.blockHeader = line
	pc.blockName = extractJavaAnnotationTypeName(trimmed)
	pc.blockStartLine = lineNum
	pc.blockDocComment = doc
	pc.blockAttrs = nil
	pc.blockFields = nil
	pc.blockMethods = nil
	pc.blockKind = KindType

	braces := javaCountBraces(trimmed)
	if braces > 0 {
		pc.braceDepth = braces
		pc.state = javaStateInAnnotationBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty annotation type on one line.
		p.emitBlockSignature(pc, lineNum)
	}
}

// handleAnnotationBody processes lines inside an annotation type body.
func (p *javaParser) handleAnnotationBody(pc *javaParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += javaCountBraces(trimmed)

	// Annotation type closed.
	if pc.braceDepth <= 0 {
		p.emitBlockSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = javaStateTopLevel
		return
	}

	// Extract members at depth 1.
	if prevDepth != 1 {
		return
	}

	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return
	}

	// Annotation elements and constants.
	pc.blockMethods = append(pc.blockMethods, line)
}

// ---------------------------------------------------------------------------
// Block signature emission (class, interface, annotation type)
// ---------------------------------------------------------------------------

// emitBlockSignature builds and emits a signature for a class/interface/annotation block.
func (p *javaParser) emitBlockSignature(pc *javaParseCtx, endLine int) {
	var b strings.Builder
	headerTrimmed := strings.TrimRight(pc.blockHeader, " \t\n\r")

	// Ensure the header ends with {.
	if !strings.HasSuffix(headerTrimmed, "{") {
		headerTrimmed += " {"
	}
	b.WriteString(headerTrimmed)

	for _, f := range pc.blockFields {
		b.WriteString("\n")
		b.WriteString(f)
	}

	for _, m := range pc.blockMethods {
		b.WriteString("\n")
		b.WriteString(m)
	}

	b.WriteString("\n}")

	sig := Signature{
		Kind:      pc.blockKind,
		Name:      pc.blockName,
		Source:    maybePrependDoc(pc.blockDocComment, nil, b.String()),
		StartLine: pc.blockStartLine,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)

	pc.blockHeader = ""
	pc.blockName = ""
	pc.blockFields = nil
	pc.blockMethods = nil
	pc.blockDocComment = ""
	pc.blockAttrs = nil
}

// ---------------------------------------------------------------------------
// Java-specific detection helpers
// ---------------------------------------------------------------------------

// isJavaClassDecl checks if a trimmed line declares a class.
func isJavaClassDecl(trimmed string) bool {
	s := stripJavaModifiers(trimmed)
	return strings.HasPrefix(s, "class ")
}

// isJavaRecordDecl checks if a trimmed line declares a record (Java 16+).
func isJavaRecordDecl(trimmed string) bool {
	s := stripJavaModifiers(trimmed)
	return strings.HasPrefix(s, "record ")
}

// isJavaInterfaceDecl checks if a trimmed line declares an interface.
func isJavaInterfaceDecl(trimmed string) bool {
	s := stripJavaModifiers(trimmed)
	return strings.HasPrefix(s, "interface ")
}

// isJavaEnumDecl checks if a trimmed line declares an enum.
func isJavaEnumDecl(trimmed string) bool {
	s := stripJavaModifiers(trimmed)
	return strings.HasPrefix(s, "enum ")
}

// isJavaAnnotationTypeDecl checks if a trimmed line declares an annotation type (@interface).
func isJavaAnnotationTypeDecl(trimmed string) bool {
	s := stripJavaModifiers(trimmed)
	return strings.HasPrefix(s, "@interface ")
}

// isJavaMethodOrConstructor checks if a trimmed line inside a class body
// is a method or constructor declaration.
func isJavaMethodOrConstructor(trimmed, className string) bool {
	// Must contain parentheses to be a method/constructor.
	if !strings.Contains(trimmed, "(") {
		return false
	}

	// Skip lines that are clearly not declarations.
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "@") {
		return false
	}

	// If it ends with ; it's likely a field with an initializer that has parens
	// (e.g., List<String> items = new ArrayList<>()), but we need to be careful.
	// Method declarations either end with { or have abstract/native modifier and end with ;
	s := stripJavaModifiers(trimmed)

	// Constructor: ClassName(...)
	if strings.HasPrefix(s, className+"(") {
		return true
	}

	// Method: returnType name(...)
	// Has a return type followed by a name followed by (
	// Check for common patterns like "void foo(" or "String bar(" or generic "List<X> baz("
	parenIdx := strings.Index(s, "(")
	if parenIdx == -1 {
		return false
	}
	before := s[:parenIdx]
	// Must have at least two tokens (return type + name) or be a constructor.
	tokens := javaTokenize(before)
	if len(tokens) >= 2 {
		return true
	}

	// Single token before ( could be a constructor.
	if len(tokens) == 1 {
		return true
	}

	return false
}

// isJavaFieldDeclaration checks if a trimmed line inside a class is a field declaration.
func isJavaFieldDeclaration(trimmed string) bool {
	// Fields end with ; and don't contain {.
	if !strings.HasSuffix(trimmed, ";") {
		return false
	}
	// If it has parentheses, it might be a method/constructor, not a field,
	// but it could also be a field with an initializer like: int x = foo();
	// We consider it a field if it ends with ; (method declarations with bodies have {).
	// Abstract methods also end with ; but they contain parentheses and have modifiers.
	// We check: if there's no { on the line and it ends with ;, it's likely a field or
	// an abstract method. For simplicity we include both -- abstract methods in a class
	// body are structurally useful.
	return true
}

// stripJavaModifiers removes Java access modifiers and other keywords from the start of a line.
func stripJavaModifiers(trimmed string) string {
	s := trimmed
	modifiers := []string{
		"public ", "private ", "protected ",
		"static ", "final ", "abstract ",
		"synchronized ", "native ", "strictfp ",
		"transient ", "volatile ",
		"sealed ", "non-sealed ", "default ",
	}
	changed := true
	for changed {
		changed = false
		for _, mod := range modifiers {
			if strings.HasPrefix(s, mod) {
				s = strings.TrimPrefix(s, mod)
				changed = true
				break
			}
		}
	}
	return s
}

// javaTokenize splits a string into whitespace-delimited tokens,
// treating < > as part of generic type tokens.
func javaTokenize(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder
	angleDepth := 0

	for _, ch := range s {
		if ch == '<' {
			angleDepth++
			current.WriteRune(ch)
			continue
		}
		if ch == '>' {
			if angleDepth > 0 {
				angleDepth--
			}
			current.WriteRune(ch)
			continue
		}
		if unicode.IsSpace(ch) && angleDepth == 0 {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// ---------------------------------------------------------------------------
// Java-specific name extraction
// ---------------------------------------------------------------------------

// extractJavaPackageName extracts the package name from a "package xxx.yyy;" line.
func extractJavaPackageName(trimmed string) string {
	s := strings.TrimPrefix(trimmed, "package ")
	s = strings.TrimSuffix(s, ";")
	return strings.TrimSpace(s)
}

// extractJavaClassName extracts the class name from a class declaration line.
func extractJavaClassName(trimmed string) string {
	s := stripJavaModifiers(trimmed)
	s = strings.TrimPrefix(s, "class ")
	return extractJavaIdentifier(s)
}

// extractJavaRecordName extracts the record name from a record declaration line.
func extractJavaRecordName(trimmed string) string {
	s := stripJavaModifiers(trimmed)
	s = strings.TrimPrefix(s, "record ")
	return extractJavaIdentifier(s)
}

// extractJavaInterfaceName extracts the interface name from an interface declaration line.
func extractJavaInterfaceName(trimmed string) string {
	s := stripJavaModifiers(trimmed)
	s = strings.TrimPrefix(s, "interface ")
	return extractJavaIdentifier(s)
}

// extractJavaEnumName extracts the enum name from an enum declaration line.
func extractJavaEnumName(trimmed string) string {
	s := stripJavaModifiers(trimmed)
	s = strings.TrimPrefix(s, "enum ")
	return extractJavaIdentifier(s)
}

// extractJavaAnnotationTypeName extracts the annotation type name from an @interface declaration.
func extractJavaAnnotationTypeName(trimmed string) string {
	s := stripJavaModifiers(trimmed)
	s = strings.TrimPrefix(s, "@interface ")
	return extractJavaIdentifier(s)
}

// extractJavaIdentifier extracts the first Java identifier from a string.
func extractJavaIdentifier(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for i, ch := range s {
		if i == 0 {
			if !unicode.IsLetter(ch) && ch != '_' && ch != '$' {
				break
			}
		} else {
			if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '$' {
				break
			}
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// extractJavaMethodSignature extracts a method/constructor signature up to
// (but not including) the opening brace of the body.
func extractJavaMethodSignature(line, trimmed string) string {
	if !strings.Contains(trimmed, "{") {
		// No body on this line (abstract or interface method).
		return strings.TrimRight(line, " \t")
	}

	// Find the opening brace in the original line.
	bodyIdx := findJavaBodyBrace(line)
	if bodyIdx == -1 {
		return strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(line[:bodyIdx], " \t")
}

// extractJavaSignatureBeforeBrace extracts text up to but not including
// the opening brace for declarations like nested classes.
func extractJavaSignatureBeforeBrace(line string) string {
	bodyIdx := findJavaBodyBrace(line)
	if bodyIdx == -1 {
		return strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(line[:bodyIdx], " \t")
}

// findJavaBodyBrace finds the index of the `{` that opens the body.
// After all parentheses and angle brackets are balanced, the next `{` is
// the body opener.
func findJavaBodyBrace(line string) int {
	parenDepth := 0
	angleDepth := 0
	inDoubleQuote := false
	inChar := false
	escaped := false

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

		// Block comment.
		if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "*/")
			if end >= 0 {
				i = i + 2 + end + 1
				continue
			}
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inChar = true
		case '(':
			parenDepth++
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
			if parenDepth == 0 && angleDepth == 0 {
				return i
			}
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// Java doc/annotation helpers
// ---------------------------------------------------------------------------

// buildJavaDoc combines accumulated Javadoc comment lines and annotation lines
// into a single doc string. Returns empty string if there are none.
func buildJavaDoc(javadocLines, annotationLines []string) string {
	if len(javadocLines) == 0 && len(annotationLines) == 0 {
		return ""
	}
	var parts []string
	parts = append(parts, javadocLines...)
	parts = append(parts, annotationLines...)
	return strings.Join(parts, "\n")
}

// ---------------------------------------------------------------------------
// Java brace counting
// ---------------------------------------------------------------------------

// javaCountBraces counts the net brace depth for a Java source line,
// ignoring braces inside string literals, char literals, and comments.
func javaCountBraces(line string) int {
	depth := 0
	inDoubleQuote := false
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

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inChar = true
		case '{':
			depth++
		case '}':
			depth--
		}
	}

	return depth
}