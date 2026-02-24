package compression

// CompressorRegistry manages language compressor implementations.
// It maps language identifiers to their LanguageCompressor implementations
// and uses a LanguageDetector to resolve file paths to languages.
type CompressorRegistry struct {
	compressors map[string]LanguageCompressor
	detector    *LanguageDetector
}

// NewCompressorRegistry creates a registry with the given language detector.
// No compressors are registered by default; call Register to add them.
func NewCompressorRegistry(detector *LanguageDetector) *CompressorRegistry {
	return &CompressorRegistry{
		compressors: make(map[string]LanguageCompressor),
		detector:    detector,
	}
}

// Register adds a compressor for a language. If a compressor is already
// registered for the language, it is replaced.
func (r *CompressorRegistry) Register(c LanguageCompressor) {
	r.compressors[c.Language()] = c
}

// Get returns the compressor for a given file path, or nil if unsupported.
// It uses the language detector to resolve the file extension to a language
// identifier, then looks up the compressor in the registry.
func (r *CompressorRegistry) Get(filePath string) LanguageCompressor {
	lang := r.detector.DetectLanguage(filePath)
	if lang == "" {
		return nil
	}
	return r.compressors[lang]
}

// GetByLanguage returns the compressor for a given language identifier,
// or nil if no compressor is registered for that language.
func (r *CompressorRegistry) GetByLanguage(lang string) LanguageCompressor {
	return r.compressors[lang]
}

// IsSupported checks if compression is available for a file path.
func (r *CompressorRegistry) IsSupported(filePath string) bool {
	return r.Get(filePath) != nil
}

// Languages returns a list of language identifiers that have registered compressors.
func (r *CompressorRegistry) Languages() []string {
	langs := make([]string, 0, len(r.compressors))
	for lang := range r.compressors {
		langs = append(langs, lang)
	}
	return langs
}