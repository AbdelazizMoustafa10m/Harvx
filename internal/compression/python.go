package compression

import (
	"context"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*PythonCompressor)(nil)

// PythonCompressor implements LanguageCompressor for Python source code.
// It uses a line-by-line state machine parser to extract structural signatures
// including functions, classes, imports, decorators, docstrings, and type-annotated
// top-level assignments. The compressor is stateless and safe for concurrent use.
type PythonCompressor struct{}

// NewPythonCompressor creates a Python compressor.
func NewPythonCompressor() *PythonCompressor {
	return &PythonCompressor{}
}

// Compress parses Python source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *PythonCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	p := &pyParser{}
	return p.parse(ctx, source)
}

// Language returns "python".
func (c *PythonCompressor) Language() string {
	return "python"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *PythonCompressor) SupportedNodeTypes() []string {
	return []string{
		"import_statement",
		"import_from_statement",
		"function_definition",
		"async_function_definition",
		"class_definition",
		"decorated_definition",
		"expression_statement",
		"type_alias_statement",
	}
}

// ---------------------------------------------------------------------------
// Python parser state machine
// ---------------------------------------------------------------------------

// pyState tracks the current state of the Python line-by-line parser.
type pyState int

const (
	pyStateTopLevel       pyState = iota // Scanning for declarations at indent level 0
	pyStateInDecorator                   // Accumulating decorator lines
	pyStateInFuncSig                     // Accumulating multi-line function signature
	pyStateInFuncBody                    // Skipping function body lines
	pyStateInClassBody                   // Extracting class members
	pyStateInMethodSig                   // Accumulating multi-line method signature inside class
	pyStateInMethodBody                  // Skipping method body inside class
	pyStateInDocstring                   // Accumulating a multi-line docstring
	pyStateInClassDocstring              // Accumulating a multi-line docstring inside a class
)

// pyParseCtx holds all mutable state for the Python line-by-line parser.
type pyParseCtx struct {
	state      pyState
	sigs       []Signature
	decorators []string // accumulated decorator lines

	// Function/method signature accumulation (for multi-line signatures).
	sigAccum      strings.Builder
	sigStartLine  int
	sigParenDepth int

	// Docstring accumulation.
	docAccum     strings.Builder
	docStartLine int
	docDelimiter string // `"""` or `'''`

	// Function body tracking.
	funcIndent int // indentation of the def/async def line
	funcSig    string
	funcName   string
	funcStart  int

	// Class tracking.
	classIndent     int // indentation of the class line
	classHeader     string
	className       string
	classStartLine  int
	classDocComment string
	classDecorators []string
	classFields     []string
	classMethods    []string

	// Method tracking inside class.
	methodIndent     int
	methodSig        string
	methodName       string
	methodStart      int
	methodDecorators []string
	methodDocComment string

	// Module docstring tracking.
	seenNonBlank  bool // true once we've seen any non-blank, non-comment line
	moduleDocDone bool // true once module docstring opportunity has passed

	// Pending docstring to attach to the next declaration.
	pendingDocstring string
}

// pyParser provides Python source parsing using a line-by-line state machine.
type pyParser struct{}

