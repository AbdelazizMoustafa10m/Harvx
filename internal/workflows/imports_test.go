package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// TestParseImports -- top-level dispatcher
// ---------------------------------------------------------------------------

func TestParseImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		content  string
		want     []string
	}{
		{
			name:     "Go file with internal import extracts relative path",
			filePath: "main.go",
			content:  `import "github.com/harvx/harvx/internal/config"`,
			want:     []string{"internal/config"},
		},
		{
			name:     "Go file with stdlib imports returns nil",
			filePath: "main.go",
			content: `import (
	"fmt"
	"os"
	"strings"
)`,
			want: nil,
		},
		{
			name:     "Go file with relative import",
			filePath: "main.go",
			content:  `import "./foo"`,
			want:     []string{"./foo"},
		},
		{
			name:     "TypeScript file with relative imports",
			filePath: "app.ts",
			content: `import { Component } from './component'
import { Utils } from './utils/helper'`,
			want: []string{"./component", "./utils/helper"},
		},
		{
			name:     "JavaScript file with require relative paths",
			filePath: "app.js",
			content: `const foo = require('./foo')
const bar = require('./bar/baz')`,
			want: []string{"./bar/baz", "./foo"},
		},
		{
			name:     "Python file with relative imports",
			filePath: "app.py",
			content: `from .models import User
from .utils import helper`,
			want: []string{"models", "utils"},
		},
		{
			name:     "Unknown extension returns nil",
			filePath: "file.rb",
			content:  `require 'foo'`,
			want:     nil,
		},
		{
			name:     "Empty content Go file returns nil",
			filePath: "empty.go",
			content:  "",
			want:     nil,
		},
		{
			name:     "Empty content JS file returns nil",
			filePath: "empty.js",
			content:  "",
			want:     nil,
		},
		{
			name:     "Empty content Python file returns nil",
			filePath: "empty.py",
			content:  "",
			want:     nil,
		},
		{
			name:     "TSX extension dispatches to JS parser",
			filePath: "component.tsx",
			content:  `import { App } from './App'`,
			want:     []string{"./App"},
		},
		{
			name:     "JSX extension dispatches to JS parser",
			filePath: "component.jsx",
			content:  `import { App } from './App'`,
			want:     []string{"./App"},
		},
		{
			name:     "Case-insensitive extension matching",
			filePath: "main.GO",
			content:  `import "github.com/harvx/harvx/internal/config"`,
			want:     []string{"internal/config"},
		},
		{
			name:     "Deduplicates and sorts results",
			filePath: "app.js",
			content: `import { a } from './zebra'
import { b } from './alpha'
import { c } from './zebra'`,
			want: []string{"./alpha", "./zebra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseImports(tt.filePath, tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestParseGoImports -- Go import parsing
// ---------------------------------------------------------------------------

func TestParseGoImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single import with internal marker",
			content: `import "github.com/harvx/harvx/internal/config"`,
			want:    []string{"internal/config"},
		},
		{
			name: "multi-line import block",
			content: `import (
	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/discovery"
)`,
			want: []string{"internal/config", "internal/discovery"},
		},
		{
			name: "import with alias",
			content: `import (
	cfg "github.com/harvx/harvx/internal/config"
)`,
			want: []string{"internal/config"},
		},
		{
			name: "stdlib-only imports return empty",
			content: `import (
	"fmt"
	"os"
	"strings"
)`,
			want: nil,
		},
		{
			name:    "import with /internal/ marker",
			content: `import "github.com/example/project/internal/util"`,
			want:    []string{"internal/util"},
		},
		{
			name:    "import with /cmd/ marker",
			content: `import "github.com/example/project/cmd/tool"`,
			want:    []string{"cmd/tool"},
		},
		{
			name:    "import with /pkg/ marker",
			content: `import "github.com/example/project/pkg/lib"`,
			want:    []string{"pkg/lib"},
		},
		{
			name: "mixed stdlib and project imports",
			content: `import (
	"fmt"
	"os"
	"github.com/harvx/harvx/internal/config"
	"strings"
	"github.com/harvx/harvx/pkg/util"
)`,
			want: []string{"internal/config", "pkg/util"},
		},
		{
			name:    "relative import ./foo",
			content: `import "./foo"`,
			want:    []string{"./foo"},
		},
		{
			name:    "relative import ../bar",
			content: `import "../bar"`,
			want:    []string{"../bar"},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name: "third-party imports without markers are excluded",
			content: `import (
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)`,
			want: nil,
		},
		{
			name: "deeply nested internal path",
			content: `import "github.com/harvx/harvx/internal/config/profiles/defaults"`,
			want:    []string{"internal/config/profiles/defaults"},
		},
		{
			name: "blank import with underscore alias",
			content: `import (
	_ "github.com/harvx/harvx/internal/driver"
)`,
			// The underscore alias matches the regex pattern since it captures
			// the quoted path after optional alias text.
			want: []string{"internal/driver"},
		},
		{
			name: "dot import",
			content: `import (
	. "github.com/harvx/harvx/internal/testutil"
)`,
			want: []string{"internal/testutil"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseGoImports(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestParseJSImports -- JavaScript/TypeScript import parsing
// ---------------------------------------------------------------------------

func TestParseJSImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "ES import from relative path with single quotes",
			content: `import { foo } from './foo'`,
			want:    []string{"./foo"},
		},
		{
			name:    "ES import from relative path with double quotes",
			content: `import { foo } from "./foo"`,
			want:    []string{"./foo"},
		},
		{
			name:    "ES import from package (non-relative) excluded",
			content: `import React from 'react'`,
			want:    nil,
		},
		{
			name:    "CommonJS require with relative path",
			content: `const foo = require('./foo')`,
			want:    []string{"./foo"},
		},
		{
			name:    "CommonJS require with package excluded",
			content: `const express = require('express')`,
			want:    nil,
		},
		{
			name: "mixed import and require",
			content: `import { a } from './alpha'
const b = require('./beta')`,
			want: []string{"./alpha", "./beta"},
		},
		{
			name:    "import type from (TypeScript)",
			content: `import type { Config } from './config'`,
			want:    []string{"./config"},
		},
		{
			name:    "parent-relative import preserved",
			content: `import { util } from '../utils/helper'`,
			want:    []string{"../utils/helper"},
		},
		{
			name:    "require with parent-relative path",
			content: `const x = require('../lib/core')`,
			want:    []string{"../lib/core"},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name: "mixed relative and non-relative imports",
			content: `import { a } from './local'
import { b } from 'lodash'
const c = require('./other')
const d = require('express')`,
			want: []string{"./local", "./other"},
		},
		{
			name:    "default import from relative path",
			content: `import App from './App'`,
			want:    []string{"./App"},
		},
		{
			name:    "namespace import",
			content: `import * as utils from './utils'`,
			want:    []string{"./utils"},
		},
		{
			name: "both import and require on same line captured separately",
			content: `import { x } from './x'
const y = require('./y')`,
			want: []string{"./x", "./y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseJSImports(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestParsePythonImports -- Python import parsing
// ---------------------------------------------------------------------------

func TestParsePythonImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "from .module import name (relative)",
			content: `from .module import MyClass`,
			want:    []string{"module"},
		},
		{
			name:    "from ..module import name (parent relative)",
			content: `from ..module import MyClass`,
			want:    []string{"../module"},
		},
		{
			name:    "import module.submodule (absolute dotted)",
			content: `import os.path`,
			want:    []string{"os/path"},
		},
		{
			name:    "from module import name (absolute)",
			content: `from collections import OrderedDict`,
			want:    []string{"collections"},
		},
		{
			name: "empty file",
			content: ``,
			want:    nil,
		},
		{
			name: "comment-only file",
			content: `# this is a comment
# another comment`,
			want: nil,
		},
		{
			name: "mixed relative and absolute imports",
			content: `from .models import User
from ..utils import helper
import json
from collections import OrderedDict`,
			want: []string{"models", "../utils", "json", "collections"},
		},
		{
			name:    "deeply dotted relative import",
			content: `from ...deep.module import thing`,
			want:    []string{"../../deep/module"},
		},
		{
			name:    "from . import name (bare relative)",
			content: `from . import something`,
			// "." with no remainder after stripping the leading dot yields "",
			// which is filtered out by convertPythonDottedPath.
			want: nil,
		},
		{
			name:    "dotted absolute import converts to path",
			content: `import foo.bar.baz`,
			want:    []string{"foo/bar/baz"},
		},
		{
			name: "from with dotted relative path",
			content: `from .foo.bar import something`,
			want:    []string{"foo/bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parsePythonImports(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestDeduplicateAndSort
// ---------------------------------------------------------------------------

func TestDeduplicateAndSort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "empty input returns nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty slice returns nil",
			input: []string{},
			want:  nil,
		},
		{
			name:  "already unique and sorted",
			input: []string{"alpha", "beta", "gamma"},
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "duplicates removed and sorted",
			input: []string{"gamma", "alpha", "gamma", "beta", "alpha"},
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "single element",
			input: []string{"only"},
			want:  []string{"only"},
		},
		{
			name:  "all duplicates",
			input: []string{"same", "same", "same"},
			want:  []string{"same"},
		},
		{
			name:  "already unique but unsorted",
			input: []string{"z", "a", "m"},
			want:  []string{"a", "m", "z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deduplicateAndSort(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsRelativeImport
// ---------------------------------------------------------------------------

func TestIsRelativeImport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "dot-slash prefix is relative",
			path: "./foo",
			want: true,
		},
		{
			name: "dot-dot-slash prefix is relative",
			path: "../bar",
			want: true,
		},
		{
			name: "module path is not relative",
			path: "module/foo",
			want: false,
		},
		{
			name: "empty string is not relative",
			path: "",
			want: false,
		},
		{
			name: "bare dot is not relative (no slash)",
			path: ".",
			want: false,
		},
		{
			name: "absolute path is not relative",
			path: "/usr/local/lib",
			want: false,
		},
		{
			name: "package name is not relative",
			path: "react",
			want: false,
		},
		{
			name: "dot-dot without slash is not relative",
			path: "..foo",
			want: false,
		},
		{
			name: "deeply nested relative",
			path: "../../deeply/nested",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isRelativeImport(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestNormalizeJSImport
// ---------------------------------------------------------------------------

func TestNormalizeJSImport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "dot-slash preserved",
			input: "./foo",
			want:  "./foo",
		},
		{
			name:  "parent-relative unchanged",
			input: "../bar",
			want:  "../bar",
		},
		{
			name:  "non-relative unchanged",
			input: "foo",
			want:  "foo",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
		{
			name:  "deeply nested dot-slash preserved",
			input: "./deep/nested/path",
			want:  "./deep/nested/path",
		},
		{
			name:  "parent relative deeply nested unchanged",
			input: "../../foo/bar",
			want:  "../../foo/bar",
		},
		{
			name:  "just dot-slash preserved",
			input: "./",
			want:  "./",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeJSImport(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestConvertPythonDottedPath
// ---------------------------------------------------------------------------

func TestConvertPythonDottedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single dot with module",
			input: ".foo.bar",
			want:  "foo/bar",
		},
		{
			name:  "double dot with module",
			input: "..foo.bar",
			want:  "../foo/bar",
		},
		{
			name:  "triple dot with module",
			input: "...foo",
			want:  "../../foo",
		},
		{
			name:  "single dot only returns empty",
			input: ".",
			want:  "",
		},
		{
			name:  "double dot only returns empty",
			input: "..",
			want:  "",
		},
		{
			name:  "single dot with simple module",
			input: ".models",
			want:  "models",
		},
		{
			name:  "four dots with module",
			input: "....deep",
			want:  "../../../deep",
		},
		{
			name:  "double dot with deeply nested module",
			input: "..a.b.c.d",
			want:  "../a/b/c/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := convertPythonDottedPath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
