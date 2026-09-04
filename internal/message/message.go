// Package message turns a shaped change into a committed-quality message.
//
// Inference goes through a llama-server child on a loopback port: spawned for the one
// request and killed before returning — a subprocess, not a daemon. The alternative,
// scraping llama-cli's stdout, was tried first and rejected: the CLI prints banners,
// prompt echoes and perf lines to stdout, and a commit message assembled by guessing
// which lines are the message is a bug with a delay on it.
package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The rules come AFTER the diff in the prompt: if anything ever truncates, it is
// diff that goes, never instructions. Phrased without inline dashes and quotes a
// small model likes to parrot into the subject line.
const rules = `The above is a staged git change. Write its commit message in Conventional Commits style.

Rules:
1. The first line is: type(optional scope): short imperative summary
2. Keep the first line under 72 characters.
3. Allowed types: feat, fix, refactor, chore, docs, test, style, perf, ci, build
4. Describe the change as a whole, not one file of it; the overview lists every file.
5. Optionally add a blank line and a short body of at most 4 bullet points explaining why.
6. Output only the commit message itself. No backticks, no quotes, no preamble, no rule text.`

func intEnv(name string, fallback int) int {
	if raw := os.Getenv(name); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return fallback
}

// Generate answers the cleaned commit message for a shaped change.
func Generate(runtime, model, shaped string) (string, error) {
	base, stop, err := startServer(runtime, model)
	if err != nil {
		return "", err
	}
	defer stop()

	raw, err := complete(base, shaped+"\n\n"+rules)
	if err != nil {
		return "", err
	}
	msg := Clean(raw)
	if msg == "" {
		return "", fmt.Errorf("empty response from the model")
	}
	return msg, nil
}

// startServer spawns llama-server on a free loopback port and waits until it is
// ready. Loopback is hard-coded, not configured: nothing this tool does is ever
// served to a network.
func startServer(runtime, model string) (string, func(), error) {
	server := strings.TrimSuffix(runtime, "llama-cli") + "llama-server"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	var stderr bytes.Buffer
	command := exec.Command(server,
		"-m", model,
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"-c", strconv.Itoa(intEnv("FF_CTX", 16384)),
		"-ngl", "99",
		"--no-webui",
	)
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return "", nil, fmt.Errorf("llama-server: %w", err)
	}
	exited := make(chan struct{})
	go func() { _ = command.Wait(); close(exited) }()
	stop := func() {
		_ = command.Process.Kill()
		<-exited
	}
	// Safe only once the process is gone: the buffer races with a live writer.
	lastWords := func() string {
		tail := stderr.String()
		if len(tail) > 800 {
			tail = tail[len(tail)-800:]
		}
		return tail
	}

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.After(2 * time.Minute)
	for {
		select {
		case <-exited:
			return "", nil, fmt.Errorf("llama-server exited before becoming ready\n%s", lastWords())
		case <-deadline:
			stop()
			return "", nil, fmt.Errorf("llama-server never became ready\n%s", lastWords())
		case <-time.After(150 * time.Millisecond):
			response, err := http.Get(base + "/health")
			if err != nil {
				continue
			}
			ok := response.StatusCode == http.StatusOK
			_ = response.Body.Close()
			if ok {
				return base, stop, nil
			}
		}
	}
}

// complete asks for one chat completion and answers the raw message content.
func complete(base, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.2,
		// Greedy sampling loves a loop: without this a 3B model has committed the
		// same bullet a dozen times, riding one sentence to the token cap.
		"repeat_penalty": 1.2,
		"max_tokens":     intEnv("FF_MAX_TOKENS", 400),
	})
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Post(base+"/v1/chat/completions", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llama-server answered %s", response.Status)
	}
	var answer struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		return "", err
	}
	if len(answer.Choices) == 0 {
		return "", fmt.Errorf("llama-server answered no choices")
	}
	return answer.Choices[0].Message.Content, nil
}

var (
	fence    = regexp.MustCompile("(?s)^```[a-zA-Z]*\n(.*?)\n?```\\s*$")
	preamble = regexp.MustCompile(`(?i)^(here'?s?( is)?( the)?|commit message)[^\n]*:\s*\n+`)
)

// The body cap the prompt promises. Enforced here too, because a rule a small model
// follows most days is not a rule.
const maxBodyLines = 4

// Clean is the message and nothing else, whatever the model wrapped it in.
func Clean(raw string) string {
	msg := strings.TrimSpace(raw)
	msg = strings.TrimSuffix(msg, "[end of text]")
	msg = strings.TrimSpace(msg)
	if match := fence.FindStringSubmatch(msg); match != nil {
		msg = strings.TrimSpace(match[1])
	}
	msg = preamble.ReplaceAllString(msg, "")
	msg = strings.Trim(msg, `"'`)
	return capped(strings.TrimSpace(msg))
}

// capped drops repeated body lines and holds the body to its promised length — the
// mechanical backstop for a sampler that once committed one bullet a dozen times.
func capped(msg string) string {
	lines := strings.Split(msg, "\n")
	kept := []string{lines[0]}
	seen := map[string]bool{}
	body := 0
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if last := kept[len(kept)-1]; last != "" {
				kept = append(kept, "")
			}
			continue
		}
		if seen[trimmed] || body >= maxBodyLines {
			continue
		}
		seen[trimmed] = true
		body++
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