// parse extracts structural signatures from Python source code.
func (p *pyParser) parse(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "python",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	pc := &pyParseCtx{
		state: pyStateTopLevel,
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
		case pyStateTopLevel:
			p.handleTopLevel(pc, trimmed, line, lineNum)

		case pyStateInDecorator:
			p.handleDecorator(pc, trimmed, line, lineNum)

		case pyStateInFuncSig:
			p.handleFuncSig(pc, trimmed, line, lineNum)

		case pyStateInFuncBody:
			p.handleFuncBody(pc, trimmed, line, lineNum)

		case pyStateInClassBody:
			p.handleClassBody(pc, trimmed, line, lineNum)

		case pyStateInMethodSig:
			p.handleMethodSig(pc, trimmed, line, lineNum)

		case pyStateInMethodBody:
			p.handleMethodBody(pc, trimmed, line, lineNum)

		case pyStateInDocstring:
			p.handleDocstring(pc, trimmed, line, lineNum)

		case pyStateInClassDocstring:
			p.handleClassDocstring(pc, trimmed, line, lineNum)
		}
	}

	// Flush any pending state at EOF.
	p.flushPending(pc, len(lines))

	output := &CompressedOutput{
		Signatures:   pc.sigs,
		Language:     "python",
		OriginalSize: len(source),
		NodeCount:    len(pc.sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// flushPending emits any accumulated state at EOF.
func (p *pyParser) flushPending(pc *pyParseCtx, lastLine int) {
	switch pc.state {
	case pyStateInFuncBody:
		// Emit the function signature (body was being skipped).
		p.emitFuncSig(pc, lastLine)

	case pyStateInClassBody, pyStateInMethodBody:
		// If we were in a method body, emit the method first.
		if pc.state == pyStateInMethodBody {
			p.emitMethodSig(pc)
		}
		// Emit the class.
		p.emitClass(pc, lastLine)

	case pyStateInFuncSig:
		// Unterminated multi-line function signature.
		sig := pc.sigAccum.String()
		pc.sigs = append(pc.sigs, Signature{
			Kind:      KindFunction,
			Name:      pc.funcName,
			Source:    maybePrependDoc(pc.pendingDocstring, pc.decorators, sig),
			StartLine: pc.sigStartLine,
			EndLine:   lastLine,
		})
		pc.decorators = nil
		pc.pendingDocstring = ""

	case pyStateInMethodSig:
		// Unterminated multi-line method signature inside class -- emit class.
		p.emitClass(pc, lastLine)
	}
}

// ---------------------------------------------------------------------------
// Top-level line handling
// ---------------------------------------------------------------------------

// handleTopLevel processes a line when the parser is at the top level (indent 0).
func (p *pyParser) handleTopLevel(pc *pyParseCtx, trimmed, line string, lineNum int) {
	indent := pyLineIndent(line)

	// Skip empty lines.
	if trimmed == "" {
		return
	}

	// Skip comment-only lines (don't count as "seen non-blank" for module docstring).
	if strings.HasPrefix(trimmed, "#") {
		return
	}

	// Check for module-level docstring (must be the first non-blank, non-comment thing).
	if !pc.moduleDocDone && !pc.seenNonBlank {
		if delim, ok := pyTripleQuoteStart(trimmed); ok {
			pc.seenNonBlank = true
			pc.moduleDocDone = true

			// Check if the docstring closes on the same line.
			if pyDocstringClosesOnLine(trimmed, delim) {
				pc.sigs = append(pc.sigs, Signature{
					Kind:      KindDocComment,
					Source:    line,
					StartLine: lineNum,
					EndLine:   lineNum,
				})
				return
			}
			// Multi-line module docstring.
			pc.docAccum.Reset()
			pc.docAccum.WriteString(line)
			pc.docStartLine = lineNum
			pc.docDelimiter = delim
			pc.state = pyStateInDocstring
			return
		}
	}

	pc.seenNonBlank = true
	pc.moduleDocDone = true

	// Import statements.
	if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
		pc.sigs = append(pc.sigs, Signature{
			Kind:      KindImport,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		})
		pc.pendingDocstring = ""
		return
	}

	// Decorator.
	if strings.HasPrefix(trimmed, "@") && indent == 0 {
		pc.decorators = append(pc.decorators, line)
		pc.state = pyStateInDecorator
		return
	}

	// Function definition (def or async def).
	if indent == 0 && (strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ")) {
		p.startFuncDef(pc, trimmed, line, lineNum, indent)
		return
	}

	// Class definition.
	if indent == 0 && strings.HasPrefix(trimmed, "class ") {
		p.startClassDef(pc, trimmed, line, lineNum, indent)
		return
	}

	// __all__ assignment.
	if indent == 0 && pyIsAllAssignment(trimmed) {
		pc.sigs = append(pc.sigs, Signature{
			Kind:      KindExport,
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		})
		pc.pendingDocstring = ""
		pc.decorators = nil
		return
	}

	// Top-level type-annotated assignment: NAME: type = value
	if indent == 0 && pyIsTypeAnnotatedAssignment(trimmed) {
		pc.sigs = append(pc.sigs, Signature{
			Kind:      KindConstant,
			Name:      pyExtractAssignmentName(trimmed),
			Source:    line,
			StartLine: lineNum,
			EndLine:   lineNum,
		})
		pc.pendingDocstring = ""
		pc.decorators = nil
		return
	}

	// Anything else at top level -- discard pending state.
	pc.decorators = nil
	pc.pendingDocstring = ""
}

// ---------------------------------------------------------------------------
// Decorator handling
// ---------------------------------------------------------------------------

// handleDecorator continues accumulating decorator lines until a def/class is found.
func (p *pyParser) handleDecorator(pc *pyParseCtx, trimmed, line string, lineNum int) {
	if trimmed == "" {
		return
	}

	// Another decorator.
	if strings.HasPrefix(trimmed, "@") {
		pc.decorators = append(pc.decorators, line)
		return
	}

	// Comment line between decorators.
	if strings.HasPrefix(trimmed, "#") {
		return
	}

	indent := pyLineIndent(line)

	// Function definition.
	if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
		p.startFuncDef(pc, trimmed, line, lineNum, indent)
		return
	}

	// Class definition.
	if strings.HasPrefix(trimmed, "class ") {
		p.startClassDef(pc, trimmed, line, lineNum, indent)
		return
	}

	// Unexpected line after decorator -- discard decorators.
	pc.decorators = nil
	pc.state = pyStateTopLevel
	p.handleTopLevel(pc, trimmed, line, lineNum)
}

