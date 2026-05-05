# smart-cd bash integration
# Usage: eval "$(sd --init bash)"

function sd() {
    # No arguments: go home
    if [ $# -eq 0 ]; then
        builtin cd "$HOME"
        return $?
    fi

    # Capture sd output; UI and errors go to the terminal via /dev/tty
    local target
    target=$(command sd "$@" 2>/dev/tty)
    local exit_code=$?

    if [ $exit_code -eq 0 ] && [ -n "$target" ]; then
        if builtin cd "$target"; then
            command sd --record "$target" &>/dev/null &
        fi
    fi
    return $exit_code
}

# Tab completion for bookmark names
_sd_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == @* ]]; then
        local names
        names=$(sd --list-bookmarks 2>/dev/null)
        COMPREPLY=($(compgen -W "$names" -- "$cur"))
    else
        # Fall back to directory completion
        COMPREPLY=($(compgen -d -- "$cur"))
    fi
}
complete -F _sd_completion sd
