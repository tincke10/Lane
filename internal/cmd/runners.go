package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// cmdServe is the auto-port-aware wrapper around `php artisan serve`.
// It injects --port=$APP_PORT (falling back to artisan's default 8000)
// and execs the PHP binary, so the user's shell is replaced by the
// server process. Signals and stdin/stdout/stderr flow naturally.
//
// Any extra args are forwarded to artisan after the injected --port.
// Run `lane serve` inside a registered project and the dev server lands
// on Lane's allocated port without thinking about it.
func cmdServe(args []string, stdout, stderr io.Writer) int {
	port := envOrDefault("APP_PORT", "8000")
	fullArgs := buildServeArgs(args, port)

	php, err := exec.LookPath("php")
	if err != nil {
		fmt.Fprintln(stderr, "lane: php not found in PATH")
		fmt.Fprintln(stderr, "  hint: install PHP, or invoke `php artisan serve --port=$APP_PORT` directly")
		return ExitError
	}

	if err := syscall.Exec(php, fullArgs, os.Environ()); err != nil {
		fmt.Fprintf(stderr, "lane: exec php failed: %v\n", err)
		return ExitError
	}
	return ExitOK // unreachable on success
}

// cmdVite is the auto-port-aware wrapper around `npx vite` for the Vite
// dev server. Same model as cmdServe but for the Node.js side.
//
// We invoke vite via `npx` rather than `npm run dev` so the wrapper
// works in any project that has vite as a dev dependency, regardless of
// the user's package.json script naming.
func cmdVite(args []string, stdout, stderr io.Writer) int {
	port := envOrDefault("VITE_PORT", "5173")
	fullArgs := buildViteArgs(args, port)

	npx, err := exec.LookPath("npx")
	if err != nil {
		fmt.Fprintln(stderr, "lane: npx not found in PATH")
		fmt.Fprintln(stderr, "  hint: install Node.js (with npm), or invoke `npx vite --port=$VITE_PORT` directly")
		return ExitError
	}

	if err := syscall.Exec(npx, fullArgs, os.Environ()); err != nil {
		fmt.Fprintf(stderr, "lane: exec npx failed: %v\n", err)
		return ExitError
	}
	return ExitOK // unreachable on success
}

// buildServeArgs returns the full argv (argv[0] included) for
// `php artisan serve --port=PORT [extras...]`. syscall.Exec needs the
// program name in argv[0]; we return it together so the caller passes
// the same slice to both LookPath and Exec.
func buildServeArgs(extras []string, port string) []string {
	base := []string{"php", "artisan", "serve", "--port=" + port}
	return append(base, extras...)
}

// buildViteArgs returns argv for `npx vite --port=PORT [extras...]`.
func buildViteArgs(extras []string, port string) []string {
	base := []string{"npx", "vite", "--port=" + port}
	return append(base, extras...)
}

func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