// ---------------------------------------------------------------------------
// Function definition handling
// ---------------------------------------------------------------------------

// startFuncDef begins processing a function definition.
func (p *pyParser) startFuncDef(pc *pyParseCtx, trimmed, line string, lineNum, indent int) {
	pc.funcIndent = indent
	pc.funcName = pyExtractFuncName(trimmed)
	pc.funcStart = lineNum

	// Check if signature is complete on this line (has closing paren and colon).
	parenDepth := pyCountParens(trimmed)
	if parenDepth <= 0 && strings.HasSuffix(trimmed, ":") {
		// Single-line signature. Extract up to and including the colon.
		pc.funcSig = line
		pc.state = pyStateInFuncBody
		return
	}

	// Multi-line signature -- accumulate until balanced parens and colon.
	pc.sigAccum.Reset()
	pc.sigAccum.WriteString(line)
	pc.sigStartLine = lineNum
	pc.sigParenDepth = parenDepth
	pc.state = pyStateInFuncSig
}

// handleFuncSig accumulates multi-line function signatures.
func (p *pyParser) handleFuncSig(pc *pyParseCtx, trimmed, line string, lineNum int) {
	pc.sigAccum.WriteString("\n")
	pc.sigAccum.WriteString(line)
	pc.sigParenDepth += pyCountParens(trimmed)

	if pc.sigParenDepth <= 0 && strings.HasSuffix(trimmed, ":") {
		pc.funcSig = pc.sigAccum.String()
		pc.sigAccum.Reset()
		pc.state = pyStateInFuncBody
	}
}

// handleFuncBody skips function body lines based on indentation.
func (p *pyParser) handleFuncBody(pc *pyParseCtx, trimmed, line string, lineNum int) {
	// Empty or comment-only lines don't end the function body.
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return
	}

	indent := pyLineIndent(line)

	// Check for docstring as first statement in the body.
	if pc.funcSig != "" && indent > pc.funcIndent {
		// Only capture docstring on the very first non-empty body line.
		if !pyFuncHasDocstring(pc) {
			if delim, ok := pyTripleQuoteStart(trimmed); ok {
				if pyDocstringClosesOnLine(trimmed, delim) {
					pc.funcSig += "\n" + line
					return
				}
				// Multi-line docstring inside function.
				pc.docAccum.Reset()
				pc.docAccum.WriteString(line)
				pc.docStartLine = lineNum
				pc.docDelimiter = delim
				pc.state = pyStateInDocstring
				return
			}
			// First body line is not a docstring; mark that we've passed that opportunity.
			// We use a sentinel by appending a NUL to funcSig -- not ideal but functional.
			// Instead, we set a flag using the pending docstring field.
			pc.pendingDocstring = "\x00" // sentinel: docstring opportunity passed
		}
	}

	// If indentation is <= the function's indent, the body has ended.
	if indent <= pc.funcIndent {
		p.emitFuncSig(pc, lineNum-1)
		pc.state = pyStateTopLevel
		p.handleTopLevel(pc, trimmed, line, lineNum)
		return
	}
}

