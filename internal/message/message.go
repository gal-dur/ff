// Package message turns a shaped change into a committed-quality message.
package message

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// The rules come AFTER the diff in the prompt: if anything ever truncates, it is
// diff that goes, never instructions — the failure mode the Ollama version had.
const rules = `The above is a staged git change. Write its commit message, Conventional Commits style.

Rules:
- First line: <type>(<optional scope>): <short imperative summary> — max 72 chars
- Types: feat, fix, refactor, chore, docs, test, style, perf, ci, build
- Describe the change as a whole, not one file of it; the overview lists every file
- Optionally add a blank line + short body (bullet points, max 4 lines) explaining WHY
- Output ONLY the commit message — no backticks, no quotes, no preamble, no explanation`

func intEnv(name string, fallback int) int {
	if raw := os.Getenv(name); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return fallback
}

// Generate runs one llama-cli turn over the shaped change and answers the cleaned
// message. One-shot by design: no server, no port, no state between runs.
func Generate(runtime, model, shaped string) (string, error) {
	command := exec.Command(runtime,
		"-m", model,
		"--single-turn", "--no-display-prompt",
		"--temp", "0.2",
		"-c", strconv.Itoa(intEnv("FF_CTX", 16384)),
		"-n", strconv.Itoa(intEnv("FF_MAX_TOKENS", 400)),
		"-ngl", "99",
		"-p", shaped+"\n\n"+rules,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		// The runtime's own last words are the only useful diagnostic.
		tail := stderr.String()
		if len(tail) > 800 {
			tail = tail[len(tail)-800:]
		}
		return "", fmt.Errorf("llama-cli: %w\n%s", err, tail)
	}
	msg := Clean(stdout.String())
	if msg == "" {
		return "", fmt.Errorf("empty response from the model")
	}
	return msg, nil
}

var (
	fence    = regexp.MustCompile("(?s)^```[a-zA-Z]*\n(.*?)\n?```\\s*$")
	preamble = regexp.MustCompile(`(?i)^(here'?s?( is)?( the)?|commit message)[^\n]*:\s*\n+`)
)

// Clean is the message and nothing else, whatever the model wrapped it in.
func Clean(raw string) string {
	msg := strings.TrimSpace(raw)
	// llama-cli's single-turn mode may append an end-of-generation marker.
	msg = strings.TrimSuffix(msg, "[end of text]")
	msg = strings.TrimSpace(msg)
	if match := fence.FindStringSubmatch(msg); match != nil {
		msg = strings.TrimSpace(match[1])
	}
	msg = preamble.ReplaceAllString(msg, "")
	msg = strings.Trim(msg, `"'`)
	return strings.TrimSpace(msg)
}
