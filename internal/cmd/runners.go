package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

// cmdNext wraps `npx next dev -p $APP_PORT`. Next.js uses -p (single dash)
// rather than --port; both work but -p matches the form shown in the
// official docs.
func cmdNext(args []string, stdout, stderr io.Writer) int {
	port := envOrDefault("APP_PORT", "3000")
	fullArgs := buildNextArgs(args, port)

	npx, err := exec.LookPath("npx")
	if err != nil {
		fmt.Fprintln(stderr, "lane: npx not found in PATH")
		fmt.Fprintln(stderr, "  hint: install Node.js (with npm), or invoke `npx next dev -p $APP_PORT` directly")
		return ExitError
	}

	if err := syscall.Exec(npx, fullArgs, os.Environ()); err != nil {
		fmt.Fprintf(stderr, "lane: exec npx failed: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func buildNextArgs(extras []string, port string) []string {
	base := []string{"npx", "next", "dev", "-p", port}
	return append(base, extras...)
}

// cmdFlask wraps `flask run --port=$APP_PORT`. Flask requires FLASK_APP
// to be set in the environment (or via a .flaskenv file) to know which
// module to load; Lane intentionally does not set FLASK_APP — it would
// be guessing.
func cmdFlask(args []string, stdout, stderr io.Writer) int {
	port := envOrDefault("APP_PORT", "5000")
	fullArgs := buildFlaskArgs(args, port)

	flask, err := exec.LookPath("flask")
	if err != nil {
		fmt.Fprintln(stderr, "lane: flask not found in PATH")
		fmt.Fprintln(stderr, "  hint: activate your venv or install Flask; or invoke `flask run --port=$APP_PORT` directly")
		return ExitError
	}

	if err := syscall.Exec(flask, fullArgs, os.Environ()); err != nil {
		fmt.Fprintf(stderr, "lane: exec flask failed: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func buildFlaskArgs(extras []string, port string) []string {
	base := []string{"flask", "run", "--port=" + port}
	return append(base, extras...)
}

// cmdDjango wraps `python manage.py runserver $APP_PORT`. Falls back to
// `python3` when `python` is not on PATH (common on systems where macOS
// or distros only ship the python3 binary). Django's runserver takes
// the port as a positional argument, not a flag.
func cmdDjango(args []string, stdout, stderr io.Writer) int {
	port := envOrDefault("APP_PORT", "8000")

	py, err := exec.LookPath("python")
	if err != nil {
		py, err = exec.LookPath("python3")
		if err != nil {
			fmt.Fprintln(stderr, "lane: python or python3 not found in PATH")
			fmt.Fprintln(stderr, "  hint: install Python (or activate a venv); or invoke `python manage.py runserver $APP_PORT` directly")
			return ExitError
		}
	}

	// argv[0] reflects the binary that was actually resolved (python or
	// python3); buildDjangoArgs defaults to "python" for predictable test output.
	fullArgs := buildDjangoArgs(args, port)
	fullArgs[0] = filepath.Base(py)

	if err := syscall.Exec(py, fullArgs, os.Environ()); err != nil {
		fmt.Fprintf(stderr, "lane: exec %s failed: %v\n", filepath.Base(py), err)
		return ExitError
	}
	return ExitOK
}

func buildDjangoArgs(extras []string, port string) []string {
	base := []string{"python", "manage.py", "runserver", port}
	return append(base, extras...)
}

func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
