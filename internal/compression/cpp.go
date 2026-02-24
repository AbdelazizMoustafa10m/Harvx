package compression

import (
	"context"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*CppCompressor)(nil)

// CppCompressor implements LanguageCompressor for C++ source code.
// It uses a line-by-line state machine parser to extract structural signatures,
// handling all C constructs plus C++-specific features such as classes,
// templates, namespaces, and using declarations.
// The compressor is stateless and safe for concurrent use.
type CppCompressor struct{}

// NewCppCompressor creates a C++ compressor.
func NewCppCompressor() *CppCompressor {
	return &CppCompressor{}
}

// Compress parses C++ source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *CppCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &cppParser{}
	return p.parse(ctx, source)
}

// Language returns "cpp".
func (c *CppCompressor) Language() string {
	return "cpp"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *CppCompressor) SupportedNodeTypes() []string {
	return []string{
		"preproc_include",
		"preproc_def",
		"function_definition",
		"declaration",
		"struct_specifier",
		"enum_specifier",
		"type_definition",
		"class_specifier",
		"template_declaration",
		"namespace_definition",
		"using_declaration",
	}
}

// ---------------------------------------------------------------------------
// C++ parser state machine
// ---------------------------------------------------------------------------

// cppState tracks the current state of the C++ parser.
type cppState int

const (
	cppStateTopLevel       cppState = iota
	cppStateInBlockComment          // Inside /* ... */ block comment
	cppStateInFuncBody              // Skipping function/method body
	cppStateInStructBody            // Accumulating struct body
	cppStateInEnumBody              // Accumulating enum body
	cppStateInPreproc               // Multi-line preprocessor
	cppStateInClassBody             // Extracting class members
	cppStateInNamespace             // Extracting namespace members
)

// cppParseCtx holds all mutable state for the C++ parser.
type cppParseCtx struct {
	state      cppState
	braceDepth int

	// Accumulator for multi-line constructs.
	accum          strings.Builder
	accumStartLine int

	// Preprocessor accumulation.
	preprocAccum strings.Builder

	// Doc comment tracking.
	docComment string

	// Template prefix (captured from "template<...>" line to attach to next decl).
	templatePrefix string

	// Class body tracking.
	classHeader     string
	className       string
	classStartLine  int
	classDocComment string
	classFields     []string
	classMethods    []string

	// Namespace tracking.
	nsHeader    string
	nsName      string
	nsStartLine int
	nsMembers   []string
	nsBraceBase int // brace depth when namespace body started

	// Collected signatures.
	sigs []Signature
}

// cppParser provides C++ source parsing using a line-by-line state machine.
type cppParser struct{}

