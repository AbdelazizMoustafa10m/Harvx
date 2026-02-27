package pipeline

// PipelineOption configures a Pipeline during construction.
type PipelineOption func(*Pipeline)

// WithDiscovery sets the discovery stage implementation.
func WithDiscovery(d DiscoveryService) PipelineOption {
	return func(p *Pipeline) {
		p.discovery = d
	}
}

// WithRelevance sets the relevance classification stage implementation.
func WithRelevance(r RelevanceService) PipelineOption {
	return func(p *Pipeline) {
		p.relevance = r
	}
}

// WithTokenizer sets the tokenizer service for token counting.
func WithTokenizer(t TokenizerService) PipelineOption {
	return func(p *Pipeline) {
		p.tokenizer = t
	}
}

// WithBudget sets the budget enforcement service.
func WithBudget(b BudgetService) PipelineOption {
	return func(p *Pipeline) {
		p.budget = b
	}
}

// WithRedactor sets the redaction service for secret scanning.
func WithRedactor(r RedactionService) PipelineOption {
	return func(p *Pipeline) {
		p.redactor = r
	}
}

// WithCompressor sets the compression service.
func WithCompressor(c CompressionService) PipelineOption {
	return func(p *Pipeline) {
		p.compressor = c
	}
}

// WithRenderer sets the output rendering service.
func WithRenderer(r RenderService) PipelineOption {
	return func(p *Pipeline) {
		p.renderer = r
	}
}