// pyFuncHasDocstring checks whether a docstring has already been attached.
func pyFuncHasDocstring(pc *pyParseCtx) bool {
	// If pendingDocstring is the sentinel, opportunity passed.
	if pc.pendingDocstring == "\x00" {
		return true
	}
	// Check if funcSig already has a triple-quote in it beyond the first line.
	lines := strings.Split(pc.funcSig, "\n")
	if len(lines) <= 1 {
		return false
	}
	// Check the second-to-last and last appended parts.
	for i := 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.Contains(t, `"""`) || strings.Contains(t, `'''`) {
			return true
		}
	}
	return false
}

// emitFuncSig emits the accumulated function signature.
func (p *pyParser) emitFuncSig(pc *pyParseCtx, endLine int) {
	source := pc.funcSig
	doc := ""
	if pc.pendingDocstring != "" && pc.pendingDocstring != "\x00" {
		doc = pc.pendingDocstring
	}
	sig := Signature{
		Kind:      KindFunction,
		Name:      pc.funcName,
		Source:    maybePrependDoc(doc, pc.decorators, source),
		StartLine: pc.funcStart,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.decorators = nil
	pc.pendingDocstring = ""
	pc.funcSig = ""
}

// ---------------------------------------------------------------------------
// Class definition handling
// ---------------------------------------------------------------------------

// startClassDef begins processing a class definition.
func (p *pyParser) startClassDef(pc *pyParseCtx, trimmed, line string, lineNum, indent int) {
	pc.classIndent = indent
	pc.className = pyExtractClassName(trimmed)
	pc.classStartLine = lineNum
	pc.classHeader = line
	pc.classDocComment = ""
	pc.classDecorators = make([]string, len(pc.decorators))
	copy(pc.classDecorators, pc.decorators)
	pc.classFields = nil
	pc.classMethods = nil
	pc.decorators = nil
	pc.pendingDocstring = ""
	pc.methodDecorators = nil
	pc.methodDocComment = ""
	pc.state = pyStateInClassBody
}

// handleClassBody processes lines inside a class body.
func (p *pyParser) handleClassBody(pc *pyParseCtx, trimmed, line string, lineNum int) {
	// Empty lines are fine in class body.
	if trimmed == "" {
		return
	}

	// Comments inside class body -- skip.
	if strings.HasPrefix(trimmed, "#") {
		return
	}

	indent := pyLineIndent(line)

	// If indentation is <= class indent, the class body has ended.
	if indent <= pc.classIndent {
		p.emitClass(pc, lineNum-1)
		pc.state = pyStateTopLevel
		p.handleTopLevel(pc, trimmed, line, lineNum)
		return
	}

	// Class docstring (first non-empty line after class header).
	if pc.classDocComment == "" && len(pc.classFields) == 0 && len(pc.classMethods) == 0 {
		if delim, ok := pyTripleQuoteStart(trimmed); ok {
			if pyDocstringClosesOnLine(trimmed, delim) {
				pc.classDocComment = line
				return
			}
			// Multi-line class docstring.
			pc.docAccum.Reset()
			pc.docAccum.WriteString(line)
			pc.docStartLine = lineNum
			pc.docDelimiter = delim
			pc.state = pyStateInClassDocstring
			return
		}
	}

	// Decorator inside class (for method).
	if strings.HasPrefix(trimmed, "@") {
		pc.methodDecorators = append(pc.methodDecorators, line)
		return
	}

	// Method definition (def or async def inside class).
	if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
		p.startMethodDef(pc, trimmed, line, lineNum, indent)
		return
	}

	// Class-level field/assignment: indented, contains ':' or '=' but not a function call.
	// For dataclass fields: name: type = value
	if indent > pc.classIndent && pyIsClassField(trimmed) {
		pc.classFields = append(pc.classFields, line)
		return
	}
}

