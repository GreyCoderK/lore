// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "strings"

// LangRule defines a detection rule for a programming language.
type LangRule struct {
	Lang     string   // language tag for code fences (e.g., "java", "sql")
	Prefixes []string // line prefixes that indicate this language (case-sensitive unless CaseInsensitive)
	Contains []string // substrings that indicate this language (checked if no prefix matches)
	CaseInsensitive bool // if true, prefixes are matched case-insensitively
}

// langRules is the ordered registry of language detection rules.
// First match wins. Add new languages here.
var langRules = []LangRule{
	// SQL — case-insensitive keywords
	{
		Lang: "sql",
		Prefixes: []string{
			"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "CREATE ", "ALTER ", "DROP ",
			"WHERE ", "JOIN ", "LEFT JOIN ", "RIGHT JOIN ", "INNER JOIN ",
			"FROM ", "ORDER BY ", "GROUP BY ", "HAVING ", "UNION ",
			"WITH ", "GRANT ", "REVOKE ", "TRUNCATE ",
		},
		CaseInsensitive: true,
	},
	// Java
	{
		Lang: "java",
		Prefixes: []string{
			"public ", "private ", "protected ", "import ", "package ",
			"@Query", "@Param", "@Override", "@Autowired", "@Service",
			"@Controller", "@Repository", "@Entity", "@Table",
			"class ", "interface ", "enum ", "abstract ",
		},
	},
	// Kotlin
	{
		Lang: "kotlin",
		Prefixes: []string{
			"fun ", "val ", "var ", "data class ", "sealed class ",
			"object ", "suspend fun ", "companion object",
		},
	},
	// Go
	{
		Lang: "go",
		Prefixes: []string{
			"func ", "package ", "type ", "import (",
			"var ", "const ", "defer ", "go func",
		},
	},
	// Python
	{
		Lang: "python",
		Prefixes: []string{
			"def ", "class ", "import ", "from ", "if __name__",
			"@app.", "@pytest", "async def ", "lambda ",
		},
	},
	// TypeScript / JavaScript
	{
		Lang: "typescript",
		Prefixes: []string{
			"interface ", "export ", "const ", "let ", "async function",
			"import {", "import type", "type ",
		},
		Contains: []string{": string", ": number", ": boolean", "=> {"},
	},
	{
		Lang: "javascript",
		Prefixes: []string{
			"function ", "module.exports", "require(", "const ", "let ", "var ",
		},
	},
	// Rust
	{
		Lang: "rust",
		Prefixes: []string{
			"fn ", "let mut ", "pub fn ", "impl ", "struct ", "enum ",
			"use ", "mod ", "#[derive", "trait ",
		},
	},
	// C / C++
	{
		Lang: "c",
		Prefixes: []string{
			"#include ", "#define ", "int main(", "void ", "typedef ",
		},
	},
	// PHP
	{
		Lang: "php",
		Prefixes: []string{
			"<?php", "namespace ", "use ", "public function ",
			"private function ", "protected function ",
		},
		Contains: []string{"$this->", "->"},
	},
	// Ruby
	{
		Lang: "ruby",
		Prefixes: []string{
			"require ", "def ", "class ", "module ", "attr_",
			"describe ", "it ", "end", "gem ",
		},
	},
	// Swift
	{
		Lang: "swift",
		Prefixes: []string{
			"import Foundation", "import UIKit", "struct ", "class ",
			"func ", "let ", "var ", "@objc", "protocol ",
		},
	},
	// Dockerfile
	{
		Lang: "dockerfile",
		Prefixes: []string{
			"FROM ", "RUN ", "COPY ", "WORKDIR ", "EXPOSE ", "CMD ",
			"ENTRYPOINT ", "ENV ", "ARG ", "LABEL ",
		},
	},
	// YAML
	{
		Lang: "yaml",
		Prefixes: []string{"---"},
		Contains: []string{": "},
	},
	// TOML
	{
		Lang: "toml",
		Prefixes: []string{"[package]", "[dependencies]", "[workspace]", "[tool."},
	},
	// JSON
	{
		Lang: "json",
		Prefixes: []string{"{", "["},
	},
	// XML / HTML
	{
		Lang: "xml",
		Prefixes: []string{"<?xml", "<!", "<html", "<div", "<span", "<head", "<body"},
	},
	// CSS
	{
		Lang: "css",
		Contains: []string{"{ ", "margin:", "padding:", "display:", "color:", "font-"},
	},
	// VHS (terminal recorder tape files)
	{
		Lang: "vhs",
		Prefixes: []string{
			"Output ", "Type ", "Set ", "Sleep ", "Hide", "Show",
			"Require", "Enter", "Backspace", "Ctrl+", "Alt+",
			"Left", "Right", "Up", "Down", "Tab", "Space",
			"Screenshot", "Copy", "Paste", "Source ",
		},
	},
	// Shell / Bash
	{
		Lang: "bash",
		Prefixes: []string{
			"#!/", "$ ", "curl ", "git ", "lore ", "brew ", "npm ", "yarn ",
			"go ", "docker ", "kubectl ", "helm ", "make ", "pip ",
			"apt ", "yum ", "dnf ", "pacman ", "cargo ", "rustup ",
			"echo ", "export ", "source ", "chmod ", "mkdir ", "cd ",
		},
	},
	// HTTP
	{
		Lang: "http",
		Prefixes: []string{
			"GET ", "POST ", "PUT ", "PATCH ", "DELETE ", "HEAD ", "OPTIONS ",
			"HTTP/", "Content-Type:", "Authorization:", "Accept:",
		},
		CaseInsensitive: true,
	},
	// GraphQL
	{
		Lang: "graphql",
		Prefixes: []string{"query ", "mutation ", "subscription ", "type ", "schema "},
		Contains: []string{"@deprecated", "implements "},
	},
	// Protobuf
	{
		Lang: "protobuf",
		Prefixes: []string{"syntax = ", "message ", "service ", "rpc ", "package "},
	},
	// Terraform / HCL
	{
		Lang: "hcl",
		Prefixes: []string{"resource ", "variable ", "output ", "provider ", "terraform "},
	},
}

// DetectLanguage guesses the programming language from the first line(s) of a code block.
// Returns the language tag (e.g., "java", "sql") or "" if unknown.
func DetectLanguage(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}

	for _, rule := range langRules {
		// Check prefixes
		for _, prefix := range rule.Prefixes {
			if rule.CaseInsensitive {
				if len(trimmed) >= len(prefix) && strings.EqualFold(trimmed[:len(prefix)], prefix) {
					return rule.Lang
				}
			} else {
				if strings.HasPrefix(trimmed, prefix) {
					return rule.Lang
				}
			}
		}
	}

	// Second pass: check contains rules (lower priority)
	for _, rule := range langRules {
		for _, substr := range rule.Contains {
			if strings.Contains(trimmed, substr) {
				return rule.Lang
			}
		}
	}

	return ""
}

// DetectLanguageMultiLine guesses the language from multiple lines for better accuracy.
// Checks first 5 non-empty lines and uses majority vote.
func DetectLanguageMultiLine(lines []string) string {
	votes := map[string]int{}
	checked := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lang := DetectLanguage(line)
		if lang != "" {
			votes[lang]++
		}
		checked++
		if checked >= 5 {
			break
		}
	}

	if len(votes) == 0 {
		return ""
	}

	// Return language with most votes
	best := ""
	bestCount := 0
	for lang, count := range votes {
		if count > bestCount {
			best = lang
			bestCount = count
		}
	}
	return best
}