// parse extracts structural signatures from C++ source code.
func (p *cppParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "cpp",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &cppParseCtx{
		state: cppStateTopLevel,
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
		case cppStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case cppStateInBlockComment:
			p.handleBlockComment(pc, trimmed, line, lineNum)

		case cppStateInFuncBody:
			p.handleFuncBody(pc, trimmed)

		case cppStateInStructBody:
			p.handleStructBody(pc, trimmed, line, lineNum)

		case cppStateInEnumBody:
			p.handleEnumBody(pc, trimmed, line, lineNum)

		case cppStateInPreproc:
			p.handlePreproc(pc, trimmed, line, lineNum)

		case cppStateInClassBody:
			p.handleClassBody(pc, trimmed, line, lineNum)

		case cppStateInNamespace:
			p.handleNamespace(pc, trimmed, line, lineNum)
		}
	}

	// Flush pending state at EOF.
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "cpp",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *cppParser) flushPending(pc *cppParseCtx, lastLine int) {
	switch pc.state {
	case cppStateInStructBody:
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

	case cppStateInEnumBody:
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

	case cppStateInClassBody:
		p.emitClassSignature(pc, lastLine)

	case cppStateInNamespace:
		p.emitNamespaceSignature(pc, lastLine)

	case cppStateInPreproc:
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

// handleTopLevel processes a line when the C++ parser is at the top level.
func (p *cppParser) handleTopLevel(pc *cppParseCtx, trimmed, line string, lineNum int) {
	// Empty line resets doc comment.
	if trimmed == "" {
		pc.docComment = ""
		pc.templatePrefix = ""
		return
	}

	// Doc comment start: /** ...
	if isCDocCommentStart(trimmed) {
		if strings.Contains(trimmed, "*/") {
			pc.docComment = line
			return
		}
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.state = cppStateInBlockComment
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
		pc.state = cppStateInBlockComment
		return
	}

	// Single-line // comments as doc comments.
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

	// Template declaration: template<...>
	if isCppTemplateDecl(trimmed) {
		pc.templatePrefix = line
		return
	}

	// Using declaration/directive.
	if isCppUsingDecl(trimmed) {
		source := line
		if pc.templatePrefix != "" {
			source = pc.templatePrefix + "\n" + line
			pc.templatePrefix = ""
		}
		sig := Signature{
			Kind:      KindType,
			Source:    maybePrependDoc(pc.docComment, nil, source),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		return
	}

	// Namespace definition.
	if isCppNamespaceDecl(trimmed) {
		p.handleNamespaceDecl(pc, trimmed, line, lineNum)
		return
	}

	// Class declaration.
	if isCppClassDecl(trimmed) {
		p.handleClassDecl(pc, trimmed, line, lineNum)
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

	// Enum declaration (including enum class).
	if isCppEnumDecl(trimmed) {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

	// Forward declarations.
	if isCForwardDecl(trimmed) || isCppClassForwardDecl(trimmed) {
		sig := Signature{
			Kind:      KindType,
			Source:    maybePrependDoc(pc.docComment, nil, line),
			StartLine: lineNum,
			EndLine:   lineNum,
		}
		pc.sigs = append(pc.sigs, sig)
		pc.docComment = ""
		pc.templatePrefix = ""
		return
	}

	// Function definition.
	if isCFuncDefinition(trimmed) || isCppMethodDefinition(trimmed) {
		p.handleFuncDef(pc, trimmed, line, lineNum)
		return
	}

	// Function prototype.
	if isCFuncPrototype(trimmed) || isCppMethodPrototype(trimmed) {
		name := extractCFuncName(trimmed)
		source := line
		if pc.templatePrefix != "" {
			source = pc.templatePrefix + "\n" + line
			pc.templatePrefix = ""
		}
		sig := Signature{
			Kind:      KindFunction,
			Name:      name,
			Source:    maybePrependDoc(pc.docComment, nil, source),
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
		pc.templatePrefix = ""
		return
	}

	// Anything else -- discard doc comment and template prefix.
	pc.docComment = ""
	pc.templatePrefix = ""
}

// ---------------------------------------------------------------------------
// Block comment handler
// ---------------------------------------------------------------------------

// handleBlockComment accumulates /* ... */ block comment lines.
func (p *cppParser) handleBlockComment(pc *cppParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	if strings.Contains(trimmed, "*/") {
		text := pc.accum.String()
		firstLine := getFirstLine(text)
		if strings.Contains(strings.TrimSpace(firstLine), "/**") {
			pc.docComment = text
		}
		pc.accum.Reset()
		pc.state = cppStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Function body handler (skipping)
// ---------------------------------------------------------------------------

// handleFuncBody skips function body by counting braces.
func (p *cppParser) handleFuncBody(pc *cppParseCtx, trimmed string) {
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		pc.braceDepth = 0
		pc.state = cppStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Struct body handler (accumulating)
// ---------------------------------------------------------------------------

// handleStructBody accumulates struct body lines.
func (p *cppParser) handleStructBody(pc *cppParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		name := extractCStructName(getFirstLine(text))
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
		pc.state = cppStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Enum body handler (accumulating)
// ---------------------------------------------------------------------------

// handleEnumBody accumulates enum body lines.
func (p *cppParser) handleEnumBody(pc *cppParseCtx, trimmed, line string, lineNum int) {
	pc.accum.WriteString("\n")
	pc.accum.WriteString(line)
	pc.braceDepth += cCountBraces(trimmed)
	if pc.braceDepth <= 0 {
		text := pc.accum.String()
		name := extractCppEnumName(getFirstLine(text))
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
		pc.state = cppStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Preprocessor handler
// ---------------------------------------------------------------------------

// handlePreprocessor processes a preprocessor directive at the top level.
func (p *cppParser) handlePreprocessor(pc *cppParseCtx, trimmed, line string, lineNum int) {
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

	if isCDefine(trimmed) {
		if isMultiLineDefine(trimmed) {
			pc.preprocAccum.Reset()
			pc.preprocAccum.WriteString(line)
			pc.accumStartLine = lineNum
			pc.state = cppStateInPreproc
			return
		}
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

	pc.docComment = ""
}

// handlePreproc accumulates multi-line preprocessor directive lines.
func (p *cppParser) handlePreproc(pc *cppParseCtx, trimmed, line string, lineNum int) {
	pc.preprocAccum.WriteString("\n")
	pc.preprocAccum.WriteString(line)

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
		pc.state = cppStateTopLevel
	}
}

// ---------------------------------------------------------------------------
// Declaration handlers
// ---------------------------------------------------------------------------

// handleFuncDef processes a function definition line.
func (p *cppParser) handleFuncDef(pc *cppParseCtx, trimmed, line string, lineNum int) {
	sigText := extractCFuncSignature(line)
	name := extractCFuncName(trimmed)

	if pc.templatePrefix != "" {
		sigText = pc.templatePrefix + "\n" + sigText
		pc.templatePrefix = ""
	}

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
		pc.state = cppStateInFuncBody
	}
}

// handleTypedef processes a typedef statement.
func (p *cppParser) handleTypedef(pc *cppParseCtx, trimmed, line string, lineNum int) {
	if strings.HasPrefix(trimmed, "typedef struct") {
		p.handleStructDecl(pc, trimmed, line, lineNum)
		return
	}
	if strings.HasPrefix(trimmed, "typedef enum") {
		p.handleEnumDecl(pc, trimmed, line, lineNum)
		return
	}

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
func (p *cppParser) handleStructDecl(pc *cppParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

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

	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = cppStateInStructBody
		return
	}

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

// handleEnumDecl processes an enum or enum class declaration.
func (p *cppParser) handleEnumDecl(pc *cppParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

	if strings.Contains(trimmed, "{") && braces == 0 {
		name := extractCppEnumName(trimmed)
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

	if strings.Contains(trimmed, "{") && braces > 0 {
		pc.accum.Reset()
		pc.accum.WriteString(line)
		pc.accumStartLine = lineNum
		pc.braceDepth = braces
		pc.state = cppStateInEnumBody
		return
	}

	name := extractCppEnumName(trimmed)
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
// Class body handler
// ---------------------------------------------------------------------------

// handleClassDecl processes a class declaration.
func (p *cppParser) handleClassDecl(pc *cppParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

	header := line
	if pc.templatePrefix != "" {
		header = pc.templatePrefix + "\n" + line
		pc.templatePrefix = ""
	}

	pc.classHeader = header
	pc.className = extractCppClassName(trimmed)
	pc.classStartLine = lineNum
	pc.classDocComment = pc.docComment
	pc.classFields = nil
	pc.classMethods = nil
	pc.docComment = ""

	if braces > 0 {
		pc.braceDepth = braces
		pc.state = cppStateInClassBody
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty class on one line: class Foo {};
		p.emitClassSignature(pc, lineNum)
	}
	// Forward declaration like "class Foo;" is handled by isCppClassForwardDecl.
}

// handleClassBody processes lines inside a class body.
func (p *cppParser) handleClassBody(pc *cppParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += cCountBraces(trimmed)

	// Class closed.
	if pc.braceDepth <= 0 {
		p.emitClassSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = cppStateTopLevel
		return
	}

	// Only extract members at depth 1 (the class body level).
	if prevDepth != 1 {
		return
	}

	// Skip empty lines.
	if trimmed == "" {
		return
	}

	// Skip single-line comments inside class body.
	if isCLineComment(trimmed) {
		return
	}

	// Doc comments inside class body.
	if isCDocCommentStart(trimmed) {
		if strings.Contains(trimmed, "*/") {
			pc.docComment = line
		}
		return
	}

	// Access specifiers: public:, private:, protected:
	if isCppAccessSpecifier(trimmed) {
		pc.classFields = append(pc.classFields, line)
		return
	}

	// Method declaration or definition.
	if isCppClassMethodDecl(trimmed) {
		sigText := extractCppClassMethodSig(line, trimmed)
		if pc.docComment != "" {
			sigText = pc.docComment + "\n" + sigText
			pc.docComment = ""
		}
		pc.classMethods = append(pc.classMethods, sigText)
		return
	}

	// Constructor/destructor.
	if isCppCtorDtor(trimmed, pc.className) {
		sigText := extractCppClassMethodSig(line, trimmed)
		if pc.docComment != "" {
			sigText = pc.docComment + "\n" + sigText
			pc.docComment = ""
		}
		pc.classMethods = append(pc.classMethods, sigText)
		return
	}

	// Friend declaration.
	if strings.HasPrefix(trimmed, "friend ") {
		pc.classFields = append(pc.classFields, line)
		pc.docComment = ""
		return
	}

	// Using declaration inside class.
	if isCppUsingDecl(trimmed) {
		pc.classFields = append(pc.classFields, line)
		pc.docComment = ""
		return
	}

	// Nested type (enum, struct, class, typedef) -- emit as field.
	if strings.HasPrefix(trimmed, "enum ") || strings.HasPrefix(trimmed, "struct ") ||
		strings.HasPrefix(trimmed, "typedef ") || strings.HasPrefix(trimmed, "class ") {
		pc.classFields = append(pc.classFields, line)
		pc.docComment = ""
		return
	}

	// Field declaration (ends with ; and no parens -- not a method).
	if strings.HasSuffix(trimmed, ";") && !strings.Contains(trimmed, "(") {
		fieldLine := line
		if pc.docComment != "" {
			fieldLine = pc.docComment + "\n" + line
			pc.docComment = ""
		}
		pc.classFields = append(pc.classFields, fieldLine)
		return
	}

	// Everything else that has parens might be a method.
	if strings.Contains(trimmed, "(") {
		sigText := extractCppClassMethodSig(line, trimmed)
		if pc.docComment != "" {
			sigText = pc.docComment + "\n" + sigText
			pc.docComment = ""
		}
		pc.classMethods = append(pc.classMethods, sigText)
		return
	}

	// Catch-all for other lines (e.g., static_assert, etc.).
	pc.classFields = append(pc.classFields, line)
	pc.docComment = ""
}

// emitClassSignature builds and emits a class signature.
func (p *cppParser) emitClassSignature(pc *cppParseCtx, endLine int) {
	var b strings.Builder
	headerTrimmed := strings.TrimRight(pc.classHeader, " \t\n\r")
	if !strings.HasSuffix(headerTrimmed, "{") {
		headerTrimmed += " {"
	}
	b.WriteString(headerTrimmed)

	for _, f := range pc.classFields {
		b.WriteString("\n")
		b.WriteString(f)
	}
	for _, m := range pc.classMethods {
		b.WriteString("\n\n")
		b.WriteString(m)
	}
	b.WriteString("\n};")

	sig := Signature{
		Kind:      KindClass,
		Name:      pc.className,
		Source:    maybePrependDoc(pc.classDocComment, nil, b.String()),
		StartLine: pc.classStartLine,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)

	pc.classHeader = ""
	pc.className = ""
	pc.classDocComment = ""
	pc.classFields = nil
	pc.classMethods = nil
}

// ---------------------------------------------------------------------------
// Namespace handler
// ---------------------------------------------------------------------------

// handleNamespaceDecl processes a namespace declaration.
func (p *cppParser) handleNamespaceDecl(pc *cppParseCtx, trimmed, line string, lineNum int) {
	braces := cCountBraces(trimmed)

	pc.nsHeader = line
	pc.nsName = extractCppNamespaceName(trimmed)
	pc.nsStartLine = lineNum
	pc.nsMembers = nil
	pc.docComment = ""

	if braces > 0 {
		pc.nsBraceBase = 0
		pc.braceDepth = braces
		pc.state = cppStateInNamespace
	} else if braces == 0 && strings.Contains(trimmed, "{") {
		// Empty namespace on one line.
		p.emitNamespaceSignature(pc, lineNum)
	}
}

// handleNamespace processes lines inside a namespace body.
func (p *cppParser) handleNamespace(pc *cppParseCtx, trimmed, line string, lineNum int) {
	prevDepth := pc.braceDepth
	pc.braceDepth += cCountBraces(trimmed)

	// Namespace closed.
	if pc.braceDepth <= 0 {
		p.emitNamespaceSignature(pc, lineNum)
		pc.braceDepth = 0
		pc.state = cppStateTopLevel
		return
	}

	// Only extract members at depth 1 (namespace body level).
	if prevDepth != 1 {
		return
	}

	if trimmed == "" || isCLineComment(trimmed) {
		return
	}

	// Collect top-level declarations inside the namespace.
	// We record structural lines: function signatures, class forward decls, etc.
	if isCppClassDecl(trimmed) || isCStructDecl(trimmed) || isCppEnumDecl(trimmed) ||
		isCppNamespaceDecl(trimmed) || isCFuncPrototype(trimmed) ||
		isCppUsingDecl(trimmed) || isCTypedef(trimmed) ||
		isCppTemplateDecl(trimmed) {
		pc.nsMembers = append(pc.nsMembers, line)
		return
	}

	// Function definitions: extract just the signature.
	if isCFuncDefinition(trimmed) || isCppMethodDefinition(trimmed) {
		sigText := extractCFuncSignature(line)
		pc.nsMembers = append(pc.nsMembers, sigText)
		return
	}

	// Constants and variables.
	if isCGlobalVarDecl(trimmed) || isCForwardDecl(trimmed) || isCppClassForwardDecl(trimmed) {
		pc.nsMembers = append(pc.nsMembers, line)
	}
}

// emitNamespaceSignature builds and emits a namespace signature.
func (p *cppParser) emitNamespaceSignature(pc *cppParseCtx, endLine int) {
	var b strings.Builder
	headerTrimmed := strings.TrimRight(pc.nsHeader, " \t\n\r")
	if !strings.HasSuffix(headerTrimmed, "{") {
		headerTrimmed += " {"
	}
	b.WriteString(headerTrimmed)

	for _, m := range pc.nsMembers {
		b.WriteString("\n")
		b.WriteString(m)
	}
	b.WriteString("\n}")

	sig := Signature{
		Kind:      KindType,
		Name:      pc.nsName,
		Source:    b.String(),
		StartLine: pc.nsStartLine,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)

	pc.nsHeader = ""
	pc.nsName = ""
	pc.nsMembers = nil
}

// ---------------------------------------------------------------------------
// C++-specific detection helpers
// ---------------------------------------------------------------------------

// isCppClassDecl checks if a trimmed line declares a class.
func isCppClassDecl(trimmed string) bool {
	// Ensure it is not "class;" (forward decl) -- that is handled separately.
	if strings.HasSuffix(trimmed, ";") && !strings.Contains(trimmed, "{") {
		return false
	}
	s := trimmed
	// Remove qualifiers.
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	return strings.HasPrefix(s, "class ") && strings.Contains(s, "{")
}

// isCppClassForwardDecl checks for class forward declarations: "class Foo;".
func isCppClassForwardDecl(trimmed string) bool {
	if !strings.HasSuffix(trimmed, ";") {
		return false
	}
	s := trimmed
	s = strings.TrimPrefix(s, "extern ")
	return strings.HasPrefix(s, "class ") && !strings.Contains(s, "{") &&
		!strings.Contains(s, "(")
}

// isCppTemplateDecl checks if a trimmed line is a template declaration prefix.
func isCppTemplateDecl(trimmed string) bool {
	return strings.HasPrefix(trimmed, "template<") || strings.HasPrefix(trimmed, "template <")
}

// isCppNamespaceDecl checks if a trimmed line declares a namespace.
func isCppNamespaceDecl(trimmed string) bool {
	if strings.HasPrefix(trimmed, "namespace ") {
		return true
	}
	// Inline namespace.
	if strings.HasPrefix(trimmed, "inline namespace ") {
		return true
	}
	return false
}

// isCppUsingDecl checks for using declarations and directives.
func isCppUsingDecl(trimmed string) bool {
	return strings.HasPrefix(trimmed, "using ")
}

// isCppEnumDecl detects enum declarations including enum class.
func isCppEnumDecl(trimmed string) bool {
	if isCEnumDecl(trimmed) {
		return true
	}
	s := trimmed
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	return strings.HasPrefix(s, "enum class ") || strings.HasPrefix(s, "enum struct ")
}

// isCppAccessSpecifier checks for public:, private:, protected:.
func isCppAccessSpecifier(trimmed string) bool {
	return trimmed == "public:" || trimmed == "private:" || trimmed == "protected:" ||
		strings.HasPrefix(trimmed, "public :") || strings.HasPrefix(trimmed, "private :") ||
		strings.HasPrefix(trimmed, "protected :")
}

// isCppClassMethodDecl checks if a trimmed line inside a class body is a method
// declaration or definition.
func isCppClassMethodDecl(trimmed string) bool {
	if !strings.Contains(trimmed, "(") {
		return false
	}
	// Must not be a control keyword.
	for _, kw := range cControlKeywords {
		if strings.HasPrefix(trimmed, kw+" ") || strings.HasPrefix(trimmed, kw+"(") {
			return false
		}
	}
	return true
}

// isCppCtorDtor checks if a line is a constructor or destructor declaration.
func isCppCtorDtor(trimmed, className string) bool {
	if className == "" {
		return false
	}
	// Constructor: ClassName(
	if strings.HasPrefix(trimmed, className+"(") {
		return true
	}
	// Destructor: ~ClassName( or virtual ~ClassName(
	if strings.HasPrefix(trimmed, "~"+className+"(") {
		return true
	}
	if strings.HasPrefix(trimmed, "virtual ~"+className+"(") {
		return true
	}
	// Qualified variants with access specifiers already stripped.
	s := trimmed
	for _, prefix := range []string{"explicit ", "virtual ", "inline ", "constexpr "} {
		s = strings.TrimPrefix(s, prefix)
	}
	return strings.HasPrefix(s, className+"(") || strings.HasPrefix(s, "~"+className+"(")
}

// isCppMethodDefinition checks for out-of-class method definitions:
// ReturnType ClassName::method(params) {
func isCppMethodDefinition(trimmed string) bool {
	if !strings.Contains(trimmed, "::") {
		return false
	}
	if !strings.Contains(trimmed, "(") || !strings.Contains(trimmed, "{") {
		return false
	}
	for _, kw := range cControlKeywords {
		if strings.HasPrefix(trimmed, kw+" ") || strings.HasPrefix(trimmed, kw+"(") {
			return false
		}
	}
	parenClose := strings.LastIndex(trimmed, ")")
	braceOpen := strings.Index(trimmed, "{")
	return parenClose != -1 && braceOpen > parenClose
}

// isCppMethodPrototype checks for out-of-class method prototypes:
// ReturnType ClassName::method(params);
func isCppMethodPrototype(trimmed string) bool {
	if !strings.Contains(trimmed, "::") {
		return false
	}
	if !strings.Contains(trimmed, "(") || !strings.HasSuffix(trimmed, ";") {
		return false
	}
	for _, kw := range cControlKeywords {
		if strings.HasPrefix(trimmed, kw+" ") || strings.HasPrefix(trimmed, kw+"(") {
			return false
		}
	}
	semiIdx := strings.LastIndex(trimmed, ";")
	parenClose := strings.LastIndex(trimmed, ")")
	return parenClose != -1 && parenClose < semiIdx
}

// ---------------------------------------------------------------------------
// C++-specific extraction helpers
// ---------------------------------------------------------------------------

// extractCppClassName extracts the class name from a class declaration line.
func extractCppClassName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	s = strings.TrimPrefix(s, "class ")
	s = strings.TrimSpace(s)
	// Name is up to the first space, <, :, or {.
	return extractCIdentifier(s)
}

// extractCppNamespaceName extracts the namespace name.
// Handles "namespace foo {", "inline namespace foo {", "namespace a::b::c {".
func extractCppNamespaceName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "inline ")
	s = strings.TrimPrefix(s, "namespace ")
	s = strings.TrimSpace(s)
	// Name goes up to {.
	if idx := strings.Index(s, "{"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	return s
}

// extractCppEnumName extracts the enum name, handling "enum class Color : int {".
func extractCppEnumName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "typedef ")
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	s = strings.TrimPrefix(s, "enum struct ")
	s = strings.TrimPrefix(s, "enum class ")
	s = strings.TrimPrefix(s, "enum ")
	s = strings.TrimSpace(s)
	return extractCIdentifier(s)
}

// extractCppClassMethodSig extracts a method signature from within a class body.
// If the method has a body on the same line, extract up to { and add { ... }.
func extractCppClassMethodSig(line, trimmed string) string {
	braceIdx := strings.Index(trimmed, "{")
	if braceIdx != -1 {
		lineBraceIdx := strings.Index(line, "{")
		if lineBraceIdx != -1 {
			sig := strings.TrimRight(line[:lineBraceIdx], " \t")
			return sig + " { ... }"
		}
	}
	return strings.TrimRight(line, " \t")
}