// startMethodDef begins processing a method definition inside a class.
func (p *pyParser) startMethodDef(pc *pyParseCtx, trimmed, line string, lineNum, indent int) {
	pc.methodIndent = indent
	pc.methodName = pyExtractFuncName(trimmed)
	pc.methodStart = lineNum
	pc.methodDocComment = ""

	// Check if signature is complete on this line.
	parenDepth := pyCountParens(trimmed)
	if parenDepth <= 0 && strings.HasSuffix(trimmed, ":") {
		pc.methodSig = line
		pc.state = pyStateInMethodBody
		return
	}

	// Multi-line method signature.
	pc.sigAccum.Reset()
	pc.sigAccum.WriteString(line)
	pc.sigStartLine = lineNum
	pc.sigParenDepth = parenDepth
	pc.state = pyStateInMethodSig
}

// handleMethodSig accumulates multi-line method signatures.
func (p *pyParser) handleMethodSig(pc *pyParseCtx, trimmed, line string, lineNum int) {
	pc.sigAccum.WriteString("\n")
	pc.sigAccum.WriteString(line)
	pc.sigParenDepth += pyCountParens(trimmed)

	if pc.sigParenDepth <= 0 && strings.HasSuffix(trimmed, ":") {
		pc.methodSig = pc.sigAccum.String()
		pc.sigAccum.Reset()
		pc.state = pyStateInMethodBody
	}
}

// handleMethodBody skips method body lines, capturing docstring.
func (p *pyParser) handleMethodBody(pc *pyParseCtx, trimmed, line string, lineNum int) {
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return
	}

	indent := pyLineIndent(line)

	// Capture docstring on first non-empty body line.
	if pc.methodDocComment == "" && indent > pc.methodIndent {
		if delim, ok := pyTripleQuoteStart(trimmed); ok {
			if pyDocstringClosesOnLine(trimmed, delim) {
				pc.methodDocComment = line
				return
			}
			// Multi-line docstring inside method -- we still need to skip it.
			pc.docAccum.Reset()
			pc.docAccum.WriteString(line)
			pc.docStartLine = lineNum
			pc.docDelimiter = delim
			pc.state = pyStateInClassDocstring
			return
		}
		// Not a docstring -- mark as done by setting a placeholder.
		if pc.methodDocComment == "" {
			pc.methodDocComment = "\x00"
		}
	}

	// If indentation is <= method indent, method body ended.
	if indent <= pc.methodIndent {
		p.emitMethodSig(pc)

		// If indentation is also <= class indent, class ended too.
		if indent <= pc.classIndent {
			p.emitClass(pc, lineNum-1)
			pc.state = pyStateTopLevel
			p.handleTopLevel(pc, trimmed, line, lineNum)
			return
		}

		// Still in class body.
		pc.state = pyStateInClassBody
		p.handleClassBody(pc, trimmed, line, lineNum)
		return
	}
}

// emitMethodSig adds the accumulated method signature to class methods.
func (p *pyParser) emitMethodSig(pc *pyParseCtx) {
	var parts []string
	for _, d := range pc.methodDecorators {
		parts = append(parts, d)
	}
	parts = append(parts, pc.methodSig)
	if pc.methodDocComment != "" && pc.methodDocComment != "\x00" {
		parts = append(parts, pc.methodDocComment)
	}
	pc.classMethods = append(pc.classMethods, strings.Join(parts, "\n"))
	pc.methodDecorators = nil
	pc.methodSig = ""
	pc.methodDocComment = ""
}

// emitClass emits the accumulated class signature.
func (p *pyParser) emitClass(pc *pyParseCtx, endLine int) {
	var b strings.Builder
	b.WriteString(pc.classHeader)

	if pc.classDocComment != "" {
		b.WriteString("\n")
		b.WriteString(pc.classDocComment)
	}

	for _, f := range pc.classFields {
		b.WriteString("\n")
		b.WriteString(f)
	}

	for _, m := range pc.classMethods {
		b.WriteString("\n\n")
		b.WriteString(m)
	}

	sig := Signature{
		Kind:      KindClass,
		Name:      pc.className,
		Source:    maybePrependDoc("", pc.classDecorators, b.String()),
		StartLine: pc.classStartLine,
		EndLine:   endLine,
	}
	pc.sigs = append(pc.sigs, sig)
	pc.classDecorators = nil
	pc.classFields = nil
	pc.classMethods = nil
}

