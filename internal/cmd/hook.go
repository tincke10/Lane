package cmd

import (
	"fmt"
	"io"
)

func cmdHook(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "lane: usage: lane hook <bash|zsh|fish>")
		return ExitUsage
	}

	switch args[0] {
	case "bash":
		fmt.Fprint(stdout, bashHook)
	case "zsh":
		fmt.Fprint(stdout, zshHook)
	case "fish":
		fmt.Fprint(stdout, fishHook)
	default:
		fmt.Fprintf(stderr, "lane: unsupported shell %q (supported: bash, zsh, fish)\n", args[0])
		return ExitUsage
	}
	return ExitOK
}

// bashHook registers _lane_hook in PROMPT_COMMAND. The guard checks if
// the hook is already installed so that re-sourcing .bashrc does not
// stack multiple copies. stderr is discarded so a broken registry never
// breaks the user's prompt.
const bashHook = `_lane_hook() {
  local output
  output="$(lane export 2>/dev/null)" || return
  if [ -n "$output" ]; then
    eval "$output"
  fi
}

if [[ ";$PROMPT_COMMAND;" != *";_lane_hook;"* ]]; then
  PROMPT_COMMAND="_lane_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
`

// zshHook registers _lane_hook in precmd_functions, the zsh-native
// equivalent of bash's PROMPT_COMMAND. The (I) flag in the test makes
// the guard idempotent across re-sources.
const zshHook = `_lane_hook() {
  eval "$(lane export 2>/dev/null)"
}

typeset -ag precmd_functions
if (( ! ${precmd_functions[(I)_lane_hook]} )); then
  precmd_functions+=(_lane_hook)
fi
`

// fishHook binds to the fish_prompt event, which fires before each
// prompt — equivalent in spirit to bash's PROMPT_COMMAND and zsh's
// precmd. Output is piped into `source` so fish executes it in the
// current scope. The functions -q guard makes re-sourcing idempotent.
const fishHook = `if not functions -q _lane_hook
    function _lane_hook --on-event fish_prompt
        lane export --shell fish 2>/dev/null | source
    end
end
`
