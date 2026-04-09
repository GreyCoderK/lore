// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "testing"

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		// SQL (case-insensitive)
		{"sql_select", "SELECT * FROM users", "sql"},
		{"sql_lower", "select * from users", "sql"},
		{"sql_insert", "INSERT INTO orders VALUES (1)", "sql"},

		// Java
		{"java_public_class", "public class Main {", "java"},
		{"java_import", "import java.util.List;", "java"},
		{"java_annotation", "@Override", "java"},

		// Go
		{"go_func", "func main() {", "go"},
		{"go_package_matches_java", "package main", "java"}, // Java "package " prefix wins
		{"go_type", "type Server struct {", "go"},
		{"go_import_paren_matches_java", "import (", "java"}, // Java "import " prefix wins
		{"go_defer", "defer close()", "go"},

		// Python (note: "import " matches Java, "from " matches SQL, "class " matches Java)
		{"python_def", "def main():", "python"},
		{"python_import_matches_java", "import os", "java"},                              // Java "import " wins
		{"python_from_matches_sql", "from collections import OrderedDict", "sql"},        // SQL "FROM " (case-insensitive) wins
		{"python_class_matches_java", "class MyClass:", "java"},                          // Java "class " wins
		{"python_if_name", "if __name__", "python"},
		{"python_async_def", "async def handler():", "python"},
		{"python_pytest", "@pytest.mark.skip", "python"},

		// Rust
		{"rust_fn", "fn main() {", "rust"},
		{"rust_pub_fn", "pub fn new() -> Self {", "rust"},
		{"rust_let_mut_matches_ts", "let mut x = 5;", "typescript"}, // TypeScript "let " prefix wins
		{"rust_derive", "#[derive(Debug)]", "rust"},
		{"rust_impl", "impl Display for Foo {", "rust"},
		{"rust_use", "use std::io;", "rust"},

		// Bash / Shell
		{"bash_shebang", "#!/bin/bash", "bash"},
		{"bash_curl", "curl https://example.com", "bash"},
		{"bash_git", "git commit -m 'fix'", "bash"},

		// JSON
		{"json_object", `{"key": "value"}`, "json"},
		{"json_array", `["a", "b", "c"]`, "json"},

		// YAML (contains ": ")
		{"yaml_apiversion", "apiVersion: v1", "yaml"},
		{"yaml_triple_dash", "---", "yaml"},

		// Dockerfile (note: "FROM " matches SQL case-insensitive first)
		{"dockerfile_from_matches_sql", "FROM ubuntu:22.04", "sql"}, // SQL "FROM " (case-insensitive) wins
		{"dockerfile_run", "RUN apt-get update", "dockerfile"},
		{"dockerfile_copy", "COPY . /app", "dockerfile"},
		{"dockerfile_workdir", "WORKDIR /app", "dockerfile"},
		{"dockerfile_expose", "EXPOSE 8080", "dockerfile"},

		// HTTP (case-insensitive)
		{"http_post", "POST /api/users", "http"},
		{"http_get", "GET /health HTTP/1.1", "http"},
		{"http_content_type", "Content-Type: application/json", "http"},

		// CSS (via Contains rules — YAML ": " Contains rule comes before CSS in scan order)
		{"css_display_no_colon_space", "display:flex;", "css"}, // no ": " so YAML won't match
		{"css_margin_no_colon_space", "margin:0;", "css"},      // no ": " so YAML won't match
		{"css_with_colon_space_matches_yaml", "color: red;", "yaml"}, // YAML ": " wins over CSS "color:"

		// TypeScript (note: "interface " also matches Java which comes first)
		{"typescript_interface_matches_java", "interface User {", "java"}, // Java "interface " wins
		{"typescript_export", "export default function", "typescript"},
		{"typescript_import_type_matches_java", "import type { Foo } from 'bar'", "java"}, // Java "import " wins
		{"typescript_const_matches_go", "const x: string = ''", "go"},       // Go "const " prefix wins
		{"typescript_arrow", "const fn = () => { return 1 }", "go"},  // Go "const " prefix wins; test "=> {" Contains separately
		{"typescript_string_annotation", "name: string", "typescript"}, // TS ": string" Contains wins (TS rule before YAML)
		{"typescript_arrow_fn", "items.map(x => { return x })", "typescript"},          // Contains "=> {"

		// Unknown / empty
		{"unknown_text", "Hello, this is just a sentence.", ""},
		{"empty_string", "", ""},
		{"whitespace_only", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.line)
			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestDetectLanguageMultiLine(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			"go_majority",
			[]string{"func main() {", "", "package main", "  var x int", "defer close()"},
			"go",
		},
		{
			"python_majority",
			[]string{"def main():", "  class Foo:", "from os import path", "  x = 1"},
			"python",
		},
		{
			"all_empty",
			[]string{"", "  ", ""},
			"",
		},
		{
			"single_line",
			[]string{"SELECT * FROM users"},
			"sql",
		},
		{
			"no_detectable",
			[]string{"some random text", "another line", "nothing here"},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguageMultiLine(tt.lines)
			if got != tt.want {
				t.Errorf("DetectLanguageMultiLine(%v) = %q, want %q", tt.lines, got, tt.want)
			}
		})
	}
}