// ---------------------------------------------------------------------------
// Docstring handling
// ---------------------------------------------------------------------------

// handleDocstring accumulates multi-line docstring at the top level or inside a function.
func (p *pyParser) handleDocstring(pc *pyParseCtx, trimmed, line string, lineNum int) {
	pc.docAccum.WriteString("\n")
	pc.docAccum.WriteString(line)

	if pyDocstringEndsWith(trimmed, pc.docDelimiter) {
		docText := pc.docAccum.String()

		if pc.state == pyStateInDocstring {
			// Were we in a function body capturing docstring?
			if pc.funcSig != "" {
				// Attach docstring to function signature.
				pc.funcSig += "\n" + docText
				pc.docAccum.Reset()
				pc.state = pyStateInFuncBody
				return
			}
			// Module-level docstring.
			pc.sigs = append(pc.sigs, Signature{
				Kind:      KindDocComment,
				Source:    docText,
				StartLine: pc.docStartLine,
				EndLine:   lineNum,
			})
			pc.docAccum.Reset()
			pc.state = pyStateTopLevel
		}
	}
}

// handleClassDocstring accumulates multi-line docstring inside a class or method.
func (p *pyParser) handleClassDocstring(pc *pyParseCtx, trimmed, line string, lineNum int) {
	pc.docAccum.WriteString("\n")
	pc.docAccum.WriteString(line)

	if pyDocstringEndsWith(trimmed, pc.docDelimiter) {
		docText := pc.docAccum.String()
		pc.docAccum.Reset()

		// Were we in a method body capturing docstring?
		if pc.methodSig != "" {
			pc.methodDocComment = docText
			pc.state = pyStateInMethodBody
			return
		}

		// Class-level docstring.
		pc.classDocComment = docText
		pc.state = pyStateInClassBody
	}
}

// ---------------------------------------------------------------------------
// Python-specific helper functions
// ---------------------------------------------------------------------------

// pyLineIndent returns the number of leading spaces in a line.
// Tabs are counted as 1 character (consistent with Python's basic indent tracking).
func pyLineIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4 // treat tabs as 4 spaces
		} else {
			break
		}
	}
	return count
}

// pyTripleQuoteStart checks if a trimmed line starts with a triple quote.
// Returns the delimiter (`"""` or `'''`) and true if found.
func pyTripleQuoteStart(trimmed string) (string, bool) {
	if strings.HasPrefix(trimmed, `"""`) {
		return `"""`, true
	}
	if strings.HasPrefix(trimmed, `'''`) {
		return `'''`, true
	}
	return "", false
}

// pyDocstringClosesOnLine checks if a triple-quoted string opens and closes on the same line.
func pyDocstringClosesOnLine(trimmed, delim string) bool {
	// After the opening delimiter, look for the closing delimiter.
	rest := trimmed[len(delim):]
	return strings.Contains(rest, delim)
}

// pyDocstringEndsWith checks if a line ends the current docstring.
func pyDocstringEndsWith(trimmed, delim string) bool {
	return strings.Contains(trimmed, delim)
}

// pyCountParens counts net parenthesis depth for a line.
func pyCountParens(line string) int {
	depth := 0
	inSingleQuote := false
	inDoubleQuote := false
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
		default:
			switch ch {
			case '\'':
				inSingleQuote = true
			case '"':
				inDoubleQuote = true
			case '(':
				depth++
			case ')':
				depth--
			case '#':
				// Rest of line is a comment.
				return depth
			}
		}
	}
	return depth
}

// pyExtractFuncName extracts the function name from a def or async def line.
func pyExtractFuncName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "async ")
	s = strings.TrimPrefix(s, "def ")
	// Name is everything up to '('.
	if idx := strings.Index(s, "("); idx != -1 {
		return s[:idx]
	}
	if idx := strings.Index(s, ":"); idx != -1 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

// pyExtractClassName extracts the class name from a class definition line.
func pyExtractClassName(trimmed string) string {
	s := strings.TrimPrefix(trimmed, "class ")
	// Name is everything up to '(' or ':'.
	for i, ch := range s {
		if ch == '(' || ch == ':' {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

// pyIsTypeAnnotatedAssignment checks if a trimmed line is a type-annotated assignment.
// Pattern: IDENTIFIER: Type = value  or  IDENTIFIER: Type
func pyIsTypeAnnotatedAssignment(trimmed string) bool {
	// Must start with an identifier character.
	if len(trimmed) == 0 {
		return false
	}
	firstCh := rune(trimmed[0])
	if !pyIsIdentStart(firstCh) {
		return false
	}

	// Find the first colon that's not inside brackets.
	colonIdx := pyFindAnnotationColon(trimmed)
	if colonIdx == -1 {
		return false
	}

	// The part before the colon must be a simple identifier.
	name := strings.TrimSpace(trimmed[:colonIdx])
	if !pyIsSimpleIdentifier(name) {
		return false
	}

	// The part after the colon must be non-empty (the type annotation).
	rest := strings.TrimSpace(trimmed[colonIdx+1:])
	return len(rest) > 0
}

// pyFindAnnotationColon finds the index of the type annotation colon,
// skipping colons inside brackets or strings.
func pyFindAnnotationColon(s string) int {
	depth := 0 // bracket depth
	for i, ch := range s {
		switch ch {
		case '[', '(':
			depth++
		case ']', ')':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// pyIsSimpleIdentifier checks if a string is a valid Python identifier.
func pyIsSimpleIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if !pyIsIdentStart(ch) {
				return false
			}
		} else {
			if !pyIsIdentContinue(ch) {
				return false
			}
		}
	}
	return true
}

// pyIsIdentStart checks if a rune can start a Python identifier.
func pyIsIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// pyIsIdentContinue checks if a rune can continue a Python identifier.
func pyIsIdentContinue(ch rune) bool {
	return pyIsIdentStart(ch) || (ch >= '0' && ch <= '9')
}

// pyExtractAssignmentName extracts the name from a type-annotated assignment.
func pyExtractAssignmentName(trimmed string) string {
	colonIdx := pyFindAnnotationColon(trimmed)
	if colonIdx == -1 {
		return ""
	}
	return strings.TrimSpace(trimmed[:colonIdx])
}

// pyIsClassField checks if a trimmed line inside a class is a field/assignment.
func pyIsClassField(trimmed string) bool {
	// Skip method definitions and decorators.
	if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
		return false
	}
	if strings.HasPrefix(trimmed, "@") {
		return false
	}
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	if strings.HasPrefix(trimmed, "class ") {
		return false
	}

	// Must contain a colon (type annotation) or = (assignment).
	// For dataclass fields, we want: name: type = value
	// For regular assignments: name = value
	return pyIsTypeAnnotatedAssignment(trimmed) || pyIsSimpleAssignment(trimmed)
}

// pyIsSimpleAssignment checks for simple top-level assignments like `name = value`
// that might be class variables but don't have type annotations.
// Only captures assignments where the LHS is a simple identifier.
func pyIsSimpleAssignment(trimmed string) bool {
	eqIdx := strings.Index(trimmed, "=")
	if eqIdx == -1 {
		return false
	}
	// Make sure it's not ==, !=, <=, >=, :=
	if eqIdx > 0 && (trimmed[eqIdx-1] == '!' || trimmed[eqIdx-1] == '<' || trimmed[eqIdx-1] == '>' || trimmed[eqIdx-1] == ':') {
		return false
	}
	if eqIdx+1 < len(trimmed) && trimmed[eqIdx+1] == '=' {
		return false
	}

	name := strings.TrimSpace(trimmed[:eqIdx])
	return pyIsSimpleIdentifier(name)
}

// pyIsAllAssignment checks if a trimmed line is an __all__ assignment.
func pyIsAllAssignment(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "__all__") {
		return false
	}
	rest := trimmed[len("__all__"):]
	if len(rest) == 0 {
		return false
	}
	// Next char must be whitespace, '=', ':', or '+' (for __all__ += [...])
	ch := rest[0]
	return ch == ' ' || ch == '=' || ch == ':' || ch == '+' || ch == '\t'
}

